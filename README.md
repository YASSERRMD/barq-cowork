# Barq Cowork

A cross-platform desktop agent workspace for macOS and Windows, built on Go + Tauri v2 + React.

## Architecture

```
barq-cowork/
  apps/
    desktop/
      src/            # React + TypeScript frontend (Vite)
      src-tauri/      # Tauri v2 Rust shell
  backend/
    cmd/barq-coworkd/ # Go HTTP backend daemon
    internal/
      config/         # Layered config loader (YAML + env)
      server/         # HTTP server + routes
    pkg/
  configs/
    default.yaml      # Default config (Z.AI coding API)
    default.zai-general.yaml  # Alt preset: Z.AI general API
  docs/
```

## Quick Start

### Prerequisites

| Tool | Version |
|------|---------|
| Go   | 1.22+   |
| Rust | stable  |
| Node | 20+     |
| pnpm | 9+      |
| Tauri CLI | v2 |

Install Tauri CLI:
```sh
cargo install tauri-cli --version "^2"
```

### 1. Run the Go backend

```sh
cd backend
go run ./cmd/barq-coworkd
# Listens on :7331 by default
```

Override the listen address:
```sh
BARQ_LISTEN_ADDR=:8080 go run ./cmd/barq-coworkd
```

### 2. Configure LLM providers (env vars)

```sh
# Z.AI (default)
export ZAI_API_KEY=your_key_here
export ZAI_MODEL=GLM-4.7                             # optional override
export ZAI_BASE_URL=https://api.z.ai/api/coding/paas/v4  # optional override

# OpenAI (fallback)
export OPENAI_API_KEY=your_key_here
```

### 3. Run the desktop app

```sh
cd apps/desktop
pnpm install
pnpm tauri dev
```

The Tauri app will launch, load the React frontend at `http://localhost:1420`, and probe the Go backend health endpoint.

### 4. Build for release

```sh
cd apps/desktop
pnpm tauri build
# Outputs to apps/desktop/src-tauri/target/release/bundle/
```

## Config File

Copy and edit `configs/default.yaml`:
```sh
cp configs/default.yaml configs/local.yaml
# Edit configs/local.yaml with your settings
```

The config loader searches in order:
1. `$BARQ_CONFIG_FILE`
2. `configs/local.yaml`
3. `configs/default.yaml`

Environment variables always override file values.

## Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `BARQ_APP_ENV` | `development` or `production` | `development` |
| `BARQ_DATA_DIR` | Local data storage directory | `~/.barq-cowork` |
| `BARQ_LISTEN_ADDR` | Backend HTTP listen address | `:7331` |
| `BARQ_LLM_PROVIDER` | Default provider name | `zai` |
| `ZAI_API_KEY` | Z.AI API key | — |
| `ZAI_BASE_URL` | Z.AI base URL override | see config |
| `ZAI_MODEL` | Z.AI model override | `GLM-4.7` |
| `OPENAI_API_KEY` | OpenAI API key | — |
| `OPENAI_BASE_URL` | OpenAI base URL override | `https://api.openai.com/v1` |
| `OPENAI_MODEL` | OpenAI model override | `gpt-4.1` |
| `BARQ_REQUIRE_APPROVAL` | Require approval for destructive actions | `true` |

## Health Check

```sh
curl http://localhost:7331/health
# {"status":"ok","service":"barq-coworkd","timestamp":"..."}
```

## Roadmap

| Phase | Description |
|-------|-------------|
| ✅ 1 | Monorepo foundations, Go backend, Tauri shell, React frontend |
| ⬜ 2 | Domain model + SQLite persistence |
| ⬜ 3 | Z.AI + OpenAI provider layer |
| ⬜ 4 | Tool registry + safe file operations |
| ⬜ 5 | Task planning + execution engine |
| ⬜ 6 | Desktop UX maturity |
| ⬜ 7 | Project memory + reusable context |
| ⬜ 8 | Sub-agents + parallel execution |
| ⬜ 9 | macOS + Windows packaging |
| ⬜ 10 | Hardening + diagnostics |
