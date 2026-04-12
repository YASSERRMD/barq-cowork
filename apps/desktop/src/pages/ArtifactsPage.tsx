import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  FileText, Braces, File, ScrollText, Package,
  Search, ChevronRight, Copy, ExternalLink,
  Clock, ArrowUpRight,
} from "lucide-react";
import { executionApi, type Artifact, type ArtifactType } from "../lib/api";
import { TopBar } from "../components/TopBar";
import { useAppStore } from "../store/appStore";

// ── Config ────────────────────────────────────────────────────────

const TYPE_ICON: Record<ArtifactType, React.ElementType> = {
  markdown: FileText,
  json:     Braces,
  file:     File,
  log:      ScrollText,
  html:     Package,
};

const TYPE_COLOR: Record<ArtifactType, string> = {
  markdown: "#60a5fa",
  json:     "#34d399",
  file:     "#a78bfa",
  log:      "#9090a8",
  html:     "#f59e0b",
};

const TYPE_BADGE: Record<ArtifactType, string> = {
  markdown: "badge-blue",
  json:     "badge-green",
  file:     "badge-purple",
  log:      "badge-gray",
  html:     "badge-yellow",
};

const ALL_TYPES: ArtifactType[] = ["markdown", "json", "file", "log", "html"];

function formatBytes(bytes: number): string {
  if (bytes <= 0) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return new Date(iso).toLocaleDateString();
}

// ── Artifact row ─────────────────────────────────────────────────

function ArtifactRow({
  artifact: a,
  selected,
  onSelect,
  navigate,
}: {
  artifact: Artifact;
  selected: boolean;
  onSelect: () => void;
  navigate: ReturnType<typeof useNavigate>;
}) {
  const Icon = TYPE_ICON[a.type] ?? Package;
  const color = TYPE_COLOR[a.type] ?? "var(--text-faint)";

  return (
    <div
      onClick={onSelect}
      style={{
        display: "flex", alignItems: "center", gap: 12,
        padding: "10px 20px", cursor: "pointer",
        background: selected ? "var(--surface-2)" : "transparent",
        borderBottom: "1px solid var(--border)",
        transition: "background 100ms",
      }}
      onMouseEnter={(e) => { if (!selected) (e.currentTarget as HTMLDivElement).style.background = "var(--surface-1)"; }}
      onMouseLeave={(e) => { if (!selected) (e.currentTarget as HTMLDivElement).style.background = "transparent"; }}
    >
      {/* Icon */}
      <div style={{
        width: 32, height: 32, borderRadius: 8, flexShrink: 0,
        background: `${color}15`, border: `1px solid ${color}25`,
        display: "flex", alignItems: "center", justifyContent: "center",
      }}>
        <Icon size={14} color={color} />
      </div>

      {/* Name + path */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          fontSize: 13, fontWeight: 500, color: "var(--text-primary)",
          overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
          letterSpacing: "-0.005em",
        }}>
          {a.name.split("/").pop() || a.name}
        </div>
        {a.content_path && (
          <div style={{
            fontSize: 11, color: "var(--text-faint)",
            overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
            fontFamily: "monospace", marginTop: 1,
          }}>
            {a.content_path}
          </div>
        )}
      </div>

      {/* Metadata */}
      <div style={{ display: "flex", alignItems: "center", gap: 10, flexShrink: 0 }}>
        <span className={TYPE_BADGE[a.type] ?? "badge-gray"} style={{ fontSize: 10 }}>{a.type}</span>
        <span style={{ fontSize: 11, color: "var(--text-faint)", minWidth: 48, textAlign: "right" }}>
          {formatBytes(a.size)}
        </span>
        <span style={{
          fontSize: 11, color: "var(--text-faint)", minWidth: 60,
          display: "flex", alignItems: "center", gap: 3,
        }}>
          <Clock size={10} /> {relativeTime(a.created_at)}
        </span>
        <button
          className="btn-ghost btn-xs"
          style={{ padding: "3px 5px" }}
          title="Open run"
          onClick={(e) => { e.stopPropagation(); navigate(`/tasks/${a.task_id}/run`); }}
        >
          <ArrowUpRight size={11} />
        </button>
      </div>
    </div>
  );
}

// ── Detail panel ─────────────────────────────────────────────────

function ArtifactDetail({ artifact: a, navigate }: { artifact: Artifact; navigate: ReturnType<typeof useNavigate> }) {
  const Icon = TYPE_ICON[a.type] ?? Package;
  const color = TYPE_COLOR[a.type] ?? "var(--text-faint)";

  return (
    <div style={{ padding: "20px", display: "flex", flexDirection: "column", gap: 20 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "flex-start", gap: 12 }}>
        <div style={{
          width: 42, height: 42, borderRadius: 11, flexShrink: 0,
          background: `${color}15`, border: `1px solid ${color}25`,
          display: "flex", alignItems: "center", justifyContent: "center",
        }}>
          <Icon size={20} color={color} />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{
            fontSize: 14, fontWeight: 600, color: "var(--text-primary)",
            letterSpacing: "-0.015em", wordBreak: "break-all",
          }}>
            {a.name.split("/").pop() || a.name}
          </div>
          <span className={TYPE_BADGE[a.type] ?? "badge-gray"} style={{ fontSize: 10, marginTop: 5, display: "inline-block" }}>
            {a.type}
          </span>
        </div>
      </div>

      {/* Metadata grid */}
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
        {[
          { label: "Size", value: formatBytes(a.size) },
          { label: "Created", value: new Date(a.created_at).toLocaleString() },
        ].map(({ label, value }) => (
          <div key={label} style={{
            padding: "8px 12px", background: "var(--surface-3)",
            border: "1px solid var(--border)", borderRadius: 7,
          }}>
            <div style={{ fontSize: 10, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase", marginBottom: 3 }}>
              {label}
            </div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>{value}</div>
          </div>
        ))}
      </div>

      {/* Path */}
      {a.content_path && (
        <div>
          <div className="section-label" style={{ marginBottom: 6 }}>File Path</div>
          <div style={{
            display: "flex", alignItems: "center", gap: 8,
            padding: "8px 10px", background: "var(--surface-2)",
            border: "1px solid var(--border)", borderRadius: 7,
          }}>
            <code style={{
              flex: 1, fontSize: 11, color: "var(--text-secondary)",
              fontFamily: "JetBrains Mono, monospace", wordBreak: "break-all",
            }}>
              {a.content_path}
            </code>
            <button
              className="btn-ghost btn-xs"
              onClick={() => navigator.clipboard.writeText(a.content_path)}
              title="Copy path"
            >
              <Copy size={11} />
            </button>
          </div>
        </div>
      )}

      {/* Inline preview */}
      {a.content_inline && (
        <div>
          <div className="section-label" style={{ marginBottom: 6 }}>Preview</div>
          <div style={{
            background: "var(--surface-1)", border: "1px solid var(--border)",
            borderRadius: 8, overflow: "hidden",
          }}>
            <pre style={{
              margin: 0, padding: "12px 14px",
              fontSize: 11.5, color: "var(--text-secondary)",
              fontFamily: "JetBrains Mono, monospace",
              whiteSpace: "pre-wrap", wordBreak: "break-all",
              lineHeight: 1.6, maxHeight: 240, overflowY: "auto",
              userSelect: "text",
            }} className="selectable">
              {a.content_inline.length > 600
                ? a.content_inline.slice(0, 600) + "\n…"
                : a.content_inline}
            </pre>
          </div>
        </div>
      )}

      {/* Actions */}
      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <button
          className="btn-secondary btn-sm"
          style={{ justifyContent: "flex-start", gap: 8 }}
          onClick={() => navigate(`/tasks/${a.task_id}/run`)}
        >
          <ExternalLink size={13} />
          Open source run
          <ChevronRight size={11} style={{ marginLeft: "auto" }} />
        </button>
        {a.content_path && (
          <button
            className="btn-secondary btn-sm"
            style={{ justifyContent: "flex-start", gap: 8 }}
            onClick={() => navigator.clipboard.writeText(a.content_path)}
          >
            <Copy size={13} />
            Copy file path
          </button>
        )}
      </div>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────

export function ArtifactsPage() {
  const { backendReachable } = useAppStore();
  const navigate = useNavigate();
  const [filterType, setFilterType] = useState<ArtifactType | "">("");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<string | null>(null);

  const { data: artifacts = [], isLoading, error } = useQuery({
    queryKey: ["artifacts", "recent"],
    queryFn: () => executionApi.listRecent(300),
    enabled: backendReachable,
    refetchInterval: 15_000,
  });

  const filtered = artifacts.filter((a) => {
    if (filterType && a.type !== filterType) return false;
    if (search) {
      const q = search.toLowerCase();
      if (!a.name.toLowerCase().includes(q) && !a.task_id.toLowerCase().includes(q)) return false;
    }
    return true;
  });

  const selectedArtifact = selected ? artifacts.find((a) => a.id === selected) : null;

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title="Artifacts"
        subtitle={`${filtered.length} of ${artifacts.length}`}
      />

      <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>
        {/* Main */}
        <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
          {/* Filter bar */}
          <div style={{
            padding: "10px 20px", borderBottom: "1px solid var(--border)",
            display: "flex", gap: 10, alignItems: "center",
          }}>
            <div style={{ position: "relative", flex: 1, maxWidth: 260 }}>
              <Search size={13} style={{
                position: "absolute", left: 9, top: "50%",
                transform: "translateY(-50%)", color: "var(--text-faint)", pointerEvents: "none",
              }} />
              <input
                className="input" placeholder="Search by name or task ID…"
                value={search} onChange={(e) => setSearch(e.target.value)}
                style={{ paddingLeft: 28, height: 30, fontSize: 12 }}
              />
            </div>

            {/* Type filters */}
            <div style={{ display: "flex", gap: 3 }}>
              <button onClick={() => setFilterType("")} style={{
                padding: "3px 10px", borderRadius: 5, fontSize: 12, fontWeight: 500,
                cursor: "pointer", border: "1px solid",
                background: filterType === "" ? "var(--accent-dim)" : "transparent",
                borderColor: filterType === "" ? "var(--accent)" : "transparent",
                color: filterType === "" ? "var(--accent)" : "var(--text-faint)",
                transition: "all 120ms",
              }}>
                All
              </button>
              {ALL_TYPES.map((t) => (
                <button key={t} onClick={() => setFilterType(t === filterType ? "" : t)} style={{
                  padding: "3px 10px", borderRadius: 5, fontSize: 12, fontWeight: 500,
                  cursor: "pointer", border: "1px solid", textTransform: "capitalize",
                  background: filterType === t ? "var(--accent-dim)" : "transparent",
                  borderColor: filterType === t ? "var(--accent)" : "transparent",
                  color: filterType === t ? "var(--accent)" : "var(--text-faint)",
                  transition: "all 120ms",
                }}>
                  {t}
                </button>
              ))}
            </div>
          </div>

          {/* Content */}
          <div style={{ flex: 1, overflowY: "auto" }}>
            {isLoading && (
              <div style={{ padding: "16px 20px", display: "flex", flexDirection: "column", gap: 8 }}>
                {[1,2,3,4].map(i => (
                  <div key={i} className="skeleton" style={{ height: 52, borderRadius: 8 }} />
                ))}
              </div>
            )}

            {error && (
              <div style={{ padding: 20 }}>
                <p style={{ color: "var(--red)", fontSize: 13 }}>Failed to load artifacts.</p>
              </div>
            )}

            {!isLoading && !error && filtered.length === 0 && (
              <div className="empty-state">
                <div className="empty-state-icon">
                  <Package size={20} color="var(--text-faint)" />
                </div>
                <p style={{ fontSize: 13, fontWeight: 500, color: "var(--text-secondary)", margin: 0 }}>
                  {search || filterType ? "No artifacts match your filter" : "No artifacts yet"}
                </p>
                <p style={{ fontSize: 12, color: "var(--text-muted)", margin: 0 }}>
                  Run a task that writes files to generate artifacts.
                </p>
              </div>
            )}

            {!isLoading && !error && filtered.length > 0 && (
              <>
                {/* Column headers */}
                <div style={{
                  display: "flex", alignItems: "center", gap: 12,
                  padding: "7px 20px", borderBottom: "1px solid var(--border)",
                  background: "var(--surface-1)",
                }}>
                  <div style={{ width: 32, flexShrink: 0 }} />
                  <div style={{ flex: 1, fontSize: 10.5, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase" }}>Name</div>
                  <div style={{ width: 70, fontSize: 10.5, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase" }}>Type</div>
                  <div style={{ width: 48, fontSize: 10.5, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase", textAlign: "right" }}>Size</div>
                  <div style={{ width: 60, fontSize: 10.5, fontWeight: 600, color: "var(--text-faint)", letterSpacing: "0.05em", textTransform: "uppercase" }}>Age</div>
                  <div style={{ width: 28 }} />
                </div>

                {filtered.map((a) => (
                  <ArtifactRow
                    key={a.id}
                    artifact={a}
                    selected={selected === a.id}
                    onSelect={() => setSelected(selected === a.id ? null : a.id)}
                    navigate={navigate}
                  />
                ))}
              </>
            )}
          </div>
        </div>

        {/* Detail panel */}
        {selectedArtifact && (
          <div className="right-panel">
            <ArtifactDetail artifact={selectedArtifact} navigate={navigate} />
          </div>
        )}
      </div>
    </div>
  );
}
