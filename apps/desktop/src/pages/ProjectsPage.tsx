import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { projectsApi, workspacesApi, type Project } from "../lib/api";

export function ProjectsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);

  const { data: workspace } = useQuery({
    queryKey: ["workspaces", workspaceId],
    queryFn: () => workspacesApi.get(workspaceId!),
    enabled: !!workspaceId,
  });

  const { data: projects = [], isLoading, error } = useQuery({
    queryKey: ["projects", workspaceId],
    queryFn: () => projectsApi.listByWorkspace(workspaceId!),
    enabled: !!workspaceId,
  });

  const createMutation = useMutation({
    mutationFn: projectsApi.create,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["projects", workspaceId] });
      setShowForm(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: projectsApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", workspaceId] }),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Omit<Parameters<typeof projectsApi.update>[1], never> }) =>
      projectsApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", workspaceId] }),
  });

  return (
    <div className="p-6 space-y-5 max-w-3xl">
      {/* Breadcrumb */}
      <nav className="text-xs text-gray-500 flex items-center gap-1">
        <Link to="/" className="hover:text-gray-300">Workspaces</Link>
        <span>/</span>
        <span className="text-gray-300">{workspace?.name ?? workspaceId}</span>
      </nav>

      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-white">Projects</h1>
        <button className="btn-primary" onClick={() => setShowForm((v) => !v)}>
          {showForm ? "Cancel" : "+ New Project"}
        </button>
      </div>

      {showForm && (
        <CreateProjectForm
          workspaceId={workspaceId!}
          onSubmit={(data) => createMutation.mutate(data)}
          loading={createMutation.isPending}
          error={createMutation.error?.message}
        />
      )}

      {isLoading && <p className="text-gray-400 text-sm">Loading…</p>}
      {error && <p className="text-red-400 text-sm">Failed to load projects.</p>}

      {!isLoading && !error && projects.length === 0 && (
        <p className="text-gray-400 text-sm">No projects yet.</p>
      )}

      <ul className="space-y-2">
        {projects.map((p) => (
          <ProjectCard
            key={p.id}
            project={p}
            onDelete={() => deleteMutation.mutate(p.id)}
            onUpdate={(data) => updateMutation.mutate({ id: p.id, data })}
            updateLoading={
              updateMutation.isPending &&
              (updateMutation.variables as { id: string } | undefined)?.id === p.id
            }
          />
        ))}
      </ul>
    </div>
  );
}

// ─────────────────────────────────────────────
// ProjectCard with inline edit
// ─────────────────────────────────────────────

type UpdateData = { name: string; description?: string; instructions?: string };

function ProjectCard({
  project: p,
  onDelete,
  onUpdate,
  updateLoading,
}: {
  project: Project;
  onDelete: () => void;
  onUpdate: (data: UpdateData) => void;
  updateLoading: boolean;
}) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(p.name);
  const [description, setDescription] = useState(p.description || "");
  const [instructions, setInstructions] = useState(p.instructions || "");

  const openEdit = () => {
    // Reset to latest values when opening.
    setName(p.name);
    setDescription(p.description || "");
    setInstructions(p.instructions || "");
    setEditing(true);
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    onUpdate({ name, description, instructions });
    setEditing(false);
  };

  if (editing) {
    return (
      <li className="card p-4 space-y-3 border-barq-700/50">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-300">Edit Project</h3>
          <button
            className="btn-ghost text-xs text-gray-500"
            onClick={() => setEditing(false)}
          >
            Cancel
          </button>
        </div>
        <form className="space-y-2" onSubmit={handleSave}>
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
          <div>
            <label className="block text-xs text-gray-400 mb-1">
              Project Instructions
              <span className="text-gray-600 ml-1">
                — prepended to every task prompt
              </span>
            </label>
            <textarea
              className="input resize-y"
              rows={5}
              placeholder="Describe the project context, conventions, and goals the agent should follow…"
              value={instructions}
              onChange={(e) => setInstructions(e.target.value)}
            />
          </div>
          <div className="flex items-center gap-2 pt-1">
            <button type="submit" className="btn-primary text-sm" disabled={updateLoading}>
              {updateLoading ? "Saving…" : "Save Changes"}
            </button>
            <button
              type="button"
              className="btn-ghost text-sm"
              onClick={() => setEditing(false)}
            >
              Cancel
            </button>
          </div>
        </form>
      </li>
    );
  }

  return (
    <li className="card p-4 space-y-2 hover:border-gray-700 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-0.5">
          <Link
            to={`/projects/${p.id}/tasks`}
            className="text-white font-medium hover:text-barq-400 transition-colors block truncate"
          >
            {p.name}
          </Link>
          {p.description && (
            <p className="text-gray-400 text-xs truncate">{p.description}</p>
          )}
          {p.instructions ? (
            <p className="text-gray-500 text-xs mt-1 line-clamp-2 italic">
              {p.instructions}
            </p>
          ) : (
            <p className="text-gray-700 text-xs mt-1 italic">
              No instructions — click Edit to add context for the agent.
            </p>
          )}
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <Link to={`/projects/${p.id}/tasks`} className="btn-ghost text-xs">
            Tasks
          </Link>
          <Link to={`/projects/${p.id}/memory`} className="btn-ghost text-xs text-gray-400">
            Memory
          </Link>
          <button className="btn-ghost text-xs text-barq-400 hover:text-barq-300" onClick={openEdit}>
            Edit
          </button>
          <button
            className="btn-ghost text-xs text-red-400 hover:text-red-300"
            onClick={onDelete}
          >
            Delete
          </button>
        </div>
      </div>

      <div className="flex items-center gap-4 text-xs text-gray-600">
        <span>Created {new Date(p.created_at).toLocaleDateString()}</span>
        <span>Updated {new Date(p.updated_at).toLocaleDateString()}</span>
      </div>
    </li>
  );
}

// ─────────────────────────────────────────────
// Create form
// ─────────────────────────────────────────────

function CreateProjectForm({
  workspaceId,
  onSubmit,
  loading,
  error,
}: {
  workspaceId: string;
  onSubmit: (data: {
    workspace_id: string;
    name: string;
    description: string;
    instructions: string;
  }) => void;
  loading: boolean;
  error?: string;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [instructions, setInstructions] = useState("");

  return (
    <form
      className="card p-4 space-y-3"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({ workspace_id: workspaceId, name, description, instructions });
      }}
    >
      <h2 className="text-sm font-semibold text-gray-300">New Project</h2>
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
        <div>
          <label className="block text-xs text-gray-400 mb-1">
            Project Instructions
            <span className="text-gray-600 ml-1">— prepended to every task</span>
          </label>
          <textarea
            className="input resize-y"
            rows={3}
            placeholder="Describe conventions, goals, or context the agent should keep in mind…"
            value={instructions}
            onChange={(e) => setInstructions(e.target.value)}
          />
        </div>
      </div>
      {error && <p className="text-red-400 text-xs">{error}</p>}
      <button type="submit" className="btn-primary" disabled={loading}>
        {loading ? "Creating…" : "Create Project"}
      </button>
    </form>
  );
}
