import { NavLink } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  FolderOpen,
  Play,
  Calendar,
  Plug,
  CheckCircle,
  FileText,
  Settings,
  Activity,
  Zap,
} from "lucide-react";
import clsx from "clsx";
import { useAppStore } from "../store/appStore";
import { toolsApi } from "../lib/api";

export function Sidebar() {
  const { backendReachable } = useAppStore();

  const { data: approvals = [] } = useQuery({
    queryKey: ["approvals"],
    queryFn: toolsApi.listApprovals,
    refetchInterval: 5000,
    retry: 1,
    enabled: backendReachable,
  });
  const pendingCount = approvals.filter((a) => a.status === "pending").length;

  return (
    <aside
      className="flex flex-col"
      style={{
        width: "var(--sidebar-w)",
        minWidth: "var(--sidebar-w)",
        background: "#16161f",
        borderRight: "1px solid #2a2a3a",
        minHeight: 0,
      }}
    >
      {/* Brand */}
      <div className="flex items-center gap-2.5 px-4 py-3" style={{ borderBottom: "1px solid #2a2a3a" }}>
        <div
          style={{
            width: 26,
            height: 26,
            borderRadius: 6,
            background: "#4f46e5",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
          }}
        >
          <Zap size={14} color="#fff" strokeWidth={2.5} />
        </div>
        <span style={{ fontSize: 13, fontWeight: 600, color: "#e2e2e8", letterSpacing: "-0.01em" }}>
          Barq Cowork
        </span>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-2 px-2 space-y-0.5">
        <NavSection label="Workspace">
          <SidebarLink to="/projects" icon={FolderOpen} label="Projects" />
          <SidebarLink to="/runs" icon={Play} label="Runs" />
          <SidebarLink to="/schedules" icon={Calendar} label="Schedules" />
        </NavSection>

        <NavSection label="System">
          <SidebarLink
            to="/approvals"
            icon={CheckCircle}
            label="Approvals"
            badge={pendingCount}
          />
          <SidebarLink to="/artifacts" icon={FileText} label="Artifacts" />
          <SidebarLink to="/connectors" icon={Plug} label="Connectors" />
          <SidebarLink to="/logs" icon={Activity} label="Logs" />
        </NavSection>

        <NavSection label="App">
          <SidebarLink to="/settings" icon={Settings} label="Settings" />
        </NavSection>
      </nav>

      {/* Status footer */}
      <StatusFooter />
    </aside>
  );
}

function NavSection({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="pb-1">
      <div
        className="px-3 pb-1 pt-2"
        style={{ fontSize: 10, fontWeight: 600, letterSpacing: "0.07em", textTransform: "uppercase", color: "#40404f" }}
      >
        {label}
      </div>
      {children}
    </div>
  );
}

function SidebarLink({
  to,
  icon: Icon,
  label,
  badge = 0,
}: {
  to: string;
  icon: React.ElementType;
  label: string;
  badge?: number;
}) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        clsx("nav-item", isActive && "active")
      }
    >
      <Icon size={15} strokeWidth={1.75} style={{ flexShrink: 0 }} />
      <span style={{ flex: 1 }}>{label}</span>
      {badge > 0 && (
        <span
          style={{
            background: "#4f46e5",
            color: "#fff",
            fontSize: 10,
            fontWeight: 700,
            borderRadius: 10,
            minWidth: 16,
            height: 16,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            padding: "0 4px",
            flexShrink: 0,
          }}
        >
          {badge > 99 ? "99+" : badge}
        </span>
      )}
    </NavLink>
  );
}

function StatusFooter() {
  const { backendReachable, version } = useAppStore();
  return (
    <div
      style={{
        borderTop: "1px solid #2a2a3a",
        padding: "10px 14px",
        display: "flex",
        alignItems: "center",
        gap: 8,
      }}
    >
      <div
        style={{
          width: 6,
          height: 6,
          borderRadius: "50%",
          background: backendReachable ? "#10b981" : "#ef4444",
          flexShrink: 0,
          boxShadow: backendReachable ? "0 0 6px rgba(16,185,129,0.5)" : "none",
        }}
      />
      <span style={{ fontSize: 11, color: "#50505f", flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
        {backendReachable ? "Connected" : "Disconnected"}
      </span>
      {version && (
        <span style={{ fontSize: 11, color: "#35354a", flexShrink: 0 }}>
          v{version}
        </span>
      )}
    </div>
  );
}
