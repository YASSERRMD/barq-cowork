import { NavLink } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import clsx from "clsx";
import { useAppStore } from "../store/appStore";
import { toolsApi } from "../lib/api";
import { checkHealth } from "../lib/tauri";

export function Sidebar() {
  // Poll pending approvals so the badge count stays fresh.
  const { data: approvals = [] } = useQuery({
    queryKey: ["approvals"],
    queryFn: toolsApi.listApprovals,
    refetchInterval: 5000,
    retry: 1,
  });
  const pendingCount = approvals.filter((a) => a.status === "pending").length;

  return (
    <aside
      className="flex flex-col bg-gray-900 border-r border-gray-800"
      style={{ width: "var(--sidebar-width)", minHeight: 0 }}
    >
      {/* Logo / brand */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800">
        <div className="w-7 h-7 rounded-md bg-barq-600 flex items-center justify-center text-white font-bold text-sm select-none">
          B
        </div>
        <span className="font-semibold text-white tracking-wide text-sm">
          Barq Cowork
        </span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-2 px-2 space-y-0.5">
        <SidebarLink to="/" icon="⬡" label="Workspaces" exact />
        <SidebarLink to="/tasks" icon="▶" label="Tasks" />
        <SidebarLink to="/artifacts" icon="⊞" label="Artifacts" />
        <SidebarLink
          to="/approvals"
          icon="✓"
          label="Approvals"
          badge={pendingCount}
        />
        <SidebarLink to="/logs" icon="≡" label="Logs" />
        <SidebarLink to="/settings" icon="⚙" label="Settings" />
      </nav>

      <BackendStatusFooter />
    </aside>
  );
}

// ─────────────────────────────────────────────
// SidebarLink
// ─────────────────────────────────────────────

function SidebarLink({
  to,
  icon,
  label,
  badge = 0,
  exact = false,
}: {
  to: string;
  icon: string;
  label: string;
  badge?: number;
  exact?: boolean;
}) {
  return (
    <NavLink
      to={to}
      end={exact}
      className={({ isActive }) =>
        clsx(
          "flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors",
          isActive
            ? "bg-barq-700/60 text-white"
            : "text-gray-400 hover:bg-gray-800 hover:text-gray-100"
        )
      }
    >
      <span className="text-base leading-none w-4 text-center shrink-0">
        {icon}
      </span>
      <span className="flex-1">{label}</span>
      {badge > 0 && (
        <span className="bg-orange-500 text-white text-[10px] font-bold rounded-full min-w-[18px] h-[18px] flex items-center justify-center px-1 shrink-0">
          {badge > 99 ? "99+" : badge}
        </span>
      )}
    </NavLink>
  );
}

// ─────────────────────────────────────────────
// BackendStatusFooter
// ─────────────────────────────────────────────

function BackendStatusFooter() {
  const { backendReachable, backendMessage, version, setBackendStatus } = useAppStore();
  const qc = useQueryClient();

  const handleReconnect = () => {
    checkHealth()
      .then((res) => {
        setBackendStatus(res.backend.reachable, res.backend.message);
        // Invalidate all queries so fresh data loads immediately.
        qc.invalidateQueries();
      })
      .catch(() => setBackendStatus(false, "backend unreachable"));
  };

  return (
    <div className="px-3 py-2 border-t border-gray-800 text-xs text-gray-500 space-y-1">
      <div className="flex items-center gap-1.5">
        <span
          className={clsx(
            "w-1.5 h-1.5 rounded-full shrink-0",
            backendReachable ? "bg-green-500" : "bg-red-500 animate-pulse"
          )}
        />
        <span className="truncate flex-1">{backendMessage}</span>
        {!backendReachable && (
          <button
            className="text-barq-400 hover:text-barq-300 shrink-0 transition-colors"
            title="Retry connection"
            onClick={handleReconnect}
          >
            ↻
          </button>
        )}
      </div>
      <div className="text-gray-600">v{version}</div>
    </div>
  );
}
