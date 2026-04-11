/**
 * Thin wrappers around Tauri IPC commands.
 * Falls back to direct HTTP fetch when running outside the Tauri runtime
 * (e.g. `vite dev` in a plain browser).
 */

export interface BackendStatus {
  reachable: boolean;
  message: string;
}

export interface HealthResponse {
  status: string;
  backend: BackendStatus;
}

const BACKEND_URL = "http://localhost:7331";

/** Returns true when running inside the Tauri desktop shell. */
function isTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

/** Call the Go backend health endpoint.
 *  - Inside Tauri: routes through the Rust sidecar command.
 *  - In browser dev mode: direct HTTP fetch to localhost:7331.
 */
export async function checkHealth(): Promise<HealthResponse> {
  if (isTauri()) {
    const { invoke } = await import("@tauri-apps/api/core");
    return invoke<HealthResponse>("health_check", { backendUrl: BACKEND_URL });
  }

  // Browser / Vite dev fallback — hit the backend directly
  const res = await fetch(`${BACKEND_URL}/health`);
  if (!res.ok) throw new Error(`health ${res.status}`);
  const data = await res.json() as { status: string };
  return {
    status: data.status,
    backend: { reachable: true, message: "ok" },
  };
}

/** Get the desktop app version.
 *  Returns "dev" when running outside Tauri.
 */
export async function getAppVersion(): Promise<string> {
  if (isTauri()) {
    const { invoke } = await import("@tauri-apps/api/core");
    return invoke<string>("app_version");
  }
  return "dev";
}
