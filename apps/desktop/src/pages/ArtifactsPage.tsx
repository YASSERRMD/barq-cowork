import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { FileText, Braces, File, ScrollText, Package } from "lucide-react";
import { executionApi, type Artifact, type ArtifactType } from "../lib/api";
import clsx from "clsx";

const TYPE_ICON: Record<ArtifactType, React.ElementType> = {
  markdown: FileText,
  json:     Braces,
  file:     File,
  log:      ScrollText,
};

const TYPE_BADGE: Record<ArtifactType, string> = {
  markdown: "badge-blue",
  json:     "badge-green",
  file:     "badge-gray",
  log:      "badge-gray",
};

const ALL_TYPES: ArtifactType[] = ["markdown", "json", "file", "log"];

export function ArtifactsPage() {
  const [filterType, setFilterType] = useState<ArtifactType | "">("");
  const [search, setSearch] = useState("");

  const { data: artifacts = [], isLoading, error, dataUpdatedAt } = useQuery({
    queryKey: ["artifacts", "recent"],
    queryFn: () => executionApi.listRecent(200),
    refetchInterval: 10_000,
  });

  const filtered = artifacts.filter((a) => {
    if (filterType && a.type !== filterType) return false;
    if (search) {
      const q = search.toLowerCase();
      if (!a.name.toLowerCase().includes(q) && !a.task_id.toLowerCase().includes(q)) {
        return false;
      }
    }
    return true;
  });

  return (
    <div className="p-6 space-y-5 max-w-4xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">Artifacts</h1>
          {dataUpdatedAt > 0 && (
            <p className="text-xs text-gray-500 mt-0.5">
              Updated {new Date(dataUpdatedAt).toLocaleTimeString()}
            </p>
          )}
        </div>
        <span className="text-xs text-gray-500">
          {filtered.length} of {artifacts.length}
        </span>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 flex-wrap">
        <input
          className="input max-w-xs text-sm"
          placeholder="Search by name or task ID…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <div className="flex items-center gap-1">
          <button
            className={clsx(
              "px-2.5 py-1 rounded text-xs transition-colors",
              filterType === ""
                ? "bg-barq-700 text-white"
                : "text-gray-400 hover:text-gray-200 hover:bg-gray-800"
            )}
            onClick={() => setFilterType("")}
          >
            All
          </button>
          {ALL_TYPES.map((t) => (
            <button
              key={t}
              className={clsx(
                "px-2.5 py-1 rounded text-xs transition-colors capitalize",
                filterType === t
                  ? "bg-barq-700 text-white"
                  : "text-gray-400 hover:text-gray-200 hover:bg-gray-800"
              )}
              onClick={() => setFilterType(t)}
            >
              {TYPE_ICON[t]} {t}
            </button>
          ))}
        </div>
      </div>

      {/* States */}
      {isLoading && <p className="text-gray-400 text-sm">Loading…</p>}
      {error && <p className="text-red-400 text-sm">Failed to load artifacts.</p>}

      {!isLoading && !error && filtered.length === 0 && (
        <div className="surface-2 rounded-lg border border-surface-3 p-8 text-center space-y-2">
          <Package size={24} className="mx-auto text-text-muted opacity-30" />
          <p className="text-sm text-text-secondary">No artifacts yet</p>
          <p className="text-xs text-text-muted">
            Run a task with write_file, write_markdown_report, or export_json to generate artifacts.
          </p>
        </div>
      )}

      {/* Table */}
      {filtered.length > 0 && (
        <div className="card overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-left">
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-400 uppercase tracking-wide w-8" />
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-400 uppercase tracking-wide">Name</th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-400 uppercase tracking-wide w-24">Type</th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-400 uppercase tracking-wide w-20">Size</th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-400 uppercase tracking-wide w-36">Task</th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-400 uppercase tracking-wide w-36">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800/60">
              {filtered.map((a) => (
                <ArtifactRow key={a.id} artifact={a} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function ArtifactRow({ artifact: a }: { artifact: Artifact }) {
  const Icon = TYPE_ICON[a.type] ?? Package;
  return (
    <tr className="hover:bg-gray-900/40 transition-colors">
      <td className="px-4 py-3 text-center">
        <Icon size={15} strokeWidth={1.75} className="mx-auto text-text-muted" />
      </td>
      <td className="px-4 py-3">
        <div className="min-w-0">
          <p className="text-white font-medium truncate max-w-xs" title={a.name}>
            {a.name.split("/").pop() || a.name}
          </p>
          {a.content_path && (
            <p className="text-gray-600 text-xs font-mono truncate max-w-xs" title={a.content_path}>
              {a.content_path}
            </p>
          )}
          {a.content_inline && (
            <p className="text-gray-500 text-xs truncate max-w-xs italic">
              {a.content_inline.slice(0, 60)}…
            </p>
          )}
        </div>
      </td>
      <td className="px-4 py-3">
        <span className={clsx("text-xs", TYPE_BADGE[a.type] ?? "badge-gray")}>
          {a.type}
        </span>
      </td>
      <td className="px-4 py-3 text-gray-400 text-xs font-mono">
        {formatBytes(a.size)}
      </td>
      <td className="px-4 py-3">
        <Link
          to={`/tasks/${a.task_id}/run`}
          className="text-barq-400 text-xs hover:text-barq-300 font-mono truncate block max-w-[120px]"
          title={a.task_id}
        >
          {a.task_id.slice(0, 8)}…
        </Link>
      </td>
      <td className="px-4 py-3 text-gray-500 text-xs">
        {new Date(a.created_at).toLocaleString()}
      </td>
    </tr>
  );
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
