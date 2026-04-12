import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Play, FileText, Activity, Users, ChevronDown, ChevronRight,
  Clock, CheckCircle, XCircle, Loader, Circle, ArrowLeft,
  AlertTriangle, Download, Copy, Zap, Terminal, Maximize2, X,
  PanelRight, MessageSquare, Send,
} from "lucide-react";
import ReactMarkdown from "react-markdown";
import {
  tasksApi, executionApi, toolsApi, agentsApi,
  type Task, type PlanStep, type TaskEvent, type Artifact,
  type StepStatus, type TaskStatus, type SubAgent, type PendingInput,
} from "../lib/api";

// ── Helpers ───────────────────────────────────────────────────────

const TASK_BADGE: Record<TaskStatus, string> = {
  pending:   "badge-gray",
  planning:  "badge-accent",
  running:   "badge-yellow",
  completed: "badge-green",
  failed:    "badge-red",
  cancelled: "badge-gray",
};

const ACTIVE: TaskStatus[] = ["planning", "running"];

function formatDuration(start: string | null | undefined, end: string | null | undefined): string {
  if (!start) return "—";
  const s = new Date(start).getTime();
  const e = end ? new Date(end).getTime() : Date.now();
  const sec = Math.floor((e - s) / 1000);
  if (sec < 60) return `${sec}s`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m ${sec % 60}s`;
  return `${Math.floor(sec / 3600)}h ${Math.floor((sec % 3600) / 60)}m`;
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function formatBytes(b: number) {
  if (b <= 0) return "—";
  if (b < 1024) return `${b} B`;
  if (b < 1024 ** 2) return `${(b / 1024).toFixed(1)} KB`;
  return `${(b / 1024 ** 2).toFixed(1)} MB`;
}

// ── Step icon ─────────────────────────────────────────────────────

function StepIcon({ status }: { status: StepStatus }) {
  switch (status) {
    case "running":   return <Loader size={14} color="var(--yellow)" className="animate-spin" />;
    case "completed": return <CheckCircle size={14} color="var(--green)" />;
    case "failed":    return <XCircle size={14} color="var(--red)" />;
    case "skipped":   return <Circle size={14} color="var(--text-faint)" />;
    default:          return <Circle size={14} color="var(--surface-4)" style={{ opacity: 0.5 }} />;
  }
}

// ── Step timeline item ────────────────────────────────────────────

function StepItem({
  step, expanded, onToggle, isLast,
}: {
  step: PlanStep; expanded: boolean; onToggle: () => void; isLast: boolean;
}) {
  const hasDetail = !!(step.tool_output || step.tool_input);
  const isRunning = step.status === "running";
  const isCompleted = step.status === "completed";
  const isFailed = step.status === "failed";

  return (
    <div style={{ display: "flex", gap: 0, position: "relative" }}>
      {/* Connector + dot column */}
      <div style={{ display: "flex", flexDirection: "column", alignItems: "center", width: 28, flexShrink: 0 }}>
        <div style={{
          width: 28, height: 28, borderRadius: "50%", flexShrink: 0,
          background: isRunning ? "rgba(251,191,36,0.12)" : isCompleted ? "rgba(52,211,153,0.10)" : isFailed ? "rgba(248,113,113,0.10)" : "var(--surface-3)",
          border: `1px solid ${isRunning ? "rgba(251,191,36,0.3)" : isCompleted ? "rgba(52,211,153,0.25)" : isFailed ? "rgba(248,113,113,0.25)" : "var(--border)"}`,
          display: "flex", alignItems: "center", justifyContent: "center",
          transition: "all 200ms",
        }}>
          <StepIcon status={step.status} />
        </div>
        {!isLast && (
          <div style={{ width: 1, flex: 1, minHeight: 12, background: isCompleted ? "rgba(52,211,153,0.2)" : "var(--border)", marginTop: 2 }} />
        )}
      </div>

      {/* Content */}
      <div style={{ flex: 1, minWidth: 0, paddingLeft: 12, paddingBottom: isLast ? 0 : 16 }}>
        <div
          onClick={hasDetail ? onToggle : undefined}
          style={{
            display: "flex", alignItems: "flex-start", gap: 8,
            cursor: hasDetail ? "pointer" : "default",
            padding: "4px 8px", borderRadius: 6, marginLeft: -8,
            background: isRunning ? "rgba(251,191,36,0.04)" : "transparent",
            transition: "background 120ms",
          }}
        >
          <div style={{ flex: 1, minWidth: 0, paddingTop: 1 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
              <span style={{
                fontSize: 13, fontWeight: isRunning ? 600 : 500,
                color: isRunning ? "var(--text-primary)" : "var(--text-secondary)",
                letterSpacing: "-0.005em",
              }}>
                {step.title}
              </span>
              {step.tool_name && (
                <span style={{
                  fontSize: 10.5, color: "var(--text-faint)",
                  background: "var(--surface-3)", border: "1px solid var(--border)",
                  borderRadius: 4, padding: "1px 6px", fontFamily: "monospace",
                }}>
                  {step.tool_name}
                </span>
              )}
              {step.started_at && (
                <span style={{ fontSize: 11, color: "var(--text-faint)", display: "flex", alignItems: "center", gap: 3 }}>
                  <Clock size={10} />
                  {formatDuration(step.started_at, step.completed_at)}
                </span>
              )}
            </div>
            {step.description && !expanded && (
              <p style={{ fontSize: 12, color: "var(--text-muted)", margin: "2px 0 0", lineHeight: 1.4 }}>
                {step.description}
              </p>
            )}
          </div>
          {hasDetail && (
            <div style={{ flexShrink: 0, marginTop: 4 }}>
              {expanded
                ? <ChevronDown size={13} color="var(--text-faint)" />
                : <ChevronRight size={13} color="var(--text-faint)" />}
            </div>
          )}
        </div>

        {/* Expanded detail */}
        {expanded && hasDetail && (
          <div style={{
            marginTop: 8, marginLeft: 0,
            background: "var(--surface-1)", border: "1px solid var(--border)",
            borderRadius: 8, overflow: "hidden",
          }}>
            {step.tool_input && (
              <div style={{ borderBottom: step.tool_output ? "1px solid var(--border)" : "none" }}>
                <div style={{
                  display: "flex", alignItems: "center", gap: 6,
                  padding: "6px 12px", borderBottom: "1px solid var(--border)",
                  background: "var(--surface-2)",
                }}>
                  <Terminal size={11} color="var(--text-faint)" />
                  <span style={{ fontSize: 10, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase" }}>Input</span>
                </div>
                <pre style={{
                  fontSize: 11.5, color: "var(--text-secondary)", margin: 0,
                  padding: "10px 12px", fontFamily: "JetBrains Mono, ui-monospace, monospace",
                  whiteSpace: "pre-wrap", wordBreak: "break-all", lineHeight: 1.6,
                  maxHeight: 120, overflowY: "auto",
                }}>
                  {step.tool_input}
                </pre>
              </div>
            )}
            {step.tool_output && (
              <div>
                <div style={{
                  display: "flex", alignItems: "center", gap: 6,
                  padding: "6px 12px", borderBottom: "1px solid var(--border)",
                  background: "var(--surface-2)",
                }}>
                  <Zap size={11} color="var(--green)" />
                  <span style={{ fontSize: 10, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase" }}>Output</span>
                </div>
                <pre style={{
                  fontSize: 11.5, color: "var(--text-primary)", margin: 0,
                  padding: "10px 12px", fontFamily: "JetBrains Mono, ui-monospace, monospace",
                  whiteSpace: "pre-wrap", wordBreak: "break-all", lineHeight: 1.6,
                  maxHeight: 180, overflowY: "auto",
                }}>
                  {step.tool_output.length > 800 ? step.tool_output.slice(0, 800) + "\n…" : step.tool_output}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ── Right panel tabs ──────────────────────────────────────────────

type RightTab = "artifacts" | "events" | "agents";

function ArtifactsPanel({
  artifacts,
  onPreview,
  previewId,
}: {
  artifacts: Artifact[];
  onPreview?: (a: Artifact) => void;
  previewId?: string;
}) {
  if (!artifacts.length) return (
    <div className="empty-state" style={{ padding: "32px 16px" }}>
      <div className="empty-state-icon"><FileText size={16} color="var(--text-faint)" /></div>
      <p style={{ fontSize: 12, color: "var(--text-muted)", margin: 0 }}>No artifacts yet</p>
    </div>
  );
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      {artifacts.map((a) => (
        <div key={a.id} style={{
          padding: "10px 12px", background: "var(--surface-2)",
          border: `1px solid ${a.id === previewId ? "var(--accent)" : "var(--border)"}`,
          borderRadius: 8,
        }}>
          <div style={{ display: "flex", alignItems: "flex-start", gap: 8 }}>
            <FileText size={13} color="var(--accent)" style={{ flexShrink: 0, marginTop: 2 }} />
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{
                fontSize: 12.5, fontWeight: 500, color: "var(--text-primary)",
                overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                letterSpacing: "-0.005em",
              }}>
                {a.name.split("/").pop() || a.name}
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 4 }}>
                <span className="badge-gray" style={{ fontSize: 10 }}>{a.type}</span>
                <span style={{ fontSize: 11, color: "var(--text-faint)" }}>{formatBytes(a.size)}</span>
              </div>
            </div>
            <div style={{ display: "flex", gap: 4, flexShrink: 0 }}>
              {onPreview && (
                <button
                  className="btn-ghost btn-xs"
                  title={a.id === previewId ? "Close preview" : "Preview"}
                  onClick={() => onPreview(a)}
                  style={{ color: a.id === previewId ? "var(--accent)" : undefined }}
                >
                  <PanelRight size={11} />
                </button>
              )}
              <button
                className="btn-ghost btn-xs"
                title="Copy path"
                onClick={() => navigator.clipboard.writeText(a.content_path || a.name)}
              >
                <Copy size={11} />
              </button>
              {a.content_path && (
                <a
                  href={`http://localhost:7331/api/v1/artifacts/${a.id}/download`}
                  download
                  className="btn-ghost btn-xs"
                  title="Download file"
                  style={{ display: "flex", alignItems: "center" }}
                >
                  <Download size={11} />
                </a>
              )}
              {a.type === "html" && (
                <a
                  href={`http://localhost:7331/api/v1/artifacts/${a.id}/download`}
                  target="_blank"
                  rel="noreferrer"
                  className="btn-ghost btn-xs"
                  title="Open in browser"
                  style={{ display: "flex", alignItems: "center", fontSize: 10, gap: 3 }}
                >
                  <Maximize2 size={10} />
                </a>
              )}
            </div>
          </div>
          {a.content_inline && (
            <pre style={{
              fontSize: 11, color: "var(--text-muted)", margin: "8px 0 0",
              fontFamily: "JetBrains Mono, monospace", whiteSpace: "pre-wrap",
              wordBreak: "break-all", maxHeight: 60, overflow: "hidden",
              lineHeight: 1.5,
            }}>
              {a.content_inline.slice(0, 140)}{a.content_inline.length > 140 ? "…" : ""}
            </pre>
          )}
        </div>
      ))}
    </div>
  );
}

function EventsPanel({ events }: { events: TaskEvent[] }) {
  if (!events.length) return (
    <div className="empty-state" style={{ padding: "32px 16px" }}>
      <div className="empty-state-icon"><Activity size={16} color="var(--text-faint)" /></div>
      <p style={{ fontSize: 12, color: "var(--text-muted)", margin: 0 }}>No events yet</p>
    </div>
  );
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {[...events].reverse().slice(0, 60).map((e) => (
        <div key={e.id} style={{
          padding: "7px 10px", background: "var(--surface-2)",
          border: "1px solid var(--border)", borderRadius: 7,
        }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 3 }}>
            <span className="badge-accent" style={{ fontSize: 10 }}>{e.type}</span>
            <span style={{ fontSize: 10, color: "var(--text-faint)" }}>{formatTime(e.created_at)}</span>
          </div>
          {e.payload && (
            <pre style={{
              fontSize: 10.5, color: "var(--text-muted)", margin: 0,
              fontFamily: "JetBrains Mono, monospace",
              whiteSpace: "pre-wrap", wordBreak: "break-all",
              maxHeight: 48, overflow: "hidden", lineHeight: 1.5,
            }}>
              {e.payload.length > 100 ? e.payload.slice(0, 100) + "…" : e.payload}
            </pre>
          )}
        </div>
      ))}
    </div>
  );
}

function AgentsPanel({ agents }: { agents: SubAgent[] }) {
  if (!agents.length) return (
    <div className="empty-state" style={{ padding: "32px 16px" }}>
      <div className="empty-state-icon"><Users size={16} color="var(--text-faint)" /></div>
      <p style={{ fontSize: 12, color: "var(--text-muted)", margin: 0 }}>No sub-agents spawned</p>
    </div>
  );
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      {agents.map((a) => (
        <div key={a.id} style={{
          padding: "10px 12px", background: "var(--surface-2)",
          border: "1px solid var(--border)", borderRadius: 8,
        }}>
          <div style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-primary)", marginBottom: 5, letterSpacing: "-0.005em" }}>
            {a.title}
          </div>
          <div style={{ display: "flex", gap: 5 }}>
            <span className="badge-purple" style={{ fontSize: 10 }}>{a.role}</span>
            <span className={
              a.status === "completed" ? "badge-green" :
              a.status === "failed" ? "badge-red" :
              a.status === "running" ? "badge-yellow" : "badge-gray"
            } style={{ fontSize: 10 }}>{a.status}</span>
          </div>
        </div>
      ))}
    </div>
  );
}

// ── JSON inline preview ───────────────────────────────────────────

function JsonPreview({ downloadUrl }: { downloadUrl: string }) {
  const [content, setContent] = useState<string | null>(null);
  const [error, setError] = useState(false);
  useEffect(() => {
    fetch(downloadUrl)
      .then(r => r.text())
      .then(t => {
        try { setContent(JSON.stringify(JSON.parse(t), null, 2)); }
        catch { setContent(t); }
      })
      .catch(() => setError(true));
  }, [downloadUrl]);
  if (error) return <p style={{ color: "var(--red)", fontSize: 12, padding: 16 }}>Failed to load JSON.</p>;
  if (!content) return <div style={{ padding: 16, color: "var(--text-faint)", fontSize: 12 }}><Loader size={12} className="animate-spin" style={{ marginRight: 6 }} />Loading…</div>;
  return (
    <pre style={{
      margin: 0, padding: "12px 16px",
      fontSize: 11, color: "var(--text-secondary)",
      fontFamily: "JetBrains Mono, monospace", whiteSpace: "pre-wrap",
      wordBreak: "break-all", lineHeight: 1.6, overflowY: "auto",
    }}>
      {content}
    </pre>
  );
}

// ── Content preview panel ────────────────────────────────────────

function ContentPreviewPanel({
  artifact,
  onClose,
}: {
  artifact: Artifact;
  onClose: () => void;
}) {
  const downloadUrl = `http://localhost:7331/api/v1/artifacts/${artifact.id}/download`;
  const [markdownContent, setMarkdownContent] = useState<string | null>(null);
  const [loadError, setLoadError] = useState(false);

  // For markdown with inline content or remote fetch
  useEffect(() => {
    setMarkdownContent(null);
    setLoadError(false);
    if (artifact.type === "markdown") {
      if (artifact.content_inline) {
        setMarkdownContent(artifact.content_inline);
      } else if (artifact.content_path) {
        fetch(downloadUrl)
          .then((r) => r.text())
          .then(setMarkdownContent)
          .catch(() => setLoadError(true));
      }
    }
  }, [artifact.id, artifact.type, artifact.content_inline, artifact.content_path, downloadUrl]);

  const fileName = artifact.name.split("/").pop() || artifact.name;

  return (
    <div style={{
      display: "flex", flexDirection: "column", height: "100%",
      background: "var(--surface-1)", borderLeft: "1px solid var(--border)",
    }}>
      {/* Header */}
      <div style={{
        display: "flex", alignItems: "center", gap: 8,
        padding: "8px 12px", borderBottom: "1px solid var(--border)",
        flexShrink: 0, background: "var(--surface-2)",
      }}>
        <FileText size={13} color="var(--accent)" />
        <span style={{
          flex: 1, fontSize: 12.5, fontWeight: 500, color: "var(--text-primary)",
          overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
        }}>
          {fileName}
        </span>
        <span className="badge-gray" style={{ fontSize: 10, flexShrink: 0 }}>{artifact.type}</span>
        <a
          href={downloadUrl}
          download
          className="btn-ghost btn-xs"
          title="Download"
          style={{ display: "flex", alignItems: "center", flexShrink: 0 }}
        >
          <Download size={12} />
        </a>
        <button
          className="btn-ghost btn-xs"
          onClick={onClose}
          title="Close preview"
          style={{ flexShrink: 0 }}
        >
          <X size={12} />
        </button>
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflow: "auto" }}>
        {artifact.type === "markdown" && (
          <div style={{
            padding: "16px 20px",
            fontSize: 13, lineHeight: 1.7,
            color: "var(--text-primary)",
          }}>
            {markdownContent != null ? (
              <div className="markdown-preview" style={{
                maxWidth: "100%",
                wordBreak: "break-word",
              }}>
                <ReactMarkdown>{markdownContent}</ReactMarkdown>
              </div>
            ) : loadError ? (
              <p style={{ color: "var(--red)", fontSize: 12 }}>Failed to load content.</p>
            ) : (
              <div style={{ display: "flex", alignItems: "center", gap: 8, color: "var(--text-faint)", fontSize: 12 }}>
                <Loader size={12} className="animate-spin" /> Loading…
              </div>
            )}
          </div>
        )}

        {artifact.type === "html" && (
          <iframe
            src={downloadUrl}
            style={{ width: "100%", height: "100%", border: "none", background: "#fff" }}
            title={fileName}
            sandbox="allow-scripts allow-same-origin"
          />
        )}

        {artifact.type === "json" && (
          <JsonPreview downloadUrl={downloadUrl} />
        )}

        {(artifact.type !== "markdown" && artifact.type !== "html" && artifact.type !== "json") && (
          <div style={{
            display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
            height: "100%", gap: 16, padding: 32,
          }}>
            {/* File type icon */}
            <div style={{
              width: 64, height: 64, borderRadius: 16,
              background: artifact.type === "file" && fileName.endsWith(".pptx")
                ? "rgba(249,115,22,0.12)"
                : artifact.type === "file" && fileName.endsWith(".docx")
                ? "rgba(59,130,246,0.12)"
                : "var(--accent-dim)",
              display: "flex", alignItems: "center", justifyContent: "center",
              fontSize: 28,
            }}>
              {fileName.endsWith(".pptx") ? "📊"
                : fileName.endsWith(".docx") ? "📄"
                : fileName.endsWith(".pdf")  ? "📕"
                : fileName.endsWith(".xlsx") ? "📈"
                : <FileText size={28} color="var(--accent)" />}
            </div>
            <div style={{ textAlign: "center" }}>
              <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text-primary)", marginBottom: 4, wordBreak: "break-all" }}>
                {fileName}
              </div>
              <div style={{ fontSize: 12, color: "var(--text-muted)", marginBottom: 16 }}>
                {artifact.type.toUpperCase()} · {formatBytes(artifact.size)}
              </div>
              {artifact.content_path && (
                <div style={{
                  fontSize: 10.5, color: "var(--text-faint)", fontFamily: "monospace",
                  marginBottom: 16, wordBreak: "break-all",
                  background: "var(--surface-2)", padding: "4px 8px", borderRadius: 5,
                }}>
                  {artifact.content_path}
                </div>
              )}
              <a
                href={downloadUrl}
                download
                className="btn-primary"
                style={{ display: "inline-flex", alignItems: "center", gap: 6, fontSize: 12, textDecoration: "none" }}
              >
                <Download size={13} />
                Download {fileName.split(".").pop()?.toUpperCase()}
              </a>
            </div>
            <button
              className="btn-ghost btn-sm"
              onClick={() => navigator.clipboard.writeText(artifact.content_path || artifact.name)}
              style={{ fontSize: 11 }}
            >
              <Copy size={11} /> Copy path
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

// ── Main page ─────────────────────────────────────────────────────

export function TaskRunPage() {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [rightTab, setRightTab] = useState<RightTab>("artifacts");
  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set());
  const [previewArtifact, setPreviewArtifact] = useState<Artifact | null>(null);
  const [inputAnswers, setInputAnswers] = useState<Record<string, string>>({});
  const inputRefs = useRef<Record<string, HTMLInputElement | null>>({});

  const { data: task, isLoading: taskLoading } = useQuery({
    queryKey: ["tasks", taskId],
    queryFn: () => tasksApi.get(taskId!),
    enabled: !!taskId,
    refetchInterval: (q) => {
      const t = q.state.data as Task | undefined;
      return t && ACTIVE.includes(t.status) ? 1500 : false;
    },
  });

  const isActive = task ? ACTIVE.includes(task.status) : false;

  const { data: plan } = useQuery({
    queryKey: ["tasks", taskId, "plan"],
    queryFn: () => executionApi.getPlan(taskId!),
    enabled: !!taskId && task?.status !== "pending",
    refetchInterval: isActive ? 1500 : false,
  });

  const { data: events = [] } = useQuery({
    queryKey: ["tasks", taskId, "events"],
    queryFn: () => executionApi.listEvents(taskId!),
    enabled: !!taskId && task?.status !== "pending",
    refetchInterval: isActive ? 1500 : false,
  });

  const { data: artifacts = [] } = useQuery({
    queryKey: ["tasks", taskId, "artifacts"],
    queryFn: () => executionApi.listArtifactsByTask(taskId!),
    enabled: !!taskId,
    refetchInterval: isActive ? 2000 : false,
  });

  const { data: agents = [] } = useQuery({
    queryKey: ["agents", taskId],
    queryFn: () => agentsApi.list(taskId!),
    enabled: !!taskId,
    refetchInterval: isActive ? 2000 : false,
  });

  const { data: approvals = [] } = useQuery({
    queryKey: ["approvals"],
    queryFn: toolsApi.listApprovals,
    refetchInterval: isActive ? 1500 : 10000,
  });

  const { data: pendingInputs = [] } = useQuery<PendingInput[]>({
    queryKey: ["tasks", taskId, "pending-inputs"],
    queryFn: () => executionApi.listPendingInputs(taskId!),
    enabled: !!taskId && isActive,
    refetchInterval: isActive ? 1000 : false,
  });

  const runMutation = useMutation({
    mutationFn: () => executionApi.runTask(taskId!, { require_approval: false }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tasks", taskId] }),
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, res }: { id: string; res: "approved" | "rejected" }) =>
      toolsApi.resolveApproval(id, res),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["approvals"] }),
  });

  const respondMutation = useMutation({
    mutationFn: ({ inputId, answer }: { inputId: string; answer: string }) =>
      executionApi.respondToInput(taskId!, inputId, answer),
    onSuccess: (_data, vars) => {
      setInputAnswers(prev => { const n = { ...prev }; delete n[vars.inputId]; return n; });
      qc.invalidateQueries({ queryKey: ["tasks", taskId, "pending-inputs"] });
    },
  });

  const toggleStep = (id: string) =>
    setExpandedSteps((p) => { const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n; });

  // Auto-preview the first artifact that can be previewed inline
  useEffect(() => {
    if (!previewArtifact && artifacts.length > 0) {
      const previewable = artifacts.find(a => a.type === "markdown" || a.type === "html");
      if (previewable) setPreviewArtifact(previewable);
    }
  }, [artifacts.length]); // eslint-disable-line react-hooks/exhaustive-deps

  const pendingApprovals = approvals.filter(
    (a) => a.task_id === taskId && a.status === "pending"
  );

  const completedSteps = plan?.steps.filter((s) => s.status === "completed").length ?? 0;
  const totalSteps = plan?.steps.length ?? 0;
  const progress = totalSteps > 0 ? (completedSteps / totalSteps) * 100 : 0;

  if (taskLoading) return (
    <div style={{ padding: 24 }}>
      {[1,2,3].map(i => (
        <div key={i} className="skeleton" style={{ height: 20, marginBottom: 10, borderRadius: 6 }} />
      ))}
    </div>
  );

  if (!task) return (
    <div style={{ padding: 24 }}>
      <p style={{ color: "var(--red)", fontSize: 13 }}>Run not found.</p>
    </div>
  );

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>

      {/* ── Top bar ── */}
      <div style={{
        height: "var(--topbar-h)", borderBottom: "1px solid var(--border)",
        display: "flex", alignItems: "center", padding: "0 16px", gap: 10,
        flexShrink: 0, background: "var(--surface-1)",
      }}>
        <button
          className="btn-ghost btn-sm"
          style={{ padding: "4px 6px" }}
          onClick={() => navigate(task.project_id ? `/projects/${task.project_id}/tasks` : "/runs")}
        >
          <ArrowLeft size={14} />
        </button>
        <div style={{ width: 1, height: 18, background: "var(--border)" }} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span style={{
              fontSize: 13.5, fontWeight: 600, color: "var(--text-primary)",
              overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
              letterSpacing: "-0.015em",
            }}>
              {task.title}
            </span>
            <span className={TASK_BADGE[task.status] ?? "badge-gray"}>{task.status}</span>
            {isActive && (
              <span style={{ fontSize: 11, color: "var(--yellow)", display: "flex", alignItems: "center", gap: 4 }}>
                <Loader size={11} className="animate-spin" /> Working…
              </span>
            )}
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 12, flexShrink: 0 }}>
          {totalSteps > 0 && (
            <span style={{ fontSize: 11, color: "var(--text-muted)" }}>
              {completedSteps}/{totalSteps} steps
            </span>
          )}
          {task.started_at && (
            <span style={{ fontSize: 11, color: "var(--text-faint)", display: "flex", alignItems: "center", gap: 4 }}>
              <Clock size={11} />
              {formatDuration(task.started_at, task.completed_at)}
            </span>
          )}
          {task.status === "pending" && (
            <button
              className="btn-primary btn-sm"
              disabled={runMutation.isPending}
              onClick={() => runMutation.mutate()}
            >
              {runMutation.isPending ? <><Loader size={12} className="animate-spin" />Starting…</> : <><Play size={12} />Run</>}
            </button>
          )}
        </div>
      </div>

      {/* ── Progress bar ── */}
      {totalSteps > 0 && (
        <div style={{ height: 2, background: "var(--surface-3)", flexShrink: 0 }}>
          <div style={{
            height: "100%",
            width: `${progress}%`,
            background: task.status === "completed"
              ? "var(--green)"
              : task.status === "failed"
              ? "var(--red)"
              : "var(--accent)",
            transition: "width 400ms ease",
          }} />
        </div>
      )}

      {/* ── Approval banner ── */}
      {pendingApprovals.length > 0 && (
        <div style={{
          background: "rgba(251,191,36,0.06)", borderBottom: "1px solid rgba(251,191,36,0.18)",
          padding: "10px 20px", flexShrink: 0, display: "flex", flexDirection: "column", gap: 8,
        }}>
          {pendingApprovals.map((ap) => (
            <div key={ap.id} style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <AlertTriangle size={14} color="var(--yellow)" style={{ flexShrink: 0 }} />
              <span style={{ fontSize: 12, fontWeight: 600, color: "var(--yellow)" }}>Approval needed:</span>
              <span style={{ fontSize: 12, color: "var(--text-secondary)", flex: 1 }}>
                <code style={{ fontFamily: "monospace", background: "var(--surface-3)", padding: "1px 5px", borderRadius: 3, fontSize: 11 }}>{ap.tool_name}</code>
                {" "}— {ap.action}
              </span>
              <button className="btn-primary btn-sm" onClick={() => approveMutation.mutate({ id: ap.id, res: "approved" })}>Approve</button>
              <button className="btn-danger btn-sm" onClick={() => approveMutation.mutate({ id: ap.id, res: "rejected" })}>Reject</button>
            </div>
          ))}
        </div>
      )}

      {/* ── Ask-user input banner ── */}
      {pendingInputs.length > 0 && (
        <div style={{
          background: "rgba(139,92,246,0.07)",
          borderBottom: "1px solid rgba(139,92,246,0.22)",
          padding: "12px 20px", flexShrink: 0,
          display: "flex", flexDirection: "column", gap: 10,
        }}>
          {pendingInputs.map((pi) => (
            <div key={pi.id} style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              <div style={{ display: "flex", alignItems: "flex-start", gap: 8 }}>
                <MessageSquare size={14} color="#a78bfa" style={{ flexShrink: 0, marginTop: 2 }} />
                <div style={{ flex: 1 }}>
                  <span style={{ fontSize: 12, fontWeight: 600, color: "#a78bfa" }}>Agent is asking: </span>
                  <span style={{ fontSize: 12, color: "var(--text-primary)", lineHeight: 1.5 }}>{pi.question}</span>
                </div>
              </div>
              <div style={{ display: "flex", gap: 8, paddingLeft: 22 }}>
                <input
                  ref={el => { inputRefs.current[pi.id] = el; }}
                  type="text"
                  placeholder="Type your answer…"
                  value={inputAnswers[pi.id] ?? ""}
                  onChange={e => setInputAnswers(prev => ({ ...prev, [pi.id]: e.target.value }))}
                  onKeyDown={e => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      respondMutation.mutate({ inputId: pi.id, answer: inputAnswers[pi.id] ?? "" });
                    }
                  }}
                  style={{
                    flex: 1, background: "var(--surface-2)", border: "1px solid rgba(139,92,246,0.35)",
                    borderRadius: 7, padding: "6px 10px", fontSize: 12,
                    color: "var(--text-primary)", outline: "none",
                  }}
                  autoFocus
                />
                <button
                  className="btn-primary btn-sm"
                  disabled={respondMutation.isPending}
                  onClick={() => respondMutation.mutate({ inputId: pi.id, answer: inputAnswers[pi.id] ?? "" })}
                  style={{ display: "flex", alignItems: "center", gap: 5, paddingLeft: 10, paddingRight: 10 }}
                >
                  {respondMutation.isPending ? <Loader size={12} className="animate-spin" /> : <Send size={12} />}
                  Send
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* ── Body ── */}
      <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>

        {/* Left: timeline */}
        <div style={{ flex: previewArtifact ? "0 0 38%" : 1, minWidth: 0, overflowY: "auto", padding: "24px 24px 32px", transition: "flex 200ms ease" }}>

          {/* Description */}
          {task.description && task.description !== task.title && (
            <div style={{
              padding: "10px 14px", background: "var(--surface-2)",
              border: "1px solid var(--border)", borderRadius: 9,
              marginBottom: 24, fontSize: 13, color: "var(--text-secondary)", lineHeight: 1.6,
            }}>
              {task.description}
            </div>
          )}

          {/* Timeline */}
          {plan && plan.steps.length > 0 && (
            <div>
              <div style={{ fontSize: 11, fontWeight: 600, letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-faint)", marginBottom: 16 }}>
                Execution Plan — {plan.steps.length} step{plan.steps.length !== 1 ? "s" : ""}
              </div>
              <div style={{ paddingLeft: 4 }}>
                {plan.steps.map((step, idx) => (
                  <StepItem
                    key={step.id}
                    step={step}
                    expanded={expandedSteps.has(step.id)}
                    onToggle={() => toggleStep(step.id)}
                    isLast={idx === plan.steps.length - 1}
                  />
                ))}
              </div>
            </div>
          )}

          {/* States */}
          {task.status === "pending" && !plan && (
            <div style={{ textAlign: "center", padding: "48px 0" }}>
              <div style={{ fontSize: 13, color: "var(--text-faint)", marginBottom: 16 }}>
                Ready to execute. Click Run to start the agent.
              </div>
              <button className="btn-primary" disabled={runMutation.isPending} onClick={() => runMutation.mutate()}>
                <Play size={14} /> {runMutation.isPending ? "Starting…" : "Run Task"}
              </button>
            </div>
          )}

          {isActive && !plan && (
            <div style={{ textAlign: "center", padding: "48px 0" }}>
              <Loader size={24} className="animate-spin" style={{ color: "var(--accent)", margin: "0 auto 12px" }} />
              <div style={{ fontSize: 13, color: "var(--text-secondary)" }}>Planning execution…</div>
            </div>
          )}

          {task.status === "completed" && (
            <div style={{
              marginTop: plan?.steps.length ? 24 : 0,
              padding: "12px 16px",
              background: "rgba(52,211,153,0.06)",
              border: "1px solid rgba(52,211,153,0.18)",
              borderRadius: 9,
              display: "flex", alignItems: "center", gap: 10,
            }}>
              <CheckCircle size={16} color="var(--green)" />
              <span style={{ fontSize: 13, color: "var(--green)", fontWeight: 500 }}>
                Completed in {formatDuration(task.started_at, task.completed_at)}
              </span>
              {artifacts.length > 0 && (
                <span style={{ fontSize: 12, color: "var(--text-muted)", marginLeft: 4 }}>
                  · {artifacts.length} artifact{artifacts.length !== 1 ? "s" : ""} produced
                </span>
              )}
            </div>
          )}

          {task.status === "failed" && (
            <div style={{
              marginTop: 16, padding: "12px 16px",
              background: "rgba(248,113,113,0.06)",
              border: "1px solid rgba(248,113,113,0.2)",
              borderRadius: 9, display: "flex", alignItems: "center", gap: 10,
            }}>
              <XCircle size={16} color="var(--red)" />
              <span style={{ fontSize: 13, color: "var(--red)", fontWeight: 500 }}>Run failed</span>
            </div>
          )}
        </div>

        {/* Center: content preview panel (split view) */}
        {previewArtifact && (
          <div style={{ flex: 1, minWidth: 0, overflow: "hidden", display: "flex", flexDirection: "column" }}>
            <ContentPreviewPanel
              artifact={previewArtifact}
              onClose={() => setPreviewArtifact(null)}
            />
          </div>
        )}

        {/* Right panel */}
        <div className="right-panel" style={{ display: "flex", flexDirection: "column" }}>
          {/* Tabs */}
          <div style={{
            display: "flex", borderBottom: "1px solid var(--border)", flexShrink: 0,
          }}>
            {([
              { id: "artifacts", icon: FileText, label: "Artifacts", count: artifacts.length },
              { id: "events",    icon: Activity, label: "Events",    count: events.length },
              { id: "agents",    icon: Users,    label: "Agents",    count: agents.length },
            ] as { id: RightTab; icon: React.ElementType; label: string; count: number }[]).map(({ id, icon: Icon, label, count }) => (
              <button
                key={id}
                onClick={() => setRightTab(id)}
                style={{
                  flex: 1, display: "flex", flexDirection: "column", alignItems: "center",
                  gap: 3, padding: "9px 4px",
                  background: "transparent", border: "none", cursor: "pointer",
                  borderBottom: `2px solid ${rightTab === id ? "var(--accent)" : "transparent"}`,
                  color: rightTab === id ? "#a5b4fc" : "var(--text-faint)",
                  transition: "all 120ms", marginBottom: -1,
                }}
              >
                <Icon size={13} />
                <span style={{ fontSize: 10.5, fontWeight: 600, letterSpacing: "0.03em" }}>
                  {label}{count > 0 ? ` (${count})` : ""}
                </span>
              </button>
            ))}
          </div>

          {/* Panel content */}
          <div style={{ flex: 1, overflowY: "auto", padding: 12 }}>
            {rightTab === "artifacts" && (
              <ArtifactsPanel
                artifacts={artifacts}
                onPreview={(a) => setPreviewArtifact(previewArtifact?.id === a.id ? null : a)}
                previewId={previewArtifact?.id}
              />
            )}
            {rightTab === "events"    && <EventsPanel events={events} />}
            {rightTab === "agents"    && <AgentsPanel agents={agents} />}
          </div>
        </div>
      </div>
    </div>
  );
}
