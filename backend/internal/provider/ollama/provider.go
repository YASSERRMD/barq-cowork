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

	type ollamaMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var messages []ollamaMsg
	for _, m := range req.Messages {
		messages = append(messages, ollamaMsg{Role: m.Role, Content: m.Content})
	}

	body := map[string]any{
		"model":    req.Model,
		"messages": messages,
		"stream":   true,
		"options": map[string]any{
			"num_predict": req.MaxTokens,
		},
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	timeout := time.Duration(p.timeoutSec) * time.Second
	client := &http.Client{Timeout: timeout}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/chat", bytes.NewReader(rawBody))
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
		for scanner.Scan() {
			var event struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				continue
			}
			if event.Message.Content != "" {
				ch <- provider.ChatCompletionChunk{ContentDelta: event.Message.Content}
			}
			if event.Done {
				break
			}
		}
		ch <- provider.ChatCompletionChunk{Done: true}
	}()

	return ch, nil
}

// stripBearer removes any accidental Bearer prefix from error strings.
func stripBearer(s string) string {
	if idx := strings.Index(s, "Bearer "); idx >= 0 {
		return s[:idx] + "Bearer [REDACTED]"
	}
	return s
}
