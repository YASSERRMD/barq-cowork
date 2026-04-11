import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { providersApi, type AvailableProvider, type ProviderProfile, type TestResult } from "../lib/api";
import clsx from "clsx";

export function SettingsPage() {
  const [tab, setTab] = useState<"providers" | "security">("providers");

  return (
    <div className="p-6 space-y-5 max-w-3xl">
      <h1 className="text-xl font-semibold text-white">Settings</h1>

      {/* Tab bar */}
      <div className="flex gap-1 border-b border-gray-800 pb-0">
        {(["providers", "security"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={clsx(
              "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors capitalize",
              tab === t
                ? "border-barq-500 text-white"
                : "border-transparent text-gray-500 hover:text-gray-300"
            )}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "providers" && <ProvidersTab />}
      {tab === "security" && <SecurityTab />}
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

  const { data: available = [] } = useQuery({
    queryKey: ["providers-available"],
    queryFn: providersApi.listAvailable,
    retry: 1,
  });

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

  const quickTestMutation = useMutation({
    mutationFn: (p: AvailableProvider) =>
      providersApi.test({ provider_name: p.name, api_key_env: p.key_env }),
    onSuccess: (result, p) =>
      setTestResults((prev) => ({ ...prev, [p.name]: result })),
  });

  return (
    <div className="space-y-5">
      {/* Available providers (from config) */}
      <section className="space-y-2">
        <h2 className="text-xs font-semibold text-gray-400 uppercase tracking-wider">
          Configured Providers
        </h2>
        {available.length === 0 && (
          <p className="text-gray-500 text-sm">Backend not reachable.</p>
        )}
        {available.map((p) => (
          <AvailableProviderCard
            key={p.name}
            provider={p}
            testResult={testResults[p.name]}
            onTest={() => quickTestMutation.mutate(p)}
            testing={quickTestMutation.isPending && quickTestMutation.variables?.name === p.name}
          />
        ))}
      </section>

      {/* Saved profiles */}
      <section className="space-y-2">
        <div className="flex items-center justify-between">
          <h2 className="text-xs font-semibold text-gray-400 uppercase tracking-wider">
            Saved Profiles
          </h2>
          <button className="btn-primary text-xs" onClick={() => setShowForm((v) => !v)}>
            {showForm ? "Cancel" : "+ Add Profile"}
          </button>
        </div>

        {showForm && (
          <CreateProfileForm
            onSuccess={() => {
              qc.invalidateQueries({ queryKey: ["provider-profiles"] });
              setShowForm(false);
            }}
          />
        )}

        {isLoading && <p className="text-gray-500 text-sm">Loading…</p>}

        {!isLoading && profiles.length === 0 && (
          <p className="text-gray-500 text-sm">No saved profiles yet.</p>
        )}

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
      </section>
    </div>
  );
}

function AvailableProviderCard({
  provider: p,
  testResult,
  onTest,
  testing,
}: {
  provider: AvailableProvider;
  testResult?: TestResult;
  onTest: () => void;
  testing: boolean;
}) {
  return (
    <div className="card p-4 space-y-2">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-white font-medium uppercase text-sm">{p.name}</span>
            {p.has_key ? (
              <span className="badge-green">key set</span>
            ) : (
              <span className="badge-red">no key</span>
            )}
          </div>
          <p className="text-gray-500 text-xs font-mono mt-0.5">{p.base_url}</p>
          <p className="text-gray-500 text-xs mt-0.5">
            Model: <span className="text-gray-300">{p.model}</span>
            {" · "}Env: <span className="text-gray-300 font-mono">{p.key_env}</span>
          </p>
        </div>
        <button
          className="btn-ghost text-xs shrink-0"
          onClick={onTest}
          disabled={testing || !p.has_key}
          title={!p.has_key ? `Set ${p.key_env} env var first` : "Test connection"}
        >
          {testing ? "Testing…" : "Test"}
        </button>
      </div>
      {testResult && (
        <p className={clsx("text-xs px-2 py-1 rounded", testResult.ok ? "bg-green-900/30 text-green-300" : "bg-red-900/30 text-red-300")}>
          {testResult.ok ? "✓" : "✗"} {testResult.message}
        </p>
      )}
    </div>
  );
}

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
  return (
    <div className="card p-4 space-y-2">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-white font-medium">{p.name}</span>
            {p.is_default && <span className="badge-blue">default</span>}
            <span className="badge-gray">{p.provider_name}</span>
          </div>
          <p className="text-gray-500 text-xs font-mono mt-0.5">{p.base_url}</p>
          <p className="text-gray-500 text-xs mt-0.5">
            Model: <span className="text-gray-300">{p.model}</span>
            {" · "}Key env: <span className="text-gray-300 font-mono">{p.api_key_env}</span>
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button className="btn-ghost text-xs" onClick={onTest} disabled={testing}>
            {testing ? "Testing…" : "Test"}
          </button>
          <button
            className="btn-ghost text-xs text-red-400 hover:text-red-300"
            onClick={onDelete}
          >
            Delete
          </button>
        </div>
      </div>
      {testResult && (
        <p className={clsx("text-xs px-2 py-1 rounded", testResult.ok ? "bg-green-900/30 text-green-300" : "bg-red-900/30 text-red-300")}>
          {testResult.ok ? "✓" : "✗"} {testResult.message}
        </p>
      )}
    </div>
  );
}

// ─────────────────────────────────────────────
// Create profile form
// ─────────────────────────────────────────────

const PROVIDER_PRESETS: Record<string, { baseURL: string; model: string; keyEnv: string }> = {
  zai: { baseURL: "https://api.z.ai/api/coding/paas/v4", model: "GLM-4.7", keyEnv: "ZAI_API_KEY" },
  openai: { baseURL: "https://api.openai.com/v1", model: "gpt-4.1", keyEnv: "OPENAI_API_KEY" },
};

function CreateProfileForm({ onSuccess }: { onSuccess: () => void }) {
  const [name, setName] = useState("");
  const [providerName, setProviderName] = useState("zai");
  const [baseURL, setBaseURL] = useState(PROVIDER_PRESETS.zai.baseURL);
  const [apiKeyEnv, setApiKeyEnv] = useState(PROVIDER_PRESETS.zai.keyEnv);
  const [model, setModel] = useState(PROVIDER_PRESETS.zai.model);
  const [timeoutSec, setTimeoutSec] = useState(120);
  const [isDefault, setIsDefault] = useState(false);

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
      setApiKeyEnv(preset.keyEnv);
    }
  };

  return (
    <form
      className="card p-4 space-y-3"
      onSubmit={(e) => {
        e.preventDefault();
        createMutation.mutate({
          name, provider_name: providerName, base_url: baseURL,
          api_key_env: apiKeyEnv, model, timeout_sec: timeoutSec, is_default: isDefault,
        });
      }}
    >
      <h3 className="text-sm font-semibold text-gray-300">New Provider Profile</h3>
      <div className="grid grid-cols-2 gap-2">
        <input className="input col-span-2" placeholder="Profile name *" value={name}
          onChange={(e) => setName(e.target.value)} required />

        <div className="col-span-2">
          <label className="text-xs text-gray-500 mb-1 block">Provider</label>
          <div className="flex gap-2">
            {Object.keys(PROVIDER_PRESETS).map((p) => (
              <button
                key={p} type="button"
                className={clsx("btn text-xs", providerName === p ? "btn-primary" : "btn-ghost")}
                onClick={() => applyPreset(p)}
              >
                {p}
              </button>
            ))}
          </div>
        </div>

        <div className="col-span-2">
          <label className="text-xs text-gray-500 mb-1 block">Base URL</label>
          <input className="input font-mono text-xs" value={baseURL}
            onChange={(e) => setBaseURL(e.target.value)} />
        </div>
        <div>
          <label className="text-xs text-gray-500 mb-1 block">Model</label>
          <input className="input" value={model} onChange={(e) => setModel(e.target.value)} />
        </div>
        <div>
          <label className="text-xs text-gray-500 mb-1 block">API Key Env Var</label>
          <input className="input font-mono text-xs" value={apiKeyEnv}
            onChange={(e) => setApiKeyEnv(e.target.value)} />
        </div>
        <div>
          <label className="text-xs text-gray-500 mb-1 block">Timeout (sec)</label>
          <input className="input" type="number" min={5} max={300} value={timeoutSec}
            onChange={(e) => setTimeoutSec(Number(e.target.value))} />
        </div>
        <div className="flex items-center gap-2 pt-4">
          <input id="is-default" type="checkbox" checked={isDefault}
            onChange={(e) => setIsDefault(e.target.checked)} />
          <label htmlFor="is-default" className="text-sm text-gray-300 select-none">
            Set as default
          </label>
        </div>
      </div>
      {createMutation.error && (
        <p className="text-red-400 text-xs">{(createMutation.error as Error).message}</p>
      )}
      <button type="submit" className="btn-primary" disabled={createMutation.isPending}>
        {createMutation.isPending ? "Saving…" : "Save Profile"}
      </button>
    </form>
  );
}

// ─────────────────────────────────────────────
// Security tab
// ─────────────────────────────────────────────

function SecurityTab() {
  return (
    <section className="card p-4 space-y-3">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">Security</h2>
      <p className="text-gray-400 text-sm">
        Approval-required destructive actions are <span className="text-green-400 font-medium">enabled</span>.
      </p>
      <p className="text-gray-400 text-sm">
        Workspace root scoping will be configurable in Phase 4 when the tool system is active.
      </p>
      <div className="bg-yellow-900/20 border border-yellow-800 rounded-md p-3 text-xs text-yellow-300">
        API keys are never stored in the database. Only environment variable names are saved.
        Keys are resolved at request time by the backend and are never sent to the frontend.
      </div>
    </section>
  );
}
