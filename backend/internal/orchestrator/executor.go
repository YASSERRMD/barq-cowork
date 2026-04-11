package orchestrator

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/tools"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────
// Storage interfaces (ports)
// ─────────────────────────────────────────────

// PlanStore is the persistence interface the Executor needs.
type PlanStore interface {
	CreatePlan(ctx context.Context, p *domain.Plan) error
	GetPlanByTask(ctx context.Context, taskID string) (*domain.Plan, error)
	CreateStep(ctx context.Context, step *domain.PlanStep) error
	UpdateStep(ctx context.Context, step *domain.PlanStep) error
}

// ArtifactStore is the persistence interface the Executor uses to record outputs.
type ArtifactStore interface {
	Create(ctx context.Context, a *domain.Artifact) error
}

// EventEmitter lets the Executor record structured events.
type EventEmitter interface {
	Create(ctx context.Context, e *domain.Event) error
}

// ─────────────────────────────────────────────
// Executor
// ─────────────────────────────────────────────

// Executor runs plan steps sequentially, dispatches tools, and records results.
type Executor struct {
	plans     PlanStore
	artifacts ArtifactStore
	events    EventEmitter
	registry  *tools.Registry
	logger    *slog.Logger
}

// NewExecutor creates an Executor.
func NewExecutor(
	plans PlanStore,
	artifacts ArtifactStore,
	events EventEmitter,
	registry *tools.Registry,
	logger *slog.Logger,
) *Executor {
	return &Executor{
		plans:     plans,
		artifacts: artifacts,
		events:    events,
		registry:  registry,
		logger:    logger,
	}
}

// ExecuteResult summarises an execution run.
type ExecuteResult struct {
	Completed int
	Failed    int
	Skipped   int
}

// Execute runs all steps of the given plan sequentially.
// workspaceRoot gates filesystem tool access.
// requireApproval controls whether destructive tools are approval-gated.
func (e *Executor) Execute(
	ctx context.Context,
	plan *domain.Plan,
	task *domain.Task,
	workspaceRoot string,
	requireApproval bool,
) ExecuteResult {
	var result ExecuteResult

	for _, step := range plan.Steps {
		if ctx.Err() != nil {
			// Context cancelled — skip remaining steps
			e.markStep(ctx, step, domain.StepStatusSkipped, `{"reason":"context cancelled"}`)
			result.Skipped++
			continue
		}

		e.logger.Info("executing step", "task_id", task.ID, "step", step.Order, "title", step.Title)
		e.emitEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
			"step_id": step.ID, "step_order": step.Order, "title": step.Title,
		})

		now := time.Now().UTC()
		step.StartedAt = &now
		step.Status = domain.StepStatusRunning
		_ = e.plans.UpdateStep(ctx, step)

		var toolOutput string
		var stepErr error

		if step.ToolName != "" {
			toolOutput, stepErr = e.runTool(ctx, step, task, workspaceRoot, requireApproval)
		} else {
			// Pure reasoning/thinking step — mark complete immediately
			toolOutput = `{"status":"ok","content":"reasoning step completed"}`
		}

		completed := time.Now().UTC()
		step.CompletedAt = &completed

		if stepErr != nil {
			step.Status = domain.StepStatusFailed
			step.ToolOutput = marshalError(stepErr)
			result.Failed++
		} else {
			step.Status = domain.StepStatusCompleted
			step.ToolOutput = toolOutput
			result.Completed++

			// Detect artifact-producing tools and record them
			e.maybeRecordArtifact(ctx, step, task)
		}

		_ = e.plans.UpdateStep(ctx, step)

		e.emitEvent(ctx, task.ID, domain.EventTypeStepCompleted, map[string]any{
			"step_id": step.ID, "step_order": step.Order,
			"status": step.Status, "tool": step.ToolName,
		})
	}

	return result
}

// ─────────────────────────────────────────────
// Tool dispatch
// ─────────────────────────────────────────────

func (e *Executor) runTool(
	ctx context.Context,
	step *domain.PlanStep,
	task *domain.Task,
	workspaceRoot string,
	requireApproval bool,
) (string, error) {
	t, ok := e.registry.Get(step.ToolName)
	if !ok {
		r := tools.Err("tool not found: %s", step.ToolName)
		return r.ToJSON(), nil // not a fatal error — step just fails
	}

	ictx := tools.InvocationContext{
		WorkspaceRoot: workspaceRoot,
		TaskID:        task.ID,
	}

	if requireApproval {
		// In Phase 5 the synchronous approval guard creates an approval record
		// and denies immediately so the user can review it in the Approvals UI.
		// Phase 6 will add async waiting.
		ictx.RequireApproval = func(_ context.Context, _, _ string) bool { return false }
	}

	result := t.Execute(ctx, ictx, step.ToolInput)
	return result.ToJSON(), nil
}

// ─────────────────────────────────────────────
// Artifact detection
// ─────────────────────────────────────────────

var artifactTools = map[string]domain.ArtifactType{
	"write_markdown_report": domain.ArtifactTypeMarkdown,
	"export_json":           domain.ArtifactTypeJSON,
	"write_file":            domain.ArtifactTypeFile,
}

func (e *Executor) maybeRecordArtifact(ctx context.Context, step *domain.PlanStep, task *domain.Task) {
	artType, ok := artifactTools[step.ToolName]
	if !ok {
		return
	}

	// Parse tool output to extract path and size
	var output struct {
		Data struct {
			Path string `json:"path"`
			Size int64  `json:"size"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(step.ToolOutput), &output); err != nil || output.Data.Path == "" {
		return
	}

	artifact := &domain.Artifact{
		ID:          uuid.NewString(),
		TaskID:      task.ID,
		ProjectID:   task.ProjectID,
		Name:        output.Data.Path,
		Type:        artType,
		ContentPath: output.Data.Path,
		Size:        output.Data.Size,
		CreatedAt:   time.Now().UTC(),
	}
	if err := e.artifacts.Create(ctx, artifact); err != nil {
		e.logger.Warn("failed to record artifact", "error", err)
		return
	}

	e.emitEvent(ctx, task.ID, domain.EventTypeArtifactReady, map[string]any{
		"artifact_id": artifact.ID,
		"name":        artifact.Name,
		"type":        artType,
	})
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func (e *Executor) markStep(ctx context.Context, step *domain.PlanStep, status domain.StepStatus, output string) {
	step.Status = status
	step.ToolOutput = output
	_ = e.plans.UpdateStep(ctx, step)
}

func (e *Executor) emitEvent(ctx context.Context, taskID string, t domain.EventType, data map[string]any) {
	payload, _ := json.Marshal(data)
	ev := &domain.Event{
		ID:        uuid.NewString(),
		TaskID:    taskID,
		Type:      t,
		Payload:   string(payload),
		CreatedAt: time.Now().UTC(),
	}
	if err := e.events.Create(ctx, ev); err != nil {
		e.logger.Warn("event emit failed", "error", err)
	}
}

func marshalError(err error) string {
	b, _ := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
	return string(b)
}
