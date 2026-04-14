package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/barq-cowork/barq-cowork/internal/provider"
)

func TestProvider_SendsToolsAndParsesToolCalls(t *testing.T) {
	var captured map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"","tool_calls":[{"function":{"name":"write_file","arguments":{"path":"reports/out.md","content":"hello"}}}]},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":""},"done":true,"done_reason":"tool_calls"}` + "\n"))
	}))
	defer srv.Close()

	p := New(10)
	ch, err := p.Chat(context.Background(), provider.ProviderConfig{
		ProviderName: "ollama",
		BaseURL:      srv.URL,
		Model:        "qwen2.5-coder:7b",
	}, provider.ChatCompletionRequest{
		Model:  "qwen2.5-coder:7b",
		Stream: true,
		Tools: []provider.ToolDefinition{
			{
				Name:        "write_file",
				Description: "write a file",
				InputSchema: map[string]any{"type": "object"},
			},
		},
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "use tools"},
			{Role: "user", Content: "create a file"},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	var toolCalls []provider.ToolCall
	for chunk := range ch {
		toolCalls = append(toolCalls, chunk.ToolCalls...)
	}

	if len(toolCalls) != 1 {
		t.Fatalf("expected one tool call, got %+v", toolCalls)
	}
	if toolCalls[0].Name != "write_file" {
		t.Fatalf("unexpected tool call name: %+v", toolCalls[0])
	}
	if !strings.Contains(toolCalls[0].Arguments, `"path":"reports/out.md"`) {
		t.Fatalf("unexpected tool call arguments: %s", toolCalls[0].Arguments)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected one tool in request, got %+v", captured["tools"])
	}
}
