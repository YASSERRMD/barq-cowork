import { Search } from "lucide-react";

interface TopBarProps {
  title?: string;
  subtitle?: string;
  actions?: React.ReactNode;
  onSearch?: () => void;
}

export function TopBar({ title, subtitle, actions, onSearch }: TopBarProps) {
  return (
    <div
      style={{
        height: "var(--topbar-h)",
        borderBottom: "1px solid var(--border)",
        display: "flex",
        alignItems: "center",
        padding: "0 20px",
        gap: 12,
        flexShrink: 0,
        background: "var(--surface-1)",
        backdropFilter: "blur(12px)",
      }}
    >
      {/* Title area */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {title && (
          <div style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
            <span
              style={{
                fontSize: 13.5,
                fontWeight: 600,
                color: "var(--text-primary)",
                letterSpacing: "-0.015em",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
            >
              {title}
            </span>
            {subtitle && (
              <span style={{ fontSize: 12, color: "var(--text-muted)" }}>
                {subtitle}
              </span>
            )}
          </div>
        )}
      </div>

      {/* Search / command palette trigger */}
      <button
        onClick={onSearch}
        style={{
          display: "flex",
          alignItems: "center",
          gap: 7,
          background: "var(--surface-3)",
          border: "1px solid var(--border)",
          borderRadius: 7,
          padding: "5px 11px",
          color: "var(--text-muted)",
          fontSize: 12,
          cursor: "pointer",
          transition: "background 120ms, border-color 120ms, color 120ms",
          userSelect: "none",
          minWidth: 160,
        }}
        onMouseEnter={(e) => {
          const el = e.currentTarget as HTMLButtonElement;
          el.style.background = "var(--surface-4)";
          el.style.borderColor = "var(--border-mid)";
          el.style.color = "var(--text-secondary)";
        }}
        onMouseLeave={(e) => {
          const el = e.currentTarget as HTMLButtonElement;
          el.style.background = "var(--surface-3)";
          el.style.borderColor = "var(--border)";
          el.style.color = "var(--text-muted)";
        }}
      >
        <Search size={12} strokeWidth={2} />
        <span style={{ flex: 1 }}>Quick search…</span>
        <kbd
          style={{
            fontSize: 10,
            color: "var(--text-faint)",
            background: "var(--surface-2)",
            border: "1px solid var(--border)",
            borderRadius: 4,
            padding: "1px 5px",
            letterSpacing: "0.01em",
          }}
        >
          ⌘K
        </kbd>
      </button>

      {/* Custom actions */}
      {actions && (
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
          {actions}
        </div>
      )}
    </div>
  );
}
