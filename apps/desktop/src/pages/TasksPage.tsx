import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { tasksApi, projectsApi, type Task } from "../lib/api";
import clsx from "clsx";

const STATUS_BADGE: Record<string, string> = {
  pending:   "badge-gray",
  planning:  "badge-blue",
  running:   "badge-yellow",
  completed: "badge-green",
  failed:    "badge-red",
  cancelled: "badge-gray",
};

export function TasksPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);

  const { data: project } = useQuery({
    queryKey: ["projects", projectId],
    queryFn: () => projectsApi.get(projectId!),
    enabled: !!projectId,
  });

  const { data: tasks = [], isLoading, error } = useQuery({
    queryKey: ["tasks", projectId],
    queryFn: () => tasksApi.listByProject(projectId!),
    enabled: !!projectId,
    refetchInterval: 5000, // poll every 5s while tasks may be running
  });

  const createMutation = useMutation({
    mutationFn: tasksApi.create,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tasks", projectId] });
      setShowForm(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: tasksApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tasks", projectId] }),
  });

  // Show general tasks page if no projectId (top-level nav)
  if (!projectId) {
    return (
      <div className="p-6">
        <h1 className="text-xl font-semibold text-white mb-4">Tasks</h1>
        <p className="text-gray-400 text-sm">
          Select a project from{" "}
          <Link to="/" className="text-barq-400 hover:underline">
            Workspaces
          </Link>{" "}
          to view its tasks.
        </p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-5 max-w-3xl">
      {/* Breadcrumb */}
      <nav className="text-xs text-gray-500 flex items-center gap-1">
        <Link to="/" className="hover:text-gray-300">Workspaces</Link>
        <span>/</span>
        {project && (
          <>
            <Link
              to={`/workspaces/${project.workspace_id}/projects`}
              className="hover:text-gray-300"
            >
              Projects
            </Link>
            <span>/</span>
          </>
        )}
        <span className="text-gray-300">{project?.name ?? projectId}</span>
      </nav>

      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-white">Tasks</h1>
        <button className="btn-primary" onClick={() => setShowForm((v) => !v)}>
          {showForm ? "Cancel" : "+ New Task"}
        </button>
      </div>

      {showForm && (
        <CreateTaskForm
          projectId={projectId}
          onSubmit={(data) => createMutation.mutate(data)}
          loading={createMutation.isPending}
          error={createMutation.error?.message}
        />
      )}

      {isLoading && <p className="text-gray-400 text-sm">Loading…</p>}
      {error && <p className="text-red-400 text-sm">Failed to load tasks.</p>}

      {!isLoading && !error && tasks.length === 0 && (
        <p className="text-gray-400 text-sm">No tasks yet. Submit one to get started.</p>
      )}

      <ul className="space-y-2">
        {tasks.map((t) => (
          <TaskCard
            key={t.id}
            task={t}
            onDelete={() => deleteMutation.mutate(t.id)}
          />
        ))}
      </ul>
    </div>
  );
}

function TaskCard({
  task: t,
  onDelete,
}: {
  task: Task;
  onDelete: () => void;
}) {
  return (
    <li className="card p-4 space-y-2 hover:border-gray-700 transition-colors">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-white font-medium truncate">{t.title}</p>
          {t.description && (
            <p className="text-gray-400 text-xs mt-0.5 truncate">{t.description}</p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className={clsx(STATUS_BADGE[t.status] ?? "badge-gray")}>
            {t.status}
          </span>
          <button
            className="btn-ghost text-xs text-red-400 hover:text-red-300"
            onClick={onDelete}
          >
            Delete
          </button>
        </div>
      </div>
      <p className="text-gray-600 text-xs font-mono">
        {new Date(t.created_at).toLocaleString()}
      </p>
    </li>
  );
}

function CreateTaskForm({
  projectId,
  onSubmit,
  loading,
  error,
}: {
  projectId: string;
  onSubmit: (data: {
    project_id: string;
    title: string;
    description: string;
  }) => void;
  loading: boolean;
  error?: string;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");

  return (
    <form
      className="card p-4 space-y-3"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({ project_id: projectId, title, description });
      }}
    >
      <h2 className="text-sm font-semibold text-gray-300">New Task</h2>
      <div className="space-y-2">
        <input
          className="input"
          placeholder="Task title *"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
        />
        <textarea
          className="input resize-none"
          rows={4}
          placeholder="Describe what you want the agent to accomplish…"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      {error && <p className="text-red-400 text-xs">{error}</p>}
      <button type="submit" className="btn-primary" disabled={loading}>
        {loading ? "Submitting…" : "Submit Task"}
      </button>
    </form>
  );
}
