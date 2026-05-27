import { useState } from "react";

export type PageKey =
  | "operator-chat"
  | "operator-permissions"
  | "operator-models"
  | "admin-users-permissions"
  | "admin-llm"
  | "admin-audit";

interface NavGroup {
  key: string;
  title: string;
  items: NavItem[];
  roles: ("admin" | "operator")[];
}

interface NavItem {
  page: PageKey;
  label: string;
  icon: string;
}

const navGroups: NavGroup[] = [
  {
    key: "workbench",
    title: "运维工作台",
    roles: ["admin", "operator"],
    items: [
      { page: "operator-chat", label: "Chat 运维", icon: "💬" },
      { page: "operator-permissions", label: "我的授权", icon: "🔑" },
      { page: "operator-models", label: "可用模型", icon: "🧩" },
    ],
  },
  {
    key: "platform",
    title: "平台管理",
    roles: ["admin"],
    items: [
      { page: "admin-users-permissions", label: "用户与权限", icon: "👥" },
      { page: "admin-llm", label: "LLM 配置", icon: "⚙️" },
      { page: "admin-audit", label: "审计日志", icon: "📋" },
    ],
  },
];

interface SidebarProps {
  role: "admin" | "operator";
  activePage: PageKey;
  collapsed: boolean;
  onNavigate: (page: PageKey) => void;
  onToggleCollapse: () => void;
}

export default function Sidebar({
  role,
  activePage,
  collapsed,
  onNavigate,
  onToggleCollapse,
}: SidebarProps) {
  const [collapsedGroups, setCollapsedGroups] = useState<Record<string, boolean>>({});
  const visibleGroups = navGroups.filter((g) => g.roles.includes(role));
  const allCollapsed = visibleGroups.every((g) => collapsedGroups[g.key]);

  function toggleGroup(key: string) {
    setCollapsedGroups((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  function setAllGroups(collapse: boolean) {
    setCollapsedGroups(
      Object.fromEntries(visibleGroups.map((g) => [g.key, collapse]))
    );
  }

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

      {!collapsed && (
        <button
          className="navCollapseAll"
          onClick={() => setAllGroups(!allCollapsed)}
        >
          {allCollapsed ? "一键展开导航" : "一键收起导航"}
        </button>
      )}

      <nav className="navGroups">
        {visibleGroups.map((group) => (
          <div className="navGroup" key={group.key}>
            {!collapsed && (
              <button
                className="navGroupTitle"
                onClick={() => toggleGroup(group.key)}
              >
                <span>{group.title}</span>
                <span>{collapsedGroups[group.key] ? "＋" : "－"}</span>
              </button>
            )}
            {!collapsedGroups[group.key] &&
              group.items.map((item) => (
                <button
                  key={item.page}
                  className={activePage === item.page ? "active" : ""}
                  onClick={() => onNavigate(item.page)}
                  title={collapsed ? item.label : undefined}
                >
                  <span className="nav-icon">{item.icon}</span>
                  {!collapsed && (
                    <span className="nav-label">{item.label}</span>
                  )}
                </button>
              ))}
          </div>
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
