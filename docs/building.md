# Building Barq Cowork

This document explains how to build the full desktop application from source for **macOS** and **Windows**.

---

## Prerequisites

| Tool | Minimum Version | Notes |
|------|-----------------|-------|
| **Go** | 1.22 | `go version` |
| **Rust + Cargo** | 1.77 (stable) | `rustup update stable` |
| **Node.js** | 20 LTS | `node --version` |
| **npm** | 10 | bundled with Node 20 |
| **Tauri CLI** | 2.x | `cargo install tauri-cli --version "^2"` |

### macOS extra requirements
* Xcode Command Line Tools — `xcode-select --install`
* For notarisation: Apple Developer account + provisioning profile

### Windows extra requirements
* Visual Studio 2022 Build Tools with the **Desktop development with C++** workload
* [WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) runtime (pre-installed on Windows 11)

---

## Repository layout

```
barq-cowork/
├── backend/               # Go service (barq-coworkd)
│   └── cmd/barq-coworkd/  # main package
├── apps/
│   └── desktop/           # Tauri + React frontend
│       └── src-tauri/
│           ├── binaries/  # place compiled sidecar here
│           └── icons/     # app icons (run gen-icons.go first)
├── scripts/
│   └── gen-icons.go       # placeholder icon generator
└── docs/
    └── building.md        # this file
```

---

## Step 1 — Generate icons

```bash
go run scripts/gen-icons.go
```

This creates placeholder icons in `apps/desktop/src-tauri/icons/`.
Replace them with professional artwork before publishing a release.

---

## Step 2 — Build the Go backend sidecar

The Tauri app bundles `barq-coworkd` as an external binary (sidecar).
The binary name **must** include the Rust target triple.

### macOS Apple Silicon (aarch64)
```bash
cd backend
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o ../apps/desktop/src-tauri/binaries/barq-coworkd-aarch64-apple-darwin \
  ./cmd/barq-coworkd
chmod +x ../apps/desktop/src-tauri/binaries/barq-coworkd-aarch64-apple-darwin
```

### macOS Intel (x86_64)
```bash
cd backend
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o ../apps/desktop/src-tauri/binaries/barq-coworkd-x86_64-apple-darwin \
  ./cmd/barq-coworkd
```

### Windows (x86_64)
```powershell
cd backend
$env:GOOS   = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" `
  -o ..\apps\desktop\src-tauri\binaries\barq-coworkd-x86_64-pc-windows-msvc.exe `
  .\cmd\barq-coworkd
```

---

## Step 3 — Install frontend dependencies

```bash
cd apps/desktop
npm ci
```

---

## Step 4 — Development run (hot-reload)

```bash
cd apps/desktop
npm run tauri dev
```

> **Note:** `tauri dev` will attempt to start the sidecar automatically.
> Make sure you built the sidecar in Step 2 first, otherwise the app will
> start in degraded mode (backend unreachable).

---

## Step 5 — Production build

```bash
cd apps/desktop
npm run tauri build
```

Built artefacts appear in:

| Platform | Path |
|----------|------|
| macOS `.dmg` | `apps/desktop/src-tauri/target/release/bundle/dmg/` |
| macOS `.app` | `apps/desktop/src-tauri/target/release/bundle/macos/` |
| Windows NSIS `.exe` | `apps/desktop/src-tauri/target/release/bundle/nsis/` |
| Windows MSI `.msi` | `apps/desktop/src-tauri/target/release/bundle/msi/` |

---

## Configuration

The Go backend is configured via environment variables and an optional
`barq.yaml` config file placed in the OS data directory
(`~/.local/share/barq-cowork/` on Linux/macOS,
`%APPDATA%\barq-cowork\` on Windows).

Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `BARQ_LISTEN_ADDR` | `127.0.0.1:7331` | HTTP listen address |
| `BARQ_DATA_DIR` | `~/.local/share/barq-cowork` | SQLite DB and artifact storage |
| `OPENAI_API_KEY` | — | OpenAI-compatible provider key |
| `ZAI_API_KEY` | — | Zai provider API key |

---

## macOS notarisation (optional)

Set the following GitHub Actions secrets (or local environment variables) to
enable code-signing and notarisation:

```
APPLE_CERTIFICATE                # base64-encoded .p12
APPLE_CERTIFICATE_PASSWORD
APPLE_SIGNING_IDENTITY           # e.g. "Developer ID Application: ..."
APPLE_ID                         # your Apple ID email
APPLE_PASSWORD                   # app-specific password
APPLE_TEAM_ID
```

## Windows code-signing (optional)

```
TAURI_SIGNING_PRIVATE_KEY        # base64-encoded private key
TAURI_SIGNING_PRIVATE_KEY_PASSWORD
```

---

## CI / Release

Pushing a tag matching `v*.*.*` triggers the [release workflow](../.github/workflows/release.yml).
The workflow:
1. Builds `barq-coworkd` for each platform in parallel.
2. Passes the sidecar binary to the corresponding Tauri build runner.
3. Publishes a GitHub Release with platform-specific installers.

To trigger a release manually:
```bash
git tag v0.2.0
git push origin v0.2.0
```
