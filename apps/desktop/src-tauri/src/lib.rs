use serde::{Deserialize, Serialize};
use tauri::Manager;

/// Response returned by the health_check Tauri command.
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

/// health_check calls the Go backend's /health endpoint and returns the result.
/// Falls back gracefully if the backend is not yet running.
#[tauri::command]
async fn health_check(backend_url: String) -> Result<HealthResponse, String> {
    let url = format!("{}/health", backend_url.trim_end_matches('/'));

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

/// app_version returns the current application version from Cargo.toml.
#[tauri::command]
fn app_version(app: tauri::AppHandle) -> String {
    app.package_info().version.to_string()
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_http::init())
        .invoke_handler(tauri::generate_handler![health_check, app_version])
        .run(tauri::generate_context!())
        .expect("error while running Barq Cowork");
}
