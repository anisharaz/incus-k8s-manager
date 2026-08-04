import { Navigate, Route, Routes } from "react-router";
import { AuthGate } from "./components/AuthGate";
import { ProtectedLayout } from "./components/ProtectedLayout";
import { Clusters } from "./pages/Clusters";
import { ClusterDetail } from "./pages/ClusterDetail";
import { Networks } from "./pages/Networks";
import { Users } from "./pages/Users";
import { Settings } from "./pages/Settings";
import { TerminalPage } from "./pages/TerminalPage";

function App() {
  return (
    <Routes>
      <Route path="/login" element={<AuthGate />} />
      {/* Deliberately outside ProtectedLayout — no sidebar/header, so the
          terminal gets the whole viewport. Not linked from the sidebar;
          only reached via a node's terminal button. */}
      <Route
        path="/terminal/:clusterId/:nodeId"
        element={<TerminalPage />}
      />
      <Route element={<ProtectedLayout />}>
        <Route path="/" element={<Navigate to="/clusters" replace />} />
        <Route path="/clusters" element={<Clusters />} />
        <Route path="/clusters/:clusterId" element={<ClusterDetail />} />
        <Route path="/networks" element={<Networks />} />
        <Route path="/users" element={<Users />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/clusters" replace />} />
      </Route>
    </Routes>
  );
}

export default App;
