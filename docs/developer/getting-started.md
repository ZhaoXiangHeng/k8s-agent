# 快速接手

这篇文档面向第一次接手 k8s-agent 仓库的开发者，目标是在最短时间内回答三件事：

1. 这套系统是做什么的
2. 代码都放在哪里
3. 应该怎么跑起来并开始改

## 1. 项目定位

K8S AI Ops 是一个基于 MCP 的 Kubernetes AI 运维助手平台：

- 管理员通过 Web 控制台管理用户、K8s namespace 级权限和 LLM 模型
- 操作员通过自然语言 Chat 在授权范围内巡检和操作 K8s 资源
- Backend 通过 gRPC 调用 Agent Server，Agent Server 使用 Eino 执行 ReAct agent loop
- MCP Server 将 K8s API 封装为标准 MCP 工具，per-user ServiceAccount 隔离
- Helm 支持本地 tar 镜像和 registry 两种部署模式

## 2. 先看哪些目录

```text
k8s-agent/
├── backend/                    # Backend API（Go, DDD 分层）
├── agent-server/               # Agent Server（Go, Eino + gRPC）
├── mcp-server/                 # MCP Server（Go, MCP 协议 + SSE）
├── frontend/                   # React Web UI（TypeScript + Vite）
├── proto/                      # gRPC 契约定义（agent/v1 + identity/v1）
├── deploy/helm/k8s-ai-ops/    # Helm Chart
├── scripts/                    # 构建、部署、卸载脚本
├── docs/                       # 企业级分层文档
└── AGENTS.md                   # 项目协作规则（必读）
```

### 目录职责

- `backend/`：认证授权、业务授权、用户管理、LLM 管理、Chat 编排、审计。DDD 分层：`domain/` → `app/` → `infra/` → `interfaces/`
- `agent-server/`：基于 Eino ADK ChatModelAgent 的无状态 ReAct agent loop，通过 gRPC `RunStream` 对外服务
- `mcp-server/`：将 K8s API 封装为标准 MCP 工具，per-user K8s client 隔离，SSE transport
- `frontend/`：React Web UI，管理员和操作员双控制台，SSE 流式 Chat
- `proto/`：gRPC 契约唯一来源，修改后必须重新生成 Go 代码
- `deploy/helm/`：Helm Chart，支持 tar 和 registry 两种镜像来源

## 3. 关键代码入口

第一次读代码建议从这些文件开始：

### Backend
- `backend/cmd/api/main.go` — 入口，依赖注入、路由注册
- `backend/internal/interfaces/http/routes.go` — HTTP 路由和中间件
- `backend/internal/app/permission_service.go` — 权限管理核心逻辑
- `backend/internal/infra/postgres/` — PostgreSQL 仓储实现
- `backend/internal/infra/k8s/` — K8s RBAC Manager
- `backend/internal/infra/agent/client.go` — gRPC AgentService 客户端

### Agent Server
- `agent-server/cmd/server/main.go` — 入口，初始化 Eino + MCP 客户端 + gRPC server
- `agent-server/internal/eino/runner.go` — ReAct agent runner，实现 RunStream
- `agent-server/internal/eino/mcp/client.go` — MCP Server 客户端，发现并注册工具
- `agent-server/internal/eino/llm/factory.go` — LLM ChatModel 工厂（openai/anthropic）

### MCP Server
- `mcp-server/cmd/server/main.go` — 入口
- `mcp-server/internal/handler/` — MCP 工具处理器（pods/deployments/events/namespaces/logging）
- `mcp-server/internal/k8s/` — Per-user K8s client 工厂
- `mcp-server/internal/identity/client.go` — gRPC IdentityService 客户端

### Frontend
- `frontend/src/App.tsx` — 路由和布局
- `frontend/src/pages/` — 页面组件（AdminConsole, OperatorPermissionsPage, UserManagement 等）
- `frontend/src/infrastructure/api/` — API 调用封装

### Proto
- `proto/agent/v1/agent.proto` — AgentService.RunStream + 13 种 StreamEvent
- `proto/identity/v1/identity.proto` — IdentityService.GetServiceAccount

## 4. 本地开发环境

### 4.1 依赖服务（PostgreSQL + Redis）

在 Windows + WSL Docker 环境中启动：

```bash
wsl bash /mnt/e/k8s-agent/scripts/dev-infra-wsl.sh
```

默认端口：

```text
PostgreSQL: localhost:55432
Redis: localhost:56379
```

### 4.2 启动 Backend（接入真实 DB/Redis）

```powershell
$env:STORE_DRIVER='postgres'
$env:CACHE_DRIVER='redis'
$env:DATABASE_URL='postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable'
$env:REDIS_ADDR='localhost:56379'
$env:AGENT_SERVER_ADDR='localhost:8082'
go run ./backend/cmd/api
```

需要 K8s RBAC 同步时：

```powershell
$env:K8S_RBAC_SYNC_ENABLED='true'
$env:KUBECONFIG='C:\Users\you\.kube\config'
```

### 4.3 启动 Agent Server

```bash
MCP_SERVER_URL=http://localhost:8081/sse go run ./agent-server/cmd/server
```

### 4.4 启动 MCP Server

```bash
IDENTITY_SERVICE_ADDR=localhost:8083 go run ./mcp-server/cmd/server
```

### 4.5 启动 Frontend

```bash
cd frontend && npm install && npm run dev
```

### 4.6 跑测试

```bash
cd backend && go test ./...
cd agent-server && go test ./...
cd mcp-server && go test ./...
cd proto && go test ./...
```

### 4.7 PostgreSQL/Redis 集成测试

```powershell
cd backend
$env:K8S_AI_TEST_DATABASE_URL='postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable'
$env:K8S_AI_TEST_REDIS_ADDR='localhost:56379'
go test ./internal/store ./internal/cache -count=1 -v
```

## 5. 常见开发任务从哪里下手

### 5.1 改用户/权限/LLM 管理逻辑

看：
- `backend/internal/app/permission_service.go`
- `backend/internal/interfaces/http/handler_permissions.go`
- `backend/internal/infra/postgres/`

### 5.2 改 Chat 编排和 SSE 流式

看：
- `backend/internal/interfaces/http/handler_auth.go`（Chat 相关 handler）
- `backend/internal/infra/agent/client.go`（gRPC AgentService client）
- `frontend/src/pages/`（SSE 消费端）

### 5.3 改 Agent loop 和 LLM 调用

看：
- `agent-server/internal/eino/runner.go`
- `agent-server/internal/eino/llm/factory.go`
- `agent-server/internal/eino/config.go`

### 5.4 新增 MCP 工具

看：
- `mcp-server/internal/handler/`
- [扩展开发指南](extension-guide.md)

### 5.5 改 K8s RBAC 同步

看：
- `backend/internal/infra/k8s/`
- `backend/internal/infra/k8s/token.go`
- `deploy/helm/k8s-ai-ops/templates/rbac.yaml`

### 5.6 改部署和 Helm Chart

看：
- `deploy/helm/k8s-ai-ops/`
- `scripts/build-images.sh`
- `scripts/helm-install.sh`

## 6. 接手时最容易踩的坑

- **误以为 Agent Server 有状态**：Agent Server 不保存 Chat 历史，每次请求独立建立 agent loop，多轮上下文由 Backend 传入
- **误以为前端直接调 MCP Server**：前端只调 Backend HTTP API，Agent Server 才调 MCP Server
- **误以为 MCP 工具权限由 Agent Server 校验**：Agent Server 只消费权限快照，实际工具调用前的权限校验在 MCP Server
- **误以为 K8s RBAC 是可选的**：Kubernetes RBAC 是最终权限边界，即使上层校验有缺陷，K8s 也会拒绝越权访问
- **误以为 proto 生成代码可以手动修改**：proto 是契约唯一来源，修改后必须用 `protoc` 重新生成
- **误以为所有功能都已实现**：Keycloak JWT audience 校验、Keycloak Admin API、PostgreSQL migration 版本管理、Redis 业务缓存等尚未完成

## 7. 推荐的接手路径

如果要在半天内快速熟悉项目，建议这样走：

1. 读 [架构总览](./architecture-overview.md)
2. 看 `backend/cmd/api/main.go` 理解依赖注入和启动流程
3. 看 `backend/internal/interfaces/http/routes.go` 理解 API 路由
4. 看 `proto/agent/v1/agent.proto` 理解 gRPC 契约
5. 看 `agent-server/internal/eino/runner.go` 理解 agent loop
6. 看 `mcp-server/internal/handler/` 理解工具实现
7. 跑 `cd backend && go test ./...`
8. 启动全套服务验证
