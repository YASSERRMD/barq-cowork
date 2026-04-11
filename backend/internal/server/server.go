package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server wraps the HTTP router and its configuration.
type Server struct {
	addr   string
	router *chi.Mux
	logger *slog.Logger
}

// New creates a new Server bound to addr.
func New(addr string, logger *slog.Logger) *Server {
	s := &Server{
		addr:   addr,
		router: chi.NewRouter(),
		logger: logger,
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

	// Allow frontend / Tauri to call backend from localhost.
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:1420")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	s.router.Get("/health", s.handleHealth)
}

// handleHealth returns a simple alive JSON response.
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
