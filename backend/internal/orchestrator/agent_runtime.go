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
	return ""
}
