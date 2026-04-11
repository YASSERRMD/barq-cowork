import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { workspacesApi, type Workspace } from "../lib/api";

export function WorkspacesPage() {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);

  const { data: workspaces = [], isLoading, error } = useQuery({
    queryKey: ["workspaces"],
    queryFn: workspacesApi.list,
    retry: 1,
  });

  const createMutation = useMutation({
    mutationFn: workspacesApi.create,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["workspaces"] });
      setShowForm(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: workspacesApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workspaces"] }),
  });

  return (
    <div className="p-6 space-y-5 max-w-3xl">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-white">Workspaces</h1>
        <button className="btn-primary" onClick={() => setShowForm((v) => !v)}>
          {showForm ? "Cancel" : "+ New Workspace"}
        </button>
      </div>

      {showForm && (
        <CreateWorkspaceForm
          onSubmit={(data) => createMutation.mutate(data)}
          loading={createMutation.isPending}
          error={createMutation.error?.message}
        />
      )}

      {isLoading && <p className="text-gray-400 text-sm">Loading…</p>}

      {error && (
        <p className="text-red-400 text-sm">
          Cannot reach backend — start <code className="font-mono">barq-coworkd</code> on :7331
        </p>
      )}

      {!isLoading && !error && workspaces.length === 0 && (
        <p className="text-gray-400 text-sm">No workspaces yet. Create one to get started.</p>
      )}

      <ul className="space-y-2">
        {workspaces.map((ws) => (
          <WorkspaceCard
            key={ws.id}
            workspace={ws}
            onDelete={() => deleteMutation.mutate(ws.id)}
          />
        ))}
      </ul>
    </div>
  );
}

function WorkspaceCard({
  workspace: ws,
  onDelete,
}: {
  workspace: Workspace;
  onDelete: () => void;
}) {
  return (
    <li className="card p-4 flex items-start justify-between gap-4 hover:border-gray-700 transition-colors">
      <div className="min-w-0">
        <Link
          to={`/workspaces/${ws.id}/projects`}
          className="text-white font-medium hover:text-barq-400 transition-colors"
        >
          {ws.name}
        </Link>
        {ws.description && (
          <p className="text-gray-400 text-xs mt-0.5 truncate">{ws.description}</p>
        )}
        {ws.root_path && (
          <p className="text-gray-600 text-xs font-mono mt-1 truncate">{ws.root_path}</p>
        )}
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <Link
          to={`/workspaces/${ws.id}/projects`}
          className="btn-ghost text-xs"
        >
          Open
        </Link>
        <button
          className="btn-ghost text-xs text-red-400 hover:text-red-300"
          onClick={onDelete}
        >
          Delete
        </button>
      </div>
    </li>
  );
}

function CreateWorkspaceForm({
  onSubmit,
  loading,
  error,
}: {
  onSubmit: (data: { name: string; description: string; root_path: string }) => void;
  loading: boolean;
  error?: string;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [rootPath, setRootPath] = useState("");

  return (
    <form
      className="card p-4 space-y-3"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({ name, description, root_path: rootPath });
      }}
    >
      <h2 className="text-sm font-semibold text-gray-300">New Workspace</h2>
      <div className="space-y-2">
        <input
          className="input"
          placeholder="Name *"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <input
          className="input"
          placeholder="Description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <input
          className="input font-mono text-xs"
          placeholder="Root path (e.g. /Users/you/projects/my-project)"
          value={rootPath}
          onChange={(e) => setRootPath(e.target.value)}
        />
      </div>
      {error && <p className="text-red-400 text-xs">{error}</p>}
      <button type="submit" className="btn-primary" disabled={loading}>
        {loading ? "Creating…" : "Create Workspace"}
      </button>
    </form>
  );
}
