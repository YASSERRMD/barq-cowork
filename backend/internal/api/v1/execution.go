package v1

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/orchestrator"
	"github.com/barq-cowork/barq-cowork/internal/tools"
	"github.com/go-chi/chi/v5"
)

func parsePositiveInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}

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
	ListRecent(ctx context.Context, limit int) ([]*domain.Artifact, error)
}

// EventQuerier fetches events for a task or globally.
type EventQuerier interface {
	ListByTask(ctx context.Context, taskID string) ([]*domain.Event, error)
	ListRecent(ctx context.Context, limit int) ([]*domain.Event, error)
}

// ─────────────────────────────────────────────
// ExecutionHandler
// ─────────────────────────────────────────────

// ExecutionHandler mounts the task execution + observation endpoints.
type ExecutionHandler struct {
	runner        TaskRunner
	plans         PlanQuerier
	artifacts     ArtifactQuerier
	events        EventQuerier
	workspaceRoot string
	userInput     *tools.UserInputStore // may be nil if ask_user not wired
}

// NewExecutionHandler creates an ExecutionHandler.
func NewExecutionHandler(
	runner TaskRunner,
	plans PlanQuerier,
	artifacts ArtifactQuerier,
	events EventQuerier,
	workspaceRoot string,
	userInput *tools.UserInputStore,
) *ExecutionHandler {
	return &ExecutionHandler{
		runner:        runner,
		plans:         plans,
		artifacts:     artifacts,
		events:        events,
		workspaceRoot: workspaceRoot,
		userInput:     userInput,
	}
}

// Register mounts the execution routes onto r (an /api/v1 sub-router).
func (h *ExecutionHandler) Register(r chi.Router) {
	// File upload
	r.Post("/workspace/upload", h.uploadFiles)

	// Task execution control
	r.Post("/tasks/{id}/run", h.runTask)

	// Interactive mid-task user input (ask_user tool)
	r.Post("/tasks/{id}/respond", h.respondToInput)
	r.Get("/tasks/{id}/pending-inputs", h.listPendingInputs)

	// Task execution observation
	r.Get("/tasks/{id}/plan", h.getPlan)
	r.Get("/tasks/{id}/events", h.listEvents)
	r.Get("/tasks/{id}/artifacts", h.listArtifactsByTask)

	// Project-level artifact listing
	r.Get("/projects/{projectID}/artifacts", h.listArtifactsByProject)

	// Global artifact listing + single artifact + download
	r.Get("/artifacts", h.listArtifactsRecent)
	r.Get("/artifacts/{id}", h.getArtifact)
	r.Get("/artifacts/{id}/download", h.downloadArtifact)

	// Global event log
	r.Get("/events", h.listEventsRecent)
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

// listArtifactsRecent GET /api/v1/artifacts[?limit=N]
func (h *ExecutionHandler) listArtifactsRecent(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			limit = n
		}
	}
	artifacts, err := h.artifacts.ListRecent(r.Context(), limit)
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

// listEventsRecent GET /api/v1/events[?limit=N]
func (h *ExecutionHandler) listEventsRecent(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			limit = n
		}
	}
	events, err := h.events.ListRecent(r.Context(), limit)
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

// downloadArtifact GET /api/v1/artifacts/{id}/download
// Reads the file from disk and streams it to the client with correct Content-Type.
func (h *ExecutionHandler) downloadArtifact(w http.ResponseWriter, r *http.Request) {
	a, err := h.artifacts.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}

	// Resolve full path: workspaceRoot + content_path
	root := h.workspaceRoot
	if root == "" {
		http.Error(w, "workspace root not configured", http.StatusInternalServerError)
		return
	}

	fullPath := filepath.Join(root, filepath.FromSlash(a.ContentPath))
	// Security: ensure the resolved path is still under workspaceRoot
	if !strings.HasPrefix(fullPath, filepath.Clean(root)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("file not found: %v", err), http.StatusNotFound)
		return
	}

	// Determine MIME type from extension
	ext := strings.ToLower(filepath.Ext(fullPath))
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		switch ext {
		case ".md":
			ct = "text/markdown; charset=utf-8"
		case ".html":
			ct = "text/html; charset=utf-8"
		case ".json":
			ct = "application/json"
		case ".pptx":
			ct = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		case ".docx":
			ct = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		case ".xlsx":
			ct = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case ".pdf":
			ct = "application/pdf"
		default:
			ct = "application/octet-stream"
		}
	}

	filename := filepath.Base(fullPath)
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// Prevent browser from caching artifact downloads — always serve fresh file
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// respondToInput POST /api/v1/tasks/{id}/respond
// Delivers the user's answer to a blocked ask_user tool call.
func (h *ExecutionHandler) respondToInput(w http.ResponseWriter, r *http.Request) {
	if h.userInput == nil {
		http.Error(w, "interactive input not enabled", http.StatusNotImplemented)
		return
	}
	var req struct {
		InputID string `json:"input_id"`
		Answer  string `json:"answer"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.InputID == "" {
		http.Error(w, "input_id is required", http.StatusBadRequest)
		return
	}
	if !h.userInput.Answer(req.InputID, req.Answer) {
		http.Error(w, "input_id not found or already answered", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// listPendingInputs GET /api/v1/tasks/{id}/pending-inputs
// Returns all open ask_user questions for the given task.
func (h *ExecutionHandler) listPendingInputs(w http.ResponseWriter, r *http.Request) {
	if h.userInput == nil {
		jsonOK(w, []tools.PendingQuestion{})
		return
	}
	questions := h.userInput.List(chi.URLParam(r, "id"))
	if questions == nil {
		questions = []tools.PendingQuestion{}
	}
	jsonOK(w, questions)
}

// uploadFiles POST /api/v1/workspace/upload
// Accepts multipart form data with files under the "files" key and saves them
// under {workspaceRoot}/uploads/. Returns relative paths.
func (h *ExecutionHandler) uploadFiles(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Error(w, "request too large", http.StatusBadRequest)
		return
	}

	root := h.workspaceRoot
	if root == "" {
		http.Error(w, "workspace not configured", http.StatusInternalServerError)
		return
	}

	uploadDir := filepath.Join(root, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		http.Error(w, "cannot create upload dir", http.StatusInternalServerError)
		return
	}

	var paths []string
	files := r.MultipartForm.File["files"]
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		// Sanitize filename
		name := filepath.Base(fh.Filename)
		dest := filepath.Join(uploadDir, name)
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			continue
		}
		paths = append(paths, filepath.Join("uploads", name))
	}

	jsonOK(w, map[string]any{"paths": paths})
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
