<div align="center">

<img src="docs/banner.png" alt="Barq Cowork" width="100%" />

[![Release](https://img.shields.io/github/v/release/YASSERRMD/barq-cowork?style=flat-square&color=f97316)](https://github.com/YASSERRMD/barq-cowork/releases)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Rust](https://img.shields.io/badge/Rust-stable-CE422B?style=flat-square&logo=rust&logoColor=white)](https://www.rust-lang.org/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=black)](https://react.dev/)
[![Tauri](https://img.shields.io/badge/Tauri-v2-FFC131?style=flat-square&logo=tauri&logoColor=black)](https://tauri.app/)
[![License](https://img.shields.io/badge/License-MIT-6366f1?style=flat-square)](LICENSE)

**Your AI team, on your machine**

*A cross-platform desktop AI agent workspace for outcome-based tasks*

[Features](#features) · [Architecture](#architecture) · [Getting Started](#getting-started) · [Building](#building) · [API Reference](#api-reference) · [Contributing](#contributing)

</div>

---

## What is Barq Cowork?

Barq Cowork is a desktop application that turns natural-language task descriptions into multi-step, tool-using AI plans — then executes them. Think of it as a local command centre for AI agents: you define projects, attach context files, choose an LLM provider, and let specialised agents plan and carry out work in parallel while you watch the live timeline.

Everything runs on your machine. The backend is a single self-contained Go binary (`barq-coworkd`) bundled inside the Tauri desktop shell; no cloud account is required beyond your LLM API key.

---

## Features

| Feature | Description |
|---------|-------------|
| Multi-agent orchestration | Spawn parallel sub-agents (Researcher, Writer, Coder, Reviewer, Analyst) each with an isolated plan and tool access |
| Live plan timeline | Watch every step execute in real-time with tool calls, output, and status badges |
| Project memory | Attach context files and reusable task templates to any project |
| Tool system | File operations, shell commands, web search, and a human-approval gate |
| Provider flexibility | Built-in support for OpenAI, Anthropic, Gemini, Z.AI, Ollama, and any OpenAI-compatible endpoint — API keys stored locally, no env vars needed |
| Artifact management | Automatic capture, storage, and browsing of files produced by agents |
| Diagnostics | Runtime stats, goroutine counts, and one-click log-bundle download |
| Cross-platform | macOS (Apple Silicon + Intel) and Windows 10/11 |

---

## Architecture

```
+------------------------------------------------------------------+
|                   Tauri Desktop Shell (Rust)                     |
|  +------------------------------------------------------------+  |
|  |             React + TypeScript Frontend                    |  |
|  |  Workspaces -> Projects -> Tasks -> Plan Timeline          |  |
|  |  Sub-Agent Panel  Artifacts  Logs  Diagnostics             |  |
|  +-------------------------+----------------------------------+  |
|                            | HTTP (localhost:7331)               |
+----------------------------+------------------------------------+
                             | sidecar process
+----------------------------v------------------------------------+
|                    barq-coworkd  (Go)                           |
|                                                                  |
|  +-------------+   +--------------+   +---------------------+   |
|  |  REST API   |   | Orchestrator |   |   Provider Layer    |   |
|  | /api/v1/*   +-->|  Planner +   +-->|  OpenAI-compatible  |   |
|  | Chi router  |   |  Executor +  |   |  + retry/backoff    |   |
|  +-------------+   |  Sub-Agents  |   +---------------------+   |
|                    +------+-------+                              |
|  +----------------------------+------------------------------+   |
|  |         SQLite  (modernc.org/sqlite)                      |   |
|  |  workspaces  projects  tasks  plans  steps  events        |   |
|  |  artifacts  sub_agents  context_files  task_templates     |   |
|  |  tool_approvals  provider_profiles                        |   |
|  +-----------------------------------------------------------+   |
+------------------------------------------------------------------+
```

### Key design decisions

- **Hexagonal architecture** — domain types live in `internal/domain`; all I/O goes through narrow port interfaces; adapters (sqlite, providers) are swappable.
- **Sidecar pattern** — Tauri spawns `barq-coworkd` as a managed child process; on app exit the process is killed cleanly.
- **Detached goroutines** — task and sub-agent execution run in background goroutines; the HTTP layer returns `202 Accepted` immediately and the frontend polls for progress.
- **Embedded migrations** — SQLite schema migrations are embedded Go files applied automatically at startup; no external migration tool needed.
- **Provider retry** — all LLM calls use exponential back-off with jitter; 429/5xx/timeout errors are retried automatically up to 3 times.

---

## Getting Started

### Prerequisites

- macOS 12+ or Windows 10/11
- An API key for any OpenAI-compatible LLM provider

### Download

Grab the latest installer from [Releases](https://github.com/YASSERRMD/barq-cowork/releases):

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `Barq_Cowork_*_aarch64.dmg` |
| macOS (Intel) | `Barq_Cowork_*_x64.dmg` |
| Windows 10/11 | `Barq_Cowork_*_x64-setup.exe` |

### First run

1. Launch the app — the backend starts automatically in the background.
2. Open **Settings** and add your LLM provider. Enter the API key directly — no environment variables needed.
3. Create a **Project** and write a name, description, and optional system instructions.
4. Create a **Task** and click **Run** — watch the plan unfold live in the timeline.
5. Browse produced **Artifacts** or check the **Logs** for the full event trail.

### Configuration

API keys and provider settings are configured entirely inside the app under **Settings > Providers**. No environment variables or config files are required for normal use.

For advanced overrides, an optional `barq.yaml` can be placed in:

| OS | Path |
|----|------|
| macOS / Linux | `~/.local/share/barq-cowork/` |
| Windows | `%APPDATA%\barq-cowork\` |

The only environment variable you may need:

| Variable | Default | Description |
|----------|---------|-------------|
| `BARQ_LISTEN_ADDR` | `127.0.0.1:7331` | Override the backend listen address |

---

## Building

See **[docs/building.md](docs/building.md)** for the full guide. Quick summary:

```bash
# 1. Build the Go sidecar (macOS ARM example)
cd backend
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o ../apps/desktop/src-tauri/binaries/barq-coworkd-aarch64-apple-darwin \
  ./cmd/barq-coworkd

# 2. Install frontend dependencies
cd ../apps/desktop && npm ci

# 3. Dev run (hot-reload)
npm run tauri dev

# 4. Production bundle
npm run tauri build
```

CI/CD is handled by [`.github/workflows/release.yml`](.github/workflows/release.yml). Push a `v*.*.*` tag to trigger a cross-platform release build.

---

## API Reference

The backend exposes a JSON REST API at `http://localhost:7331/api/v1`.

### Core resources

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/workspaces` | List workspaces |
| `POST` | `/workspaces` | Create workspace |
| `GET` | `/workspaces/:id/projects` | List projects |
| `POST` | `/projects` | Create project |
| `GET` | `/projects/:id/tasks` | List tasks |
| `POST` | `/tasks` | Create task |

### Execution

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/tasks/:id/run` | Start async execution (202) |
| `GET` | `/tasks/:id/plan` | Fetch generated plan + steps |
| `GET` | `/tasks/:id/events` | Task execution events |
| `GET` | `/tasks/:id/artifacts` | Artifacts produced by task |
| `GET` | `/events?limit=N` | Global event log |
| `GET` | `/artifacts?limit=N` | Global artifact list |

### Sub-agents

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/tasks/:id/agents` | Spawn parallel sub-agents (202) |
| `GET` | `/tasks/:id/agents` | List sub-agents and status |
| `DELETE` | `/tasks/:id/agents/:agentId` | Cancel a sub-agent |

### Memory

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/projects/:id/context-files` | List context files |
| `POST` | `/projects/:id/context-files` | Attach context file |
| `PUT` | `/context-files/:id` | Update context file |
| `GET` | `/projects/:id/templates` | List task templates |
| `POST` | `/projects/:id/templates` | Create template |

### Diagnostics

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/diagnostics/info` | Runtime stats JSON |
| `GET` | `/diagnostics/bundle` | Download diagnostic ZIP |

### Tool approvals

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/tools/approvals` | List pending approvals |
| `POST` | `/tools/approvals/:id/approve` | Approve a tool call |
| `POST` | `/tools/approvals/:id/reject` | Reject a tool call |

---

## Repository Structure

```
barq-cowork/
├── backend/                        # Go service
│   ├── cmd/barq-coworkd/main.go    # Entry point
│   └── internal/
│       ├── domain/                 # Core types, errors
│       ├── config/                 # YAML + env config
│       ├── provider/               # LLM provider abstraction + retry
│       │   ├── openai/             # OpenAI-compatible adapter
│       │   └── zai/                # Zai provider adapter
│       ├── orchestrator/           # Planner, Executor, Sub-agent pool
│       ├── service/                # Business logic + tool registry
│       ├── store/sqlite/           # SQLite adapters + migrations
│       ├── memory/                 # Workspace memory (context injection)
│       ├── api/v1/                 # HTTP handlers (Chi)
│       └── server/                 # Router assembly + CORS
│
├── apps/desktop/                   # Tauri + React application
│   ├── src/
│   │   ├── pages/                  # Route-level page components
│   │   ├── components/             # Shared UI components
│   │   ├── lib/api.ts              # Typed REST client
│   │   └── store/appStore.ts       # Zustand global state
│   └── src-tauri/
│       ├── src/lib.rs              # Sidecar lifecycle manager
│       ├── icons/                  # App icons (all platform formats)
│       └── tauri.conf.json         # Tauri configuration
│
├── docs/
│   ├── banner.png                  # GitHub repository banner
│   └── building.md                 # Build guide
├── scripts/gen-icons.py            # Icon generator script
└── .github/workflows/release.yml  # CI/CD release workflow
```

---

## Roadmap

- [x] In-app API key storage — no environment variables required
- [x] Built-in Anthropic, Gemini, Ollama, OpenAI, and Z.AI providers
- [x] Schedules — recurring tasks with cron expressions
- [x] Connectors view — browse and test all configured providers
- [x] Command palette — keyboard-first navigation
- [ ] Vector-based workspace memory (semantic search over context files)
- [ ] Real-time WebSocket event stream (replace polling)
- [ ] Agent-to-agent communication protocol
- [ ] Plugin system for custom tools
- [ ] Apple code signing (remove Gatekeeper prompt on macOS)

---

## Contributing

Contributions are welcome. Please:

1. Fork the repository and create a feature branch.
2. Follow the existing code style (`gofmt` for Go, ESLint for TypeScript).
3. Add or update tests for any changed behaviour.
4. Open a pull request against `main` with a clear description.

For major changes, open an issue first to discuss the design.

---

## License

MIT — see [LICENSE](LICENSE).

---

<div align="center">
  <sub>Built with Go · Rust · React · Tauri · SQLite</sub>
</div>
