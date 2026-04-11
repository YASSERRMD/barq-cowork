import { useEffect, useState } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { Sidebar } from "./components/Sidebar";
import { CommandPalette } from "./components/CommandPalette";
import { HomePage } from "./pages/HomePage";
import { ProjectsPage } from "./pages/ProjectsPage";
import { ProjectDetailPage } from "./pages/ProjectDetailPage";
import { TasksPage } from "./pages/TasksPage";
import { TaskRunPage } from "./pages/TaskRunPage";
import { SchedulesPage } from "./pages/SchedulesPage";
import { ConnectorsPage } from "./pages/ConnectorsPage";
import { ArtifactsPage } from "./pages/ArtifactsPage";
import { ApprovalsPage } from "./pages/ApprovalsPage";
import { LogsPage } from "./pages/LogsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { SkillsPage } from "./pages/SkillsPage";
import { PluginsPage } from "./pages/PluginsPage";
import { useAppStore } from "./store/appStore";
import { checkHealth, getAppVersion } from "./lib/tauri";

export default function App() {
  const { setBackendStatus, setVersion } = useAppStore();
  const [paletteOpen, setPaletteOpen] = useState(false);

  useEffect(() => {
    getAppVersion()
      .then(setVersion)
      .catch(() => {});

    const probe = () =>
      checkHealth()
        .then((res) => setBackendStatus(res.backend.reachable, res.backend.message))
        .catch(() => setBackendStatus(false, "backend unreachable"));

    probe();
    const id = setInterval(probe, 15_000);
    return () => clearInterval(id);
  }, [setBackendStatus, setVersion]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setPaletteOpen((v) => !v);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  return (
    <div style={{ display: "flex", height: "100%", overflow: "hidden" }}>
      <Sidebar />
      <main
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
          background: "var(--bg)",
        }}
      >
        <Routes>
          {/* Primary entry point — direct task */}
          <Route path="/" element={<HomePage />} />

          {/* Runs */}
          <Route path="/runs" element={<TasksPage globalView />} />
          <Route path="/tasks/:taskId/run" element={<TaskRunPage />} />

          {/* Skills & Plugins */}
          <Route path="/skills" element={<SkillsPage />} />
          <Route path="/plugins" element={<PluginsPage />} />

          {/* Projects (optional org layer) */}
          <Route path="/projects" element={<ProjectsPage />} />
          <Route path="/projects/:projectId" element={<ProjectDetailPage />} />
          <Route path="/projects/:projectId/tasks" element={<TasksPage />} />

          {/* Schedules */}
          <Route path="/schedules" element={<SchedulesPage />} />
          <Route path="/projects/:projectId/schedules" element={<SchedulesPage />} />

          {/* Connectors */}
          <Route path="/connectors" element={<ConnectorsPage />} />

          {/* System */}
          <Route path="/artifacts" element={<ArtifactsPage />} />
          <Route path="/approvals" element={<ApprovalsPage />} />
          <Route path="/logs" element={<LogsPage />} />
          <Route path="/settings" element={<SettingsPage />} />

          {/* Legacy redirects */}
          <Route path="/workspaces" element={<Navigate to="/" replace />} />
          <Route path="/workspaces/:id/projects" element={<Navigate to="/projects" replace />} />
        </Routes>
      </main>

      {paletteOpen && (
        <CommandPalette onClose={() => setPaletteOpen(false)} />
      )}
    </div>
  );
}
