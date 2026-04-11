package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxFetchBytes   = 512 * 1024 // 512 KB response cap
	fetchTimeoutSec = 30
)

// HTTPFetchTool fetches the content of a URL. POST requires approval.
type HTTPFetchTool struct {
	client *http.Client
}

// NewHTTPFetchTool creates an HTTPFetchTool with a default timeout client.
func NewHTTPFetchTool() *HTTPFetchTool {
	return &HTTPFetchTool{
		client: &http.Client{Timeout: fetchTimeoutSec * time.Second},
	}
}

func (t *HTTPFetchTool) Name() string        { return "http_fetch" }
func (t *HTTPFetchTool) Description() string {
	return "Fetch the content of a URL via HTTP GET (or POST with approval). " +
		"Returns the response body as text. Maximum response size: 512 KB."
}
func (t *HTTPFetchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":     map[string]any{"type": "string", "description": "The URL to fetch"},
			"method":  map[string]any{"type": "string", "enum": []string{"GET", "POST"}, "description": "HTTP method (default: GET)"},
			"body":    map[string]any{"type": "string", "description": "Request body for POST"},
			"headers": map[string]any{"type": "object", "description": "Additional request headers"},
		},
		"required": []string{"url"},
	}
}

type httpFetchArgs struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

func (t *HTTPFetchTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args httpFetchArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.URL == "" {
		return Err("url is required")
	}
	if args.Method == "" {
		args.Method = "GET"
	}
	args.Method = strings.ToUpper(args.Method)

	// POST requires approval — it can have side effects.
	if args.Method == "POST" {
		action := fmt.Sprintf("HTTP POST to %s", args.URL)
		if ictx.RequireApproval != nil && !ictx.RequireApproval(ctx, action, argsJSON) {
			return Denied(action)
		}
	}

	var bodyReader io.Reader
	if args.Body != "" {
		bodyReader = strings.NewReader(args.Body)
	}

	req, err := http.NewRequestWithContext(ctx, args.Method, args.URL, bodyReader)
	if err != nil {
		return Err("build request: %v", err)
	}
	for k, v := range args.Headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return Err("fetch: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes))
	if err != nil {
		return Err("read response: %v", err)
	}

	summary := fmt.Sprintf("Fetched %s — HTTP %d — %d bytes", args.URL, resp.StatusCode, len(data))
	return OKData(summary, map[string]any{
		"url":         args.URL,
		"status_code": resp.StatusCode,
		"body":        string(data),
		"size":        len(data),
	})
}
