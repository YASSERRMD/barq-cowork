import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Play, FileText, Activity, Users, ChevronDown, ChevronRight, Clock, CheckCircle, XCircle, Loader, Circle } from "lucide-react";
import {
  tasksApi,
  executionApi,
  toolsApi,
  type Task,
  type PlanStep,
  type TaskEvent,
  type Artifact,
  type StepStatus,
  type TaskStatus,
} from "../lib/api";
import { TopBar } from "../components/TopBar";
import { Breadcrumb } from "../components/Breadcrumb";
import { SkeletonText } from "../components/ui";

const TASK_BADGE: Record<TaskStatus, string> = {
  pending:   "badge-gray",
  planning:  "badge-blue",
  running:   "badge-yellow",
  completed: "badge-green",
  failed:    "badge-red",
  cancelled: "badge-gray",
};

const ACTIVE_STATUSES: TaskStatus[] = ["planning", "running"];

function StepIcon({ status }: { status: StepStatus }) {
  switch (status) {
    case "running":   return <Loader size={14} color="#fbbf24" className="animate-spin" />;
    case "completed": return <CheckCircle size={14} color="#34d399" />;
    case "failed":    return <XCircle size={14} color="#f87171" />;
    case "skipped":   return <Circle size={14} color="#50505f" />;
    default:          return <Circle size={14} color="#3a3a4e" />;
  }
}

type RightPanelTab = "artifacts" | "events" | "agents";

export function TaskRunPage() {
  const { taskId } = useParams<{ taskId: string }>();
  const qc = useQueryClient();
  const [rightTab, setRightTab] = useState<RightPanelTab>("artifacts");
  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set());

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
    enabled: !!taskId,
    refetchInterval: isActive ? 3000 : false,
  });

  const { data: approvals = [] } = useQuery({
    queryKey: ["approvals"],
    queryFn: toolsApi.listApprovals,
    refetchInterval: isActive ? 2000 : false,
    enabled: isActive,
  });

  const runMutation = useMutation({
    mutationFn: () => executionApi.runTask(taskId!, { require_approval: false }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["tasks", taskId] }); },
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, res }: { id: string; res: "approved" | "rejected" }) =>
      toolsApi.resolveApproval(id, res),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["approvals"] }); },
  });

  const toggleStep = (id: string) => {
    setExpandedSteps((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const pendingApprovals = approvals.filter((a) => a.task_id === taskId && a.status === "pending");

  if (taskLoading) {
    return (
      <div style={{ padding: 24 }}>
        <SkeletonText lines={4} />
      </div>
    );
  }

  if (!task) {
    return (
      <div style={{ padding: 24 }}>
        <p style={{ color: "#f87171", fontSize: 13 }}>Task not found.</p>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar title={task.title} />

      {/* Breadcrumb */}
      <div style={{ padding: "8px 20px", borderBottom: "1px solid #2a2a3a" }}>
        <Breadcrumb items={[
          { label: "Projects", to: "/projects" },
          { label: "Tasks", to: `/projects/${task.project_id}/tasks` },
          { label: task.title },
        ]} />
      </div>

      {/* Approval banner */}
      {pendingApprovals.length > 0 && (
        <div style={{ background: "rgba(245,158,11,0.08)", borderBottom: "1px solid rgba(245,158,11,0.2)", padding: "10px 20px" }}>
          {pendingApprovals.map((approval) => (
            <div key={approval.id} style={{ display: "flex", alignItems: "center", gap: 12 }}>
              <span style={{ fontSize: 12, color: "#fbbf24", fontWeight: 600 }}>Approval Required:</span>
              <span style={{ fontSize: 12, color: "#c4c4d0", flex: 1 }}>
                Tool <strong>{approval.tool_name}</strong> — {approval.action}
              </span>
              <button
                className="btn-primary btn-sm"
                onClick={() => approveMutation.mutate({ id: approval.id, res: "approved" })}
              >
                Approve
              </button>
              <button
                className="btn-danger btn-sm"
                onClick={() => approveMutation.mutate({ id: approval.id, res: "rejected" })}
              >
                Reject
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Main layout */}
      <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>
        {/* Left: timeline */}
        <div style={{ flex: 1, overflowY: "auto", padding: "20px" }}>
          {/* Task header */}
          <div className="card" style={{ padding: "16px 20px", marginBottom: 20 }}>
            <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 12 }}>
              <div style={{ minWidth: 0 }}>
                <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: task.description ? 4 : 0 }}>
                  <span className={TASK_BADGE[task.status] ?? "badge-gray"}>{task.status}</span>
                  {isActive && (
                    <span style={{ fontSize: 11, color: "#fbbf24", display: "flex", alignItems: "center", gap: 4 }}>
                      <Loader size={11} className="animate-spin" />
                      Working...
                    </span>
                  )}
                </div>
                {task.description && (
                  <p style={{ fontSize: 13, color: "#7a7a90", marginTop: 4 }}>{task.description}</p>
                )}
                <div style={{ display: "flex", alignItems: "center", gap: 16, marginTop: 8, fontSize: 11, color: "#40404f" }}>
                  {task.started_at && (
                    <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
                      <Clock size={11} />
                      Started {new Date(task.started_at).toLocaleTimeString()}
                    </span>
                  )}
                  {task.completed_at && (
                    <span>Completed {new Date(task.completed_at).toLocaleTimeString()}</span>
                  )}
                </div>
              </div>
              {task.status === "pending" && (
                <button
                  className="btn-primary"
                  style={{ flexShrink: 0 }}
                  disabled={runMutation.isPending}
                  onClick={() => runMutation.mutate()}
                >
                  <Play size={13} />
                  {runMutation.isPending ? "Starting..." : "Run Task"}
                </button>
              )}
            </div>
          </div>

          {/* Timeline */}
          {plan && plan.steps.length > 0 && (
            <div>
              <div style={{ fontSize: 11, fontWeight: 600, letterSpacing: "0.06em", textTransform: "uppercase", color: "#50505f", marginBottom: 12 }}>
                Execution Plan — {plan.steps.length} steps
              </div>
              <div style={{ position: "relative" }}>
                {/* Connector line */}
                <div style={{
                  position: "absolute",
                  left: 11,
                  top: 20,
                  bottom: 20,
                  width: 1,
                  background: "#2a2a3a",
                }} />
                {plan.steps.map((step) => (
                  <StepItem
                    key={step.id}
                    step={step}
                    expanded={expandedSteps.has(step.id)}
                    onToggle={() => toggleStep(step.id)}
                  />
                ))}
              </div>
            </div>
          )}

          {task.status === "pending" && !plan && (
            <div style={{ textAlign: "center", padding: "32px 0", color: "#40404f", fontSize: 13 }}>
              Click "Run Task" to begin execution.
            </div>
          )}

          {isActive && !plan && (
            <div style={{ textAlign: "center", padding: "32px 0", color: "#fbbf24", fontSize: 13 }}>
              <Loader size={20} className="animate-spin" style={{ margin: "0 auto 8px" }} />
              Planning...
            </div>
          )}
        </div>

        {/* Right panel */}
        <div className="right-panel">
          {/* Tabs */}
          <div style={{ display: "flex", borderBottom: "1px solid #2a2a3a" }}>
            {(["artifacts", "events", "agents"] as RightPanelTab[]).map((tab) => (
              <button
                key={tab}
                onClick={() => setRightTab(tab)}
                style={{
                  flex: 1,
                  padding: "10px 4px",
                  fontSize: 11,
                  fontWeight: 600,
                  letterSpacing: "0.04em",
                  textTransform: "uppercase",
                  background: "transparent",
                  border: "none",
                  cursor: "pointer",
                  color: rightTab === tab ? "#a5b4fc" : "#50505f",
                  borderBottom: `2px solid ${rightTab === tab ? "#6366f1" : "transparent"}`,
                  transition: "all 150ms",
                }}
              >
                {tab === "artifacts" ? <FileText size={13} style={{ margin: "0 auto" }} /> :
                 tab === "events" ? <Activity size={13} style={{ margin: "0 auto" }} /> :
                 <Users size={13} style={{ margin: "0 auto" }} />}
                <span style={{ display: "block", marginTop: 2 }}>{tab.charAt(0).toUpperCase() + tab.slice(1)}</span>
              </button>
            ))}
          </div>

          {/* Panel content */}
          <div style={{ padding: "12px" }}>
            {rightTab === "artifacts" && (
              <ArtifactsPanel artifacts={artifacts} />
            )}
            {rightTab === "events" && (
              <EventsPanel events={events} />
            )}
            {rightTab === "agents" && (
              <AgentsPanel taskId={taskId!} />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function StepItem({ step, expanded, onToggle }: { step: PlanStep; expanded: boolean; onToggle: () => void }) {
  const hasOutput = !!(step.tool_output || step.tool_input);

  return (
    <div
      style={{
        display: "flex",
        gap: 12,
        paddingBottom: 12,
        position: "relative",
        zIndex: 1,
      }}
    >
      {/* Status dot */}
      <div style={{ flexShrink: 0, marginTop: 2 }}>
        <StepIcon status={step.status} />
      </div>

      {/* Content */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          onClick={hasOutput ? onToggle : undefined}
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            cursor: hasOutput ? "pointer" : "default",
            padding: "4px 8px",
            borderRadius: 4,
            background: step.status === "running" ? "rgba(245,158,11,0.05)" : "transparent",
            transition: "background 120ms",
          }}
        >
          <span style={{ fontSize: 13, color: step.status === "running" ? "#e2e2e8" : "#c4c4d0", flex: 1, fontWeight: step.status === "running" ? 600 : 400 }}>
            {step.title}
          </span>
          {step.tool_name && (
            <span className="badge-gray" style={{ fontSize: 10, padding: "1px 5px" }}>{step.tool_name}</span>
          )}
          {hasOutput && (
            expanded ? <ChevronDown size={12} color="#50505f" /> : <ChevronRight size={12} color="#50505f" />
          )}
        </div>
        {step.description && (
          <p style={{ fontSize: 12, color: "#50505f", paddingLeft: 8, marginTop: 2 }}>{step.description}</p>
        )}
        {expanded && hasOutput && (
          <div style={{ marginTop: 6, padding: "8px 10px", background: "#16161f", borderRadius: 4, border: "1px solid #2a2a3a" }}>
            {step.tool_input && (
              <div style={{ marginBottom: 6 }}>
                <div style={{ fontSize: 10, color: "#40404f", fontWeight: 600, marginBottom: 3, textTransform: "uppercase" }}>Input</div>
                <pre style={{ fontSize: 11, color: "#7a7a90", fontFamily: "JetBrains Mono, monospace", whiteSpace: "pre-wrap", wordBreak: "break-all", margin: 0 }}>
                  {step.tool_input}
                </pre>
              </div>
            )}
            {step.tool_output && (
              <div>
                <div style={{ fontSize: 10, color: "#40404f", fontWeight: 600, marginBottom: 3, textTransform: "uppercase" }}>Output</div>
                <pre style={{ fontSize: 11, color: "#c4c4d0", fontFamily: "JetBrains Mono, monospace", whiteSpace: "pre-wrap", wordBreak: "break-all", margin: 0 }}>
                  {step.tool_output.length > 500 ? step.tool_output.slice(0, 500) + "..." : step.tool_output}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function ArtifactsPanel({ artifacts }: { artifacts: Artifact[] }) {
  if (artifacts.length === 0) {
    return <p style={{ fontSize: 12, color: "#50505f", textAlign: "center", paddingTop: 16 }}>No artifacts yet</p>;
  }
  return (
    <div style={{ display: "grid", gap: 8 }}>
      {artifacts.map((a) => (
        <div key={a.id} className="card" style={{ padding: "8px 10px" }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: "#c4c4d0", marginBottom: 2 }}>{a.name}</div>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span className="badge-gray">{a.type}</span>
            <span style={{ fontSize: 11, color: "#40404f" }}>{a.size > 0 ? `${(a.size / 1024).toFixed(1)} KB` : "0 B"}</span>
          </div>
        </div>
      ))}
    </div>
  );
}

function EventsPanel({ events }: { events: TaskEvent[] }) {
  if (events.length === 0) {
    return <p style={{ fontSize: 12, color: "#50505f", textAlign: "center", paddingTop: 16 }}>No events yet</p>;
  }
  return (
    <div style={{ display: "grid", gap: 6 }}>
      {[...events].reverse().slice(0, 50).map((e) => (
        <div key={e.id} style={{ fontSize: 11, borderBottom: "1px solid rgba(42,42,58,0.5)", paddingBottom: 6 }}>
          <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 2 }}>
            <span className="badge-blue" style={{ fontSize: 10 }}>{e.type}</span>
            <span style={{ color: "#40404f" }}>{new Date(e.created_at).toLocaleTimeString()}</span>
          </div>
          {e.payload && (
            <pre style={{ fontSize: 10, color: "#50505f", fontFamily: "JetBrains Mono, monospace", whiteSpace: "pre-wrap", wordBreak: "break-all", margin: 0, maxHeight: 60, overflow: "hidden" }}>
              {e.payload.length > 120 ? e.payload.slice(0, 120) + "..." : e.payload}
            </pre>
          )}
        </div>
      ))}
    </div>
  );
}

function AgentsPanel({ taskId }: { taskId: string }) {
  const { data: agents = [] } = useQuery({
    queryKey: ["agents", taskId],
    queryFn: () => import("../lib/api").then(m => m.agentsApi.list(taskId)),
    refetchInterval: 3000,
  });

  if (agents.length === 0) {
    return <p style={{ fontSize: 12, color: "#50505f", textAlign: "center", paddingTop: 16 }}>No sub-agents</p>;
  }
  return (
    <div style={{ display: "grid", gap: 8 }}>
      {agents.map((a) => (
        <div key={a.id} className="card" style={{ padding: "8px 10px" }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: "#c4c4d0", marginBottom: 2 }}>{a.title}</div>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span className="badge-purple">{a.role}</span>
            <span className={a.status === "completed" ? "badge-green" : a.status === "failed" ? "badge-red" : "badge-yellow"}>
              {a.status}
            </span>
          </div>
        </div>
      ))}
    </div>
  );
}
