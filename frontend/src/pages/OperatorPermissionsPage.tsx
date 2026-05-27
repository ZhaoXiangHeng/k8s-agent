import type { Permission } from "../domain/permission";
import { DataTable } from "../components/DataTable";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";
import { StatusBadge } from "../components/StatusBadge";

export function OperatorPermissionsPage({
  permissions,
  loading,
  error
}: {
  permissions: Permission[];
  loading: boolean;
  error: string;
}) {
  return (
    <div className="workspace">
      <header className="toolbar"><h2>我的权限</h2></header>
      {error ? <Notice type="error">{error}</Notice> : null}
      <section className="panel">
        {loading ? <p>正在加载...</p> : null}
        {!loading && permissions.length === 0 ? <EmptyState title="暂无授权权限" /> : null}
        {permissions.length > 0 ? (
          <DataTable>
            <thead><tr><th>Namespace</th><th>API Group</th><th>Resource</th><th>Verbs</th><th>状态</th></tr></thead>
            <tbody>
              {permissions.map((permission, index) => (
                <tr key={permission.id ?? `${permission.namespace}-${permission.resource}-${index}`}>
                  <td>{permission.namespace}</td>
                  <td>{permission.apiGroup || "-"}</td>
                  <td>{permission.resource}</td>
                  <td>{permission.verbs.join(", ")}</td>
                  <td><StatusBadge active={permission.enabled !== false} text={permission.enabled === false ? "停用" : "启用"} /></td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        ) : null}
      </section>
    </div>
  );
}
