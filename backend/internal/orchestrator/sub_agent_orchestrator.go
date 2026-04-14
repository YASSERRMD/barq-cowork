package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────
// Port interface
// ─────────────────────────────────────────────

// SubAgentRepo is the persistence interface the SubAgentOrchestrator needs.
type SubAgentRepo interface {
	Create(ctx context.Context, a *domain.SubAgent) error
	GetByID(ctx context.Context, id string) (*domain.SubAgent, error)
	ListByTask(ctx context.Context, parentTaskID string) ([]*domain.SubAgent, error)
	UpdateStatus(ctx context.Context, a *domain.SubAgent) error
	Delete(ctx context.Context, id string) error
}

// ─────────────────────────────────────────────
// SpawnOptions
// ─────────────────────────────────────────────

// SpawnOptions configures a sub-agent spawn request.
type SpawnOptions struct {
	// WorkspaceRoot is passed to each agent's tool invocations.
	WorkspaceRoot string

	// MaxConcurrency caps parallel goroutines (default 3, min 1).
	MaxConcurrency int

	// TimeoutPerAgent is the per-agent execution deadline (default 5 min).
	TimeoutPerAgent time.Duration

	// ProviderConfig to use for all sub-agents.
	ProviderConfig provider.ProviderConfig
}

// ─────────────────────────────────────────────
// SubAgentOrchestrator
// ─────────────────────────────────────────────

// SubAgentOrchestrator spawns and manages specialised child agents for a
// parent task. Each child runs the full Planner+Executor pipeline with a
// role-flavoured system prompt and its own isolated plan. Agents execute in
// parallel up to MaxConcurrency, tracked in the sub_agents table.
type SubAgentOrchestrator struct {
	agents   SubAgentRepo
	planner  *Planner
	executor *Executor
	plans    PlanStore
	events   EventEmitter
	logger   *slog.Logger
}

// NewSubAgentOrchestrator creates a SubAgentOrchestrator.
func NewSubAgentOrchestrator(
	agents SubAgentRepo,
	planner *Planner,
	executor *Executor,
	plans PlanStore,
	events EventEmitter,
	logger *slog.Logger,
) *SubAgentOrchestrator {
	return &SubAgentOrchestrator{
		agents:   agents,
		planner:  planner,
		executor: executor,
		plans:    plans,
		events:   events,
		logger:   logger,
	}
}

// Spawn creates sub-agent records and runs them concurrently in a detached
// goroutine pool. Returns immediately after persisting the records.
func (s *SubAgentOrchestrator) Spawn(
	ctx context.Context,
	parentTaskID string,
	specs []domain.SubAgentSpec,
	opts SpawnOptions,
) ([]*domain.SubAgent, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("at least one sub-agent spec is required")
	}

	now := time.Now().UTC()
	agents := make([]*domain.SubAgent, len(specs))
	for i, spec := range specs {
		a := &domain.SubAgent{
			ID:           uuid.NewString(),
			ParentTaskID: parentTaskID,
			Role:         spec.Role,
			Title:        spec.Title,
			Instructions: spec.Instructions,
			Status:       domain.TaskStatusPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.agents.Create(ctx, a); err != nil {
			return nil, fmt.Errorf("persist sub-agent %d: %w", i, err)
		}
		agents[i] = a
	}

	maxConc := opts.MaxConcurrency
	safeMaxConc := provider.SuggestedMaxConcurrentRequests(opts.ProviderConfig)
	if safeMaxConc <= 0 {
		safeMaxConc = 3
	}
	if maxConc <= 0 {
		maxConc = safeMaxConc
	}
	if maxConc > safeMaxConc {
		maxConc = safeMaxConc
	}
	timeout := opts.TimeoutPerAgent
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	go s.runPool(context.Background(), agents, opts.WorkspaceRoot, opts.ProviderConfig, maxConc, timeout, parentTaskID)
	return agents, nil
}

// Cancel marks a sub-agent as failed immediately (best-effort cooperative cancel).
func (s *SubAgentOrchestrator) Cancel(ctx context.Context, agentID string) error {
	a, err := s.agents.GetByID(ctx, agentID)
	if err != nil {
		return err
	}
	if a.Status == domain.TaskStatusCompleted || a.Status == domain.TaskStatusFailed {
		return nil // already terminal
	}
	now := time.Now().UTC()
	a.Status = domain.TaskStatusFailed
	a.CompletedAt = &now
	return s.agents.UpdateStatus(ctx, a)
}

// ListByTask returns all sub-agents for a parent task.
func (s *SubAgentOrchestrator) ListByTask(ctx context.Context, parentTaskID string) ([]*domain.SubAgent, error) {
	return s.agents.ListByTask(ctx, parentTaskID)
}

// ─────────────────────────────────────────────
// Pool execution
// ─────────────────────────────────────────────

func (s *SubAgentOrchestrator) runPool(
	ctx context.Context,
	agents []*domain.SubAgent,
	workspaceRoot string,
	provCfg provider.ProviderConfig,
	maxConc int,
	timeout time.Duration,
	parentTaskID string,
) {
	sem := make(chan struct{}, maxConc)
	var wg sync.WaitGroup

	for _, a := range agents {
		wg.Add(1)
		agent := a
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			agentCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			s.runOne(agentCtx, agent, workspaceRoot, provCfg, parentTaskID)
		}()
	}

	wg.Wait()

	s.emitAgentEvent(ctx, parentTaskID, domain.EventTypeStepCompleted, map[string]any{
		"event": "sub_agents_pool_done", "count": len(agents),
	})
}

// runOne drives a single sub-agent: pending → planning → running → completed/failed.
func (s *SubAgentOrchestrator) runOne(
	ctx context.Context,
	a *domain.SubAgent,
	workspaceRoot string,
	provCfg provider.ProviderConfig,
	parentTaskID string,
) {
	log := s.logger.With("sub_agent_id", a.ID, "role", a.Role, "title", a.Title)

	// ── Planning ─────────────────────────────────────────────────────
	now := time.Now().UTC()
	a.Status = domain.TaskStatusPlanning
	a.StartedAt = &now
	_ = s.agents.UpdateStatus(ctx, a)

	s.emitAgentEvent(ctx, parentTaskID, domain.EventTypeStepStarted, map[string]any{
		"sub_agent_id": a.ID, "role": string(a.Role), "title": a.Title, "phase": "planning",
	})

	proxyTask := &domain.Task{
		ID:          a.ID,
		ProjectID:   "",
		Title:       a.Title,
		Description: buildSubAgentPrompt(a),
	}

	plan, err := s.planner.Plan(ctx, proxyTask, nil, provCfg)
	if err != nil || plan == nil {
		log.Error("sub-agent planning failed", "error", err)
		s.markFailed(ctx, a, parentTaskID)
		return
	}

	if err := s.plans.CreatePlan(ctx, plan); err != nil {
		log.Error("sub-agent plan persist failed", "error", err)
		s.markFailed(ctx, a, parentTaskID)
		return
	}
	for _, step := range plan.Steps {
		_ = s.plans.CreateStep(ctx, step)
	}

	a.PlanID = plan.ID
	a.Status = domain.TaskStatusRunning
	_ = s.agents.UpdateStatus(ctx, a)

	s.emitAgentEvent(ctx, parentTaskID, domain.EventTypeStepStarted, map[string]any{
		"sub_agent_id": a.ID, "role": string(a.Role), "phase": "running", "steps": len(plan.Steps),
	})

	// ── Execution ────────────────────────────────────────────────────
	result := s.executor.Execute(ctx, plan, proxyTask, workspaceRoot, false)
	log.Info("sub-agent done", "completed", result.Completed, "failed", result.Failed)

	// ── Final status ─────────────────────────────────────────────────
	done := time.Now().UTC()
	a.CompletedAt = &done
	if result.Failed > 0 {
		a.Status = domain.TaskStatusFailed
	} else {
		a.Status = domain.TaskStatusCompleted
	}
	_ = s.agents.UpdateStatus(ctx, a)

	s.emitAgentEvent(ctx, parentTaskID, domain.EventTypeStepCompleted, map[string]any{
		"sub_agent_id": a.ID, "role": string(a.Role), "status": string(a.Status),
		"completed": result.Completed, "failed": result.Failed,
	})
}

func (s *SubAgentOrchestrator) markFailed(ctx context.Context, a *domain.SubAgent, parentTaskID string) {
	now := time.Now().UTC()
	a.Status = domain.TaskStatusFailed
	a.CompletedAt = &now
	_ = s.agents.UpdateStatus(ctx, a)
	s.emitAgentEvent(ctx, parentTaskID, domain.EventTypeStepCompleted, map[string]any{
		"sub_agent_id": a.ID, "role": string(a.Role), "status": "failed",
	})
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func buildSubAgentPrompt(a *domain.SubAgent) string {
	var sb strings.Builder
	if prefix := a.Role.SystemPromptPrefix(); prefix != "" {
		sb.WriteString(prefix)
		sb.WriteString("\n\n")
	}
	sb.WriteString(a.Instructions)
	return sb.String()
}

func (s *SubAgentOrchestrator) emitAgentEvent(ctx context.Context, taskID string, t domain.EventType, data map[string]any) {
	payload, _ := json.Marshal(data)
	ev := &domain.Event{
		ID:        uuid.NewString(),
		TaskID:    taskID,
		Type:      t,
		Payload:   string(payload),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.events.Create(ctx, ev); err != nil {
		s.logger.Warn("sub-agent event emit failed", "error", err)
	}
}
