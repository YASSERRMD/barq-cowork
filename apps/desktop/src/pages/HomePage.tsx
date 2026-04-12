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
  Clock,
  ChevronRight,
  Loader,
  CheckCircle,
  XCircle,
  Circle,
  Zap,
} from "lucide-react";
import {
  tasksApi,
  executionApi,
  providersApi,
  type Task,
  type TaskStatus,
  type ProviderProfile,
} from "../lib/api";
import { useAppStore } from "../store/appStore";

// ── Suggested actions ────────────────────────────────────────────
const SUGGESTIONS = [
  {
    icon: FileText,
    label: "Summarize a PDF",
    prompt: "Summarize this PDF and create a DOCX report.",
  },
  {
    icon: Presentation,
    label: "Create a presentation",
    prompt:
      "Create a subject-specific PowerPoint presentation. Plan the deck first, choose a fresh visual style, cover style, color story, motif, and full palette from the subject, do not reuse an old cover pattern, vary slide types, and use real icons, charts, timelines, comparisons, or tables only where they improve the slide.",
  },
  {
    icon: FileSpreadsheet,
    label: "Analyze a spreadsheet",
    prompt:
      "Analyze this Excel file and produce an XLSX summary with key insights.",
  },
  {
    icon: Search,
    label: "Search documents",
    prompt:
      "Search this folder of documents and produce a summary of findings.",
  },
  {
    icon: Archive,
    label: "Organize files",
    prompt: "Organize files in this folder by type and date.",
  },
  {
    icon: FileText,
    label: "Compare PDFs",
    prompt: "Compare these PDF documents and generate an executive summary.",
  },
];

// ── Status helpers ────────────────────────────────────────────────
const STATUS_BADGE: Record<TaskStatus, string> = {
  pending: "badge-gray",
  planning: "badge-accent",
  running: "badge-yellow",
  completed: "badge-green",
  failed: "badge-red",
  cancelled: "badge-gray",
};

function StatusDot({ status }: { status: TaskStatus }) {
  switch (status) {
    case "running":
      return (
        <Loader size={11} color="var(--yellow)" className="animate-spin" />
      );
    case "planning":
      return (
        <Loader size={11} color="var(--accent)" className="animate-spin" />
      );
    case "completed":
      return <CheckCircle size={11} color="var(--green)" />;
    case "failed":
      return <XCircle size={11} color="var(--red)" />;
    default:
      return <Circle size={11} color="var(--text-faint)" />;
  }
}

function elapsed(task: Task): string {
  const start = task.started_at
    ? new Date(task.started_at)
    : new Date(task.created_at);
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
  const [selectedProfile, setSelectedProfile] = useState("");
  const [attachedFiles, setAttachedFiles] = useState<File[]>([]);
  const [uploadedPaths, setUploadedPaths] = useState<string[]>([]);
  const [folderPath, setFolderPath] = useState("");
  const [showFolderInput, setShowFolderInput] = useState(false);
  const [uploading, setUploading] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const qc = useQueryClient();

  // Auto-focus on mount
  useEffect(() => {
    textareaRef.current?.focus();
  }, []);

  // Auto-grow textarea
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`;
  }, [text]);

  const createMutation = useMutation({
    mutationFn: async () => {
      // Upload files if any
      let filePaths: string[] = uploadedPaths;
      if (attachedFiles.length > 0) {
        setUploading(true);
        try {
          const result = await executionApi.uploadFiles(attachedFiles);
          filePaths = result.paths;
        } finally {
          setUploading(false);
        }
      }

      // Build description
      let desc = text.trim();
      if (filePaths.length > 0) {
        desc += `\n\nAttached files for you to work on: ${filePaths.join(", ")}`;
      }
      if (folderPath.trim()) {
        desc += `\n\nWork on files in this folder: ${folderPath.trim()}`;
      }

      const task = await tasksApi.create({
        title: text.trim().slice(0, 120),
        description: desc,
        provider_id: selectedProfile || undefined,
      });
      await executionApi.runTask(task.id, {
        require_approval: false,
        workspace_root: folderPath.trim() || undefined,
      });
      return task;
    },
    onSuccess: (task) => {
      qc.invalidateQueries({ queryKey: ["tasks"] });
      setText("");
      setAttachedFiles([]);
      setUploadedPaths([]);
      setFolderPath("");
      setShowFolderInput(false);
      onRunCreated(task.id);
    },
  });

  const canSubmit =
    text.trim().length > 0 && !createMutation.isPending && !uploading;

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
        {/* Hidden file input */}
        <input
          ref={fileInputRef}
          type="file"
          multiple
          style={{ display: "none" }}
          onChange={(e) => {
            const files = Array.from(e.target.files || []);
            setAttachedFiles((prev) => [...prev, ...files]);
            e.target.value = "";
          }}
        />

        <textarea
          ref={textareaRef}
          className="composer-textarea selectable"
          placeholder="Describe a task — summarize documents, create a presentation, organize a folder…"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKey}
          rows={3}
          disabled={false}
        />

        {/* Attached files chips */}
        {attachedFiles.length > 0 && (
          <div
            style={{
              display: "flex",
              flexWrap: "wrap",
              gap: 4,
              padding: "6px 12px",
              borderTop: "1px solid var(--border)",
            }}
          >
            {attachedFiles.map((f, i) => (
              <span
                key={i}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 4,
                  background: "var(--surface-3)",
                  border: "1px solid var(--border)",
                  borderRadius: 5,
                  padding: "2px 8px",
                  fontSize: 11,
                  color: "var(--text-secondary)",
                }}
              >
                <Paperclip size={10} />
                {f.name}
                <button
                  onClick={() =>
                    setAttachedFiles((p) => p.filter((_, j) => j !== i))
                  }
                  style={{
                    background: "none",
                    border: "none",
                    cursor: "pointer",
                    color: "var(--text-faint)",
                    padding: 0,
                    lineHeight: 1,
                  }}
                >
                  ✕
                </button>
              </span>
            ))}
          </div>
        )}

        {/* Folder path chip */}
        {folderPath && (
          <div
            style={{
              padding: "4px 12px",
              borderTop: "1px solid var(--border)",
              display: "flex",
              alignItems: "center",
              gap: 6,
            }}
          >
            <FolderOpen size={11} color="var(--accent)" />
            <span
              style={{
                fontSize: 11,
                color: "var(--accent)",
                fontFamily: "monospace",
              }}
            >
              {folderPath}
            </span>
            <button
              onClick={() => setFolderPath("")}
              style={{
                background: "none",
                border: "none",
                cursor: "pointer",
                color: "var(--text-faint)",
                fontSize: 11,
              }}
            >
              ✕
            </button>
          </div>
        )}

        {/* Folder path input */}
        {showFolderInput && (
          <div
            style={{
              padding: "8px 12px",
              borderTop: "1px solid var(--border)",
              display: "flex",
              alignItems: "center",
              gap: 8,
            }}
          >
            <FolderOpen size={13} color="var(--accent)" />
            <input
              style={{
                flex: 1,
                background: "transparent",
                border: "none",
                outline: "none",
                fontSize: 12,
                color: "var(--text-secondary)",
              }}
              placeholder="Paste folder path, e.g. /Users/you/Documents/Reports"
              value={folderPath}
              onChange={(e) => setFolderPath(e.target.value)}
            />
            {folderPath && (
              <button
                className="btn-ghost btn-xs"
                onClick={() => {
                  setFolderPath("");
                  setShowFolderInput(false);
                }}
              >
                ✕
              </button>
            )}
          </div>
        )}

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
          {/* File attach button */}
          <button
            className="btn-ghost btn-sm"
            title="Attach file"
            onClick={() => fileInputRef.current?.click()}
          >
            <Paperclip size={13} />
            <span>
              {attachedFiles.length > 0
                ? `${attachedFiles.length} file${attachedFiles.length > 1 ? "s" : ""}`
                : "File"}
            </span>
          </button>

          {/* Folder attach button */}
          <button
            className="btn-ghost btn-sm"
            title="Attach folder"
            onClick={() => setShowFolderInput((v) => !v)}
            style={{ color: folderPath ? "var(--accent)" : undefined }}
          >
            <FolderOpen size={13} />
            <span>{folderPath ? "Folder set" : "Folder"}</span>
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
            {uploading ? (
              <>
                <Loader size={12} className="animate-spin" />
                Uploading…
              </>
            ) : createMutation.isPending ? (
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
        (t) => t.status === "planning" || t.status === "running",
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
    (t) => t.status === "planning" || t.status === "running",
  );
  const recentTasks = tasks.slice(0, 8);
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
      {/* ── Slim top bar ── */}
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
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <div
            style={{
              width: 22,
              height: 22,
              borderRadius: 6,
              background: "var(--accent)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Zap size={11} color="#fff" strokeWidth={2.5} />
          </div>
          <span
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: "var(--text-primary)",
              letterSpacing: "-0.01em",
            }}
          >
            Barq
          </span>
        </div>
        <div style={{ flex: 1 }} />
        {activeTask && (
          <button
            className="btn-ghost btn-sm"
            onClick={() => goToRun(activeTask.id)}
            style={{ gap: 6, color: "var(--yellow)" }}
          >
            <Loader size={12} className="animate-spin" />
            Running…
          </button>
        )}
        <button
          className="btn-ghost btn-sm"
          style={{ marginLeft: 4 }}
          onClick={() => navigate("/runs")}
        >
          History
          <ChevronRight size={12} />
        </button>
      </div>

      {/* ── Main: two-column layout ── */}
      <div style={{ flex: 1, overflow: "hidden", display: "flex" }}>
        {/* Left: Composer + suggestions (centered) */}
        <div
          style={{
            flex: 1,
            overflowY: "auto",
            display: "flex",
            flexDirection: "column",
          }}
        >
          {/* Hero area */}
          <div
            style={{
              flex: 1,
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              justifyContent: "center",
              padding: "48px 40px 32px",
              minHeight: 340,
            }}
          >
            <div style={{ width: "100%", maxWidth: 680 }}>
              {/* Greeting */}
              <div style={{ marginBottom: 28, textAlign: "center" }}>
                <h1
                  style={{
                    fontSize: 30,
                    fontWeight: 700,
                    color: "var(--text-primary)",
                    letterSpacing: "-0.04em",
                    margin: "0 0 8px",
                    lineHeight: 1.15,
                  }}
                >
                  What can I help you with?
                </h1>
                <p
                  style={{
                    fontSize: 14,
                    color: "var(--text-muted)",
                    margin: 0,
                  }}
                >
                  Describe a task and your AI agent will take care of it.
                </p>
              </div>

              {/* Composer */}
              {!backendReachable ? (
                <div
                  style={{
                    padding: "18px 20px",
                    background: "var(--surface-2)",
                    border: "1px solid var(--border)",
                    borderRadius: 14,
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
            </div>
          </div>

          {/* Recent runs — horizontal scroll row */}
          {!tasksLoading && recentTasks.length > 0 && (
            <div style={{ padding: "0 40px 32px", flexShrink: 0 }}>
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
                    fontWeight: 700,
                    letterSpacing: "0.08em",
                    textTransform: "uppercase",
                    color: "var(--text-faint)",
                  }}
                >
                  Recent
                </span>
                <button
                  className="btn-ghost btn-xs"
                  onClick={() => navigate("/runs")}
                  style={{ gap: 3 }}
                >
                  All <ChevronRight size={10} />
                </button>
              </div>
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fill, minmax(220px, 1fr))",
                  gap: 8,
                }}
              >
                {recentTasks.map((task) => (
                  <div
                    key={task.id}
                    onClick={() => goToRun(task.id)}
                    style={{
                      padding: "11px 14px",
                      background: "var(--surface-1)",
                      border: "1px solid var(--border)",
                      borderRadius: 10,
                      cursor: "pointer",
                      transition: "all 120ms",
                      display: "flex",
                      flexDirection: "column",
                      gap: 6,
                    }}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLElement).style.borderColor =
                        "var(--border-mid)";
                      (e.currentTarget as HTMLElement).style.boxShadow =
                        "var(--shadow-md)";
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLElement).style.borderColor =
                        "var(--border)";
                      (e.currentTarget as HTMLElement).style.boxShadow = "none";
                    }}
                  >
                    <div
                      style={{ display: "flex", alignItems: "center", gap: 6 }}
                    >
                      <StatusDot status={task.status} />
                      <span
                        className={STATUS_BADGE[task.status] ?? "badge-gray"}
                        style={{ fontSize: 10 }}
                      >
                        {task.status}
                      </span>
                    </div>
                    <div
                      style={{
                        fontSize: 12.5,
                        fontWeight: 500,
                        color: "var(--text-primary)",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                        lineHeight: 1.4,
                      }}
                    >
                      {task.title}
                    </div>
                    <div
                      style={{
                        fontSize: 11,
                        color: "var(--text-faint)",
                        display: "flex",
                        alignItems: "center",
                        gap: 3,
                      }}
                    >
                      <Clock size={9} />
                      {elapsed(task)}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {tasksLoading && (
            <div style={{ padding: "0 40px 32px" }}>
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fill, minmax(220px, 1fr))",
                  gap: 8,
                }}
              >
                {[1, 2, 3, 4].map((i) => (
                  <div
                    key={i}
                    className="skeleton"
                    style={{ height: 72, borderRadius: 10 }}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
