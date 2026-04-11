import { useState } from "react";
import {
  Sparkles,
  FileText,
  FileSpreadsheet,
  Presentation,
  File,
  AlignLeft,
  Search,
  ChevronRight,
  Zap,
} from "lucide-react";
import { TopBar } from "../components/TopBar";

// ── Built-in skill definitions ─────────────────────────────────────
const BUILTIN_SKILLS = [
  {
    id: "docx",
    name: "Word Document",
    kind: "doc",
    icon: FileText,
    accentColor: "#60a5fa",
    description: "Create and transform Word documents — summaries, reports, business documents, and structured content.",
    outputExt: ".docx",
    inputTypes: ["text", "pdf", "md", "txt", "docx"],
    tags: ["document", "report", "word"],
    enabled: true,
    builtin: true,
    examples: ["Summarize this PDF as a DOCX report", "Write an executive summary from these notes"],
  },
  {
    id: "xlsx",
    name: "Spreadsheet",
    kind: "sheet",
    icon: FileSpreadsheet,
    accentColor: "#34d399",
    description: "Create and analyze Excel spreadsheets — tables, summaries, comparisons across multiple files.",
    outputExt: ".xlsx",
    inputTypes: ["csv", "xlsx", "txt", "json"],
    tags: ["spreadsheet", "excel", "data"],
    enabled: true,
    builtin: true,
    examples: ["Extract all invoices and produce an XLSX summary", "Build a comparison table from these CSVs"],
  },
  {
    id: "pptx",
    name: "Presentation",
    kind: "deck",
    icon: Presentation,
    accentColor: "#f97316",
    description: "Generate PowerPoint slide decks from documents, notes, or outlines with structured content.",
    outputExt: ".pptx",
    inputTypes: ["txt", "md", "docx", "pdf"],
    tags: ["slides", "powerpoint", "deck"],
    enabled: true,
    builtin: true,
    examples: ["Convert this brief into a 10-slide deck", "Create a pitch deck from this outline"],
  },
  {
    id: "pdf",
    name: "PDF",
    kind: "pdf",
    icon: File,
    accentColor: "#f87171",
    description: "Summarize, extract, compare, and generate PDF documents. Extract tables, text, and structured data.",
    outputExt: ".pdf",
    inputTypes: ["pdf", "docx", "md", "txt"],
    tags: ["pdf", "extract", "summarize"],
    enabled: true,
    builtin: true,
    examples: ["Summarize these 3 PDFs into one brief", "Compare contracts and list key differences"],
  },
  {
    id: "text",
    name: "Text & Markdown",
    kind: "text",
    icon: AlignLeft,
    accentColor: "#a78bfa",
    description: "Process and generate plain text, Markdown, CSV, and JSON content for any workflow.",
    outputExt: ".md / .txt / .csv / .json",
    inputTypes: ["txt", "md", "csv", "json"],
    tags: ["text", "markdown", "csv", "json"],
    enabled: true,
    builtin: true,
    examples: ["Clean and reformat these Markdown files", "Convert this JSON log to a readable report"],
  },
];

type KindFilter = "all" | "doc" | "sheet" | "deck" | "pdf" | "text";

const KIND_LABELS: Record<KindFilter, string> = {
  all: "All",
  doc: "Document",
  sheet: "Spreadsheet",
  deck: "Presentation",
  pdf: "PDF",
  text: "Text",
};

function SkillCard({
  skill,
  selected,
  onSelect,
}: {
  skill: typeof BUILTIN_SKILLS[0];
  selected: boolean;
  onSelect: () => void;
}) {
  const Icon = skill.icon;
  return (
    <div
      className={selected ? "card" : "card-hover"}
      onClick={onSelect}
      style={{
        padding: "16px",
        cursor: "pointer",
        borderColor: selected ? "var(--accent)" : undefined,
        boxShadow: selected ? "0 0 0 1px var(--accent), 0 4px 16px var(--accent-glow)" : undefined,
        transition: "all 150ms",
      }}
    >
      <div style={{ display: "flex", alignItems: "flex-start", gap: 12 }}>
        <div
          style={{
            width: 36,
            height: 36,
            borderRadius: 9,
            background: `${skill.accentColor}18`,
            border: `1px solid ${skill.accentColor}30`,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
          }}
        >
          <Icon size={17} color={skill.accentColor} />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 7, marginBottom: 4 }}>
            <span
              style={{
                fontSize: 13.5,
                fontWeight: 600,
                color: "var(--text-primary)",
                letterSpacing: "-0.01em",
              }}
            >
              {skill.name}
            </span>
            {skill.builtin && (
              <span className="badge-accent" style={{ fontSize: 10 }}>Built-in</span>
            )}
            {skill.enabled ? (
              <span className="badge-green" style={{ fontSize: 10 }}>Active</span>
            ) : (
              <span className="badge-gray" style={{ fontSize: 10 }}>Disabled</span>
            )}
          </div>
          <p
            style={{
              fontSize: 12,
              color: "var(--text-secondary)",
              margin: 0,
              lineHeight: 1.5,
            }}
          >
            {skill.description}
          </p>
          <div style={{ display: "flex", gap: 5, marginTop: 8, flexWrap: "wrap" }}>
            <span
              style={{
                fontSize: 11,
                color: "var(--text-faint)",
                background: "var(--surface-4)",
                borderRadius: 4,
                padding: "1px 6px",
                fontFamily: "monospace",
              }}
            >
              → {skill.outputExt}
            </span>
            {skill.tags.map((tag) => (
              <span
                key={tag}
                style={{
                  fontSize: 11,
                  color: "var(--text-faint)",
                  background: "var(--surface-3)",
                  borderRadius: 4,
                  padding: "1px 6px",
                }}
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function SkillDetailPanel({ skill }: { skill: typeof BUILTIN_SKILLS[0] }) {
  const Icon = skill.icon;
  return (
    <div style={{ padding: "20px", display: "flex", flexDirection: "column", gap: 20 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <div
          style={{
            width: 42,
            height: 42,
            borderRadius: 11,
            background: `${skill.accentColor}18`,
            border: `1px solid ${skill.accentColor}30`,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
          }}
        >
          <Icon size={20} color={skill.accentColor} />
        </div>
        <div>
          <div style={{ fontSize: 15, fontWeight: 600, color: "var(--text-primary)", letterSpacing: "-0.015em" }}>
            {skill.name}
          </div>
          <div style={{ fontSize: 11.5, color: "var(--text-muted)", marginTop: 1 }}>
            {skill.kind} skill · built-in
          </div>
        </div>
      </div>

      {/* Description */}
      <p style={{ fontSize: 12.5, color: "var(--text-secondary)", margin: 0, lineHeight: 1.6 }}>
        {skill.description}
      </p>

      {/* Output */}
      <div>
        <div className="section-label" style={{ marginBottom: 6 }}>Output Format</div>
        <span
          style={{
            fontSize: 12,
            color: "var(--text-secondary)",
            background: "var(--surface-3)",
            borderRadius: 5,
            padding: "4px 10px",
            fontFamily: "monospace",
          }}
        >
          {skill.outputExt}
        </span>
      </div>

      {/* Accepted inputs */}
      <div>
        <div className="section-label" style={{ marginBottom: 6 }}>Accepts</div>
        <div style={{ display: "flex", gap: 5, flexWrap: "wrap" }}>
          {skill.inputTypes.map((t) => (
            <span
              key={t}
              style={{
                fontSize: 11,
                color: "var(--text-muted)",
                background: "var(--surface-3)",
                border: "1px solid var(--border)",
                borderRadius: 4,
                padding: "2px 8px",
                fontFamily: "monospace",
              }}
            >
              .{t}
            </span>
          ))}
        </div>
      </div>

      {/* Example prompts */}
      <div>
        <div className="section-label" style={{ marginBottom: 8 }}>Example Prompts</div>
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          {skill.examples.map((ex) => (
            <div
              key={ex}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                padding: "8px 10px",
                background: "var(--surface-3)",
                border: "1px solid var(--border)",
                borderRadius: 7,
                cursor: "pointer",
              }}
            >
              <Zap size={12} color="var(--accent)" style={{ flexShrink: 0 }} />
              <span style={{ fontSize: 12, color: "var(--text-secondary)", flex: 1 }}>{ex}</span>
              <ChevronRight size={12} color="var(--text-faint)" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export function SkillsPage() {
  const [kindFilter, setKindFilter] = useState<KindFilter>("all");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<string | null>(null);

  const filtered = BUILTIN_SKILLS.filter((s) => {
    const matchesKind = kindFilter === "all" || s.kind === kindFilter;
    const matchesSearch =
      !search ||
      s.name.toLowerCase().includes(search.toLowerCase()) ||
      s.description.toLowerCase().includes(search.toLowerCase()) ||
      s.tags.some((t) => t.includes(search.toLowerCase()));
    return matchesKind && matchesSearch;
  });

  const selectedSkill = selected ? BUILTIN_SKILLS.find((s) => s.id === selected) : null;

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title="Skills"
        subtitle={`${BUILTIN_SKILLS.length} built-in skills`}
      />

      <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>
        {/* Main content */}
        <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
          {/* Filters */}
          <div
            style={{
              padding: "10px 20px",
              borderBottom: "1px solid var(--border)",
              display: "flex",
              gap: 12,
              alignItems: "center",
            }}
          >
            {/* Search */}
            <div style={{ position: "relative", flex: 1, maxWidth: 260 }}>
              <Search
                size={13}
                style={{
                  position: "absolute",
                  left: 9,
                  top: "50%",
                  transform: "translateY(-50%)",
                  color: "var(--text-faint)",
                  pointerEvents: "none",
                }}
              />
              <input
                className="input"
                placeholder="Search skills…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                style={{ paddingLeft: 28, height: 30, fontSize: 12 }}
              />
            </div>

            {/* Kind filter pills */}
            <div style={{ display: "flex", gap: 4 }}>
              {(Object.keys(KIND_LABELS) as KindFilter[]).map((k) => (
                <button
                  key={k}
                  onClick={() => setKindFilter(k)}
                  style={{
                    padding: "3px 10px",
                    borderRadius: 5,
                    fontSize: 12,
                    fontWeight: 500,
                    cursor: "pointer",
                    border: "1px solid",
                    background: kindFilter === k ? "var(--accent-dim)" : "transparent",
                    borderColor: kindFilter === k ? "rgba(99,102,241,0.3)" : "transparent",
                    color: kindFilter === k ? "#a5b4fc" : "var(--text-faint)",
                    transition: "all 120ms",
                  }}
                >
                  {KIND_LABELS[k]}
                </button>
              ))}
            </div>
          </div>

          {/* Skill grid */}
          <div
            style={{
              flex: 1,
              overflowY: "auto",
              padding: "16px 20px",
            }}
          >
            {/* Built-ins section */}
            <div
              className="section-label"
              style={{ marginBottom: 10, paddingLeft: 2 }}
            >
              Built-in Skills — {filtered.length}
            </div>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
              {filtered.map((skill) => (
                <SkillCard
                  key={skill.id}
                  skill={skill}
                  selected={selected === skill.id}
                  onSelect={() =>
                    setSelected(selected === skill.id ? null : skill.id)
                  }
                />
              ))}
            </div>

            {filtered.length === 0 && (
              <div className="empty-state">
                <div className="empty-state-icon">
                  <Sparkles size={20} color="var(--text-faint)" />
                </div>
                <p style={{ fontSize: 13, color: "var(--text-muted)" }}>No skills match your filter.</p>
              </div>
            )}

            {/* Custom skills placeholder */}
            <div style={{ marginTop: 24 }}>
              <div className="section-label" style={{ marginBottom: 10, paddingLeft: 2 }}>
                Custom Skills — 0
              </div>
              <div
                className="card"
                style={{
                  padding: "20px",
                  textAlign: "center",
                  borderStyle: "dashed",
                }}
              >
                <p style={{ fontSize: 13, color: "var(--text-muted)", margin: 0 }}>
                  Custom skills can be added via the <code style={{ fontSize: 11, background: "var(--surface-3)", padding: "1px 5px", borderRadius: 3 }}>skills/</code> directory.
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Right panel — skill detail */}
        {selectedSkill && (
          <div className="right-panel">
            <SkillDetailPanel skill={selectedSkill} />
          </div>
        )}
      </div>
    </div>
  );
}
