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
          />
        ))}
      </ul>
    </div>
  );
}

function ProjectCard({
  project: p,
  onDelete,
}: {
  project: Project;
  onDelete: () => void;
}) {
  return (
    <li className="card p-4 flex items-start justify-between gap-4 hover:border-gray-700 transition-colors">
      <div className="min-w-0">
        <Link
          to={`/projects/${p.id}/tasks`}
          className="text-white font-medium hover:text-barq-400 transition-colors"
        >
          {p.name}
        </Link>
        {p.description && (
          <p className="text-gray-400 text-xs mt-0.5 truncate">{p.description}</p>
        )}
        {p.instructions && (
          <p className="text-gray-500 text-xs mt-1 truncate italic">
            Instructions: {p.instructions.slice(0, 80)}
            {p.instructions.length > 80 ? "…" : ""}
          </p>
        )}
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <Link to={`/projects/${p.id}/tasks`} className="btn-ghost text-xs">
          Tasks
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
        <textarea
          className="input resize-none"
          rows={3}
          placeholder="Project instructions (prepended to every task)"
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
        />
      </div>
      {error && <p className="text-red-400 text-xs">{error}</p>}
      <button type="submit" className="btn-primary" disabled={loading}>
        {loading ? "Creating…" : "Create Project"}
      </button>
    </form>
  );
}
