import { useState, useRef, useEffect } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Play, Pause, ArrowDown } from "lucide-react";
import { eventsApi, type TaskEvent } from "../lib/api";
import { TopBar } from "../components/TopBar";

// ── Event colour/label maps ─────────────────────────────────────

const EVENT_COLOR: Record<string, string> = {
  "task.created":    "var(--text-muted)",
  "task.started":    "var(--blue)",
  "task.completed":  "var(--green)",
  "task.failed":     "var(--red)",
  "step.started":    "var(--yellow)",
  "step.completed":  "#86efac",
  "tool.called":     "#a5b4fc",
  "tool.result":     "#c7d2fe",
  "approval.needed": "#fb923c",
  "artifact.ready":  "var(--purple)",
  "log.line":        "var(--text-muted)",
};

const EVENT_LABEL: Record<string, string> = {
  "task.created":    "TASK CREATED",
  "task.started":    "TASK STARTED",
  "task.completed":  "TASK DONE   ",
  "task.failed":     "TASK FAILED ",
  "step.started":    "STEP START  ",
  "step.completed":  "STEP DONE   ",
  "tool.called":     "TOOL CALL   ",
  "tool.result":     "TOOL RESULT ",
  "approval.needed": "APPROVAL    ",
  "artifact.ready":  "ARTIFACT    ",
  "log.line":        "LOG         ",
};

const EVENT_CATEGORIES = [
  { label: "All",      value: "" },
  { label: "Tasks",    value: "task." },
  { label: "Steps",    value: "step." },
  { label: "Tools",    value: "tool." },
  { label: "Approval", value: "approval." },
  { label: "Artifact", value: "artifact." },
];

// ── Page ─────────────────────────────────────────────────────────

export function LogsPage() {
  const [autoScroll, setAutoScroll] = useState(true);
  const [filterType, setFilterType] = useState("");
  const [paused, setPaused] = useState(false);
  const logRef = useRef<HTMLDivElement>(null);

  const { data: events = [], isLoading, error, dataUpdatedAt } = useQuery({
    queryKey: ["events", "recent"],
    queryFn: () => eventsApi.listRecent(500),
    refetchInterval: paused ? false : 3_000,
  });

  const displayed = filterType
    ? events.filter((e) => e.type.startsWith(filterType))
    : events;

  useEffect(() => {
    if (autoScroll && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [displayed.length, autoScroll]);

  const handleScroll = () => {
    const el = logRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    setAutoScroll(atBottom);
  };

  const scrollToBottom = () => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
      setAutoScroll(true);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title="Event Log"
        subtitle={dataUpdatedAt > 0 ? `${displayed.length} events` : undefined}
        actions={
          <button
            className={paused ? "btn-primary btn-sm" : "btn-secondary btn-sm"}
            style={{ gap: 5 }}
            onClick={() => setPaused((v) => !v)}
          >
            {paused ? <><Play size={12} /> Resume</> : <><Pause size={12} /> Pause</>}
          </button>
        }
      />

      {/* Filter bar */}
      <div style={{
        padding: "8px 20px",
        borderBottom: "1px solid var(--border)",
        display: "flex", gap: 3, alignItems: "center", flexShrink: 0,
      }}>
        {EVENT_CATEGORIES.map((cat) => (
          <button
            key={cat.value}
            onClick={() => setFilterType(cat.value)}
            style={{
              padding: "3px 10px", borderRadius: 5, fontSize: 12, fontWeight: 500,
              cursor: "pointer", border: "1px solid",
              background: filterType === cat.value ? "var(--accent-dim)" : "transparent",
              borderColor: filterType === cat.value ? "rgba(99,102,241,0.3)" : "transparent",
              color: filterType === cat.value ? "#a5b4fc" : "var(--text-faint)",
              transition: "all 120ms",
            }}
          >
            {cat.label}
          </button>
        ))}
        <span style={{ marginLeft: "auto", fontSize: 11, color: "var(--text-faint)" }}>
          {paused ? "Paused" : "Live · 3s"}
        </span>
      </div>

      {/* Log terminal */}
      <div
        ref={logRef}
        onScroll={handleScroll}
        style={{
          flex: 1, overflowY: "auto",
          background: "var(--bg)",
          fontFamily: "JetBrains Mono, monospace",
          fontSize: 12, lineHeight: 1.7,
          minHeight: 0,
        }}
      >
        {isLoading && (
          <p style={{ padding: "16px 20px", color: "var(--text-faint)" }}>
            Loading events…
          </p>
        )}
        {error && (
          <p style={{ padding: "16px 20px", color: "var(--red)" }}>
            Failed to load events.
          </p>
        )}
        {!isLoading && !error && displayed.length === 0 && (
          <p style={{ padding: "16px 20px", color: "var(--text-faint)" }}>
            No events yet. Run a task to see live execution logs.
          </p>
        )}
        {[...displayed].reverse().map((ev) => (
          <EventLine key={ev.id} event={ev} />
        ))}
      </div>

      {/* Scroll-to-bottom nudge */}
      {!autoScroll && (
        <button
          onClick={scrollToBottom}
          style={{
            flexShrink: 0,
            display: "flex", alignItems: "center", justifyContent: "center", gap: 5,
            padding: "6px 20px",
            background: "var(--surface-3)", borderTop: "1px solid var(--border)",
            color: "var(--text-muted)", fontSize: 12, cursor: "pointer",
            border: "none", transition: "color 120ms",
          }}
          onMouseEnter={(e) => (e.currentTarget.style.color = "var(--text-primary)")}
          onMouseLeave={(e) => (e.currentTarget.style.color = "var(--text-muted)")}
        >
          <ArrowDown size={12} /> New events below
        </button>
      )}
    </div>
  );
}

function EventLine({ event: ev }: { event: TaskEvent }) {
  let payload: string;
  try {
    const parsed = JSON.parse(ev.payload);
    payload = JSON.stringify(parsed);
  } catch {
    payload = ev.payload || "{}";
  }

  const color = EVENT_COLOR[ev.type] ?? "var(--text-muted)";
  const label = (EVENT_LABEL[ev.type] ?? ev.type.padEnd(12)).slice(0, 12);
  const time = new Date(ev.created_at).toLocaleTimeString("en-GB", {
    hour: "2-digit", minute: "2-digit", second: "2-digit",
  });

  return (
    <div style={{
      display: "flex", alignItems: "flex-start", gap: 16,
      padding: "1px 20px",
      transition: "background 80ms",
    }}
    onMouseEnter={(e) => ((e.currentTarget as HTMLDivElement).style.background = "var(--surface-1)")}
    onMouseLeave={(e) => ((e.currentTarget as HTMLDivElement).style.background = "transparent")}
    >
      <span style={{ color: "var(--text-faint)", flexShrink: 0, width: 72 }}>{time}</span>
      <span style={{ color, flexShrink: 0, width: 100 }}>{label}</span>
      <Link
        to={`/tasks/${ev.task_id}/run`}
        style={{
          color: "var(--text-faint)", flexShrink: 0, width: 72,
          overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
          textDecoration: "none", transition: "color 100ms",
        }}
        title={ev.task_id}
        onMouseEnter={(e) => (e.currentTarget.style.color = "#a5b4fc")}
        onMouseLeave={(e) => (e.currentTarget.style.color = "var(--text-faint)")}
      >
        {ev.task_id.slice(0, 8)}
      </Link>
      <span style={{ color: "var(--text-muted)", wordBreak: "break-all", flex: 1 }}>
        {payload}
      </span>
    </div>
  );
}
