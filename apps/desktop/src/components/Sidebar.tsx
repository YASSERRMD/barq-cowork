import { NavLink } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Home,
  Play,
  Sparkles,
  Plug,
  Puzzle,
  FileText,
  FolderOpen,
  Calendar,
  Settings,
  CheckCircle,
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
      style={{
        width: "var(--sidebar-w)",
        minWidth: "var(--sidebar-w)",
        background: "var(--surface-1)",
        borderRight: "1px solid var(--border)",
        display: "flex",
        flexDirection: "column",
        minHeight: 0,
        flexShrink: 0,
      }}
    >
      {/* Brand */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
          padding: "0 14px",
          height: "var(--topbar-h)",
          borderBottom: "1px solid var(--border)",
          flexShrink: 0,
        }}
      >
        <div
          style={{
            width: 26,
            height: 26,
            borderRadius: 7,
            background: "linear-gradient(135deg, #f97316 0%, #ea6c0a 100%)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
            boxShadow: "0 2px 8px var(--accent-glow)",
          }}
        >
          <Zap size={13} color="#fff" strokeWidth={2.5} />
        </div>
        <span
          style={{
            fontSize: 13.5,
            fontWeight: 600,
            color: "var(--text-primary)",
            letterSpacing: "-0.02em",
          }}
        >
          Barq Cowork
        </span>
      </div>

      {/* Nav */}
      <nav
        style={{
          flex: 1,
          overflowY: "auto",
          padding: "8px 8px",
          display: "flex",
          flexDirection: "column",
          gap: 1,
        }}
      >
        {/* Primary */}
        <SidebarLink to="/" icon={Home} label="Home" end />

        <div style={{ height: 12 }} />
        <SectionLabel label="Work" />
        <SidebarLink to="/runs" icon={Play} label="Runs" />
        <SidebarLink to="/skills" icon={Sparkles} label="Skills" />
        <SidebarLink to="/connectors" icon={Plug} label="Connectors" />
        <SidebarLink to="/plugins" icon={Puzzle} label="Plugins" />

        <div style={{ height: 12 }} />
        <SectionLabel label="Library" />
        <SidebarLink to="/artifacts" icon={FileText} label="Artifacts" />
        <SidebarLink to="/projects" icon={FolderOpen} label="Projects" />
        <SidebarLink to="/schedules" icon={Calendar} label="Schedules" />

        <div style={{ height: 12 }} />
        <SectionLabel label="System" />
        <SidebarLink
          to="/approvals"
          icon={CheckCircle}
          label="Approvals"
          badge={pendingCount}
        />
        <SidebarLink to="/settings" icon={Settings} label="Settings" />
      </nav>

      {/* Status footer */}
      <StatusFooter />
    </aside>
  );
}

function SectionLabel({ label }: { label: string }) {
  return (
    <div
      style={{
        padding: "0 9px 4px",
        fontSize: 10,
        fontWeight: 600,
        letterSpacing: "0.07em",
        textTransform: "uppercase",
        color: "var(--text-faint)",
      }}
    >
      {label}
    </div>
  );
}

function SidebarLink({
  to,
  icon: Icon,
  label,
  badge = 0,
  end,
}: {
  to: string;
  icon: React.ElementType;
  label: string;
  badge?: number;
  end?: boolean;
}) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) => clsx("nav-item", isActive && "active")}
    >
      <Icon
        size={15}
        strokeWidth={1.75}
        style={{ flexShrink: 0, opacity: 0.85 }}
        className="nav-icon"
      />
      <span style={{ flex: 1, fontSize: 13 }}>{label}</span>
      {badge > 0 && (
        <span
          style={{
            background: "var(--accent)",
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
        borderTop: "1px solid var(--border)",
        padding: "10px 14px",
        display: "flex",
        alignItems: "center",
        gap: 8,
      }}
    >
      <div
        className={
          backendReachable ? "status-dot status-dot-green" : "status-dot status-dot-red"
        }
      />
      <span
        style={{
          fontSize: 11,
          color: "var(--text-faint)",
          flex: 1,
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
      >
        {backendReachable ? "Connected" : "Disconnected"}
      </span>
      {version && (
        <span style={{ fontSize: 11, color: "var(--text-faint)", flexShrink: 0 }}>
          v{version}
        </span>
      )}
    </div>
  );
}
