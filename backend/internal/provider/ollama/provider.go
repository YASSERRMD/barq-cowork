package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/provider"
)

const defaultBaseURL = "http://localhost:11434"

type Provider struct {
	timeoutSec int
}

type ollamaMsg struct {
	Role      string               `json:"role"`
	Content   string               `json:"content,omitempty"`
	ToolCalls []ollamaWireToolCall `json:"tool_calls,omitempty"`
	ToolName  string               `json:"tool_name,omitempty"`
}

type ollamaWireToolCall struct {
	Function ollamaWireFunction `json:"function"`
}

type ollamaWireFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Arguments   json.RawMessage `json:"arguments,omitempty"`
}

type ollamaWireTool struct {
	Type     string             `json:"type"`
	Function ollamaWireToolSpec `json:"function"`
}

type ollamaWireToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

func New(timeoutSec int) *Provider {
	if timeoutSec <= 0 {
		timeoutSec = 300 // local models can be slow
	}
	return &Provider{timeoutSec: timeoutSec}
}

func (p *Provider) Name() string { return "ollama" }

func (p *Provider) ValidateConfig(cfg provider.ProviderConfig) error {
	if cfg.Model == "" {
		return fmt.Errorf("ollama: model is required")
	}
	return nil // no API key needed for local ollama
}

func (p *Provider) Chat(ctx context.Context, cfg provider.ProviderConfig, req provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if err := p.ValidateConfig(provider.ProviderConfig{Model: firstNonEmpty(req.Model, cfg.Model)}); err != nil {
		return nil, err
	}

	messages := make([]ollamaMsg, 0, len(req.Messages))
	toolNameByID := map[string]string{}
	for _, m := range req.Messages {
		msg := ollamaMsg{
			Role:    m.Role,
			Content: m.Content,
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]ollamaWireToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				toolNameByID[tc.ID] = tc.Name
				msg.ToolCalls = append(msg.ToolCalls, ollamaWireToolCall{
					Function: ollamaWireFunction{
						Name:      tc.Name,
						Arguments: encodeOllamaArguments(tc.Arguments),
					},
				})
			}
		}
		if m.Role == "tool" && m.ToolCallID != "" {
			msg.ToolName = toolNameByID[m.ToolCallID]
		}
		messages = append(messages, msg)
	}

	body := map[string]any{
		"model":    firstNonEmpty(req.Model, cfg.Model),
		"messages": messages,
		"stream":   req.Stream,
		"options": map[string]any{
			"num_predict": req.MaxTokens,
		},
	}
	if req.Temperature > 0 {
		body["options"].(map[string]any)["temperature"] = req.Temperature
	}
	if len(req.Tools) > 0 {
		tools := make([]ollamaWireTool, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, ollamaWireTool{
				Type: "function",
				Function: ollamaWireToolSpec{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			})
		}
		body["tools"] = tools
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	timeout := time.Duration(p.timeoutSec) * time.Second
	client := &http.Client{Timeout: timeout}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/chat", bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: request failed (is ollama running?): %w", err)
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("ollama: HTTP %d", resp.StatusCode)
	}

	ch := make(chan provider.ChatCompletionChunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
		for scanner.Scan() {
			var event struct {
				Message struct {
					Content   string               `json:"content"`
					Thinking  string               `json:"thinking"`
					ToolCalls []ollamaWireToolCall `json:"tool_calls"`
				} `json:"message"`
				Done       bool   `json:"done"`
				DoneReason string `json:"done_reason"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				continue
			}

			out := provider.ChatCompletionChunk{
				ContentDelta: firstNonEmpty(event.Message.Content, event.Message.Thinking),
				FinishReason: event.DoneReason,
			}
			if len(event.Message.ToolCalls) > 0 {
				for i, tc := range event.Message.ToolCalls {
					out.ToolCalls = append(out.ToolCalls, provider.ToolCall{
						ID:        fmt.Sprintf("ollama-tool-%d", i),
						Name:      tc.Function.Name,
						Arguments: decodeOllamaArguments(tc.Function.Arguments),
					})
				}
				if out.FinishReason == "" {
					out.FinishReason = "tool_calls"
				}
			}
			if out.ContentDelta != "" || len(out.ToolCalls) > 0 || out.FinishReason != "" {
				ch <- out
			}
			if event.Done {
				break
			}
		}
		ch <- provider.ChatCompletionChunk{Done: true}
	}()

	return ch, nil
}

func encodeOllamaArguments(args string) json.RawMessage {
	args = strings.TrimSpace(args)
	if args == "" {
		return json.RawMessage(`{}`)
	}
	if json.Valid([]byte(args)) {
		return json.RawMessage(args)
	}
	raw, _ := json.Marshal(args)
	return raw
}

func decodeOllamaArguments(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "{}"
	}
	if raw[0] == '"' {
		var text string
		if err := json.Unmarshal(raw, &text); err == nil {
			text = strings.TrimSpace(text)
			if text == "" {
				return "{}"
			}
			return text
		}
	}
	return string(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// stripBearer removes any accidental Bearer prefix from error strings.
func stripBearer(s string) string {
	if idx := strings.Index(s, "Bearer "); idx >= 0 {
		return s[:idx] + "Bearer [REDACTED]"
	}
	return s
}
