import { useState, useRef, useEffect } from "react";
import { listUsers, createUser, resetPassword, updateUser } from "../store/auth";
import type { User } from "../store/auth";

interface UserManagementProps {
  onNavigateToTab?: (tab: "models" | "permissions", userId: string) => void;
}

export default function UserManagement({ onNavigateToTab }: UserManagementProps) {
  const [users, setUsers] = useState<User[]>(listUsers());
  const [showCreate, setShowCreate] = useState(false);
  const [resetTarget, setResetTarget] = useState<User | null>(null);
  const [editTarget, setEditTarget] = useState<User | null>(null);
  const [editForm, setEditForm] = useState({ username: "", role: "operator" as "admin" | "operator", displayName: "", email: "" });
  const [menuOpen, setMenuOpen] = useState<string | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // Create form state
  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newRole, setNewRole] = useState<"admin" | "operator">("operator");
  const [newDisplayName, setNewDisplayName] = useState("");
  const [newEmail, setNewEmail] = useState("");

  // Reset password state
  const [resetPasswordValue, setResetPasswordValue] = useState("");

  // Close menu when clicking outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(null);
      }
    }
    if (menuOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [menuOpen]);

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

  const handleEdit = (user: User) => {
    setMenuOpen(null);
    setEditTarget(user);
    setEditForm({
      username: user.username,
      role: user.role,
      displayName: user.displayName,
      email: user.email,
    });
  };

  const handleSaveEdit = () => {
    if (!editTarget) return;
    updateUser(editTarget.id, editForm);
    setEditTarget(null);
    setUsers(listUsers());
  };

  const handleMenuAction = (tab: "models" | "permissions", user: User) => {
    setMenuOpen(null);
    onNavigateToTab?.(tab, user.id);
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
                    <div className="rowActions" style={{ position: "relative" }}>
                      <button
                        className="link-btn"
                        onClick={() => setResetTarget(u)}
                      >
                        重置密码
                      </button>
                      <button
                        className="iconButton"
                        onClick={(e) => {
                          e.stopPropagation();
                          setMenuOpen(menuOpen === u.id ? null : u.id);
                        }}
                        title="更多操作"
                      >
                        ···
                      </button>
                      {menuOpen === u.id && (
                        <div className="contextMenu" ref={menuRef}>
                          <button onClick={() => handleEdit(u)}>
                            ✏️ 编辑用户
                          </button>
                          <button onClick={() => handleMenuAction("models", u)}>
                            🔗 配置模型
                          </button>
                          <button onClick={() => handleMenuAction("permissions", u)}>
                            🔑 配置权限
                          </button>
                        </div>
                      )}
                    </div>
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

      {/* Edit user drawer */}
      {editTarget && (
        <div className="drawer-overlay" onClick={() => setEditTarget(null)}>
          <div className="drawer" onClick={(e) => e.stopPropagation()}>
            <h3>编辑用户 — {editTarget.username}</h3>
            <div className="form-group">
              <label>用户名</label>
              <input
                value={editForm.username}
                onChange={(e) => setEditForm({ ...editForm, username: e.target.value })}
              />
            </div>
            <div className="form-group">
              <label>角色</label>
              <select
                value={editForm.role}
                onChange={(e) => setEditForm({ ...editForm, role: e.target.value as "admin" | "operator" })}
              >
                <option value="operator">Operator</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div className="form-group">
              <label>显示名称</label>
              <input
                value={editForm.displayName}
                onChange={(e) => setEditForm({ ...editForm, displayName: e.target.value })}
              />
            </div>
            <div className="form-group">
              <label>邮箱</label>
              <input
                value={editForm.email}
                onChange={(e) => setEditForm({ ...editForm, email: e.target.value })}
              />
            </div>
            <div className="drawer-actions">
              <button onClick={() => setEditTarget(null)}>取消</button>
              <button onClick={handleSaveEdit} className="primary-btn">保存</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
