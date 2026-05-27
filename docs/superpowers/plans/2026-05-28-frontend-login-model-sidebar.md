# 前端三大功能改造：实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不引入新依赖的前提下改造前端：登录与密码管理、模型分配穿梭框、侧栏收起展开

**Architecture:** 新增 7 个文件（5 pages + 1 component + 1 store），拆分 App.tsx 内联组件，纯 useState 驱动。所有 mock 数据在 store/auth.ts 维护。

**Tech Stack:** React 19 + TypeScript + Vite，无额外依赖

**Spec:** `docs/superpowers/specs/2026-05-28-frontend-login-model-sidebar-design.md`

---

### Task 1: 创建 store/auth.ts — Mock 用户数据 + 登录逻辑

**Files:**
- Create: `frontend/src/store/auth.ts`

- [ ] **Step 1: 创建文件**

```typescript
// Mock user accounts with passwords
export interface User {
  id: string;
  username: string;
  password: string;
  role: "admin" | "operator";
  displayName: string;
  email: string;
  createdAt: string;
}

export interface Model {
  id: string;
  name: string;
  provider: string;
}

const defaultUsers: User[] = [
  {
    id: "user-admin",
    username: "admin",
    password: "admin123",
    role: "admin",
    displayName: "管理员",
    email: "admin@example.com",
    createdAt: "2026-01-01",
  },
  {
    id: "user-operator",
    username: "operator",
    password: "operator123",
    role: "operator",
    displayName: "操作员",
    email: "operator@example.com",
    createdAt: "2026-01-15",
  },
];

let users: User[] = [...defaultUsers];

// Per-user model bindings: userId -> Set<modelId>
const bindings = new Map<string, Set<string>>();
const defaultModel = new Map<string, string>();

// Default bindings: operator has gpt-4.1 and claude-sonnet-4-5
bindings.set("user-operator", new Set(["model-gpt4", "model-claude-sonnet"]));
defaultModel.set("user-operator", "model-gpt4");
bindings.set("user-admin", new Set(["model-gpt4", "model-claude-sonnet", "model-deepseek"]));

const allModels: Model[] = [
  { id: "model-gpt4", name: "gpt-4.1", provider: "OpenAI" },
  { id: "model-claude-sonnet", name: "claude-sonnet-4-5", provider: "Anthropic" },
  { id: "model-deepseek", name: "deepseek-v3", provider: "DeepSeek" },
];

export function login(username: string, password: string): User | null {
  const user = users.find((u) => u.username === username && u.password === password);
  return user ?? null;
}

export function listUsers(): User[] {
  return [...users];
}

export function createUser(
  username: string,
  password: string,
  role: "admin" | "operator",
  displayName: string,
  email: string
): User {
  const newUser: User = {
    id: `user-${Date.now()}`,
    username,
    password,
    role,
    displayName,
    email,
    createdAt: new Date().toISOString().split("T")[0],
  };
  users.push(newUser);
  return newUser;
}

export function resetPassword(userId: string, newPassword: string): boolean {
  const user = users.find((u) => u.id === userId);
  if (!user) return false;
  user.password = newPassword;
  return true;
}

export function getAllModels(): Model[] {
  return [...allModels];
}

export function getAssignedModels(userId: string): Model[] {
  const boundIds = bindings.get(userId);
  if (!boundIds) return [];
  return allModels.filter((m) => boundIds.has(m.id));
}

export function getAvailableModels(userId: string): Model[] {
  const boundIds = bindings.get(userId);
  if (!boundIds) return allModels;
  return allModels.filter((m) => !boundIds.has(m.id));
}

export function getDefaultModel(userId: string): string | undefined {
  return defaultModel.get(userId);
}

export function updateBindings(
  userId: string,
  add: string[],
  remove: string[],
  newDefault?: string
): void {
  let bound = bindings.get(userId);
  if (!bound) {
    bound = new Set<string>();
    bindings.set(userId, bound);
  }
  for (const id of add) bound.add(id);
  for (const id of remove) bound.delete(id);
  if (newDefault && bound.has(newDefault)) {
    defaultModel.set(userId, newDefault);
  }
}
```

- [ ] **Step 2: 验证编译**

```bash
cd e:/k8s-agent/frontend && npx tsc --noEmit src/store/auth.ts
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/store/auth.ts
git commit -m "feat: add mock auth store with login, user CRUD, model bindings"
```

---

### Task 2: 创建 components/Sidebar.tsx — 侧栏组件

**Files:**
- Create: `frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: 创建 Sidebar 组件**

```tsx
import { useState } from "react";

interface SidebarProps {
  role: "admin" | "operator";
  activeView: "operator" | "admin";
  collapsed: boolean;
  onNavigate: (view: "operator" | "admin") => void;
  onToggleCollapse: () => void;
}

interface NavItem {
  view: "operator" | "admin";
  label: string;
  icon: string;
  roles: ("admin" | "operator")[];
}

const navItems: NavItem[] = [
  { view: "operator", label: "Chat 运维", icon: "💬", roles: ["admin", "operator"] },
  { view: "admin", label: "用户与权限", icon: "👥", roles: ["admin"] },
  { view: "admin", label: "模型分配", icon: "🔗", roles: ["admin"] },
];

// Map admin nav items to internal tabs
const adminViewMap: Record<string, "operator" | "admin"> = {
  "Chat 运维": "operator",
  "用户与权限": "admin",
  "模型分配": "admin",
};

export default function Sidebar({
  role,
  activeView,
  collapsed,
  onNavigate,
  onToggleCollapse,
}: SidebarProps) {
  const [hoveredLabel, setHoveredLabel] = useState<string | null>(null);
  const visibleItems = navItems.filter((item) => item.roles.includes(role));

  const handleNavClick = (item: NavItem) => {
    // For admin, 'admin' tab switches to UserManagement by default.
    // We signal via a custom event or direct callback. Here we use item.label.
    onNavigate(item.view);
    // Store the specific sub-tab label for AdminConsole to read
    sessionStorage.setItem("adminSubTab", item.label);
  };

  return (
    <aside
      className={`sidebar${collapsed ? " collapsed" : ""}`}
      style={{ width: collapsed ? 64 : 280, transition: "width 0.2s ease" }}
    >
      <div className="sidebar-brand">
        {collapsed ? (
          <span className="sidebar-logo">⚙️</span>
        ) : (
          <>
            <p className="eyebrow">K8S AI Ops</p>
            <h1>AI 运维控制台</h1>
          </>
        )}
      </div>

      <nav>
        {visibleItems.map((item) => (
          <button
            key={item.label}
            className={activeView === item.view ? "active" : ""}
            onClick={() => handleNavClick(item)}
            onMouseEnter={() => collapsed && setHoveredLabel(item.label)}
            onMouseLeave={() => setHoveredLabel(null)}
            title={collapsed ? item.label : undefined}
          >
            <span className="nav-icon">{item.icon}</span>
            {!collapsed && <span className="nav-label">{item.label}</span>}
          </button>
        ))}
      </nav>

      <div className="sidebar-footer">
        <button onClick={onToggleCollapse} className="collapse-btn">
          {collapsed ? "▶" : "◀ 收起侧栏"}
        </button>
      </div>
    </aside>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/Sidebar.tsx
git commit -m "feat: add collapsible Sidebar component with tooltip support"
```

---

### Task 3: 创建 pages/LoginPage.tsx — 登录页面

**Files:**
- Create: `frontend/src/pages/LoginPage.tsx`

- [ ] **Step 1: 创建登录页**

```tsx
import { useState } from "react";
import { login } from "../store/auth";
import type { User } from "../store/auth";

interface LoginPageProps {
  onLogin: (user: User) => void;
}

export default function LoginPage({ onLogin }: LoginPageProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    const user = login(username, password);
    if (user) {
      onLogin(user);
    } else {
      setError("用户名或密码错误");
    }
  };

  return (
    <div className="login-page">
      <form className="login-form" onSubmit={handleSubmit}>
        <div className="login-header">
          <h2>K8S AI Ops</h2>
          <p className="eyebrow">AI 运维控制台</p>
        </div>

        <div className="form-group">
          <label htmlFor="username">用户名</label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="admin 或 operator"
            autoFocus
          />
        </div>

        <div className="form-group">
          <label htmlFor="password">密码</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="请输入密码"
          />
        </div>

        {error && <div className="login-error">{error}</div>}

        <button type="submit" className="login-btn">
          登 录
        </button>

        <div className="login-hint">
          默认账号：admin / admin123 &nbsp;|&nbsp; operator / operator123
        </div>
      </form>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/LoginPage.tsx
git commit -m "feat: add login page with mock credential validation"
```

---

### Task 4: 提取 pages/OperatorConsole.tsx

**Files:**
- Create: `frontend/src/pages/OperatorConsole.tsx`

- [ ] **Step 1: 从 App.tsx 提取 OperatorConsole**

```tsx
const abnormalPods = [
  {
    namespace: "dev",
    name: "api-7b8f9",
    phase: "Pending",
    reason: "ImagePullBackOff",
    message: "Back-off pulling image",
    restartCount: 0,
    node: "kind-worker",
  },
  {
    namespace: "dev",
    name: "worker-5d9c7",
    phase: "Running",
    reason: "CrashLoopBackOff",
    message: "Container exits after startup",
    restartCount: 6,
    node: "kind-worker2",
  },
];

export default function OperatorConsole() {
  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Operator</p>
          <h2>Chat 运维</h2>
        </div>
        <select defaultValue="mock-local">
          <option value="mock-local">Mock Local</option>
          <option value="gpt">GPT Production</option>
          <option value="claude">Claude Production</option>
        </select>
      </header>

      <section className="chat">
        <div className="message user">帮我看看现在集群里有什么异常吗？</div>
        <div className="message assistant">
          dev namespace 中有 2 个异常 Pod，主要集中在镜像拉取和容器启动失败。
        </div>
        <div className="composer">
          <input placeholder="输入自然语言运维指令" />
          <button>发送</button>
        </div>
      </section>

      <section>
        <h3>异常 Pod</h3>
        <div className="tableWrap">
          <table>
            <thead>
              <tr>
                <th>Namespace</th>
                <th>Pod</th>
                <th>Phase</th>
                <th>Reason</th>
                <th>Restarts</th>
                <th>Node</th>
              </tr>
            </thead>
            <tbody>
              {abnormalPods.map((pod) => (
                <tr key={pod.name}>
                  <td>{pod.namespace}</td>
                  <td>{pod.name}</td>
                  <td>{pod.phase}</td>
                  <td>{pod.reason}</td>
                  <td>{pod.restartCount}</td>
                  <td>{pod.node}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/OperatorConsole.tsx
git commit -m "refactor: extract OperatorConsole to separate page component"
```

---

### Task 5: 创建 pages/UserManagement.tsx — 用户管理 + 密码重置

**Files:**
- Create: `frontend/src/pages/UserManagement.tsx`

- [ ] **Step 1: 创建用户管理页面**

```tsx
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

  const refresh = () => setUsers(listUsers());

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
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/UserManagement.tsx
git commit -m "feat: add user management page with create user and reset password"
```

---

### Task 6: 创建 pages/ModelAssignment.tsx — 双列表穿梭框

**Files:**
- Create: `frontend/src/pages/ModelAssignment.tsx`

- [ ] **Step 1: 创建模型分配页面**

```tsx
import { useState, useMemo } from "react";
import {
  listUsers,
  getAllModels,
  getAssignedModels,
  getAvailableModels,
  getDefaultModel,
  updateBindings,
} from "../store/auth";
import type { User, Model } from "../store/auth";

export default function ModelAssignment() {
  const users = listUsers();
  const [selectedUserId, setSelectedUserId] = useState<string>(users[0]?.id ?? "");

  const assignedModels = useMemo(() => getAssignedModels(selectedUserId), [selectedUserId]);
  const availableModels = useMemo(() => getAvailableModels(selectedUserId), [selectedUserId]);

  const [leftSelected, setLeftSelected] = useState<Set<string>>(new Set());
  const [rightSelected, setRightSelected] = useState<Set<string>>(new Set());

  // Track dirty changes
  const [dirtyAdded, setDirtyAdded] = useState<string[]>([]);
  const [dirtyRemoved, setDirtyRemoved] = useState<string[]>([]);
  const [displayAssigned, setDisplayAssigned] = useState<Model[]>(assignedModels);
  const [displayAvailable, setDisplayAvailable] = useState<Model[]>(availableModels);
  const [pendingDefault, setPendingDefault] = useState<string | undefined>(getDefaultModel(selectedUserId));

  // Reset when user changes
  const selectUser = (userId: string) => {
    setSelectedUserId(userId);
    setDisplayAssigned(getAssignedModels(userId));
    setDisplayAvailable(getAvailableModels(userId));
    setPendingDefault(getDefaultModel(userId));
    setDirtyAdded([]);
    setDirtyRemoved([]);
    setLeftSelected(new Set());
    setRightSelected(new Set());
  };

  const isDirty = dirtyAdded.length > 0 || dirtyRemoved.length > 0 || pendingDefault !== getDefaultModel(selectedUserId);

  const toggleLeft = (modelId: string) => {
    const next = new Set(leftSelected);
    next.has(modelId) ? next.delete(modelId) : next.add(modelId);
    setLeftSelected(next);
  };

  const toggleRight = (modelId: string) => {
    const next = new Set(rightSelected);
    next.has(modelId) ? next.delete(modelId) : next.add(modelId);
    setRightSelected(next);
  };

  const assignModels = () => {
    const ids = Array.from(leftSelected);
    const toMove = displayAvailable.filter((m) => ids.includes(m.id));
    setDisplayAvailable(displayAvailable.filter((m) => !ids.includes(m.id)));
    setDisplayAssigned([...displayAssigned, ...toMove]);
    setDirtyAdded([...dirtyAdded, ...ids]);
    setDirtyRemoved(dirtyRemoved.filter((id) => !ids.includes(id)));
    setLeftSelected(new Set());
  };

  const revokeModels = () => {
    const ids = Array.from(rightSelected);
    const toMove = displayAssigned.filter((m) => ids.includes(m.id));
    setDisplayAssigned(displayAssigned.filter((m) => !ids.includes(m.id)));
    setDisplayAvailable([...displayAvailable, ...toMove]);
    setDirtyRemoved([...dirtyRemoved, ...ids]);
    setDirtyAdded(dirtyAdded.filter((id) => !ids.includes(id)));
    if (pendingDefault && ids.includes(pendingDefault)) {
      setPendingDefault(undefined);
    }
    setRightSelected(new Set());
  };

  const handleSave = () => {
    updateBindings(selectedUserId, dirtyAdded, dirtyRemoved, pendingDefault);
    setDirtyAdded([]);
    setDirtyRemoved([]);
    // Re-read to sync
    setDisplayAssigned(getAssignedModels(selectedUserId));
    setDisplayAvailable(getAvailableModels(selectedUserId));
  };

  const handleCancel = () => {
    setDisplayAssigned(getAssignedModels(selectedUserId));
    setDisplayAvailable(getAvailableModels(selectedUserId));
    setPendingDefault(getDefaultModel(selectedUserId));
    setDirtyAdded([]);
    setDirtyRemoved([]);
    setLeftSelected(new Set());
    setRightSelected(new Set());
  };

  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Admin</p>
          <h2>模型分配</h2>
        </div>
      </header>

      {/* User selector */}
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

      {/* Shuttle box */}
      <div className="shuttle-box">
        {/* Left: Available */}
        <div className="shuttle-panel">
          <div className="shuttle-header">
            可分配模型
            <span className="count">{displayAvailable.length} 个</span>
          </div>
          <div className="shuttle-list">
            {displayAvailable.map((m) => (
              <label key={m.id} className="shuttle-item">
                <input
                  type="checkbox"
                  checked={leftSelected.has(m.id)}
                  onChange={() => toggleLeft(m.id)}
                />
                <div>
                  <div className="model-name">{m.name}</div>
                  <div className="model-provider">{m.provider}</div>
                </div>
              </label>
            ))}
            {displayAvailable.length === 0 && (
              <div className="shuttle-empty">所有模型已分配</div>
            )}
          </div>
        </div>

        {/* Middle buttons */}
        <div className="shuttle-actions">
          <button onClick={assignModels} disabled={leftSelected.size === 0}>
            分配 &gt;
          </button>
          <button onClick={revokeModels} disabled={rightSelected.size === 0}>
            &lt; 回收
          </button>
        </div>

        {/* Right: Assigned */}
        <div className="shuttle-panel assigned">
          <div className="shuttle-header">
            已分配模型
            <span className="count">{displayAssigned.length} 个</span>
          </div>
          <div className="shuttle-list">
            {displayAssigned.map((m) => (
              <label key={m.id} className="shuttle-item">
                <input
                  type="checkbox"
                  checked={rightSelected.has(m.id)}
                  onChange={() => toggleRight(m.id)}
                />
                <div className="shuttle-item-info">
                  <div className="model-name">{m.name}</div>
                  <div className="model-provider">{m.provider}</div>
                </div>
                {pendingDefault === m.id ? (
                  <span className="default-badge">默认</span>
                ) : (
                  <button
                    className="link-btn"
                    onClick={(e) => {
                      e.preventDefault();
                      setPendingDefault(m.id);
                    }}
                  >
                    设为默认
                  </button>
                )}
              </label>
            ))}
            {displayAssigned.length === 0 && (
              <div className="shuttle-empty">尚未分配模型</div>
            )}
          </div>
        </div>
      </div>

      {/* Bottom save/cancel */}
      <div className="shuttle-footer">
        <button onClick={handleCancel} disabled={!isDirty}>
          取消
        </button>
        <button onClick={handleSave} disabled={!isDirty} className="primary-btn">
          保存变更
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/ModelAssignment.tsx
git commit -m "feat: add model assignment shuttle box with batch assign/revoke"
```

---

### Task 7: 创建 pages/AdminConsole.tsx — Admin 主页（Tab 切换）

**Files:**
- Create: `frontend/src/pages/AdminConsole.tsx`

- [ ] **Step 1: 创建 AdminConsole**

```tsx
import { useState, useEffect } from "react";
import UserManagement from "./UserManagement";
import ModelAssignment from "./ModelAssignment";

type AdminTab = "users" | "models";

export default function AdminConsole() {
  const [tab, setTab] = useState<AdminTab>("users");

  // Read sub-tab from Sidebar navigation
  useEffect(() => {
    const subTab = sessionStorage.getItem("adminSubTab");
    if (subTab === "模型分配") setTab("models");
    else if (subTab === "用户与权限") setTab("users");
    sessionStorage.removeItem("adminSubTab");
  }, []);

  return (
    <div className="admin-console">
      <div className="admin-tabs">
        <button
          className={tab === "users" ? "tab active" : "tab"}
          onClick={() => setTab("users")}
        >
          用户与权限
        </button>
        <button
          className={tab === "models" ? "tab active" : "tab"}
          onClick={() => setTab("models")}
        >
          模型分配
        </button>
      </div>
      {tab === "users" ? <UserManagement /> : <ModelAssignment />}
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/AdminConsole.tsx
git commit -m "feat: add AdminConsole with tab-based navigation"
```

---

### Task 8: 重构 App.tsx — 集成所有组件

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: 重写 App.tsx**

```tsx
import { useState } from "react";
import Sidebar from "./components/Sidebar";
import LoginPage from "./pages/LoginPage";
import OperatorConsole from "./pages/OperatorConsole";
import AdminConsole from "./pages/AdminConsole";
import type { User } from "./store/auth";

export default function App() {
  const [auth, setAuth] = useState<User | null>(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  // For admin: which main view is active (operator chat or admin pages)
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
```

- [ ] **Step 2: 验证构建**

```bash
cd e:/k8s-agent/frontend && npm run build
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "refactor: integrate login, sidebar, and new pages into App"
```

---

### Task 9: 更新 styles.css — 追加新组件样式

**Files:**
- Modify: `frontend/src/styles.css`

- [ ] **Step 1: 追加样式到 styles.css 末尾**

```css
/* ===== Login Page ===== */
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #f0f2f5;
}

.login-form {
  width: 400px;
  background: #ffffff;
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-header h2 {
  font-size: 20px;
  color: #182226;
  margin: 0 0 4px;
}

.login-header .eyebrow {
  text-transform: none;
  font-size: 13px;
  color: #8899a6;
  font-weight: 400;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 13px;
  color: #445566;
  margin-bottom: 6px;
}

.form-group input,
.form-group select {
  width: 100%;
  box-sizing: border-box;
}

.login-error {
  background: #fff0f0;
  color: #c0392b;
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 13px;
  margin-bottom: 16px;
}

.login-btn {
  width: 100%;
  padding: 12px;
  background: #182226;
  color: #ffffff;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 4px;
}

.login-hint {
  text-align: center;
  font-size: 12px;
  color: #8899a6;
  margin-top: 16px;
  padding: 12px;
  background: #f7f9fa;
  border-radius: 8px;
}

/* ===== Sidebar Collapse ===== */
.sidebar {
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 40px;
  padding: 32px 24px;
}

.sidebar.collapsed {
  padding: 32px 12px;
  align-items: center;
}

.sidebar.collapsed .sidebar-brand {
  text-align: center;
}

.sidebar-logo {
  font-size: 20px;
}

.sidebar-footer {
  margin-top: auto;
  border-top: 1px solid #2a3a40;
  padding-top: 12px;
}

.collapse-btn {
  width: 100%;
  padding: 8px;
  background: #2a3a40;
  color: #aabbcc;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  text-align: center;
  min-height: auto;
}

.sidebar.collapsed .collapse-btn {
  font-size: 14px;
  padding: 6px;
}

.nav-icon {
  font-size: 16px;
  margin-right: 10px;
  display: inline-block;
  vertical-align: middle;
}

.sidebar.collapsed .nav-icon {
  margin-right: 0;
}

.nav-label {
  vertical-align: middle;
}

/* ===== Admin Tabs ===== */
.admin-tabs {
  display: flex;
  gap: 0;
  margin-bottom: 0;
  background: #ffffff;
  border: 1px solid #dce4e6;
  border-radius: 8px 8px 0 0;
  border-bottom: none;
  overflow: hidden;
}

.admin-tabs .tab {
  padding: 12px 24px;
  background: #f7f9fa;
  border: none;
  border-radius: 0;
  cursor: pointer;
  font-size: 13px;
  color: #667788;
  min-height: auto;
}

.admin-tabs .tab.active {
  background: #ffffff;
  color: #182226;
  font-weight: 600;
  border-bottom: 2px solid #6fbf9f;
}

/* ===== Form Card ===== */
.form-card {
  background: #ffffff;
  border: 1px solid #dce4e6;
  border-radius: 8px;
  padding: 20px;
}

.form-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.primary-btn {
  background: #182226;
  color: #ffffff;
  border-color: #182226;
}

.primary-btn:hover {
  background: #2a3a40;
}

.link-btn {
  background: none;
  border: none;
  color: #3a8fbf;
  cursor: pointer;
  font-size: 13px;
  padding: 0;
  min-height: auto;
  text-decoration: underline;
}

.link-btn:hover {
  color: #2a6f9f;
}

.role-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.role-badge.admin {
  background: #e8f5ef;
  color: #2a7a5f;
}

.role-badge.operator {
  background: #e8f0fa;
  color: #3a6fbf;
}

/* ===== Drawer (password reset) ===== */
.drawer-overlay {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: 100;
  display: flex;
  justify-content: flex-end;
}

.drawer {
  width: 360px;
  height: 100%;
  background: #ffffff;
  padding: 32px 24px;
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.1);
  overflow-y: auto;
}

.drawer h3 {
  margin: 0 0 24px;
}

.drawer-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
}

/* ===== Shuttle Box ===== */
.user-selector {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
}

.user-selector label {
  font-size: 13px;
  font-weight: 600;
  color: #182226;
}

.shuttle-box {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 0;
  background: #ffffff;
  border: 1px solid #dce4e6;
  border-radius: 8px;
  overflow: hidden;
}

.shuttle-panel {
  border-right: 1px solid #e1e8ed;
}

.shuttle-panel.assigned {
  border-right: none;
  border-left: 1px solid #e1e8ed;
}

.shuttle-header {
  padding: 10px 14px;
  font-weight: 600;
  font-size: 13px;
  background: #f7f9fa;
  border-bottom: 1px solid #e1e8ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.shuttle-panel.assigned .shuttle-header {
  background: #e8f5ef;
  border-color: #c5e8d8;
}

.count {
  font-size: 11px;
  color: #8899a6;
  font-weight: 400;
}

.shuttle-list {
  padding: 4px 0;
  min-height: 240px;
  max-height: 400px;
  overflow-y: auto;
}

.shuttle-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-bottom: 1px solid #f0f2f5;
  cursor: pointer;
}

.shuttle-item input[type="checkbox"] {
  width: 16px;
  height: 16px;
  min-height: auto;
  flex-shrink: 0;
}

.shuttle-item-info {
  flex: 1;
}

.model-name {
  font-size: 13px;
  font-weight: 600;
}

.model-provider {
  font-size: 11px;
  color: #8899a6;
}

.default-badge {
  font-size: 10px;
  background: #6fbf9f;
  color: #ffffff;
  padding: 2px 6px;
  border-radius: 3px;
  white-space: nowrap;
}

.shuttle-empty {
  padding: 40px 14px;
  text-align: center;
  color: #8899a6;
  font-size: 13px;
}

.shuttle-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 0 16px;
}

.shuttle-actions button {
  width: 64px;
  padding: 8px;
  font-size: 13px;
  min-height: auto;
}

.shuttle-actions button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.shuttle-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}

.shuttle-footer button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
```

- [ ] **Step 2: 验证构建**

```bash
cd e:/k8s-agent/frontend && npm run build
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/styles.css
git commit -m "style: add login, shuttle-box, drawer, sidebar-collapse, and tab styles"
```

---

## Plan Verification Checklist

Before claiming completion:

1. `cd frontend && npm run build` — 构建成功，无 TypeScript 错误
2. 所有 5 个新 pages + 1 个 component + 1 个 store 文件存在
3. App.tsx 不再包含内联组件，只做顶层状态管理和布局
