/**
 * Thin wrappers around Tauri IPC commands.
 * All backend communication goes through this module.
 */
import { invoke } from "@tauri-apps/api/core";

export interface BackendStatus {
  reachable: boolean;
  message: string;
}

export interface HealthResponse {
  status: string;
  backend: BackendStatus;
}

const BACKEND_URL = "http://localhost:7331";

/** Call the Go backend health endpoint via the Rust sidecar command. */
export async function checkHealth(): Promise<HealthResponse> {
  return invoke<HealthResponse>("health_check", { backendUrl: BACKEND_URL });
}

/** Get the desktop app version from Tauri. */
export async function getAppVersion(): Promise<string> {
  return invoke<string>("app_version");
}
