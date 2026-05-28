import { useEffect, useRef, useState } from "react";
import type { CreateUserRequest, User } from "../domain/user";
import { EmptyState } from "../components/EmptyState";

interface UserManagementProps {
  users: User[];
  onCreateUser: (body: CreateUserRequest) => Promise<void>;
  onDeleteUser: (id: string) => Promise<void>;
  onResetPassword: (id: string, password: string) => Promise<void>;
  onNavigateToTab?: (tab: "models" | "permissions", userId: string) => void;
}

export default function UserManagement({
  users,
  onCreateUser,
  onDeleteUser,
  onResetPassword,
  onNavigateToTab,
}: UserManagementProps) {
  const [showCreate, setShowCreate] = useState(false);
  const [resetTarget, setResetTarget] = useState<User | null>(null);
  const [menuOpen, setMenuOpen] = useState<string | null>(null);
  const [menuPos, setMenuPos] = useState({ top: 0, left: 0 });
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState("");
  const menuRef = useRef<HTMLDivElement>(null);

  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newRole, setNewRole] = useState<"admin" | "operator">("operator");
  const [newDisplayName, setNewDisplayName] = useState("");
  const [newEmail, setNewEmail] = useState("");
  const [resetPasswordValue, setResetPasswordValue] = useState("");

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

  const handleCreate = async () => {
    if (!newUsername || !newPassword) return;
    setSubmitting(true);
    setFormError("");
    try {
      await onCreateUser({
        username: newUsername,
        password: newPassword,
        role: newRole,
        displayName: newDisplayName || newUsername,
        email: newEmail || `${newUsername}@example.com`,
      });
      setNewUsername("");
      setNewPassword("");
      setNewDisplayName("");
      setNewEmail("");
      setShowCreate(false);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "创建用户失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleReset = async () => {
    if (!resetTarget || !resetPasswordValue) return;
    setSubmitting(true);
    setFormError("");
    try {
      await onResetPassword(resetTarget.id, resetPasswordValue);
      setResetPasswordValue("");
      setResetTarget(null);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "重置密码失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (user: User) => {
    setMenuOpen(null);
    if (!window.confirm(`确认删除用户 ${user.username}？`)) return;
    setSubmitting(true);
    setFormError("");
    try {
      await onDeleteUser(user.id);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "删除用户失败");
    } finally {
      setSubmitting(false);
    }
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

      {formError ? <p className="errorText">{formError}</p> : null}

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
          <button onClick={handleCreate} className="primary-btn" disabled={submitting}>
            创建
          </button>
        </section>
      )}

      <section>
        <div className="tableWrap">
          <table>
            <thead>
              <tr>
                <th>用户名</th>
                <th>角色</th>
                <th>邮箱</th>
                <th>状态</th>
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
                  <td>{u.email || "-"}</td>
                  <td>{u.status || "active"}</td>
                  <td>
                    <div className="rowActions" style={{ position: "relative" }}>
                      <button className="link-btn" onClick={() => setResetTarget(u)}>
                        重置密码
                      </button>
                      <button
                        className="iconButton"
                        onClick={(e) => {
                          e.stopPropagation();
                          if (menuOpen === u.id) {
                            setMenuOpen(null);
                          } else {
                            setMenuPos({ top: e.clientY, left: e.clientX - 140 });
                            setMenuOpen(u.id);
                          }
                        }}
                        title="更多操作"
                      >
                        ···
                      </button>
                      {menuOpen === u.id && (
                        <div className="contextMenu" ref={menuRef} style={{ top: menuPos.top, left: menuPos.left }}>
                          <button onClick={() => handleMenuAction("models", u)}>配置模型</button>
                          <button onClick={() => handleMenuAction("permissions", u)}>配置权限</button>
                          <button onClick={() => handleDelete(u)}>删除用户</button>
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {users.length === 0 ? <EmptyState title="暂无用户" /> : null}
        </div>
      </section>

      {resetTarget && (
        <div className="drawer-overlay" onClick={() => setResetTarget(null)}>
          <div className="drawer" onClick={(e) => e.stopPropagation()}>
            <h3>重置密码 - {resetTarget.username}</h3>
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
              <button onClick={handleReset} className="primary-btn" disabled={submitting}>
                确认
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
