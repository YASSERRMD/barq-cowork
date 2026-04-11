/**
 * Typed REST API client for the barq-coworkd backend.
 * The frontend calls the Go backend directly over HTTP (CORS is configured).
 */

const BASE = "http://localhost:7331/api/v1";

// ──────────────────── Types ────────────────────

export interface Workspace {
  id: string;
  name: string;
  description: string;
  root_path: string;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  instructions: string;
  created_at: string;
  updated_at: string;
}

export type TaskStatus =
  | "pending"
  | "planning"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface Task {
  id: string;
  project_id: string;
  title: string;
  description: string;
  status: TaskStatus;
  provider_id: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
}

// ──────────────────── HTTP helpers ────────────────────

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body.error ?? `HTTP ${res.status}`);
  }
  return body.data as T;
}

// ──────────────────── Tools + Approvals + Events ────────────────────

export interface ToolInfo {
  name: string;
  description: string;
  input_schema: Record<string, unknown>;
}

export interface ToolResult {
  status: "ok" | "error" | "denied" | "pending";
  content: string;
  data?: unknown;
  error?: string;
}

export interface Approval {
  id: string;
  task_id: string;
  tool_name: string;
  action: string;
  payload: string;
  status: "pending" | "approved" | "rejected";
  resolution?: string;
  created_at: string;
}

export interface TaskEvent {
  id: string;
  task_id: string;
  type: string;
  payload: string;
  created_at: string;
}

export const toolsApi = {
  list: (): Promise<ToolInfo[]> => request("/tools"),

  invoke: (data: {
    task_id?: string;
    workspace_root: string;
    tool_name: string;
    args_json: string;
    require_approval?: boolean;
  }): Promise<ToolResult> =>
    request("/tools/invoke", { method: "POST", body: JSON.stringify(data) }),

  listApprovals: (): Promise<Approval[]> => request("/approvals"),

  resolveApproval: (id: string, resolution: "approved" | "rejected"): Promise<void> =>
    request(`/approvals/${id}/resolve`, {
      method: "POST",
      body: JSON.stringify({ resolution }),
    }),

  listEvents: (taskId: string): Promise<TaskEvent[]> =>
    request(`/tasks/${taskId}/events`),
};

// ──────────────────── Workspaces ────────────────────

export const workspacesApi = {
  list: (): Promise<Workspace[]> => request("/workspaces"),

  get: (id: string): Promise<Workspace> => request(`/workspaces/${id}`),

  create: (data: {
    name: string;
    description?: string;
    root_path?: string;
  }): Promise<Workspace> =>
    request("/workspaces", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    id: string,
    data: { name: string; description?: string; root_path?: string }
  ): Promise<Workspace> =>
    request(`/workspaces/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  delete: (id: string): Promise<void> =>
    request(`/workspaces/${id}`, { method: "DELETE" }),
};

// ──────────────────── Projects ────────────────────

export const projectsApi = {
  listByWorkspace: (workspaceID: string): Promise<Project[]> =>
    request(`/workspaces/${workspaceID}/projects`),

  get: (id: string): Promise<Project> => request(`/projects/${id}`),

  create: (data: {
    workspace_id: string;
    name: string;
    description?: string;
    instructions?: string;
  }): Promise<Project> =>
    request("/projects", { method: "POST", body: JSON.stringify(data) }),

  update: (
    id: string,
    data: { name: string; description?: string; instructions?: string }
  ): Promise<Project> =>
    request(`/projects/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  delete: (id: string): Promise<void> =>
    request(`/projects/${id}`, { method: "DELETE" }),
};

// ──────────────────── Providers ────────────────────

export interface AvailableProvider {
  name: string;
  enabled: boolean;
  base_url: string;
  model: string;
  has_key: boolean;
  key_env: string;
}

export interface ProviderProfile {
  id: string;
  name: string;
  provider_name: string;
  base_url: string;
  api_key_env: string;
  model: string;
  timeout_sec: number;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface TestResult {
  ok: boolean;
  message: string;
}

export const providersApi = {
  listAvailable: (): Promise<AvailableProvider[]> => request("/providers"),

  test: (data: {
    provider_name: string;
    base_url?: string;
    api_key_env: string;
    model?: string;
  }): Promise<TestResult> =>
    request("/providers/test", { method: "POST", body: JSON.stringify(data) }),

  listProfiles: (): Promise<ProviderProfile[]> => request("/provider-profiles"),

  createProfile: (data: Omit<ProviderProfile, "id" | "created_at" | "updated_at">): Promise<ProviderProfile> =>
    request("/provider-profiles", { method: "POST", body: JSON.stringify(data) }),

  updateProfile: (id: string, data: Omit<ProviderProfile, "id" | "created_at" | "updated_at">): Promise<ProviderProfile> =>
    request(`/provider-profiles/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  deleteProfile: (id: string): Promise<void> =>
    request(`/provider-profiles/${id}`, { method: "DELETE" }),

  testProfile: (id: string): Promise<TestResult> =>
    request(`/provider-profiles/${id}/test`, { method: "POST" }),
};

// ──────────────────── Tasks ────────────────────

export const tasksApi = {
  listByProject: (projectID: string): Promise<Task[]> =>
    request(`/projects/${projectID}/tasks`),

  get: (id: string): Promise<Task> => request(`/tasks/${id}`),

  create: (data: {
    project_id: string;
    title: string;
    description?: string;
    provider_id?: string;
  }): Promise<Task> =>
    request("/tasks", { method: "POST", body: JSON.stringify(data) }),

  updateStatus: (id: string, status: TaskStatus): Promise<Task> =>
    request(`/tasks/${id}/status`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    }),

  delete: (id: string): Promise<void> =>
    request(`/tasks/${id}`, { method: "DELETE" }),
};
