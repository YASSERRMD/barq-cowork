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
	result := loop.Run(context.Background(), task, taskWorkspace)
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

	lastMessage := prov.requests[1].Messages[len(prov.requests[1].Messages)-1].Content
	if !strings.Contains(lastMessage, "call write_pptx now") {
		t.Fatalf("expected forced tool reminder in second request, got %q", lastMessage)
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
