package orchestrator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/barq-cowork/barq-cowork/internal/tools"
	"log/slog"
)

type fakeProvider struct {
	requests []provider.ChatCompletionRequest
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) ValidateConfig(provider.ProviderConfig) error { return nil }

func (f *fakeProvider) Chat(_ context.Context, _ provider.ProviderConfig, req provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	f.requests = append(f.requests, req)
	ch := make(chan provider.ChatCompletionChunk, 2)
	switch len(f.requests) {
	case 1:
		ch <- provider.ChatCompletionChunk{ContentDelta: "Here is the plan for the presentation."}
	case 2:
		ch <- provider.ChatCompletionChunk{ToolCalls: []provider.ToolCall{{
			ID:        "call-1",
			Name:      "write_pptx",
			Arguments: `{"filename":"forced-presentation","title":"Forced Presentation","slides":[{"heading":"Intro","type":"bullets","points":["One","Two","Three"]}]}`,
		}}}
	default:
		ch <- provider.ChatCompletionChunk{ContentDelta: "File written."}
	}
	ch <- provider.ChatCompletionChunk{Done: true}
	close(ch)
	return ch, nil
}

type fakePlanStore struct{}

func (fakePlanStore) CreatePlan(context.Context, *domain.Plan) error              { return nil }
func (fakePlanStore) GetPlanByTask(context.Context, string) (*domain.Plan, error) { return nil, nil }
func (fakePlanStore) CreateStep(context.Context, *domain.PlanStep) error          { return nil }
func (fakePlanStore) UpdateStep(context.Context, *domain.PlanStep) error          { return nil }

type fakeArtifactStore struct {
	artifacts []*domain.Artifact
}

func (s *fakeArtifactStore) Create(_ context.Context, a *domain.Artifact) error {
	s.artifacts = append(s.artifacts, a)
	return nil
}

type fakeEventEmitter struct{}

func (fakeEventEmitter) Create(context.Context, *domain.Event) error { return nil }

type fakeWritePPTXTool struct{}

func (fakeWritePPTXTool) Name() string { return "write_pptx" }

func (fakeWritePPTXTool) Description() string { return "fake write_pptx" }

func (fakeWritePPTXTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }

func (fakeWritePPTXTool) Execute(context.Context, tools.InvocationContext, string) tools.Result {
	return tools.OKData("ok", map[string]any{"path": "slides/forced-presentation.pptx", "size": int64(1234)})
}

func TestAgentLoop_NudgesPresentationTaskToCallWritePPTX(t *testing.T) {
	prov := &fakeProvider{}
	registry := tools.NewRegistry()
	registry.Register(fakeWritePPTXTool{})
	artifacts := &fakeArtifactStore{}
	loop := NewAgentLoop(
		prov,
		provider.ProviderConfig{Model: "fake"},
		registry,
		fakePlanStore{},
		artifacts,
		fakeEventEmitter{},
		slog.New(slog.NewTextHandler(testWriter{t}, nil)),
	)

	task := &domain.Task{
		ID:          "task-1",
		Title:       "Create a presentation about kids and AI",
		Description: "Need a PowerPoint deck with six slides",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	taskWorkspace := t.TempDir()
	result := loop.Run(context.Background(), task, taskWorkspace, "Active skill prompt:\nUse a professional proposal deck.", "Project instructions:\nDo not use visible slide counters.")
	if result.Completed != 1 {
		t.Fatalf("expected one completed tool call, got %+v", result)
	}
	if len(artifacts.artifacts) != 1 {
		t.Fatalf("expected one recorded artifact, got %d", len(artifacts.artifacts))
	}
	if got := artifacts.artifacts[0].ContentPath; got != filepath.Join(taskWorkspace, "slides", "forced-presentation.pptx") {
		t.Fatalf("expected absolute artifact path, got %q", got)
	}
	if len(prov.requests) < 2 {
		t.Fatalf("expected at least two provider calls, got %d", len(prov.requests))
	}
	firstMessages := prov.requests[0].Messages
	if len(firstMessages) < 3 {
		t.Fatalf("expected extra system prompts in first request, got %d messages", len(firstMessages))
	}
	if prov.requests[0].MaxTokens != 8192 {
		t.Fatalf("expected larger max token budget for presentation tasks, got %d", prov.requests[0].MaxTokens)
	}
	if !strings.Contains(firstMessages[1].Content, "Active skill prompt") || !strings.Contains(firstMessages[2].Content, "Project instructions") {
		t.Fatalf("expected injected skill and project prompts, got %+v", firstMessages)
	}

	lastMessage := prov.requests[1].Messages[len(prov.requests[1].Messages)-1].Content
	if !strings.Contains(lastMessage, "MUST be exactly one tool call to write_pptx") {
		t.Fatalf("expected forced tool reminder in second request, got %q", lastMessage)
	}
}

type fakeNoToolProvider struct {
	requests []provider.ChatCompletionRequest
}

func (f *fakeNoToolProvider) Name() string { return "fake-no-tool" }

func (f *fakeNoToolProvider) ValidateConfig(provider.ProviderConfig) error { return nil }

func (f *fakeNoToolProvider) Chat(_ context.Context, _ provider.ProviderConfig, req provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	f.requests = append(f.requests, req)
	ch := make(chan provider.ChatCompletionChunk, 2)
	ch <- provider.ChatCompletionChunk{ContentDelta: "Still planning the deck."}
	ch <- provider.ChatCompletionChunk{Done: true}
	close(ch)
	return ch, nil
}

func TestAgentLoop_FailsPresentationTaskWithoutRequiredToolCall(t *testing.T) {
	prov := &fakeNoToolProvider{}
	registry := tools.NewRegistry()
	registry.Register(fakeWritePPTXTool{})
	loop := NewAgentLoop(
		prov,
		provider.ProviderConfig{Model: "fake"},
		registry,
		fakePlanStore{},
		&fakeArtifactStore{},
		fakeEventEmitter{},
		slog.New(slog.NewTextHandler(testWriter{t}, nil)),
	)

	task := &domain.Task{
		ID:          "task-2",
		Title:       "Create a presentation about AI in healthcare",
		Description: "Need a PowerPoint deck",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	result := loop.Run(context.Background(), task, t.TempDir())
	if result.Completed != 0 || result.Failed == 0 {
		t.Fatalf("expected failure when write_pptx is never called, got %+v", result)
	}
	if len(prov.requests) != 5 {
		t.Fatalf("expected four forced retries plus final failure turn, got %d", len(prov.requests))
	}
}

type fakeJSONArgsProvider struct {
	requests []provider.ChatCompletionRequest
}

func (f *fakeJSONArgsProvider) Name() string { return "fake-json-args" }

func (f *fakeJSONArgsProvider) ValidateConfig(provider.ProviderConfig) error { return nil }

func (f *fakeJSONArgsProvider) Chat(_ context.Context, _ provider.ProviderConfig, req provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	f.requests = append(f.requests, req)
	ch := make(chan provider.ChatCompletionChunk, 2)
	switch len(f.requests) {
	case 1:
		ch <- provider.ChatCompletionChunk{ContentDelta: `{"filename":"local-deck","title":"Local Deck","deck":{"subject":"Local Deck","audience":"operators","narrative":"issue -> action","theme":"data","visual_style":"concise dashboard","cover_style":"editorial","color_story":"clean slate","motif":"chart","kicker":"Local mode","theme_css":".deck{display:grid;gap:18px}.panel{padding:24px;border:1px solid var(--border)}.hero{display:grid;gap:20px}.tag{display:inline-flex;padding:8px 12px}","cover_html":"<div class='deck' style='padding:96px'><div class='eyebrow'>LOCAL MODE</div><h1 class='display-title'>Local Deck</h1><div class='panel'>Operators need a clean handoff path.</div></div>","palette":{"background":"F8FAFC","card":"FFFFFF","accent":"0EA5E9","accent2":"38BDF8","text":"0F172A","muted":"64748B","border":"CBD5E1"}},"slides":[{"heading":"Operating signal","type":"html","html":"<div class='hero' style='padding:96px'><div class='eyebrow'>SIGNAL</div><h2 class='section-title'>Operating signal</h2><div class='panel'>One JSON handoff is enough for weak local models.</div></div>"}]}`}
	default:
		ch <- provider.ChatCompletionChunk{ContentDelta: "done"}
	}
	ch <- provider.ChatCompletionChunk{Done: true}
	close(ch)
	return ch, nil
}

func TestAgentLoop_RecoversToolArgsFromAssistantJSON(t *testing.T) {
	prov := &fakeJSONArgsProvider{}
	registry := tools.NewRegistry()
	registry.Register(fakeWritePPTXTool{})
	artifacts := &fakeArtifactStore{}
	loop := NewAgentLoop(
		prov,
		provider.ProviderConfig{ProviderName: "ollama", Model: "qwen2.5-coder:7b"},
		registry,
		fakePlanStore{},
		artifacts,
		fakeEventEmitter{},
		slog.New(slog.NewTextHandler(testWriter{t}, nil)),
	)

	task := &domain.Task{
		ID:          "task-3",
		Title:       "Create a presentation about local operations",
		Description: "Need a PowerPoint deck",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	result := loop.Run(context.Background(), task, t.TempDir())
	if result.Completed != 1 || result.Failed != 0 {
		t.Fatalf("expected recovered JSON args to complete one tool call, got %+v", result)
	}
	if len(artifacts.artifacts) != 1 {
		t.Fatalf("expected one recorded artifact, got %d", len(artifacts.artifacts))
	}
	if len(prov.requests) == 0 {
		t.Fatalf("expected provider request")
	}
	foundCompatibilityPrompt := false
	for _, msg := range prov.requests[0].Messages {
		if strings.Contains(msg.Content, "Model compatibility mode") {
			foundCompatibilityPrompt = true
			break
		}
	}
	if !foundCompatibilityPrompt {
		t.Fatalf("expected weak-model compatibility prompt in first request, got %+v", prov.requests[0].Messages)
	}
}

func TestResolveArtifactContentPath(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "barq-workspace")
	got := resolveArtifactContentPath(root, "slides/deck.pptx")
	want := filepath.Join(root, "slides", "deck.pptx")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	abs := filepath.Join(root, "documents", "brief.docx")
	if got := resolveArtifactContentPath(root, abs); got != abs {
		t.Fatalf("expected absolute path passthrough, got %q", got)
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimSpace(string(p)))
	return len(p), nil
}
