package v1

import (
	"net/http"

	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/go-chi/chi/v5"
)

// WorkspaceHandler wires workspace HTTP routes.
type WorkspaceHandler struct {
	svc *service.WorkspaceService
}

// NewWorkspaceHandler creates a WorkspaceHandler.
func NewWorkspaceHandler(svc *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

// Register mounts the workspace routes on r.
func (h *WorkspaceHandler) Register(r chi.Router) {
	r.Get("/workspaces", h.list)
	r.Post("/workspaces", h.create)
	r.Get("/workspaces/{id}", h.get)
	r.Put("/workspaces/{id}", h.update)
	r.Delete("/workspaces/{id}", h.delete)
}

// list GET /api/v1/workspaces
func (h *WorkspaceHandler) list(w http.ResponseWriter, r *http.Request) {
	workspaces, err := h.svc.List(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*workspaceDTO, len(workspaces))
	for i, ws := range workspaces {
		out[i] = toWorkspaceDTO(ws)
	}
	if out == nil {
		out = []*workspaceDTO{}
	}
	jsonOK(w, out)
}

// create POST /api/v1/workspaces
func (h *WorkspaceHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		RootPath    string `json:"root_path"`
	}
	if !decode(w, r, &req) {
		return
	}
	ws, err := h.svc.Create(r.Context(), req.Name, req.Description, req.RootPath)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonCreated(w, toWorkspaceDTO(ws))
}

// get GET /api/v1/workspaces/{id}
func (h *WorkspaceHandler) get(w http.ResponseWriter, r *http.Request) {
	ws, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toWorkspaceDTO(ws))
}

// update PUT /api/v1/workspaces/{id}
func (h *WorkspaceHandler) update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		RootPath    string `json:"root_path"`
	}
	if !decode(w, r, &req) {
		return
	}
	ws, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), req.Name, req.Description, req.RootPath)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toWorkspaceDTO(ws))
}

// delete DELETE /api/v1/workspaces/{id}
func (h *WorkspaceHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}
