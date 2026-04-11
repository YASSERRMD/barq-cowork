package v1

import (
	"net/http"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/go-chi/chi/v5"
)

// ToolHandler wires tool-related HTTP routes.
type ToolHandler struct {
	svc *service.ToolService
}

// NewToolHandler creates a ToolHandler.
func NewToolHandler(svc *service.ToolService) *ToolHandler {
	return &ToolHandler{svc: svc}
}

// Register mounts the tool routes on r.
func (h *ToolHandler) Register(r chi.Router) {
	r.Get("/tools", h.listTools)
	r.Post("/tools/invoke", h.invoke)

	r.Get("/approvals", h.listApprovals)
	r.Post("/approvals/{id}/resolve", h.resolve)
}

// listTools GET /api/v1/tools
func (h *ToolHandler) listTools(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, h.svc.ListTools())
}

// invoke POST /api/v1/tools/invoke
func (h *ToolHandler) invoke(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID          string `json:"task_id"`
		WorkspaceRoot   string `json:"workspace_root"`
		ToolName        string `json:"tool_name"`
		ArgsJSON        string `json:"args_json"`
		RequireApproval bool   `json:"require_approval"`
	}
	if !decode(w, r, &req) {
		return
	}
	result, err := h.svc.Invoke(
		r.Context(),
		req.TaskID, req.WorkspaceRoot,
		req.ToolName, req.ArgsJSON,
		req.RequireApproval,
	)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, result)
}

// listApprovals GET /api/v1/approvals
func (h *ToolHandler) listApprovals(w http.ResponseWriter, r *http.Request) {
	approvals, err := h.svc.ListPendingApprovals(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*approvalDTO, len(approvals))
	for i, a := range approvals {
		out[i] = toApprovalDTO(a)
	}
	if out == nil {
		out = []*approvalDTO{}
	}
	jsonOK(w, out)
}

// resolve POST /api/v1/approvals/{id}/resolve
func (h *ToolHandler) resolve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Resolution string `json:"resolution"` // "approved" | "rejected"
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.svc.ResolveApproval(r.Context(), chi.URLParam(r, "id"), req.Resolution); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}

// ─────────────────────────────────────────────
// DTOs
// ─────────────────────────────────────────────

type approvalDTO struct {
	ID         string                 `json:"id"`
	TaskID     string                 `json:"task_id"`
	ToolName   string                 `json:"tool_name"`
	Action     string                 `json:"action"`
	Payload    string                 `json:"payload"`
	Status     domain.ApprovalStatus  `json:"status"`
	Resolution string                 `json:"resolution,omitempty"`
	CreatedAt  string                 `json:"created_at"`
}

func toApprovalDTO(a *domain.ApprovalRequest) *approvalDTO {
	return &approvalDTO{
		ID:         a.ID,
		TaskID:     a.TaskID,
		ToolName:   a.ToolName,
		Action:     a.Action,
		Payload:    a.Payload,
		Status:     a.Status,
		Resolution: a.Resolution,
		CreatedAt:  a.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

