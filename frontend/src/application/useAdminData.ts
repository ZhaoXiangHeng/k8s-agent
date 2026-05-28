import { useCallback, useEffect, useState } from "react";
import type { AuditLog } from "../domain/audit";
import type { CreateModelRequest, CreateProviderRequest, Model, Provider } from "../domain/llm";
import type { UpdatePermissionsRequest } from "../domain/permission";
import type { CreateUserRequest, User } from "../domain/user";
import type { UserModelBindingMap } from "../domain/userModelBinding";
import { listAuditLogs } from "../infrastructure/api/auditApi";
import type { ApiAuth } from "../infrastructure/api/client";
import {
  createModel,
  createProvider,
  deleteModel,
  listModels,
  listProviders,
  updateModel,
  updateProvider
} from "../infrastructure/api/llmApi";
import { listUserPermissions, updateUserPermissions } from "../infrastructure/api/permissionApi";
import { createUser, deleteUser, listUsers, resetUserPassword } from "../infrastructure/api/userApi";
import { listUserModelBindings, updateUserModelBindings } from "../infrastructure/api/userModelBindingApi";
import { appConfig } from "../config";
import {
  mockAuditLogs,
  mockModelBindingsByUserId,
  mockModels,
  mockPermissionsByUserId,
  mockProviders,
  mockUserPasswords,
  mockUsers
} from "./mockData";

export function useAdminData(auth: ApiAuth, enabled = true) {
  const [users, setUsers] = useState<User[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [permissionsByUserId, setPermissionsByUserId] = useState<Record<string, UpdatePermissionsRequest["permissions"]>>({});
  const [modelBindingsByUserId, setModelBindingsByUserId] = useState<UserModelBindingMap>({});
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    if (!enabled) return;
    setLoading(true);
    setError("");
    try {
      if (appConfig.dataMode === "mock") {
        setUsers(mockUsers);
        setProviders(mockProviders);
        setModels(mockModels);
        setPermissionsByUserId(mockPermissionsByUserId);
        setModelBindingsByUserId(mockModelBindingsByUserId);
        setAuditLogs(mockAuditLogs);
        return;
      }
      const [nextUsers, nextProviders, nextModels, nextModelBindings, nextAuditLogs] = await Promise.all([
        listUsers(auth),
        listProviders(auth),
        listModels(auth),
        listUserModelBindings(auth),
        listAuditLogs(auth)
      ]);
      const permissionEntries = await Promise.all(
        nextUsers.map(async (user) => [user.id, await listUserPermissions(auth, user.id)] as const)
      );
      setUsers(nextUsers);
      setProviders(nextProviders);
      setModels(nextModels);
      setPermissionsByUserId(Object.fromEntries(permissionEntries));
      setModelBindingsByUserId(nextModelBindings);
      setAuditLogs(nextAuditLogs);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载管理员数据失败");
    } finally {
      setLoading(false);
    }
  }, [auth, enabled]);

  useEffect(() => {
    if (enabled) void reload();
  }, [enabled, reload]);

  return {
    users,
    providers,
    models,
    permissionsByUserId,
    modelBindingsByUserId,
    auditLogs,
    loading,
    error,
    reload,
    createUser: async (body: CreateUserRequest) => {
      if (appConfig.dataMode === "mock") {
        const id = `user-${Date.now()}`;
        setUsers((current) => [...current, { ...body, id, status: "active" }]);
        setPermissionsByUserId((current) => ({ ...current, [id]: [] }));
        setModelBindingsByUserId((current) => ({ ...current, [id]: [] }));
        return;
      }
      await createUser(auth, body);
      await reload();
    },
    deleteUser: async (id: string) => {
      if (appConfig.dataMode === "mock") {
        setUsers((current) => current.filter((user) => user.id !== id));
        setPermissionsByUserId((current) => {
          const { [id]: _removed, ...rest } = current;
          return rest;
        });
        setModelBindingsByUserId((current) => {
          const { [id]: _removed, ...rest } = current;
          return rest;
        });
        setAuditLogs((current) => [{
          id: `audit-${Date.now()}`,
          actorUserId: "user-admin",
          action: "admin.user.delete",
          targetType: "user",
          targetId: id,
          allowed: true,
          reason: "user deleted and related bindings cleaned",
          createdAt: new Date().toISOString()
        }, ...current]);
        return;
      }
      await deleteUser(auth, id);
      await reload();
    },
    resetPassword: async (id: string, password: string) => {
      if (appConfig.dataMode === "mock") {
        const user = users.find((item) => item.id === id);
        if (user) {
          mockUserPasswords[user.username] = password;
        }
        setAuditLogs((current) => [{
          id: `audit-${Date.now()}`,
          actorUserId: "user-admin",
          action: "admin.user.password.reset",
          targetType: "user",
          targetId: id,
          allowed: true,
          reason: "password reset",
          createdAt: new Date().toISOString()
        }, ...current]);
        return;
      }
      await resetUserPassword(auth, id, password);
      await reload();
    },
    updatePermissions: async (userId: string, body: UpdatePermissionsRequest) => {
      if (appConfig.dataMode === "mock") {
        setPermissionsByUserId((current) => ({ ...current, [userId]: body.permissions }));
        setAuditLogs((current) => [{
          id: `audit-${Date.now()}`,
          actorUserId: "user-admin",
          action: "admin.permissions.update",
          targetType: "user",
          targetId: userId,
          allowed: true,
          reason: `${body.permissions.length} permissions updated`,
          createdAt: new Date().toISOString()
        }, ...current]);
        return;
      }
      await updateUserPermissions(auth, userId, body);
      await reload();
    },
    updateModelBindings: async (userId: string, modelIds: string[]) => {
      if (appConfig.dataMode === "mock") {
        setModelBindingsByUserId((current) => ({ ...current, [userId]: modelIds }));
        setAuditLogs((current) => [{
          id: `audit-${Date.now()}`,
          actorUserId: "user-admin",
          action: "admin.user_models.update",
          targetType: "user",
          targetId: userId,
          allowed: true,
          reason: `${modelIds.length} models bound`,
          createdAt: new Date().toISOString()
        }, ...current]);
        return;
      }
      await updateUserModelBindings(auth, userId, { modelIds });
      await reload();
    },
    createProvider: async (body: CreateProviderRequest) => {
      if (appConfig.dataMode === "mock") {
        setProviders((current) => [...current, { ...body, id: `provider-${Date.now()}`, apiKeyConfigured: !!body.apiKey }]);
        return;
      }
      await createProvider(auth, body);
      await reload();
    },
    updateProvider: async (id: string, body: Partial<CreateProviderRequest>) => {
      if (appConfig.dataMode === "mock") {
        setProviders((current) => current.map((provider) => provider.id === id ? { ...provider, ...body, apiKeyConfigured: body.apiKey ? true : provider.apiKeyConfigured } : provider));
        return;
      }
      await updateProvider(auth, id, body);
      await reload();
    },
    createModel: async (body: CreateModelRequest) => {
      if (appConfig.dataMode === "mock") {
        setModels((current) => [...current, { ...body, id: `model-${Date.now()}` }]);
        return;
      }
      await createModel(auth, body);
      await reload();
    },
    updateModel: async (id: string, body: Partial<CreateModelRequest>) => {
      if (appConfig.dataMode === "mock") {
        setModels((current) => current.map((model) => model.id === id ? { ...model, ...body } : model));
        return;
      }
      await updateModel(auth, id, body);
      await reload();
    },
    deleteModel: async (id: string) => {
      if (appConfig.dataMode === "mock") {
        setModels((current) => current.filter((model) => model.id !== id));
        setModelBindingsByUserId((current) => Object.fromEntries(
          Object.entries(current).map(([userId, modelIds]) => [userId, modelIds.filter((modelId) => modelId !== id)])
        ));
        setAuditLogs((current) => [{
          id: `audit-${Date.now()}`,
          actorUserId: "user-admin",
          action: "admin.llm.model.delete",
          targetType: "llm_model",
          targetId: id,
          allowed: true,
          reason: "model deleted and user bindings cleaned",
          createdAt: new Date().toISOString()
        }, ...current]);
        return;
      }
      await deleteModel(auth, id);
      await reload();
    }
  };
}
