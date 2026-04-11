import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  projectsApi,
  contextFilesApi,
  templatesApi,
  type ContextFile,
  type TaskTemplate,
} from "../lib/api";
import clsx from "clsx";

type Tab = "context" | "templates";

export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [tab, setTab] = useState<Tab>("context");

  const { data: project, isLoading } = useQuery({
    queryKey: ["projects", projectId],
    queryFn: () => projectsApi.get(projectId!),
    enabled: !!projectId,
  });

  if (isLoading) return <div className="p-6 text-gray-400 text-sm">Loading…</div>;
  if (!project) return <div className="p-6 text-red-400 text-sm">Project not found.</div>;

  return (
    <div className="p-6 space-y-5 max-w-3xl">
      {/* Breadcrumb */}
      <nav className="text-xs text-gray-500 flex items-center gap-1">
        <Link to="/" className="hover:text-gray-300">Workspaces</Link>
        <span>/</span>
        <Link
          to={`/workspaces/${project.workspace_id}/projects`}
          className="hover:text-gray-300"
        >
          Projects
        </Link>
        <span>/</span>
        <span className="text-gray-300 truncate max-w-[200px]">{project.name}</span>
      </nav>

      {/* Project header */}
      <div className="card p-5 space-y-2">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <h1 className="text-lg font-semibold text-white">{project.name}</h1>
            {project.description && (
              <p className="text-gray-400 text-sm mt-0.5">{project.description}</p>
            )}
          </div>
          <Link
            to={`/projects/${projectId}/tasks`}
            className="btn-primary text-sm shrink-0"
          >
            ▶ Tasks
          </Link>
        </div>
        {project.instructions && (
          <div className="pt-2 border-t border-gray-800">
            <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">
              Agent Instructions
            </p>
            <p className="text-gray-300 text-sm whitespace-pre-wrap leading-relaxed">
              {project.instructions}
            </p>
          </div>
        )}
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-gray-800 pb-0">
        {(["context", "templates"] as Tab[]).map((t) => (
          <button
            key={t}
            className={clsx(
              "px-4 py-2 text-sm font-medium capitalize transition-colors border-b-2 -mb-px",
              tab === t
                ? "border-barq-500 text-white"
                : "border-transparent text-gray-400 hover:text-gray-200"
            )}
            onClick={() => setTab(t)}
          >
            {t === "context" ? "Context Files" : "Task Templates"}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {tab === "context" && <ContextFilesPanel projectId={projectId!} />}
      {tab === "templates" && <TaskTemplatesPanel projectId={projectId!} />}
    </div>
  );
}

// ─────────────────────────────────────────────
// Context Files Panel
// ─────────────────────────────────────────────

function ContextFilesPanel({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  const { data: files = [], isLoading } = useQuery({
    queryKey: ["context-files", projectId],
    queryFn: () => contextFilesApi.list(projectId),
  });

  const createMutation = useMutation({
    mutationFn: (data: Parameters<typeof contextFilesApi.create>[1]) =>
      contextFilesApi.create(projectId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["context-files", projectId] });
      setShowForm(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: contextFilesApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["context-files", projectId] }),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof contextFilesApi.update>[1] }) =>
      contextFilesApi.update(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["context-files", projectId] });
      setEditingId(null);
    },
  });

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-xs text-gray-500">
          Context files are injected into every task planning prompt automatically.
        </p>
        <button className="btn-primary text-xs" onClick={() => setShowForm((v) => !v)}>
          {showForm ? "Cancel" : "+ Add Context File"}
        </button>
      </div>

      {showForm && (
        <ContextFileForm
          onSubmit={(data) => createMutation.mutate(data)}
          loading={createMutation.isPending}
          error={createMutation.error?.message}
          onCancel={() => setShowForm(false)}
        />
      )}

      {isLoading && <p className="text-gray-400 text-sm">Loading…</p>}

      {!isLoading && files.length === 0 && !showForm && (
        <div className="card p-6 text-center text-gray-500 text-sm space-y-1">
          <p className="text-xl">📎</p>
          <p>No context files yet.</p>
          <p className="text-xs text-gray-600">
            Add coding conventions, API docs, schema files, or any reference text
            you want the agent to always consider.
          </p>
        </div>
      )}

      <ul className="space-y-2">
        {files.map((cf) =>
          editingId === cf.id ? (
            <li key={cf.id} className="card p-4">
              <ContextFileForm
                initial={cf}
                onSubmit={(data) => updateMutation.mutate({ id: cf.id, data })}
                loading={updateMutation.isPending}
                error={updateMutation.error?.message}
                onCancel={() => setEditingId(null)}
              />
            </li>
          ) : (
            <ContextFileCard
              key={cf.id}
              file={cf}
              onEdit={() => setEditingId(cf.id)}
              onDelete={() => deleteMutation.mutate(cf.id)}
            />
          )
        )}
      </ul>
    </div>
  );
}

function ContextFileCard({
  file: cf,
  onEdit,
  onDelete,
}: {
  file: ContextFile;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  return (
    <li className="card p-4 space-y-2">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-0.5">
          <p className="text-white font-medium text-sm">{cf.name}</p>
          {cf.description && (
            <p className="text-gray-400 text-xs">{cf.description}</p>
          )}
          {cf.file_path && (
            <p className="text-gray-600 text-xs font-mono">{cf.file_path}</p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {cf.content && (
            <button
              className="btn-ghost text-xs text-gray-500"
              onClick={() => setExpanded((v) => !v)}
            >
              {expanded ? "Hide" : "Preview"}
            </button>
          )}
          <button className="btn-ghost text-xs text-barq-400" onClick={onEdit}>Edit</button>
          <button className="btn-ghost text-xs text-red-400" onClick={onDelete}>Delete</button>
        </div>
      </div>
      {expanded && cf.content && (
        <pre className="text-xs text-gray-400 bg-gray-950 rounded p-3 max-h-48 overflow-y-auto font-mono whitespace-pre-wrap break-all">
          {cf.content}
        </pre>
      )}
    </li>
  );
}

function ContextFileForm({
  initial,
  onSubmit,
  loading,
  error,
  onCancel,
}: {
  initial?: ContextFile;
  onSubmit: (data: { name: string; file_path: string; content: string; description: string }) => void;
  loading: boolean;
  error?: string;
  onCancel: () => void;
}) {
  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [filePath, setFilePath] = useState(initial?.file_path ?? "");
  const [content, setContent] = useState(initial?.content ?? "");

  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({ name, file_path: filePath, content, description });
      }}
    >
      <div className="grid grid-cols-2 gap-2">
        <div className="col-span-2">
          <input
            className="input"
            placeholder="Name * (e.g. Coding Conventions)"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <input
          className="input"
          placeholder="Description (optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <input
          className="input font-mono text-xs"
          placeholder="File path (optional, relative to workspace)"
          value={filePath}
          onChange={(e) => setFilePath(e.target.value)}
        />
      </div>
      <textarea
        className="input resize-y font-mono text-xs"
        rows={6}
        placeholder="Paste inline content here — this is injected directly into the planning prompt…"
        value={content}
        onChange={(e) => setContent(e.target.value)}
      />
      {error && <p className="text-red-400 text-xs">{error}</p>}
      <div className="flex gap-2">
        <button type="submit" className="btn-primary text-sm" disabled={loading}>
          {loading ? "Saving…" : initial ? "Save Changes" : "Add Context File"}
        </button>
        <button type="button" className="btn-ghost text-sm" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  );
}

// ─────────────────────────────────────────────
// Task Templates Panel
// ─────────────────────────────────────────────

function TaskTemplatesPanel({ projectId }: { projectId: string }) {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  const { data: templates = [], isLoading } = useQuery({
    queryKey: ["templates", projectId],
    queryFn: () => templatesApi.list(projectId),
  });

  const createMutation = useMutation({
    mutationFn: (data: Parameters<typeof templatesApi.create>[1]) =>
      templatesApi.create(projectId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["templates", projectId] });
      setShowForm(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: templatesApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["templates", projectId] }),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof templatesApi.update>[1] }) =>
      templatesApi.update(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["templates", projectId] });
      setEditingId(null);
    },
  });

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-xs text-gray-500">
          Templates pre-fill the task form so you don't repeat common task structures.
        </p>
        <button className="btn-primary text-xs" onClick={() => setShowForm((v) => !v)}>
          {showForm ? "Cancel" : "+ New Template"}
        </button>
      </div>

      {showForm && (
        <TemplateForm
          onSubmit={(data) => createMutation.mutate(data)}
          loading={createMutation.isPending}
          error={createMutation.error?.message}
          onCancel={() => setShowForm(false)}
        />
      )}

      {isLoading && <p className="text-gray-400 text-sm">Loading…</p>}

      {!isLoading && templates.length === 0 && !showForm && (
        <div className="card p-6 text-center text-gray-500 text-sm space-y-1">
          <p className="text-xl">📋</p>
          <p>No templates yet.</p>
          <p className="text-xs text-gray-600">
            Save common task structures like "Weekly report", "Code review",
            or "Data export" to load them instantly.
          </p>
        </div>
      )}

      <ul className="space-y-2">
        {templates.map((t) =>
          editingId === t.id ? (
            <li key={t.id} className="card p-4">
              <TemplateForm
                initial={t}
                onSubmit={(data) => updateMutation.mutate({ id: t.id, data })}
                loading={updateMutation.isPending}
                error={updateMutation.error?.message}
                onCancel={() => setEditingId(null)}
              />
            </li>
          ) : (
            <TemplateCard
              key={t.id}
              template={t}
              onEdit={() => setEditingId(t.id)}
              onDelete={() => deleteMutation.mutate(t.id)}
              projectId={projectId}
            />
          )
        )}
      </ul>
    </div>
  );
}

function TemplateCard({
  template: t,
  onEdit,
  onDelete,
  projectId,
}: {
  template: TaskTemplate;
  onEdit: () => void;
  onDelete: () => void;
  projectId: string;
}) {
  return (
    <li className="card p-4 space-y-1.5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-0.5">
          <p className="text-white font-medium text-sm">{t.name}</p>
          <p className="text-barq-300 text-xs truncate">
            "{t.title}"
          </p>
          {t.description && (
            <p className="text-gray-400 text-xs truncate">{t.description}</p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Link
            to={`/projects/${projectId}/tasks?template=${t.id}`}
            className="btn-primary text-xs"
          >
            Use
          </Link>
          <button className="btn-ghost text-xs text-barq-400" onClick={onEdit}>Edit</button>
          <button className="btn-ghost text-xs text-red-400" onClick={onDelete}>Delete</button>
        </div>
      </div>
    </li>
  );
}

function TemplateForm({
  initial,
  onSubmit,
  loading,
  error,
  onCancel,
}: {
  initial?: TaskTemplate;
  onSubmit: (data: { name: string; title: string; description: string; provider_id: string }) => void;
  loading: boolean;
  error?: string;
  onCancel: () => void;
}) {
  const [name, setName] = useState(initial?.name ?? "");
  const [title, setTitle] = useState(initial?.title ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [providerId, setProviderId] = useState(initial?.provider_id ?? "");

  return (
    <form
      className="space-y-2"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({ name, title, description, provider_id: providerId });
      }}
    >
      <input
        className="input"
        placeholder="Template name * (e.g. Weekly Summary)"
        value={name}
        onChange={(e) => setName(e.target.value)}
        required
      />
      <input
        className="input"
        placeholder="Task title * (pre-fills the title field)"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        required
      />
      <textarea
        className="input resize-y"
        rows={3}
        placeholder="Task description (optional pre-fill)"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
      />
      <input
        className="input text-xs font-mono"
        placeholder="Provider profile ID (optional override)"
        value={providerId}
        onChange={(e) => setProviderId(e.target.value)}
      />
      {error && <p className="text-red-400 text-xs">{error}</p>}
      <div className="flex gap-2">
        <button type="submit" className="btn-primary text-sm" disabled={loading}>
          {loading ? "Saving…" : initial ? "Save Changes" : "Create Template"}
        </button>
        <button type="button" className="btn-ghost text-sm" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  );
}
