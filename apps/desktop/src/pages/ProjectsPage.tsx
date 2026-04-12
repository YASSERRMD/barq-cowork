import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { FolderOpen, Plus, ChevronRight, X } from "lucide-react";
import { projectsApi, workspacesApi, type Project } from "../lib/api";
import { TopBar } from "../components/TopBar";
import { EmptyState, SkeletonCard } from "../components/ui";

export function ProjectsPage() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({ name: "", description: "", instructions: "" });

  // Load all projects directly
  const { data: projects = [], isLoading, error } = useQuery({
    queryKey: ["projects"],
    queryFn: projectsApi.list,
  });

  // Load workspaces to get default workspace id for creating projects
  const { data: workspaces = [] } = useQuery({
    queryKey: ["workspaces"],
    queryFn: workspacesApi.list,
  });

  const createMutation = useMutation({
    mutationFn: projectsApi.create,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["projects"] });
      setShowForm(false);
      setFormData({ name: "", description: "", instructions: "" });
    },
  });

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.name.trim()) return;
    const wsId = workspaces[0]?.id;
    if (!wsId) {
      // Auto-create a default workspace if none exists
      workspacesApi.create({ name: "Default", description: "Default workspace" }).then((ws) => {
        createMutation.mutate({
          workspace_id: ws.id,
          name: formData.name,
          description: formData.description,
          instructions: formData.instructions,
        });
      });
      return;
    }
    createMutation.mutate({
      workspace_id: wsId,
      name: formData.name,
      description: formData.description,
      instructions: formData.instructions,
    });
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title="Projects"
        subtitle={projects.length > 0 ? `${projects.length} project${projects.length === 1 ? "" : "s"}` : undefined}
        actions={
          <button className="btn-primary btn-sm" onClick={() => setShowForm((v) => !v)}>
            <Plus size={13} />
            New Project
          </button>
        }
      />

      {/* Inline create form */}
      {showForm && (
        <div
          style={{
            background: "var(--surface-2)",
            borderBottom: "1px solid var(--border)",
            padding: "16px 20px",
          }}
        >
          <form onSubmit={handleCreate}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
              <span style={{ fontSize: 13, fontWeight: 600, color: "var(--text-primary)" }}>New Project</span>
              <button type="button" className="btn-ghost btn-sm" onClick={() => setShowForm(false)}>
                <X size={13} />
              </button>
            </div>
            <div style={{ display: "grid", gap: 10 }}>
              <input
                className="input"
                placeholder="Project name *"
                value={formData.name}
                onChange={(e) => setFormData((p) => ({ ...p, name: e.target.value }))}
                autoFocus
                required
              />
              <input
                className="input"
                placeholder="Short description"
                value={formData.description}
                onChange={(e) => setFormData((p) => ({ ...p, description: e.target.value }))}
              />
              <textarea
                className="input"
                placeholder="System instructions (optional) — prepended to every task in this project"
                value={formData.instructions}
                onChange={(e) => setFormData((p) => ({ ...p, instructions: e.target.value }))}
                style={{ minHeight: 60 }}
              />
              <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
                <button type="button" className="btn-secondary btn-sm" onClick={() => setShowForm(false)}>
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-primary btn-sm"
                  disabled={createMutation.isPending || !formData.name.trim()}
                >
                  {createMutation.isPending ? "Creating..." : "Create Project"}
                </button>
              </div>
            </div>
            {createMutation.isError && (
              <p style={{ color: "var(--red)", fontSize: 12, marginTop: 8 }}>
                {(createMutation.error as Error).message}
              </p>
            )}
          </form>
        </div>
      )}

      {/* Project list */}
      <div style={{ flex: 1, overflowY: "auto", padding: "8px 0" }}>
        {isLoading ? (
          <div style={{ padding: "16px 20px", display: "grid", gap: 8 }}>
            {[1, 2, 3].map((i) => <SkeletonCard key={i} />)}
          </div>
        ) : error ? (
          <div style={{ padding: 20 }}>
            <p style={{ color: "var(--red)", fontSize: 13 }}>
              Failed to load projects. Is the backend running?
            </p>
          </div>
        ) : projects.length === 0 ? (
          <EmptyState
            icon={FolderOpen}
            title="No projects yet"
            description="Create a project to start organizing your AI tasks."
            action={
              <button className="btn-primary" onClick={() => setShowForm(true)}>
                <Plus size={14} />
                Create Project
              </button>
            }
          />
        ) : (
          <div>
            {projects.map((project) => (
              <ProjectRow
                key={project.id}
                project={project}
                onClick={() => navigate(`/projects/${project.id}/tasks`)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function ProjectRow({
  project,
  onClick,
}: {
  project: Project;
  onClick: () => void;
}) {
  return (
    <div
      onClick={onClick}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 20px",
        cursor: "pointer",
        transition: "background 120ms",
        borderBottom: "1px solid var(--border)",
      }}
      onMouseEnter={(e) => ((e.currentTarget as HTMLDivElement).style.background = "var(--surface-2)")}
      onMouseLeave={(e) => ((e.currentTarget as HTMLDivElement).style.background = "transparent")}
    >
      {/* Icon */}
      <div
        style={{
          width: 32,
          height: 32,
          borderRadius: 8,
          background: "var(--accent-dim)",
          border: "1px solid var(--accent-glow)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexShrink: 0,
        }}
      >
        <FolderOpen size={15} color="var(--accent)" strokeWidth={1.75} />
      </div>

      {/* Content */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: "var(--text-primary)" }}>{project.name}</div>
        {project.description && (
          <div style={{ fontSize: 12, color: "var(--text-muted)", marginTop: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {project.description}
          </div>
        )}
      </div>

      {/* Date */}
      <div style={{ fontSize: 11, color: "var(--text-faint)", flexShrink: 0, whiteSpace: "nowrap" }}>
        {new Date(project.created_at).toLocaleDateString()}
      </div>

      <ChevronRight size={14} color="var(--text-faint)" />
    </div>
  );
}
