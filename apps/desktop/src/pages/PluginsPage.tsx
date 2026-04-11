import { Puzzle } from "lucide-react";
import { TopBar } from "../components/TopBar";

// Full implementation comes in Phase 6. This is a minimal scaffold.
export function PluginsPage() {
  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar title="Plugins" subtitle="Bundled skills, connectors, and sub-agents" />
      <div className="empty-state" style={{ flex: 1 }}>
        <div className="empty-state-icon">
          <Puzzle size={20} color="var(--accent)" />
        </div>
        <p className="page-title" style={{ fontSize: 15 }}>Plugins coming in Phase 6</p>
        <p className="page-subtitle">Plugin bundles combining skills, connectors, and slash commands will appear here.</p>
      </div>
    </div>
  );
}
