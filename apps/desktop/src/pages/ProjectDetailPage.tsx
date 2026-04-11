import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Edit3, Check, X, FileText, Layout, Calendar, Activity, Plus, Trash2 } from "lucide-react";
import {
  projectsApi,
  contextFilesApi,
  templatesApi,
} from "../lib/api";
import { TopBar } from "../components/TopBar";
import { Breadcrumb } from "../components/Breadcrumb";
import { EmptyState, Skeleton } from "../components/ui";

type Tab = "overview" | "context" | "templates" | "schedules";

export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [tab, setTab] = useState<Tab>("overview");
  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState({ name: "", description: "", instructions: "" });

  const { data: project, isLoading } = useQuery({
    queryKey: ["projects", projectId],
    queryFn: () => projectsApi.get(projectId!),
    enabled: !!projectId,
  });

  const updateMutation = useMutation({
    mutationFn: () => projectsApi.update(projectId!, editForm),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["projects", projectId] });
      setEditing(false);
    },
  });

  const startEdit = () => {
    if (!project) return;
    setEditForm({ name: project.name, description: project.description, instructions: project.instructions });
    setEditing(true);
  };

  if (isLoading) {
    return (
      <div style={{ padding: 24, display: "grid", gap: 12 }}>
        <Skeleton style={{ height: 20, width: "30%" }} />
        <Skeleton style={{ height: 14, width: "60%" }} />
      </div>
    );
  }

  if (!project) {
    return <div style={{ padding: 24, color: "#f87171", fontSize: 13 }}>Project not found.</div>;
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title={project.name}
        actions={
          !editing && (
            <button className="btn-ghost btn-sm" onClick={startEdit}>
              <Edit3 size={13} />
              Edit
            </button>
          )
        }
      />

      <div style={{ padding: "8px 20px", borderBottom: "1px solid #2a2a3a" }}>
        <Breadcrumb items={[{ label: "Projects", to: "/projects" }, { label: project.name }]} />
      </div>

      {/* Project header / edit form */}
      <div style={{ padding: "16px 20px", borderBottom: "1px solid #2a2a3a" }}>
        {editing ? (
          <form onSubmit={(e) => { e.preventDefault(); updateMutation.mutate(); }}>
            <div style={{ display: "grid", gap: 10, maxWidth: 600 }}>
              <input
                className="input"
                value={editForm.name}
                onChange={(e) => setEditForm((p) => ({ ...p, name: e.target.value }))}
                placeholder="Project name"
                required
                autoFocus
              />
              <input
                className="input"
                value={editForm.description}
                onChange={(e) => setEditForm((p) => ({ ...p, description: e.target.value }))}
                placeholder="Description"
              />
              <textarea
                className="input"
                value={editForm.instructions}
                onChange={(e) => setEditForm((p) => ({ ...p, instructions: e.target.value }))}
                placeholder="System instructions"
                style={{ minHeight: 80 }}
              />
              <div style={{ display: "flex", gap: 8 }}>
                <button type="submit" className="btn-primary btn-sm" disabled={updateMutation.isPending}>
                  <Check size={12} /> Save
                </button>
                <button type="button" className="btn-secondary btn-sm" onClick={() => setEditing(false)}>
                  <X size={12} /> Cancel
                </button>
              </div>
            </div>
          </form>
        ) : (
          <div>
            {project.description && (
              <p style={{ fontSize: 13, color: "#7a7a90", marginBottom: 4 }}>{project.description}</p>
            )}
            {project.instructions && (
              <div style={{ marginTop: 6 }}>
                <span style={{ fontSize: 10, fontWeight: 600, letterSpacing: "0.06em", textTransform: "uppercase", color: "#40404f" }}>
                  System instructions
                </span>
                <p style={{ fontSize: 12, color: "#50505f", marginTop: 3, lineHeight: 1.5 }}>
                  {project.instructions.length > 200 ? project.instructions.slice(0, 200) + "..." : project.instructions}
                </p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Tabs */}
      <div style={{ display: "flex", borderBottom: "1px solid #2a2a3a", padding: "0 20px" }}>
        {(["overview", "context", "templates", "schedules"] as Tab[]).map((t) => {
          const icons = { overview: Activity, context: FileText, templates: Layout, schedules: Calendar };
          const Icon = icons[t];
          return (
            <button
              key={t}
              onClick={() => setTab(t)}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 6,
                padding: "10px 12px",
                fontSize: 12,
                fontWeight: 500,
                background: "transparent",
                border: "none",
                borderBottom: `2px solid ${tab === t ? "#6366f1" : "transparent"}`,
                color: tab === t ? "#a5b4fc" : "#50505f",
                cursor: "pointer",
                transition: "all 150ms",
              }}
            >
              <Icon size={13} />
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          );
        })}
      </div>

      {/* Tab content */}
      <div style={{ flex: 1, overflowY: "auto" }}>
        {tab === "overview" && <OverviewTab projectId={projectId!} />}
        {tab === "context" && <ContextFilesTab projectId={projectId!} />}
        {tab === "templates" && <TemplatesTab projectId={projectId!} />}
        {tab === "schedules" && (
          <div style={{ padding: "16px 20px" }}>
            <button
              className="btn-primary btn-sm"
              onClick={() => navigate(`/schedules`)}
            >
              Manage Schedules
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function OverviewTab({ projectId }: { projectId: string }) {
  const navigate = useNavigate();
  const { data: tasks = [] } = useQuery({
    queryKey: ["tasks", projectId],
    queryFn: () => import("../lib/api").then(m => m.tasksApi.listByProject(projectId)),
    refetchInterval: 10000,
  });

  const recent = tasks.slice(0, 5);

  return (
    <div style={{ padding: "16px 20px" }}>
      <div style={{ marginBottom: 16, display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <span style={{ fontSize: 12, fontWeight: 600, color: "#7a7a90" }}>Recent Tasks</span>
        <button
          className="btn-ghost btn-sm"
          onClick={() => navigate(`/projects/${projectId}/tasks`)}
        >
          View all
        </button>
      </div>
      {recent.length === 0 ? (
        <EmptyState
          icon={Activity}
          title="No tasks yet"
          description="Create a task to start an AI agent run."
          action={
            <button className="btn-primary btn-sm" onClick={() => navigate(`/projects/${projectId}/tasks`)}>
              <Plus size={13} /> New Task
            </button>
          }
        />
      ) : (
        <div style={{ display: "grid", gap: 6 }}>
          {recent.map((t) => (
            <div
              key={t.id}
              className="card-hover"
              style={{ padding: "8px 12px", display: "flex", alignItems: "center", gap: 10 }}
              onClick={() => navigate(`/tasks/${t.id}/run`)}
            >
              <span style={{ fontSize: 13, color: "#c4c4d0", flex: 1 }}>{t.title}</span>
              <span className={`badge-${t.status === "completed" ? "green" : t.status === "failed" ? "red" : t.status === "running" ? "yellow" : "gray"}`}>
                {t.status}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ContextFilesTab({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: "", content: "", description: "" });

  const { data: files = [], isLoading } = useQuery({
    queryKey: ["context-files", projectId],
    queryFn: () => contextFilesApi.list(projectId),
  });

  const createMutation = useMutation({
    mutationFn: () => contextFilesApi.create(projectId, form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["context-files", projectId] });
      setShowForm(false);
      setForm({ name: "", content: "", description: "" });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: contextFilesApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["context-files", projectId] }),
  });

  return (
    <div style={{ padding: "16px 20px" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
        <span style={{ fontSize: 12, fontWeight: 600, color: "#7a7a90" }}>Context Files</span>
        <button className="btn-ghost btn-sm" onClick={() => setShowForm((v) => !v)}>
          <Plus size={13} /> Add File
        </button>
      </div>
      {showForm && (
        <div className="card" style={{ padding: "12px", marginBottom: 12 }}>
          <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(); }}>
            <div style={{ display: "grid", gap: 8 }}>
              <input className="input" placeholder="File name" value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} required autoFocus />
              <textarea className="input" placeholder="Content" value={form.content} onChange={(e) => setForm((p) => ({ ...p, content: e.target.value }))} style={{ minHeight: 80 }} />
              <input className="input" placeholder="Description (optional)" value={form.description} onChange={(e) => setForm((p) => ({ ...p, description: e.target.value }))} />
              <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
                <button type="button" className="btn-secondary btn-sm" onClick={() => setShowForm(false)}>Cancel</button>
                <button type="submit" className="btn-primary btn-sm" disabled={createMutation.isPending}>Save</button>
              </div>
            </div>
          </form>
        </div>
      )}
      {isLoading ? (
        <Skeleton style={{ height: 60 }} />
      ) : files.length === 0 ? (
        <EmptyState icon={FileText} title="No context files" description="Add files to provide context to your AI agent." />
      ) : (
        <div style={{ display: "grid", gap: 6 }}>
          {files.map((f) => (
            <div key={f.id} className="card" style={{ padding: "8px 12px", display: "flex", alignItems: "center", gap: 10 }}>
              <FileText size={13} color="#818cf8" />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13, color: "#c4c4d0" }}>{f.name}</div>
                {f.description && <div style={{ fontSize: 11, color: "#50505f" }}>{f.description}</div>}
              </div>
              <button className="btn-ghost btn-sm" style={{ color: "#f87171", padding: "2px 4px" }} onClick={() => deleteMutation.mutate(f.id)}>
                <Trash2 size={12} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function TemplatesTab({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: "", title: "", description: "" });

  const { data: templates = [], isLoading } = useQuery({
    queryKey: ["templates", projectId],
    queryFn: () => templatesApi.list(projectId),
  });

  const createMutation = useMutation({
    mutationFn: () => templatesApi.create(projectId, form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["templates", projectId] });
      setShowForm(false);
      setForm({ name: "", title: "", description: "" });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: templatesApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["templates", projectId] }),
  });

  return (
    <div style={{ padding: "16px 20px" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
        <span style={{ fontSize: 12, fontWeight: 600, color: "#7a7a90" }}>Task Templates</span>
        <button className="btn-ghost btn-sm" onClick={() => setShowForm((v) => !v)}>
          <Plus size={13} /> Add Template
        </button>
      </div>
      {showForm && (
        <div className="card" style={{ padding: "12px", marginBottom: 12 }}>
          <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(); }}>
            <div style={{ display: "grid", gap: 8 }}>
              <input className="input" placeholder="Template name" value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} required autoFocus />
              <input className="input" placeholder="Default task title" value={form.title} onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))} required />
              <textarea className="input" placeholder="Default task description" value={form.description} onChange={(e) => setForm((p) => ({ ...p, description: e.target.value }))} />
              <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
                <button type="button" className="btn-secondary btn-sm" onClick={() => setShowForm(false)}>Cancel</button>
                <button type="submit" className="btn-primary btn-sm" disabled={createMutation.isPending}>Save</button>
              </div>
            </div>
          </form>
        </div>
      )}
      {isLoading ? (
        <Skeleton style={{ height: 60 }} />
      ) : templates.length === 0 ? (
        <EmptyState icon={Layout} title="No templates" description="Create task templates for common workflows." />
      ) : (
        <div style={{ display: "grid", gap: 6 }}>
          {templates.map((t) => (
            <div key={t.id} className="card" style={{ padding: "8px 12px", display: "flex", alignItems: "center", gap: 10 }}>
              <Layout size={13} color="#818cf8" />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13, color: "#c4c4d0" }}>{t.name}</div>
                {t.description && <div style={{ fontSize: 11, color: "#50505f" }}>{t.description}</div>}
              </div>
              <button
                className="btn-ghost btn-sm"
                style={{ fontSize: 11 }}
                onClick={() => navigate(`/projects/${projectId}/tasks?template=${t.id}`)}
              >
                Use
              </button>
              <button className="btn-ghost btn-sm" style={{ color: "#f87171", padding: "2px 4px" }} onClick={() => deleteMutation.mutate(t.id)}>
                <Trash2 size={12} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
