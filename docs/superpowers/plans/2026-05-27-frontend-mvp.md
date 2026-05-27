# Frontend 完整 MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现带 Keycloak 登录、真实 Backend API 对接、操作员 Chat 和管理员配置能力的完整 Frontend MVP。

**Architecture:** 前端按轻量 DDD 分层：`domain` 定义业务类型，`infrastructure/api` 封装请求、认证和 SSE，`application` 编排用例状态，`interfaces` 承载页面与组件。认证支持 `dev` 与 `keycloak` 双模式，生产模式使用 Authorization Code + PKCE 并通过 `Authorization: Bearer <token>` 调用 Backend。

**Tech Stack:** React 19、TypeScript、Vite、Vitest、原生 `fetch`、Web Crypto、CSS。

---

## 文件结构

- Modify: `frontend/package.json`，增加 `test` 脚本和 Vitest 依赖。
- Modify: `frontend/package-lock.json`，由 `npm install` 更新。
- Create: `frontend/src/config.ts`，读取 Vite 环境变量并提供认证配置。
- Create: `frontend/src/domain/auth.ts`，认证模式、Token、当前用户类型。
- Create: `frontend/src/domain/user.ts`，用户类型和创建请求。
- Create: `frontend/src/domain/permission.ts`，权限类型和权限 payload 工具。
- Create: `frontend/src/domain/llm.ts`，Provider、Model 类型。
- Create: `frontend/src/domain/audit.ts`，审计日志类型。
- Create: `frontend/src/domain/chat.ts`，Chat 会话、消息、资源和 SSE 事件类型。
- Create: `frontend/src/infrastructure/api/client.ts`，统一 HTTP 请求、错误解析、认证 header 注入。
- Create: `frontend/src/infrastructure/api/authApi.ts`，PKCE、Keycloak URL、Token exchange、logout URL。
- Create: `frontend/src/infrastructure/api/userApi.ts`，当前用户和管理员用户接口。
- Create: `frontend/src/infrastructure/api/permissionApi.ts`，操作员权限和管理员权限更新接口。
- Create: `frontend/src/infrastructure/api/llmApi.ts`，Provider 和 Model 接口。
- Create: `frontend/src/infrastructure/api/auditApi.ts`，审计日志接口。
- Create: `frontend/src/infrastructure/api/chatApi.ts`，Chat session 和 SSE message 接口。
- Create: `frontend/src/application/useAuth.ts`，登录、回调、会话恢复、退出登录。
- Create: `frontend/src/application/useOperatorData.ts`，操作员权限和模型数据。
- Create: `frontend/src/application/useAdminData.ts`，管理员用户、权限、LLM、审计数据和提交动作。
- Create: `frontend/src/application/useChatOps.ts`，Chat 会话、消息发送、SSE 结果合并。
- Create: `frontend/src/interfaces/components/*.tsx`，基础表格、状态标签、错误提示、空态、表单区。
- Create: `frontend/src/interfaces/layout/AppLayout.tsx`，控制台导航和顶部栏。
- Create: `frontend/src/interfaces/pages/*.tsx`，登录、回调、操作员和管理员页面。
- Modify: `frontend/src/App.tsx`，从静态演示改为认证入口和页面路由状态。
- Modify: `frontend/src/styles.css`，TO B 控制台样式。
- Modify: `README.md`，更新 Frontend 验证命令和 Keycloak 环境变量。
- Modify: `docs/product/requirements.md`，更新 Frontend 当前实现状态。
- Modify: `docs/reference/api-design.md`，更新 Frontend 真实 API 集成状态。
- Modify: `docs/developer/developer-guide.md`，补充 Frontend 分层和测试说明。
- Modify: `docs/operations/deployment-guide.md`，补充 Keycloak 前端配置。

## Task 1: 测试工具链和领域类型

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Create: `frontend/src/domain/auth.ts`
- Create: `frontend/src/domain/user.ts`
- Create: `frontend/src/domain/permission.ts`
- Create: `frontend/src/domain/llm.ts`
- Create: `frontend/src/domain/audit.ts`
- Create: `frontend/src/domain/chat.ts`
- Create: `frontend/src/domain/permission.test.ts`

- [ ] **Step 1: 安装测试依赖**

Run:

```powershell
cd frontend
npm install -D vitest jsdom @testing-library/react @testing-library/user-event
```

Expected: `package.json` 和 `package-lock.json` 更新，`devDependencies` 包含 `vitest`、`jsdom`、`@testing-library/react`、`@testing-library/user-event`。

- [ ] **Step 2: 增加测试脚本**

Update `frontend/package.json` scripts:

```json
{
  "scripts": {
    "dev": "vite --host 0.0.0.0",
    "build": "tsc && vite build",
    "preview": "vite preview --host 0.0.0.0",
    "test": "vitest run --environment jsdom"
  }
}
```

- [ ] **Step 3: 写失败测试**

Create `frontend/src/domain/permission.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { buildPermissionPayload, parseVerbs } from "./permission";

describe("permission domain helpers", () => {
  it("splits comma separated verbs and removes blanks", () => {
    expect(parseVerbs("get, list,watch, ")).toEqual(["get", "list", "watch"]);
  });

  it("builds backend permission update payload", () => {
    const payload = buildPermissionPayload([
      {
        namespace: "dev",
        apiGroup: "",
        resource: "pods",
        verbsText: "get,list,watch"
      }
    ]);

    expect(payload).toEqual({
      permissions: [
        {
          namespace: "dev",
          apiGroup: "",
          resource: "pods",
          verbs: ["get", "list", "watch"]
        }
      ]
    });
  });
});
```

- [ ] **Step 4: 运行测试确认失败**

Run:

```powershell
cd frontend
npm test -- src/domain/permission.test.ts
```

Expected: FAIL，错误包含 `Failed to resolve import "./permission"` 或 `buildPermissionPayload` 未定义。

- [ ] **Step 5: 实现领域类型和权限工具**

Create `frontend/src/domain/permission.ts`:

```ts
export type Permission = {
  id?: string;
  namespace: string;
  apiGroup: string;
  resource: string;
  verbs: string[];
  enabled?: boolean;
};

export type PermissionFormRow = {
  namespace: string;
  apiGroup: string;
  resource: string;
  verbsText: string;
};

export type UpdatePermissionsRequest = {
  permissions: Array<{
    namespace: string;
    apiGroup: string;
    resource: string;
    verbs: string[];
  }>;
};

export function parseVerbs(value: string): string[] {
  return value
    .split(",")
    .map((verb) => verb.trim())
    .filter(Boolean);
}

export function buildPermissionPayload(rows: PermissionFormRow[]): UpdatePermissionsRequest {
  return {
    permissions: rows
      .filter((row) => row.namespace.trim() && row.resource.trim())
      .map((row) => ({
        namespace: row.namespace.trim(),
        apiGroup: row.apiGroup.trim(),
        resource: row.resource.trim(),
        verbs: parseVerbs(row.verbsText)
      }))
  };
}
```

Create `frontend/src/domain/auth.ts`:

```ts
export type AuthMode = "dev" | "keycloak";

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
  tokenType: string;
};

export type CurrentUser = {
  id: string;
  username: string;
  displayName?: string;
  email?: string;
  role: "admin" | "operator" | string;
  status?: string;
};
```

Create `frontend/src/domain/user.ts`:

```ts
export type User = {
  id: string;
  username: string;
  displayName: string;
  email?: string;
  role: "admin" | "operator";
  status: string;
};

export type CreateUserRequest = {
  username: string;
  email: string;
  role: "admin" | "operator";
  displayName: string;
};
```

Create `frontend/src/domain/llm.ts`:

```ts
export type Provider = {
  id: string;
  name: string;
  protocol: "openai" | "anthropic" | string;
  baseUrl: string;
  enabled: boolean;
  apiKeyConfigured: boolean;
};

export type CreateProviderRequest = {
  name: string;
  protocol: "openai" | "anthropic";
  baseUrl: string;
  apiKey: string;
  enabled: boolean;
};

export type Model = {
  id: string;
  providerId: string;
  modelName: string;
  displayName: string;
  supportsTools: boolean;
  supportsStreaming: boolean;
  enabled: boolean;
};

export type CreateModelRequest = {
  providerId: string;
  modelName: string;
  displayName: string;
  supportsTools: boolean;
  supportsStreaming: boolean;
  enabled: boolean;
};
```

Create `frontend/src/domain/audit.ts`:

```ts
export type AuditLog = {
  id: string;
  actorUserId: string;
  action: string;
  targetType: string;
  targetId: string;
  namespace?: string;
  resource?: string;
  verb?: string;
  allowed: boolean;
  reason: string;
  createdAt: string;
};
```

Create `frontend/src/domain/chat.ts`:

```ts
export type ChatSession = {
  id: string;
  userId: string;
  status: string;
  createdAt: string;
};

export type ChatResource = {
  namespace?: string;
  kind?: string;
  name?: string;
  phase?: string;
  reason?: string;
  message?: string;
  restartCount?: number;
  node?: string;
};

export type ChatResult = {
  messageId?: string;
  summary?: string;
  resources?: ChatResource[];
  error?: string;
};

export type ChatMessage = {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  resources?: ChatResource[];
  pending?: boolean;
};
```

- [ ] **Step 6: 运行测试确认通过**

Run:

```powershell
cd frontend
npm test -- src/domain/permission.test.ts
```

Expected: PASS。

- [ ] **Step 7: 提交**

Run:

```powershell
git add frontend/package.json frontend/package-lock.json frontend/src/domain
git commit -m "feat(frontend): add domain types and test setup"
```

## Task 2: 配置、Keycloak PKCE 和 API 客户端

**Files:**
- Create: `frontend/src/config.ts`
- Create: `frontend/src/infrastructure/api/authApi.ts`
- Create: `frontend/src/infrastructure/api/client.ts`
- Create: `frontend/src/infrastructure/api/authApi.test.ts`
- Create: `frontend/src/infrastructure/api/client.test.ts`

- [ ] **Step 1: 写 Keycloak PKCE 失败测试**

Create `frontend/src/infrastructure/api/authApi.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { buildAuthorizeUrl, verifyCallbackState } from "./authApi";

describe("auth api", () => {
  it("builds keycloak authorize url with pkce params", () => {
    const url = buildAuthorizeUrl(
      {
        authMode: "keycloak",
        keycloakUrl: "http://localhost:8089",
        keycloakRealm: "k8s-ai",
        keycloakClientId: "k8s-ai-frontend",
        keycloakRedirectUri: "http://localhost:5173/auth/callback"
      },
      "challenge-001",
      "state-001"
    );

    expect(url).toContain("http://localhost:8089/realms/k8s-ai/protocol/openid-connect/auth");
    expect(url).toContain("client_id=k8s-ai-frontend");
    expect(url).toContain("code_challenge=challenge-001");
    expect(url).toContain("code_challenge_method=S256");
    expect(url).toContain("state=state-001");
  });

  it("rejects callback when state does not match", () => {
    expect(() => verifyCallbackState("actual", "expected")).toThrow("登录状态校验失败");
  });
});
```

- [ ] **Step 2: 写 API 客户端失败测试**

Create `frontend/src/infrastructure/api/client.test.ts`:

```ts
import { describe, expect, it, vi } from "vitest";
import { ApiError, apiRequest, buildAuthHeaders } from "./client";

describe("api client", () => {
  it("adds authorization header in keycloak mode", () => {
    expect(buildAuthHeaders({ mode: "keycloak", accessToken: "token-001" })).toEqual({
      Authorization: "Bearer token-001"
    });
  });

  it("adds demo headers in dev mode", () => {
    expect(buildAuthHeaders({ mode: "dev", demoUser: "admin", demoRole: "admin" })).toEqual({
      "X-Demo-User": "admin",
      "X-Demo-Role": "admin"
    });
  });

  it("throws backend error payload", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: false,
        status: 400,
        json: async () => ({
          error: { code: "INVALID_REQUEST", message: "Invalid request body.", requestId: "req-001" }
        })
      }))
    );

    await expect(apiRequest("/api/demo", { auth: { mode: "dev" } })).rejects.toMatchObject({
      code: "INVALID_REQUEST",
      message: "Invalid request body.",
      requestId: "req-001"
    } satisfies Partial<ApiError>);
  });
});
```

- [ ] **Step 3: 运行测试确认失败**

Run:

```powershell
cd frontend
npm test -- src/infrastructure/api/authApi.test.ts src/infrastructure/api/client.test.ts
```

Expected: FAIL，错误包含模块未找到。

- [ ] **Step 4: 实现配置和认证 API**

Create `frontend/src/config.ts`:

```ts
import type { AuthMode } from "./domain/auth";

export type AppConfig = {
  authMode: AuthMode;
  keycloakUrl: string;
  keycloakRealm: string;
  keycloakClientId: string;
  keycloakRedirectUri: string;
};

export const appConfig: AppConfig = {
  authMode: (import.meta.env.VITE_AUTH_MODE as AuthMode | undefined) ?? "dev",
  keycloakUrl: import.meta.env.VITE_KEYCLOAK_URL ?? "http://localhost:8089",
  keycloakRealm: import.meta.env.VITE_KEYCLOAK_REALM ?? "k8s-ai",
  keycloakClientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID ?? "k8s-ai-frontend",
  keycloakRedirectUri: import.meta.env.VITE_KEYCLOAK_REDIRECT_URI ?? `${window.location.origin}/auth/callback`
};
```

Create `frontend/src/infrastructure/api/authApi.ts`:

```ts
import type { AppConfig } from "../../config";
import type { AuthSession } from "../../domain/auth";

const verifierKey = "k8s-ai-pkce-verifier";
const stateKey = "k8s-ai-pkce-state";
const sessionKey = "k8s-ai-auth-session";

function base64Url(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function randomToken(length = 32): string {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return base64Url(bytes);
}

export async function createCodeChallenge(verifier: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  return base64Url(new Uint8Array(digest));
}

export function buildAuthorizeUrl(config: AppConfig, challenge: string, state: string): string {
  const url = new URL(`${config.keycloakUrl}/realms/${config.keycloakRealm}/protocol/openid-connect/auth`);
  url.searchParams.set("client_id", config.keycloakClientId);
  url.searchParams.set("redirect_uri", config.keycloakRedirectUri);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("scope", "openid profile email");
  url.searchParams.set("code_challenge", challenge);
  url.searchParams.set("code_challenge_method", "S256");
  url.searchParams.set("state", state);
  return url.toString();
}

export function savePkce(verifier: string, state: string): void {
  sessionStorage.setItem(verifierKey, verifier);
  sessionStorage.setItem(stateKey, state);
}

export function readPkce(): { verifier: string; state: string } | null {
  const verifier = sessionStorage.getItem(verifierKey);
  const state = sessionStorage.getItem(stateKey);
  return verifier && state ? { verifier, state } : null;
}

export function clearPkce(): void {
  sessionStorage.removeItem(verifierKey);
  sessionStorage.removeItem(stateKey);
}

export function verifyCallbackState(actual: string | null, expected: string): void {
  if (!actual || actual !== expected) {
    throw new Error("登录状态校验失败");
  }
}

export async function exchangeCodeForToken(config: AppConfig, code: string, verifier: string): Promise<AuthSession> {
  const body = new URLSearchParams();
  body.set("grant_type", "authorization_code");
  body.set("client_id", config.keycloakClientId);
  body.set("redirect_uri", config.keycloakRedirectUri);
  body.set("code", code);
  body.set("code_verifier", verifier);

  const response = await fetch(`${config.keycloakUrl}/realms/${config.keycloakRealm}/protocol/openid-connect/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body
  });
  if (!response.ok) {
    throw new Error("Keycloak Token 换取失败");
  }
  const payload = await response.json();
  return {
    accessToken: payload.access_token,
    refreshToken: payload.refresh_token,
    tokenType: payload.token_type ?? "Bearer",
    expiresAt: Date.now() + Number(payload.expires_in ?? 0) * 1000
  };
}

export function saveSession(session: AuthSession): void {
  sessionStorage.setItem(sessionKey, JSON.stringify(session));
}

export function readSession(): AuthSession | null {
  const raw = sessionStorage.getItem(sessionKey);
  return raw ? (JSON.parse(raw) as AuthSession) : null;
}

export function clearSession(): void {
  sessionStorage.removeItem(sessionKey);
}

export function buildLogoutUrl(config: AppConfig): string {
  const url = new URL(`${config.keycloakUrl}/realms/${config.keycloakRealm}/protocol/openid-connect/logout`);
  url.searchParams.set("client_id", config.keycloakClientId);
  url.searchParams.set("post_logout_redirect_uri", window.location.origin);
  return url.toString();
}
```

- [ ] **Step 5: 实现 API 客户端**

Create `frontend/src/infrastructure/api/client.ts`:

```ts
export type ApiAuth =
  | { mode: "dev"; demoUser?: string; demoRole?: string; accessToken?: never }
  | { mode: "keycloak"; accessToken: string; demoUser?: never; demoRole?: never };

export type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
    requestId?: string;
  };
};

export class ApiError extends Error {
  code: string;
  requestId?: string;
  status: number;

  constructor(status: number, code: string, message: string, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

export function buildAuthHeaders(auth: ApiAuth): Record<string, string> {
  if (auth.mode === "keycloak") {
    return { Authorization: `Bearer ${auth.accessToken}` };
  }
  return {
    ...(auth.demoUser ? { "X-Demo-User": auth.demoUser } : {}),
    ...(auth.demoRole ? { "X-Demo-Role": auth.demoRole } : {})
  };
}

export async function apiRequest<T>(
  path: string,
  options: { method?: string; body?: unknown; auth: ApiAuth; headers?: Record<string, string> }
): Promise<T> {
  const response = await fetch(path, {
    method: options.method ?? "GET",
    headers: {
      ...buildAuthHeaders(options.auth),
      ...(options.body ? { "Content-Type": "application/json" } : {}),
      ...options.headers
    },
    body: options.body ? JSON.stringify(options.body) : undefined
  });

  if (!response.ok) {
    let payload: ApiErrorPayload = {};
    try {
      payload = await response.json();
    } catch {
      payload = {};
    }
    throw new ApiError(
      response.status,
      payload.error?.code ?? "HTTP_ERROR",
      payload.error?.message ?? `HTTP ${response.status}`,
      payload.error?.requestId
    );
  }

  return (await response.json()) as T;
}
```

- [ ] **Step 6: 运行测试确认通过**

Run:

```powershell
cd frontend
npm test -- src/infrastructure/api/authApi.test.ts src/infrastructure/api/client.test.ts
```

Expected: PASS。

- [ ] **Step 7: 提交**

Run:

```powershell
git add frontend/src/config.ts frontend/src/infrastructure/api/authApi.ts frontend/src/infrastructure/api/client.ts frontend/src/infrastructure/api/*.test.ts
git commit -m "feat(frontend): add auth and api clients"
```

## Task 3: 资源 API 封装和 SSE 解析

**Files:**
- Create: `frontend/src/infrastructure/api/userApi.ts`
- Create: `frontend/src/infrastructure/api/permissionApi.ts`
- Create: `frontend/src/infrastructure/api/llmApi.ts`
- Create: `frontend/src/infrastructure/api/auditApi.ts`
- Create: `frontend/src/infrastructure/api/chatApi.ts`
- Create: `frontend/src/infrastructure/api/chatApi.test.ts`

- [ ] **Step 1: 写 SSE 失败测试**

Create `frontend/src/infrastructure/api/chatApi.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { parseSseChunk } from "./chatApi";

describe("chat api sse parser", () => {
  it("parses data lines and ignores empty lines", () => {
    const events = parseSseChunk('data: {"summary":"ok"}\n\n\n');
    expect(events).toEqual([{ summary: "ok" }]);
  });

  it("keeps raw payload when json parse fails", () => {
    const events = parseSseChunk("data: plain text\n\n");
    expect(events).toEqual([{ raw: "plain text" }]);
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```powershell
cd frontend
npm test -- src/infrastructure/api/chatApi.test.ts
```

Expected: FAIL，错误包含模块未找到或 `parseSseChunk` 未定义。

- [ ] **Step 3: 实现资源 API**

Create `frontend/src/infrastructure/api/userApi.ts`:

```ts
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
```

Create `frontend/src/infrastructure/api/permissionApi.ts`:

```ts
import type { Permission, UpdatePermissionsRequest } from "../../domain/permission";
import { apiRequest, type ApiAuth } from "./client";

export function listOperatorPermissions(auth: ApiAuth): Promise<Permission[]> {
  return apiRequest<Permission[]>("/api/operator/permissions", { auth });
}

export function updateUserPermissions(auth: ApiAuth, userId: string, body: UpdatePermissionsRequest): Promise<Permission[]> {
  return apiRequest<Permission[]>(`/api/admin/users/${userId}/permissions`, { method: "PUT", body, auth });
}
```

Create `frontend/src/infrastructure/api/llmApi.ts`:

```ts
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
```

Create `frontend/src/infrastructure/api/auditApi.ts`:

```ts
import type { AuditLog } from "../../domain/audit";
import { apiRequest, type ApiAuth } from "./client";

export function listAuditLogs(auth: ApiAuth): Promise<AuditLog[]> {
  return apiRequest<AuditLog[]>("/api/admin/audit-logs", { auth });
}
```

- [ ] **Step 4: 实现 Chat API 和 SSE 解析**

Create `frontend/src/infrastructure/api/chatApi.ts`:

```ts
import type { ChatResult, ChatSession } from "../../domain/chat";
import { buildAuthHeaders, type ApiAuth } from "./client";

export type RawSseEvent = ChatResult | { raw: string };

export function parseSseChunk(chunk: string): RawSseEvent[] {
  return chunk
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice(5).trim())
    .filter(Boolean)
    .map((data) => {
      try {
        return JSON.parse(data) as ChatResult;
      } catch {
        return { raw: data };
      }
    });
}

export async function createChatSession(auth: ApiAuth): Promise<ChatSession> {
  const { apiRequest } = await import("./client");
  return apiRequest<ChatSession>("/api/operator/chat/sessions", { method: "POST", auth });
}

export async function sendChatMessage(
  auth: ApiAuth,
  sessionId: string,
  body: { modelId: string; content: string },
  onEvent: (event: RawSseEvent) => void
): Promise<void> {
  const response = await fetch(`/api/operator/chat/sessions/${sessionId}/messages`, {
    method: "POST",
    headers: {
      ...buildAuthHeaders(auth),
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body)
  });
  if (!response.ok || !response.body) {
    onEvent({ error: `Chat 请求失败：HTTP ${response.status}` });
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    parseSseChunk(decoder.decode(value, { stream: true })).forEach(onEvent);
  }
}
```

- [ ] **Step 5: 运行测试确认通过**

Run:

```powershell
cd frontend
npm test -- src/infrastructure/api/chatApi.test.ts
```

Expected: PASS。

- [ ] **Step 6: 提交**

Run:

```powershell
git add frontend/src/infrastructure/api
git commit -m "feat(frontend): add backend api wrappers"
```

## Task 4: 应用层 Hooks

**Files:**
- Create: `frontend/src/application/useAuth.ts`
- Create: `frontend/src/application/useOperatorData.ts`
- Create: `frontend/src/application/useAdminData.ts`
- Create: `frontend/src/application/useChatOps.ts`

- [ ] **Step 1: 写 Auth Hook 行为测试**

Create `frontend/src/application/useAuth.test.tsx`:

```tsx
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useAuth } from "./useAuth";

vi.mock("../config", () => ({
  appConfig: {
    authMode: "dev",
    keycloakUrl: "http://localhost:8089",
    keycloakRealm: "k8s-ai",
    keycloakClientId: "k8s-ai-frontend",
    keycloakRedirectUri: "http://localhost:5173/auth/callback"
  }
}));

vi.mock("../infrastructure/api/userApi", () => ({
  getCurrentUser: vi.fn(async () => ({
    id: "user-demo",
    username: "demo",
    displayName: "Demo",
    role: "operator",
    status: "active"
  }))
}));

describe("useAuth", () => {
  it("loads dev user on mount", async () => {
    const { result } = renderHook(() => useAuth());
    await waitFor(() => expect(result.current.user?.username).toBe("demo"));
    expect(result.current.auth.mode).toBe("dev");
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```powershell
cd frontend
npm test -- src/application/useAuth.test.tsx
```

Expected: FAIL，错误包含 `useAuth` 模块不存在。

- [ ] **Step 3: 实现应用 Hooks**

Create `frontend/src/application/useAuth.ts`:

```ts
import { useCallback, useEffect, useMemo, useState } from "react";
import { appConfig } from "../config";
import type { CurrentUser } from "../domain/auth";
import {
  buildAuthorizeUrl,
  buildLogoutUrl,
  clearPkce,
  clearSession,
  createCodeChallenge,
  exchangeCodeForToken,
  randomToken,
  readPkce,
  readSession,
  savePkce,
  saveSession,
  verifyCallbackState
} from "../infrastructure/api/authApi";
import type { ApiAuth } from "../infrastructure/api/client";
import { getCurrentUser } from "../infrastructure/api/userApi";

export function useAuth() {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const auth: ApiAuth = useMemo(() => {
    if (appConfig.authMode === "keycloak") {
      return { mode: "keycloak", accessToken: readSession()?.accessToken ?? "" };
    }
    return { mode: "dev", demoUser: "demo", demoRole: "operator" };
  }, [user]);

  const loadMe = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const current = await getCurrentUser(auth);
      setUser(current);
    } catch (err) {
      setUser(null);
      setError(err instanceof Error ? err.message : "加载当前用户失败");
    } finally {
      setLoading(false);
    }
  }, [auth]);

  useEffect(() => {
    if (appConfig.authMode === "keycloak" && !readSession()) {
      setLoading(false);
      return;
    }
    void loadMe();
  }, [loadMe]);

  const login = useCallback(async () => {
    if (appConfig.authMode === "dev") {
      await loadMe();
      return;
    }
    const verifier = randomToken();
    const state = randomToken(16);
    const challenge = await createCodeChallenge(verifier);
    savePkce(verifier, state);
    window.location.href = buildAuthorizeUrl(appConfig, challenge, state);
  }, [loadMe]);

  const handleCallback = useCallback(async (search: string) => {
    const params = new URLSearchParams(search);
    const code = params.get("code");
    const pkce = readPkce();
    if (!code || !pkce) {
      throw new Error("登录回调参数不完整");
    }
    verifyCallbackState(params.get("state"), pkce.state);
    const session = await exchangeCodeForToken(appConfig, code, pkce.verifier);
    saveSession(session);
    clearPkce();
    await loadMe();
  }, [loadMe]);

  const logout = useCallback(() => {
    clearSession();
    setUser(null);
    if (appConfig.authMode === "keycloak") {
      window.location.href = buildLogoutUrl(appConfig);
    }
  }, []);

  return { user, loading, error, auth, login, logout, handleCallback, reload: loadMe };
}
```

Create the remaining hooks with direct API orchestration:

```ts
// frontend/src/application/useOperatorData.ts
import { useCallback, useEffect, useState } from "react";
import type { Model } from "../domain/llm";
import type { Permission } from "../domain/permission";
import type { ApiAuth } from "../infrastructure/api/client";
import { listOperatorModels } from "../infrastructure/api/llmApi";
import { listOperatorPermissions } from "../infrastructure/api/permissionApi";

export function useOperatorData(auth: ApiAuth) {
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [nextPermissions, nextModels] = await Promise.all([
        listOperatorPermissions(auth),
        listOperatorModels(auth)
      ]);
      setPermissions(nextPermissions);
      setModels(nextModels);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载操作员数据失败");
    } finally {
      setLoading(false);
    }
  }, [auth]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { permissions, models, loading, error, reload };
}
```

```ts
// frontend/src/application/useAdminData.ts
import { useCallback, useEffect, useState } from "react";
import type { AuditLog } from "../domain/audit";
import type { CreateModelRequest, CreateProviderRequest, Model, Provider } from "../domain/llm";
import type { UpdatePermissionsRequest } from "../domain/permission";
import type { CreateUserRequest, User } from "../domain/user";
import { listAuditLogs } from "../infrastructure/api/auditApi";
import type { ApiAuth } from "../infrastructure/api/client";
import { createModel, createProvider, listModels, listProviders, updateModel, updateProvider } from "../infrastructure/api/llmApi";
import { updateUserPermissions } from "../infrastructure/api/permissionApi";
import { createUser, listUsers } from "../infrastructure/api/userApi";

export function useAdminData(auth: ApiAuth) {
  const [users, setUsers] = useState<User[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [nextUsers, nextProviders, nextModels, nextAuditLogs] = await Promise.all([
        listUsers(auth),
        listProviders(auth),
        listModels(auth),
        listAuditLogs(auth)
      ]);
      setUsers(nextUsers);
      setProviders(nextProviders);
      setModels(nextModels);
      setAuditLogs(nextAuditLogs);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载管理员数据失败");
    } finally {
      setLoading(false);
    }
  }, [auth]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    users, providers, models, auditLogs, loading, error, reload,
    createUser: async (body: CreateUserRequest) => { await createUser(auth, body); await reload(); },
    updatePermissions: async (userId: string, body: UpdatePermissionsRequest) => { await updateUserPermissions(auth, userId, body); await reload(); },
    createProvider: async (body: CreateProviderRequest) => { await createProvider(auth, body); await reload(); },
    updateProvider: async (id: string, body: Partial<CreateProviderRequest>) => { await updateProvider(auth, id, body); await reload(); },
    createModel: async (body: CreateModelRequest) => { await createModel(auth, body); await reload(); },
    updateModel: async (id: string, body: Partial<CreateModelRequest>) => { await updateModel(auth, id, body); await reload(); }
  };
}
```

```ts
// frontend/src/application/useChatOps.ts
import { useState } from "react";
import type { ChatMessage, ChatResult, ChatSession } from "../domain/chat";
import type { Model } from "../domain/llm";
import type { ApiAuth } from "../infrastructure/api/client";
import { createChatSession, sendChatMessage } from "../infrastructure/api/chatApi";

export function useChatOps(auth: ApiAuth) {
  const [session, setSession] = useState<ChatSession | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [sending, setSending] = useState(false);

  async function send(content: string, model: Model | undefined) {
    if (!model || !content.trim()) return;
    setSending(true);
    const activeSession = session ?? await createChatSession(auth);
    setSession(activeSession);
    const userMessage: ChatMessage = { id: `user-${Date.now()}`, role: "user", content };
    const assistantMessage: ChatMessage = { id: `assistant-${Date.now()}`, role: "assistant", content: "正在分析...", pending: true };
    setMessages((current) => [...current, userMessage, assistantMessage]);
    await sendChatMessage(auth, activeSession.id, { modelId: model.id, content }, (event) => {
      const result = event as ChatResult;
      setMessages((current) => current.map((message) => {
        if (message.id !== assistantMessage.id) return message;
        if (result.error) return { ...message, content: result.error, pending: false };
        if (result.summary) return { ...message, content: result.summary, resources: result.resources ?? [], pending: false };
        if ("raw" in event) return { ...message, content: event.raw, pending: true };
        return message;
      }));
    });
    setSending(false);
  }

  return { session, messages, sending, send };
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```powershell
cd frontend
npm test -- src/application/useAuth.test.tsx
```

Expected: PASS。

- [ ] **Step 5: 提交**

Run:

```powershell
git add frontend/src/application
git commit -m "feat(frontend): add application hooks"
```

## Task 5: 页面组件和 TO B 样式

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/styles.css`
- Create: `frontend/src/interfaces/components/StatusBadge.tsx`
- Create: `frontend/src/interfaces/components/Notice.tsx`
- Create: `frontend/src/interfaces/components/EmptyState.tsx`
- Create: `frontend/src/interfaces/components/DataTable.tsx`
- Create: `frontend/src/interfaces/layout/AppLayout.tsx`
- Create: `frontend/src/interfaces/pages/LoginPage.tsx`
- Create: `frontend/src/interfaces/pages/AuthCallbackPage.tsx`
- Create: `frontend/src/interfaces/pages/OperatorChatPage.tsx`
- Create: `frontend/src/interfaces/pages/OperatorPermissionsPage.tsx`
- Create: `frontend/src/interfaces/pages/OperatorModelsPage.tsx`
- Create: `frontend/src/interfaces/pages/AdminUsersPage.tsx`
- Create: `frontend/src/interfaces/pages/AdminPermissionsPage.tsx`
- Create: `frontend/src/interfaces/pages/AdminProvidersPage.tsx`
- Create: `frontend/src/interfaces/pages/AdminModelsPage.tsx`
- Create: `frontend/src/interfaces/pages/AdminAuditPage.tsx`

- [ ] **Step 1: 写页面烟雾测试**

Create `frontend/src/App.test.tsx`:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import App from "./App";

vi.mock("./infrastructure/api/userApi", () => ({
  getCurrentUser: vi.fn(async () => ({
    id: "user-demo",
    username: "demo",
    displayName: "Demo",
    role: "operator",
    status: "active"
  }))
}));

vi.mock("./infrastructure/api/permissionApi", () => ({
  listOperatorPermissions: vi.fn(async () => []),
  updateUserPermissions: vi.fn(async () => [])
}));

vi.mock("./infrastructure/api/llmApi", () => ({
  listOperatorModels: vi.fn(async () => []),
  listProviders: vi.fn(async () => []),
  listModels: vi.fn(async () => []),
  createProvider: vi.fn(),
  updateProvider: vi.fn(),
  createModel: vi.fn(),
  updateModel: vi.fn()
}));

vi.mock("./infrastructure/api/auditApi", () => ({
  listAuditLogs: vi.fn(async () => [])
}));

describe("App", () => {
  it("renders operator console after loading current user", async () => {
    render(<App />);
    await waitFor(() => expect(screen.getByText("Chat 运维")).toBeInTheDocument());
    expect(screen.getByText("AI 运维控制台")).toBeInTheDocument();
  });
});
```

Also add `frontend/src/vitest.d.ts`:

```ts
import "@testing-library/jest-dom/vitest";
```

Run:

```powershell
cd frontend
npm install -D @testing-library/jest-dom
npm test -- src/App.test.tsx
```

Expected: FAIL until components are implemented.

- [ ] **Step 2: 实现基础组件**

Implement concise reusable components:

```tsx
// frontend/src/interfaces/components/StatusBadge.tsx
export function StatusBadge({ active, text }: { active: boolean; text: string }) {
  return <span className={`statusBadge ${active ? "ok" : "muted"}`}>{text}</span>;
}
```

```tsx
// frontend/src/interfaces/components/Notice.tsx
export function Notice({ type = "info", children }: { type?: "info" | "error"; children: React.ReactNode }) {
  return <div className={`notice ${type}`}>{children}</div>;
}
```

```tsx
// frontend/src/interfaces/components/EmptyState.tsx
export function EmptyState({ title }: { title: string }) {
  return <div className="emptyState">{title}</div>;
}
```

```tsx
// frontend/src/interfaces/components/DataTable.tsx
export function DataTable({ children }: { children: React.ReactNode }) {
  return <div className="tableWrap"><table>{children}</table></div>;
}
```

- [ ] **Step 3: 实现布局和页面**

Implement pages using existing hooks and domain fields. Keep forms controlled with local state. Use Chinese UI labels:

```tsx
// frontend/src/interfaces/layout/AppLayout.tsx
import type { CurrentUser } from "../../domain/auth";

export type PageKey =
  | "operator-chat" | "operator-permissions" | "operator-models"
  | "admin-users" | "admin-permissions" | "admin-providers" | "admin-models" | "admin-audit";

export function AppLayout({
  user, page, setPage, logout, children
}: {
  user: CurrentUser;
  page: PageKey;
  setPage: (page: PageKey) => void;
  logout: () => void;
  children: React.ReactNode;
}) {
  const isAdmin = user.role === "admin";
  const nav = [
    ["operator-chat", "Chat 运维"],
    ["operator-permissions", "我的权限"],
    ["operator-models", "可用模型"],
    ...(isAdmin ? [
      ["admin-users", "用户管理"],
      ["admin-permissions", "权限配置"],
      ["admin-providers", "LLM Provider"],
      ["admin-models", "LLM Model"],
      ["admin-audit", "审计日志"]
    ] : [])
  ] as Array<[PageKey, string]>;

  return (
    <main className="shell">
      <aside className="sidebar">
        <div><p className="eyebrow">K8S AI Ops</p><h1>AI 运维控制台</h1></div>
        <nav>{nav.map(([key, label]) => <button key={key} className={page === key ? "active" : ""} onClick={() => setPage(key)}>{label}</button>)}</nav>
      </aside>
      <section className="content">
        <header className="topbar">
          <div><strong>{user.displayName || user.username}</strong><span>{user.role}</span></div>
          <button onClick={logout}>退出登录</button>
        </header>
        {children}
      </section>
    </main>
  );
}
```

For each page, render table/form according to the spec. Use these exact headings so acceptance tests can target them:

- `Chat 运维`
- `我的权限`
- `可用模型`
- `用户管理`
- `权限配置`
- `LLM Provider`
- `LLM Model`
- `审计日志`

- [ ] **Step 4: 改造 App.tsx**

Replace static demo with auth and page switch:

```tsx
import { useMemo, useState } from "react";
import { useAdminData } from "./application/useAdminData";
import { useAuth } from "./application/useAuth";
import { useOperatorData } from "./application/useOperatorData";
import { AppLayout, type PageKey } from "./interfaces/layout/AppLayout";
import { LoginPage } from "./interfaces/pages/LoginPage";
import { AuthCallbackPage } from "./interfaces/pages/AuthCallbackPage";
import { OperatorChatPage } from "./interfaces/pages/OperatorChatPage";
import { OperatorPermissionsPage } from "./interfaces/pages/OperatorPermissionsPage";
import { OperatorModelsPage } from "./interfaces/pages/OperatorModelsPage";
import { AdminUsersPage } from "./interfaces/pages/AdminUsersPage";
import { AdminPermissionsPage } from "./interfaces/pages/AdminPermissionsPage";
import { AdminProvidersPage } from "./interfaces/pages/AdminProvidersPage";
import { AdminModelsPage } from "./interfaces/pages/AdminModelsPage";
import { AdminAuditPage } from "./interfaces/pages/AdminAuditPage";

export default function App() {
  const authState = useAuth();
  const [page, setPage] = useState<PageKey>("operator-chat");
  const operator = useOperatorData(authState.auth);
  const admin = useAdminData(authState.auth);
  const isCallback = window.location.pathname === "/auth/callback";

  const content = useMemo(() => {
    if (page === "operator-chat") return <OperatorChatPage auth={authState.auth} models={operator.models} />;
    if (page === "operator-permissions") return <OperatorPermissionsPage permissions={operator.permissions} loading={operator.loading} error={operator.error} />;
    if (page === "operator-models") return <OperatorModelsPage models={operator.models} loading={operator.loading} error={operator.error} />;
    if (page === "admin-users") return <AdminUsersPage users={admin.users} onCreate={admin.createUser} />;
    if (page === "admin-permissions") return <AdminPermissionsPage users={admin.users} onUpdate={admin.updatePermissions} />;
    if (page === "admin-providers") return <AdminProvidersPage providers={admin.providers} onCreate={admin.createProvider} onUpdate={admin.updateProvider} />;
    if (page === "admin-models") return <AdminModelsPage models={admin.models} providers={admin.providers} onCreate={admin.createModel} onUpdate={admin.updateModel} />;
    return <AdminAuditPage logs={admin.auditLogs} />;
  }, [admin, authState.auth, operator, page]);

  if (isCallback) return <AuthCallbackPage onCallback={authState.handleCallback} />;
  if (authState.loading) return <div className="boot">正在加载...</div>;
  if (!authState.user) return <LoginPage error={authState.error} onLogin={authState.login} />;

  return (
    <AppLayout user={authState.user} page={page} setPage={setPage} logout={authState.logout}>
      {content}
    </AppLayout>
  );
}
```

- [ ] **Step 5: 实现 TO B CSS**

Update `frontend/src/styles.css` with:

```css
:root {
  color: #162128;
  background: #f4f6f8;
  font-family: Inter, "Segoe UI", Arial, sans-serif;
}
* { box-sizing: border-box; }
body { margin: 0; }
button, input, select, textarea { font: inherit; }
.shell { display: grid; grid-template-columns: 260px 1fr; min-height: 100vh; }
.sidebar { background: #152027; color: #f8fbfc; display: flex; flex-direction: column; gap: 28px; padding: 28px 20px; }
.sidebar h1 { font-size: 22px; margin: 0; letter-spacing: 0; }
.eyebrow { color: #7cc7a4; font-size: 12px; font-weight: 700; margin: 0 0 8px; letter-spacing: 0; }
nav { display: grid; gap: 8px; }
button { min-height: 38px; border: 1px solid #cbd5dc; border-radius: 6px; background: #fff; color: #162128; cursor: pointer; padding: 7px 12px; }
.sidebar button { background: transparent; border-color: #31424d; color: #eaf0f2; text-align: left; }
.sidebar button.active { background: #7cc7a4; border-color: #7cc7a4; color: #0d2118; }
.content { padding: 22px; min-width: 0; }
.topbar { align-items: center; display: flex; justify-content: space-between; margin-bottom: 18px; }
.topbar span { color: #667681; margin-left: 10px; }
.workspace { display: grid; gap: 16px; max-width: 1240px; }
.panel { background: #fff; border: 1px solid #dce4e8; border-radius: 8px; padding: 16px; }
.toolbar { align-items: center; display: flex; gap: 12px; justify-content: space-between; }
h2, h3 { margin: 0; letter-spacing: 0; }
input, select, textarea { border: 1px solid #cbd5dc; border-radius: 6px; min-height: 38px; padding: 7px 10px; width: 100%; }
textarea { min-height: 84px; resize: vertical; }
.formGrid { display: grid; gap: 12px; grid-template-columns: repeat(2, minmax(0, 1fr)); }
.formRow { display: grid; gap: 6px; }
.tableWrap { overflow-x: auto; }
table { border-collapse: collapse; min-width: 760px; width: 100%; }
th, td { border-bottom: 1px solid #e5ebef; padding: 10px; text-align: left; vertical-align: top; }
th { color: #586873; font-size: 13px; font-weight: 700; }
.statusBadge { border-radius: 999px; display: inline-flex; font-size: 12px; padding: 3px 8px; }
.statusBadge.ok { background: #e6f5ed; color: #17643a; }
.statusBadge.muted { background: #eef2f5; color: #60717c; }
.notice { border-radius: 6px; padding: 10px 12px; }
.notice.info { background: #edf5ff; color: #17456f; }
.notice.error { background: #fff1f0; color: #9f1d1d; }
.emptyState { border: 1px dashed #cbd5dc; border-radius: 8px; color: #667681; padding: 28px; text-align: center; }
.chatMessages { display: grid; gap: 10px; }
.message { border-radius: 8px; max-width: 820px; padding: 12px; }
.message.user { background: #e8f5ef; justify-self: end; }
.message.assistant { background: #eef2f5; }
.loginPage, .boot { align-items: center; display: flex; min-height: 100vh; justify-content: center; padding: 24px; }
.loginPanel { background: #fff; border: 1px solid #dce4e8; border-radius: 8px; max-width: 420px; padding: 24px; width: 100%; }
@media (max-width: 860px) {
  .shell { grid-template-columns: 1fr; }
  .sidebar { gap: 16px; }
  .formGrid { grid-template-columns: 1fr; }
}
```

- [ ] **Step 6: 运行测试和构建**

Run:

```powershell
cd frontend
npm test -- src/App.test.tsx
npm run build
```

Expected: PASS and build completes.

- [ ] **Step 7: 提交**

Run:

```powershell
git add frontend/src
git commit -m "feat(frontend): build mvp console pages"
```

## Task 6: 文档同步和最终验证

**Files:**
- Modify: `README.md`
- Modify: `docs/product/requirements.md`
- Modify: `docs/reference/api-design.md`
- Modify: `docs/developer/developer-guide.md`
- Modify: `docs/operations/deployment-guide.md`

- [ ] **Step 1: 更新 README 验证命令和 Keycloak 配置**

Add Frontend env example to `README.md` local validation section:

```powershell
cd frontend
$env:VITE_AUTH_MODE='dev'
npm install
npm test
npm run build
```

Add Keycloak mode example:

```powershell
$env:VITE_AUTH_MODE='keycloak'
$env:VITE_KEYCLOAK_URL='http://localhost:8089'
$env:VITE_KEYCLOAK_REALM='k8s-ai'
$env:VITE_KEYCLOAK_CLIENT_ID='k8s-ai-frontend'
$env:VITE_KEYCLOAK_REDIRECT_URI='http://localhost:5173/auth/callback'
npm run dev
```

- [ ] **Step 2: 更新产品和 API 文档当前状态**

In `docs/product/requirements.md`, move `Frontend 真实 API 集成` and `Keycloak 登录流程与 Frontend 对接` from “尚未实现” to “已实现”.

In `docs/reference/api-design.md`, update current implementation status to say Frontend calls the listed APIs without changing paths or fields.

- [ ] **Step 3: 更新开发和部署文档**

In `docs/developer/developer-guide.md`, add Frontend layering note:

```markdown
Frontend 按 `domain`、`application`、`infrastructure/api`、`interfaces` 组织。新增页面应先定义领域类型和 API 契约，再在应用层编排状态，最后在 `interfaces` 中实现 UI。
```

In `docs/operations/deployment-guide.md`, add Vite env variables from the spec under Frontend deployment configuration.

- [ ] **Step 4: 最终验证**

Run:

```powershell
cd frontend
npm test
npm run build
```

Expected: tests pass and build completes.

Run:

```powershell
git status --short
```

Expected: only intended documentation files are modified before commit.

- [ ] **Step 5: 提交**

Run:

```powershell
git add README.md docs/product/requirements.md docs/reference/api-design.md docs/developer/developer-guide.md docs/operations/deployment-guide.md
git commit -m "docs: update frontend mvp usage"
```

## 自查结果

- 规格覆盖：计划覆盖 Keycloak 登录、dev 模式、API 客户端、SSE、操作员页面、管理员页面、TO B 样式、测试和文档同步。
- 完整性扫描：计划不包含未完成标记或未定义的后续步骤。
- 类型一致性：`AuthSession`、`CurrentUser`、`Permission`、`Provider`、`Model`、`AuditLog`、`ChatSession`、`ChatMessage` 在领域层定义，并被 API、Hook 和页面任务复用。
