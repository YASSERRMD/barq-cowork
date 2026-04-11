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
        borderBottom: "1px solid #2a2a3a",
        display: "flex",
        alignItems: "center",
        padding: "0 20px",
        gap: 12,
        flexShrink: 0,
        background: "#111118",
      }}
    >
      {/* Title area */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {title && (
          <div style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
            <span style={{ fontSize: 14, fontWeight: 600, color: "#e2e2e8", letterSpacing: "-0.01em" }}>
              {title}
            </span>
            {subtitle && (
              <span style={{ fontSize: 12, color: "#50505f" }}>{subtitle}</span>
            )}
          </div>
        )}
      </div>

      {/* Search */}
      <button
        onClick={onSearch}
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          background: "#1c1c27",
          border: "1px solid #2a2a3a",
          borderRadius: 6,
          padding: "5px 10px",
          color: "#50505f",
          fontSize: 12,
          cursor: "pointer",
          transition: "all 150ms",
        }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLButtonElement).style.borderColor = "#3a3a4e";
          (e.currentTarget as HTMLButtonElement).style.color = "#7a7a90";
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLButtonElement).style.borderColor = "#2a2a3a";
          (e.currentTarget as HTMLButtonElement).style.color = "#50505f";
        }}
      >
        <Search size={13} />
        <span>Search</span>
        <kbd style={{ fontSize: 10, opacity: 0.6, marginLeft: 4 }}>⌘K</kbd>
      </button>

      {/* Custom actions */}
      {actions && <div style={{ display: "flex", alignItems: "center", gap: 8 }}>{actions}</div>}
    </div>
  );
}
