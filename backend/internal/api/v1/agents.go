package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/orchestrator"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/go-chi/chi/v5"
)

// ─────────────────────────────────────────────
// Port interface
// ─────────────────────────────────────────────

// SubAgentRunner is the narrow port the handler needs.
type SubAgentRunner interface {
	Spawn(ctx context.Context, parentTaskID string, specs []domain.SubAgentSpec, opts orchestrator.SpawnOptions) ([]*domain.SubAgent, error)
	Cancel(ctx context.Context, agentID string) error
	ListByTask(ctx context.Context, parentTaskID string) ([]*domain.SubAgent, error)
}

// ─────────────────────────────────────────────
// AgentsHandler
// ─────────────────────────────────────────────

// AgentsHandler mounts the sub-agent HTTP routes.
type AgentsHandler struct {
	runner SubAgentRunner
	// providerResolver provides the default provider config when the caller
	// doesn't specify one (falls back to config default).
	providerResolver func() provider.ProviderConfig
}

// NewAgentsHandler creates an AgentsHandler.
func NewAgentsHandler(runner SubAgentRunner, defaultProvider func() provider.ProviderConfig) *AgentsHandler {
	return &AgentsHandler{runner: runner, providerResolver: defaultProvider}
}

// Register mounts the agent routes on an /api/v1 sub-router.
func (h *AgentsHandler) Register(r chi.Router) {
	r.Post("/tasks/{id}/agents", h.spawnAgents)
	r.Get("/tasks/{id}/agents", h.listAgents)
	r.Delete("/tasks/{id}/agents/{agentId}", h.cancelAgent)
}

// ─────────────────────────────────────────────
// Handlers
// ─────────────────────────────────────────────

// spawnAgents POST /api/v1/tasks/{id}/agents
// Spawns one or more sub-agents. Returns 202 with the created agent records.
func (h *AgentsHandler) spawnAgents(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agents []struct {
			Role         string `json:"role"`
			Title        string `json:"title"`
			Instructions string `json:"instructions"`
		} `json:"agents"`
		WorkspaceRoot   string `json:"workspace_root"`
		MaxConcurrency  int    `json:"max_concurrency"`
		TimeoutMinutes  int    `json:"timeout_minutes"`
	}
	if !decode(w, r, &req) {
		return
	}
	if len(req.Agents) == 0 {
		jsonError(w, http.StatusUnprocessableEntity, "agents array must not be empty")
		return
	}

	specs := make([]domain.SubAgentSpec, len(req.Agents))
	for i, a := range req.Agents {
		role := domain.AgentRole(a.Role)
		if role == "" {
			role = domain.AgentRoleCustom
		}
		specs[i] = domain.SubAgentSpec{
			Role:         role,
			Title:        a.Title,
			Instructions: a.Instructions,
		}
	}

	timeout := time.Duration(req.TimeoutMinutes) * time.Minute
	opts := orchestrator.SpawnOptions{
		WorkspaceRoot:   req.WorkspaceRoot,
		MaxConcurrency:  req.MaxConcurrency,
		TimeoutPerAgent: timeout,
		ProviderConfig:  h.providerResolver(),
	}

	agents, err := h.runner.Spawn(r.Context(), chi.URLParam(r, "id"), specs, opts)
	if err != nil {
		handleErr(w, err)
		return
	}

	out := make([]*subAgentDTO, len(agents))
	for i, a := range agents {
		out[i] = toSubAgentDTO(a)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = encodeJSON(w, envelope{Data: out})
}

// listAgents GET /api/v1/tasks/{id}/agents
func (h *AgentsHandler) listAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.runner.ListByTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*subAgentDTO, len(agents))
	for i, a := range agents {
		out[i] = toSubAgentDTO(a)
	}
	if out == nil {
		out = []*subAgentDTO{}
	}
	jsonOK(w, out)
}

// cancelAgent DELETE /api/v1/tasks/{id}/agents/{agentId}
func (h *AgentsHandler) cancelAgent(w http.ResponseWriter, r *http.Request) {
	if err := h.runner.Cancel(r.Context(), chi.URLParam(r, "agentId")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}
