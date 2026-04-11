package v1

import (
	"net/http"
	"strconv"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/go-chi/chi/v5"
)

// TaskHandler wires task HTTP routes.
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler creates a TaskHandler.
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Register mounts the task routes on r.
func (h *TaskHandler) Register(r chi.Router) {
	r.Get("/tasks", h.listAll)
	r.Get("/projects/{projectID}/tasks", h.list)
	r.Post("/tasks", h.create)
	r.Get("/tasks/{id}", h.get)
	r.Put("/tasks/{id}", h.update)
	r.Patch("/tasks/{id}/status", h.updateStatus)
	r.Delete("/tasks/{id}", h.delete)
}

// listAll GET /api/v1/tasks
func (h *TaskHandler) listAll(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	tasks, err := h.svc.ListAll(r.Context(), limit)
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*taskDTO, len(tasks))
	for i, t := range tasks {
		out[i] = toTaskDTO(t)
	}
	if out == nil {
		out = []*taskDTO{}
	}
	jsonOK(w, out)
}

// list GET /api/v1/projects/{projectID}/tasks
func (h *TaskHandler) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListByProject(r.Context(), chi.URLParam(r, "projectID"))
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*taskDTO, len(tasks))
	for i, t := range tasks {
		out[i] = toTaskDTO(t)
	}
	if out == nil {
		out = []*taskDTO{}
	}
	jsonOK(w, out)
}

// create POST /api/v1/tasks
func (h *TaskHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID   string `json:"project_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ProviderID  string `json:"provider_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	t, err := h.svc.Create(r.Context(), req.ProjectID, req.Title, req.Description, req.ProviderID)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonCreated(w, toTaskDTO(t))
}

// get GET /api/v1/tasks/{id}
func (h *TaskHandler) get(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toTaskDTO(t))
}

// update PUT /api/v1/tasks/{id}
func (h *TaskHandler) update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ProviderID  string `json:"provider_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	t, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), req.Title, req.Description, req.ProviderID)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toTaskDTO(t))
}

// updateStatus PATCH /api/v1/tasks/{id}/status
func (h *TaskHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &req) {
		return
	}
	t, err := h.svc.UpdateStatus(r.Context(), chi.URLParam(r, "id"), domain.TaskStatus(req.Status))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toTaskDTO(t))
}

// delete DELETE /api/v1/tasks/{id}
func (h *TaskHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}
