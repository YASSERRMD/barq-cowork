use std::sync::Mutex;

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, RunEvent};
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};

// ─────────────────────────────────────────────────────────────────────────────
// Managed state
// ─────────────────────────────────────────────────────────────────────────────

/// Holds the running barq-coworkd sidecar process so we can kill it on exit.
struct SidecarChild(Mutex<Option<CommandChild>>);

/// Cached backend URL so commands can surface it to the frontend.
struct BackendUrl(String);

// ─────────────────────────────────────────────────────────────────────────────
// DTOs
// ─────────────────────────────────────────────────────────────────────────────

#[derive(Debug, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub backend: BackendStatus,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct BackendStatus {
    pub reachable: bool,
    pub message: String,
}

// ─────────────────────────────────────────────────────────────────────────────
// Tauri commands
// ─────────────────────────────────────────────────────────────────────────────

/// Returns the URL at which the embedded barq-coworkd backend is listening.
#[tauri::command]
fn get_backend_url(app: AppHandle) -> String {
    app.state::<BackendUrl>().0.clone()
}

/// Probes the Go backend's /health endpoint and returns the result.
/// Falls back gracefully if the backend is not yet running.
#[tauri::command]
async fn health_check(app: AppHandle) -> Result<HealthResponse, String> {
    let base_url = app.state::<BackendUrl>().0.clone();
    let url = format!("{base_url}/health");

    match reqwest::get(&url).await {
        Ok(resp) if resp.status().is_success() => {
            let body: serde_json::Value = resp.json().await.unwrap_or_default();
            Ok(HealthResponse {
                status: "ok".into(),
                backend: BackendStatus {
                    reachable: true,
                    message: body
                        .get("status")
                        .and_then(|v| v.as_str())
                        .unwrap_or("ok")
                        .to_string(),
                },
            })
        }
        Ok(resp) => Ok(HealthResponse {
            status: "degraded".into(),
            backend: BackendStatus {
                reachable: false,
                message: format!("backend returned HTTP {}", resp.status()),
            },
        }),
        Err(e) => Ok(HealthResponse {
            status: "degraded".into(),
            backend: BackendStatus {
                reachable: false,
                message: format!("backend unreachable: {e}"),
            },
        }),
    }
}

/// Returns the current application version from Cargo.toml.
#[tauri::command]
fn app_version(app: AppHandle) -> String {
    app.package_info().version.to_string()
}

// ─────────────────────────────────────────────────────────────────────────────
// Sidecar lifecycle
// ─────────────────────────────────────────────────────────────────────────────

const BACKEND_ADDR: &str = "127.0.0.1:7331";

fn start_sidecar(app: &AppHandle) {
    let shell = app.shell();

    let cmd = match shell.sidecar("barq-coworkd") {
        Ok(c) => c,
        Err(e) => {
            eprintln!("[barq] failed to locate sidecar binary: {e}");
            return;
        }
    };

    // Pass the listen address as an environment variable so barq-coworkd
    // does not need positional args — existing BARQ_LISTEN_ADDR support.
    let cmd = cmd.env("BARQ_LISTEN_ADDR", format!("{BACKEND_ADDR}"));

    match cmd.spawn() {
        Ok((mut rx, child)) => {
            // Store child so we can kill it on exit.
            if let Some(state) = app.try_state::<SidecarChild>() {
                *state.0.lock().unwrap() = Some(child);
            }

            // Forward sidecar stdout/stderr to the host console asynchronously.
            tauri::async_runtime::spawn(async move {
                while let Some(event) = rx.recv().await {
                    match event {
                        CommandEvent::Stdout(line) => {
                            print!("[barq-coworkd] {}", String::from_utf8_lossy(&line));
                        }
                        CommandEvent::Stderr(line) => {
                            eprint!("[barq-coworkd:err] {}", String::from_utf8_lossy(&line));
                        }
                        CommandEvent::Error(msg) => {
                            eprintln!("[barq-coworkd:ipc-error] {msg}");
                        }
                        CommandEvent::Terminated(status) => {
                            eprintln!(
                                "[barq-coworkd] process terminated, code={:?} signal={:?}",
                                status.code, status.signal
                            );
                            break;
                        }
                        _ => {}
                    }
                }
            });

            println!("[barq] sidecar started, listening on {BACKEND_ADDR}");
        }
        Err(e) => {
            eprintln!("[barq] failed to spawn sidecar: {e}");
        }
    }
}

fn kill_sidecar(app: &AppHandle) {
    if let Some(state) = app.try_state::<SidecarChild>() {
        if let Ok(mut guard) = state.0.lock() {
            if let Some(child) = guard.take() {
                if let Err(e) = child.kill() {
                    eprintln!("[barq] failed to kill sidecar: {e}");
                } else {
                    println!("[barq] sidecar stopped");
                }
            }
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// App entry-point
// ─────────────────────────────────────────────────────────────────────────────

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let backend_url = format!("http://{BACKEND_ADDR}/api/v1");

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_http::init())
        // Register managed state before setup so commands can access it.
        .manage(SidecarChild(Mutex::new(None)))
        .manage(BackendUrl(backend_url))
        .setup(|app| {
            start_sidecar(app.handle());
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            health_check,
            app_version,
            get_backend_url
        ])
        .build(tauri::generate_context!())
        .expect("error building Barq Cowork")
        .run(|app, event| {
            if let RunEvent::ExitRequested { .. } = event {
                kill_sidecar(app);
            }
        });
}
