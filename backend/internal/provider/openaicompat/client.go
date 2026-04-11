// Package openaicompat implements a shared HTTP client for any provider
// that exposes an OpenAI-compatible /chat/completions endpoint with SSE streaming.
package openaicompat

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

// ─────────────────────────────────────────────
// Wire types (OpenAI JSON shapes)
// ─────────────────────────────────────────────

type chatRequest struct {
	Model       string          `json:"model"`
	Messages    []wireMessage   `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream"`
	Tools       []wireTool      `json:"tools,omitempty"`
}

type wireMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type wireTool struct {
	Type     string           `json:"type"` // always "function"
	Function wireToolFunction `json:"function"`
}

type wireToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// SSE chunk shapes
type sseChunk struct {
	Choices []sseChoice `json:"choices"`
}

type sseChoice struct {
	Delta        sseDelta `json:"delta"`
	FinishReason string   `json:"finish_reason"`
}

type sseDelta struct {
	Content   string          `json:"content"`
	ToolCalls []sseToolCall   `json:"tool_calls"`
}

type sseToolCall struct {
	Index    int             `json:"index"`
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function sseToolFunction `json:"function"`
}

type sseToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Non-streaming response shape
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content   string        `json:"content"`
			ToolCalls []sseToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *openAIError `json:"error"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"` // can be string or int
}

// ─────────────────────────────────────────────
// Client
// ─────────────────────────────────────────────

// Client is a reusable HTTP client for OpenAI-compatible APIs.
type Client struct {
	http *http.Client
}

// NewClient creates a Client with the given timeout.
func NewClient(timeoutSec int) *Client {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &Client{
		http: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Chat sends a completion request. If cfg.Stream is false it buffers the full
// response into a single chunk. Either way it returns a channel of chunks.
func (c *Client) Chat(
	ctx context.Context,
	cfg provider.ProviderConfig,
	req provider.ChatCompletionRequest,
) (<-chan provider.ChatCompletionChunk, error) {
	wireReq := buildRequest(cfg, req)

	body, err := json.Marshal(wireReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	// Authorization header — key is resolved in the service layer, never logged.
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	for k, v := range cfg.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}
	if req.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Cache-Control", "no-cache")
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, mapNetworkError(err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, mapHTTPError(resp)
	}

	if req.Stream {
		return streamSSE(ctx, resp), nil
	}
	return readFull(resp), nil
}

// ─────────────────────────────────────────────
// SSE streaming reader
// ─────────────────────────────────────────────

func streamSSE(ctx context.Context, resp *http.Response) <-chan provider.ChatCompletionChunk {
	ch := make(chan provider.ChatCompletionChunk, 32)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// Accumulate partial tool-call arguments across chunks by index.
		toolCalls := map[int]*provider.ToolCall{}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*64), 1024*64)

		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- provider.ChatCompletionChunk{Done: true}
				return
			}

			var chunk sseChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue // skip malformed lines
			}
			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]
			out := provider.ChatCompletionChunk{
				ContentDelta: choice.Delta.Content,
				FinishReason: choice.FinishReason,
			}

			// Assemble streaming tool-call fragments.
			for _, tc := range choice.Delta.ToolCalls {
				if tc.ID != "" {
					toolCalls[tc.Index] = &provider.ToolCall{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					}
				} else if existing, ok := toolCalls[tc.Index]; ok {
					existing.Arguments += tc.Function.Arguments
				}
			}

			if choice.FinishReason == "tool_calls" {
				for _, tc := range toolCalls {
					out.ToolCalls = append(out.ToolCalls, *tc)
				}
			}

			if out.ContentDelta != "" || len(out.ToolCalls) > 0 || out.FinishReason != "" {
				ch <- out
			}
		}

		// Scanner ended without [DONE] — send Done marker anyway.
		ch <- provider.ChatCompletionChunk{Done: true}
	}()

	return ch
}

// ─────────────────────────────────────────────
// Non-streaming reader
// ─────────────────────────────────────────────

func readFull(resp *http.Response) <-chan provider.ChatCompletionChunk {
	ch := make(chan provider.ChatCompletionChunk, 2)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		var result chatResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			ch <- provider.ChatCompletionChunk{Done: true}
			return
		}
		if result.Error != nil {
			ch <- provider.ChatCompletionChunk{Done: true}
			return
		}
		if len(result.Choices) > 0 {
			c := result.Choices[0]
			var tcs []provider.ToolCall
			for _, tc := range c.Message.ToolCalls {
				tcs = append(tcs, provider.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}
			ch <- provider.ChatCompletionChunk{
				ContentDelta: c.Message.Content,
				ToolCalls:    tcs,
				FinishReason: c.FinishReason,
			}
		}
		ch <- provider.ChatCompletionChunk{Done: true}
	}()
	return ch
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func buildRequest(cfg provider.ProviderConfig, req provider.ChatCompletionRequest) chatRequest {
	msgs := make([]wireMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = wireMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
	}

	var tools []wireTool
	for _, t := range req.Tools {
		tools = append(tools, wireTool{
			Type: "function",
			Function: wireToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	model := req.Model
	if model == "" {
		model = cfg.Model
	}

	return chatRequest{
		Model:       model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Tools:       tools,
	}
}

// mapHTTPError converts a non-200 response to a descriptive error.
// API keys are never included in error messages.
func mapHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var apiErr struct {
		Error *openAIError `json:"error"`
	}
	_ = json.Unmarshal(body, &apiErr)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("provider auth failed (HTTP 401) — check your API key env var")
	case http.StatusForbidden:
		return fmt.Errorf("provider access denied (HTTP 403)")
	case http.StatusTooManyRequests:
		return fmt.Errorf("provider rate limit exceeded (HTTP 429)")
	case http.StatusNotFound:
		return fmt.Errorf("provider endpoint not found (HTTP 404) — check base_url")
	case http.StatusUnprocessableEntity:
		if apiErr.Error != nil {
			return fmt.Errorf("provider rejected request: %s", apiErr.Error.Message)
		}
		return fmt.Errorf("provider rejected request (HTTP 422)")
	default:
		if apiErr.Error != nil {
			return fmt.Errorf("provider error (HTTP %d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return fmt.Errorf("provider error (HTTP %d)", resp.StatusCode)
	}
}

func mapNetworkError(err error) error {
	if strings.Contains(err.Error(), "context canceled") {
		return fmt.Errorf("request cancelled")
	}
	if strings.Contains(err.Error(), "context deadline exceeded") {
		return fmt.Errorf("provider request timed out")
	}
	return fmt.Errorf("network error: %w", err)
}
