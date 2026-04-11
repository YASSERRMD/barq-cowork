import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toolsApi, type Approval } from "../lib/api";
import clsx from "clsx";

export function ApprovalsPage() {
  const qc = useQueryClient();

  const { data: approvals = [], isLoading, error, refetch } = useQuery({
    queryKey: ["approvals"],
    queryFn: toolsApi.listApprovals,
    refetchInterval: 5000, // poll for new approvals while page is open
    retry: 1,
  });

  const resolveMutation = useMutation({
    mutationFn: ({ id, resolution }: { id: string; resolution: "approved" | "rejected" }) =>
      toolsApi.resolveApproval(id, resolution),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["approvals"] });
      refetch();
    },
  });

  return (
    <div className="p-6 space-y-5 max-w-3xl">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-white">Approvals Queue</h1>
        <span className="text-xs text-gray-500">Auto-refreshes every 5s</span>
      </div>

      {isLoading && <p className="text-gray-400 text-sm">Loading…</p>}
      {error && <p className="text-red-400 text-sm">Failed to load approvals.</p>}

      {!isLoading && approvals.length === 0 && (
        <div className="card p-6 text-center text-gray-500 text-sm">
          No pending approvals.
        </div>
      )}

      <ul className="space-y-3">
        {approvals.map((a) => (
          <ApprovalCard
            key={a.id}
            approval={a}
            onApprove={() => resolveMutation.mutate({ id: a.id, resolution: "approved" })}
            onReject={() => resolveMutation.mutate({ id: a.id, resolution: "rejected" })}
            loading={resolveMutation.isPending && resolveMutation.variables?.id === a.id}
          />
        ))}
      </ul>
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
    <li className="card p-4 space-y-3 border-yellow-800/50">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <div className="flex items-center gap-2">
            <span className="badge-yellow">approval required</span>
            <span className="text-xs text-gray-500 font-mono">{a.tool_name}</span>
          </div>
          <p className="text-white text-sm font-medium">{a.action}</p>
          <p className="text-gray-500 text-xs">
            Task: <span className="font-mono">{a.task_id || "—"}</span>
            {" · "}
            {new Date(a.created_at).toLocaleString()}
          </p>
        </div>
      </div>

      {/* Payload preview */}
      <div className="bg-gray-950 rounded p-3 text-xs font-mono text-gray-300 max-h-32 overflow-y-auto selectable">
        <pre className="whitespace-pre-wrap break-all">
          {typeof payloadParsed === "object"
            ? JSON.stringify(payloadParsed, null, 2)
            : String(payloadParsed)}
        </pre>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3">
        <button
          className={clsx("btn-primary", loading && "opacity-50")}
          onClick={onApprove}
          disabled={loading}
        >
          Approve
        </button>
        <button
          className={clsx("btn-danger", loading && "opacity-50")}
          onClick={onReject}
          disabled={loading}
        >
          Reject
        </button>
        {loading && <span className="text-xs text-gray-500">Resolving…</span>}
      </div>
    </li>
  );
}
