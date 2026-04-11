import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import { agentsApi, type SubAgent, type AgentRole, type TaskStatus } from "../lib/api";
import clsx from "clsx";

// ─────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────

const ROLE_ICON: Record<AgentRole, string> = {
  researcher: "🔍",
  writer:     "✍️",
  coder:      "💻",
  reviewer:   "👁️",
  analyst:    "📊",
  custom:     "🤖",
};

const ROLE_COLOR: Record<AgentRole, string> = {
  researcher: "text-blue-400",
  writer:     "text-purple-400",
  coder:      "text-green-400",
  reviewer:   "text-orange-400",
  analyst:    "text-yellow-400",
  custom:     "text-gray-400",
};

const STATUS_BADGE: Record<TaskStatus, string> = {
  pending:   "badge-gray",
  planning:  "badge-blue",
  running:   "badge-yellow",
  completed: "badge-green",
  failed:    "badge-red",
  cancelled: "badge-gray",
};

const ALL_ROLES: AgentRole[] = ["researcher", "writer", "coder", "reviewer", "analyst", "custom"];

const ACTIVE: TaskStatus[] = ["planning", "running"];

// ─────────────────────────────────────────────
// SubAgentsPanel (exported)
// ─────────────────────────────────────────────

export function SubAgentsPanel({
  taskId,
  workspaceRoot = "",
}: {
  taskId: string;
  workspaceRoot?: string;
}) {
  const qc = useQueryClient();
  const [showSpawn, setShowSpawn] = useState(false);

  const { data: agents = [], isLoading } = useQuery({
    queryKey: ["agents", taskId],
    queryFn: () => agentsApi.list(taskId),
    refetchInterval: (query) => {
      const list = query.state.data as SubAgent[] | undefined;
      return list?.some((a) => ACTIVE.includes(a.status)) ? 2000 : false;
    },
  });

  const cancelMutation = useMutation({
    mutationFn: (agentId: string) => agentsApi.cancel(taskId, agentId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["agents", taskId] }),
  });

  const spawnMutation = useMutation({
    mutationFn: (data: Parameters<typeof agentsApi.spawn>[1]) =>
      agentsApi.spawn(taskId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["agents", taskId] });
      setShowSpawn(false);
    },
  });

  const activeCount = agents.filter((a) => ACTIVE.includes(a.status)).length;
  const doneCount   = agents.filter((a) => a.status === "completed").length;
  const failCount   = agents.filter((a) => a.status === "failed").length;

  return (
    <section className="space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
            Sub-Agents
          </h2>
          {agents.length > 0 && (
            <div className="flex items-center gap-2 text-xs">
              {activeCount > 0 && (
                <span className="badge-yellow">{activeCount} active</span>
              )}
              {doneCount > 0 && (
                <span className="badge-green">{doneCount} done</span>
              )}
              {failCount > 0 && (
                <span className="badge-red">{failCount} failed</span>
              )}
            </div>
          )}
        </div>
        <button
          className="btn-primary text-xs"
          onClick={() => setShowSpawn((v) => !v)}
        >
          {showSpawn ? "Cancel" : "+ Spawn Agents"}
        </button>
      </div>

      {/* Spawn form */}
      {showSpawn && (
        <SpawnForm
          workspaceRoot={workspaceRoot}
          onSubmit={(data) => spawnMutation.mutate(data)}
          loading={spawnMutation.isPending}
          error={spawnMutation.error?.message}
          onCancel={() => setShowSpawn(false)}
        />
      )}

      {/* List */}
      {isLoading && <p className="text-gray-400 text-sm">Loading…</p>}

      {!isLoading && agents.length === 0 && !showSpawn && (
        <div className="card p-5 text-center text-gray-500 text-sm space-y-1">
          <p className="text-xl">🤖</p>
          <p>No sub-agents yet.</p>
          <p className="text-xs text-gray-600">
            Spawn specialised agents (researcher, coder, writer…) to work in parallel.
          </p>
        </div>
      )}

      {agents.length > 0 && (
        <ul className="space-y-2">
          {agents.map((a) => (
            <AgentCard
              key={a.id}
              agent={a}
              taskId={taskId}
              onCancel={() => cancelMutation.mutate(a.id)}
              cancelling={cancelMutation.isPending && cancelMutation.variables === a.id}
            />
          ))}
        </ul>
      )}
    </section>
  );
}

// ─────────────────────────────────────────────
// AgentCard
// ─────────────────────────────────────────────

function AgentCard({
  agent: a,
  taskId: _taskId,
  onCancel,
  cancelling,
}: {
  agent: SubAgent;
  taskId: string;
  onCancel: () => void;
  cancelling: boolean;
}) {
  const isActive = ACTIVE.includes(a.status);
  const [expanded, setExpanded] = useState(false);

  return (
    <li
      className={clsx(
        "card p-3 space-y-2 border-l-2 transition-colors",
        a.status === "running"   && "border-l-yellow-500",
        a.status === "planning"  && "border-l-blue-500",
        a.status === "completed" && "border-l-green-500",
        a.status === "failed"    && "border-l-red-500",
        a.status === "pending"   && "border-l-gray-700",
      )}
    >
      <div className="flex items-start gap-3">
        {/* Role icon */}
        <span className="text-lg shrink-0 mt-0.5">{ROLE_ICON[a.role] ?? "🤖"}</span>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 justify-between">
            <div className="flex items-center gap-2 min-w-0">
              <span className={clsx("text-xs font-semibold uppercase tracking-wide shrink-0", ROLE_COLOR[a.role])}>
                {a.role}
              </span>
              <p className="text-white text-sm font-medium truncate">{a.title}</p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <span className={clsx("text-xs", STATUS_BADGE[a.status] ?? "badge-gray")}>
                {a.status}
                {isActive && <span className="ml-1 animate-pulse">…</span>}
              </span>
              {(a.status === "pending" || isActive) && (
                <button
                  className="btn-ghost text-xs text-red-400 hover:text-red-300"
                  onClick={onCancel}
                  disabled={cancelling}
                  title="Cancel this agent"
                >
                  <X size={12} strokeWidth={2} />
                </button>
              )}
            </div>
          </div>

          {/* Timing */}
          <div className="flex gap-4 text-xs text-gray-600 mt-0.5">
            {a.started_at && (
              <span>Started {new Date(a.started_at).toLocaleTimeString()}</span>
            )}
            {a.completed_at && (
              <span>Done {new Date(a.completed_at).toLocaleTimeString()}</span>
            )}
          </div>

          {/* Instructions preview toggle */}
          {a.instructions && (
            <button
              className="text-xs text-gray-600 hover:text-gray-400 mt-1"
              onClick={() => setExpanded((v) => !v)}
            >
              {expanded ? "Hide instructions" : "Show instructions"}
            </button>
          )}
          {expanded && a.instructions && (
            <pre className="mt-1 text-xs text-gray-400 bg-gray-950 rounded p-2 max-h-32 overflow-y-auto font-mono whitespace-pre-wrap break-all">
              {a.instructions}
            </pre>
          )}
        </div>
      </div>
    </li>
  );
}

// ─────────────────────────────────────────────
// Spawn form
// ─────────────────────────────────────────────

type AgentDraft = { role: AgentRole; title: string; instructions: string };

function SpawnForm({
  workspaceRoot,
  onSubmit,
  loading,
  error,
  onCancel,
}: {
  workspaceRoot: string;
  onSubmit: (data: {
    agents: AgentDraft[];
    workspace_root: string;
    max_concurrency: number;
    timeout_minutes: number;
  }) => void;
  loading: boolean;
  error?: string;
  onCancel: () => void;
}) {
  const [agents, setAgents] = useState<AgentDraft[]>([
    { role: "researcher", title: "", instructions: "" },
  ]);
  const [maxConc, setMaxConc] = useState(3);
  const [timeoutMin, setTimeoutMin] = useState(5);

  const addAgent = () =>
    setAgents((prev) => [...prev, { role: "custom", title: "", instructions: "" }]);

  const removeAgent = (i: number) =>
    setAgents((prev) => prev.filter((_, idx) => idx !== i));

  const updateAgent = (i: number, field: keyof AgentDraft, value: string) =>
    setAgents((prev) =>
      prev.map((a, idx) => (idx === i ? { ...a, [field]: value } : a))
    );

  return (
    <form
      className="card p-4 space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({
          agents,
          workspace_root: workspaceRoot,
          max_concurrency: maxConc,
          timeout_minutes: timeoutMin,
        });
      }}
    >
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-300">Spawn Sub-Agents</h3>
        <div className="flex items-center gap-4 text-xs text-gray-400">
          <label className="flex items-center gap-1.5">
            Max concurrency
            <input
              type="number"
              min={1}
              max={10}
              value={maxConc}
              onChange={(e) => setMaxConc(Number(e.target.value))}
              className="w-12 bg-gray-800 border border-gray-700 rounded px-1.5 py-0.5 text-white focus:outline-none focus:border-barq-500"
            />
          </label>
          <label className="flex items-center gap-1.5">
            Timeout (min)
            <input
              type="number"
              min={1}
              max={60}
              value={timeoutMin}
              onChange={(e) => setTimeoutMin(Number(e.target.value))}
              className="w-12 bg-gray-800 border border-gray-700 rounded px-1.5 py-0.5 text-white focus:outline-none focus:border-barq-500"
            />
          </label>
        </div>
      </div>

      {/* Agent rows */}
      <div className="space-y-3">
        {agents.map((a, i) => (
          <div key={i} className="flex gap-2 items-start bg-gray-900/50 rounded p-3">
            <div className="flex-1 space-y-2">
              <div className="flex gap-2">
                <select
                  className="bg-gray-800 border border-gray-700 rounded text-xs text-gray-300 px-2 py-1.5 focus:outline-none focus:border-barq-500 w-36"
                  value={a.role}
                  onChange={(e) => updateAgent(i, "role", e.target.value)}
                >
                  {ALL_ROLES.map((r) => (
                    <option key={r} value={r}>
                      {ROLE_ICON[r]} {r}
                    </option>
                  ))}
                </select>
                <input
                  className="input flex-1 text-sm"
                  placeholder="Agent title *"
                  value={a.title}
                  onChange={(e) => updateAgent(i, "title", e.target.value)}
                  required
                />
              </div>
              <textarea
                className="input resize-none text-xs w-full"
                rows={2}
                placeholder="Instructions — what should this agent do?"
                value={a.instructions}
                onChange={(e) => updateAgent(i, "instructions", e.target.value)}
              />
            </div>
            {agents.length > 1 && (
              <button
                type="button"
                className="btn-ghost text-xs text-red-400 hover:text-red-300 mt-1 shrink-0"
                onClick={() => removeAgent(i)}
              >
                ✕
              </button>
            )}
          </div>
        ))}
      </div>

      <div className="flex items-center gap-3">
        <button type="button" className="btn-ghost text-xs" onClick={addAgent}>
          + Add Agent
        </button>
        <div className="flex-1" />
        {error && <p className="text-red-400 text-xs">{error}</p>}
        <button type="button" className="btn-ghost text-sm" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="btn-primary text-sm" disabled={loading}>
          {loading ? "Spawning…" : `Spawn ${agents.length} Agent${agents.length !== 1 ? "s" : ""}`}
        </button>
      </div>
    </form>
  );
}
