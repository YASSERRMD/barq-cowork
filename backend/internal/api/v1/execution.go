package v1

import (
	"context"
	"net/http"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/orchestrator"
	"github.com/go-chi/chi/v5"
)

// ─────────────────────────────────────────────
// Port interfaces (narrow, owned by this layer)
// ─────────────────────────────────────────────

// TaskRunner dispatches an async task run.
type TaskRunner interface {
	RunTask(ctx context.Context, taskID string, opts orchestrator.RunOptions) error
}

// PlanQuerier fetches the plan (with steps) for a task.
type PlanQuerier interface {
	GetPlanByTask(ctx context.Context, taskID string) (*domain.Plan, error)
}

// ArtifactQuerier fetches artifacts.
type ArtifactQuerier interface {
	GetByID(ctx context.Context, id string) (*domain.Artifact, error)
	ListByTask(ctx context.Context, taskID string) ([]*domain.Artifact, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Artifact, error)
}

// EventQuerier fetches events for a task.
type EventQuerier interface {
	ListByTask(ctx context.Context, taskID string) ([]*domain.Event, error)
}

// ─────────────────────────────────────────────
// ExecutionHandler
// ─────────────────────────────────────────────

// ExecutionHandler mounts the task execution + observation endpoints.
type ExecutionHandler struct {
	runner    TaskRunner
	plans     PlanQuerier
	artifacts ArtifactQuerier
	events    EventQuerier
}

// NewExecutionHandler creates an ExecutionHandler.
func NewExecutionHandler(
	runner TaskRunner,
	plans PlanQuerier,
	artifacts ArtifactQuerier,
	events EventQuerier,
) *ExecutionHandler {
	return &ExecutionHandler{
		runner:    runner,
		plans:     plans,
		artifacts: artifacts,
		events:    events,
	}
}

// Register mounts the execution routes onto r (an /api/v1 sub-router).
func (h *ExecutionHandler) Register(r chi.Router) {
	// Task execution control
	r.Post("/tasks/{id}/run", h.runTask)

	// Task execution observation
	r.Get("/tasks/{id}/plan", h.getPlan)
	r.Get("/tasks/{id}/events", h.listEvents)
	r.Get("/tasks/{id}/artifacts", h.listArtifactsByTask)

	// Project-level artifact listing
	r.Get("/projects/{projectID}/artifacts", h.listArtifactsByProject)

	// Single artifact
	r.Get("/artifacts/{id}", h.getArtifact)
}

// ─────────────────────────────────────────────
// Handlers
// ─────────────────────────────────────────────

// runTask POST /api/v1/tasks/{id}/run
// Starts async execution; returns 202 Accepted immediately.
func (h *ExecutionHandler) runTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkspaceRoot   string `json:"workspace_root"`
		RequireApproval bool   `json:"require_approval"`
	}
	// Body is optional — decode only if Content-Type header is present.
	if r.ContentLength > 0 {
		if !decode(w, r, &req) {
			return
		}
	}

	opts := orchestrator.RunOptions{
		WorkspaceRoot:   req.WorkspaceRoot,
		RequireApproval: req.RequireApproval,
	}

	if err := h.runner.RunTask(r.Context(), chi.URLParam(r, "id"), opts); err != nil {
		handleErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}

// getPlan GET /api/v1/tasks/{id}/plan
func (h *ExecutionHandler) getPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.plans.GetPlanByTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toPlanDTO(plan))
}

// listEvents GET /api/v1/tasks/{id}/events
func (h *ExecutionHandler) listEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.events.ListByTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*eventDTO, len(events))
	for i, e := range events {
		out[i] = toEventDTO(e)
	}
	if out == nil {
		out = []*eventDTO{}
	}
	jsonOK(w, out)
}

// listArtifactsByTask GET /api/v1/tasks/{id}/artifacts
func (h *ExecutionHandler) listArtifactsByTask(w http.ResponseWriter, r *http.Request) {
	artifacts, err := h.artifacts.ListByTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	h.respondArtifacts(w, artifacts)
}

// listArtifactsByProject GET /api/v1/projects/{projectID}/artifacts
func (h *ExecutionHandler) listArtifactsByProject(w http.ResponseWriter, r *http.Request) {
	artifacts, err := h.artifacts.ListByProject(r.Context(), chi.URLParam(r, "projectID"))
	if err != nil {
		handleErr(w, err)
		return
	}
	h.respondArtifacts(w, artifacts)
}

// getArtifact GET /api/v1/artifacts/{id}
func (h *ExecutionHandler) getArtifact(w http.ResponseWriter, r *http.Request) {
	a, err := h.artifacts.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toArtifactDTO(a))
}

func (h *ExecutionHandler) respondArtifacts(w http.ResponseWriter, artifacts []*domain.Artifact) {
	out := make([]*artifactDTO, len(artifacts))
	for i, a := range artifacts {
		out[i] = toArtifactDTO(a)
	}
	if out == nil {
		out = []*artifactDTO{}
	}
	jsonOK(w, out)
}
