import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Play, FileText, Activity, Users, ChevronDown, ChevronRight,
  Clock, CheckCircle, XCircle, Loader, Circle, ArrowLeft,
  AlertTriangle, Download, Copy, Zap, Terminal, Maximize2, X,
  MessageSquare, Send, HelpCircle, Bot,
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
    case "running":   return <Loader size={13} color="var(--yellow)" className="animate-spin" />;
    case "completed": return <CheckCircle size={13} color="var(--green)" />;
    case "failed":    return <XCircle size={13} color="var(--red)" />;
    case "skipped":   return <Circle size={13} color="var(--text-faint)" />;
    default:          return <Circle size={13} color="var(--surface-4)" style={{ opacity: 0.5 }} />;
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
      <div style={{ display: "flex", flexDirection: "column", alignItems: "center", width: 26, flexShrink: 0 }}>
        <div style={{
          width: 26, height: 26, borderRadius: "50%", flexShrink: 0,
          background: isRunning ? "rgba(251,191,36,0.12)" : isCompleted ? "rgba(52,211,153,0.10)" : isFailed ? "rgba(248,113,113,0.10)" : "var(--surface-3)",
          border: `1px solid ${isRunning ? "rgba(251,191,36,0.3)" : isCompleted ? "rgba(52,211,153,0.25)" : isFailed ? "rgba(248,113,113,0.25)" : "var(--border)"}`,
          display: "flex", alignItems: "center", justifyContent: "center",
          transition: "all 200ms",
        }}>
          <StepIcon status={step.status} />
        </div>
        {!isLast && (
          <div style={{ width: 1, flex: 1, minHeight: 10, background: isCompleted ? "rgba(52,211,153,0.2)" : "var(--border)", marginTop: 2 }} />
        )}
      </div>

      <div style={{ flex: 1, minWidth: 0, paddingLeft: 10, paddingBottom: isLast ? 0 : 14 }}>
        <div
          onClick={hasDetail ? onToggle : undefined}
          style={{
            display: "flex", alignItems: "flex-start", gap: 6,
            cursor: hasDetail ? "pointer" : "default",
            padding: "3px 6px", borderRadius: 6, marginLeft: -6,
            background: isRunning ? "rgba(251,191,36,0.04)" : "transparent",
            transition: "background 120ms",
          }}
        >
          <div style={{ flex: 1, minWidth: 0, paddingTop: 2 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
              <span style={{
                fontSize: 12, fontWeight: isRunning ? 600 : 500,
                color: isRunning ? "var(--text-primary)" : "var(--text-secondary)",
              }}>
                {step.title}
              </span>
              {step.tool_name && (
                <span style={{
                  fontSize: 10, color: "var(--text-faint)",
                  background: "var(--surface-3)", border: "1px solid var(--border)",
                  borderRadius: 4, padding: "1px 5px", fontFamily: "monospace",
                }}>
                  {step.tool_name}
                </span>
              )}
            </div>
            {step.started_at && (
              <span style={{ fontSize: 10.5, color: "var(--text-faint)", display: "flex", alignItems: "center", gap: 3, marginTop: 2 }}>
                <Clock size={9} />
                {formatDuration(step.started_at, step.completed_at)}
              </span>
            )}
          </div>
          {hasDetail && (
            <div style={{ flexShrink: 0, marginTop: 4 }}>
              {expanded
                ? <ChevronDown size={12} color="var(--text-faint)" />
                : <ChevronRight size={12} color="var(--text-faint)" />}
            </div>
          )}
        </div>

        {expanded && hasDetail && (
          <div style={{
            marginTop: 6,
            background: "var(--surface-1)", border: "1px solid var(--border)",
            borderRadius: 7, overflow: "hidden",
          }}>
            {step.tool_input && (
              <div style={{ borderBottom: step.tool_output ? "1px solid var(--border)" : "none" }}>
                <div style={{
                  display: "flex", alignItems: "center", gap: 5,
                  padding: "5px 10px", borderBottom: "1px solid var(--border)",
                  background: "var(--surface-2)",
                }}>
                  <Terminal size={10} color="var(--text-faint)" />
                  <span style={{ fontSize: 9.5, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase" }}>Input</span>
                </div>
                <pre style={{
                  fontSize: 10.5, color: "var(--text-secondary)", margin: 0,
                  padding: "8px 10px", fontFamily: "JetBrains Mono, ui-monospace, monospace",
                  whiteSpace: "pre-wrap", wordBreak: "break-all", lineHeight: 1.6,
                  maxHeight: 100, overflowY: "auto",
                }}>
                  {step.tool_input}
                </pre>
              </div>
            )}
            {step.tool_output && (
              <div>
                <div style={{
                  display: "flex", alignItems: "center", gap: 5,
                  padding: "5px 10px", borderBottom: "1px solid var(--border)",
                  background: "var(--surface-2)",
                }}>
                  <Zap size={10} color="var(--green)" />
                  <span style={{ fontSize: 9.5, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase" }}>Output</span>
                </div>
                <pre style={{
                  fontSize: 10.5, color: "var(--text-primary)", margin: 0,
                  padding: "8px 10px", fontFamily: "JetBrains Mono, ui-monospace, monospace",
                  whiteSpace: "pre-wrap", wordBreak: "break-all", lineHeight: 1.6,
                  maxHeight: 140, overflowY: "auto",
                }}>
                  {step.tool_output.length > 600 ? step.tool_output.slice(0, 600) + "\n…" : step.tool_output}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ── Sidebar panel ────────────────────────────────────────────────

type SideTab = "steps" | "artifacts" | "events" | "agents";

function ArtifactsList({
  artifacts,
  onPreview,
  previewId,
}: {
  artifacts: Artifact[];
  onPreview?: (a: Artifact) => void;
  previewId?: string;
}) {
  if (!artifacts.length) return (
    <div style={{ padding: "24px 16px", textAlign: "center" }}>
      <FileText size={16} color="var(--text-faint)" style={{ margin: "0 auto 8px", display: "block" }} />
      <p style={{ fontSize: 12, color: "var(--text-muted)", margin: 0 }}>No artifacts yet</p>
    </div>
  );
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      {artifacts.map((a) => {
        const fileName = a.name.split("/").pop() || a.name;
        const ext = fileName.split(".").pop()?.toLowerCase() || "";
        const fileEmoji = ext === "pptx" ? "📊" : ext === "docx" ? "📄" : ext === "pdf" ? "📕" : ext === "xlsx" ? "📈" : null;
        const isActive = a.id === previewId;
        return (
          <div key={a.id} style={{
            padding: "10px 12px",
            background: isActive ? "rgba(99,102,241,0.08)" : "var(--surface-2)",
            border: `1px solid ${isActive ? "rgba(99,102,241,0.35)" : "var(--border)"}`,
            borderRadius: 8,
            cursor: onPreview ? "pointer" : "default",
            transition: "all 120ms",
          }}
            onClick={() => onPreview?.(a)}
          >
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <div style={{ fontSize: 18, flexShrink: 0 }}>
                {fileEmoji || <FileText size={16} color="var(--accent)" />}
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{
                  fontSize: 12, fontWeight: 500, color: "var(--text-primary)",
                  overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                }}>
                  {fileName}
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 3 }}>
                  <span className="badge-gray" style={{ fontSize: 9.5 }}>{a.type}</span>
                  <span style={{ fontSize: 10.5, color: "var(--text-faint)" }}>{formatBytes(a.size)}</span>
                </div>
              </div>
              <div style={{ display: "flex", gap: 4, flexShrink: 0 }} onClick={e => e.stopPropagation()}>
                <button
                  type="button"
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
                    title="Download"
                    style={{ display: "flex", alignItems: "center" }}
                  >
                    <Download size={11} />
                  </a>
                )}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function EventsList({ events }: { events: TaskEvent[] }) {
  if (!events.length) return (
    <div style={{ padding: "24px 16px", textAlign: "center" }}>
      <Activity size={16} color="var(--text-faint)" style={{ margin: "0 auto 8px", display: "block" }} />
      <p style={{ fontSize: 12, color: "var(--text-muted)", margin: 0 }}>No events yet</p>
    </div>
  );
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 3 }}>
      {[...events].reverse().slice(0, 60).map((e) => (
        <div key={e.id} style={{
          padding: "6px 10px", background: "var(--surface-2)",
          border: "1px solid var(--border)", borderRadius: 6,
        }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 2 }}>
            <span className="badge-accent" style={{ fontSize: 9.5 }}>{e.type}</span>
            <span style={{ fontSize: 10, color: "var(--text-faint)" }}>{formatTime(e.created_at)}</span>
          </div>
          {e.payload && (
            <pre style={{
              fontSize: 10, color: "var(--text-muted)", margin: 0,
              fontFamily: "JetBrains Mono, monospace",
              whiteSpace: "pre-wrap", wordBreak: "break-all",
              maxHeight: 40, overflow: "hidden", lineHeight: 1.5,
            }}>
              {e.payload.length > 80 ? e.payload.slice(0, 80) + "…" : e.payload}
            </pre>
          )}
        </div>
      ))}
    </div>
  );
}

function AgentsList({ agents }: { agents: SubAgent[] }) {
  if (!agents.length) return (
    <div style={{ padding: "24px 16px", textAlign: "center" }}>
      <Users size={16} color="var(--text-faint)" style={{ margin: "0 auto 8px", display: "block" }} />
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
          <div style={{ fontSize: 12, fontWeight: 500, color: "var(--text-primary)", marginBottom: 5 }}>
            {a.title}
          </div>
          <div style={{ display: "flex", gap: 5 }}>
            <span className="badge-purple" style={{ fontSize: 9.5 }}>{a.role}</span>
            <span className={
              a.status === "completed" ? "badge-green" :
              a.status === "failed" ? "badge-red" :
              a.status === "running" ? "badge-yellow" : "badge-gray"
            } style={{ fontSize: 9.5 }}>{a.status}</span>
          </div>
        </div>
      ))}
    </div>
  );
}

// ── Chat UI ───────────────────────────────────────────────────────

interface ChatMessage {
  id: string;
  kind: "agent" | "user" | "system";
  text: string;
  isQuestion?: boolean;
  inputId?: string;
  timestamp: string;
}

function buildChatMessages(
  events: TaskEvent[],
  localUserMessages: { id: string; text: string; timestamp: string }[],
  answeredInputIds: Record<string, string>,
): ChatMessage[] {
  const msgs: ChatMessage[] = [];

  for (const ev of events) {
    if (ev.type === "agent.message") {
      let text = "";
      try {
        const p = JSON.parse(ev.payload || "{}");
        text = p.text || "";
      } catch { text = ev.payload || ""; }
      if (text) {
        msgs.push({ id: ev.id, kind: "agent", text, timestamp: ev.created_at });
      }
    } else if (ev.type === "input.needed") {
      let question = "";
      let inputId = "";
      try {
        const p = JSON.parse(ev.payload || "{}");
        question = p.question || p.message || ev.payload || "";
        inputId = p.input_id || p.id || ev.id;
      } catch { question = ev.payload || ""; inputId = ev.id; }
      if (question) {
        msgs.push({ id: ev.id, kind: "agent", text: question, isQuestion: true, inputId, timestamp: ev.created_at });
        if (answeredInputIds[inputId]) {
          msgs.push({ id: `answered-${inputId}`, kind: "user", text: answeredInputIds[inputId], timestamp: ev.created_at });
        }
      }
    } else if (ev.type === "input.answered") {
      let answer = "";
      try {
        const p = JSON.parse(ev.payload || "{}");
        answer = p.answer || p.text || "";
      } catch { answer = ev.payload || ""; }
      if (answer) {
        msgs.push({ id: ev.id, kind: "user", text: answer, timestamp: ev.created_at });
      }
    } else if (ev.type === "step.started") {
      let tool = "";
      try { tool = JSON.parse(ev.payload || "{}").tool || ""; } catch { /* noop */ }
      msgs.push({ id: ev.id, kind: "system", text: `Running ${tool || "step"}…`, timestamp: ev.created_at });
    } else if (ev.type === "artifact.ready") {
      let name = "";
      try { name = JSON.parse(ev.payload || "{}").name || ""; } catch { /* noop */ }
      const fileName = name.split("/").pop() || name || "file";
      msgs.push({ id: ev.id, kind: "system", text: `✅ Artifact ready: ${fileName}`, timestamp: ev.created_at });
    }
  }

  for (const lm of localUserMessages) {
    if (!msgs.find(m => m.id === lm.id)) {
      msgs.push({ id: lm.id, kind: "user", text: lm.text, timestamp: lm.timestamp });
    }
  }

  msgs.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  return msgs;
}

function ChatPanel({
  events,
  pendingInputs,
  isActive,
  taskStatus,
  onRespond,
  respondPending,
}: {
  events: TaskEvent[];
  pendingInputs: PendingInput[];
  isActive: boolean;
  taskStatus: TaskStatus;
  onRespond: (inputId: string, answer: string) => void;
  respondPending: boolean;
}) {
  const [inputText, setInputText] = useState("");
  const [localUserMessages, setLocalUserMessages] = useState<{ id: string; text: string; timestamp: string }[]>([]);
  const [answeredInputIds, setAnsweredInputIds] = useState<Record<string, string>>({});
  const scrollRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const hasPendingInput = pendingInputs.length > 0;
  const pendingInput = pendingInputs[0];
  const messages = buildChatMessages(events, localUserMessages, answeredInputIds);
  const showInput = isActive || hasPendingInput;

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages.length]);

  // Auto-focus textarea when there's a pending input
  useEffect(() => {
    if (hasPendingInput && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [hasPendingInput]);

  const handleSend = () => {
    const text = inputText.trim();
    if (!text) return;
    if (hasPendingInput && pendingInput) {
      setAnsweredInputIds(prev => ({ ...prev, [pendingInput.id]: text }));
      onRespond(pendingInput.id, text);
    } else {
      setLocalUserMessages(prev => [...prev, {
        id: `local-${Date.now()}`,
        text,
        timestamp: new Date().toISOString(),
      }]);
    }
    setInputText("");
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  // Empty state: task not started yet
  if (taskStatus === "pending" && !events.length) {
    return (
      <div style={{
        flex: 1, display: "flex", flexDirection: "column",
        alignItems: "center", justifyContent: "center",
        gap: 12, padding: 40, minHeight: 0,
      }}>
        <div style={{
          width: 56, height: 56, borderRadius: 16,
          background: "linear-gradient(135deg, rgba(99,102,241,0.2), rgba(139,92,246,0.2))",
          border: "1px solid rgba(99,102,241,0.3)",
          display: "flex", alignItems: "center", justifyContent: "center",
        }}>
          <Bot size={24} color="#a5b4fc" />
        </div>
        <div style={{ textAlign: "center" }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text-primary)", marginBottom: 6 }}>
            Ready to start
          </div>
          <div style={{ fontSize: 13, color: "var(--text-muted)" }}>
            Click Run to launch the agent. It will greet you and ask for preferences before starting work.
          </div>
        </div>
      </div>
    );
  }

  return (
    <div style={{ flex: 1, display: "flex", flexDirection: "column", minHeight: 0, overflow: "hidden" }}>
      {/* Messages area */}
      <div
        ref={scrollRef}
        className="selectable"
        style={{
          flex: 1, overflowY: "auto",
          padding: "20px 24px",
          display: "flex", flexDirection: "column", gap: 16,
          minHeight: 0,
        }}
      >
        {messages.length === 0 && isActive && (
          <div style={{ display: "flex", justifyContent: "center", padding: "32px 0" }}>
            <div style={{ display: "flex", alignItems: "center", gap: 10, color: "var(--text-faint)", fontSize: 13 }}>
              <Loader size={14} className="animate-spin" />
              Agent is starting up…
            </div>
          </div>
        )}

        {messages.map((msg) => {
          if (msg.kind === "system") {
            return (
              <div key={msg.id} style={{ display: "flex", justifyContent: "center" }}>
                <span style={{
                  fontSize: 11, color: "var(--text-faint)",
                  background: "var(--surface-3)",
                  border: "1px solid var(--border)",
                  borderRadius: 20, padding: "3px 12px",
                  maxWidth: "70%", textAlign: "center",
                }}>
                  {msg.text}
                </span>
              </div>
            );
          }

          if (msg.kind === "agent") {
            return (
              <div key={msg.id} style={{ display: "flex", gap: 10, maxWidth: "80%" }}>
                {/* Avatar */}
                <div style={{
                  width: 30, height: 30, borderRadius: 10, flexShrink: 0,
                  background: "linear-gradient(135deg, #312e81, #4c1d95)",
                  border: "1px solid rgba(99,102,241,0.35)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  marginTop: 2,
                }}>
                  <Bot size={15} color="#a5b4fc" />
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                  <span style={{ fontSize: 11, fontWeight: 600, color: "var(--text-faint)", paddingLeft: 2 }}>
                    Agent {msg.isQuestion && <span style={{ color: "#a78bfa" }}>· asking</span>}
                  </span>
                  <div style={{
                    background: "#1a1830",
                    border: msg.isQuestion
                      ? "1.5px solid rgba(124,58,237,0.5)"
                      : "1px solid rgba(99,102,241,0.18)",
                    borderRadius: "4px 14px 14px 14px",
                    padding: "10px 14px",
                    fontSize: 13.5, color: "#c4c9f0", lineHeight: 1.65,
                    wordBreak: "break-word",
                    boxShadow: msg.isQuestion ? "0 0 0 3px rgba(124,58,237,0.08)" : "none",
                  }}>
                    {msg.text}
                  </div>
                  {msg.isQuestion && (
                    <div style={{
                      display: "flex", alignItems: "center", gap: 5,
                      paddingLeft: 2, fontSize: 11, color: "#7c3aed", fontWeight: 600,
                    }}>
                      <HelpCircle size={11} />
                      Waiting for your reply
                    </div>
                  )}
                </div>
              </div>
            );
          }

          // user
          return (
            <div key={msg.id} style={{
              display: "flex", justifyContent: "flex-end",
            }}>
              <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-end", gap: 4, maxWidth: "80%" }}>
                <span style={{ fontSize: 11, fontWeight: 600, color: "var(--text-faint)", paddingRight: 2 }}>
                  You
                </span>
                <div style={{
                  background: "linear-gradient(135deg, #4f46e5, #6366f1)",
                  borderRadius: "14px 4px 14px 14px",
                  padding: "10px 14px",
                  fontSize: 13.5, color: "#fff", lineHeight: 1.65,
                  wordBreak: "break-word",
                  boxShadow: "0 2px 8px rgba(99,102,241,0.25)",
                }}>
                  {msg.text}
                </div>
              </div>
            </div>
          );
        })}

        {/* Typing indicator when active but waiting */}
        {isActive && messages.length > 0 && !hasPendingInput && (
          <div style={{ display: "flex", gap: 10, maxWidth: "80%" }}>
            <div style={{
              width: 30, height: 30, borderRadius: 10, flexShrink: 0,
              background: "linear-gradient(135deg, #312e81, #4c1d95)",
              border: "1px solid rgba(99,102,241,0.35)",
              display: "flex", alignItems: "center", justifyContent: "center",
            }}>
              <Bot size={15} color="#a5b4fc" />
            </div>
            <div style={{
              background: "#1a1830",
              border: "1px solid rgba(99,102,241,0.18)",
              borderRadius: "4px 14px 14px 14px",
              padding: "12px 16px",
              display: "flex", alignItems: "center", gap: 5,
            }}>
              {[0, 1, 2].map(i => (
                <div key={i} style={{
                  width: 6, height: 6, borderRadius: "50%",
                  background: "#6366f1", opacity: 0.6,
                  animation: `bounce 1.2s ease-in-out ${i * 0.2}s infinite`,
                }} />
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Input area */}
      <div style={{
        flexShrink: 0,
        padding: "12px 20px 16px",
        borderTop: "1px solid var(--border)",
        background: "var(--surface-1)",
      }}>
        {!showInput && taskStatus !== "pending" && (
          <div style={{
            textAlign: "center", fontSize: 12, color: "var(--text-faint)",
            padding: "8px 0",
            display: "flex", alignItems: "center", justifyContent: "center", gap: 6,
          }}>
            {taskStatus === "completed"
              ? <><CheckCircle size={13} color="var(--green)" /> Task completed</>
              : taskStatus === "failed"
              ? <><XCircle size={13} color="var(--red)" /> Task failed</>
              : "Task inactive"}
          </div>
        )}

        {showInput && (
          <div style={{ display: "flex", gap: 8, alignItems: "flex-end" }}>
            {hasPendingInput && (
              <div style={{
                flexShrink: 0, width: 6, height: 6, borderRadius: "50%",
                background: "#7c3aed", marginBottom: 14,
                boxShadow: "0 0 8px rgba(124,58,237,0.6)",
              }} />
            )}
            <textarea
              ref={textareaRef}
              value={inputText}
              onChange={e => setInputText(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={hasPendingInput ? "Reply to the agent…" : "Send a message…"}
              rows={2}
              style={{
                flex: 1, resize: "none",
                background: "var(--surface-3)",
                border: `1.5px solid ${hasPendingInput ? "rgba(124,58,237,0.45)" : "var(--border-mid)"}`,
                borderRadius: 10, padding: "10px 14px",
                fontSize: 13.5, color: "var(--text-primary)", outline: "none",
                lineHeight: 1.5, maxHeight: 120, overflowY: "auto",
                fontFamily: "inherit",
                transition: "border-color 150ms",
                boxShadow: hasPendingInput ? "0 0 0 3px rgba(124,58,237,0.08)" : "none",
              }}
            />
            <button
              type="button"
              className="btn-primary"
              onClick={handleSend}
              disabled={!inputText.trim() || respondPending}
              style={{
                flexShrink: 0, display: "flex", alignItems: "center",
                gap: 6, height: 44, padding: "0 16px", borderRadius: 10,
                fontSize: 13, fontWeight: 600,
              }}
            >
              {respondPending
                ? <Loader size={14} className="animate-spin" />
                : <Send size={14} />}
              Send
            </button>
          </div>
        )}
      </div>
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
  if (!content) return (
    <div style={{ padding: 16, color: "var(--text-faint)", fontSize: 12, display: "flex", alignItems: "center", gap: 8 }}>
      <Loader size={12} className="animate-spin" /> Loading…
    </div>
  );
  return (
    <pre className="selectable" style={{
      margin: 0, padding: "14px 18px",
      fontSize: 11.5, color: "var(--text-secondary)",
      fontFamily: "JetBrains Mono, monospace", whiteSpace: "pre-wrap",
      wordBreak: "break-all", lineHeight: 1.6, overflowY: "auto", flex: 1,
    }}>
      {content}
    </pre>
  );
}

// ── Content preview panel ─────────────────────────────────────────

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
  const ext = fileName.split(".").pop()?.toLowerCase() || "";

  const fileEmoji =
    ext === "pptx" ? "📊" :
    ext === "docx" ? "📄" :
    ext === "pdf"  ? "📕" :
    ext === "xlsx" ? "📈" : null;

  return (
    <div style={{
      display: "flex", flexDirection: "column", flex: 1,
      background: "var(--surface-1)", minHeight: 0,
    }}>
      {/* Header */}
      <div style={{
        display: "flex", alignItems: "center", gap: 8,
        padding: "10px 16px", borderBottom: "1px solid var(--border)",
        flexShrink: 0, background: "var(--surface-2)",
      }}>
        {fileEmoji
          ? <span style={{ fontSize: 16 }}>{fileEmoji}</span>
          : <FileText size={14} color="var(--accent)" />
        }
        <span style={{
          flex: 1, fontSize: 13, fontWeight: 600, color: "var(--text-primary)",
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
          <Download size={13} />
        </a>
        <button
          type="button"
          className="btn-ghost btn-xs"
          onClick={onClose}
          title="Close preview"
          style={{ flexShrink: 0 }}
        >
          <X size={13} />
        </button>
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflow: "auto", minHeight: 0, display: "flex", flexDirection: "column" }}>
        {artifact.type === "markdown" && (
          <div className="selectable" style={{
            padding: "20px 28px", flex: 1,
            fontSize: 13.5, lineHeight: 1.75,
            color: "var(--text-primary)",
          }}>
            {markdownContent != null ? (
              <div className="markdown-preview" style={{ maxWidth: 720, wordBreak: "break-word" }}>
                <ReactMarkdown>{markdownContent}</ReactMarkdown>
              </div>
            ) : loadError ? (
              <p style={{ color: "var(--red)", fontSize: 13 }}>Failed to load content.</p>
            ) : (
              <div style={{ display: "flex", alignItems: "center", gap: 8, color: "var(--text-faint)", fontSize: 13 }}>
                <Loader size={14} className="animate-spin" /> Loading…
              </div>
            )}
          </div>
        )}

        {artifact.type === "html" && (
          <iframe
            src={downloadUrl}
            style={{ width: "100%", flex: 1, border: "none", background: "#fff", display: "block", minHeight: 0, height: "100%" }}
            title={fileName}
            sandbox="allow-scripts allow-same-origin"
          />
        )}

        {artifact.type === "json" && <JsonPreview downloadUrl={downloadUrl} />}

        {/* PPTX and DOCX: rendered preview via backend conversion */}
        {(ext === "pptx" || ext === "docx") && artifact.type !== "markdown" && artifact.type !== "html" && (
          <div style={{ flex: 1, display: "flex", flexDirection: "column", minHeight: 0 }}>
            <div style={{
              padding: "8px 16px", background: "var(--surface-2)",
              borderBottom: "1px solid var(--border)",
              display: "flex", alignItems: "center", gap: 8, flexShrink: 0,
            }}>
              <a href={downloadUrl} download className="btn-primary btn-sm"
                style={{ display: "inline-flex", alignItems: "center", gap: 6, textDecoration: "none", fontSize: 12 }}>
                <Download size={13} /> Download {ext.toUpperCase()}
              </a>
              <a href={downloadUrl} target="_blank" rel="noreferrer"
                className="btn-ghost btn-sm"
                style={{ display: "inline-flex", alignItems: "center", gap: 5, textDecoration: "none", fontSize: 12 }}>
                <Maximize2 size={12} /> Open
              </a>
            </div>
            <iframe
              src={`http://localhost:7331/api/v1/artifacts/${artifact.id}/preview`}
              style={{ width: "100%", flex: 1, border: "none", display: "block", minHeight: 0 }}
              title={`Preview: ${fileName}`}
            />
          </div>
        )}

        {/* PDF: use browser native viewer */}
        {ext === "pdf" && artifact.type !== "markdown" && artifact.type !== "html" && (
          <div style={{ flex: 1, display: "flex", flexDirection: "column", minHeight: 0 }}>
            <div style={{
              padding: "8px 16px", background: "var(--surface-2)",
              borderBottom: "1px solid var(--border)",
              display: "flex", alignItems: "center", gap: 8, flexShrink: 0,
            }}>
              <a href={downloadUrl} download className="btn-primary btn-sm"
                style={{ display: "inline-flex", alignItems: "center", gap: 6, textDecoration: "none", fontSize: 12 }}>
                <Download size={13} /> Download PDF
              </a>
            </div>
            <iframe
              src={downloadUrl}
              style={{ width: "100%", flex: 1, border: "none", display: "block", minHeight: 0 }}
              title={`Preview: ${fileName}`}
            />
          </div>
        )}

        {/* Other file types: download card */}
        {ext !== "pptx" && ext !== "docx" && ext !== "pdf" &&
         artifact.type !== "markdown" && artifact.type !== "html" && artifact.type !== "json" && (
          <div style={{
            flex: 1, display: "flex", flexDirection: "column",
            alignItems: "center", justifyContent: "center",
            gap: 20, padding: 40,
          }}>
            <div style={{
              width: 80, height: 80, borderRadius: 20,
              background: "var(--accent-dim)",
              display: "flex", alignItems: "center", justifyContent: "center",
              fontSize: 40,
            }}>
              {fileEmoji || <FileText size={36} color="var(--accent)" />}
            </div>
            <div style={{ textAlign: "center" }}>
              <div style={{ fontSize: 16, fontWeight: 700, color: "var(--text-primary)", marginBottom: 6 }}>
                {fileName}
              </div>
              <div style={{ fontSize: 13, color: "var(--text-muted)", marginBottom: 24 }}>
                {ext.toUpperCase()} · {formatBytes(artifact.size)}
              </div>
              <a href={downloadUrl} download className="btn-primary"
                style={{ display: "inline-flex", alignItems: "center", gap: 8, fontSize: 14, textDecoration: "none", padding: "10px 24px", borderRadius: 10 }}>
                <Download size={16} /> Download {ext.toUpperCase()}
              </a>
            </div>
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
  const [sideTab, setSideTab] = useState<SideTab>("steps");
  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set());
  const [previewArtifact, setPreviewArtifact] = useState<Artifact | null>(null);

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
      void vars;
      qc.invalidateQueries({ queryKey: ["tasks", taskId, "pending-inputs"] });
    },
  });

  const toggleStep = (id: string) =>
    setExpandedSteps((p) => { const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n; });

  // Auto-switch sidebar to artifacts when new artifact arrives
  useEffect(() => {
    if (artifacts.length > 0 && sideTab === "steps" && !isActive) {
      setSideTab("artifacts");
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
      <p style={{ color: "var(--red)", fontSize: 13 }}>Task not found.</p>
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
          type="button"
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
              fontSize: 14, fontWeight: 600, color: "var(--text-primary)",
              overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
              letterSpacing: "-0.02em",
            }}>
              {task.title}
            </span>
            <span className={TASK_BADGE[task.status] ?? "badge-gray"}>{task.status}</span>
            {isActive && (
              <span style={{ fontSize: 11.5, color: "var(--yellow)", display: "flex", alignItems: "center", gap: 4 }}>
                <Loader size={11} className="animate-spin" /> Working…
              </span>
            )}
            {pendingInputs.length > 0 && (
              <span style={{
                fontSize: 11, color: "#a78bfa",
                background: "rgba(124,58,237,0.12)",
                border: "1px solid rgba(124,58,237,0.3)",
                borderRadius: 20, padding: "2px 8px",
                display: "flex", alignItems: "center", gap: 4,
              }}>
                <HelpCircle size={10} />
                Waiting for input
              </span>
            )}
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 12, flexShrink: 0 }}>
          {totalSteps > 0 && (
            <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>
              {completedSteps}/{totalSteps} steps
            </span>
          )}
          {task.started_at && (
            <span style={{ fontSize: 11.5, color: "var(--text-faint)", display: "flex", alignItems: "center", gap: 4 }}>
              <Clock size={11} />
              {formatDuration(task.started_at, task.completed_at)}
            </span>
          )}
          {task.status === "pending" && (
            <button
              type="button"
              className="btn-primary btn-sm"
              disabled={runMutation.isPending}
              onClick={() => runMutation.mutate()}
            >
              {runMutation.isPending
                ? <><Loader size={12} className="animate-spin" />Starting…</>
                : <><Play size={12} />Run Task</>}
            </button>
          )}
        </div>
      </div>

      {/* ── Progress bar ── */}
      {totalSteps > 0 && (
        <div style={{ height: 2, background: "var(--surface-3)", flexShrink: 0 }}>
          <div style={{
            height: "100%", width: `${progress}%`,
            background: task.status === "completed" ? "var(--green)" :
              task.status === "failed" ? "var(--red)" : "var(--accent)",
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
              <span style={{ fontSize: 12.5, fontWeight: 600, color: "var(--yellow)" }}>Approval needed:</span>
              <span style={{ fontSize: 12.5, color: "var(--text-secondary)", flex: 1 }}>
                <code style={{ fontFamily: "monospace", background: "var(--surface-3)", padding: "1px 5px", borderRadius: 3, fontSize: 11 }}>{ap.tool_name}</code>
                {" "}— {ap.action}
              </span>
              <button type="button" className="btn-primary btn-sm" onClick={() => approveMutation.mutate({ id: ap.id, res: "approved" })}>Approve</button>
              <button type="button" className="btn-danger btn-sm" onClick={() => approveMutation.mutate({ id: ap.id, res: "rejected" })}>Reject</button>
            </div>
          ))}
        </div>
      )}

      {/* ── Body: Chat (main) + Sidebar ── */}
      <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>

        {/* ═══ MAIN AREA: Chat or Preview ═══ */}
        <div style={{
          flex: 1, minWidth: 0,
          display: "flex", flexDirection: "column",
          background: "var(--bg)",
          borderRight: "1px solid var(--border)",
        }}>
          {previewArtifact ? (
            <ContentPreviewPanel
              artifact={previewArtifact}
              onClose={() => setPreviewArtifact(null)}
            />
          ) : (
            <>
              {/* Chat header */}
              <div style={{
                display: "flex", alignItems: "center", gap: 8,
                padding: "0 20px", height: 42, flexShrink: 0,
                borderBottom: "1px solid var(--border)",
                background: "var(--surface-1)",
              }}>
                <MessageSquare size={13} color="var(--text-faint)" />
                <span style={{ fontSize: 12, fontWeight: 600, color: "var(--text-muted)", letterSpacing: "0.04em", textTransform: "uppercase" }}>
                  Conversation
                </span>
                {pendingInputs.length > 0 && (
                  <div style={{
                    width: 7, height: 7, borderRadius: "50%",
                    background: "#7c3aed",
                    boxShadow: "0 0 6px rgba(124,58,237,0.7)",
                    marginLeft: 2,
                    animation: "pulse 2s ease-in-out infinite",
                  }} />
                )}
              </div>
              <ChatPanel
                events={events}
                pendingInputs={pendingInputs}
                isActive={isActive}
                taskStatus={task.status}
                onRespond={(inputId, answer) => respondMutation.mutate({ inputId, answer })}
                respondPending={respondMutation.isPending}
              />
            </>
          )}
        </div>

        {/* ═══ RIGHT SIDEBAR ═══ */}
        <div style={{
          width: 340, flexShrink: 0,
          display: "flex", flexDirection: "column",
          background: "var(--surface-1)",
        }}>
          {/* Sidebar tabs */}
          <div style={{
            display: "flex", flexShrink: 0,
            borderBottom: "1px solid var(--border)",
            background: "var(--surface-2)",
          }}>
            {([
              { id: "steps" as SideTab,     icon: Activity,     label: "Steps",     count: totalSteps },
              { id: "artifacts" as SideTab, icon: FileText,     label: "Files",     count: artifacts.length },
              { id: "events" as SideTab,    icon: Terminal,     label: "Events",    count: events.length },
              { id: "agents" as SideTab,    icon: Users,        label: "Agents",    count: agents.length },
            ]).map(({ id, icon: Icon, label, count }) => (
              <button
                key={id}
                type="button"
                onClick={() => setSideTab(id)}
                style={{
                  flex: 1, display: "flex", flexDirection: "column", alignItems: "center",
                  gap: 3, padding: "9px 4px",
                  background: "transparent", border: "none", cursor: "pointer",
                  borderBottom: `2px solid ${sideTab === id ? "var(--accent)" : "transparent"}`,
                  color: sideTab === id ? "#a5b4fc" : "var(--text-faint)",
                  transition: "all 120ms", marginBottom: -1,
                  position: "relative",
                }}
              >
                <Icon size={13} />
                <span style={{ fontSize: 10, fontWeight: 600, letterSpacing: "0.04em" }}>
                  {label}
                </span>
                {count > 0 && (
                  <span style={{
                    position: "absolute", top: 5, right: 6,
                    fontSize: 9, fontWeight: 700, color: sideTab === id ? "#a5b4fc" : "var(--text-faint)",
                    background: "var(--surface-3)",
                    borderRadius: 10, padding: "0 4px", minWidth: 14, textAlign: "center",
                    lineHeight: "14px",
                  }}>
                    {count}
                  </span>
                )}
                {/* dot for artifacts new arrivals */}
                {id === "artifacts" && artifacts.length > 0 && sideTab !== "artifacts" && (
                  <div style={{
                    position: "absolute", top: 5, right: 4,
                    width: 6, height: 6, borderRadius: "50%",
                    background: "var(--green)",
                  }} />
                )}
              </button>
            ))}
          </div>

          {/* Sidebar content */}
          <div style={{ flex: 1, overflowY: "auto", padding: 14 }}>
            {sideTab === "steps" && (
              <>
                {plan && plan.steps.length > 0 ? (
                  <div>
                    <div style={{
                      fontSize: 10.5, fontWeight: 600, letterSpacing: "0.07em",
                      textTransform: "uppercase", color: "var(--text-faint)",
                      marginBottom: 14,
                    }}>
                      {plan.steps.length} step{plan.steps.length !== 1 ? "s" : ""} · {completedSteps} done
                    </div>
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
                ) : isActive ? (
                  <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 10, padding: "32px 0" }}>
                    <Loader size={20} className="animate-spin" style={{ color: "var(--accent)" }} />
                    <div style={{ fontSize: 12, color: "var(--text-muted)" }}>Planning…</div>
                  </div>
                ) : task.status === "pending" ? (
                  <div style={{ padding: "32px 0", textAlign: "center" }}>
                    <div style={{ fontSize: 12, color: "var(--text-faint)", marginBottom: 16 }}>
                      Steps will appear once the agent starts.
                    </div>
                  </div>
                ) : (
                  <div style={{ padding: "24px 0", textAlign: "center" }}>
                    <div style={{ fontSize: 12, color: "var(--text-faint)" }}>No steps recorded.</div>
                  </div>
                )}

                {/* Completion/failure banner */}
                {task.status === "completed" && (
                  <div style={{
                    marginTop: 16, padding: "10px 14px",
                    background: "rgba(52,211,153,0.06)",
                    border: "1px solid rgba(52,211,153,0.2)",
                    borderRadius: 9, display: "flex", alignItems: "center", gap: 8,
                  }}>
                    <CheckCircle size={14} color="var(--green)" />
                    <div>
                      <div style={{ fontSize: 12, fontWeight: 600, color: "var(--green)" }}>Completed</div>
                      <div style={{ fontSize: 11, color: "var(--text-muted)" }}>
                        {formatDuration(task.started_at, task.completed_at)} · {artifacts.length} artifact{artifacts.length !== 1 ? "s" : ""}
                      </div>
                    </div>
                  </div>
                )}
                {task.status === "failed" && (
                  <div style={{
                    marginTop: 16, padding: "10px 14px",
                    background: "rgba(248,113,113,0.06)",
                    border: "1px solid rgba(248,113,113,0.2)",
                    borderRadius: 9, display: "flex", alignItems: "center", gap: 8,
                  }}>
                    <XCircle size={14} color="var(--red)" />
                    <span style={{ fontSize: 12, fontWeight: 600, color: "var(--red)" }}>Run failed</span>
                  </div>
                )}
              </>
            )}

            {sideTab === "artifacts" && (
              <ArtifactsList
                artifacts={artifacts}
                onPreview={(a) => setPreviewArtifact(previewArtifact?.id === a.id ? null : a)}
                previewId={previewArtifact?.id}
              />
            )}

            {sideTab === "events" && <EventsList events={events} />}
            {sideTab === "agents" && <AgentsList agents={agents} />}
          </div>
        </div>
      </div>

      <style>{`
        @keyframes bounce {
          0%, 60%, 100% { transform: translateY(0); }
          30% { transform: translateY(-4px); }
        }
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.4; }
        }
      `}</style>
    </div>
  );
}
