import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
  ToggleLeft,
  ToggleRight,
} from "lucide-react";
import { TopBar } from "../components/TopBar";
import { skillsApi, type Skill, type SkillKind } from "../lib/api";
import { useAppStore } from "../store/appStore";

// ── Icon map ──────────────────────────────────────────────────────
const KIND_ICON: Record<SkillKind, React.ElementType> = {
  doc:   FileText,
  sheet: FileSpreadsheet,
  deck:  Presentation,
  pdf:   File,
  text:  AlignLeft,
};

const KIND_COLOR: Record<SkillKind, string> = {
  doc:   "#60a5fa",
  sheet: "#34d399",
  deck:  "#f97316",
  pdf:   "#f87171",
  text:  "#a78bfa",
};

// Example prompts per kind (static, no DB needed)
const KIND_EXAMPLES: Record<SkillKind, string[]> = {
  doc:   ["Summarize this PDF as a DOCX report", "Write an executive summary from these notes"],
  sheet: ["Extract all invoices and produce an XLSX summary", "Build a comparison table from these CSVs"],
  deck:  ["Convert this brief into a 10-slide deck", "Create a pitch deck from this outline"],
  pdf:   ["Summarize these 3 PDFs into one brief", "Compare contracts and list key differences"],
  text:  ["Clean and reformat these Markdown files", "Convert this JSON log to a readable report"],
};

const KIND_LABELS: Record<SkillKind | "all", string> = {
  all:   "All",
  doc:   "Document",
  sheet: "Spreadsheet",
  deck:  "Presentation",
  pdf:   "PDF",
  text:  "Text",
};

// ── Skill card ────────────────────────────────────────────────────
function SkillCard({
  skill, selected, onSelect,
}: {
  skill: Skill; selected: boolean; onSelect: () => void;
}) {
  const Icon = KIND_ICON[skill.kind] ?? Sparkles;
  const color = KIND_COLOR[skill.kind] ?? "var(--accent)";

  return (
    <div
      className={selected ? "card" : "card-hover"}
      onClick={onSelect}
      style={{
        padding: "14px 16px", cursor: "pointer",
        borderColor: selected ? "var(--accent)" : undefined,
        boxShadow: selected ? "0 0 0 1px var(--accent), 0 4px 16px var(--accent-glow)" : undefined,
        opacity: skill.enabled ? 1 : 0.55,
        transition: "all 150ms",
      }}
    >
      <div style={{ display: "flex", alignItems: "flex-start", gap: 12 }}>
        <div style={{
          width: 34, height: 34, borderRadius: 9, flexShrink: 0,
          background: `${color}18`, border: `1px solid ${color}30`,
          display: "flex", alignItems: "center", justifyContent: "center",
        }}>
          <Icon size={16} color={color} />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 7, marginBottom: 4, flexWrap: "wrap" }}>
            <span style={{ fontSize: 13.5, fontWeight: 600, color: "var(--text-primary)", letterSpacing: "-0.01em" }}>
              {skill.name}
            </span>
            {skill.built_in && <span className="badge-accent" style={{ fontSize: 10 }}>Built-in</span>}
            <span className={skill.enabled ? "badge-green" : "badge-gray"} style={{ fontSize: 10 }}>
              {skill.enabled ? "Active" : "Disabled"}
            </span>
          </div>
          <p style={{ fontSize: 12, color: "var(--text-secondary)", margin: 0, lineHeight: 1.5 }}>
            {skill.description}
          </p>
          <div style={{ display: "flex", gap: 5, marginTop: 8, flexWrap: "wrap" }}>
            <span style={{
              fontSize: 11, color: "var(--text-faint)", background: "var(--surface-3)",
              borderRadius: 4, padding: "1px 6px", fontFamily: "monospace",
            }}>
              → {skill.output_file_ext || ".out"}
            </span>
            {skill.tags.slice(0, 3).map((t) => (
              <span key={t} style={{
                fontSize: 11, color: "var(--text-faint)", background: "var(--surface-3)", borderRadius: 4, padding: "1px 6px",
              }}>
                {t}
              </span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Detail panel ──────────────────────────────────────────────────
function SkillDetail({ skill, onToggle }: { skill: Skill; onToggle: () => void }) {
  const Icon = KIND_ICON[skill.kind] ?? Sparkles;
  const color = KIND_COLOR[skill.kind] ?? "var(--accent)";
  const examples = KIND_EXAMPLES[skill.kind] ?? [];

  return (
    <div style={{ padding: "20px", display: "flex", flexDirection: "column", gap: 20 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <div style={{
          width: 42, height: 42, borderRadius: 11, flexShrink: 0,
          background: `${color}18`, border: `1px solid ${color}30`,
          display: "flex", alignItems: "center", justifyContent: "center",
        }}>
          <Icon size={20} color={color} />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 15, fontWeight: 600, color: "var(--text-primary)", letterSpacing: "-0.015em" }}>
            {skill.name}
          </div>
          <div style={{ fontSize: 11.5, color: "var(--text-muted)", marginTop: 1 }}>
            {skill.kind} skill · {skill.built_in ? "built-in" : "custom"}
          </div>
        </div>
        <button
          className={skill.enabled ? "btn-secondary btn-sm" : "btn-primary btn-sm"}
          onClick={onToggle}
          style={{ flexShrink: 0, gap: 5 }}
        >
          {skill.enabled ? <ToggleRight size={13} /> : <ToggleLeft size={13} />}
          {skill.enabled ? "Enabled" : "Disabled"}
        </button>
      </div>

      <p style={{ fontSize: 12.5, color: "var(--text-secondary)", margin: 0, lineHeight: 1.6 }}>
        {skill.description}
      </p>

      {/* Output */}
      <div>
        <div className="section-label" style={{ marginBottom: 6 }}>Output Format</div>
        <span style={{
          fontSize: 12, color: "var(--text-secondary)", background: "var(--surface-3)",
          borderRadius: 5, padding: "4px 10px", fontFamily: "monospace",
        }}>
          {skill.output_file_ext || ".out"} — {skill.output_mime_type || "unknown"}
        </span>
      </div>

      {/* Accepted inputs */}
      {skill.input_mime_types.length > 0 && (
        <div>
          <div className="section-label" style={{ marginBottom: 6 }}>Accepts</div>
          <div style={{ display: "flex", gap: 5, flexWrap: "wrap" }}>
            {skill.input_mime_types.map((t) => (
              <span key={t} style={{
                fontSize: 11, color: "var(--text-muted)", background: "var(--surface-3)",
                border: "1px solid var(--border)", borderRadius: 4,
                padding: "2px 8px", fontFamily: "monospace",
              }}>
                {t}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Tags */}
      {skill.tags.length > 0 && (
        <div>
          <div className="section-label" style={{ marginBottom: 6 }}>Tags</div>
          <div style={{ display: "flex", gap: 5, flexWrap: "wrap" }}>
            {skill.tags.map((t) => (
              <span key={t} style={{
                fontSize: 11, color: "var(--text-faint)", background: "var(--surface-3)", borderRadius: 4, padding: "1px 6px",
              }}>
                {t}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Example prompts */}
      {examples.length > 0 && (
        <div>
          <div className="section-label" style={{ marginBottom: 8 }}>Example Prompts</div>
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            {examples.map((ex) => (
              <div key={ex} style={{
                display: "flex", alignItems: "center", gap: 8,
                padding: "8px 10px", background: "var(--surface-3)",
                border: "1px solid var(--border)", borderRadius: 7, cursor: "pointer",
              }}>
                <Zap size={12} color="var(--accent)" style={{ flexShrink: 0 }} />
                <span style={{ fontSize: 12, color: "var(--text-secondary)", flex: 1 }}>{ex}</span>
                <ChevronRight size={12} color="var(--text-faint)" />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────
export function SkillsPage() {
  const { backendReachable } = useAppStore();
  const qc = useQueryClient();
  const [kindFilter, setKindFilter] = useState<SkillKind | "all">("all");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<string | null>(null);

  const { data: skills = [], isLoading } = useQuery({
    queryKey: ["skills"],
    queryFn: skillsApi.list,
    enabled: backendReachable,
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      skillsApi.updateEnabled(id, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["skills"] }),
  });

  const filtered = skills.filter((s) => {
    const matchKind = kindFilter === "all" || s.kind === kindFilter;
    const matchSearch =
      !search ||
      s.name.toLowerCase().includes(search.toLowerCase()) ||
      s.description.toLowerCase().includes(search.toLowerCase()) ||
      s.tags.some((t) => t.includes(search.toLowerCase()));
    return matchKind && matchSearch;
  });

  const selectedSkill = selected ? skills.find((s) => s.id === selected) : null;
  const builtIns = filtered.filter((s) => s.built_in);
  const custom = filtered.filter((s) => !s.built_in);

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title="Skills"
        subtitle={`${skills.filter((s) => s.enabled).length} active · ${skills.length} total`}
      />

      <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>
        {/* Main */}
        <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
          {/* Filter bar */}
          <div style={{
            padding: "10px 20px", borderBottom: "1px solid var(--border)",
            display: "flex", gap: 10, alignItems: "center",
          }}>
            <div style={{ position: "relative", flex: 1, maxWidth: 240 }}>
              <Search size={13} style={{
                position: "absolute", left: 9, top: "50%",
                transform: "translateY(-50%)", color: "var(--text-faint)", pointerEvents: "none",
              }} />
              <input
                className="input" placeholder="Search skills…"
                value={search} onChange={(e) => setSearch(e.target.value)}
                style={{ paddingLeft: 28, height: 30, fontSize: 12 }}
              />
            </div>
            <div style={{ display: "flex", gap: 3 }}>
              {(["all", "doc", "sheet", "deck", "pdf", "text"] as (SkillKind | "all")[]).map((k) => (
                <button key={k} onClick={() => setKindFilter(k)} style={{
                  padding: "3px 10px", borderRadius: 5, fontSize: 12, fontWeight: 500,
                  cursor: "pointer", border: "1px solid",
                  background: kindFilter === k ? "var(--accent-dim)" : "transparent",
                  borderColor: kindFilter === k ? "var(--accent)" : "transparent",
                  color: kindFilter === k ? "var(--accent)" : "var(--text-faint)",
                  transition: "all 120ms",
                }}>
                  {KIND_LABELS[k]}
                </button>
              ))}
            </div>
          </div>

          {/* Skill list */}
          <div style={{ flex: 1, overflowY: "auto", padding: "16px 20px" }}>
            {isLoading ? (
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
                {[1,2,3,4].map(i => (
                  <div key={i} className="skeleton" style={{ height: 100, borderRadius: 9 }} />
                ))}
              </div>
            ) : (
              <>
                {builtIns.length > 0 && (
                  <>
                    <div className="section-label" style={{ marginBottom: 10, paddingLeft: 2 }}>
                      Built-in Skills — {builtIns.length}
                    </div>
                    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10, marginBottom: 24 }}>
                      {builtIns.map((s) => (
                        <SkillCard key={s.id} skill={s} selected={selected === s.id}
                          onSelect={() => setSelected(selected === s.id ? null : s.id)} />
                      ))}
                    </div>
                  </>
                )}

                {custom.length > 0 && (
                  <>
                    <div className="section-label" style={{ marginBottom: 10, paddingLeft: 2 }}>
                      Custom Skills — {custom.length}
                    </div>
                    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10, marginBottom: 24 }}>
                      {custom.map((s) => (
                        <SkillCard key={s.id} skill={s} selected={selected === s.id}
                          onSelect={() => setSelected(selected === s.id ? null : s.id)} />
                      ))}
                    </div>
                  </>
                )}

                {filtered.length === 0 && (
                  <div className="empty-state">
                    <div className="empty-state-icon"><Sparkles size={18} color="var(--text-faint)" /></div>
                    <p style={{ fontSize: 13, color: "var(--text-muted)" }}>No skills match your filter.</p>
                  </div>
                )}

                {/* Custom skills placeholder */}
                <div className="section-label" style={{ marginBottom: 8, paddingLeft: 2 }}>
                  Add Custom Skills
                </div>
                <div className="card" style={{ padding: "16px 20px", borderStyle: "dashed" }}>
                  <p style={{ fontSize: 12.5, color: "var(--text-muted)", margin: 0 }}>
                    Drop skill definition files in the{" "}
                    <code style={{ fontSize: 11, background: "var(--surface-3)", padding: "1px 5px", borderRadius: 3 }}>
                      skills/
                    </code>{" "}
                    directory, or create one via the API.
                  </p>
                </div>
              </>
            )}
          </div>
        </div>

        {/* Right detail panel */}
        {selectedSkill && (
          <div className="right-panel">
            <SkillDetail
              skill={selectedSkill}
              onToggle={() =>
                toggleMutation.mutate({ id: selectedSkill.id, enabled: !selectedSkill.enabled })
              }
            />
          </div>
        )}
      </div>
    </div>
  );
}
