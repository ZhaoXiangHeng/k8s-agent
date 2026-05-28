import { useMemo, useState } from "react";
import { useAdminData } from "./application/useAdminData";
import { useAuth } from "./application/useAuth";
import { useOperatorData } from "./application/useOperatorData";
import Sidebar, { type PageKey } from "./components/Sidebar";
import LoginPage from "./pages/LoginPage";
import { AuthCallbackPage } from "./pages/AuthCallbackPage";
import { OperatorChatPage } from "./pages/OperatorChatPage";
import { OperatorPermissionsPage } from "./pages/OperatorPermissionsPage";
import { OperatorModelsPage } from "./pages/OperatorModelsPage";
import AdminConsole from "./pages/AdminConsole";
import { AdminLlmConfigPage } from "./pages/AdminLlmConfigPage";
import { AdminAuditPage } from "./pages/AdminAuditPage";

export default function App() {
  const authState = useAuth();
  const [page, setPage] = useState<PageKey>("operator-chat");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const operator = useOperatorData(authState.auth, !!authState.user);
  const admin = useAdminData(authState.auth, authState.user?.role === "admin");
  const isCallback = window.location.pathname === "/auth/callback";

  const content = useMemo(() => {
    if (page === "operator-chat")
      return <OperatorChatPage auth={authState.auth} models={operator.models} />;
    if (page === "operator-permissions")
      return <OperatorPermissionsPage permissions={operator.permissions} loading={operator.loading} error={operator.error} role={authState.user?.role ?? "operator"} />;
    if (page === "operator-models")
      return <OperatorModelsPage models={operator.models} loading={operator.loading} error={operator.error} />;
    if (page === "admin-users-permissions")
      return (
        <AdminConsole
          users={admin.users}
          models={admin.models}
          providers={admin.providers}
          permissionsByUserId={admin.permissionsByUserId}
          modelBindingsByUserId={admin.modelBindingsByUserId}
          loading={admin.loading}
          error={admin.error}
          onCreateUser={admin.createUser}
          onDeleteUser={admin.deleteUser}
          onResetPassword={admin.resetPassword}
          onUpdatePermissions={admin.updatePermissions}
          onUpdateModelBindings={admin.updateModelBindings}
        />
      );
    if (page === "admin-llm")
      return (
        <AdminLlmConfigPage
          models={admin.models}
          providers={admin.providers}
          onCreateProvider={admin.createProvider}
          onUpdateProvider={admin.updateProvider}
          onCreateModel={admin.createModel}
          onUpdateModel={admin.updateModel}
          onDeleteModel={admin.deleteModel}
        />
      );
    return <AdminAuditPage logs={admin.auditLogs} />;
  }, [admin, authState.auth, operator, page]);

  if (isCallback)
    return <AuthCallbackPage onCallback={authState.handleCallback} />;
  if (authState.loading) return <div className="boot">正在加载...</div>;
  if (!authState.user)
    return (
      <LoginPage
        onLogin={(username, password) => authState.login(username, password)}
      />
    );

  return (
    <main className="shell">
      <Sidebar
        role={authState.user.role as "admin" | "operator"}
        activePage={page}
        collapsed={sidebarCollapsed}
        onNavigate={(p) => setPage(p)}
        onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)}
      />
      <section className="content">
        <header className="topbar">
          <div>
            <strong>{authState.user.displayName || authState.user.username}</strong>
            <span className="topbar-role">{authState.user.role}</span>
          </div>
          <button onClick={authState.logout} className="link-btn">切换账户</button>
        </header>
        {content}
      </section>
    </main>
  );
}
