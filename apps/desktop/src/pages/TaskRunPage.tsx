import { useParams, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { SubAgentsPanel } from "../components/SubAgentsPanel";
import {
  tasksApi,
  executionApi,
  type Task,
  type Plan,
  type PlanStep,
  type TaskEvent,
  type Artifact,
  type StepStatus,
  type TaskStatus,
} from "../lib/api";
import clsx from "clsx";

// ─────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────

const TASK_BADGE: Record<TaskStatus, string> = {
  pending:   "badge-gray",
  planning:  "badge-blue",
  running:   "badge-yellow",
  completed: "badge-green",
  failed:    "badge-red",
  cancelled: "badge-gray",
};

const STEP_BADGE: Record<StepStatus, string> = {
  pending:   "badge-gray",
  running:   "badge-yellow",
  completed: "badge-green",
  failed:    "badge-red",
  skipped:   "badge-gray",
};

const STEP_ICON: Record<StepStatus, string> = {
  pending:   "○",
  running:   "◌",
  completed: "✓",
  failed:    "✕",
  skipped:   "–",
};

const ACTIVE_STATUSES: TaskStatus[] = ["planning", "running"];

// ─────────────────────────────────────────────
// Page
// ─────────────────────────────────────────────

export function TaskRunPage() {
  const { taskId } = useParams<{ taskId: string }>();
  const qc = useQueryClient();

  const { data: task, isLoading: taskLoading } = useQuery({
    queryKey: ["tasks", taskId],
    queryFn: () => tasksApi.get(taskId!),
    enabled: !!taskId,
    refetchInterval: (query) => {
      const t = query.state.data as Task | undefined;
      return t && ACTIVE_STATUSES.includes(t.status) ? 2000 : false;
    },
  });

  const isActive = task ? ACTIVE_STATUSES.includes(task.status) : false;

  const { data: plan } = useQuery({
    queryKey: ["tasks", taskId, "plan"],
    queryFn: () => executionApi.getPlan(taskId!),
    enabled: !!taskId && task?.status !== "pending",
    refetchInterval: isActive ? 2000 : false,
  });

  const { data: events = [] } = useQuery({
    queryKey: ["tasks", taskId, "events"],
    queryFn: () => executionApi.listEvents(taskId!),
    enabled: !!taskId && task?.status !== "pending",
    refetchInterval: isActive ? 2000 : false,
  });

  const { data: artifacts = [] } = useQuery({
    queryKey: ["tasks", taskId, "artifacts"],
    queryFn: () => executionApi.listArtifactsByTask(taskId!),
    enabled: !!taskId && task?.status === "completed",
  });

  const runMutation = useMutation({
    mutationFn: () => executionApi.runTask(taskId!, { require_approval: false }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tasks", taskId] });
    },
  });

  if (taskLoading) {
    return (
      <div className="p-6">
        <p className="text-gray-400 text-sm">Loading…</p>
      </div>
    );
  }

  if (!task) {
    return (
      <div className="p-6">
        <p className="text-red-400 text-sm">Task not found.</p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6 max-w-3xl">
      {/* Breadcrumb */}
      <nav className="text-xs text-gray-500 flex items-center gap-1">
        <Link to="/" className="hover:text-gray-300">Workspaces</Link>
        <span>/</span>
        <Link to={`/projects/${task.project_id}/tasks`} className="hover:text-gray-300">
          Tasks
        </Link>
        <span>/</span>
        <span className="text-gray-300 truncate max-w-[200px]">{task.title}</span>
      </nav>

      {/* Task header */}
      <div className="card p-5 space-y-3">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <h1 className="text-lg font-semibold text-white truncate">{task.title}</h1>
            {task.description && (
              <p className="text-gray-400 text-sm mt-1">{task.description}</p>
            )}
          </div>
          <div className="flex items-center gap-3 shrink-0">
            <span className={clsx(TASK_BADGE[task.status] ?? "badge-gray")}>
              {task.status}
            </span>
            {task.status === "pending" && (
              <button
                className="btn-primary"
                disabled={runMutation.isPending}
                onClick={() => runMutation.mutate()}
              >
                {runMutation.isPending ? "Starting…" : "▶ Run Task"}
              </button>
            )}
          </div>
        </div>

        {/* Timing */}
        <div className="flex gap-6 text-xs text-gray-500">
          <span>Created {new Date(task.created_at).toLocaleString()}</span>
          {task.started_at && (
            <span>Started {new Date(task.started_at).toLocaleString()}</span>
          )}
          {task.completed_at && (
            <span>Finished {new Date(task.completed_at).toLocaleString()}</span>
          )}
        </div>

        {runMutation.error && (
          <p className="text-red-400 text-xs">{runMutation.error.message}</p>
        )}
      </div>

      {/* Plan / Step timeline */}
      {plan && <PlanTimeline plan={plan} />}

      {/* Sub-agents panel — always shown when task is running or has agents */}
      {task.status !== "pending" && (
        <SubAgentsPanel taskId={task.id} workspaceRoot="" />
      )}

      {/* Artifacts */}
      {artifacts.length > 0 && <ArtifactList artifacts={artifacts} />}

      {/* Event log */}
      {events.length > 0 && <EventLog events={events} />}
    </div>
  );
}

// ─────────────────────────────────────────────
// PlanTimeline
// ─────────────────────────────────────────────

function PlanTimeline({ plan }: { plan: Plan }) {
  return (
    <section className="space-y-2">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
        Execution Plan
        <span className="ml-2 text-gray-500 font-normal normal-case">
          {plan.steps.length} step{plan.steps.length !== 1 ? "s" : ""}
        </span>
      </h2>

      <ol className="space-y-2">
        {plan.steps.map((step) => (
          <StepRow key={step.id} step={step} />
        ))}
      </ol>
    </section>
  );
}

function StepRow({ step }: { step: PlanStep }) {
  const isExpanded = step.status === "failed" || step.tool_output !== "";
  const icon = STEP_ICON[step.status] ?? "○";

  return (
    <li
      className={clsx(
        "card p-3 space-y-2 border-l-2 transition-colors",
        step.status === "running"   && "border-l-yellow-500",
        step.status === "completed" && "border-l-green-500",
        step.status === "failed"    && "border-l-red-500",
        step.status === "skipped"   && "border-l-gray-600",
        step.status === "pending"   && "border-l-gray-700",
      )}
    >
      <div className="flex items-start gap-3">
        {/* Ordinal + icon */}
        <div className="w-6 text-center shrink-0">
          <span
            className={clsx(
              "font-mono text-sm",
              step.status === "running"   && "text-yellow-400 animate-pulse",
              step.status === "completed" && "text-green-400",
              step.status === "failed"    && "text-red-400",
              step.status === "skipped"   && "text-gray-600",
              step.status === "pending"   && "text-gray-600",
            )}
          >
            {icon}
          </span>
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 justify-between">
            <p className="text-white text-sm font-medium truncate">
              {step.order}. {step.title}
            </p>
            <div className="flex items-center gap-2 shrink-0">
              {step.tool_name && (
                <code className="text-xs text-barq-400 bg-gray-900 px-1.5 py-0.5 rounded font-mono">
                  {step.tool_name}
                </code>
              )}
              <span className={clsx("text-xs", STEP_BADGE[step.status] ?? "badge-gray")}>
                {step.status}
              </span>
            </div>
          </div>

          {step.description && (
            <p className="text-gray-400 text-xs mt-0.5">{step.description}</p>
          )}

          {/* Timing */}
          {(step.started_at || step.completed_at) && (
            <div className="flex gap-4 text-xs text-gray-600 mt-1">
              {step.started_at && (
                <span>Started {new Date(step.started_at).toLocaleTimeString()}</span>
              )}
              {step.completed_at && (
                <span>Done {new Date(step.completed_at).toLocaleTimeString()}</span>
              )}
            </div>
          )}

          {/* Tool output */}
          {isExpanded && step.tool_output && (
            <OutputBlock raw={step.tool_output} />
          )}
        </div>
      </div>
    </li>
  );
}

function OutputBlock({ raw }: { raw: string }) {
  let pretty: string;
  try {
    pretty = JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    pretty = raw;
  }
  return (
    <pre className="mt-2 text-xs text-gray-400 bg-gray-900 rounded p-2 overflow-x-auto max-h-40 font-mono whitespace-pre-wrap break-all">
      {pretty}
    </pre>
  );
}

// ─────────────────────────────────────────────
// ArtifactList
// ─────────────────────────────────────────────

const ARTIFACT_ICON: Record<string, string> = {
  markdown: "📄",
  json:     "📋",
  file:     "📁",
  log:      "📜",
};

function ArtifactList({ artifacts }: { artifacts: Artifact[] }) {
  return (
    <section className="space-y-2">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
        Artifacts
        <span className="ml-2 text-gray-500 font-normal normal-case">
          {artifacts.length}
        </span>
      </h2>

      <ul className="space-y-2">
        {artifacts.map((a) => (
          <li key={a.id} className="card p-3 flex items-center gap-3">
            <span className="text-lg">{ARTIFACT_ICON[a.type] ?? "📦"}</span>
            <div className="flex-1 min-w-0">
              <p className="text-white text-sm font-medium truncate">{a.name}</p>
              <p className="text-gray-500 text-xs">
                {a.type} · {formatBytes(a.size)}
              </p>
            </div>
            <span className="text-gray-600 text-xs shrink-0">
              {new Date(a.created_at).toLocaleTimeString()}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

// ─────────────────────────────────────────────
// EventLog
// ─────────────────────────────────────────────

const EVENT_COLOR: Record<string, string> = {
  "task.created":    "text-gray-400",
  "task.started":    "text-blue-400",
  "task.completed":  "text-green-400",
  "task.failed":     "text-red-400",
  "step.started":    "text-yellow-300",
  "step.completed":  "text-green-300",
  "tool.called":     "text-barq-300",
  "tool.result":     "text-barq-200",
  "approval.needed": "text-orange-400",
  "artifact.ready":  "text-purple-400",
  "log.line":        "text-gray-400",
};

function EventLog({ events }: { events: TaskEvent[] }) {
  return (
    <section className="space-y-2">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
        Event Log
        <span className="ml-2 text-gray-500 font-normal normal-case">
          {events.length} event{events.length !== 1 ? "s" : ""}
        </span>
      </h2>

      <div className="card bg-gray-950 rounded-lg overflow-hidden">
        <ul className="divide-y divide-gray-800/50 max-h-72 overflow-y-auto">
          {events.map((ev) => (
            <EventRow key={ev.id} event={ev} />
          ))}
        </ul>
      </div>
    </section>
  );
}

function EventRow({ event: ev }: { event: TaskEvent }) {
  let payload: string;
  try {
    payload = JSON.stringify(JSON.parse(ev.payload), null, 0);
  } catch {
    payload = ev.payload;
  }

  return (
    <li className="px-3 py-2 flex items-start gap-3 text-xs hover:bg-gray-900/40 transition-colors">
      <span className="text-gray-600 font-mono shrink-0 w-20">
        {new Date(ev.created_at).toLocaleTimeString()}
      </span>
      <span className={clsx("font-mono shrink-0 w-32 truncate", EVENT_COLOR[ev.type] ?? "text-gray-400")}>
        {ev.type}
      </span>
      <span className="text-gray-500 font-mono break-all">{payload}</span>
    </li>
  );
}
