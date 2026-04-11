import { Sparkles } from "lucide-react";
import { TopBar } from "../components/TopBar";

// Full implementation comes in Phase 4. This is a minimal scaffold.
export function SkillsPage() {
  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar title="Skills" subtitle="Document and file processing capabilities" />
      <div className="empty-state" style={{ flex: 1 }}>
        <div className="empty-state-icon">
          <Sparkles size={20} color="var(--accent)" />
        </div>
        <p className="page-title" style={{ fontSize: 15 }}>Skills coming in Phase 4</p>
        <p className="page-subtitle">DOCX, XLSX, PPTX, PDF, and text skills will appear here.</p>
      </div>
    </div>
  );
}
