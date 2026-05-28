import { useEffect, useMemo, useState } from "react";
import { Notice } from "../components/Notice";
import type { PermissionFormRow, UpdatePermissionsRequest } from "../domain/permission";
import { buildPermissionPayload } from "../domain/permission";
import type { User } from "../domain/user";

const initialRow: PermissionFormRow = {
  namespace: "dev",
  apiGroup: "",
  resource: "pods",
  verbsText: "get,list,watch",
};

type PermissionsManagementProps = {
  users: User[];
  permissionsByUserId: Record<string, UpdatePermissionsRequest["permissions"]>;
  onUpdatePermissions: (userId: string, body: UpdatePermissionsRequest) => Promise<void>;
  preselectedUserId?: string;
};

function rowsFromPermissions(permissions: UpdatePermissionsRequest["permissions"] | undefined): PermissionFormRow[] {
  if (!permissions || permissions.length === 0) return [initialRow];
  return permissions.map((permission) => ({
    namespace: permission.namespace,
    apiGroup: permission.apiGroup,
    resource: permission.resource,
    verbsText: permission.verbs.join(","),
  }));
}

export default function PermissionsManagement({
  users,
  permissionsByUserId,
  onUpdatePermissions,
  preselectedUserId,
}: PermissionsManagementProps) {
  const [selectedUserId, setSelectedUserId] = useState(preselectedUserId ?? (users[0]?.id || ""));
  const selectedUser = users.find((u) => u.id === selectedUserId);
  const isAdmin = selectedUser?.role === "admin";
  const selectedPermissions = useMemo(
    () => permissionsByUserId[selectedUserId],
    [permissionsByUserId, selectedUserId]
  );
  const [rows, setRows] = useState<PermissionFormRow[]>(() => rowsFromPermissions(selectedPermissions));
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (preselectedUserId) {
      setSelectedUserId(preselectedUserId);
    }
  }, [preselectedUserId]);

  useEffect(() => {
    if (!selectedUserId && users[0]?.id) {
      setSelectedUserId(users[0].id);
    }
  }, [selectedUserId, users]);

  useEffect(() => {
    setRows(rowsFromPermissions(selectedPermissions));
  }, [selectedPermissions]);

  const selectUser = (userId: string) => {
    setSelectedUserId(userId);
    setMessage("");
    setError("");
  };

  const updateRow = (index: number, field: keyof PermissionFormRow, value: string) => {
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: value } : r)));
  };

  const addRow = () => setRows([...rows, initialRow]);
  const removeRow = (index: number) => setRows(rows.filter((_, i) => i !== index));

  const handleSave = async () => {
    if (!selectedUserId) return;
    setSubmitting(true);
    setMessage("");
    setError("");
    try {
      await onUpdatePermissions(selectedUserId, buildPermissionPayload(rows));
      setMessage("权限已保存");
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存权限失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Admin</p>
          <h2>权限管理</h2>
        </div>
      </header>

      {error ? <Notice type="error">{error}</Notice> : null}
      {message ? <Notice type="info">{message}</Notice> : null}

      <div className="user-selector">
        <label>当前用户：</label>
        <select value={selectedUserId} onChange={(e) => selectUser(e.target.value)}>
          {users.map((u) => (
            <option key={u.id} value={u.id}>
              {u.username} ({u.role === "admin" ? "管理员" : "操作员"})
            </option>
          ))}
        </select>
      </div>

      {!selectedUser ? (
        <Notice type="info">暂无可配置用户</Notice>
      ) : isAdmin ? (
        <Notice type="info">
          该用户为管理员，拥有集群最大权限（cluster-admin），无需单独分配 namespace 级别权限。
        </Notice>
      ) : (
        <section className="panel">
          <h3>K8s 资源操作权限</h3>

          {rows.map((row, index) => (
            <div className="formGrid permissionRow" key={index}>
              <label className="formRow">
                Namespace
                <input
                  value={row.namespace}
                  onChange={(e) => updateRow(index, "namespace", e.target.value)}
                  placeholder="dev / *"
                />
              </label>
              <label className="formRow">
                API Group
                <input
                  value={row.apiGroup}
                  onChange={(e) => updateRow(index, "apiGroup", e.target.value)}
                  placeholder="apps / 留空"
                />
              </label>
              <label className="formRow">
                Resource
                <input
                  value={row.resource}
                  onChange={(e) => updateRow(index, "resource", e.target.value)}
                  placeholder="pods / deployments"
                />
              </label>
              <label className="formRow">
                Verbs
                <input
                  value={row.verbsText}
                  onChange={(e) => updateRow(index, "verbsText", e.target.value)}
                  placeholder="get,list,watch"
                />
              </label>
              <button className="dangerButton" onClick={() => removeRow(index)}>
                删除
              </button>
            </div>
          ))}

          <div className="actions">
            <button onClick={addRow}>新增权限</button>
            <button onClick={handleSave} className="primary-btn" disabled={submitting}>
              保存权限
            </button>
          </div>
        </section>
      )}
    </div>
  );
}
