# K8S AI Ops

K8S AI Ops 是一个基于 MCP 的 Kubernetes AI 运维助手。系统提供多角色 Web 控制台：管理员负责用户、Kubernetes 权限和 LLM 模型管理；操作员通过自然语言 Chat 在授权范围内巡检和操作 Kubernetes 资源。

## 快速理解

```mermaid
flowchart LR
  Admin["管理员"] --> API["Backend API"]
  Operator["操作员"] --> UI["React Web UI"]
  UI --> API
  API --> KC["Keycloak"]
  API --> DB["PostgreSQL / Redis"]
  API -->|"gRPC / AgentService.Run"| Agent["Agent Server / Eino"]
  Agent --> LLM["OpenAI / Anthropic"]
  Agent --> MCP["MCP Server"]
  MCP --> K8S["Kubernetes API"]
```

第一阶段目标是交付一个适合面试展示、同时具备企业级架构意识的 MVP：

- Keycloak 认证，支持管理员和操作员。
- 管理员创建用户、分配 Kubernetes namespace 级权限、管理 LLM。
- 操作员通过 Chat 巡检授权范围内的异常 Pod。
- Backend 管理 Chat 历史、权限、审计，并通过 gRPC 调用 Agent Server。
- Agent Server 使用 Eino 执行单轮 agent loop，消费 Backend 传入的 `messages` 和 `runtimeContext`。
- Backend 和 MCP Server 在 LLM 工具调用前后做双层权限校验。
- Backend 可将管理员配置的 namespace 级权限同步为 Kubernetes ServiceAccount、Role、RoleBinding。
- MCP Server 使用操作员 ServiceAccount 调用 Kubernetes。
- Helm 支持本地 tar 镜像包和 registry 镜像两种部署模式。

## 文档入口

请从 [文档中心](docs/INDEX.md) 开始阅读。文档按读者角色分层：

- 快速了解项目：[产品概览](docs/product/overview.md)
- 产品/面试官：[业务需求](docs/product/requirements.md)、[用户流程](docs/product/user-journeys.md)
- 架构/技术负责人：[系统架构](docs/architecture/system-architecture.md)、[权限模型](docs/architecture/permission-model.md)、[Chat 与 MCP 流程](docs/architecture/chat-mcp-flow.md)
- 二开开发者：[二开指南](docs/developer/developer-guide.md)、[API 设计](docs/reference/api-design.md)
- 部署运维：[部署指南](docs/operations/deployment-guide.md)、[日志、审计与排错](docs/operations/observability-and-troubleshooting.md)
- 公有云测试：[公有云 Kubernetes 测试计划](docs/operations/public-cloud-test-plan.md)
- 安全审查：[安全设计](docs/security/security-design.md)

## 本地验证

```bash
cd backend && go test ./...
cd proto && go test ./...
cd agent-server && go test ./...
cd mcp-server && go test ./...
cd frontend && npm install && npm run build
bash -n scripts/bootstrap-local.sh scripts/helm-install.sh scripts/helm-upgrade.sh scripts/uninstall.sh scripts/build-images.sh
```

## 本地 PostgreSQL/Redis 集成测试

在 Windows + WSL Docker 环境中启动本项目专用依赖：

```bash
wsl bash /mnt/e/k8s-agent/scripts/dev-infra-wsl.sh
```

如果 Windows 侧无法访问 WSL Docker 暴露的端口，可以临时保活 WSL 会话：

```bash
wsl bash /mnt/e/k8s-agent/scripts/dev-infra-wsl.sh --hold-seconds 600
```

然后运行集成测试：

```powershell
cd E:\k8s-agent\backend
$env:K8S_AI_TEST_DATABASE_URL='postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable'
$env:K8S_AI_TEST_REDIS_ADDR='localhost:56379'
go test ./internal/store ./internal/cache -count=1 -v
```

启动 Backend 使用真实 PostgreSQL/Redis：

```powershell
$env:STORE_DRIVER='postgres'
$env:CACHE_DRIVER='redis'
$env:DATABASE_URL='postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable'
$env:REDIS_ADDR='localhost:56379'
$env:AGENT_SERVER_ADDR='localhost:8082'
go run ./cmd/api
```

如需在真实 Kubernetes 环境中同步 namespace 级 RBAC，再增加：

```powershell
$env:K8S_RBAC_SYNC_ENABLED='true'
$env:KUBECONFIG='C:\Users\you\.kube\config'
```

## 构建镜像

```bash
scripts/build-images.sh --tag local --output-dir image-tars
```

构建脚本会生成：

- `image-tars/backend-api-amd64.tar`
- `image-tars/agent-server-amd64.tar`
- `image-tars/mcp-server-amd64.tar`
- `image-tars/frontend-amd64.tar`

## 部署

从零启动本地 Kind 集群并部署：

```bash
scripts/bootstrap-local.sh --image-source tar --image-dir image-tars --cluster-name k8s-ai
```

部署到已有集群：

```bash
scripts/helm-install.sh --image-source tar --image-dir image-tars
```

使用镜像仓库升级：

```bash
scripts/helm-upgrade.sh --image-source registry --registry registry.example.com/k8s-ai --tag v1.0.0
```

更多说明见 [部署指南](docs/operations/deployment-guide.md)。

## 当前状态

当前仓库包含企业级分层文档、Backend 骨架、Agent Server 骨架、MCP Server 骨架、Frontend 骨架、Helm Chart、Dockerfile 和部署脚本。Backend 已接入 PostgreSQL、Redis 连通性检查、namespace 级 Kubernetes RBAC 同步，并通过生成的 proto/gRPC 契约调用 Agent Server。后续开发应遵循 [项目协作规则](AGENTS.md)，保持代码和文档一致。
