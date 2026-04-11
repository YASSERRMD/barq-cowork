import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ShieldAlert, CheckCircle2, XCircle, RefreshCw } from "lucide-react";
import { toolsApi, type Approval } from "../lib/api";
import { TopBar } from "../components/TopBar";

export function ApprovalsPage() {
  const qc = useQueryClient();

  const { data: approvals = [], isLoading, error, refetch } = useQuery({
    queryKey: ["approvals"],
    queryFn: toolsApi.listApprovals,
    refetchInterval: 5_000,
    retry: 1,
  });

  const pending = approvals.filter((a) => a.status === "pending");

  const resolveMutation = useMutation({
    mutationFn: ({ id, resolution }: { id: string; resolution: "approved" | "rejected" }) =>
      toolsApi.resolveApproval(id, resolution),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["approvals"] });
      refetch();
    },
  });

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      <TopBar
        title="Approvals"
        subtitle={pending.length > 0 ? `${pending.length} pending` : "Queue empty"}
      />

      <div style={{ flex: 1, overflowY: "auto", padding: "16px 20px", display: "flex", flexDirection: "column", gap: 10 }}>
        {isLoading && (
          <div style={{ padding: "16px 0", display: "flex", flexDirection: "column", gap: 8 }}>
            {[1, 2].map((i) => (
              <div key={i} className="skeleton" style={{ height: 80, borderRadius: 8 }} />
            ))}
          </div>
        )}

        {error && (
          <div style={{
            display: "flex", alignItems: "center", gap: 10,
            padding: "12px 16px", borderRadius: 8,
            background: "var(--red-dim)", border: "1px solid rgba(248,113,113,0.2)",
          }}>
            <XCircle size={14} style={{ color: "var(--red)", flexShrink: 0 }} />
            <span style={{ fontSize: 13, color: "var(--red)" }}>Failed to load approvals.</span>
            <button className="btn-ghost btn-sm" style={{ marginLeft: "auto" }} onClick={() => refetch()}>
              <RefreshCw size={12} /> Retry
            </button>
          </div>
        )}

        {!isLoading && !error && pending.length === 0 && (
          <div className="empty-state">
            <div className="empty-state-icon">
              <CheckCircle2 size={20} color="var(--green)" />
            </div>
            <p style={{ fontSize: 13, fontWeight: 500, color: "var(--text-secondary)", margin: 0 }}>
              No pending approvals
            </p>
            <p style={{ fontSize: 12, color: "var(--text-muted)", margin: 0 }}>
              Approvals will appear here when a task needs permission to execute a tool.
            </p>
          </div>
        )}

        {pending.map((a) => (
          <ApprovalCard
            key={a.id}
            approval={a}
            onApprove={() => resolveMutation.mutate({ id: a.id, resolution: "approved" })}
            onReject={() => resolveMutation.mutate({ id: a.id, resolution: "rejected" })}
            loading={resolveMutation.isPending && resolveMutation.variables?.id === a.id}
          />
        ))}
      </div>
    </div>
  );
}

function ApprovalCard({
  approval: a,
  onApprove,
  onReject,
  loading,
}: {
  approval: Approval;
  onApprove: () => void;
  onReject: () => void;
  loading: boolean;
}) {
  let payloadParsed: unknown = null;
  try {
    payloadParsed = JSON.parse(a.payload);
  } catch {
    payloadParsed = a.payload;
  }

  return (
    <div style={{
      background: "var(--surface-2)",
      border: "1px solid rgba(251,191,36,0.2)",
      borderRadius: 10,
      padding: "14px 16px",
      display: "flex", flexDirection: "column", gap: 12,
    }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "flex-start", gap: 12 }}>
        <div style={{
          width: 34, height: 34, borderRadius: 8, flexShrink: 0,
          background: "var(--yellow-dim)", border: "1px solid rgba(251,191,36,0.2)",
          display: "flex", alignItems: "center", justifyContent: "center",
        }}>
          <ShieldAlert size={16} color="var(--yellow)" />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 3 }}>
            <span className="badge-yellow">approval required</span>
            <code style={{ fontSize: 11, color: "var(--text-muted)", fontFamily: "monospace" }}>
              {a.tool_name}
            </code>
          </div>
          <p style={{ fontSize: 13, fontWeight: 500, color: "var(--text-primary)", margin: 0 }}>
            {a.action}
          </p>
          <p style={{ fontSize: 11, color: "var(--text-faint)", marginTop: 2 }}>
            Task: <code style={{ fontFamily: "monospace" }}>{a.task_id?.slice(0, 8) || "—"}</code>
            {" · "}
            {new Date(a.created_at).toLocaleString()}
          </p>
        </div>
      </div>

      {/* Payload */}
      <div style={{
        background: "var(--bg)", border: "1px solid var(--border)",
        borderRadius: 6, overflow: "hidden",
      }}>
        <pre style={{
          margin: 0, padding: "10px 14px",
          fontSize: 11.5, color: "var(--text-secondary)",
          fontFamily: "JetBrains Mono, monospace",
          whiteSpace: "pre-wrap", wordBreak: "break-all",
          lineHeight: 1.6, maxHeight: 120, overflowY: "auto",
          userSelect: "text",
        }} className="selectable">
          {typeof payloadParsed === "object"
            ? JSON.stringify(payloadParsed, null, 2)
            : String(payloadParsed)}
        </pre>
      </div>

      {/* Actions */}
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <button
          className="btn-primary btn-sm"
          onClick={onApprove}
          disabled={loading}
          style={{ gap: 6 }}
        >
          <CheckCircle2 size={13} />
          Approve
        </button>
        <button
          className="btn-danger btn-sm"
          onClick={onReject}
          disabled={loading}
          style={{ gap: 6 }}
        >
          <XCircle size={13} />
          Reject
        </button>
        {loading && (
          <span style={{ fontSize: 12, color: "var(--text-faint)" }}>Resolving…</span>
        )}
      </div>
    </div>
  );
}
