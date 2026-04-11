package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/provider"
)

const defaultBaseURL = "https://api.anthropic.com/v1"
const anthropicVersion = "2023-06-01"

// Provider implements the Anthropic Messages API.
type Provider struct {
	timeoutSec int
}

func New(timeoutSec int) *Provider {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &Provider{timeoutSec: timeoutSec}
}

func (p *Provider) Name() string { return "anthropic" }

func (p *Provider) ValidateConfig(cfg provider.ProviderConfig) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("anthropic: api_key is required")
	}
	if cfg.Model == "" {
		return fmt.Errorf("anthropic: model is required")
	}
	return nil
}

func (p *Provider) Chat(ctx context.Context, cfg provider.ProviderConfig, req provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	// Convert messages
	type anthropicMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var systemPrompt string
	var messages []anthropicMsg
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		role := m.Role
		if role == "tool" {
			role = "user"
		}
		messages = append(messages, anthropicMsg{Role: role, Content: m.Content})
	}

	body := map[string]any{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": req.MaxTokens,
		"stream":     true,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	timeout := time.Duration(p.timeoutSec) * time.Second
	client := &http.Client{Timeout: timeout}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/messages", bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("anthropic: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", cfg.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	ch := make(chan provider.ChatCompletionChunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
				ch <- provider.ChatCompletionChunk{ContentDelta: event.Delta.Text}
			}
			if event.Type == "message_stop" {
				break
			}
		}
		ch <- provider.ChatCompletionChunk{Done: true}
	}()

	return ch, nil
}
