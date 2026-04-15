package orchestrator

import (
	"context"
	"fmt"
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

type fakePlanStore struct {
	plan  *domain.Plan
	steps []*domain.PlanStep
}

func (s *fakePlanStore) CreatePlan(_ context.Context, plan *domain.Plan) error {
	clone := *plan
	s.plan = &clone
	return nil
}

func (s *fakePlanStore) GetPlanByTask(context.Context, string) (*domain.Plan, error) {
	if s.plan == nil {
		return nil, nil
	}
	clone := *s.plan
	clone.Steps = append([]*domain.PlanStep(nil), s.steps...)
	return &clone, nil
}

func (s *fakePlanStore) CreateStep(_ context.Context, step *domain.PlanStep) error {
	clone := *step
	s.steps = append(s.steps, &clone)
	return nil
}

func (s *fakePlanStore) UpdateStep(_ context.Context, step *domain.PlanStep) error {
	for i, existing := range s.steps {
		if existing.ID != step.ID {
			continue
		}
		clone := *step
		s.steps[i] = &clone
		return nil
	}
	clone := *step
	s.steps = append(s.steps, &clone)
	return nil
}

type fakeArtifactStore struct {
	artifacts []*domain.Artifact
}

func (s *fakeArtifactStore) Create(_ context.Context, a *domain.Artifact) error {
	s.artifacts = append(s.artifacts, a)
	return nil
}

type fakeEventEmitter struct{}

func (fakeEventEmitter) Create(context.Context, *domain.Event) error { return nil }

type recordingEventEmitter struct {
	events []*domain.Event
}

func (r *recordingEventEmitter) Create(_ context.Context, event *domain.Event) error {
	clone := *event
	r.events = append(r.events, &clone)
	return nil
}

type fakeWritePPTXTool struct{}

func (fakeWritePPTXTool) Name() string { return "write_pptx" }

func (fakeWritePPTXTool) Description() string { return "fake write_pptx" }

func (fakeWritePPTXTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }

func (fakeWritePPTXTool) Execute(context.Context, tools.InvocationContext, string) tools.Result {
	return tools.OKData("ok", map[string]any{"path": "slides/forced-presentation.pptx", "size": int64(1234)})
}

type fakeSegmentedProvider struct {
	requests []provider.ChatCompletionRequest
}

func (f *fakeSegmentedProvider) Name() string { return "fake-segmented" }

func (f *fakeSegmentedProvider) ValidateConfig(provider.ProviderConfig) error { return nil }

func (f *fakeSegmentedProvider) Chat(_ context.Context, _ provider.ProviderConfig, req provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	f.requests = append(f.requests, req)
	ch := make(chan provider.ChatCompletionChunk, 2)
	if len(f.requests) == 1 {
		ch <- provider.ChatCompletionChunk{ContentDelta: `{
			"filename":"gen-ai-kids",
			"title":"Generative AI for Kids",
			"subtitle":"A safe learning guide",
			"deck":{
				"subject":"Generative AI for Kids",
				"audience":"parents and educators",
				"narrative":"curiosity -> safe use -> practice",
				"theme":"education",
				"visual_style":"bright modern learning cards",
				"cover_style":"editorial classroom",
				"color_story":"warm paper with blue and green accents",
				"motif":"stars",
				"kicker":"Learning guide",
				"palette":{"background":"F6F8FB","card":"FFFFFF","accent":"0EA5E9","accent2":"14B8A6","text":"0F172A","muted":"475569","border":"CBD5E1"},
				"theme_css":".row{display:flex;gap:18px}.card{border:1px solid var(--border);border-radius:22px}.card-body{padding:24px}.badge{border-radius:999px}.list-group-item{border-left:5px solid var(--accent)}",
				"cover_html":"<div class='container-fluid h-100 d-grid gap-4'><span class='badge'>Learning guide</span><h1 class='display-title'>Generative AI for Kids</h1><p class='lead'>A safe and practical introduction for families and classrooms.</p></div>"
			},
			"slides":[
				{"heading":"What Gen AI Does","type":"cards","purpose":"explain the idea","visual":"three learning cards","icon":"stars","points":["It learns patterns from examples","It creates text, images, and ideas","Children need guided practice"]},
				{"heading":"Safe Practice Rules","type":"bullets","purpose":"teach guardrails","visual":"checklist","icon":"shield-check","points":["Ask an adult before sharing personal details","Check answers against trusted sources","Use AI as a helper, not a replacement for thinking"]}
			]
		}`}
	} else {
		idx := len(f.requests) - 1
		ch <- provider.ChatCompletionChunk{ContentDelta: fmt.Sprintf(`{"heading":"Drafted Slide %d","type":"html","html":"<div class='container-fluid h-100 d-grid gap-4'><div class='badge'>Learning</div><h2 class='display-4'>Drafted Slide %d</h2><div class='row'><div class='col-6'><div class='card'><div class='card-body'><span class='icon-badge'><i class='bi bi-stars' aria-hidden='true'></i></span><h3 class='card-title'>Core idea</h3><p class='card-text'>This slide has concrete readable content for the final PowerPoint export.</p></div></div></div><div class='col-6'><div class='card'><div class='card-body'><h3 class='card-title'>Classroom use</h3><p class='card-text'>The layout contains enough information density and no empty filler area.</p></div></div></div></div></div>"}`, idx, idx)}
	}
	ch <- provider.ChatCompletionChunk{Done: true}
	close(ch)
	return ch, nil
}

func TestAgentLoop_SegmentsExplicitSlideCountPresentation(t *testing.T) {
	prov := &fakeSegmentedProvider{}
	registry := tools.NewRegistry()
	registry.Register(fakeWritePPTXTool{})
	artifacts := &fakeArtifactStore{}
	plans := &fakePlanStore{}
	events := &recordingEventEmitter{}
	loop := NewAgentLoop(
		prov,
		provider.ProviderConfig{Model: "fake"},
		registry,
		plans,
		artifacts,
		events,
		slog.New(slog.NewTextHandler(testWriter{t}, nil)),
	)

	task := &domain.Task{
		ID:          "task-segmented",
		Title:       "3-slide presentation about Generative AI for kids",
		Description: "Show every slide while generating.",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	result := loop.Run(context.Background(), task, t.TempDir())
	if result.Completed != 1 || result.Failed != 0 {
		t.Fatalf("expected segmented presentation to render once, got %+v", result)
	}
	if len(prov.requests) != 3 {
		t.Fatalf("expected one plan call and one call per content slide, got %d", len(prov.requests))
	}
	if prov.requests[0].ResponseFormat == nil || prov.requests[0].ForceToolName != "" || len(prov.requests[0].Tools) != 0 {
		t.Fatalf("expected JSON response-format call without tools, got %+v", prov.requests[0])
	}
	if len(artifacts.artifacts) != 1 {
		t.Fatalf("expected one artifact, got %d", len(artifacts.artifacts))
	}
	draftEvents := 0
	for _, event := range events.events {
		if event.Type == domain.EventTypePresentationSlide {
			draftEvents++
			if strings.Contains(event.Payload, "Slide 1 of") {
				t.Fatalf("draft should not include visible slide counter payload: %s", event.Payload)
			}
		}
	}
	if draftEvents != 3 {
		t.Fatalf("expected cover plus two slide draft events, got %d", draftEvents)
	}
	if len(plans.steps) != 4 {
		t.Fatalf("expected plan, two slide drafts, and render step, got %d", len(plans.steps))
	}
}

func TestAgentLoop_NudgesPresentationTaskToCallWritePPTX(t *testing.T) {
	prov := &fakeProvider{}
	registry := tools.NewRegistry()
	registry.Register(fakeWritePPTXTool{})
	artifacts := &fakeArtifactStore{}
	plans := &fakePlanStore{}
	events := &recordingEventEmitter{}
	loop := NewAgentLoop(
		prov,
		provider.ProviderConfig{Model: "fake"},
		registry,
		plans,
		artifacts,
		events,
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
	if prov.requests[0].MaxTokens != 16384 {
		t.Fatalf("expected larger max token budget for presentation tasks, got %d", prov.requests[0].MaxTokens)
	}
	if prov.requests[0].ForceToolName != "write_pptx" || len(prov.requests[0].Tools) != 1 || prov.requests[0].Tools[0].Name != "write_pptx" {
		t.Fatalf("expected presentation request to force only write_pptx, got force=%q tools=%+v", prov.requests[0].ForceToolName, prov.requests[0].Tools)
	}
	if !strings.Contains(firstMessages[1].Content, "Active skill prompt") || !strings.Contains(firstMessages[2].Content, "Project instructions") {
		t.Fatalf("expected injected skill and project prompts, got %+v", firstMessages)
	}

	lastMessage := prov.requests[1].Messages[len(prov.requests[1].Messages)-1].Content
	if !strings.Contains(lastMessage, "MUST be exactly one tool call to write_pptx") {
		t.Fatalf("expected forced tool reminder in second request, got %q", lastMessage)
	}
	if len(plans.steps) != 4 {
		t.Fatalf("expected slide-by-slide plan steps for pptx generation, got %d", len(plans.steps))
	}
	if plans.steps[0].Title != "Plan deck system" || plans.steps[1].Title != "Draft cover slide" {
		t.Fatalf("expected deck planning steps, got %+v", plans.steps[:2])
	}
	if plans.steps[2].Title != "Draft slide 1" {
		t.Fatalf("expected authored slide step, got %q", plans.steps[2].Title)
	}
	if plans.steps[3].Title != "Render PowerPoint file" || plans.steps[3].Status != domain.StepStatusCompleted {
		t.Fatalf("expected final render step to complete, got %+v", plans.steps[3])
	}
	foundSlideDraft := false
	for _, event := range events.events {
		if event.Type != domain.EventTypePresentationSlide {
			continue
		}
		foundSlideDraft = true
		if !strings.Contains(event.Payload, "Intro") || !strings.Contains(event.Payload, "One") {
			t.Fatalf("expected slide draft event to include fallback slide content, got %s", event.Payload)
		}
	}
	if !foundSlideDraft {
		t.Fatalf("expected a live presentation slide draft event")
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
		&fakePlanStore{},
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
		&fakePlanStore{},
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

func TestRequestedPresentationDeckHint_HonorsExplicitTotalSlideCount(t *testing.T) {
	task := &domain.Task{
		Title:       "3 slide presentation on islamic parenting",
		Description: "Keep it concise.",
	}
	hint := requestedPresentationDeckHint(task)
	if !strings.Contains(hint, "3 total slides") || !strings.Contains(hint, "exactly 2 content slides") {
		t.Fatalf("expected total-slide hint, got %q", hint)
	}
}

func TestRequestedPresentationDeckHint_HonorsExplicitContentSlideCount(t *testing.T) {
	task := &domain.Task{
		Title:       "Islamic parenting deck",
		Description: "Need 3 content slides plus cover.",
	}
	hint := requestedPresentationDeckHint(task)
	if !strings.Contains(hint, "3 content slides") || !strings.Contains(hint, "exactly 3 entries") {
		t.Fatalf("expected content-slide hint, got %q", hint)
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimSpace(string(p)))
	return len(p), nil
}
