import { Home } from "lucide-react";

// Full implementation in Phase 2. Stub to satisfy routing.
export function HomePage() {
  return (
    <div className="empty-state" style={{ flex: 1, height: "100%" }}>
      <div className="empty-state-icon">
        <Home size={20} color="var(--accent)" />
      </div>
      <p className="page-title" style={{ fontSize: 15 }}>Home — Phase 2</p>
    </div>
  );
}
