import type { CurrentUser } from "../../domain/auth";
import type { CreateUserRequest, User } from "../../domain/user";
import { apiRequest, type ApiAuth } from "./client";

export function getCurrentUser(auth: ApiAuth): Promise<CurrentUser> {
  return apiRequest<CurrentUser>("/api/me", { auth });
}

export function listUsers(auth: ApiAuth): Promise<User[]> {
  return apiRequest<User[]>("/api/admin/users", { auth });
}

export function createUser(auth: ApiAuth, body: CreateUserRequest): Promise<User> {
  return apiRequest<User>("/api/admin/users", { method: "POST", body, auth });
}

export function deleteUser(auth: ApiAuth, id: string): Promise<void> {
  return apiRequest<void>(`/api/admin/users/${id}`, { method: "DELETE", auth });
}

export function resetUserPassword(auth: ApiAuth, id: string, password: string): Promise<void> {
  return apiRequest<void>(`/api/admin/users/${id}/password`, { method: "PUT", body: { password }, auth });
}
