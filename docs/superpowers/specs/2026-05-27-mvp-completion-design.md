# K8S AI Ops MVP 补齐方案设计

## 概述

本文档是在当前架构骨架基础上，将系统推进到可演示 MVP 的完整实施设计。涵盖数据库补齐、LLM+MCP 调用链打通、gRPC 内部 RPC、Keycloak 认证、Traefik Ingress 网关、前端接入和收尾优化。

## 总阶段划分

| 阶段 | 名称 | 前置依赖 |
|------|------|---------|
| 一 | 数据库和 Store 层补齐 | 无 |
| 二 | LLM Provider + MCP Client + 授权校验 | 阶段一 |
| 三 | MCP Server 真实 K8S 集成 + gRPC RPC | 阶段二 |
| 四 | Keycloak 认证 | 阶段一 |
| 五 | Traefik Ingress + Helm 架构重组 | 阶段一 |
| 六 | 前端接入真实 API | 阶段四、五 |
| 七 | 收尾优化 | 全部前置 |

---

## 阶段一：数据库和 Store 层补齐

### 新增表

在 PostgresStore 的 `InitSchema` 中添加以下表：

**user_llm_bindings**
```sql
CREATE TABLE IF NOT EXISTS user_llm_bindings (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    model_id TEXT NOT NULL,
    is_default BOOLEAN NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);
```

**k8s_service_accounts**
```sql
CREATE TABLE IF NOT EXISTS k8s_service_accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    namespace TEXT NOT NULL,
    service_account_name TEXT NOT NULL,
    token_secret_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

**chat_sessions**
```sql
CREATE TABLE IF NOT EXISTS chat_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    model_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

**chat_messages**
```sql
CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    tool_name TEXT NOT NULL DEFAULT '',
    tool_args_json TEXT NOT NULL DEFAULT '',
    tool_result_json TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);
```

### Store 接口新增方法

```go
type Store interface {
    // 已有方法...

    // Chat 会话
    CreateChatSession(session ChatSession) ChatSession
    GetChatSession(id string) (ChatSession, bool)
    ListUserChatSessions(userID string) []ChatSession

    // Chat 消息
    AppendChatMessage(msg ChatMessage) ChatMessage
    ListSessionMessages(sessionID string) []ChatMessage

    // LLM 绑定
    SetUserLLMBindings(userID string, modelIDs []string, defaultModelID string) []UserLLMBinding
    GetUserLLMBindings(userID string) []UserLLMBinding

    // ServiceAccount 绑定
    SaveServiceAccountBinding(binding ServiceAccountBinding) ServiceAccountBinding
    GetUserServiceAccountBindings(userID string) []ServiceAccountBinding
    GetServiceAccountToken(userID string) (token string, namespace string, error)
}
```

### 涉及文件

- `backend/internal/store/store.go` — 接口定义
- `backend/internal/store/models.go` — 模型无需改动（已有 ChatSession, ChatMessage, UserLLMBinding, ServiceAccountBinding）
- `backend/internal/store/memory.go` — 内存实现
- `backend/internal/store/postgres.go` — PostgreSQL 实现 + schema 初始化
- `backend/internal/store/memory_test.go` — 新增测试

---

## 阶段二：LLM Provider + MCP Client + 授权校验

### LLM Provider 接口

新建 `backend/internal/llm/provider.go`（替换当前类型定义文件为完整接口）：

```go
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

type ChatRequest struct {
    Model    string
    System   string
    Messages []Message
    Tools    []ToolDefinition
}

type Message struct {
    Role    string // user, assistant, tool
    Content string
}

type ChatResponse struct {
    Content  string
    ToolCall *ToolCall
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments map[string]any
}
```

### OpenAI 适配器

新建 `backend/internal/llm/openai.go`：
- 调用 OpenAI chat completions API
- 支持工具调用解析
- 支持 streaming（可选）

### Anthropic 适配器

新建 `backend/internal/llm/anthropic.go`：
- 调用 Anthropic messages API
- 支持工具调用解析
- 支持 streaming（可选）

### Provider 注册中心

新建 `backend/internal/llm/registry.go`：
- 根据 store 中的 Provider 配置创建对应适配器实例
- 运行时动态切换

### MCP Client

新建 `backend/internal/mcpclient/client.go`：

```go
type Client struct {
    baseURL string
    httpClient *http.Client
}

func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]any) (*ToolResult, error)
```

### Chat Service 重构

修改 `backend/internal/http/router.go` 中的 `createChatMessage`，替换硬编码 mock：

```
用户消息 → Provider.Chat() → 解析 ToolCall 
  → AuthorizeTool() 校验 → MCPClient.CallTool() 
  → 结果送回 Provider.Chat() → 返回最终总结
```

### API Key 加密

新增 `backend/internal/crypto/cipher.go`：
- `Encrypt(plaintext string) (string, error)`
- `Encrypted ciphertext string) (string, error)`
- 使用 AES-256-GCM，key 从环境变量 `APP_ENCRYPTION_KEY` 读取
- 修改 Provider 存储/读取时加解密

### 涉及文件

- 新建 `backend/internal/llm/openai.go`, `anthropic.go`, `registry.go`
- 新建 `backend/internal/mcpclient/client.go`
- 新建 `backend/internal/crypto/cipher.go`
- 修改 `backend/internal/llm/provider.go`
- 修改 `backend/internal/http/router.go`
- 修改 `backend/internal/config/config.go` — 添加 `MCPServerURL`, `EncryptionKey`
- 修改 `go.mod` — 添加依赖

---

## 阶段三：MCP Server 真实 K8S 集成 + gRPC RPC

### gRPC 服务定义

新建 `backend/api/auth.proto`：

```protobuf
service K8SAuthService {
  rpc GetServiceAccountToken (TokenRequest) returns (TokenResponse);
}

message TokenRequest {
  string user_id = 1;
}

message TokenResponse {
  string token = 1;
  string namespace = 2;  // 空字符串表示集群级权限（admin）
}
```

### Backend gRPC Server

新增 `backend/internal/rpc/server.go`：
- 在 Backend 进程中监听 `:8082`（gRPC 端口）
- 实现 `GetServiceAccountToken` — 从 Store 查询用户绑定的 SA token
- token 来源：`k8s_service_accounts` 表关联的 Secret

### MCP Server gRPC Client

新增 `mcp-server/internal/rpc/client.go`：
- 通过环境变量 `BACKEND_GRPC_ADDR` 读取 Backend gRPC 地址（默认 `backend-api:8082`）
- `GetTokenForUser(userID string) (token, namespace string, error)`

### MCP Server 环境变量

MCP Server Deployment 需注入以下环境变量：

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `BACKEND_GRPC_ADDR` | `backend-api:8082` | Backend gRPC 服务地址 |
| `BACKEND_HTTP_ADDR` | `http://backend-api:8080` | Backend HTTP 服务地址（RPC 降级/健康检查用） |
| `HTTP_ADDR` | `:8081` | MCP Server 自身监听端口 |

### MCP Server K8S Client 重构

重写 `mcp-server/internal/k8s/client.go`：
- 删除原有空壳结构
- 每个工具请求根据传入的 token 创建临时的 K8S REST client
- 使用 `client-go` 的 `rest.Config{BearerToken: token}`

### MCP 工具实现

在 `mcp-server/internal/tools/` 下实现 7 个工具：

| 工具 | 路由 | 核心逻辑 |
|------|------|---------|
| `list_pods` | `/tools/list_pods` | `client.CoreV1().Pods(namespace).List()` |
| `get_pod` | `/tools/get_pod` | `client.CoreV1().Pods(namespace).Get(name)` |
| `get_pod_logs` | `/tools/get_pod_logs` | `client.CoreV1().Pods(namespace).GetLogs(name, opts).Stream()` |
| `list_events` | `/tools/list_events` | `client.CoreV1().Events(namespace).List()` |
| `list_deployments` | `/tools/list_deployments` | `client.AppsV1().Deployments(namespace).List()` |
| `restart_deployment` | `/tools/restart_deployment` | patch deployment annotation 触发重启 |
| `list_namespaces` | `/tools/list_namespaces` | 返回提示信息（由 Backend 权限决定可见范围） |

### 内置管理员 SA（Helm）

在 Helm 中新增模板，安装时创建集群级管理员 SA：

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k8s-ai-admin
  namespace: k8s-ai-system
---
apiVersion: v1
kind: Secret
metadata:
  name: k8s-ai-admin-token
  annotations:
    kubernetes.io/service-account.name: k8s-ai-admin
type: kubernetes.io/service-account-token
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: k8s-ai-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: k8s-ai-admin
    namespace: k8s-ai-system
```

### 内置管理员账号（Store 层）

在 `PostgresStore.SeedDemoData` 中添加：
- 内置 admin 用户：`username=admin, password=bcrypt(123456), role=admin`
- 内置 operator 用户：`username=demo, password=bcrypt(demo123), role=operator`
- admin SA token 由 Backend 启动时从 K8S Secret 读取并缓存

### 统一工具响应格式

```json
{
  "success": true,
  "data": { ... },
  "error": ""
}
```

### MCP Server 入参格式（从 Backend 接收）

```json
{
  "user_id": "u123",
  "tool": "list_pods",
  "args": {
    "namespace": "dev"
  }
}
```

MCP Server 收到后先 gRPC 查 Backend，再调用 K8S API。

### 涉及文件

- 新建 `backend/api/auth.proto`
- 新建 `backend/internal/rpc/server.go`
- 新建 `mcp-server/internal/rpc/client.go`
- 新建 `mcp-server/internal/tools/list_pods.go`, `get_pod.go`, `get_pod_logs.go`, `list_events.go`, `list_deployments.go`, `restart_deployment.go`, `list_namespaces.go`
- 重写 `mcp-server/internal/k8s/client.go`
- 重写 `mcp-server/cmd/server/main.go`
- 修改 `backend/cmd/api/main.go` — 同时启动 HTTP + gRPC
- 新建 Helm template `templates/admin-serviceaccount.yaml`
- 修改 `backend/internal/store/postgres.go` — SeedDemoData 添加 admin
- `backend/go.mod` + `mcp-server/go.mod` — 添加 gRPC 和 client-go 依赖

---

## 阶段四：Keycloak 认证

### JWT 校验中间件

新建 `backend/internal/auth/jwt.go`：

```go
func NewJWTMiddleware(issuer, jwksURL string) func(http.Handler) http.Handler
```

职责：
1. 从 `Authorization: Bearer <token>` 提取 token
2. 从 JWKS 端点获取公钥验证签名
3. 校验 issuer、expiration、audience
4. 从 `realm_access.roles` 提取角色
5. 将 userID 和 role 注入请求上下文
6. JWKS 缓存到 Redis（TTL 1 小时）

### Keycloak Admin API

新建 `backend/internal/auth/keycloak.go`：

```go
type KeycloakAdmin struct {
    baseURL    string
    adminUser  string
    adminPass  string
}

func (k *KeycloakAdmin) CreateUser(ctx context.Context, username, email, password string) (keycloakID string, error)
func (k *KeycloakAdmin) DisableUser(ctx context.Context, keycloakID string) error
func (k *KeycloakAdmin) EnableUser(ctx context.Context, keycloakID string) error
```

### HTTP 路由保护

在 `backend/internal/http/router.go` 中添加中间件链：

```go
type contextKey string
const (
    ContextUserID contextKey = "userID"
    ContextRole   contextKey = "role"
)

// 路由权限表
var routePermissions = map[string]string{
    "/api/admin/*": "admin",
    "/api/operator/*": "operator",
}
```

公开路由：`/healthz`、`/api/auth/sync`
认证关闭模式：`AUTH_DISABLED=true` 时跳过中间件（开发/演示用）

### Store 新增认证方法

```go
type Store interface {
    // ... 已有方法
    VerifyPassword(username, password string) (User, bool)  // 内置用户认证（Keycloak 就绪前的过渡方案）
    GetUserByKeycloakID(keycloakID string) (User, bool)     // Keycloak 集成后的映射
}
```

### 涉及文件

- 新建 `backend/internal/auth/jwt.go`, `keycloak.go`
- 修改 `backend/internal/http/router.go`
- 修改 `backend/internal/store/store.go`, `memory.go`, `postgres.go`
- 修改 `backend/internal/config/config.go`

---

## 阶段五：Traefik Ingress + Helm 架构重组

### Helm Chart 新结构

```
deploy/helm/
├── k8s-ai-ops/                          # Umbrella Chart
│   ├── Chart.yaml                       # dependencies: 全部子 chart
│   ├── values.yaml                      # 全局 values 覆写
│   ├── templates/
│   │   ├── _helpers.tpl                 # 通用模板函数
│   │   └── namespace.yaml               # namespace
│   └── charts/
│       ├── k8s-ai-frontend/             # 自有服务子 chart
│       │   ├── Chart.yaml
│       │   ├── values.yaml
│       │   └── templates/
│       │       ├── deployment.yaml
│       │       └── service.yaml
│       ├── k8s-ai-backend/              # 自有服务子 chart
│       │   ├── Chart.yaml
│       │   ├── values.yaml
│       │   └── templates/
│       │       ├── deployment.yaml
│       │       ├── service.yaml
│       │       ├── serviceaccount.yaml
│       │       ├── admin-serviceaccount.yaml
│       │       └── configmap.yaml
│       ├── k8s-ai-mcp-server/           # 自有服务子 chart
│       │   ├── Chart.yaml
│       │   ├── values.yaml
│       │   └── templates/
│       │       └── templates/
│       │           ├── deployment.yaml
│       │           │   env:
│       │           │     - name: BACKEND_GRPC_ADDR
│       │           │       value: "{{ .Values.mcpServer.backendGrpcAddr }}"
│       │           │     - name: BACKEND_HTTP_ADDR
│       │           │       value: "{{ .Values.mcpServer.backendHttpAddr }}"
│       │           │     - name: HTTP_ADDR
│       │           │       value: ":{{ .Values.mcpServer.service.port }}"
│       │           └── service.yaml
│       ├── postgresql/                  # 外部组件子 chart（自包含）
│       │   ├── Chart.yaml
│       │   ├── values.yaml
│       │   └── templates/
│       │       ├── deployment.yaml
│       │       ├── service.yaml
│       │       └── pvc.yaml
│       ├── redis/                       # 外部组件子 chart（自包含）
│       │   ├── Chart.yaml
│       │   ├── values.yaml
│       │   └── templates/
│       │       ├── deployment.yaml
│       │       ├── service.yaml
│       │       └── pvc.yaml
│       ├── keycloak/                    # 外部组件子 chart
│       │   ├── Chart.yaml
│       │   ├── values.yaml
│       │   └── templates/
│       │       ├── deployment.yaml
│       │       ├── service.yaml
│       │       └── secret.yaml
│       └── traefik/                     # 外部组件子 chart（完整 Traefik）
│           ├── Chart.yaml
│           ├── values.yaml
│           └── templates/
│               ├── deployment.yaml
│               ├── service.yaml
│               ├── serviceaccount.yaml
│               ├── clusterrole.yaml
│               ├── clusterrolebinding.yaml
│               ├── crds/               # Traefik CRD
│               └── configmap.yaml
```

### Umbrella Chart 依赖声明

```yaml
# deploy/helm/k8s-ai-ops/Chart.yaml
apiVersion: v2
name: k8s-ai-ops
description: Kubernetes AI operations assistant
type: application
version: 0.2.0
appVersion: "0.2.0"
dependencies:
  - name: k8s-ai-frontend
    version: "0.1.0"
  - name: k8s-ai-backend
    version: "0.1.0"
  - name: k8s-ai-mcp-server
    version: "0.1.0"
  - name: postgresql
    version: "0.1.0"
  - name: redis
    version: "0.1.0"
  - name: keycloak
    version: "0.1.0"
  - name: traefik
    version: "0.1.0"
```

### Ingress 模板

Traefik 安装后，在 umbrella chart 或 backend chart 中创建 Ingress：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: k8s-ai-ingress
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
spec:
  ingressClassName: traefik
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: backend-api
                port:
                  number: 8080
          - path: /auth
            pathType: Prefix
            backend:
              service:
                name: keycloak
                port:
                  number: 8080
```

### 前端 nginx 简化

移除 nginx 中的 `/api/` proxy_pass，只保留静态文件服务：

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;
    location / {
        try_files $uri /index.html;
    }
}
```

### 涉及文件

- 重组 `deploy/helm/` 目录结构
- 新建 7 个子 chart 的 Chart.yaml + values.yaml + templates/
- 新建 umbrella chart 的 Chart.yaml + values.yaml
- 修改 `frontend/nginx.conf`
- 修改 `scripts/` 下的部署脚本以适应新结构

---

## 阶段六：前端接入真实 API

### API 客户端

新建 `frontend/src/api/client.ts`：

```typescript
const API_BASE = window.location.origin;  // 通过 Traefik 访问

async function request<T>(path: string, options?: RequestInit): Promise<T> {
    const token = localStorage.getItem('auth_token');
    const res = await fetch(`${API_BASE}${path}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
            ...options?.headers,
        },
    });
    if (!res.ok) throw new ApiError(await res.json());
    return res.json();
}
```

### 类型定义

新建 `frontend/src/api/types.ts` — 所有 API 请求/响应 TypeScript 类型。

### 页面动态化

管理员页面：
- 用户列表 → `GET /api/admin/users`
- 创建用户 → `POST /api/admin/users`
- 权限编辑 → `PUT /api/admin/users/:id/permissions`
- LLM 管理 → Provider/Model CRUD
- 审计日志 → `GET /api/admin/audit-logs?page=1&pageSize=20`

操作员页面：
- 可用模型 → `GET /api/operator/llm-models`（下拉框动态填充）
- 我的权限 → `GET /api/operator/permissions`
- Chat 会话 → `POST /api/operator/chat/sessions` → 消息发送 `POST .../messages`

### 登录页面

新建 `frontend/src/pages/Login.tsx`：
- 用户名/密码表单
- `POST /api/auth/login`（或 Keycloak OIDC 跳转）
- 存储 token → 跳转到对应角色页面

### 涉及文件

- 新建 `frontend/src/api/client.ts`, `types.ts`
- 新建 `frontend/src/pages/Login.tsx`
- 修改 `frontend/src/App.tsx`
- 可能新增依赖：无（使用原生 fetch）

---

## 阶段七：收尾优化

### 缺失 API 端点

- `GET /api/admin/users/:id` — 单用户详情
- `PUT /api/admin/users/:id/llm-models` — 绑定 LLM 模型给用户
- `PUT /api/admin/users/:id/status` — 禁用/恢复用户
- `GET /api/operator/chat/sessions/:id/events` — 流式事件

### 审计日志查询参数

```go
type AuditLogQuery struct {
    ActorUserID string
    Action      string
    Namespace   string
    Resource    string
    Allowed     *bool
    StartTime   time.Time
    EndTime     time.Time
    Page        int
    PageSize    int
}
```

### PostgreSQL Migration 版本管理

- 引入 `github.com/golang-migrate/migrate/v4`
- 将现有 `InitSchema` 拆分为版本化 migration 文件
- `PostgresStore.InitSchema` 改为执行 `migrate.Up`

### 端到端集成测试

- 使用 `testcontainers-go` 启动 PostgreSQL + Redis
- 测试完整 Chat → LLM(mock) → 授权 → MCP(mock) → 返回的调用链

### 文档同步

- `docs/architecture/system-architecture.md` — 加入 Traefik 组件和 gRPC 通信
- `docs/operations/deployment-guide.md` — 更新 Helm 新结构和 Ingress 配置
- `docs/architecture/chat-mcp-flow.md` — 更新时序图为真实调用链
- `docs/architecture/data-model.md` — 更新表状态
- `docs/developer/developer-guide.md` — 更新已完成/未完成列表

---

## 架构总图（最终状态）

```mermaid
flowchart LR
    Browser["浏览器"] --> Traefik["Traefik Ingress"]

    subgraph K8S["Kubernetes 集群"]
        subgraph SystemNS["k8s-ai-system namespace"]
            Traefik
            FE["Frontend"]
            API["Backend API"]
            MCP["MCP Server"]
            KC["Keycloak"]
            PG["PostgreSQL"]
            Redis["Redis"]

            API -- "gRPC :8082" --> MCP
            MCP -- "HTTP :8081" --> API
        end

        subgraph BizNS["业务 namespace dev/test"]
            SA["操作员 ServiceAccount"]
            Role["Role / RoleBinding"]
        end
    end

    Browser --> FE
    Traefik --> FE
    Traefik --> API
    Traefik --> KC
    API --> PG
    API --> Redis
    API --> LLM["LLM Provider"]
    API -- 创建/同步 --> SA
    MCP -- 使用 SA Token --> K8S["K8S API Server"]
    MCP -- RPC 查 Token --> API
```

---

## 附录 A：服务依赖与通信矩阵

| 源服务 | 目标服务 | 协议 | 地址 | 端口 |
|--------|---------|------|------|------|
| Browser | Traefik | HTTP | traefik | 80 |
| Traefik | Frontend | HTTP | frontend | 80 |
| Traefik | Backend | HTTP | backend-api | 8080 |
| Traefik | Keycloak | HTTP | keycloak | 8080 |
| Backend | PostgreSQL | pgx | postgresql | 5432 |
| Backend | Redis | RESP | redis | 6379 |
| Backend | Keycloak | HTTP | keycloak | 8080 |
| Backend | MCP Server | HTTP | mcp-server | 8081 |
| Backend | K8S API | HTTPS | kube-apiserver | 443 |
| MCP Server | Backend | gRPC | backend-api | 8082 |
| MCP Server | K8S API | HTTPS | kube-apiserver | 443 |

## 附录 B：各 Service 环境变量

### Backend API
| 环境变量 | 默认值 | 来源 | 阶段 |
|---------|--------|------|------|
| `HTTP_ADDR` | `:8080` | values | 已有 |
| `GRPC_ADDR` | `:8082` | values | 三 |
| `MCP_SERVER_URL` | `http://mcp-server:8081` | values | 已有 |
| `STORE_DRIVER` | `postgres` | values | 已有 |
| `CACHE_DRIVER` | `redis` | values | 已有 |
| `DATABASE_URL` | `postgres://k8s_ai:k8s_ai@postgresql:5432/k8s_ai?...` | values | 已有 |
| `REDIS_ADDR` | `redis:6379` | values | 已有 |
| `K8S_RBAC_SYNC_ENABLED` | `true` | values | 已有 |
| `KEYCLOAK_ISSUER` | `http://keycloak:8080/realms/k8s-ai` | values | 四 |
| `KEYCLOAK_ADMIN_USER` | `admin` | values | 四 |
| `KEYCLOAK_ADMIN_PASSWORD` | from Secret | secretRef | 四 |
| `APP_ENCRYPTION_KEY` | from Secret | secretRef | 已有 |
| `AUTH_DISABLED` | `true` | values | 四 |

### MCP Server
| 环境变量 | 默认值 | 来源 | 阶段 |
|---------|--------|------|------|
| `HTTP_ADDR` | `:8081` | values | 已有 |
| `BACKEND_GRPC_ADDR` | `backend-api:8082` | values | 三 |
| `BACKEND_HTTP_ADDR` | `http://backend-api:8080` | values | 三 |

### Frontend
| 环境变量 | 默认值 | 来源 | 阶段 |
|---------|--------|------|------|
| 无（静态文件，nginx 服务） | - | - | - |

## 附录 C：RBAC 权限补充

### Backend 需读取自己 namespace 的 Secret
k8s-ai-backend ServiceAccount 需要在自己 namespace 中有 Secret 读取权限（用于获取 admin SA token）：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: k8s-ai-backend-secret-reader
  namespace: {{ .Values.global.namespace }}
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: k8s-ai-backend-secret-reader
  namespace: {{ .Values.global.namespace }}
roleRef:
  kind: Role
  name: k8s-ai-backend-secret-reader
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: k8s-ai-backend
    namespace: {{ .Values.global.namespace }}
```

### 管理 namespace RBAC（已有逻辑）
Backend 在 `rbac.managedNamespaces` 中管理操作员 SA/Role/RoleBinding 的权限保持不变，迁移到 `k8s-ai-backend` 子 chart。

## 附录 D：gRPC Proto 代码生成策略

- proto 文件位置：`backend/api/auth.proto`
- 生成的 `.pb.go` 文件提交到代码库，不依赖 protoc 运行时环境
- 生成命令文档化到 `docs/developer/developer-guide.md`
- Backend `cmd/api/main.go` 同时启动 HTTP（:8080）和 gRPC（:8082）两个 listener

## 附录 E：MCP Server go.mod 依赖

当前 `mcp-server/go.mod` 只有 module 声明，需要新增：

```
require (
    google.golang.org/grpc v1.71.0
    google.golang.org/protobuf v1.36.6
    k8s.io/api v0.34.2
    k8s.io/apimachinery v0.34.2
    k8s.io/client-go v0.34.2
)
```

## 附录 F：启动顺序与探针

- **PostgreSQL / Redis**：Backend 启动时连接失败则 log.Fatal（已有逻辑）
- **Backend gRPC**：store/cache 就绪后再启动 gRPC server
- **MCP Server**：gRPC 客户端使用自动重连（`grpc.WithBlock` + 后台重试），不阻塞自身启动
- **Traefik**：独立启动，不依赖后端服务就绪

## 附录 G：Shared Secret 归属

`k8s-ai-secrets`（包含 APP_ENCRYPTION_KEY、POSTGRES_PASSWORD、KEYCLOAK_ADMIN_PASSWORD）被多个子 chart 共用，放在 umbrella chart 的 `templates/secret.yaml` 中，不属于任何子 chart。

---

## 数据流：Chat 巡检

```mermaid
sequenceDiagram
    actor User as 操作员
    participant UI as Frontend
    participant Traefik as Traefik
    participant API as Backend API
    participant LLM as LLM Provider
    participant MCP as MCP Server
    participant K8S as K8S API

    User->>UI: 帮我看看集群异常？
    UI->>Traefik: POST /api/operator/chat/sessions/:id/messages
    Traefik->>API: 转发请求
    API->>API: 加载用户权限和模型
    API->>LLM: Chat 请求 + 系统提示词
    LLM-->>API: 工具调用 list_pods
    API->>API: AuthorizeTool 校验
    API->>MCP: POST /tools/list_pods {userId, args}
    MCP->>API: gRPC GetServiceAccountToken
    API-->>MCP: 返回 SA Token
    MCP->>K8S: 使用 SA Token 查询 Pods
    K8S-->>MCP: Pod 列表
    MCP-->>API: 工具结果
    API->>LLM: 发送工具结果
    LLM-->>API: 最终总结
    API->>API: 写审计日志
    API-->>Traefik: JSON 响应
    Traefik-->>UI: 返回结果
    UI->>UI: 渲染总结 + 资源表格
```
