package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/config"
	"github.com/barq-cowork/barq-cowork/internal/server"
	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/barq-cowork/barq-cowork/internal/store/sqlite"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Info("config loaded",
		"env", cfg.App.Env,
		"data_dir", cfg.App.DataDir,
		"llm_provider", cfg.LLM.DefaultProvider,
	)

	// Resolve ~ in sqlite path.
	dbPath := cfg.Storage.SQLitePath
	if strings.HasPrefix(dbPath, "~/") {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, dbPath[2:])
	}

	// Open SQLite database (runs migrations automatically).
	db, err := sqlite.Open(dbPath)
	if err != nil {
		logger.Error("failed to open database", "error", err, "path", dbPath)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database ready", "path", dbPath)

	// Wire repositories and services.
	workspaceRepo := sqlite.NewWorkspaceStore(db)
	projectRepo := sqlite.NewProjectStore(db)
	taskRepo := sqlite.NewTaskStore(db)

	svcs := server.Services{
		Workspaces: service.NewWorkspaceService(workspaceRepo),
		Projects:   service.NewProjectService(projectRepo, workspaceRepo),
		Tasks:      service.NewTaskService(taskRepo, projectRepo),
	}

	addr := ":7331"
	if v := os.Getenv("BARQ_LISTEN_ADDR"); v != "" {
		addr = v
	}

	srv := server.New(addr, logger, svcs)

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("barq-coworkd ready", "addr", addr)
	<-quit

	logger.Info("shutting down gracefully")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx
	logger.Info("stopped")
}
