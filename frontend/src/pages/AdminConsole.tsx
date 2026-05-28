import { useState, useEffect } from "react";
import type { Model, Provider } from "../domain/llm";
import type { UpdatePermissionsRequest } from "../domain/permission";
import type { CreateUserRequest, User } from "../domain/user";
import type { UserModelBindingMap } from "../domain/userModelBinding";
import { Notice } from "../components/Notice";
import UserManagement from "./UserManagement";
import ModelAssignment from "./ModelAssignment";
import PermissionsManagement from "./PermissionsManagement";

type AdminTab = "users" | "models" | "permissions";

type AdminConsoleProps = {
  users: User[];
  models: Model[];
  providers: Provider[];
  permissionsByUserId: Record<string, UpdatePermissionsRequest["permissions"]>;
  modelBindingsByUserId: UserModelBindingMap;
  loading: boolean;
  error: string;
  onCreateUser: (body: CreateUserRequest) => Promise<void>;
  onDeleteUser: (id: string) => Promise<void>;
  onResetPassword: (id: string, password: string) => Promise<void>;
  onUpdatePermissions: (userId: string, body: UpdatePermissionsRequest) => Promise<void>;
  onUpdateModelBindings: (userId: string, modelIds: string[]) => Promise<void>;
};

export default function AdminConsole({
  users,
  models,
  providers,
  permissionsByUserId,
  modelBindingsByUserId,
  loading,
  error,
  onCreateUser,
  onDeleteUser,
  onResetPassword,
  onUpdatePermissions,
  onUpdateModelBindings,
}: AdminConsoleProps) {
  const [tab, setTab] = useState<AdminTab>("users");
  const [preselectedUserId, setPreselectedUserId] = useState<string | undefined>();

  useEffect(() => {
    const subTab = sessionStorage.getItem("adminSubTab");
    if (subTab === "模型分配") setTab("models");
    else if (subTab === "用户与权限") setTab("users");
    sessionStorage.removeItem("adminSubTab");
  }, []);

  const handleNavigateToTab = (targetTab: "models" | "permissions", userId: string) => {
    setPreselectedUserId(userId);
    setTab(targetTab);
  };

  return (
    <div className="admin-console">
      {error ? <Notice type="error">{error}</Notice> : null}
      {loading ? <Notice type="info">正在加载管理数据...</Notice> : null}
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
      {tab === "users" ? (
        <UserManagement
          users={users}
          onCreateUser={onCreateUser}
          onDeleteUser={onDeleteUser}
          onResetPassword={onResetPassword}
          onNavigateToTab={handleNavigateToTab}
        />
      ) : tab === "models" ? (
        <ModelAssignment
          users={users}
          models={models}
          providers={providers}
          modelBindingsByUserId={modelBindingsByUserId}
          onUpdateModelBindings={onUpdateModelBindings}
          preselectedUserId={preselectedUserId}
        />
      ) : (
        <PermissionsManagement
          users={users}
          permissionsByUserId={permissionsByUserId}
          onUpdatePermissions={onUpdatePermissions}
          preselectedUserId={preselectedUserId}
        />
      )}
    </div>
  );
}
