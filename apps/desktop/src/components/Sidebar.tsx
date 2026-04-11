import { NavLink } from "react-router-dom";
import clsx from "clsx";

const NAV_ITEMS = [
  { to: "/",            label: "Workspaces",  icon: "⬡" },
  { to: "/tasks",       label: "Tasks",        icon: "▶" },
  { to: "/artifacts",   label: "Artifacts",    icon: "⊞" },
  { to: "/approvals",   label: "Approvals",    icon: "✓" },
  { to: "/logs",        label: "Logs",         icon: "≡" },
  { to: "/settings",   label: "Settings",      icon: "⚙" },
];

export function Sidebar() {
  return (
    <aside
      className="flex flex-col bg-gray-900 border-r border-gray-800"
      style={{ width: "var(--sidebar-width)", minHeight: 0 }}
    >
      {/* Logo / brand */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800">
        <div className="w-7 h-7 rounded-md bg-barq-600 flex items-center justify-center text-white font-bold text-sm">
          B
        </div>
        <span className="font-semibold text-white tracking-wide text-sm">
          Barq Cowork
        </span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-2 px-2 space-y-0.5">
        {NAV_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/"}
            className={({ isActive }) =>
              clsx(
                "flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors",
                isActive
                  ? "bg-barq-700/60 text-white"
                  : "text-gray-400 hover:bg-gray-800 hover:text-gray-100"
              )
            }
          >
            <span className="text-base leading-none w-4 text-center">
              {item.icon}
            </span>
            {item.label}
          </NavLink>
        ))}
      </nav>

      {/* Footer: backend status */}
      <BackendStatusFooter />
    </aside>
  );
}

function BackendStatusFooter() {
  // Imported here to avoid circular deps; store is tiny.
  const { backendReachable, backendMessage, version } =
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    require("../store/appStore").useAppStore();

  return (
    <div className="px-3 py-2 border-t border-gray-800 text-xs text-gray-500 space-y-0.5">
      <div className="flex items-center gap-1.5">
        <span
          className={clsx(
            "w-1.5 h-1.5 rounded-full",
            backendReachable ? "bg-green-500" : "bg-red-500"
          )}
        />
        <span className="truncate">{backendMessage}</span>
      </div>
      <div className="text-gray-600">v{version}</div>
    </div>
  );
}
