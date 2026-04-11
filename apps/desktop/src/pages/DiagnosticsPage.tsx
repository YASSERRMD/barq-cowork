import { useQuery } from "@tanstack/react-query";
import { diagnosticsApi, eventsApi } from "../lib/api";

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

function StatCard({
  label,
  value,
  sub,
}: {
  label: string;
  value: string | number;
  sub?: string;
}) {
  return (
    <div className="bg-zinc-800 rounded-lg p-4 flex flex-col gap-1">
      <span className="text-xs text-zinc-400 uppercase tracking-wider">{label}</span>
      <span className="text-xl font-semibold text-zinc-100">{value}</span>
      {sub && <span className="text-xs text-zinc-500">{sub}</span>}
    </div>
  );
}

const EVENT_COLORS: Record<string, string> = {
  step_started:    "text-blue-400",
  step_completed:  "text-emerald-400",
  step_failed:     "text-red-400",
  tool_call:       "text-amber-400",
  tool_result:     "text-amber-300",
  approval_needed: "text-orange-400",
  artifact_created:"text-purple-400",
};

// ─────────────────────────────────────────────────────────────────────────────
// Page
// ─────────────────────────────────────────────────────────────────────────────

export default function DiagnosticsPage() {
  const infoQuery = useQuery({
    queryKey: ["diagnostics", "info"],
    queryFn: diagnosticsApi.getInfo,
    refetchInterval: 10_000,
    retry: false,
  });

  const eventsQuery = useQuery({
    queryKey: ["events", "recent", 100],
    queryFn: () => eventsApi.listRecent(100),
    refetchInterval: 5_000,
  });

  const info = infoQuery.data;
  const reachable = !infoQuery.isError;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Diagnostics</h1>
          <p className="text-sm text-zinc-400 mt-1">
            Runtime status and log export for barq-coworkd
          </p>
        </div>

        <div className="flex items-center gap-3">
          {/* Live status dot */}
          <span className="flex items-center gap-2 text-sm">
            <span
              className={`w-2 h-2 rounded-full ${
                reachable ? "bg-emerald-400 animate-pulse" : "bg-red-500"
              }`}
            />
            <span className={reachable ? "text-emerald-400" : "text-red-400"}>
              {reachable ? "Backend reachable" : "Backend unreachable"}
            </span>
          </span>

          {/* Download bundle */}
          <a
            href={diagnosticsApi.bundleUrl()}
            download
            className="px-4 py-2 rounded-lg bg-orange-600 hover:bg-orange-500 text-white text-sm font-medium transition-colors"
          >
            ↓ Download Bundle
          </a>
        </div>
      </div>

      {/* System stats grid */}
      {info ? (
        <section>
          <h2 className="text-sm font-semibold text-zinc-400 uppercase tracking-wider mb-3">
            System Info
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
            <StatCard label="Version" value={info.version} />
            <StatCard label="Go runtime" value={info.go_version} />
            <StatCard label="Platform" value={`${info.os}/${info.arch}`} />
            <StatCard label="CPUs" value={info.num_cpu} />
            <StatCard
              label="Goroutines"
              value={info.num_goroutine}
              sub="active goroutines"
            />
            <StatCard
              label="Heap (current)"
              value={`${info.mem_alloc_mb.toFixed(1)} MB`}
            />
            <StatCard
              label="Heap (total)"
              value={`${info.mem_total_alloc_mb.toFixed(1)} MB`}
              sub="since startup"
            />
            <StatCard
              label="Snapshot"
              value={new Date(info.generated_at).toLocaleTimeString()}
              sub="auto-refreshes every 10s"
            />
          </div>

          {info.build_info && Object.keys(info.build_info).length > 0 && (
            <div className="mt-3 bg-zinc-800/60 rounded-lg p-4">
              <p className="text-xs font-semibold text-zinc-400 mb-2">Build metadata</p>
              <dl className="grid grid-cols-2 gap-x-6 gap-y-1 text-xs">
                {Object.entries(info.build_info).map(([k, v]) => (
                  <div key={k} className="flex gap-2">
                    <dt className="text-zinc-500 shrink-0">{k}</dt>
                    <dd className="text-zinc-300 font-mono truncate">{v}</dd>
                  </div>
                ))}
              </dl>
            </div>
          )}
        </section>
      ) : infoQuery.isError ? (
        <div className="bg-red-900/30 border border-red-800 rounded-lg p-4 text-red-300 text-sm">
          Cannot reach backend. Start barq-coworkd and refresh.
        </div>
      ) : (
        <div className="text-zinc-500 text-sm">Loading system info…</div>
      )}

      {/* Recent events */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">
            Recent Events{" "}
            <span className="normal-case font-normal text-zinc-500">
              (last 100, live)
            </span>
          </h2>
          <span className="text-xs text-zinc-600">
            {eventsQuery.data?.length ?? 0} events
          </span>
        </div>

        <div className="bg-zinc-900 rounded-lg border border-zinc-800 overflow-hidden">
          {eventsQuery.data && eventsQuery.data.length > 0 ? (
            <div className="max-h-96 overflow-y-auto">
              <table className="w-full text-xs">
                <thead className="sticky top-0 bg-zinc-800 border-b border-zinc-700">
                  <tr>
                    <th className="px-3 py-2 text-left text-zinc-400 font-medium">Time</th>
                    <th className="px-3 py-2 text-left text-zinc-400 font-medium">Type</th>
                    <th className="px-3 py-2 text-left text-zinc-400 font-medium">Task ID</th>
                    <th className="px-3 py-2 text-left text-zinc-400 font-medium">Payload</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/50">
                  {[...(eventsQuery.data ?? [])].reverse().map((ev) => (
                    <tr key={ev.id} className="hover:bg-zinc-800/40">
                      <td className="px-3 py-1.5 text-zinc-500 whitespace-nowrap font-mono">
                        {new Date(ev.created_at).toLocaleTimeString()}
                      </td>
                      <td className="px-3 py-1.5 whitespace-nowrap">
                        <span
                          className={
                            EVENT_COLORS[ev.type] ?? "text-zinc-300"
                          }
                        >
                          {ev.type}
                        </span>
                      </td>
                      <td className="px-3 py-1.5 font-mono text-zinc-500 whitespace-nowrap">
                        {ev.task_id.slice(0, 8)}…
                      </td>
                      <td className="px-3 py-1.5 text-zinc-400 truncate max-w-xs">
                        {ev.payload}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-zinc-500 text-sm p-6 text-center">
              No events yet. Run a task to see activity here.
            </p>
          )}
        </div>
      </section>
    </div>
  );
}
