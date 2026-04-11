import { useState, useRef, useEffect } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { eventsApi, type TaskEvent } from "../lib/api";
import clsx from "clsx";

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

// Derive a short label for the event type so the log reads naturally.
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

export function LogsPage() {
  const [autoScroll, setAutoScroll] = useState(true);
  const [filterType, setFilterType] = useState("");
  const [paused, setPaused] = useState(false);
  const logRef = useRef<HTMLDivElement>(null);

  const { data: events = [], isLoading, error, dataUpdatedAt } = useQuery({
    queryKey: ["events", "recent"],
    queryFn: () => eventsApi.listRecent(500),
    refetchInterval: paused ? false : 3000,
  });

  const displayed = filterType
    ? events.filter((e) => e.type.startsWith(filterType))
    : events;

  // Auto-scroll to bottom when new events arrive.
  useEffect(() => {
    if (autoScroll && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [displayed.length, autoScroll]);

  // Detect manual scroll up to pause auto-scroll.
  const handleScroll = () => {
    const el = logRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    setAutoScroll(atBottom);
  };

  const EVENT_CATEGORIES = [
    { label: "All",      value: "" },
    { label: "Tasks",    value: "task." },
    { label: "Steps",    value: "step." },
    { label: "Tools",    value: "tool." },
    { label: "Approval", value: "approval." },
    { label: "Artifact", value: "artifact." },
  ];

  return (
    <div className="p-6 h-full flex flex-col gap-4" style={{ maxHeight: "calc(100vh - 1rem)" }}>
      {/* Header */}
      <div className="flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-xl font-semibold text-white">Event Log</h1>
          {dataUpdatedAt > 0 && (
            <p className="text-xs text-gray-500 mt-0.5">
              {displayed.length} events · refreshes every 3s
              {paused && " (paused)"}
            </p>
          )}
        </div>

        <div className="flex items-center gap-2">
          <button
            className={clsx(
              "px-3 py-1.5 rounded text-xs font-medium transition-colors",
              paused
                ? "bg-green-700 hover:bg-green-600 text-white"
                : "bg-gray-800 hover:bg-gray-700 text-gray-300"
            )}
            onClick={() => setPaused((v) => !v)}
          >
            {paused ? "▶ Resume" : "⏸ Pause"}
          </button>
          <button
            className="btn-ghost text-xs"
            onClick={() => {
              if (logRef.current) {
                logRef.current.scrollTop = logRef.current.scrollHeight;
                setAutoScroll(true);
              }
            }}
          >
            ↓ Bottom
          </button>
        </div>
      </div>

      {/* Filter tabs */}
      <div className="flex items-center gap-1 flex-wrap shrink-0">
        {EVENT_CATEGORIES.map((cat) => (
          <button
            key={cat.value}
            className={clsx(
              "px-2.5 py-1 rounded text-xs transition-colors",
              filterType === cat.value
                ? "bg-barq-700 text-white"
                : "text-gray-400 hover:text-gray-200 hover:bg-gray-800"
            )}
            onClick={() => setFilterType(cat.value)}
          >
            {cat.label}
          </button>
        ))}
      </div>

      {/* Log terminal */}
      <div
        ref={logRef}
        onScroll={handleScroll}
        className="flex-1 card bg-gray-950 rounded-lg overflow-y-auto font-mono text-xs min-h-0"
      >
        {isLoading && (
          <p className="p-4 text-gray-500">[barq-cowork] Loading events…</p>
        )}
        {error && (
          <p className="p-4 text-red-500">[barq-cowork] Failed to load events.</p>
        )}
        {!isLoading && !error && displayed.length === 0 && (
          <p className="p-4 text-gray-600">
            [barq-cowork] No events yet. Run a task to see live execution logs.
          </p>
        )}

        {/* Events rendered oldest-first (backend returns newest-first, so reverse) */}
        {[...displayed].reverse().map((ev) => (
          <EventLine key={ev.id} event={ev} />
        ))}
      </div>

      {/* Auto-scroll indicator */}
      {!autoScroll && (
        <div
          className="shrink-0 text-center text-xs text-gray-500 cursor-pointer hover:text-gray-300"
          onClick={() => {
            if (logRef.current) {
              logRef.current.scrollTop = logRef.current.scrollHeight;
              setAutoScroll(true);
            }
          }}
        >
          ↓ New events below — click to scroll
        </div>
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

  const color = EVENT_COLOR[ev.type] ?? "text-gray-400";
  const label = (EVENT_LABEL[ev.type] ?? ev.type.padEnd(12)).slice(0, 12);
  const time = new Date(ev.created_at).toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });

  return (
    <div className="flex items-start gap-3 px-4 py-1 hover:bg-gray-900/40 transition-colors group">
      <span className="text-gray-600 shrink-0 w-20">{time}</span>
      <span className={clsx("shrink-0 w-28", color)}>{label}</span>
      <Link
        to={`/tasks/${ev.task_id}/run`}
        className="text-gray-600 hover:text-barq-400 font-mono shrink-0 w-20 truncate"
        title={ev.task_id}
      >
        {ev.task_id.slice(0, 8)}
      </Link>
      <span className="text-gray-500 break-all">{payload}</span>
    </div>
  );
}
