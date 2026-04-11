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
	"github.com/barq-cowork/barq-cowork/internal/memory"
	"github.com/barq-cowork/barq-cowork/internal/orchestrator"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	zaiprovider "github.com/barq-cowork/barq-cowork/internal/provider/openai"
	oaiprovider "github.com/barq-cowork/barq-cowork/internal/provider/zai"
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
		logger.Error("failed to open database", "path", dbPath, "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database ready", "path", dbPath)

	// ── Startup health checks ──────────────────────────────────────────
	// Ensure data directory is writable.
	dataDir := cfg.App.DataDir
	if strings.HasPrefix(dataDir, "~/") {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, dataDir[2:])
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		logger.Error("data directory not writable", "path", dataDir, "error", err)
		os.Exit(1)
	}

	// ── Task recovery ─────────────────────────────────────────────────
	// Reset any tasks that were stuck in planning/running from a previous
	// crash or forced shutdown so they don't appear perpetually in-progress.
	taskRepo := sqlite.NewTaskStore(db)
	if recovered, err := taskRepo.RecoverStuck(context.Background()); err != nil {
		logger.Warn("task recovery failed", "error", err)
	} else if recovered > 0 {
		logger.Info("task recovery: reset stuck tasks", "count", recovered)
	}

	// ── Provider registry ──────────────────────────────────────────────
	registry := provider.NewRegistry()
	registry.Register(oaiprovider.New(120)) // zai
	registry.Register(zaiprovider.New(120)) // openai
	logger.Info("providers registered", "providers", registry.List())

	// ── Tool registry ──────────────────────────────────────────────────
	toolRegistry := service.BuildRegistry()
	logger.Info("tools registered", "tools", toolRegistry.List())

	// ── Repositories ──────────────────────────────────────────────────
	workspaceRepo       := sqlite.NewWorkspaceStore(db)
	projectRepo         := sqlite.NewProjectStore(db)
	// taskRepo already created above for startup recovery.
	providerProfileRepo := sqlite.NewProviderProfileStore(db)
	scheduleRepo        := sqlite.NewScheduleStore(db)
	approvalRepo        := sqlite.NewApprovalStore(db)
	eventRepo           := sqlite.NewEventStore(db)
	planStore           := sqlite.NewPlanStore(db)
	artifactStore       := sqlite.NewArtifactStore(db)
	contextFileStore    := sqlite.NewContextFileStore(db)
	taskTemplateStore   := sqlite.NewTaskTemplateStore(db)
	subAgentStore       := sqlite.NewSubAgentStore(db)

	// ── Orchestrator ──────────────────────────────────────────────────
	wsMemory    := memory.New(contextFileStore)
	planner     := orchestrator.NewPlanner(registry, wsMemory, logger)
	executor    := orchestrator.NewExecutor(planStore, artifactStore, eventRepo, toolRegistry, logger)
	subAgentOrch := orchestrator.NewSubAgentOrchestrator(
		subAgentStore, planner, executor, planStore, eventRepo, logger,
	)
	orch     := orchestrator.New(
		taskRepo, projectRepo, providerProfileRepo,
		planner, executor, planStore,
		cfg, logger,
	)

	// ── Services ──────────────────────────────────────────────────────
	svcs := server.Services{
		Workspaces: service.NewWorkspaceService(workspaceRepo),
		Projects:   service.NewProjectService(projectRepo, workspaceRepo),
		Tasks:      service.NewTaskService(taskRepo, projectRepo),
		Providers:  service.NewProviderService(providerProfileRepo, registry, cfg),
		Schedules:  service.NewScheduleService(scheduleRepo, projectRepo),
		Tools:      service.NewToolService(toolRegistry, approvalRepo, eventRepo),
		Execution: server.ExecutionDeps{
			Runner:    orch,
			Plans:     planStore,
			Artifacts: artifactStore,
			Events:    eventRepo,
		},
		Memory: server.MemoryDeps{
			ContextFiles:  contextFileStore,
			TaskTemplates: taskTemplateStore,
		},
		Agents: server.AgentDeps{
			Runner: subAgentOrch,
			DefaultProvider: func() provider.ProviderConfig {
				provName := cfg.LLM.DefaultProvider
				pc, ok := cfg.LLM.Providers[provName]
				if !ok {
					return provider.ProviderConfig{}
				}
				return provider.ProviderConfig{
					ProviderName: provName,
					BaseURL:      pc.BaseURL,
					APIKey:       os.Getenv(pc.APIKeyEnv),
					Model:        pc.Model,
					TimeoutSec:   pc.TimeoutSec,
					ExtraHeaders: pc.ExtraHeaders,
				}
			},
		},
		Diagnostics: server.DiagnosticDeps{
			Events:    eventRepo,
			Artifacts: artifactStore,
			Version:   "0.1.0",
		},
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
