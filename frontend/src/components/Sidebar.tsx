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
    onNavigate(item.view);
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
