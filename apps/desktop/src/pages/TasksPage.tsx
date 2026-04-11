export function TasksPage() {
  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-white">Tasks</h1>
        <button className="btn-primary">+ New Task</button>
      </div>
      <p className="text-gray-400 text-sm">No tasks yet.</p>
    </div>
  );
}
