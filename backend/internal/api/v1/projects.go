package v1

import (
	"net/http"

	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/go-chi/chi/v5"
)

// ProjectHandler wires project HTTP routes.
type ProjectHandler struct {
	svc *service.ProjectService
}

// NewProjectHandler creates a ProjectHandler.
func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// Register mounts the project routes on r.
// Projects nested under workspaces for listing, flat for individual access.
func (h *ProjectHandler) Register(r chi.Router) {
	r.Get("/projects", h.listAll)
	r.Get("/workspaces/{workspaceID}/projects", h.list)
	r.Post("/projects", h.create)
	r.Get("/projects/{id}", h.get)
	r.Put("/projects/{id}", h.update)
	r.Delete("/projects/{id}", h.delete)
}

// listAll GET /api/v1/projects
func (h *ProjectHandler) listAll(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.ListAll(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*projectDTO, len(projects))
	for i, p := range projects {
		out[i] = toProjectDTO(p)
	}
	if out == nil {
		out = []*projectDTO{}
	}
	jsonOK(w, out)
}

// list GET /api/v1/workspaces/{workspaceID}/projects
func (h *ProjectHandler) list(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.ListByWorkspace(r.Context(), chi.URLParam(r, "workspaceID"))
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*projectDTO, len(projects))
	for i, p := range projects {
		out[i] = toProjectDTO(p)
	}
	if out == nil {
		out = []*projectDTO{}
	}
	jsonOK(w, out)
}

// create POST /api/v1/projects
func (h *ProjectHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkspaceID  string `json:"workspace_id"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		Instructions string `json:"instructions"`
	}
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.Create(r.Context(), req.WorkspaceID, req.Name, req.Description, req.Instructions)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonCreated(w, toProjectDTO(p))
}

// get GET /api/v1/projects/{id}
func (h *ProjectHandler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toProjectDTO(p))
}

// update PUT /api/v1/projects/{id}
func (h *ProjectHandler) update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Instructions string `json:"instructions"`
	}
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), req.Name, req.Description, req.Instructions)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toProjectDTO(p))
}

// delete DELETE /api/v1/projects/{id}
func (h *ProjectHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}
