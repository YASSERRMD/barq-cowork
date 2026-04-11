# Phase 1 — Monorepo and Foundations

## What was delivered

### Repository structure
```
barq-cowork/
  apps/desktop/           React + Tauri v2 desktop shell
  backend/                Go 1.22 backend daemon
  configs/                YAML config presets
  docs/                   Phase docs
  .gitignore
  README.md
```

### Go backend (`backend/`)
- `cmd/barq-coworkd/main.go` — entry point with graceful shutdown
- `internal/server/server.go` — chi-based HTTP router, CORS, `GET /health`
- `internal/config/config.go` — typed config structs
- `internal/config/loader.go` — layered config: env > file > defaults

### Config system
- Reads `configs/local.yaml` → `configs/default.yaml` → built-in defaults
- `BARQ_*`, `ZAI_*`, `OPENAI_*` env vars override file values at runtime
- API keys are **never** stored directly; only env var names are stored
- `config.ResolveAPIKey()` reads the actual key in backend code only

### Tauri v2 shell (`apps/desktop/src-tauri/`)
- `src/lib.rs` — two commands registered: `health_check`, `app_version`
- `health_check` calls `http://localhost:7331/health` and returns structured result
- `capabilities/default.json` — minimal Tauri v2 permission set
- `tauri.conf.json` — window config, sidecar config, bundle targets

### React frontend (`apps/desktop/src/`)
- Vite + React 18 + TypeScript + Tailwind CSS
- React Router v6 for page navigation
- Zustand store (`appStore`) for backend status and app version
- React Query for future async data fetching
- `lib/tauri.ts` — typed wrappers around `invoke()`
- Pages: Workspaces, Tasks, Artifacts, Approvals, Logs, Settings (all scaffold-level)
- Sidebar with nav links and live backend status indicator
- `App.tsx` polls `/health` every 10s and updates the store

## How to validate Phase 1

1. **Go backend compiles and runs:**
   ```sh
   cd backend && go build ./... && go run ./cmd/barq-coworkd
   curl http://localhost:7331/health
   # → {"status":"ok","service":"barq-coworkd","timestamp":"..."}
   ```

2. **Desktop app launches (requires Rust + pnpm):**
   ```sh
   cd apps/desktop && pnpm install && pnpm tauri dev
   # → Window opens, sidebar visible, backend status dot turns green
   ```

3. **Config loads from file:**
   ```sh
   cp configs/default.yaml configs/local.yaml
   cd backend && go run ./cmd/barq-coworkd
   # → logs show "config loaded" with provider=zai
   ```

4. **Env override works:**
   ```sh
   BARQ_LLM_PROVIDER=openai go run ./cmd/barq-coworkd
   # → logs show provider=openai
   ```

## Decisions and trade-offs

| Decision | Rationale |
|----------|-----------|
| Backend runs as standalone daemon on `:7331` | Simplest Phase 1 approach; Tauri sidecar registration can be added in Phase 5 when the backend is more complete |
| `modernc.org/sqlite` will be used (Phase 2) | Pure-Go SQLite — no CGO, simpler cross-compilation for macOS + Windows |
| `chi` for HTTP routing | Minimal, stdlib-compatible, no magic |
| `gopkg.in/yaml.v3` for config | Standard, battle-tested YAML parsing |
| Tailwind CSS for UI | Utility-first, no runtime — practical for developer tooling |
| Zustand for state | Minimal boilerplate, no Provider wrapping needed |
