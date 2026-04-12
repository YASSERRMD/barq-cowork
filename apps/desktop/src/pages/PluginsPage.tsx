import { useState } from "react";
import {
  Sparkles,
  Plug,
  Terminal,
  Users,
  ChevronRight,
  Package,
} from "lucide-react";
import { TopBar } from "../components/TopBar";

// ── Plugin type ───────────────────────────────────────────────────
interface Plugin {
  id: string;
  name: string;
  description: string;
  version: string;
  author: string;
  enabled: boolean;
  builtin: boolean;
  skills: string[];
  connectors: string[];
  slashCommands: string[];
  tags: string[];
  accentColor: string;
}

const SAMPLE_PLUGINS: Plugin[] = [
  {
    id: "file-organizer",
    name: "File Organizer",
    description: "Batch file organization with smart rename, folder structure generation, and metadata extraction.",
    version: "1.0.0",
    author: "Barq Team",
    enabled: true,
    builtin: true,
    skills: ["text", "doc"],
    connectors: [],
    slashCommands: ["/organize", "/rename-batch"],
    tags: ["files", "organize", "rename"],
    accentColor: "#34d399",
  },
  {
    id: "document-suite",
    name: "Document Suite",
    description: "Full document pipeline — convert, summarize, compare, and generate Word, PDF, and Markdown files.",
    version: "1.0.0",
    author: "Barq Team",
    enabled: true,
    builtin: true,
    skills: ["docx", "pdf", "text"],
    connectors: [],
    slashCommands: ["/summarize", "/compare-docs"],
    tags: ["docs", "pdf", "word"],
    accentColor: "#60a5fa",
  },
  {
    id: "data-analyst",
    name: "Data Analyst",
    description: "Analyze spreadsheets, generate pivot summaries, and produce XLSX reports from raw data files.",
    version: "1.0.0",
    author: "Barq Team",
    enabled: false,
    builtin: true,
    skills: ["xlsx"],
    connectors: [],
    slashCommands: ["/analyze-data"],
    tags: ["excel", "data", "analytics"],
    accentColor: "#fbbf24",
  },
];

function PluginCard({
  plugin,
  selected,
  onSelect,
  onToggle,
}: {
  plugin: Plugin;
  selected: boolean;
  onSelect: () => void;
  onToggle: () => void;
}) {
  return (
    <div
      className={selected ? "card" : "card-hover"}
      onClick={onSelect}
      style={{
        padding: "14px 16px",
        display: "flex",
        alignItems: "flex-start",
        gap: 12,
        borderColor: selected ? "var(--accent)" : undefined,
        boxShadow: selected ? "0 0 0 1px var(--accent), 0 4px 16px var(--accent-glow)" : undefined,
        transition: "all 150ms",
        cursor: "pointer",
      }}
    >
      <div
        style={{
          width: 36,
          height: 36,
          borderRadius: 9,
          background: `${plugin.accentColor}18`,
          border: `1px solid ${plugin.accentColor}30`,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexShrink: 0,
        }}
      >
        <Package size={17} color={plugin.accentColor} />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 7, marginBottom: 4 }}>
          <span style={{ fontSize: 13.5, fontWeight: 600, color: "var(--text-primary)", letterSpacing: "-0.01em" }}>
            {plugin.name}
          </span>
          {plugin.builtin && (
            <span className="badge-accent" style={{ fontSize: 10 }}>Built-in</span>
          )}
          <span
            style={{ marginLeft: "auto", fontSize: 11, color: "var(--text-faint)" }}
          >
            v{plugin.version}
          </span>
        </div>
        <p style={{ fontSize: 12, color: "var(--text-secondary)", margin: 0, lineHeight: 1.5 }}>
          {plugin.description}
        </p>
        <div style={{ display: "flex", gap: 5, marginTop: 8, flexWrap: "wrap", alignItems: "center" }}>
          {plugin.skills.length > 0 && (
            <span
              style={{
                display: "flex",
                alignItems: "center",
                gap: 3,
                fontSize: 11,
                color: "var(--text-faint)",
              }}
            >
              <Sparkles size={10} />
              {plugin.skills.length} skill{plugin.skills.length !== 1 ? "s" : ""}
            </span>
          )}
          {plugin.slashCommands.length > 0 && (
            <span
              style={{
                display: "flex",
                alignItems: "center",
                gap: 3,
                fontSize: 11,
                color: "var(--text-faint)",
              }}
            >
              <Terminal size={10} />
              {plugin.slashCommands.length} command{plugin.slashCommands.length !== 1 ? "s" : ""}
            </span>
          )}
        </div>
      </div>

      {/* Enable/disable toggle */}
      <button
        className={plugin.enabled ? "btn-primary btn-xs" : "btn-secondary btn-xs"}
        style={{ flexShrink: 0, marginTop: 2 }}
        onClick={(e) => { e.stopPropagation(); onToggle(); }}
      >
        {plugin.enabled ? "Enabled" : "Enable"}
      </button>
    </div>
  );
}

function PluginDetail({ plugin }: { plugin: Plugin }) {
  return (
    <div style={{ padding: "20px", display: "flex", flexDirection: "column", gap: 20 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <div
          style={{
            width: 42,
            height: 42,
            borderRadius: 11,
            background: `${plugin.accentColor}18`,
            border: `1px solid ${plugin.accentColor}30`,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
          }}
        >
          <Package size={20} color={plugin.accentColor} />
        </div>
        <div>
          <div style={{ fontSize: 15, fontWeight: 600, color: "var(--text-primary)", letterSpacing: "-0.015em" }}>
            {plugin.name}
          </div>
          <div style={{ fontSize: 11.5, color: "var(--text-muted)", marginTop: 1 }}>
            v{plugin.version} · by {plugin.author}
          </div>
        </div>
      </div>

      <p style={{ fontSize: 12.5, color: "var(--text-secondary)", margin: 0, lineHeight: 1.6 }}>
        {plugin.description}
      </p>

      {/* Skills */}
      {plugin.skills.length > 0 && (
        <div>
          <div className="section-label" style={{ marginBottom: 8 }}>
            <Sparkles size={10} style={{ display: "inline", marginRight: 4 }} />
            Included Skills
          </div>
          <div style={{ display: "flex", gap: 5, flexWrap: "wrap" }}>
            {plugin.skills.map((s) => (
              <span
                key={s}
                style={{
                  fontSize: 12,
                  background: "var(--accent-dim)",
                  color: "var(--accent)",
                  borderRadius: 5,
                  padding: "3px 10px",
                  border: "1px solid var(--accent-glow)",
                }}
              >
                {s}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Connectors */}
      {plugin.connectors.length > 0 && (
        <div>
          <div className="section-label" style={{ marginBottom: 8 }}>
            <Plug size={10} style={{ display: "inline", marginRight: 4 }} />
            Connectors
          </div>
          <div style={{ display: "flex", gap: 5, flexWrap: "wrap" }}>
            {plugin.connectors.map((c) => (
              <span
                key={c}
                style={{
                  fontSize: 12,
                  background: "var(--surface-3)",
                  color: "var(--text-secondary)",
                  borderRadius: 5,
                  padding: "3px 10px",
                  border: "1px solid var(--border)",
                }}
              >
                {c}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Slash commands */}
      {plugin.slashCommands.length > 0 && (
        <div>
          <div className="section-label" style={{ marginBottom: 8 }}>
            <Terminal size={10} style={{ display: "inline", marginRight: 4 }} />
            Slash Commands
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 5 }}>
            {plugin.slashCommands.map((cmd) => (
              <div
                key={cmd}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                  padding: "6px 10px",
                  background: "var(--surface-3)",
                  border: "1px solid var(--border)",
                  borderRadius: 6,
                }}
              >
                <code style={{ fontSize: 12, color: "var(--accent)", flex: 1 }}>{cmd}</code>
                <ChevronRight size={11} color="var(--text-faint)" />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tags */}
      <div>
        <div className="section-label" style={{ marginBottom: 6 }}>Tags</div>
        <div style={{ display: "flex", gap: 5, flexWrap: "wrap" }}>
          {plugin.tags.map((tag) => (
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
  );
}

export function PluginsPage() {
  const [plugins, setPlugins] = useState<Plugin[]>(SAMPLE_PLUGINS);
  const [selected, setSelected] = useState<string | null>(null);

  const togglePlugin = (id: string) => {
    setPlugins((prev) =>
      prev.map((p) => (p.id === id ? { ...p, enabled: !p.enabled } : p))
    );
  };

  const selectedPlugin = selected ? plugins.find((p) => p.id === selected) : null;
  const enabledCount = plugins.filter((p) => p.enabled).length;

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title="Plugins"
        subtitle={`${enabledCount} of ${plugins.length} enabled`}
      />

      <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>
        {/* Main */}
        <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
          <div style={{ flex: 1, overflowY: "auto", padding: "16px 20px" }}>
            {/* Installed */}
            <div className="section-label" style={{ marginBottom: 10, paddingLeft: 2 }}>
              Installed — {plugins.length}
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              {plugins.map((plugin) => (
                <PluginCard
                  key={plugin.id}
                  plugin={plugin}
                  selected={selected === plugin.id}
                  onSelect={() => setSelected(selected === plugin.id ? null : plugin.id)}
                  onToggle={() => togglePlugin(plugin.id)}
                />
              ))}
            </div>

            {/* Install more placeholder */}
            <div style={{ marginTop: 24 }}>
              <div className="section-label" style={{ marginBottom: 10, paddingLeft: 2 }}>
                Available to Install
              </div>
              <div
                className="card"
                style={{ padding: "20px", textAlign: "center", borderStyle: "dashed" }}
              >
                <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: 8, marginBottom: 6 }}>
                  <Users size={14} color="var(--text-faint)" />
                  <span style={{ fontSize: 13, color: "var(--text-muted)", fontWeight: 500 }}>
                    Community plugins
                  </span>
                </div>
                <p style={{ fontSize: 12, color: "var(--text-faint)", margin: 0 }}>
                  Place plugin bundles in the <code style={{ fontSize: 11, background: "var(--surface-3)", padding: "1px 5px", borderRadius: 3 }}>plugins/</code> directory to install them.
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Right panel */}
        {selectedPlugin && (
          <div className="right-panel">
            <PluginDetail plugin={selectedPlugin} />
          </div>
        )}
      </div>
    </div>
  );
}
