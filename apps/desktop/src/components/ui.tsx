import type { ElementType, ReactNode } from "react";

// ── EmptyState ────────────────────────────────────────────────────

interface EmptyStateProps {
  icon: ElementType;
  title: string;
  description?: string;
  action?: ReactNode;
}

export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-state-icon">
        <Icon size={20} color="var(--text-faint)" />
      </div>
      <p style={{ fontSize: 13, fontWeight: 500, color: "var(--text-secondary)", margin: 0 }}>
        {title}
      </p>
      {description && (
        <p style={{ fontSize: 12, color: "var(--text-muted)", margin: 0 }}>
          {description}
        </p>
      )}
      {action && <div style={{ marginTop: 4 }}>{action}</div>}
    </div>
  );
}

// ── SkeletonCard ─────────────────────────────────────────────────

export function SkeletonCard() {
  return (
    <div className="skeleton" style={{ height: 52, borderRadius: 8 }} />
  );
}

// ── Skeleton (generic, width/height via style prop) ───────────────

export function Skeleton({ style }: { style?: React.CSSProperties }) {
  return <div className="skeleton" style={style} />;
}
