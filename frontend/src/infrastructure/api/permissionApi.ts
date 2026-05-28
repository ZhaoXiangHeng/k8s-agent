import type { Permission, UpdatePermissionsRequest } from "../../domain/permission";
import { apiRequest, type ApiAuth } from "./client";

export function listOperatorPermissions(auth: ApiAuth): Promise<Permission[]> {
  return apiRequest<Permission[]>("/api/operator/permissions", { auth });
}

export function listUserPermissions(auth: ApiAuth, userId: string): Promise<Permission[]> {
  return apiRequest<Permission[]>(`/api/admin/users/${userId}/permissions`, { auth });
}

export function updateUserPermissions(auth: ApiAuth, userId: string, body: UpdatePermissionsRequest): Promise<Permission[]> {
  return apiRequest<Permission[]>(`/api/admin/users/${userId}/permissions`, { method: "PUT", body, auth });
}
