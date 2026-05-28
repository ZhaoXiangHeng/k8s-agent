# 配置参考

这篇文档面向使用者和运维人员，汇总 k8s-agent 所有服务的关键配置项：环境变量、Helm values、端口和约定。

## 1. Helm values 完整参考

### 1.1 全局配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `global.namespace` | string | `k8s-ai-system` | 所有组件的部署命名空间 |

### 1.2 镜像配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `images.source` | string | `tar` | 镜像来源：`tar`（本地 tar 包）/ `registry`（镜像仓库） |
| `images.registry` | string | `""` | 镜像仓库前缀，`source=registry` 时必填 |
| `images.tag` | string | `local` | 所有服务镜像的统一 tag |
| `images.pullPolicy` | string | `IfNotPresent` | 镜像拉取策略 |
| `images.imagePullSecrets` | list | `[]` | 镜像拉取 Secret 列表 |

### 1.3 Frontend

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `frontend.replicas` | int | `1` | 副本数 |
| `frontend.image.repository` | string | `k8s-ai-frontend` | 镜像名 |
| `frontend.service.port` | int | `80` | ClusterIP 端口 |

### 1.4 Backend API

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `backend.replicas` | int | `1` | 副本数 |
| `backend.image.repository` | string | `k8s-ai-backend` | 镜像名 |
| `backend.service.port` | int | `8080` | HTTP API 端口 |
| `backend.service.grpcPort` | int | `8082` | gRPC IdentityService 端口 |
| `backend.encryptionKey` | string | `change-me-32-byte...` | AES-256-GCM 加密密钥（生产环境必须更换） |
| `backend.storeDriver` | string | `postgres` | 存储驱动：`postgres` / `memory` |
| `backend.cacheDriver` | string | `redis` | 缓存驱动：`redis` / `memory` |
| `backend.rbacSyncEnabled` | bool | `true` | 权限更新后是否同步 K8s RBAC |
| `backend.authMode` | string | `jwt` | 认证模式：`dev`（信任 X-Demo-User 头）/ `jwt`（Keycloak JWT）/ `none` |
| `backend.keycloakIssuer` | string | 集群内 Keycloak 地址 | OIDC Issuer URL，格式 `http://keycloak.k8s-ai-system.svc.cluster.local:8080/auth/realms/k8s-ai` |

### 1.5 Agent Server

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `agentServer.replicas` | int | `1` | 副本数 |
| `agentServer.image.repository` | string | `k8s-ai-agent-server` | 镜像名 |
| `agentServer.service.port` | int | `8082` | gRPC AgentService 端口 |

### 1.6 MCP Server

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `mcpServer.replicas` | int | `1` | 副本数 |
| `mcpServer.image.repository` | string | `k8s-ai-mcp-server` | 镜像名 |
| `mcpServer.service.port` | int | `8081` | SSE 端口 |

### 1.7 Keycloak

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `keycloak.enabled` | bool | `true` | 是否部署内置 Keycloak |
| `keycloak.image` | string | `keycloak:26.0.7` | Keycloak 镜像 |
| `keycloak.adminUser` | string | `admin` | 管理员用户名 |
| `keycloak.adminPassword` | string | `admin` | 管理员密码（生产环境必须更换） |
| `keycloak.realm` | string | `k8s-ai` | Realm 名称 |
| `keycloak.clientId` | string | `k8s-ai-frontend` | OIDC Client ID |
| `keycloak.service.port` | int | `8080` | ClusterIP 端口 |

### 1.8 PostgreSQL

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `postgresql.enabled` | bool | `true` | 是否部署内置 PostgreSQL |
| `postgresql.image` | string | `postgres:16-alpine` | PostgreSQL 镜像 |
| `postgresql.database` | string | `k8s_ai` | 数据库名 |
| `postgresql.username` | string | `k8s_ai` | 数据库用户 |
| `postgresql.password` | string | `k8s_ai` | 数据库密码（生产环境必须更换） |
| `postgresql.service.port` | int | `5432` | ClusterIP 端口 |

### 1.9 Redis

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `redis.enabled` | bool | `true` | 是否部署内置 Redis |
| `redis.image` | string | `redis:7-alpine` | Redis 镜像 |
| `redis.service.port` | int | `6379` | ClusterIP 端口 |

### 1.10 RBAC

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `rbac.managedNamespaces` | list | `[]` | Backend 可管理操作员 RBAC 的 namespace 列表 |
| `rbac.adminServiceAccount.enabled` | bool | `true` | 是否创建 cluster-admin 管理员 SA |
| `rbac.adminServiceAccount.name` | string | `k8s-ai-admin` | 管理员 SA 名称 |

### 1.11 Traefik 网关

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `traefik.enabled` | bool | `true` | 是否部署内置 Traefik |
| `traefik.image` | string | `traefik:v3.6.13` | Traefik 镜像 |
| `traefik.hostPort` | int | `80` | 宿主机端口，映射 web entrypoint (:8000) |

### 1.12 Traefik 路由规则

| 路径 | 优先级 | 目标 Service | 说明 |
|------|--------|-------------|------|
| `/auth/callback` | 300 | `frontend:{port}` | OIDC PKCE 回调，最高优先级 |
| `/auth/` | 200 | `keycloak:{port}` | Keycloak 认证端点 |
| `/api/` | 100 | `backend-api:{port}` | 业务 API |
| `/` | 1 | `frontend:{port}` | 前端 SPA 兜底 |

Agent Server gRPC（`:8082`）和 MCP Server SSE（`:8081`）不创建 IngressRoute，仅通过内部 ClusterIP 通信。

## 2. 服务端口全景

| 组件 | 端口 | 协议 | 暴露方式 | 调用方 |
|------|------|------|----------|--------|
| Traefik | 80 (hostPort) | HTTP | 宿主机 | 用户浏览器 |
| Traefik | 8000 (container) | HTTP | ClusterIP | Traefik web entrypoint |
| Traefik | 8080 (container) | HTTP | ClusterIP | Traefik health/ping |
| Frontend | 80 | HTTP | ClusterIP | Traefik |
| Backend API | 8080 | HTTP | ClusterIP | Traefik |
| Backend API | 8082 | gRPC | ClusterIP | MCP Server（IdentityService） |
| Agent Server | 8082 | gRPC | ClusterIP | Backend API（AgentService） |
| MCP Server | 8081 | SSE | ClusterIP | Agent Server |
| Keycloak | 8080 | HTTP | ClusterIP | Traefik, Backend API |
| PostgreSQL | 5432 | TCP | ClusterIP | Backend API |
| Redis | 6379 | TCP | ClusterIP | Backend API |

## 3. 服务间通信矩阵

```
外部 → Traefik:80 → /              → Frontend:80
外部 → Traefik:80 → /api/*         → Backend API:8080
外部 → Traefik:80 → /auth/*        → Keycloak:8080
外部 → Traefik:80 → /auth/callback → Frontend:80

Backend API  → agent-server:8082      (gRPC AgentService.RunStream)
Backend API  → postgresql:5432         (PostgreSQL)
Backend API  → redis:6379              (Redis)
Backend API  → keycloak:8080           (OIDC discovery/JWKS)
Agent Server → mcp-server:8081/sse     (MCP SSE)
Agent Server → 外部 LLM API            (HTTPS)
MCP Server   → backend-api:8082        (gRPC IdentityService)
MCP Server   → Kubernetes API Server   (client-go, per-user SA)
Backend API  → Kubernetes API Server   (client-go, k8s-ai-backend SA)
```

## 4. Backend API 环境变量

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `HTTP_ADDR` | string | `:8080` | HTTP 监听地址 |
| `GRPC_ADDR` | string | `:8082` | gRPC IdentityService 监听地址 |
| `AGENT_SERVER_ADDR` | string | `agent-server:8082` | Agent Server gRPC 地址 |
| `STORE_DRIVER` | string | `postgres` | 存储驱动 |
| `CACHE_DRIVER` | string | `redis` | 缓存驱动 |
| `DATABASE_URL` | string | 自动拼接 | PostgreSQL 连接字符串 |
| `REDIS_ADDR` | string | `redis:6379` | Redis 地址 |
| `K8S_RBAC_SYNC_ENABLED` | bool | `true` | 是否同步 K8s RBAC |
| `AUTH_MODE` | string | `jwt` | 认证模式 |
| `KEYCLOAK_ISSUER` | string | Keycloak 集群内地址 | OIDC Issuer URL |
| `APP_ENCRYPTION_KEY` | string | Secret 注入 | AES-256-GCM 加密密钥 |

## 5. Agent Server 环境变量

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `GRPC_ADDR` | string | `:8082` | gRPC 监听地址 |
| `MCP_SERVER_URL` | string | `http://mcp-server:8081/sse` | MCP Server SSE 端点 |

## 6. MCP Server 环境变量

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `HTTP_ADDR` | string | `:8081` | SSE 监听地址 |
| `IDENTITY_SERVER_ADDR` | string | `backend-api:8082` | Backend IdentityService gRPC 地址 |

## 7. Frontend 构建时环境变量

| 变量 | 说明 |
|------|------|
| `VITE_API_BASE_URL` | Backend API 地址 |
| `VITE_KEYCLOAK_URL` | Keycloak 地址 |
| `VITE_KEYCLOAK_REALM` | Keycloak Realm |
| `VITE_KEYCLOAK_CLIENT_ID` | Keycloak Client ID |

## 8. ServiceAccount 命名约定

Backend RBAC Manager 动态创建的对象：

| 对象 | 命名格式 | 示例 |
|------|----------|------|
| ServiceAccount | `k8s-ai-operator-{userId}` | `k8s-ai-operator-u123` |
| Role | `k8s-ai-role-{userId}-{namespace}` | `k8s-ai-role-u123-dev` |
| RoleBinding | `k8s-ai-binding-{userId}-{namespace}` | `k8s-ai-binding-u123-dev` |

托管标签：`app.kubernetes.io/managed-by=k8s-ai-ops-backend`

控制面 ServiceAccount：

| 对象 | 名称 | 用途 |
|------|------|------|
| ServiceAccount | `k8s-ai-backend` | Backend Pod 身份 |
| ServiceAccount | `k8s-ai-admin` | 管理员 SA（cluster-admin） |
| ClusterRoleBinding | `k8s-ai-admin-cluster-admin` | 管理员 SA 绑定 |

## 9. 本地开发端口映射

| 服务 | 端口 | 用途 |
|------|------|------|
| Backend API | `8080` | HTTP API |
| Agent Server | `8082` | gRPC AgentService |
| MCP Server | `8081` | SSE |
| Frontend Dev | `5173` | Vite dev server |
| PostgreSQL (WSL) | `55432` | 开发数据库 |
| Redis (WSL) | `56379` | 开发缓存 |
