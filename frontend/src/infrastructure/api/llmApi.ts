import type { CreateModelRequest, CreateProviderRequest, Model, Provider } from "../../domain/llm";
import { apiRequest, type ApiAuth } from "./client";

export function listOperatorModels(auth: ApiAuth): Promise<Model[]> {
  return apiRequest<Model[]>("/api/operator/llm-models", { auth });
}

export function listProviders(auth: ApiAuth): Promise<Provider[]> {
  return apiRequest<Provider[]>("/api/admin/llm/providers", { auth });
}

export function createProvider(auth: ApiAuth, body: CreateProviderRequest): Promise<Provider> {
  return apiRequest<Provider>("/api/admin/llm/providers", { method: "POST", body, auth });
}

export function updateProvider(auth: ApiAuth, id: string, body: Partial<CreateProviderRequest>): Promise<Provider> {
  return apiRequest<Provider>(`/api/admin/llm/providers/${id}`, { method: "PUT", body, auth });
}

export function listModels(auth: ApiAuth): Promise<Model[]> {
  return apiRequest<Model[]>("/api/admin/llm/models", { auth });
}

export function createModel(auth: ApiAuth, body: CreateModelRequest): Promise<Model> {
  return apiRequest<Model>("/api/admin/llm/models", { method: "POST", body, auth });
}

export function updateModel(auth: ApiAuth, id: string, body: Partial<CreateModelRequest>): Promise<Model> {
  return apiRequest<Model>(`/api/admin/llm/models/${id}`, { method: "PUT", body, auth });
}

export function deleteModel(auth: ApiAuth, id: string): Promise<void> {
  return apiRequest<void>(`/api/admin/llm/models/${id}`, { method: "DELETE", auth });
}
