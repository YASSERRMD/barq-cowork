import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Search,
  Home,
  FolderOpen,
  Play,
  Calendar,
  Settings,
  FileText,
  CheckCircle2,
  Activity,
  Plug,
  Wand2,
  Puzzle,
  ChevronRight,
  Command,
  Clock,
} from "lucide-react";
import { projectsApi, tasksApi, type Project, type Task } from "../lib/api";

// ── Types ─────────────────────────────────────────────────────────

interface PaletteItem {
  id: string;
  label: string;
  sublabel?: string;
  icon: React.ElementType;
  iconColor?: string;
  shortcut?: string[];
  action: () => void;
  group: string;
}

// ── Helpers ───────────────────────────────────────────────────────

const STATUS_COLOR: Record<string, string> = {
  completed: "var(--green)",
  running:   "var(--accent)",
  planning:  "var(--yellow)",
  failed:    "var(--red)",
  pending:   "var(--text-faint)",
};

function taskStatusLabel(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

// ── CommandPalette ────────────────────────────────────────────────

export function CommandPalette({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const { data: projects = [] } = useQuery({
    queryKey: ["projects"],
    queryFn: projectsApi.list,
    staleTime: 30_000,
    retry: 1,
  });

  const { data: recentTasks = [] } = useQuery({
    queryKey: ["tasks", "recent-palette"],
    queryFn: () => tasksApi.listAll(8),
    staleTime: 10_000,
    retry: 1,
  });

  const go = useCallback(
    (path: string) => { navigate(path); onClose(); },
    [navigate, onClose]
  );

  // ── Static nav items ─────────────────────────────────────────
  const staticItems: PaletteItem[] = [
    { id: "nav-home",       label: "Home",           icon: Home,         shortcut: ["H"], action: () => go("/"),            group: "Navigation" },
    { id: "nav-runs",       label: "Runs",           icon: Play,         shortcut: ["R"], action: () => go("/runs"),        group: "Navigation" },
    { id: "nav-skills",     label: "Skills",         icon: Wand2,                         action: () => go("/skills"),     group: "Navigation" },
    { id: "nav-connectors", label: "Connectors",     icon: Plug,                          action: () => go("/connectors"), group: "Navigation" },
    { id: "nav-plugins",    label: "Plugins",        icon: Puzzle,                        action: () => go("/plugins"),    group: "Navigation" },
    { id: "nav-artifacts",  label: "Artifacts",      icon: FileText,                      action: () => go("/artifacts"),  group: "Navigation" },
    { id: "nav-projects",   label: "Projects",       icon: FolderOpen,   shortcut: ["P"], action: () => go("/projects"),   group: "Navigation" },
    { id: "nav-schedules",  label: "Schedules",      icon: Calendar,     shortcut: ["S"], action: () => go("/schedules"),  group: "Navigation" },
    { id: "nav-approvals",  label: "Approvals",      icon: CheckCircle2,                  action: () => go("/approvals"),  group: "Navigation" },
    { id: "nav-logs",       label: "Logs",           icon: Activity,                      action: () => go("/logs"),       group: "Navigation" },
    { id: "nav-settings",   label: "Settings",       icon: Settings,     shortcut: [","], action: () => go("/settings"),   group: "Navigation" },
  ];

  // ── Recent task items ─────────────────────────────────────────
  const recentItems: PaletteItem[] = (recentTasks as Task[]).map((t) => ({
    id: `task-${t.id}`,
    label: t.title || t.description?.slice(0, 60) || t.id,
    sublabel: `${taskStatusLabel(t.status)} · ${new Date(t.created_at).toLocaleDateString()}`,
    icon: Clock,
    iconColor: STATUS_COLOR[t.status] ?? "var(--text-faint)",
    action: () => go(`/tasks/${t.id}/run`),
    group: "Recent Runs",
  }));

  // ── Project items ─────────────────────────────────────────────
  const projectItems: PaletteItem[] = (projects as Project[]).map((p) => ({
    id: `project-${p.id}`,
    label: p.name,
    sublabel: p.description || "Open project",
    icon: FolderOpen,
    action: () => go(`/projects/${p.id}/tasks`),
    group: "Projects",
  }));

  // ── Filter / group ────────────────────────────────────────────
  const allItems = [...staticItems, ...recentItems, ...projectItems];

  const filtered = query.trim()
    ? allItems.filter(
        (item) =>
          item.label.toLowerCase().includes(query.toLowerCase()) ||
          item.sublabel?.toLowerCase().includes(query.toLowerCase()) ||
          item.group.toLowerCase().includes(query.toLowerCase())
      )
    : [...staticItems, ...recentItems.slice(0, 5)];

  const groups = filtered.reduce<Record<string, PaletteItem[]>>((acc, item) => {
    if (!acc[item.group]) acc[item.group] = [];
    acc[item.group].push(item);
    return acc;
  }, {});

  const flatFiltered = Object.values(groups).flat();

  useEffect(() => { setSelectedIndex(0); }, [query]);
  useEffect(() => { inputRef.current?.focus(); }, []);
  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-index="${selectedIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape")     { onClose(); return; }
    if (e.key === "ArrowDown")  { e.preventDefault(); setSelectedIndex((i) => Math.min(i + 1, flatFiltered.length - 1)); return; }
    if (e.key === "ArrowUp")    { e.preventDefault(); setSelectedIndex((i) => Math.max(i - 1, 0)); return; }
    if (e.key === "Enter")      { e.preventDefault(); flatFiltered[selectedIndex]?.action(); return; }
  };

  const handleBackdrop = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) onClose();
  };

  let itemIdx = 0;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
      style={{
        position: "fixed", inset: 0, zIndex: 200,
        display: "flex", alignItems: "flex-start", justifyContent: "center",
        paddingTop: "14vh",
        background: "rgba(0,0,0,0.6)",
        backdropFilter: "blur(6px)",
      }}
      onClick={handleBackdrop}
    >
      <div
        style={{
          width: "100%", maxWidth: 560,
          background: "var(--surface-3)",
          border: "1px solid var(--border-mid)",
          borderRadius: 12,
          overflow: "hidden",
          boxShadow: "0 24px 80px rgba(0,0,0,0.65), 0 0 0 1px rgba(255,255,255,0.04)",
          animation: "page-fade-in 140ms var(--ease-out)",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Input */}
        <div style={{
          display: "flex", alignItems: "center", gap: 10,
          padding: "11px 16px",
          borderBottom: "1px solid var(--border)",
        }}>
          <Search size={16} style={{ color: "var(--text-muted)", flexShrink: 0 }} />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search pages, runs, projects…"
            style={{
              flex: 1, background: "transparent",
              border: "none", outline: "none",
              fontSize: 14, color: "var(--text-primary)",
              fontFamily: "inherit",
            }}
            autoFocus
          />
          <kbd style={{
            fontSize: 10, color: "var(--text-faint)",
            background: "var(--surface-4)",
            border: "1px solid var(--border-mid)",
            borderRadius: 4, padding: "2px 6px",
            fontFamily: "inherit", flexShrink: 0,
          }}>
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div ref={listRef} style={{ maxHeight: 380, overflowY: "auto", padding: "6px 0" }}>
          {flatFiltered.length === 0 && (
            <div style={{
              padding: "24px 16px", textAlign: "center",
              fontSize: 13, color: "var(--text-muted)",
            }}>
              No results for "{query}"
            </div>
          )}

          {Object.entries(groups).map(([groupName, items]) => (
            <div key={groupName}>
              <div style={{
                padding: "6px 16px 2px",
                fontSize: 10, fontWeight: 600,
                letterSpacing: "0.07em", textTransform: "uppercase",
                color: "var(--text-faint)",
              }}>
                {groupName}
              </div>
              {items.map((item) => {
                const idx = itemIdx++;
                const isSelected = idx === selectedIndex;
                return (
                  <button
                    key={item.id}
                    data-index={idx}
                    onClick={item.action}
                    onMouseEnter={() => setSelectedIndex(idx)}
                    style={{
                      display: "flex", alignItems: "center", gap: 10,
                      width: "100%", padding: "7px 16px",
                      background: isSelected ? "var(--surface-4)" : "transparent",
                      border: "none", cursor: "pointer", textAlign: "left",
                      transition: "background 80ms",
                    }}
                  >
                    <item.icon
                      size={15}
                      strokeWidth={1.75}
                      style={{
                        color: isSelected
                          ? (item.iconColor ?? "#a5b4fc")
                          : (item.iconColor ?? "var(--text-muted)"),
                        flexShrink: 0,
                      }}
                    />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <span style={{
                        fontSize: 13,
                        color: isSelected ? "var(--text-primary)" : "var(--text-secondary)",
                        display: "block",
                      }}>
                        {item.label}
                      </span>
                      {item.sublabel && (
                        <span style={{
                          fontSize: 11, color: "var(--text-faint)",
                          display: "block",
                          overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                        }}>
                          {item.sublabel}
                        </span>
                      )}
                    </div>
                    {item.shortcut && item.shortcut.length > 0 && !isSelected && (
                      <div style={{ display: "flex", gap: 3, flexShrink: 0 }}>
                        {item.shortcut.map((k) => (
                          <kbd key={k} style={{
                            fontSize: 10, color: "var(--text-faint)",
                            background: "var(--surface-4)",
                            border: "1px solid var(--border-mid)",
                            borderRadius: 4, padding: "1px 5px",
                            fontFamily: "inherit",
                          }}>
                            {k}
                          </kbd>
                        ))}
                      </div>
                    )}
                    {isSelected && (
                      <ChevronRight size={13} style={{ color: "#a5b4fc", flexShrink: 0 }} />
                    )}
                  </button>
                );
              })}
            </div>
          ))}
        </div>

        {/* Footer */}
        <div style={{
          borderTop: "1px solid var(--border)",
          padding: "7px 16px",
          display: "flex", alignItems: "center", gap: 14,
        }}>
          {[
            { keys: ["↑", "↓"], label: "navigate" },
            { keys: ["↵"], label: "select" },
            { keys: ["ESC"], label: "close" },
          ].map(({ keys, label }) => (
            <div key={label} style={{ display: "flex", alignItems: "center", gap: 4 }}>
              {keys.map((k) => (
                <kbd key={k} style={{
                  fontSize: 10, color: "var(--text-faint)",
                  background: "var(--surface-4)",
                  border: "1px solid var(--border-mid)",
                  borderRadius: 4, padding: "1px 5px",
                  fontFamily: "inherit",
                }}>
                  {k}
                </kbd>
              ))}
              <span style={{ fontSize: 11, color: "var(--text-faint)" }}>{label}</span>
            </div>
          ))}
          <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 4 }}>
            <Command size={11} style={{ color: "var(--text-faint)" }} />
            <span style={{ fontSize: 11, color: "var(--text-faint)" }}>K</span>
          </div>
        </div>
      </div>
    </div>
  );
}
