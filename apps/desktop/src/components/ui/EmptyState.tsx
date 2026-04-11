import { LucideIcon } from "lucide-react";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
}

export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        padding: "48px 24px",
        textAlign: "center",
        gap: 16,
      }}
    >
      <div
        style={{
          width: 48,
          height: 48,
          borderRadius: 12,
          background: "#1c1c27",
          border: "1px solid #2a2a3a",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <Icon size={22} color="#40404f" strokeWidth={1.5} />
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 6, maxWidth: 320 }}>
        <span style={{ fontSize: 14, fontWeight: 600, color: "#c4c4d0" }}>{title}</span>
        {description && (
          <span style={{ fontSize: 13, color: "#50505f", lineHeight: 1.5 }}>{description}</span>
        )}
      </div>
      {action && <div>{action}</div>}
    </div>
  );
}
