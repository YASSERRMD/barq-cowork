import { useEffect } from "react";
import { Routes, Route } from "react-router-dom";
import { Sidebar } from "./components/Sidebar";
import { WorkspacesPage } from "./pages/WorkspacesPage";
import { ProjectsPage } from "./pages/ProjectsPage";
import { TasksPage } from "./pages/TasksPage";
import { TaskRunPage } from "./pages/TaskRunPage";
import { ProjectDetailPage } from "./pages/ProjectDetailPage";
import { ArtifactsPage } from "./pages/ArtifactsPage";
import { ApprovalsPage } from "./pages/ApprovalsPage";
import { LogsPage } from "./pages/LogsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { useAppStore } from "./store/appStore";
import { checkHealth, getAppVersion } from "./lib/tauri";

export default function App() {
  const { setBackendStatus, setVersion } = useAppStore();

  useEffect(() => {
    getAppVersion()
      .then(setVersion)
      .catch(() => {/* running outside Tauri (browser dev) */});

    const probe = () =>
      checkHealth()
        .then((res) => setBackendStatus(res.backend.reachable, res.backend.message))
        .catch(() => setBackendStatus(false, "backend unreachable"));

    probe();
    const id = setInterval(probe, 10_000);
    return () => clearInterval(id);
  }, [setBackendStatus, setVersion]);

  return (
    <div className="flex h-full">
      <Sidebar />
      <main className="flex-1 overflow-y-auto bg-gray-950">
        <Routes>
          {/* Workspace hierarchy */}
          <Route path="/"                                  element={<WorkspacesPage />} />
          <Route path="/workspaces/:workspaceId/projects" element={<ProjectsPage />} />
          <Route path="/projects/:projectId/tasks"        element={<TasksPage />} />

          {/* Task run / observation view */}
          <Route path="/tasks/:taskId/run" element={<TaskRunPage />} />

          {/* Project memory (context files + templates) */}
          <Route path="/projects/:projectId/memory" element={<ProjectDetailPage />} />

          {/* Top-level nav stubs */}
          <Route path="/tasks"      element={<TasksPage />} />
          <Route path="/artifacts"  element={<ArtifactsPage />} />
          <Route path="/approvals"  element={<ApprovalsPage />} />
          <Route path="/logs"       element={<LogsPage />} />
          <Route path="/settings"   element={<SettingsPage />} />
        </Routes>
      </main>
    </div>
  );
}
