import type { AuditLog } from "../domain/audit";
import { DataTable } from "../components/DataTable";
import { EmptyState } from "../components/EmptyState";
import { StatusBadge } from "../components/StatusBadge";

export function AdminAuditPage({ logs }: { logs: AuditLog[] }) {
  return (
    <div className="workspace">
      <header className="toolbar"><h2>审计日志</h2></header>
      <section className="panel">
        {logs.length === 0 ? <EmptyState title="暂无审计日志" /> : (
          <DataTable>
            <thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>目标</th><th>Namespace</th><th>Resource</th><th>Verb</th><th>结果</th><th>原因</th></tr></thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id}>
                  <td>{log.createdAt}</td>
                  <td>{log.actorUserId}</td>
                  <td>{log.action}</td>
                  <td>{log.targetType}:{log.targetId}</td>
                  <td>{log.namespace || "-"}</td>
                  <td>{log.resource || "-"}</td>
                  <td>{log.verb || "-"}</td>
                  <td><StatusBadge active={log.allowed} text={log.allowed ? "允许" : "拒绝"} /></td>
                  <td>{log.reason}</td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        )}
      </section>
    </div>
  );
}
