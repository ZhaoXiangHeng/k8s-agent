import type { UpdateUserModelBindingsRequest, UserModelBindingMap } from "../../domain/userModelBinding";
import { apiRequest, type ApiAuth } from "./client";

export function listUserModelBindings(auth: ApiAuth) {
  return apiRequest<UserModelBindingMap>("/api/admin/user-model-bindings", { auth });
}

export function updateUserModelBindings(auth: ApiAuth, userId: string, body: UpdateUserModelBindingsRequest) {
  return apiRequest<UserModelBindingMap>(`/api/admin/users/${userId}/llm-models`, { method: "PUT", body, auth });
}
