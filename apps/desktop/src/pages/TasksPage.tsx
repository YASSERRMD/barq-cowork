import { useState, useEffect } from "react";
import { useNavigate, useParams, useSearchParams, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Play, Plus, X, Trash2, ArrowRight } from "lucide-react";
import { tasksApi, projectsApi, templatesApi, type Task, type TaskTemplate } from "../lib/api";
import { TopBar } from "../components/TopBar";
import { EmptyState, SkeletonCard } from "../components/ui";
import { Breadcrumb } from "../components/Breadcrumb";

const STATUS_BADGE: Record<string, string> = {
  pending:   "badge-gray",
  planning:  "badge-blue",
  running:   "badge-yellow",
  completed: "badge-green",
  failed:    "badge-red",
  cancelled: "badge-gray",
};

const STATUS_FILTERS = ["all", "pending", "planning", "running", "completed", "failed"] as const;
type StatusFilter = typeof STATUS_FILTERS[number];

export function TasksPage({ globalView }: { globalView?: boolean }) {
  const { projectId } = useParams<{ projectId: string }>();
  const [searchParams] = useSearchParams();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [showForm, setShowForm] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");

  const templateId = searchParams.get("template") ?? undefined;
  useEffect(() => {
    if (templateId) setShowForm(true);
  }, [templateId]);

  const { data: project } = useQuery({
    queryKey: ["projects", projectId],
    queryFn: () => projectsApi.get(projectId!),
    enabled: !!projectId,
  });

  const { data: tasks = [], isLoading, error } = useQuery({
    queryKey: ["tasks", projectId ?? "global"],
    queryFn: () => (projectId ? tasksApi.listByProject(projectId) : Promise.resolve<Task[]>([])),
    enabled: !!projectId || globalView,
    refetchInterval: 5000,
  });

  const createMutation = useMutation({
    mutationFn: tasksApi.create,
    onSuccess: (task) => {
      qc.invalidateQueries({ queryKey: ["tasks", projectId ?? "global"] });
      setShowForm(false);
      navigate(`/tasks/${task.id}/run`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: tasksApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tasks", projectId ?? "global"] }),
  });

  const filtered = statusFilter === "all" ? tasks : tasks.filter((t) => t.status === statusFilter);

  const title = globalView ? "All Runs" : (project?.name ?? "Tasks");

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title={title}
        subtitle={tasks.length > 0 ? `${tasks.length} task${tasks.length === 1 ? "" : "s"}` : undefined}
        actions={
          projectId && (
            <button className="btn-primary btn-sm" onClick={() => setShowForm((v) => !v)}>
              <Plus size={13} />
              New Task
            </button>
          )
        }
      />

      {/* Breadcrumb */}
      {projectId && project && (
        <div style={{ padding: "8px 20px", borderBottom: "1px solid #2a2a3a" }}>
          <Breadcrumb items={[
            { label: "Projects", to: "/projects" },
            { label: project.name },
          ]} />
        </div>
      )}

      {/* Create form */}
      {showForm && projectId && (
        <CreateTaskForm
          projectId={projectId}
          templateId={templateId}
          onSubmit={(data) => createMutation.mutate(data)}
          onCancel={() => setShowForm(false)}
          loading={createMutation.isPending}
          error={createMutation.error?.message}
        />
      )}

      {/* Status filter */}
      {tasks.length > 0 && (
        <div style={{ padding: "8px 20px", display: "flex", gap: 6, borderBottom: "1px solid #2a2a3a" }}>
          {STATUS_FILTERS.map((s) => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              style={{
                padding: "3px 10px",
                borderRadius: 4,
                fontSize: 12,
                fontWeight: 500,
                cursor: "pointer",
                border: "1px solid",
                transition: "all 120ms",
                background: statusFilter === s ? "#22222f" : "transparent",
                borderColor: statusFilter === s ? "#3a3a4e" : "transparent",
                color: statusFilter === s ? "#c4c4d0" : "#50505f",
              }}
            >
              {s.charAt(0).toUpperCase() + s.slice(1)}
            </button>
          ))}
        </div>
      )}

      {/* Task list */}
      <div style={{ flex: 1, overflowY: "auto", padding: "8px 0" }}>
        {!projectId && !globalView ? (
          <EmptyState
            icon={Play}
            title="Select a project"
            description="Choose a project from the sidebar to view and manage its tasks."
            action={
              <Link to="/projects" className="btn-primary">
                Go to Projects
              </Link>
            }
          />
        ) : isLoading ? (
          <div style={{ padding: "16px 20px", display: "grid", gap: 8 }}>
            {[1, 2, 3].map((i) => <SkeletonCard key={i} />)}
          </div>
        ) : error ? (
          <div style={{ padding: 20 }}>
            <p style={{ color: "#f87171", fontSize: 13 }}>Failed to load tasks.</p>
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={Play}
            title={statusFilter === "all" ? "No tasks yet" : `No ${statusFilter} tasks`}
            description={statusFilter === "all" ? "Create a task to start an AI agent run." : "Try a different filter."}
            action={
              statusFilter === "all" && projectId ? (
                <button className="btn-primary" onClick={() => setShowForm(true)}>
                  <Plus size={14} />
                  Create Task
                </button>
              ) : undefined
            }
          />
        ) : (
          <div>
            {filtered.map((task) => (
              <TaskRow
                key={task.id}
                task={task}
                onDelete={() => deleteMutation.mutate(task.id)}
                onClick={() => navigate(`/tasks/${task.id}/run`)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function TaskRow({
  task,
  onDelete,
  onClick,
}: {
  task: Task;
  onDelete: () => void;
  onClick: () => void;
}) {
  const [hovered, setHovered] = useState(false);

  return (
    <div
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 20px",
        cursor: "pointer",
        background: hovered ? "#16161f" : "transparent",
        borderBottom: "1px solid rgba(42,42,58,0.5)",
        transition: "background 120ms",
      }}
    >
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 2 }}>
          <span style={{ fontSize: 13, fontWeight: 600, color: "#e2e2e8", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {task.title}
          </span>
          <span className={STATUS_BADGE[task.status] ?? "badge-gray"}>{task.status}</span>
        </div>
        {task.description && (
          <div style={{ fontSize: 12, color: "#50505f", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {task.description}
          </div>
        )}
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 8, flexShrink: 0 }}>
        <span style={{ fontSize: 11, color: "#40404f" }}>
          {new Date(task.created_at).toLocaleDateString()}
        </span>
        {hovered && (
          <button
            className="btn-ghost btn-sm"
            style={{ color: "#f87171", padding: "3px 6px" }}
            onClick={(e) => { e.stopPropagation(); onDelete(); }}
          >
            <Trash2 size={12} />
          </button>
        )}
        <ArrowRight size={14} color={hovered ? "#818cf8" : "#3a3a4e"} />
      </div>
    </div>
  );
}

function CreateTaskForm({
  projectId,
  templateId,
  onSubmit,
  onCancel,
  loading,
  error,
}: {
  projectId: string;
  templateId?: string;
  onSubmit: (data: { project_id: string; title: string; description: string; provider_id?: string }) => void;
  onCancel: () => void;
  loading: boolean;
  error?: string;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [providerId, setProviderId] = useState("");

  const { data: templates = [] } = useQuery({
    queryKey: ["templates", projectId],
    queryFn: () => templatesApi.list(projectId),
    enabled: !!projectId,
  });

  const applyTemplate = (t: TaskTemplate) => {
    setTitle(t.title);
    setDescription(t.description);
    setProviderId(t.provider_id);
  };

  useEffect(() => {
    if (templateId && templates.length > 0) {
      const t = templates.find((tmpl) => tmpl.id === templateId);
      if (t) applyTemplate(t);
    }
  }, [templateId, templates.length]);

  return (
    <div style={{ background: "#16161f", borderBottom: "1px solid #2a2a3a", padding: "16px 20px" }}>
      <form onSubmit={(e) => {
        e.preventDefault();
        onSubmit({ project_id: projectId, title, description, provider_id: providerId || undefined });
      }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
          <span style={{ fontSize: 13, fontWeight: 600, color: "#c4c4d0" }}>New Task</span>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            {templates.length > 0 && (
              <select
                style={{
                  background: "#1c1c27", border: "1px solid #2a2a3a", borderRadius: 4,
                  fontSize: 12, color: "#c4c4d0", padding: "3px 8px", outline: "none",
                }}
                defaultValue=""
                onChange={(e) => {
                  const t = templates.find((tmpl) => tmpl.id === e.target.value);
                  if (t) applyTemplate(t);
                }}
              >
                <option value="" disabled>Load template...</option>
                {templates.map((t) => (
                  <option key={t.id} value={t.id}>{t.name}</option>
                ))}
              </select>
            )}
            <button type="button" className="btn-ghost btn-sm" onClick={onCancel}>
              <X size={13} />
            </button>
          </div>
        </div>
        <div style={{ display: "grid", gap: 10 }}>
          <input
            className="input"
            placeholder="Task title *"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            autoFocus
            required
          />
          <textarea
            className="input"
            placeholder="Describe what you want the agent to accomplish..."
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            style={{ minHeight: 80 }}
          />
          <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
            <button type="button" className="btn-secondary btn-sm" onClick={onCancel}>Cancel</button>
            <button type="submit" className="btn-primary btn-sm" disabled={loading || !title.trim()}>
              {loading ? "Creating..." : "Create & Run"}
            </button>
          </div>
        </div>
        {error && <p style={{ color: "#f87171", fontSize: 12, marginTop: 8 }}>{error}</p>}
      </form>
    </div>
  );
}
