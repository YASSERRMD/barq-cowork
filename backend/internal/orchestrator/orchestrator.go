package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/config"
	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/provider"
)

// ─────────────────────────────────────────────
// Repository interfaces used by Orchestrator
// ─────────────────────────────────────────────

// TaskRepo provides the task operations the Orchestrator needs.
type TaskRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Task, error)
	UpdateStatus(ctx context.Context, id string, status domain.TaskStatus, now time.Time) error
}

// ProjectRepo provides the project lookup the Orchestrator needs.
type ProjectRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Project, error)
}

// ProviderProfileRepo provides saved provider profile lookup.
type ProviderProfileRepo interface {
	GetByID(ctx context.Context, id string) (*domain.ProviderProfile, error)
}

// ─────────────────────────────────────────────
// Orchestrator
// ─────────────────────────────────────────────

// Orchestrator coordinates the full task lifecycle:
//   Pending → Planning → Running → Completed | Failed
type Orchestrator struct {
	tasks    TaskRepo
	projects ProjectRepo
	profiles ProviderProfileRepo
	planner  *Planner
	executor *Executor
	plans    PlanStore
	cfg      *config.Config
	logger   *slog.Logger
}

// New creates an Orchestrator.
func New(
	tasks TaskRepo,
	projects ProjectRepo,
	profiles ProviderProfileRepo,
	planner *Planner,
	executor *Executor,
	plans PlanStore,
	cfg *config.Config,
	logger *slog.Logger,
) *Orchestrator {
	return &Orchestrator{
		tasks:    tasks,
		projects: projects,
		profiles: profiles,
		planner:  planner,
		executor: executor,
		plans:    plans,
		cfg:      cfg,
		logger:   logger,
	}
}

// RunOptions carries per-run parameters supplied by the caller.
type RunOptions struct {
	// WorkspaceRoot overrides the workspace's RootPath for file tools.
	// If empty, tools will have no filesystem access.
	WorkspaceRoot string

	// RequireApproval controls whether destructive tools gate on user approval.
	RequireApproval bool
}

// RunTask starts task execution asynchronously in a background goroutine and
// returns immediately. It transitions the task through its lifecycle states
// and emits domain events for the frontend to poll.
func (o *Orchestrator) RunTask(ctx context.Context, taskID string, opts RunOptions) error {
	task, err := o.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("load task: %w", err)
	}
	if task.Status != domain.TaskStatusPending {
		return &domain.ValidationError{
			Field:   "status",
			Message: fmt.Sprintf("task is %s — only pending tasks can be started", task.Status),
		}
	}

	project, err := o.projects.GetByID(ctx, task.ProjectID)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}

	// Resolve provider config
	provCfg, err := o.resolveProviderConfig(ctx, task)
	if err != nil {
		return fmt.Errorf("resolve provider: %w", err)
	}

	// Run the full lifecycle in a detached goroutine.
	// We use a background context so the run outlives the HTTP request.
	runCtx := context.Background()
	go o.run(runCtx, task, project, provCfg, opts)

	return nil
}

// run is the goroutine body that drives the full task lifecycle.
func (o *Orchestrator) run(
	ctx context.Context,
	task *domain.Task,
	project *domain.Project,
	provCfg provider.ProviderConfig,
	opts RunOptions,
) {
	log := o.logger.With("task_id", task.ID, "title", task.Title)

	// ── Phase 1: Planning ────────────────────────────────────────────
	log.Info("task planning started")
	_ = o.tasks.UpdateStatus(ctx, task.ID, domain.TaskStatusPlanning, time.Now().UTC())

	plan, err := o.planner.Plan(ctx, task, project, provCfg)
	if err != nil {
		log.Error("planning failed", "error", err)
		_ = o.tasks.UpdateStatus(ctx, task.ID, domain.TaskStatusFailed, time.Now().UTC())
		return
	}

	// Persist the plan and its steps
	if err := o.plans.CreatePlan(ctx, plan); err != nil {
		log.Error("persist plan failed", "error", err)
		_ = o.tasks.UpdateStatus(ctx, task.ID, domain.TaskStatusFailed, time.Now().UTC())
		return
	}
	for _, step := range plan.Steps {
		if err := o.plans.CreateStep(ctx, step); err != nil {
			log.Error("persist step failed", "step_order", step.Order, "error", err)
		}
	}
	log.Info("plan created", "steps", len(plan.Steps))

	// ── Phase 2: Execution ───────────────────────────────────────────
	log.Info("task execution started")
	_ = o.tasks.UpdateStatus(ctx, task.ID, domain.TaskStatusRunning, time.Now().UTC())

	result := o.executor.Execute(ctx, plan, task, opts.WorkspaceRoot, opts.RequireApproval)

	log.Info("execution finished",
		"completed", result.Completed,
		"failed", result.Failed,
		"skipped", result.Skipped,
	)

	// ── Phase 3: Final status ────────────────────────────────────────
	finalStatus := domain.TaskStatusCompleted
	if result.Failed > 0 {
		finalStatus = domain.TaskStatusFailed
	}
	_ = o.tasks.UpdateStatus(ctx, task.ID, finalStatus, time.Now().UTC())
	log.Info("task finished", "final_status", finalStatus)
}

// ─────────────────────────────────────────────
// Provider resolution
// ─────────────────────────────────────────────

// resolveProviderConfig builds a ProviderConfig for the task.
// Priority: task.ProviderID → project default → config default.
func (o *Orchestrator) resolveProviderConfig(ctx context.Context, task *domain.Task) (provider.ProviderConfig, error) {
	// Try saved provider profile
	if task.ProviderID != "" {
		profile, err := o.profiles.GetByID(ctx, task.ProviderID)
		if err == nil {
			return profileToConfig(profile), nil
		}
	}

	// Fall back to config default
	provName := o.cfg.LLM.DefaultProvider
	pc, ok := o.cfg.LLM.Providers[provName]
	if !ok {
		return provider.ProviderConfig{}, fmt.Errorf("no provider config for %q", provName)
	}

	apiKey := os.Getenv(pc.APIKeyEnv)
	return provider.ProviderConfig{
		ProviderName: provName,
		BaseURL:      pc.BaseURL,
		APIKey:       apiKey,
		Model:        pc.Model,
		TimeoutSec:   pc.TimeoutSec,
		ExtraHeaders: pc.ExtraHeaders,
	}, nil
}

func profileToConfig(p *domain.ProviderProfile) provider.ProviderConfig {
	apiKey := os.Getenv(p.APIKeyEnv)
	return provider.ProviderConfig{
		ProviderName: p.ProviderName,
		BaseURL:      p.BaseURL,
		APIKey:       apiKey,
		Model:        p.Model,
		TimeoutSec:   p.TimeoutSec,
	}
}
