import { useState, useEffect } from "react";
import { listUsers } from "../store/auth";
import type { User } from "../store/auth";

interface Permission {
  id: string;
  namespace: string;
  apiGroup: string;
  resource: string;
  verbs: string[];
  enabled: boolean;
}

interface PermissionFormRow {
  namespace: string;
  apiGroup: string;
  resource: string;
  verbsText: string;
}

const initialRow: PermissionFormRow = {
  namespace: "dev",
  apiGroup: "",
  resource: "pods",
  verbsText: "get,list,watch",
};

// Mock permissions per user
const mockPermissionsByUser: Record<string, Permission[]> = {
  "user-admin": [
    { id: "p1", namespace: "*", apiGroup: "*", resource: "*", verbs: ["*"], enabled: true },
  ],
  "user-operator": [
    { id: "p2", namespace: "dev", apiGroup: "", resource: "pods", verbs: ["get", "list", "watch"], enabled: true },
    { id: "p3", namespace: "dev", apiGroup: "apps", resource: "deployments", verbs: ["get", "list", "patch"], enabled: true },
    { id: "p4", namespace: "dev", apiGroup: "", resource: "events", verbs: ["get", "list"], enabled: true },
  ],
};

export default function PermissionsManagement({ preselectedUserId }: { preselectedUserId?: string }) {
  const users = listUsers();
  const [selectedUserId, setSelectedUserId] = useState(preselectedUserId ?? (users[0]?.id || ""));

  useEffect(() => {
    if (preselectedUserId) {
      selectUser(preselectedUserId);
    }
  }, [preselectedUserId]);
  const [rows, setRows] = useState<PermissionFormRow[]>(() => {
    const perms = mockPermissionsByUser[selectedUserId];
    if (!perms || perms.length === 0) return [initialRow];
    return perms.map((p) => ({
      namespace: p.namespace,
      apiGroup: p.apiGroup,
      resource: p.resource,
      verbsText: p.verbs.join(","),
    }));
  });

  const selectUser = (userId: string) => {
    setSelectedUserId(userId);
    const perms = mockPermissionsByUser[userId];
    if (!perms || perms.length === 0) {
      setRows([initialRow]);
    } else {
      setRows(perms.map((p) => ({
        namespace: p.namespace,
        apiGroup: p.apiGroup,
        resource: p.resource,
        verbsText: p.verbs.join(","),
      })));
    }
  };

  const updateRow = (index: number, field: keyof PermissionFormRow, value: string) => {
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: value } : r)));
  };

  const addRow = () => setRows([...rows, initialRow]);
  const removeRow = (index: number) => setRows(rows.filter((_, i) => i !== index));

  const handleSave = () => {
    // In mock mode: update in-memory and show feedback
    const perms: Permission[] = rows.map((r, i) => ({
      id: `perm-${Date.now()}-${i}`,
      namespace: r.namespace,
      apiGroup: r.apiGroup,
      resource: r.resource,
      verbs: r.verbsText.split(",").map((v) => v.trim()).filter(Boolean),
      enabled: true,
    }));
    mockPermissionsByUser[selectedUserId] = perms;
    alert("权限已保存");
  };

  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Admin</p>
          <h2>权限管理</h2>
        </div>
      </header>

      <div className="user-selector">
        <label>当前用户：</label>
        <select value={selectedUserId} onChange={(e) => selectUser(e.target.value)}>
          {users.map((u) => (
            <option key={u.id} value={u.id}>
              {u.username} ({u.role})
            </option>
          ))}
        </select>
      </div>

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
          <button onClick={handleSave} className="primary-btn">保存权限</button>
        </div>
      </section>
    </div>
  );
}
