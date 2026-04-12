import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Calendar, Plus, X, Trash2, ToggleLeft, ToggleRight, Clock } from "lucide-react";
import { schedulesApi, projectsApi, providersApi, type Schedule } from "../lib/api";
import { TopBar } from "../components/TopBar";
import { EmptyState, Skeleton } from "../components/ui";

const CRON_PRESETS = [
  { label: "Every hour", value: "0 * * * *" },
  { label: "Every day at 9am", value: "0 9 * * *" },
  { label: "Every Monday", value: "0 9 * * 1" },
  { label: "Every weekday", value: "0 9 * * 1-5" },
  { label: "Custom…", value: "__custom__" },
];

function humanCron(expr: string): string {
  const preset = CRON_PRESETS.find((p) => p.value === expr && p.value !== "__custom__");
  if (preset) return preset.label;
  return expr;
}

function formatDate(iso?: string): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

const labelStyle: React.CSSProperties = {
  display: "block",
  fontSize: 11,
  fontWeight: 600,
  color: "var(--text-muted)",
  marginBottom: 4,
  textTransform: "uppercase",
  letterSpacing: "0.05em",
};

export function SchedulesPage() {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const { data: schedules = [], isLoading } = useQuery({
    queryKey: ["schedules"],
    queryFn: schedulesApi.list,
    refetchInterval: 30_000,
  });

  const toggleMutation = useMutation({
    mutationFn: ({ schedule }: { schedule: Schedule }) =>
      schedulesApi.update(schedule.id, {
        name: schedule.name,
        description: schedule.description,
        cron_expr: schedule.cron_expr,
        task_title: schedule.task_title,
        task_desc: schedule.task_desc,
        provider_id: schedule.provider_id,
        enabled: !schedule.enabled,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["schedules"] }),
  });

  const deleteMutation = useMutation({
    mutationFn: schedulesApi.delete,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["schedules"] });
      setDeleteId(null);
    },
  });

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title="Schedules"
        subtitle={
          schedules.length > 0
            ? `${schedules.length} schedule${schedules.length === 1 ? "" : "s"}`
            : undefined
        }
        actions={
          <button className="btn-primary btn-sm" onClick={() => setShowForm((v) => !v)}>
            <Plus size={13} />
            New Schedule
          </button>
        }
      />

      {showForm && (
        <CreateScheduleForm
          onSuccess={() => {
            qc.invalidateQueries({ queryKey: ["schedules"] });
            setShowForm(false);
          }}
          onCancel={() => setShowForm(false)}
        />
      )}

      <div style={{ flex: 1, overflowY: "auto" }}>
        {isLoading ? (
          <div style={{ padding: "16px 20px", display: "grid", gap: 8 }}>
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} style={{ height: 56 }} />
            ))}
          </div>
        ) : schedules.length === 0 ? (
          <EmptyState
            icon={Calendar}
            title="No schedules yet"
            description="Create a schedule to run tasks automatically on a cron expression."
            action={
              <button className="btn-primary btn-sm" onClick={() => setShowForm(true)}>
                <Plus size={13} /> New Schedule
              </button>
            }
          />
        ) : (
          <div>
            {/* Header row */}
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "1fr 140px 160px 140px 140px 72px",
                padding: "6px 20px",
                borderBottom: "1px solid var(--border)",
                fontSize: 11,
                fontWeight: 600,
                color: "var(--text-faint)",
                textTransform: "uppercase",
                letterSpacing: "0.06em",
              }}
            >
              <span>Name / Task</span>
              <span>Project</span>
              <span>Schedule</span>
              <span>Last Run</span>
              <span>Next Run</span>
              <span style={{ textAlign: "right" }}>Actions</span>
            </div>

            {schedules.map((s) => (
              <ScheduleRow
                key={s.id}
                schedule={s}
                onToggle={() => toggleMutation.mutate({ schedule: s })}
                onDelete={() => setDeleteId(s.id)}
                toggling={toggleMutation.isPending}
              />
            ))}
          </div>
        )}
      </div>

      {/* Delete confirmation modal */}
      {deleteId && (
        <div
          style={{
            position: "fixed",
            inset: 0,
            background: "rgba(0,0,0,0.4)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 100,
          }}
          onClick={() => setDeleteId(null)}
        >
          <div
            className="card"
            style={{ padding: 20, maxWidth: 360, width: "100%", margin: "0 20px" }}
            onClick={(e) => e.stopPropagation()}
          >
            <p style={{ fontSize: 13, color: "var(--text-primary)", marginBottom: 4 }}>
              Delete this schedule?
            </p>
            <p style={{ fontSize: 12, color: "var(--text-muted)", marginBottom: 16 }}>
              This cannot be undone. No future runs will be triggered.
            </p>
            <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
              <button className="btn-secondary btn-sm" onClick={() => setDeleteId(null)}>
                Cancel
              </button>
              <button
                className="btn-danger btn-sm"
                disabled={deleteMutation.isPending}
                onClick={() => deleteMutation.mutate(deleteId)}
              >
                {deleteMutation.isPending ? "Deleting…" : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function ScheduleRow({
  schedule,
  onToggle,
  onDelete,
  toggling,
}: {
  schedule: Schedule;
  onToggle: () => void;
  onDelete: () => void;
  toggling: boolean;
}) {
  const { data: projects = [] } = useQuery({
    queryKey: ["projects"],
    queryFn: projectsApi.list,
    staleTime: 60_000,
  });

  const project = projects.find((p) => p.id === schedule.project_id);

  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "1fr 140px 160px 140px 140px 72px",
        padding: "10px 20px",
        borderBottom: "1px solid var(--border)",
        alignItems: "center",
        transition: "background 120ms",
      }}
      onMouseEnter={(e) =>
        ((e.currentTarget as HTMLDivElement).style.background = "var(--surface-2)")
      }
      onMouseLeave={(e) =>
        ((e.currentTarget as HTMLDivElement).style.background = "transparent")
      }
    >
      {/* Name / Task */}
      <div style={{ minWidth: 0, paddingRight: 12 }}>
        <div
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: "var(--text-primary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {schedule.name}
        </div>
        <div
          style={{
            fontSize: 11,
            color: "var(--text-muted)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {schedule.task_title}
        </div>
      </div>

      {/* Project */}
      <div
        style={{
          fontSize: 12,
          color: "var(--text-secondary)",
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
      >
        {project?.name ?? "—"}
      </div>

      {/* Schedule */}
      <div style={{ display: "flex", alignItems: "center", gap: 5 }}>
        <Clock size={11} color="var(--text-faint)" />
        <span
          style={{
            fontSize: 12,
            color: "var(--text-secondary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {humanCron(schedule.cron_expr)}
        </span>
      </div>

      {/* Last run */}
      <span style={{ fontSize: 11, color: "var(--text-faint)" }}>
        {formatDate(schedule.last_run_at)}
      </span>

      {/* Next run */}
      <span
        style={{
          fontSize: 11,
          color: schedule.enabled ? "var(--accent)" : "var(--text-faint)",
        }}
      >
        {schedule.enabled ? formatDate(schedule.next_run_at) : "Paused"}
      </span>

      {/* Actions */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 4,
          justifyContent: "flex-end",
        }}
      >
        <button
          className="btn-ghost btn-sm"
          style={{
            padding: "2px 4px",
            color: schedule.enabled ? "var(--accent)" : "var(--text-faint)",
          }}
          onClick={onToggle}
          disabled={toggling}
          title={schedule.enabled ? "Pause schedule" : "Enable schedule"}
        >
          {schedule.enabled ? (
            <ToggleRight size={16} />
          ) : (
            <ToggleLeft size={16} />
          )}
        </button>
        <button
          className="btn-ghost btn-sm"
          style={{ padding: "2px 4px", color: "var(--red)" }}
          onClick={onDelete}
          title="Delete schedule"
        >
          <Trash2 size={12} />
        </button>
      </div>
    </div>
  );
}

function CreateScheduleForm({
  onSuccess,
  onCancel,
}: {
  onSuccess: () => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState({
    project_id: "",
    name: "",
    description: "",
    cron_preset: "0 9 * * *",
    custom_cron: "",
    is_custom: false,
    task_title: "",
    task_desc: "",
    provider_id: "",
    enabled: true,
  });
  const [error, setError] = useState<string | null>(null);

  const { data: projects = [] } = useQuery({
    queryKey: ["projects"],
    queryFn: projectsApi.list,
  });

  const { data: providers = [] } = useQuery({
    queryKey: ["provider-profiles"],
    queryFn: providersApi.listProfiles,
  });

  const createMutation = useMutation({
    mutationFn: () =>
      schedulesApi.create({
        project_id: form.project_id,
        name: form.name,
        description: form.description || undefined,
        cron_expr: form.is_custom ? form.custom_cron : form.cron_preset,
        task_title: form.task_title,
        task_desc: form.task_desc || undefined,
        provider_id: form.provider_id || undefined,
        enabled: form.enabled,
      }),
    onSuccess: () => {
      setError(null);
      onSuccess();
    },
    onError: (e: Error) => setError(e.message),
  });

  const handlePreset = (value: string) => {
    if (value === "__custom__") {
      setForm((p) => ({ ...p, is_custom: true }));
    } else {
      setForm((p) => ({ ...p, cron_preset: value, is_custom: false }));
    }
  };

  const activeCron = form.is_custom ? form.custom_cron : form.cron_preset;

  return (
    <div
      style={{
        background: "var(--surface-2)",
        borderBottom: "1px solid var(--border)",
        padding: "16px 20px",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 14,
        }}
      >
        <span style={{ fontSize: 13, fontWeight: 600, color: "var(--text-primary)" }}>
          New Schedule
        </span>
        <button className="btn-ghost btn-sm" onClick={onCancel}>
          <X size={13} />
        </button>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (!form.project_id) {
            setError("Please select a project.");
            return;
          }
          if (!activeCron.trim()) {
            setError("Cron expression is required.");
            return;
          }
          setError(null);
          createMutation.mutate();
        }}
      >
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            gap: 10,
            maxWidth: 640,
          }}
        >
          {/* Name */}
          <div style={{ gridColumn: "1 / -1" }}>
            <label style={labelStyle}>Schedule name *</label>
            <input
              className="input"
              placeholder="e.g. Daily standup report"
              value={form.name}
              onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
              required
              autoFocus
            />
          </div>

          {/* Project */}
          <div>
            <label style={labelStyle}>Project *</label>
            <select
              className="input"
              value={form.project_id}
              onChange={(e) =>
                setForm((p) => ({ ...p, project_id: e.target.value }))
              }
              required
            >
              <option value="" disabled>
                Select project…
              </option>
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>

          {/* Provider */}
          <div>
            <label style={labelStyle}>Provider (optional)</label>
            <select
              className="input"
              value={form.provider_id}
              onChange={(e) =>
                setForm((p) => ({ ...p, provider_id: e.target.value }))
              }
            >
              <option value="">Default provider</option>
              {providers.map((pr) => (
                <option key={pr.id} value={pr.id}>
                  {pr.name}
                </option>
              ))}
            </select>
          </div>

          {/* Cron preset */}
          <div>
            <label style={labelStyle}>Frequency</label>
            <select
              className="input"
              value={form.is_custom ? "__custom__" : form.cron_preset}
              onChange={(e) => handlePreset(e.target.value)}
            >
              {CRON_PRESETS.map((p) => (
                <option key={p.label} value={p.value}>
                  {p.label}
                </option>
              ))}
            </select>
          </div>

          {/* Custom / display cron */}
          <div>
            <label style={labelStyle}>
              Cron expression
              {!form.is_custom && (
                <span style={{ color: "var(--text-faint)", marginLeft: 4 }}>
                  ({activeCron})
                </span>
              )}
            </label>
            <input
              className="input"
              placeholder="0 9 * * *"
              value={form.is_custom ? form.custom_cron : form.cron_preset}
              onChange={(e) => {
                if (form.is_custom) {
                  setForm((p) => ({ ...p, custom_cron: e.target.value }));
                }
              }}
              readOnly={!form.is_custom}
              style={{ opacity: form.is_custom ? 1 : 0.5 }}
            />
          </div>

          {/* Task title */}
          <div style={{ gridColumn: "1 / -1" }}>
            <label style={labelStyle}>Task title *</label>
            <input
              className="input"
              placeholder="What should the agent do?"
              value={form.task_title}
              onChange={(e) =>
                setForm((p) => ({ ...p, task_title: e.target.value }))
              }
              required
            />
          </div>

          {/* Task description */}
          <div style={{ gridColumn: "1 / -1" }}>
            <label style={labelStyle}>Task description</label>
            <textarea
              className="input"
              placeholder="Detailed instructions for the agent…"
              value={form.task_desc}
              onChange={(e) =>
                setForm((p) => ({ ...p, task_desc: e.target.value }))
              }
              style={{ minHeight: 72 }}
            />
          </div>

          {/* Schedule description */}
          <div style={{ gridColumn: "1 / -1" }}>
            <label style={labelStyle}>Notes (optional)</label>
            <input
              className="input"
              placeholder="Notes about this schedule"
              value={form.description}
              onChange={(e) =>
                setForm((p) => ({ ...p, description: e.target.value }))
              }
            />
          </div>
        </div>

        {error && (
          <p style={{ color: "var(--red)", fontSize: 12, marginTop: 10 }}>
            {error}
          </p>
        )}

        <div
          style={{
            display: "flex",
            gap: 8,
            justifyContent: "flex-end",
            marginTop: 14,
          }}
        >
          <button
            type="button"
            className="btn-secondary btn-sm"
            onClick={onCancel}
          >
            Cancel
          </button>
          <button
            type="submit"
            className="btn-primary btn-sm"
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? "Creating…" : "Create Schedule"}
          </button>
        </div>
      </form>
    </div>
  );
}
