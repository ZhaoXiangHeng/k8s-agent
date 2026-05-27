import { useState } from "react";
import Sidebar from "./components/Sidebar";
import LoginPage from "./pages/LoginPage";
import OperatorConsole from "./pages/OperatorConsole";
import AdminConsole from "./pages/AdminConsole";
import type { User } from "./store/auth";

export default function App() {
  const [auth, setAuth] = useState<User | null>(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [view, setView] = useState<"operator" | "admin">("operator");

  if (!auth) {
    return <LoginPage onLogin={(user) => setAuth(user)} />;
  }

  return (
    <main className="shell">
      <Sidebar
        role={auth.role}
        activeView={view}
        collapsed={sidebarCollapsed}
        onNavigate={(v) => setView(v)}
        onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)}
      />

      <section className="content">
        {auth.role === "operator" || view === "operator" ? (
          <OperatorConsole />
        ) : (
          <AdminConsole />
        )}
      </section>
    </main>
  );
}
