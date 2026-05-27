import { useState, useEffect } from "react";
import UserManagement from "./UserManagement";
import ModelAssignment from "./ModelAssignment";
import PermissionsManagement from "./PermissionsManagement";

type AdminTab = "users" | "models" | "permissions";

export default function AdminConsole() {
  const [tab, setTab] = useState<AdminTab>("users");

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
        <button
          className={tab === "permissions" ? "tab active" : "tab"}
          onClick={() => setTab("permissions")}
        >
          权限管理
        </button>
      </div>
      {tab === "users" ? <UserManagement /> : tab === "models" ? <ModelAssignment /> : <PermissionsManagement />}
    </div>
  );
}
