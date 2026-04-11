import { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Paperclip,
  FolderOpen,
  Play,
  FileText,
  FileSpreadsheet,
  Presentation,
  Search,
  Archive,
  Sparkles,
  Clock,
  ChevronRight,
  Loader,
  CheckCircle,
  XCircle,
  Circle,
  Zap,
} from "lucide-react";
import { tasksApi, executionApi, providersApi, type Task, type TaskStatus, type ProviderProfile } from "../lib/api";
import { useAppStore } from "../store/appStore";

// ── Suggested actions ────────────────────────────────────────────
const SUGGESTIONS = [
  { icon: FileText,        label: "Summarize a PDF",            prompt: "Summarize this PDF and create a DOCX report." },
  { icon: Presentation,   label: "Create a presentation",       prompt: "Convert this document into a PowerPoint presentation with 10 slides." },
  { icon: FileSpreadsheet, label: "Analyze a spreadsheet",      prompt: "Analyze this Excel file and produce an XLSX summary with key insights." },
  { icon: Search,          label: "Search documents",            prompt: "Search this folder of documents and produce a summary of findings." },
  { icon: Archive,         label: "Organize files",              prompt: "Organize files in this folder by type and date." },
  { icon: FileText,        label: "Compare PDFs",                prompt: "Compare these PDF documents and generate an executive summary." },
];

// ── Status helpers ────────────────────────────────────────────────
const STATUS_BADGE: Record<TaskStatus, string> = {
  pending:   "badge-gray",
  planning:  "badge-accent",
  running:   "badge-yellow",
  completed: "badge-green",
  failed:    "badge-red",
  cancelled: "badge-gray",
};

function StatusDot({ status }: { status: TaskStatus }) {
  switch (status) {
    case "running":   return <Loader size={11} color="var(--yellow)" className="animate-spin" />;
    case "planning":  return <Loader size={11} color="var(--accent)" className="animate-spin" />;
    case "completed": return <CheckCircle size={11} color="var(--green)" />;
    case "failed":    return <XCircle size={11} color="var(--red)" />;
    default:          return <Circle size={11} color="var(--text-faint)" />;
  }
}

function elapsed(task: Task): string {
  const start = task.started_at ? new Date(task.started_at) : new Date(task.created_at);
  const end = task.completed_at ? new Date(task.completed_at) : new Date();
  const s = Math.floor((end.getTime() - start.getTime()) / 1000);
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m ${s % 60}s`;
  return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
}

// ── Composer ──────────────────────────────────────────────────────
function TaskComposer({
  onRunCreated,
  profiles,
}: {
  onRunCreated: (taskId: string) => void;
  profiles: ProviderProfile[];
}) {
  const [text, setText] = useState("");
  const [selectedProfile, setSelectedProfile] = useState<string>("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const qc = useQueryClient();

  // Auto-focus on mount
  useEffect(() => { textareaRef.current?.focus(); }, []);

  // Auto-grow textarea
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`;
  }, [text]);

  const createMutation = useMutation({
    mutationFn: async () => {
      const task = await tasksApi.create({
        title: text.trim().slice(0, 120),
        description: text.trim(),
        provider_id: selectedProfile || undefined,
      });
      await executionApi.runTask(task.id, { require_approval: false });
      return task;
    },
    onSuccess: (task) => {
      qc.invalidateQueries({ queryKey: ["tasks"] });
      setText("");
      onRunCreated(task.id);
    },
  });

  const canSubmit = text.trim().length > 0 && !createMutation.isPending;

  const handleKey = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter" && canSubmit) {
      e.preventDefault();
      createMutation.mutate();
    }
  };

  const fillSuggestion = (prompt: string) => {
    setText(prompt);
    textareaRef.current?.focus();
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      {/* Composer box */}
      <div className="composer-wrap">
        <textarea
          ref={textareaRef}
          className="composer-textarea selectable"
          placeholder="Describe a task — summarize documents, create a presentation, organize a folder…"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKey}
          rows={3}
        />

        {/* Composer toolbar */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            padding: "8px 12px",
            borderTop: "1px solid var(--border)",
          }}
        >
          {/* Attach actions (cosmetic for now) */}
          <button className="btn-ghost btn-sm" title="Attach file">
            <Paperclip size={13} />
            <span>File</span>
          </button>
          <button className="btn-ghost btn-sm" title="Attach folder">
            <FolderOpen size={13} />
            <span>Folder</span>
          </button>

          <div style={{ flex: 1 }} />

          {/* Provider selector */}
          {profiles.length > 0 && (
            <select
              value={selectedProfile}
              onChange={(e) => setSelectedProfile(e.target.value)}
              style={{
                background: "var(--surface-3)",
                border: "1px solid var(--border)",
                borderRadius: 6,
                padding: "4px 8px",
                fontSize: 12,
                color: "var(--text-secondary)",
                outline: "none",
                cursor: "pointer",
              }}
            >
              <option value="">Default provider</option>
              {profiles.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          )}

          {/* Run button */}
          <button
            className="btn-primary btn-sm"
            disabled={!canSubmit}
            onClick={() => createMutation.mutate()}
            style={{ minWidth: 80 }}
          >
            {createMutation.isPending ? (
              <>
                <Loader size={12} className="animate-spin" />
                Starting…
              </>
            ) : (
              <>
                <Play size={12} />
                Run
                <kbd
                  style={{
                    fontSize: 10,
                    opacity: 0.7,
                    background: "rgba(255,255,255,0.1)",
                    borderRadius: 3,
                    padding: "0 4px",
                    marginLeft: 2,
                  }}
                >
                  ⌘↵
                </kbd>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Error */}
      {createMutation.isError && (
        <div
          style={{
            padding: "8px 12px",
            background: "var(--red-dim)",
            border: "1px solid rgba(248,113,113,0.2)",
            borderRadius: 8,
            fontSize: 12,
            color: "var(--red)",
          }}
        >
          {createMutation.error instanceof Error
            ? createMutation.error.message
            : "Failed to start run"}
        </div>
      )}

      {/* Suggestions */}
      <div style={{ display: "flex", flexWrap: "wrap", gap: 7 }}>
        {SUGGESTIONS.map(({ icon: Icon, label, prompt }) => (
          <button
            key={label}
            className="suggestion-chip"
            onClick={() => fillSuggestion(prompt)}
          >
            <Icon size={13} style={{ flexShrink: 0, opacity: 0.7 }} />
            {label}
          </button>
        ))}
      </div>
    </div>
  );
}

// ── Recent Runs ───────────────────────────────────────────────────
function RecentRuns({ tasks, onSelect }: { tasks: Task[]; onSelect: (id: string) => void }) {
  if (tasks.length === 0) {
    return (
      <div
        style={{
          padding: "24px 0",
          textAlign: "center",
          color: "var(--text-faint)",
          fontSize: 12,
        }}
      >
        No runs yet. Type a task above to get started.
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {tasks.slice(0, 12).map((task) => (
        <div
          key={task.id}
          className="card-hover"
          onClick={() => onSelect(task.id)}
          style={{ padding: "10px 14px", display: "flex", alignItems: "center", gap: 10 }}
        >
          <StatusDot status={task.status} />

          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontSize: 13,
                fontWeight: 500,
                color: "var(--text-primary)",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
                letterSpacing: "-0.005em",
              }}
            >
              {task.title}
            </div>
            {task.description && task.description !== task.title && (
              <div
                style={{
                  fontSize: 11.5,
                  color: "var(--text-muted)",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                  marginTop: 1,
                }}
              >
                {task.description}
              </div>
            )}
          </div>

          <div style={{ display: "flex", alignItems: "center", gap: 8, flexShrink: 0 }}>
            <span className={STATUS_BADGE[task.status] ?? "badge-gray"}>
              {task.status}
            </span>
            <span
              style={{
                fontSize: 11,
                color: "var(--text-faint)",
                display: "flex",
                alignItems: "center",
                gap: 3,
              }}
            >
              <Clock size={10} />
              {elapsed(task)}
            </span>
            <ChevronRight size={13} color="var(--text-faint)" />
          </div>
        </div>
      ))}
    </div>
  );
}

// ── Active Run Banner ─────────────────────────────────────────────
function ActiveRunBanner({ task, onClick }: { task: Task; onClick: () => void }) {
  return (
    <div
      onClick={onClick}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 16px",
        background:
          task.status === "running" || task.status === "planning"
            ? "rgba(251,191,36,0.06)"
            : "var(--surface-2)",
        border: `1px solid ${task.status === "running" || task.status === "planning" ? "rgba(251,191,36,0.2)" : "var(--border)"}`,
        borderRadius: 9,
        cursor: "pointer",
        transition: "background 120ms",
        userSelect: "none",
      }}
    >
      <div
        style={{
          width: 28,
          height: 28,
          borderRadius: 7,
          background: "rgba(251,191,36,0.12)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexShrink: 0,
        }}
      >
        <Zap size={14} color="var(--yellow)" />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 12,
            fontWeight: 600,
            color: "var(--yellow)",
            marginBottom: 1,
          }}
        >
          Run in progress
        </div>
        <div
          style={{
            fontSize: 12,
            color: "var(--text-secondary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {task.title}
        </div>
      </div>
      <span className="btn-ghost btn-sm" style={{ pointerEvents: "none", flexShrink: 0 }}>
        View <ChevronRight size={12} />
      </span>
    </div>
  );
}

// ── HomePage ──────────────────────────────────────────────────────
export function HomePage() {
  const navigate = useNavigate();
  const { backendReachable } = useAppStore();

  const { data: tasks = [], isLoading: tasksLoading } = useQuery({
    queryKey: ["tasks"],
    queryFn: () => tasksApi.listAll(20),
    enabled: backendReachable,
    refetchInterval: (query) => {
      const list = query.state.data as Task[] | undefined;
      if (!list) return 5000;
      const hasActive = list.some(
        (t) => t.status === "planning" || t.status === "running"
      );
      return hasActive ? 2000 : 5000;
    },
  });

  const { data: profiles = [] } = useQuery({
    queryKey: ["provider-profiles"],
    queryFn: providersApi.listProfiles,
    enabled: backendReachable,
  });

  const activeTask = tasks.find(
    (t) => t.status === "planning" || t.status === "running"
  );

  const goToRun = (taskId: string) => navigate(`/tasks/${taskId}/run`);

  return (
    <div
      className="page-enter"
      style={{
        height: "100%",
        overflow: "hidden",
        display: "flex",
        flexDirection: "column",
        background: "var(--bg)",
      }}
    >
      {/* Top bar */}
      <div
        style={{
          height: "var(--topbar-h)",
          borderBottom: "1px solid var(--border)",
          display: "flex",
          alignItems: "center",
          padding: "0 24px",
          flexShrink: 0,
          background: "var(--surface-1)",
        }}
      >
        <span
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: "var(--text-primary)",
            letterSpacing: "-0.015em",
          }}
        >
          Home
        </span>
        <div style={{ flex: 1 }} />
        <button className="btn-ghost btn-sm" onClick={() => navigate("/runs")}>
          All runs
          <ChevronRight size={12} />
        </button>
      </div>

      {/* Scrollable content */}
      <div
        style={{
          flex: 1,
          overflowY: "auto",
          padding: "32px 0",
        }}
      >
        <div
          style={{
            maxWidth: 740,
            margin: "0 auto",
            padding: "0 24px",
            display: "flex",
            flexDirection: "column",
            gap: 32,
          }}
        >
          {/* Header */}
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            <h1
              style={{
                fontSize: 22,
                fontWeight: 600,
                color: "var(--text-primary)",
                letterSpacing: "-0.03em",
                margin: 0,
                lineHeight: 1.2,
              }}
            >
              What do you want to do?
            </h1>
            <p
              style={{
                fontSize: 13.5,
                color: "var(--text-secondary)",
                margin: 0,
              }}
            >
              Describe a task — summarize, transform, organize, or generate files and documents.
            </p>
          </div>

          {/* Active run banner */}
          {activeTask && (
            <ActiveRunBanner task={activeTask} onClick={() => goToRun(activeTask.id)} />
          )}

          {/* Composer */}
          {!backendReachable ? (
            <div
              style={{
                padding: "20px",
                background: "var(--surface-2)",
                border: "1px solid var(--border)",
                borderRadius: 12,
                textAlign: "center",
                color: "var(--text-muted)",
                fontSize: 13,
              }}
            >
              Connecting to backend…
            </div>
          ) : (
            <TaskComposer onRunCreated={goToRun} profiles={profiles} />
          )}

          {/* Recent runs */}
          <div>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                marginBottom: 12,
              }}
            >
              <span
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  letterSpacing: "0.07em",
                  textTransform: "uppercase",
                  color: "var(--text-faint)",
                }}
              >
                Recent Runs
              </span>
              <button
                className="btn-ghost btn-xs"
                onClick={() => navigate("/runs")}
                style={{ gap: 3 }}
              >
                View all
                <ChevronRight size={11} />
              </button>
            </div>

            {tasksLoading ? (
              <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                {[1, 2, 3].map((i) => (
                  <div
                    key={i}
                    className="skeleton"
                    style={{ height: 52, borderRadius: 9 }}
                  />
                ))}
              </div>
            ) : (
              <RecentRuns tasks={tasks} onSelect={goToRun} />
            )}
          </div>

          {/* Quick links */}
          {!tasksLoading && tasks.length === 0 && (
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "1fr 1fr",
                gap: 10,
              }}
            >
              {[
                { icon: Sparkles, label: "Browse Skills", to: "/skills" },
                { icon: FolderOpen, label: "View Projects", to: "/projects" },
              ].map(({ icon: Icon, label, to }) => (
                <div
                  key={to}
                  className="card-hover"
                  onClick={() => navigate(to)}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 10,
                    padding: "12px 16px",
                  }}
                >
                  <div
                    style={{
                      width: 30,
                      height: 30,
                      borderRadius: 8,
                      background: "var(--accent-dim)",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      flexShrink: 0,
                    }}
                  >
                    <Icon size={14} color="var(--accent)" />
                  </div>
                  <span
                    style={{
                      fontSize: 13,
                      fontWeight: 500,
                      color: "var(--text-secondary)",
                    }}
                  >
                    {label}
                  </span>
                  <ChevronRight
                    size={13}
                    color="var(--text-faint)"
                    style={{ marginLeft: "auto" }}
                  />
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
