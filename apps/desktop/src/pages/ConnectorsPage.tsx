import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plug,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Loader2,
  Settings,
  ExternalLink,
  RefreshCw,
} from "lucide-react";
import { providersApi, type ProviderProfile, type TestResult } from "../lib/api";
import clsx from "clsx";

// ─────────────────────────────────────────────
// Provider metadata (labels, descriptions, docs)
// ─────────────────────────────────────────────

const PROVIDER_META: Record<
  string,
  { label: string; description: string; category: string; docsUrl: string }
> = {
  zai: {
    label: "Z.AI",
    description: "High-performance coding-optimised LLMs from Z.AI. Ideal for code generation and analysis tasks.",
    category: "LLM",
    docsUrl: "https://docs.z.ai",
  },
  openai: {
    label: "OpenAI",
    description: "GPT-4o and o3 models from OpenAI. Best-in-class general reasoning and instruction following.",
    category: "LLM",
    docsUrl: "https://platform.openai.com/docs",
  },
  anthropic: {
    label: "Anthropic",
    description: "Claude models from Anthropic. Excellent at long-context understanding and agentic tasks.",
    category: "LLM",
    docsUrl: "https://docs.anthropic.com",
  },
  gemini: {
    label: "Google Gemini",
    description: "Gemini 2.5 Pro from Google. Multimodal capabilities and large context window.",
    category: "LLM",
    docsUrl: "https://ai.google.dev/docs",
  },
  ollama: {
    label: "Ollama",
    description: "Run open-source LLMs locally. No API key required — models run on this machine.",
    category: "Local",
    docsUrl: "https://ollama.com",
  },
};

function getMeta(name: string) {
  return (
    PROVIDER_META[name] ?? {
      label: name,
      description: "Custom OpenAI-compatible provider.",
      category: "LLM",
      docsUrl: "",
    }
  );
}

// ─────────────────────────────────────────────
// Page
// ─────────────────────────────────────────────

export function ConnectorsPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [testResults, setTestResults] = useState<Record<string, TestResult>>({});

  const { data: profiles = [], isLoading, error, refetch } = useQuery({
    queryKey: ["provider-profiles"],
    queryFn: providersApi.listProfiles,
    retry: 1,
  });

  const testMutation = useMutation({
    mutationFn: (id: string) => providersApi.testProfile(id),
    onSuccess: (result, id) =>
      setTestResults((prev) => ({ ...prev, [id]: result })),
  });

  const deleteMutation = useMutation({
    mutationFn: providersApi.deleteProfile,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["provider-profiles"] }),
  });

  // Group profiles by provider
  const grouped = profiles.reduce<Record<string, ProviderProfile[]>>((acc, p) => {
    const key = p.provider_name;
    if (!acc[key]) acc[key] = [];
    acc[key].push(p);
    return acc;
  }, {});

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <div className="px-6 pt-6 pb-5 flex items-start justify-between max-w-4xl">
        <div>
          <h1 className="page-title">Connectors</h1>
          <p className="text-sm text-text-muted mt-1">
            Manage LLM provider connections. Each connector links a provider to an API key and model.
          </p>
        </div>
        <button
          className="btn-primary flex items-center gap-1.5 text-sm shrink-0 mt-1"
          onClick={() => navigate("/settings")}
        >
          <Settings size={14} strokeWidth={1.75} />
          Manage in Settings
        </button>
      </div>

      <div className="px-6 pb-6 max-w-4xl space-y-6">
        {/* Loading */}
        {isLoading && (
          <div className="flex items-center gap-2 text-text-muted text-sm py-8">
            <Loader2 size={14} className="animate-spin" />
            Loading connectors…
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="surface-2 rounded-lg border border-red-500/20 p-4 flex items-center gap-3">
            <AlertCircle size={16} className="text-red-400 shrink-0" />
            <div>
              <p className="text-sm text-text-primary">Could not load connectors</p>
              <p className="text-xs text-text-muted">Backend may be unreachable.</p>
            </div>
            <button
              className="btn-ghost text-xs ml-auto"
              onClick={() => refetch()}
            >
              <RefreshCw size={13} />
              Retry
            </button>
          </div>
        )}

        {/* Empty */}
        {!isLoading && !error && profiles.length === 0 && (
          <div className="surface-2 rounded-lg border border-surface-3 p-10 text-center space-y-3">
            <Plug size={28} className="mx-auto text-text-muted opacity-30" />
            <p className="text-sm font-medium text-text-secondary">No connectors configured</p>
            <p className="text-xs text-text-muted max-w-sm mx-auto">
              Add a provider profile in Settings to create your first connector. Each profile
              becomes a connector available to all projects.
            </p>
            <button
              className="btn-primary text-sm mt-2"
              onClick={() => navigate("/settings")}
            >
              Go to Settings
            </button>
          </div>
        )}

        {/* Connector groups */}
        {!isLoading &&
          Object.entries(grouped).map(([providerName, group]) => {
            const meta = getMeta(providerName);
            return (
              <ConnectorGroup
                key={providerName}
                providerName={providerName}
                meta={meta}
                profiles={group}
                testResults={testResults}
                onTest={(id) => testMutation.mutate(id)}
                testing={(id) =>
                  testMutation.isPending && testMutation.variables === id
                }
                onDelete={(id) => deleteMutation.mutate(id)}
              />
            );
          })}

        {/* Provider catalog — providers not yet configured */}
        {!isLoading && (
          <ProviderCatalog configuredNames={Object.keys(grouped)} onAdd={() => navigate("/settings")} />
        )}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────
// Connector group
// ─────────────────────────────────────────────

function ConnectorGroup({
  providerName,
  meta,
  profiles,
  testResults,
  onTest,
  testing,
  onDelete,
}: {
  providerName: string;
  meta: ReturnType<typeof getMeta>;
  profiles: ProviderProfile[];
  testResults: Record<string, TestResult>;
  onTest: (id: string) => void;
  testing: (id: string) => boolean;
  onDelete: (id: string) => void;
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 mb-1">
        <span className="text-xs font-semibold text-text-secondary uppercase tracking-wider">
          {meta.label}
        </span>
        <span className="badge-gray text-[10px] px-1.5">{meta.category}</span>
        <span className="text-xs text-text-muted ml-auto">
          {profiles.length} profile{profiles.length !== 1 ? "s" : ""}
        </span>
      </div>

      {profiles.map((p) => (
        <ConnectorCard
          key={p.id}
          profile={p}
          meta={meta}
          testResult={testResults[p.id]}
          onTest={() => onTest(p.id)}
          testing={testing(p.id)}
          onDelete={() => onDelete(p.id)}
        />
      ))}
    </div>
  );
}

// ─────────────────────────────────────────────
// Single connector card
// ─────────────────────────────────────────────

function ConnectorCard({
  profile: p,
  meta,
  testResult,
  onTest,
  testing,
  onDelete,
}: {
  profile: ProviderProfile;
  meta: ReturnType<typeof getMeta>;
  testResult?: TestResult;
  onTest: () => void;
  testing: boolean;
  onDelete: () => void;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const status: "ok" | "error" | "unknown" = testResult
    ? testResult.ok
      ? "ok"
      : "error"
    : "unknown";

  const statusDot = {
    ok: "bg-green-500 shadow-[0_0_6px_rgba(16,185,129,0.5)]",
    error: "bg-red-500",
    unknown: "bg-surface-3",
  }[status];

  return (
    <div
      className={clsx(
        "surface-2 rounded-lg border p-4 transition-colors",
        status === "ok"
          ? "border-green-500/20"
          : status === "error"
          ? "border-red-500/20"
          : "border-surface-3"
      )}
    >
      <div className="flex items-start gap-3">
        {/* Status dot */}
        <div className="mt-1 shrink-0">
          <div className={clsx("w-2 h-2 rounded-full mt-0.5", statusDot)} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0 space-y-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-medium text-text-primary">{p.name}</span>
            {p.is_default && (
              <span className="badge-blue text-[10px] px-1.5 py-0.5">default</span>
            )}
            {p.api_key_set ? (
              <span className="badge-green text-[10px] px-1.5 py-0.5 flex items-center gap-1">
                <CheckCircle2 size={9} strokeWidth={2.5} />
                connected
              </span>
            ) : (
              <span className="badge-red text-[10px] px-1.5 py-0.5 flex items-center gap-1">
                <XCircle size={9} strokeWidth={2.5} />
                no key
              </span>
            )}
          </div>

          <div className="flex items-center gap-3 text-xs text-text-muted flex-wrap">
            <span className="font-mono">{p.base_url}</span>
            <span>·</span>
            <span>Model: <span className="text-text-secondary">{p.model}</span></span>
            {p.api_key_hint && (
              <>
                <span>·</span>
                <span className="font-mono opacity-60">{p.api_key_hint}</span>
              </>
            )}
          </div>

          {testResult && (
            <div
              className={clsx(
                "flex items-center gap-1.5 text-xs mt-2 px-2.5 py-1.5 rounded",
                testResult.ok
                  ? "bg-green-500/10 border border-green-500/20 text-green-400"
                  : "bg-red-500/10 border border-red-500/20 text-red-400"
              )}
            >
              {testResult.ok ? (
                <CheckCircle2 size={11} strokeWidth={2.5} />
              ) : (
                <XCircle size={11} strokeWidth={2.5} />
              )}
              {testResult.message}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 shrink-0">
          <button
            className="btn-ghost text-xs h-7"
            onClick={onTest}
            disabled={testing || !p.api_key_set}
            title={!p.api_key_set ? "Configure key in Settings first" : "Test connection"}
          >
            {testing ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <RefreshCw size={13} strokeWidth={1.75} />
            )}
          </button>

          {confirmDelete ? (
            <>
              <button
                className="btn-danger text-xs h-7"
                onClick={() => { onDelete(); setConfirmDelete(false); }}
              >
                Remove
              </button>
              <button
                className="btn-ghost text-xs h-7"
                onClick={() => setConfirmDelete(false)}
              >
                Cancel
              </button>
            </>
          ) : (
            <button
              className="btn-ghost text-xs h-7 text-text-muted hover:text-red-400"
              onClick={() => setConfirmDelete(true)}
            >
              <XCircle size={13} strokeWidth={1.75} />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────
// Provider catalog (unconfigured providers)
// ─────────────────────────────────────────────

function ProviderCatalog({
  configuredNames,
  onAdd,
}: {
  configuredNames: string[];
  onAdd: () => void;
}) {
  const unconfigured = Object.entries(PROVIDER_META).filter(
    ([key]) => !configuredNames.includes(key)
  );

  if (unconfigured.length === 0) return null;

  return (
    <div className="space-y-2">
      <div className="text-xs font-semibold text-text-muted uppercase tracking-wider mb-1">
        Available Providers
      </div>
      <div className="grid grid-cols-1 gap-2">
        {unconfigured.map(([key, meta]) => (
          <div
            key={key}
            className="surface-2 rounded-lg border border-surface-3 border-dashed p-4 flex items-start gap-4"
          >
            <div className="w-2 h-2 rounded-full bg-surface-3 mt-1.5 shrink-0" />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-text-secondary">{meta.label}</span>
                <span className="badge-gray text-[10px] px-1.5">{meta.category}</span>
              </div>
              <p className="text-xs text-text-muted mt-0.5 max-w-lg">{meta.description}</p>
            </div>
            <div className="flex items-center gap-1.5 shrink-0">
              {meta.docsUrl && (
                <a
                  href={meta.docsUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="btn-ghost text-xs h-7 flex items-center gap-1"
                >
                  <ExternalLink size={12} />
                  Docs
                </a>
              )}
              <button className="btn-ghost text-xs h-7" onClick={onAdd}>
                Configure
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
