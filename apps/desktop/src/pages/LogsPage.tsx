export function LogsPage() {
  return (
    <div className="p-6 h-full flex flex-col">
      <h1 className="text-xl font-semibold text-white mb-4">Logs</h1>
      <div className="flex-1 card p-4 font-mono text-xs text-green-400 overflow-y-auto selectable">
        <p className="text-gray-500">[barq-cowork] Ready. Waiting for events…</p>
      </div>
    </div>
  );
}
