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
	"github.com/barq-cowork/barq-cowork/internal/tools"
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
	GetDefault(ctx context.Context) (*domain.ProviderProfile, error)
}

// ─────────────────────────────────────────────
// Orchestrator
// ─────────────────────────────────────────────

// Orchestrator coordinates the full task lifecycle:
//
//	Pending → Running → Completed | Failed
type Orchestrator struct {
	tasks            TaskRepo
	projects         ProjectRepo
	profiles         ProviderProfileRepo
	plans            PlanStore
	providerRegistry LLMProviderGetter
	toolRegistry     *tools.Registry
	artifacts        ArtifactStore
	events           EventEmitter
	cfg              *config.Config
	logger           *slog.Logger
}

// New creates an Orchestrator.
func New(
	tasks TaskRepo,
	projects ProjectRepo,
	profiles ProviderProfileRepo,
	plans PlanStore,
	providerRegistry LLMProviderGetter,
	toolRegistry *tools.Registry,
	artifacts ArtifactStore,
	events EventEmitter,
	cfg *config.Config,
	logger *slog.Logger,
) *Orchestrator {
	return &Orchestrator{
		tasks:            tasks,
		projects:         projects,
		profiles:         profiles,
		plans:            plans,
		providerRegistry: providerRegistry,
		toolRegistry:     toolRegistry,
		artifacts:        artifacts,
		events:           events,
		cfg:              cfg,
		logger:           logger,
	}
}

// RunOptions carries per-run parameters supplied by the caller.
type RunOptions struct {
	// WorkspaceRoot overrides the workspace's RootPath for file tools.
	// If empty, the orchestrator falls back to <dataDir>/workspace.
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

	// project is optional — direct (project-less) tasks have an empty ProjectID
	var project *domain.Project
	if task.ProjectID != "" {
		project, err = o.projects.GetByID(ctx, task.ProjectID)
		if err != nil {
			return fmt.Errorf("load project: %w", err)
		}
	}
	// project variable consumed only for workspace root resolution (future use).
	_ = project

	// Resolve provider config
	provCfg, err := o.resolveProviderConfig(ctx, task)
	if err != nil {
		return fmt.Errorf("resolve provider: %w", err)
	}

	// Default workspace root to <dataDir>/workspace so file tools always work.
	if opts.WorkspaceRoot == "" {
		dataDir := o.cfg.App.DataDir
		if len(dataDir) > 1 && dataDir[:2] == "~/" {
			if home, err := os.UserHomeDir(); err == nil {
				dataDir = home + dataDir[1:]
			}
		}
		opts.WorkspaceRoot = dataDir + "/workspace"
		_ = os.MkdirAll(opts.WorkspaceRoot, 0o755)
	}

	// Run the full lifecycle in a detached goroutine.
	// We use a background context so the run outlives the HTTP request.
	runCtx := context.Background()
	go o.run(runCtx, task, provCfg, opts)

	return nil
}

// run is the goroutine body that drives the full task lifecycle via the agent loop.
func (o *Orchestrator) run(
	ctx context.Context,
	task *domain.Task,
	provCfg provider.ProviderConfig,
	opts RunOptions,
) {
	log := o.logger.With("task_id", task.ID, "title", task.Title)

	// Resolve provider instance
	prov, ok := o.providerRegistry.Get(provCfg.ProviderName)
	if !ok {
		log.Error("provider not registered", "provider", provCfg.ProviderName)
		_ = o.tasks.UpdateStatus(ctx, task.ID, domain.TaskStatusFailed, time.Now().UTC())
		return
	}

	log.Info("agent loop starting")
	_ = o.tasks.UpdateStatus(ctx, task.ID, domain.TaskStatusRunning, time.Now().UTC())

	loop := NewAgentLoop(prov, provCfg, o.toolRegistry, o.plans, o.artifacts, o.events, o.logger)
	result := loop.Run(ctx, task, opts.WorkspaceRoot)

	// Final status
	finalStatus := domain.TaskStatusCompleted
	if result.Failed > 0 && result.Completed == 0 {
		finalStatus = domain.TaskStatusFailed
	}
	_ = o.tasks.UpdateStatus(ctx, task.ID, finalStatus, time.Now().UTC())
	log.Info("agent finished", "completed", result.Completed, "failed", result.Failed)
}

// ─────────────────────────────────────────────
// Provider resolution
// ─────────────────────────────────────────────

// resolveProviderConfig builds a ProviderConfig for the task.
// Priority: task.ProviderID → project default → config default.
func (o *Orchestrator) resolveProviderConfig(ctx context.Context, task *domain.Task) (provider.ProviderConfig, error) {
	// 1. Explicit profile selected on the task
	if task.ProviderID != "" {
		profile, err := o.profiles.GetByID(ctx, task.ProviderID)
		if err == nil {
			return profileToConfig(profile), nil
		}
	}

	// 2. Default (or any) saved profile in the database — this is what the
	//    Settings page configures. Always prefer it over the env-var fallback.
	if profile, err := o.profiles.GetDefault(ctx); err == nil {
		return profileToConfig(profile), nil
	}

	// 3. Last resort: env-var config (no profile saved yet)
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
	// Direct key takes precedence; fall back to env var for legacy profiles.
	apiKey := p.APIKey
	if apiKey == "" && p.APIKeyEnv != "" {
		apiKey = os.Getenv(p.APIKeyEnv)
	}
	return provider.ProviderConfig{
		ProviderName: p.ProviderName,
		BaseURL:      p.BaseURL,
		APIKey:       apiKey,
		Model:        p.Model,
		TimeoutSec:   p.TimeoutSec,
	}
}
