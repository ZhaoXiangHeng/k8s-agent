import { useState } from "react";
import { listUsers, createUser, resetPassword } from "../store/auth";
import type { User } from "../store/auth";

export default function UserManagement() {
  const [users, setUsers] = useState<User[]>(listUsers());
  const [showCreate, setShowCreate] = useState(false);
  const [resetTarget, setResetTarget] = useState<User | null>(null);

  // Create form state
  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newRole, setNewRole] = useState<"admin" | "operator">("operator");
  const [newDisplayName, setNewDisplayName] = useState("");
  const [newEmail, setNewEmail] = useState("");

  // Reset password state
  const [resetPasswordValue, setResetPasswordValue] = useState("");

  const handleCreate = () => {
    if (!newUsername || !newPassword) return;
    createUser(newUsername, newPassword, newRole, newDisplayName || newUsername, newEmail || `${newUsername}@example.com`);
    setNewUsername("");
    setNewPassword("");
    setNewDisplayName("");
    setNewEmail("");
    setShowCreate(false);
    setUsers(listUsers());
  };

  const handleReset = () => {
    if (!resetTarget || !resetPasswordValue) return;
    resetPassword(resetTarget.id, resetPasswordValue);
    setResetPasswordValue("");
    setResetTarget(null);
  };

  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Admin</p>
          <h2>用户管理</h2>
        </div>
        <button onClick={() => setShowCreate(!showCreate)}>
          {showCreate ? "取消" : "创建操作员"}
        </button>
      </header>

      {/* Create user form */}
      {showCreate && (
        <section className="form-card">
          <h3>新建用户</h3>
          <div className="form-row">
            <div className="form-group">
              <label>用户名</label>
              <input value={newUsername} onChange={(e) => setNewUsername(e.target.value)} placeholder="new-user" />
            </div>
            <div className="form-group">
              <label>角色</label>
              <select value={newRole} onChange={(e) => setNewRole(e.target.value as "admin" | "operator")}>
                <option value="operator">Operator</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div className="form-group">
              <label>密码</label>
              <input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} placeholder="输入密码" />
            </div>
            <div className="form-group">
              <label>显示名称</label>
              <input value={newDisplayName} onChange={(e) => setNewDisplayName(e.target.value)} placeholder="可选" />
            </div>
            <div className="form-group">
              <label>邮箱</label>
              <input value={newEmail} onChange={(e) => setNewEmail(e.target.value)} placeholder="可选" />
            </div>
          </div>
          <button onClick={handleCreate} className="primary-btn">创建</button>
        </section>
      )}

      {/* User list */}
      <section>
        <div className="tableWrap">
          <table>
            <thead>
              <tr>
                <th>用户名</th>
                <th>角色</th>
                <th>邮箱</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td>{u.username}</td>
                  <td>
                    <span className={`role-badge ${u.role}`}>{u.role}</span>
                  </td>
                  <td>{u.email}</td>
                  <td>{u.createdAt}</td>
                  <td>
                    <button
                      className="link-btn"
                      onClick={() => setResetTarget(u)}
                    >
                      重置密码
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* Reset password drawer */}
      {resetTarget && (
        <div className="drawer-overlay" onClick={() => setResetTarget(null)}>
          <div className="drawer" onClick={(e) => e.stopPropagation()}>
            <h3>重置密码 — {resetTarget.username}</h3>
            <div className="form-group">
              <label>新密码</label>
              <input
                type="password"
                value={resetPasswordValue}
                onChange={(e) => setResetPasswordValue(e.target.value)}
                placeholder="输入新密码"
              />
            </div>
            <div className="drawer-actions">
              <button onClick={() => setResetTarget(null)}>取消</button>
              <button onClick={handleReset} className="primary-btn">确认</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
