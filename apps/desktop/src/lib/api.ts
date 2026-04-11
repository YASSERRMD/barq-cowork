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
  list: (): Promise<Project[]> => request("/projects"),

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
  api_key_set: boolean;    // true if a key is stored
  api_key_hint: string;    // masked hint like "••••abcd"
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
    api_key: string;
    model?: string;
  }): Promise<TestResult> =>
    request("/providers/test", { method: "POST", body: JSON.stringify(data) }),

  listProfiles: (): Promise<ProviderProfile[]> => request("/provider-profiles"),

  createProfile: (data: {
    name: string;
    provider_name: string;
    base_url?: string;
    api_key: string;       // direct key, write-only
    api_key_env?: string;  // legacy fallback
    model: string;
    timeout_sec?: number;
    is_default?: boolean;
  }): Promise<ProviderProfile> =>
    request("/provider-profiles", { method: "POST", body: JSON.stringify(data) }),

  updateProfile: (id: string, data: {
    name: string;
    provider_name: string;
    base_url?: string;
    api_key?: string;
    api_key_env?: string;
    model: string;
    timeout_sec?: number;
    is_default?: boolean;
  }): Promise<ProviderProfile> =>
    request(`/provider-profiles/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  deleteProfile: (id: string): Promise<void> =>
    request(`/provider-profiles/${id}`, { method: "DELETE" }),

  testProfile: (id: string): Promise<TestResult> =>
    request(`/provider-profiles/${id}/test`, { method: "POST" }),
};

// ──────────────────── Execution types ────────────────────

export type StepStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "skipped";

export interface PlanStep {
  id: string;
  plan_id: string;
  order: number;
  title: string;
  description: string;
  status: StepStatus;
  tool_name: string;
  tool_input: string;
  tool_output: string;
  started_at?: string;
  completed_at?: string;
}

export interface Plan {
  id: string;
  task_id: string;
  steps: PlanStep[];
  created_at: string;
}

export type ArtifactType = "markdown" | "json" | "file" | "log";

export interface Artifact {
  id: string;
  task_id: string;
  project_id: string;
  name: string;
  type: ArtifactType;
  content_path: string;
  content_inline?: string;
  size: number;
  created_at: string;
}

// ──────────────────── Execution API ────────────────────

export const executionApi = {
  runTask: (
    taskId: string,
    opts?: { workspace_root?: string; require_approval?: boolean }
  ): Promise<void> =>
    request(`/tasks/${taskId}/run`, {
      method: "POST",
      body: JSON.stringify(opts ?? {}),
    }),

  getPlan: (taskId: string): Promise<Plan> =>
    request(`/tasks/${taskId}/plan`),

  listEvents: (taskId: string): Promise<TaskEvent[]> =>
    request(`/tasks/${taskId}/events`),

  listArtifactsByTask: (taskId: string): Promise<Artifact[]> =>
    request(`/tasks/${taskId}/artifacts`),

  listArtifactsByProject: (projectId: string): Promise<Artifact[]> =>
    request(`/projects/${projectId}/artifacts`),

  listRecent: (limit = 100): Promise<Artifact[]> =>
    request(`/artifacts?limit=${limit}`),

  getArtifact: (id: string): Promise<Artifact> => request(`/artifacts/${id}`),
};

// ──────────────────── Sub-agents ────────────────────

export type AgentRole =
  | "researcher" | "writer" | "coder" | "reviewer" | "analyst" | "custom";

export interface SubAgent {
  id: string;
  parent_task_id: string;
  role: AgentRole;
  title: string;
  instructions: string;
  status: TaskStatus;
  plan_id?: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
}

export const agentsApi = {
  list: (taskId: string): Promise<SubAgent[]> =>
    request(`/tasks/${taskId}/agents`),

  spawn: (
    taskId: string,
    data: {
      agents: { role: AgentRole; title: string; instructions: string }[];
      workspace_root?: string;
      max_concurrency?: number;
      timeout_minutes?: number;
    }
  ): Promise<SubAgent[]> =>
    request(`/tasks/${taskId}/agents`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  cancel: (taskId: string, agentId: string): Promise<void> =>
    request(`/tasks/${taskId}/agents/${agentId}`, { method: "DELETE" }),
};

// ──────────────────── Memory — Context Files ────────────────────

export interface ContextFile {
  id: string;
  project_id: string;
  name: string;
  file_path: string;
  content: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface TaskTemplate {
  id: string;
  project_id: string;
  name: string;
  title: string;
  description: string;
  provider_id: string;
  created_at: string;
  updated_at: string;
}

export const contextFilesApi = {
  list: (projectId: string): Promise<ContextFile[]> =>
    request(`/projects/${projectId}/context-files`),

  create: (
    projectId: string,
    data: { name: string; file_path?: string; content?: string; description?: string }
  ): Promise<ContextFile> =>
    request(`/projects/${projectId}/context-files`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    id: string,
    data: { name: string; file_path?: string; content?: string; description?: string }
  ): Promise<ContextFile> =>
    request(`/context-files/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  delete: (id: string): Promise<void> =>
    request(`/context-files/${id}`, { method: "DELETE" }),
};

export const templatesApi = {
  list: (projectId: string): Promise<TaskTemplate[]> =>
    request(`/projects/${projectId}/templates`),

  create: (
    projectId: string,
    data: { name: string; title: string; description?: string; provider_id?: string }
  ): Promise<TaskTemplate> =>
    request(`/projects/${projectId}/templates`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    id: string,
    data: { name: string; title: string; description?: string; provider_id?: string }
  ): Promise<TaskTemplate> =>
    request(`/templates/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  delete: (id: string): Promise<void> =>
    request(`/templates/${id}`, { method: "DELETE" }),
};

// ──────────────────── Events (global) ────────────────────

export const eventsApi = {
  listRecent: (limit = 200): Promise<TaskEvent[]> =>
    request(`/events?limit=${limit}`),
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

// ──────────────────── Diagnostics ────────────────────

export interface SystemInfo {
  generated_at: string;
  version: string;
  go_version: string;
  os: string;
  arch: string;
  num_cpu: number;
  num_goroutine: number;
  mem_alloc_mb: number;
  mem_total_alloc_mb: number;
  build_info?: Record<string, string>;
}

export const diagnosticsApi = {
  getInfo: (): Promise<SystemInfo> => request("/diagnostics/info"),

  /** Returns the URL to download the diagnostic ZIP bundle. */
  bundleUrl: (): string => `${BASE}/diagnostics/bundle`,
};

// ──────────────────── Schedules ────────────────────

export interface Schedule {
  id: string;
  project_id: string;
  name: string;
  description: string;
  cron_expr: string;
  task_title: string;
  task_desc: string;
  provider_id: string;
  enabled: boolean;
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

export const schedulesApi = {
  list: (): Promise<Schedule[]> => request("/schedules"),

  listByProject: (projectId: string): Promise<Schedule[]> =>
    request(`/projects/${projectId}/schedules`),

  get: (id: string): Promise<Schedule> => request(`/schedules/${id}`),

  create: (data: {
    project_id: string;
    name: string;
    description?: string;
    cron_expr: string;
    task_title: string;
    task_desc?: string;
    provider_id?: string;
    enabled?: boolean;
  }): Promise<Schedule> =>
    request("/schedules", { method: "POST", body: JSON.stringify(data) }),

  update: (id: string, data: {
    name: string;
    description?: string;
    cron_expr: string;
    task_title: string;
    task_desc?: string;
    provider_id?: string;
    enabled: boolean;
  }): Promise<Schedule> =>
    request(`/schedules/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  delete: (id: string): Promise<void> =>
    request(`/schedules/${id}`, { method: "DELETE" }),
};
