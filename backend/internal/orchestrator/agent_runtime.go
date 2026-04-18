package orchestrator

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/provider"
)

type agentRuntimeProfile struct {
	MaxIterations        int
	MaxForcedToolNudges  int
	MaxTokens            int
	Temperature          float64
	Retry                provider.RetryConfig
	CompatibilityPrompt  string
	AllowRawJSONToolArgs bool
}

var rawJSONBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*({.*?})\\s*```")

func buildAgentRuntimeProfile(cfg provider.ProviderConfig, task *domain.Task) agentRuntimeProfile {
	requiredTool := requiredOutputTool(task)
	profile := agentRuntimeProfile{
		MaxIterations:        15,
		MaxForcedToolNudges:  4,
		MaxTokens:            2048,
		Temperature:          0.3,
		Retry:                provider.DefaultRetryConfig(),
		AllowRawJSONToolArgs: true,
	}

	if requiredTool == "write_pptx" {
		profile.MaxTokens = 16384
		profile.Temperature = 0.7
	}
	if requiredTool == "write_html_docx" || requiredTool == "write_html_pdf" || requiredTool == "write_docx" {
		profile.MaxTokens = 16384
		profile.Temperature = 0.5
	}

	switch strings.ToLower(strings.TrimSpace(cfg.ProviderName)) {
	case "zai":
		profile.MaxIterations = 12
		profile.MaxForcedToolNudges = 3
	case "ollama":
		profile.MaxIterations = 8
		profile.MaxForcedToolNudges = 2
		profile.Retry.MaxAttempts = 1
		if requiredTool == "write_pptx" {
			profile.MaxTokens = 4096
			profile.Temperature = 0.2
		} else {
			profile.MaxTokens = 1536
			profile.Temperature = 0.15
		}
		profile.CompatibilityPrompt = buildWeakModelPrompt(requiredTool)
		if provider.IsWeakLocalModel(cfg.Model) {
			profile.MaxIterations = 6
			profile.MaxForcedToolNudges = 1
			if requiredTool == "write_pptx" {
				profile.MaxTokens = 3072
			} else {
				profile.MaxTokens = 1024
			}
		}
	}

	return profile
}

func buildWeakModelPrompt(requiredTool string) string {
	if strings.TrimSpace(requiredTool) == "" {
		return ""
	}
	return "Model compatibility mode: native tool calling may be unreliable for this model. " +
		"If you cannot emit a proper tool call, respond with ONLY the JSON object of arguments for " + requiredTool + ". " +
		"Do not add prose, markdown, or explanation around the JSON."
}

func recoverToolCallFromContent(content, requiredTool string) (provider.ToolCall, bool) {
	requiredTool = strings.TrimSpace(requiredTool)
	if requiredTool == "" {
		return provider.ToolCall{}, false
	}

	candidate := extractJSONObject(content)
	if candidate == "" {
		return provider.ToolCall{}, false
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(candidate), &payload); err != nil {
		return provider.ToolCall{}, false
	}

	for _, key := range []string{"tool", "tool_name", "name"} {
		if rawName, ok := payload[key].(string); ok && strings.TrimSpace(rawName) != "" {
			if strings.TrimSpace(rawName) != requiredTool {
				return provider.ToolCall{}, false
			}
			if args, ok := payload["arguments"]; ok {
				argBytes, err := json.Marshal(args)
				if err != nil {
					return provider.ToolCall{}, false
				}
				return provider.ToolCall{
					ID:        "json-fallback-" + requiredTool,
					Name:      requiredTool,
					Arguments: string(argBytes),
				}, true
			}
		}
	}

	return provider.ToolCall{
		ID:        "json-fallback-" + requiredTool,
		Name:      requiredTool,
		Arguments: candidate,
	}, true
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if json.Valid([]byte(raw)) && strings.HasPrefix(raw, "{") {
		return raw
	}
	if m := rawJSONBlockRe.FindStringSubmatch(raw); len(m) > 1 && json.Valid([]byte(m[1])) {
		return strings.TrimSpace(m[1])
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		blob := strings.TrimSpace(raw[start : end+1])
		if json.Valid([]byte(blob)) {
			return blob
		}
	}
	// Last resort: repair a response that was truncated mid-stream. A large JSON
	// plan that hits the token cap loses its trailing braces/brackets and then
	// fails json.Valid above. Closing the open tokens we saw gives back JSON
	// that parses into a partial (but usable) object, instead of forcing the
	// caller into a silent template fallback.
	if start >= 0 {
		if repaired := repairTruncatedJSONObject(raw[start:]); repaired != "" {
			return repaired
		}
	}
	return ""
}

// repairTruncatedJSONObject walks a candidate JSON string (expected to start
// with '{'), tracks brace/bracket/string state, and appends the closers needed
// to balance it. It returns the repaired string if it parses as valid JSON and
// "" otherwise. Incomplete string values are terminated, trailing commas are
// stripped, and incomplete key:value pairs are dropped so the outer object
// remains parseable.
func repairTruncatedJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") {
		return ""
	}
	var (
		depthStack []byte // stack of '{' or '['
		inString   bool
		escape     bool
		lastSafe   = 0 // index up to which we've seen complete tokens at object level
	)
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{', '[':
			depthStack = append(depthStack, ch)
		case '}', ']':
			if len(depthStack) == 0 {
				return ""
			}
			depthStack = depthStack[:len(depthStack)-1]
			if len(depthStack) == 1 {
				// We just closed a top-level value inside the root object.
				lastSafe = i + 1
			} else if len(depthStack) == 0 {
				// Root object closed cleanly — nothing to repair.
				if json.Valid([]byte(raw[:i+1])) {
					return raw[:i+1]
				}
			}
		case ',':
			if len(depthStack) == 1 {
				lastSafe = i + 1
			}
		}
	}
	if len(depthStack) == 0 {
		return ""
	}
	// Drop the incomplete tail (everything after the last balanced comma/close
	// at object depth), then append the closers for the still-open containers.
	head := strings.TrimRight(strings.TrimSpace(raw[:lastSafe]), ",")
	// After trimming the trailing comma we may have lost the current container,
	// so rescan depth from the retained head.
	depthStack = depthStack[:0]
	inString = false
	escape = false
	for i := 0; i < len(head); i++ {
		ch := head[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{', '[':
			depthStack = append(depthStack, ch)
		case '}', ']':
			if len(depthStack) > 0 {
				depthStack = depthStack[:len(depthStack)-1]
			}
		}
	}
	var tail strings.Builder
	tail.WriteString(head)
	for i := len(depthStack) - 1; i >= 0; i-- {
		if depthStack[i] == '{' {
			tail.WriteByte('}')
		} else {
			tail.WriteByte(']')
		}
	}
	candidate := tail.String()
	if json.Valid([]byte(candidate)) {
		return candidate
	}
	return ""
}
