import { Navigate, Route, Routes } from "react-router";

import { Audit } from "./screens/Audit";
import { InstanceSettings } from "./screens/InstanceSettings";
import { InviteAccept } from "./screens/InviteAccept";
import { Login } from "./screens/Login";
import { Project } from "./screens/Project";
import { ProjectsList } from "./screens/ProjectsList";
import { Setup } from "./screens/Setup";
import { Shell } from "./screens/Shell";

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/setup" element={<Setup />} />
      <Route path="/invite" element={<InviteAccept />} />
      <Route element={<Shell />}>
        <Route index element={<ProjectsList />} />
        <Route path="/projects/:projectId/:tab?" element={<Project />} />
        <Route path="/audit" element={<Audit />} />
        <Route path="/settings" element={<InstanceSettings />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
