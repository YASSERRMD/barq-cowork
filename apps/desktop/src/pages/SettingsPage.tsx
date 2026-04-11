export function SettingsPage() {
  return (
    <div className="p-6 max-w-2xl space-y-6">
      <h1 className="text-xl font-semibold text-white">Settings</h1>

      <section className="card p-4 space-y-3">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
          LLM Providers
        </h2>
        <p className="text-gray-400 text-sm">
          Provider configuration will be available here in Phase 3. Set your
          API keys via environment variables in the meantime:
        </p>
        <ul className="text-xs font-mono text-gray-400 space-y-1">
          <li>ZAI_API_KEY=…</li>
          <li>ZAI_BASE_URL=https://api.z.ai/api/coding/paas/v4</li>
          <li>ZAI_MODEL=GLM-4.7</li>
          <li>OPENAI_API_KEY=…</li>
          <li>OPENAI_MODEL=gpt-4.1</li>
        </ul>
      </section>

      <section className="card p-4 space-y-3">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
          Security
        </h2>
        <p className="text-gray-400 text-sm">
          Approval-required destructive actions are <span className="text-green-400">enabled</span>.
          Workspace root scoping will be configurable in Phase 4.
        </p>
      </section>
    </div>
  );
}
