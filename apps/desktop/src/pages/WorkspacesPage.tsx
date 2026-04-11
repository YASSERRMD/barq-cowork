export function WorkspacesPage() {
  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-white">Workspaces</h1>
        <button className="btn-primary">+ New Workspace</button>
      </div>
      <p className="text-gray-400 text-sm">
        No workspaces yet. Create one to get started.
      </p>
    </div>
  );
}
