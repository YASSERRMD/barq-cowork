package gemini

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

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type Provider struct {
	timeoutSec int
}

func New(timeoutSec int) *Provider {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &Provider{timeoutSec: timeoutSec}
}

func (p *Provider) Name() string { return "gemini" }

func (p *Provider) ValidateConfig(cfg provider.ProviderConfig) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("gemini: api_key is required")
	}
	if cfg.Model == "" {
		return fmt.Errorf("gemini: model is required")
	}
	return nil
}

func (p *Provider) Chat(ctx context.Context, cfg provider.ProviderConfig, req provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Role  string `json:"role"`
		Parts []part `json:"parts"`
	}

	var contents []content
	var systemInstruction *content
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemInstruction = &content{Parts: []part{{Text: m.Content}}}
			continue
		}
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, content{Role: role, Parts: []part{{Text: m.Content}}})
	}

	body := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"maxOutputTokens": req.MaxTokens,
		},
	}
	if systemInstruction != nil {
		body["system_instruction"] = systemInstruction
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	model := req.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", baseURL, model, cfg.APIKey)

	timeout := time.Duration(p.timeoutSec) * time.Second
	client := &http.Client{Timeout: timeout}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("gemini: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
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
			var event struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
					FinishReason string `json:"finishReason"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			for _, c := range event.Candidates {
				for _, part := range c.Content.Parts {
					if part.Text != "" {
						ch <- provider.ChatCompletionChunk{ContentDelta: part.Text}
					}
				}
				if c.FinishReason == "STOP" {
					ch <- provider.ChatCompletionChunk{Done: true, FinishReason: "stop"}
					return
				}
			}
		}
		ch <- provider.ChatCompletionChunk{Done: true}
	}()

	return ch, nil
}
