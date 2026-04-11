package v1

import (
	"net/http"

	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/go-chi/chi/v5"
)

// ScheduleHandler handles schedule HTTP routes.
type ScheduleHandler struct {
	svc *service.ScheduleService
}

func NewScheduleHandler(svc *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

func (h *ScheduleHandler) Register(r chi.Router) {
	r.Get("/schedules", h.list)
	r.Post("/schedules", h.create)
	r.Get("/schedules/{id}", h.get)
	r.Put("/schedules/{id}", h.update)
	r.Delete("/schedules/{id}", h.delete)
	r.Get("/projects/{projectId}/schedules", h.listByProject)
}

type scheduleInput struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CronExpr    string `json:"cron_expr"`
	TaskTitle   string `json:"task_title"`
	TaskDesc    string `json:"task_desc"`
	ProviderID  string `json:"provider_id"`
	Enabled     bool   `json:"enabled"`
}

func (h *ScheduleHandler) list(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.svc.List(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*scheduleDTO, len(schedules))
	for i, s := range schedules {
		out[i] = toScheduleDTO(s)
	}
	if out == nil {
		out = []*scheduleDTO{}
	}
	jsonOK(w, out)
}

func (h *ScheduleHandler) listByProject(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.svc.ListByProject(r.Context(), chi.URLParam(r, "projectId"))
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*scheduleDTO, len(schedules))
	for i, s := range schedules {
		out[i] = toScheduleDTO(s)
	}
	if out == nil {
		out = []*scheduleDTO{}
	}
	jsonOK(w, out)
}

func (h *ScheduleHandler) create(w http.ResponseWriter, r *http.Request) {
	var req scheduleInput
	if !decode(w, r, &req) {
		return
	}
	s, err := h.svc.Create(r.Context(),
		req.ProjectID, req.Name, req.Description, req.CronExpr,
		req.TaskTitle, req.TaskDesc, req.ProviderID, req.Enabled,
	)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonCreated(w, toScheduleDTO(s))
}

func (h *ScheduleHandler) get(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toScheduleDTO(s))
}

func (h *ScheduleHandler) update(w http.ResponseWriter, r *http.Request) {
	var req scheduleInput
	if !decode(w, r, &req) {
		return
	}
	s, err := h.svc.Update(r.Context(),
		chi.URLParam(r, "id"),
		req.Name, req.Description, req.CronExpr,
		req.TaskTitle, req.TaskDesc, req.ProviderID, req.Enabled,
	)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toScheduleDTO(s))
}

func (h *ScheduleHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}
