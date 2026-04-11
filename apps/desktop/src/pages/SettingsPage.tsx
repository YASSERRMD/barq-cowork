import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Key,
  Eye,
  EyeOff,
  CheckCircle2,
  XCircle,
  Loader2,
  Plus,
  Trash2,
  Shield,
  ChevronDown,
} from "lucide-react";
import {
  providersApi,
  type ProviderProfile,
  type TestResult,
} from "../lib/api";
import clsx from "clsx";

// ─────────────────────────────────────────────
// Page shell
// ─────────────────────────────────────────────

export function SettingsPage() {
  const [tab, setTab] = useState<"providers" | "security">("providers");

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <div className="px-6 pt-6 pb-0 max-w-3xl">
        <h1 className="page-title">Settings</h1>

        <div className="flex gap-0 mt-5 border-b border-surface-2">
          {(["providers", "security"] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={clsx(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors capitalize",
                tab === t
                  ? "border-barq-500 text-text-primary"
                  : "border-transparent text-text-muted hover:text-text-secondary"
              )}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      <div className="px-6 py-5 max-w-3xl">
        {tab === "providers" && <ProvidersTab />}
        {tab === "security" && <SecurityTab />}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────
// Providers tab
// ─────────────────────────────────────────────

function ProvidersTab() {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [testResults, setTestResults] = useState<Record<string, TestResult>>({});

  const { data: profiles = [], isLoading } = useQuery({
    queryKey: ["provider-profiles"],
    queryFn: providersApi.listProfiles,
    retry: 1,
  });

  const deleteMutation = useMutation({
    mutationFn: providersApi.deleteProfile,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["provider-profiles"] }),
  });

  const testMutation = useMutation({
    mutationFn: (id: string) => providersApi.testProfile(id),
    onSuccess: (result, id) =>
      setTestResults((prev) => ({ ...prev, [id]: result })),
  });

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold text-text-primary">LLM Providers</h2>
          <p className="text-xs text-text-muted mt-0.5">
            Configure API keys and models. Keys are stored locally and never transmitted.
          </p>
        </div>
        <button
          className="btn-primary flex items-center gap-1.5 text-xs"
          onClick={() => setShowForm((v) => !v)}
        >
          {showForm ? (
            "Cancel"
          ) : (
            <>
              <Plus size={13} strokeWidth={2} />
              Add Provider
            </>
          )}
        </button>
      </div>

      {showForm && (
        <CreateProfileForm
          onSuccess={() => {
            qc.invalidateQueries({ queryKey: ["provider-profiles"] });
            setShowForm(false);
          }}
          onCancel={() => setShowForm(false)}
        />
      )}

      {isLoading && (
        <div className="flex items-center gap-2 text-text-muted text-sm py-4">
          <Loader2 size={14} className="animate-spin" />
          Loading providers…
        </div>
      )}

      {!isLoading && profiles.length === 0 && !showForm && (
        <div className="surface-2 rounded-lg p-8 text-center space-y-2">
          <Key size={24} className="mx-auto text-text-muted opacity-40" />
          <p className="text-sm text-text-secondary">No providers configured yet</p>
          <p className="text-xs text-text-muted">
            Add a provider profile to run tasks with an LLM.
          </p>
          <button
            className="btn-primary text-xs mt-2"
            onClick={() => setShowForm(true)}
          >
            Add your first provider
          </button>
        </div>
      )}

      <div className="space-y-2">
        {profiles.map((p) => (
          <ProfileCard
            key={p.id}
            profile={p}
            testResult={testResults[p.id]}
            onTest={() => testMutation.mutate(p.id)}
            testing={testMutation.isPending && testMutation.variables === p.id}
            onDelete={() => deleteMutation.mutate(p.id)}
          />
        ))}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────
// Profile card
// ─────────────────────────────────────────────

function ProfileCard({
  profile: p,
  testResult,
  onTest,
  testing,
  onDelete,
}: {
  profile: ProviderProfile;
  testResult?: TestResult;
  onTest: () => void;
  testing: boolean;
  onDelete: () => void;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);

  return (
    <div className="surface-2 rounded-lg border border-surface-3 p-4 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-medium text-text-primary">{p.name}</span>
            {p.is_default && (
              <span className="badge-blue text-[10px] px-1.5 py-0.5">default</span>
            )}
            <span className="badge-gray text-[10px] px-1.5 py-0.5 capitalize">{p.provider_name}</span>
            {p.api_key_set ? (
              <span className="badge-green text-[10px] px-1.5 py-0.5 flex items-center gap-1">
                <CheckCircle2 size={9} strokeWidth={2.5} />
                key configured
              </span>
            ) : (
              <span className="badge-red text-[10px] px-1.5 py-0.5 flex items-center gap-1">
                <XCircle size={9} strokeWidth={2.5} />
                no key
              </span>
            )}
          </div>
          <p className="text-xs text-text-muted font-mono">{p.base_url}</p>
          <div className="flex items-center gap-3 text-xs text-text-muted">
            <span>
              Model: <span className="text-text-secondary">{p.model}</span>
            </span>
            {p.api_key_hint && (
              <span className="font-mono text-text-muted opacity-60">{p.api_key_hint}</span>
            )}
          </div>
        </div>

        <div className="flex items-center gap-1.5 shrink-0">
          <button
            className="btn-ghost text-xs h-7"
            onClick={onTest}
            disabled={testing || !p.api_key_set}
            title={!p.api_key_set ? "Configure a key first" : "Test connection"}
          >
            {testing ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              "Test"
            )}
          </button>

          {confirmDelete ? (
            <>
              <button
                className="btn-danger text-xs h-7"
                onClick={() => { onDelete(); setConfirmDelete(false); }}
              >
                Confirm
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
              title="Delete profile"
            >
              <Trash2 size={13} strokeWidth={1.75} />
            </button>
          )}
        </div>
      </div>

      {testResult && (
        <div
          className={clsx(
            "flex items-center gap-2 text-xs px-3 py-2 rounded",
            testResult.ok
              ? "bg-green-500/10 border border-green-500/20 text-green-400"
              : "bg-red-500/10 border border-red-500/20 text-red-400"
          )}
        >
          {testResult.ok ? (
            <CheckCircle2 size={12} strokeWidth={2.5} />
          ) : (
            <XCircle size={12} strokeWidth={2.5} />
          )}
          {testResult.message}
        </div>
      )}
    </div>
  );
}

// ─────────────────────────────────────────────
// Create profile form
// ─────────────────────────────────────────────

const PROVIDER_PRESETS: Record<
  string,
  { label: string; baseURL: string; model: string; placeholder: string }
> = {
  zai: {
    label: "Z.AI",
    baseURL: "https://api.z.ai/api/coding/paas/v4",
    model: "GLM-4.7",
    placeholder: "zai-…",
  },
  openai: {
    label: "OpenAI",
    baseURL: "https://api.openai.com/v1",
    model: "gpt-4o",
    placeholder: "sk-…",
  },
  anthropic: {
    label: "Anthropic",
    baseURL: "https://api.anthropic.com/v1",
    model: "claude-opus-4-5",
    placeholder: "sk-ant-…",
  },
  gemini: {
    label: "Gemini",
    baseURL: "https://generativelanguage.googleapis.com/v1beta",
    model: "gemini-2.5-pro",
    placeholder: "AIza…",
  },
  ollama: {
    label: "Ollama",
    baseURL: "http://localhost:11434/v1",
    model: "llama3.2",
    placeholder: "ollama (no key needed)",
  },
};

function CreateProfileForm({
  onSuccess,
  onCancel,
}: {
  onSuccess: () => void;
  onCancel: () => void;
}) {
  const [name, setName] = useState("");
  const [providerName, setProviderName] = useState("zai");
  const [baseURL, setBaseURL] = useState(PROVIDER_PRESETS.zai.baseURL);
  const [apiKey, setApiKey] = useState("");
  const [model, setModel] = useState(PROVIDER_PRESETS.zai.model);
  const [timeoutSec, setTimeoutSec] = useState(120);
  const [isDefault, setIsDefault] = useState(false);
  const [showKey, setShowKey] = useState(false);
  const [advanced, setAdvanced] = useState(false);

  const createMutation = useMutation({
    mutationFn: providersApi.createProfile,
    onSuccess,
  });

  const applyPreset = (pName: string) => {
    setProviderName(pName);
    const preset = PROVIDER_PRESETS[pName];
    if (preset) {
      setBaseURL(preset.baseURL);
      setModel(preset.model);
    }
  };

  const preset = PROVIDER_PRESETS[providerName];

  return (
    <div className="surface-2 rounded-lg border border-surface-3 p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-text-primary">New Provider</h3>
      </div>

      {/* Provider picker */}
      <div className="space-y-1.5">
        <label className="text-xs font-medium text-text-secondary">Provider</label>
        <div className="flex flex-wrap gap-1.5">
          {Object.entries(PROVIDER_PRESETS).map(([key, val]) => (
            <button
              key={key}
              type="button"
              className={clsx(
                "px-3 py-1.5 rounded text-xs font-medium transition-colors border",
                providerName === key
                  ? "bg-barq-600 border-barq-500 text-white"
                  : "bg-transparent border-surface-3 text-text-secondary hover:border-barq-600/50 hover:text-text-primary"
              )}
              onClick={() => applyPreset(key)}
            >
              {val.label}
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-3">
        {/* Profile name */}
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-text-secondary">
            Profile Name <span className="text-red-400">*</span>
          </label>
          <input
            className="input"
            placeholder={`e.g. My ${preset?.label ?? providerName} profile`}
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>

        {/* API Key */}
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-text-secondary flex items-center gap-1.5">
            <Key size={11} />
            API Key
            {providerName === "ollama" && (
              <span className="text-text-muted font-normal">(not required for local Ollama)</span>
            )}
          </label>
          <div className="relative">
            <input
              className="input pr-9"
              type={showKey ? "text" : "password"}
              placeholder={preset?.placeholder ?? "Enter API key"}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              autoComplete="off"
            />
            <button
              type="button"
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
              onClick={() => setShowKey((v) => !v)}
              tabIndex={-1}
            >
              {showKey ? <EyeOff size={14} /> : <Eye size={14} />}
            </button>
          </div>
          <p className="text-[11px] text-text-muted">
            Stored locally in your SQLite database. Never sent to Barq servers.
          </p>
        </div>

        {/* Model */}
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-text-secondary">Model</label>
          <input
            className="input"
            value={model}
            onChange={(e) => setModel(e.target.value)}
            placeholder="e.g. gpt-4o"
          />
        </div>

        {/* Advanced toggle */}
        <button
          type="button"
          className="flex items-center gap-1.5 text-xs text-text-muted hover:text-text-secondary transition-colors"
          onClick={() => setAdvanced((v) => !v)}
        >
          <ChevronDown
            size={13}
            className={clsx("transition-transform", advanced && "rotate-180")}
          />
          Advanced options
        </button>

        {advanced && (
          <div className="space-y-3 pl-0">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-text-secondary">Base URL</label>
              <input
                className="input font-mono text-xs"
                value={baseURL}
                onChange={(e) => setBaseURL(e.target.value)}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-text-secondary">Timeout (sec)</label>
                <input
                  className="input"
                  type="number"
                  min={5}
                  max={600}
                  value={timeoutSec}
                  onChange={(e) => setTimeoutSec(Number(e.target.value))}
                />
              </div>
              <div className="flex items-end pb-1">
                <label className="flex items-center gap-2 cursor-pointer select-none">
                  <input
                    type="checkbox"
                    className="rounded"
                    checked={isDefault}
                    onChange={(e) => setIsDefault(e.target.checked)}
                  />
                  <span className="text-sm text-text-secondary">Set as default</span>
                </label>
              </div>
            </div>
          </div>
        )}
      </div>

      {createMutation.error && (
        <div className="flex items-center gap-2 text-xs px-3 py-2 rounded bg-red-500/10 border border-red-500/20 text-red-400">
          <XCircle size={12} />
          {(createMutation.error as Error).message}
        </div>
      )}

      <div className="flex items-center gap-2 pt-1">
        <button
          type="button"
          className="btn-primary text-sm"
          disabled={createMutation.isPending || !name.trim()}
          onClick={() =>
            createMutation.mutate({
              name,
              provider_name: providerName,
              base_url: baseURL,
              api_key: apiKey,
              model,
              timeout_sec: timeoutSec,
              is_default: isDefault,
            })
          }
        >
          {createMutation.isPending ? (
            <><Loader2 size={13} className="animate-spin" /> Saving…</>
          ) : (
            "Save Provider"
          )}
        </button>
        <button type="button" className="btn-ghost text-sm" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────
// Security tab
// ─────────────────────────────────────────────

function SecurityTab() {
  const items = [
    {
      label: "API key storage",
      value: "Local SQLite database only — never synced, never transmitted",
      status: "ok",
    },
    {
      label: "Destructive tool approval",
      value: "Required before file moves, HTTP POST, and shell operations",
      status: "ok",
    },
    {
      label: "Workspace root scoping",
      value: "File tools are restricted to the configured workspace root path",
      status: "ok",
    },
    {
      label: "Backend binding",
      value: "127.0.0.1:7331 — not exposed to the network",
      status: "ok",
    },
  ];

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-sm font-semibold text-text-primary">Security</h2>
        <p className="text-xs text-text-muted mt-0.5">
          Barq Cowork is a local desktop app. All data stays on this machine.
        </p>
      </div>

      <div className="surface-2 rounded-lg border border-surface-3 divide-y divide-surface-3">
        {items.map((item) => (
          <div key={item.label} className="flex items-start gap-4 px-4 py-3">
            <Shield size={14} className="text-green-400 mt-0.5 shrink-0" />
            <div className="min-w-0">
              <p className="text-sm text-text-primary">{item.label}</p>
              <p className="text-xs text-text-muted mt-0.5">{item.value}</p>
            </div>
          </div>
        ))}
      </div>

      <div className="surface-2 rounded-lg border border-surface-3 p-4 space-y-2">
        <p className="text-xs font-semibold text-text-secondary uppercase tracking-wider">
          Data Locations
        </p>
        <div className="space-y-1.5 text-xs">
          <div className="flex items-start gap-2">
            <span className="text-text-muted w-24 shrink-0">Database</span>
            <code className="font-mono text-text-secondary break-all">
              ~/.local/share/barq-cowork/barq.db
            </code>
          </div>
          <div className="flex items-start gap-2">
            <span className="text-text-muted w-24 shrink-0">Logs</span>
            <code className="font-mono text-text-secondary break-all">stdout (console)</code>
          </div>
          <div className="flex items-start gap-2">
            <span className="text-text-muted w-24 shrink-0">Artifacts</span>
            <code className="font-mono text-text-secondary break-all">workspace root / reports /</code>
          </div>
        </div>
      </div>
    </div>
  );
}
