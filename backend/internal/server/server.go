package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	v1 "github.com/barq-cowork/barq-cowork/internal/api/v1"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Services bundles all application services the server depends on.
type Services struct {
	Workspaces *service.WorkspaceService
	Projects   *service.ProjectService
	Tasks      *service.TaskService
	Providers  *service.ProviderService
	Schedules  *service.ScheduleService
	Tools      *service.ToolService
	Skills     *service.SkillService
	Execution   ExecutionDeps
	Memory      MemoryDeps
	Agents      AgentDeps
	Diagnostics DiagnosticDeps
}

// ExecutionDeps groups the ports needed by the execution HTTP handler.
type ExecutionDeps struct {
	Runner        v1.TaskRunner
	Plans         v1.PlanQuerier
	Artifacts     v1.ArtifactQuerier
	Events        v1.EventQuerier
	WorkspaceRoot string // base path for serving artifact files
}

// MemoryDeps groups the ports needed by the memory HTTP handler.
type MemoryDeps struct {
	ContextFiles  v1.ContextFileStore
	TaskTemplates v1.TaskTemplateStore
}

// DiagnosticDeps groups the ports needed by the diagnostics handler.
type DiagnosticDeps struct {
	Events    v1.EventQuerier
	Artifacts v1.ArtifactQuerier
	Version   string
}

// AgentDeps groups the ports needed by the sub-agent HTTP handler.
type AgentDeps struct {
	Runner          v1.SubAgentRunner
	DefaultProvider func() provider.ProviderConfig
}

// Server wraps the HTTP router and its configuration.
type Server struct {
	addr     string
	router   *chi.Mux
	logger   *slog.Logger
	services Services
}

// New creates a new Server bound to addr, wired with the given services.
func New(addr string, logger *slog.Logger, svcs Services) *Server {
	s := &Server{
		addr:     addr,
		router:   chi.NewRouter(),
		logger:   logger,
		services: svcs,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	// CORS: allow Tauri webview and dev frontend.
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "http://localhost:1420" || origin == "tauri://localhost" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Core endpoints
	s.router.Get("/health", s.handleHealth)

	// API v1
	s.router.Route("/api/v1", func(r chi.Router) {
		v1.NewWorkspaceHandler(s.services.Workspaces).Register(r)
		v1.NewProjectHandler(s.services.Projects).Register(r)
		v1.NewTaskHandler(s.services.Tasks).Register(r)
		v1.NewProviderHandler(s.services.Providers).Register(r)
		v1.NewScheduleHandler(s.services.Schedules).Register(r)
		v1.NewToolHandler(s.services.Tools).Register(r)
		v1.NewSkillHandler(s.services.Skills).Register(r)
		v1.NewExecutionHandler(
			s.services.Execution.Runner,
			s.services.Execution.Plans,
			s.services.Execution.Artifacts,
			s.services.Execution.Events,
			s.services.Execution.WorkspaceRoot,
		).Register(r)
		v1.NewMemoryHandler(
			s.services.Memory.ContextFiles,
			s.services.Memory.TaskTemplates,
		).Register(r)
		v1.NewAgentsHandler(
			s.services.Agents.Runner,
			s.services.Agents.DefaultProvider,
		).Register(r)
		v1.NewDiagnosticsHandler(
			s.services.Diagnostics.Events,
			s.services.Diagnostics.Artifacts,
			s.services.Diagnostics.Version,
		).Register(r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"service":   "barq-coworkd",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ListenAndServe starts the HTTP server. Blocks until error.
func (s *Server) ListenAndServe() error {
	s.logger.Info("barq-coworkd starting", "addr", s.addr)
	return http.ListenAndServe(s.addr, s.router)
}
