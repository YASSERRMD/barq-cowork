import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Search,
  FolderOpen,
  Play,
  Calendar,
  Settings,
  FileText,
  CheckCircle2,
  Activity,
  Plug,
  ChevronRight,
  Command,
} from "lucide-react";
import { projectsApi, type Project } from "../lib/api";

// ─────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────

interface PaletteItem {
  id: string;
  label: string;
  sublabel?: string;
  icon: React.ElementType;
  shortcut?: string[];
  action: () => void;
  group: string;
}

// ─────────────────────────────────────────────
// CommandPalette
// ─────────────────────────────────────────────

export function CommandPalette({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Data for dynamic items
  const { data: projects = [] } = useQuery({
    queryKey: ["projects"],
    queryFn: projectsApi.list,
    staleTime: 30_000,
    retry: 1,
  });

  const go = useCallback(
    (path: string) => {
      navigate(path);
      onClose();
    },
    [navigate, onClose]
  );

  // ── Static navigation items ────────────────────────────
  const staticItems: PaletteItem[] = [
    {
      id: "nav-projects",
      label: "Go to Projects",
      icon: FolderOpen,
      shortcut: ["P"],
      action: () => go("/projects"),
      group: "Navigation",
    },
    {
      id: "nav-runs",
      label: "Go to Runs",
      icon: Play,
      shortcut: ["R"],
      action: () => go("/runs"),
      group: "Navigation",
    },
    {
      id: "nav-schedules",
      label: "Go to Schedules",
      icon: Calendar,
      shortcut: ["S"],
      action: () => go("/schedules"),
      group: "Navigation",
    },
    {
      id: "nav-approvals",
      label: "Go to Approvals",
      icon: CheckCircle2,
      action: () => go("/approvals"),
      group: "Navigation",
    },
    {
      id: "nav-artifacts",
      label: "Go to Artifacts",
      icon: FileText,
      action: () => go("/artifacts"),
      group: "Navigation",
    },
    {
      id: "nav-connectors",
      label: "Go to Connectors",
      icon: Plug,
      action: () => go("/connectors"),
      group: "Navigation",
    },
    {
      id: "nav-logs",
      label: "Go to Logs",
      icon: Activity,
      action: () => go("/logs"),
      group: "Navigation",
    },
    {
      id: "nav-settings",
      label: "Go to Settings",
      icon: Settings,
      shortcut: [","],
      action: () => go("/settings"),
      group: "Navigation",
    },
  ];

  // ── Project items ──────────────────────────────────────
  const projectItems: PaletteItem[] = projects.map((p: Project) => ({
    id: `project-${p.id}`,
    label: p.name,
    sublabel: p.description || "Open project",
    icon: FolderOpen,
    action: () => go(`/projects/${p.id}`),
    group: "Projects",
  }));

  // ── All items ─────────────────────────────────────────
  const allItems = [...staticItems, ...projectItems];

  // ── Filtered ─────────────────────────────────────────
  const filtered = query.trim()
    ? allItems.filter(
        (item) =>
          item.label.toLowerCase().includes(query.toLowerCase()) ||
          item.sublabel?.toLowerCase().includes(query.toLowerCase()) ||
          item.group.toLowerCase().includes(query.toLowerCase())
      )
    : staticItems; // no query → show navigation only

  // Group the filtered items
  const groups = filtered.reduce<Record<string, PaletteItem[]>>((acc, item) => {
    if (!acc[item.group]) acc[item.group] = [];
    acc[item.group].push(item);
    return acc;
  }, {});

  const flatFiltered = Object.values(groups).flat();

  // Keep selectedIndex in bounds
  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // Scroll selected item into view
  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-index="${selectedIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  // ── Keyboard navigation ───────────────────────────────
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      onClose();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((i) => Math.min(i + 1, flatFiltered.length - 1));
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((i) => Math.max(i - 1, 0));
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      flatFiltered[selectedIndex]?.action();
      return;
    }
  };

  // Close on backdrop click
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
        position: "fixed",
        inset: 0,
        zIndex: 200,
        display: "flex",
        alignItems: "flex-start",
        justifyContent: "center",
        paddingTop: "15vh",
        background: "rgba(0,0,0,0.55)",
        backdropFilter: "blur(4px)",
      }}
      onClick={handleBackdrop}
    >
      <div
        style={{
          width: "100%",
          maxWidth: 560,
          background: "#1a1a26",
          border: "1px solid #2e2e42",
          borderRadius: 12,
          overflow: "hidden",
          boxShadow: "0 24px 80px rgba(0,0,0,0.6)",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search input */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 10,
            padding: "12px 16px",
            borderBottom: "1px solid #2e2e42",
          }}
        >
          <Search size={16} style={{ color: "#6b6b80", flexShrink: 0 }} />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search projects, navigate, run actions…"
            style={{
              flex: 1,
              background: "transparent",
              border: "none",
              outline: "none",
              fontSize: 14,
              color: "#e2e2f0",
              fontFamily: "inherit",
            }}
            autoFocus
          />
          <kbd
            style={{
              fontSize: 10,
              color: "#40404f",
              background: "#252535",
              border: "1px solid #3a3a50",
              borderRadius: 4,
              padding: "2px 5px",
              fontFamily: "inherit",
              flexShrink: 0,
            }}
          >
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div
          ref={listRef}
          style={{ maxHeight: 380, overflowY: "auto", padding: "6px 0" }}
        >
          {flatFiltered.length === 0 && (
            <div
              style={{
                padding: "24px 16px",
                textAlign: "center",
                fontSize: 13,
                color: "#50505f",
              }}
            >
              No results for "{query}"
            </div>
          )}

          {Object.entries(groups).map(([groupName, items]) => (
            <div key={groupName}>
              <div
                style={{
                  padding: "6px 16px 2px",
                  fontSize: 10,
                  fontWeight: 600,
                  letterSpacing: "0.07em",
                  textTransform: "uppercase",
                  color: "#40404f",
                }}
              >
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
                      display: "flex",
                      alignItems: "center",
                      gap: 10,
                      width: "100%",
                      padding: "7px 16px",
                      background: isSelected ? "#252535" : "transparent",
                      border: "none",
                      cursor: "pointer",
                      textAlign: "left",
                      transition: "background 0.1s",
                    }}
                  >
                    <item.icon
                      size={15}
                      strokeWidth={1.75}
                      style={{
                        color: isSelected ? "#7c78ff" : "#505065",
                        flexShrink: 0,
                      }}
                    />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <span
                        style={{
                          fontSize: 13,
                          color: isSelected ? "#e2e2f0" : "#ababc0",
                          display: "block",
                        }}
                      >
                        {item.label}
                      </span>
                      {item.sublabel && (
                        <span
                          style={{
                            fontSize: 11,
                            color: "#50505f",
                            display: "block",
                            overflow: "hidden",
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                          }}
                        >
                          {item.sublabel}
                        </span>
                      )}
                    </div>
                    {item.shortcut && item.shortcut.length > 0 && (
                      <div
                        style={{ display: "flex", gap: 3, flexShrink: 0 }}
                      >
                        {item.shortcut.map((k) => (
                          <kbd
                            key={k}
                            style={{
                              fontSize: 10,
                              color: "#50505f",
                              background: "#252535",
                              border: "1px solid #3a3a50",
                              borderRadius: 4,
                              padding: "1px 5px",
                              fontFamily: "inherit",
                            }}
                          >
                            {k}
                          </kbd>
                        ))}
                      </div>
                    )}
                    {isSelected && (
                      <ChevronRight
                        size={13}
                        style={{ color: "#7c78ff", flexShrink: 0 }}
                      />
                    )}
                  </button>
                );
              })}
            </div>
          ))}
        </div>

        {/* Footer */}
        <div
          style={{
            borderTop: "1px solid #2e2e42",
            padding: "8px 16px",
            display: "flex",
            alignItems: "center",
            gap: 16,
          }}
        >
          {[
            { keys: ["↑", "↓"], label: "navigate" },
            { keys: ["↵"], label: "select" },
            { keys: ["ESC"], label: "close" },
          ].map(({ keys, label }) => (
            <div
              key={label}
              style={{ display: "flex", alignItems: "center", gap: 4 }}
            >
              {keys.map((k) => (
                <kbd
                  key={k}
                  style={{
                    fontSize: 10,
                    color: "#50505f",
                    background: "#252535",
                    border: "1px solid #3a3a50",
                    borderRadius: 4,
                    padding: "1px 5px",
                    fontFamily: "inherit",
                  }}
                >
                  {k}
                </kbd>
              ))}
              <span style={{ fontSize: 11, color: "#40404f" }}>{label}</span>
            </div>
          ))}
          <div
            style={{
              marginLeft: "auto",
              display: "flex",
              alignItems: "center",
              gap: 4,
            }}
          >
            <Command size={11} style={{ color: "#40404f" }} />
            <span style={{ fontSize: 11, color: "#40404f" }}>K</span>
          </div>
        </div>
      </div>
    </div>
  );
}
