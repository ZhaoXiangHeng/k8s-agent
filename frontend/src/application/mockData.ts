import type { AuditLog } from "../domain/audit";
import type { ChatResource, ChatSession } from "../domain/chat";
import type { CurrentUser } from "../domain/auth";
import type { Model, Provider } from "../domain/llm";
import type { Permission } from "../domain/permission";
import type { User } from "../domain/user";
import type { UserModelBindingMap } from "../domain/userModelBinding";

export const mockCurrentUser = (role: "admin" | "operator", username: string): CurrentUser => ({
  id: `user-${username}`,
  username,
  displayName: role === "admin" ? "平台管理员" : "集群操作员",
  email: `${username}@demo.local`,
  role,
  status: "active"
});

export const mockUsers: User[] = [
  { id: "user-admin", username: "admin", displayName: "平台管理员", email: "admin@demo.local", role: "admin", status: "active" },
  { id: "user-operator", username: "operator", displayName: "集群操作员", email: "operator@demo.local", role: "operator", status: "active" }
];

export const mockUserPasswords: Record<string, string> = {
  admin: "admin123",
  operator: "operator123"
};

export const mockPermissions: Permission[] = [
  { id: "perm-pods-dev", namespace: "dev", apiGroup: "", resource: "pods", verbs: ["get", "list", "watch"], enabled: true },
  { id: "perm-events-dev", namespace: "dev", apiGroup: "", resource: "events", verbs: ["get", "list"], enabled: true },
  { id: "perm-deployments-dev", namespace: "dev", apiGroup: "apps", resource: "deployments", verbs: ["get", "list", "patch"], enabled: true }
];

export const mockPermissionsByUserId: Record<string, Permission[]> = {
  "user-admin": [
    { id: "perm-admin-all", namespace: "*", apiGroup: "*", resource: "*", verbs: ["*"], enabled: true }
  ],
  "user-operator": mockPermissions
};

export const mockProviders: Provider[] = [
  { id: "provider-openai", name: "OpenAI", protocol: "openai", baseUrl: "https://api.openai.com/v1", enabled: true, apiKeyConfigured: true },
  { id: "provider-anthropic", name: "Anthropic", protocol: "anthropic", baseUrl: "https://api.anthropic.com", enabled: false, apiKeyConfigured: false }
];

export const mockModels: Model[] = [
  { id: "model-gpt-41", providerId: "provider-openai", modelName: "gpt-4.1", displayName: "GPT-4.1 运维模型", supportsTools: true, supportsStreaming: true, enabled: true },
  { id: "model-claude-sonnet", providerId: "provider-anthropic", modelName: "claude-3-5-sonnet", displayName: "Claude Sonnet", supportsTools: true, supportsStreaming: true, enabled: false }
];

export const mockModelBindingsByUserId: UserModelBindingMap = {
  "user-admin": ["model-gpt-41"],
  "user-operator": ["model-gpt-41"]
};

export const mockAuditLogs: AuditLog[] = [
  { id: "audit-1", actorUserId: "user-admin", action: "admin.user.create", targetType: "user", targetId: "user-operator", allowed: true, reason: "created", createdAt: "2026-05-27T10:10:00Z" },
  { id: "audit-2", actorUserId: "user-operator", action: "operator.chat.message", targetType: "chat_message", targetId: "msg-001", namespace: "dev", resource: "pods", verb: "list", allowed: true, reason: "completed", createdAt: "2026-05-27T10:16:00Z" },
  { id: "audit-3", actorUserId: "user-operator", action: "mcp.tool.denied", targetType: "k8s_tool", targetId: "list_secrets", namespace: "prod", resource: "secrets", verb: "list", allowed: false, reason: "namespace not allowed", createdAt: "2026-05-27T10:20:00Z" }
];

export const mockChatSession: ChatSession = {
  id: "chat-session-mock",
  userId: "user-operator",
  status: "active",
  createdAt: "2026-05-27T10:15:00Z",
  title: "异常 Pod 巡检"
};

export const mockChatSessions: ChatSession[] = [
  mockChatSession,
  {
    id: "chat-session-network",
    userId: "user-operator",
    status: "active",
    createdAt: "2026-05-27T09:40:00Z",
    title: "服务依赖排查"
  }
];

export const mockChatResources: ChatResource[] = [
  { namespace: "dev", kind: "Pod", name: "api-7b8f9", phase: "Pending", reason: "ImagePullBackOff", message: "Back-off pulling image", restartCount: 0, node: "kind-worker" },
  { namespace: "dev", kind: "Pod", name: "worker-5d9c7", phase: "Running", reason: "CrashLoopBackOff", message: "Container exits after startup", restartCount: 6, node: "kind-worker2" }
];
