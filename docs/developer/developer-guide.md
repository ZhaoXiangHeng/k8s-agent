# 二开指南

## Agent Server 与 proto 目录

```text
proto/
  agent/v1/agent.proto
  agent/v1/agent.pb.go
  agent/v1/agent_grpc.pb.go
agent-server/
  cmd/server/main.go
  internal/eino/
  internal/server/
```

`proto/agent/v1/agent.proto` 是 Backend 与 Agent Server 之间 gRPC 契约的唯一来源。修改 proto 后必须重新生成 Go 代码并提交生成物。

常用验证命令：

```bash
cd proto && go test ./...
cd agent-server && go test ./...
cd backend && go test ./internal/agent ./internal/config ./internal/http
cd mcp-server && go test ./...
```

## 适用读者

本文面向需要继续开发本项目的工程师，重点说明代码结构、模块边界、扩展点、测试和文档维护规则。

## 仓库结构

```text
backend/                 Backend API
mcp-server/              MCP Server
frontend/                React Web UI
deploy/helm/k8s-ai-ops/  Helm Chart
scripts/                 构建、部署、卸载脚本
docs/                    企业级分层文档
```

## Backend 模块边界

```mermaid
flowchart LR
  HTTP["internal/http"] --> Auth["internal/auth"]
  HTTP --> Chat["internal/chat"]
  HTTP --> Store["internal/store"]
  Chat --> LLM["internal/llm"]
  Chat --> MCP["internal/mcpclient"]
  Chat --> Audit["internal/audit"]
  Store --> DB["PostgreSQL"]
  Auth --> Keycloak["Keycloak"]
  MCP --> MCPServer["MCP Server"]
  K8S["internal/k8s"] --> APIServer["Kubernetes API"]
```

当前代码是骨架阶段，已实现：

- 配置加载。
- HTTP health 和占位接口。
- 领域模型。
- 权限匹配。
- RBAC 命名和 RoleRule 转换。
- LLM 默认模型选择。
- Chat 工具调用授权。
- 内存版管理员用户创建、权限更新、LLM Provider/Model 创建接口。
- 内存版操作员权限、模型列表、Chat 会话和异常 Pod 巡检响应。
- 内存版 Store，集中封装用户、权限、LLM Provider/Model、审计日志状态。
- 内存版审计接口 `GET /api/admin/audit-logs`，记录用户、权限、LLM 和 Chat 动作。
- PostgreSQL Store，实现 schema 初始化、Demo 数据初始化、用户/权限/LLM/审计持久化。
- Redis Client，实现 `PING`、`SET`、`GET`，用于后续缓存和流式状态。
- Kubernetes RBAC Manager，使用 `client-go` 创建/更新操作员 ServiceAccount、Role、RoleBinding，并使用 fake client 覆盖测试。
- HTTP 权限更新接口可注入 RBAC Manager，在 `K8S_RBAC_SYNC_ENABLED=true` 时将权限同步到 Kubernetes。

当前尚未实现：

- Keycloak JWT 真实校验。
- Keycloak Admin API 创建用户。
- PostgreSQL 生产级 migration 工具和版本管理。
- Redis 业务缓存键设计和流式状态接入。
- MCP Client 和真实 LLM Provider 调用。
- LLM API Key 的生产级加密和解密。

## Store 边界

当前 HTTP 层通过 `backend/internal/store.Store` 访问业务状态。默认使用 `MemoryStore`，设置 `STORE_DRIVER=postgres` 后使用 `PostgresStore`。

Store 当前负责：

- 用户列表和用户创建。
- 操作员权限列表和权限替换。
- LLM Provider/Model 创建和查询。
- 审计日志追加和查询。

Store 不负责：

- Keycloak 用户真实创建。
- Kubernetes RBAC 真实创建。
- LLM API Key 生产级加密。

## PostgreSQL 和 Redis 本地集成测试

在 Windows + WSL Docker 环境中启动依赖：

```bash
wsl bash /mnt/e/k8s-agent/scripts/dev-infra-wsl.sh
```

如果当前 Windows/WSL 环境会在命令结束后停止 WSL Docker 容器，可以使用：

```bash
wsl bash /mnt/e/k8s-agent/scripts/dev-infra-wsl.sh --hold-seconds 600
```

运行集成测试：

```powershell
cd E:\k8s-agent\backend
$env:K8S_AI_TEST_DATABASE_URL='postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable'
$env:K8S_AI_TEST_REDIS_ADDR='localhost:56379'
go test ./internal/store ./internal/cache -count=1 -v
```

运行 Backend 并接入真实 PostgreSQL/Redis：

```powershell
$env:STORE_DRIVER='postgres'
$env:CACHE_DRIVER='redis'
$env:DATABASE_URL='postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable'
$env:REDIS_ADDR='localhost:56379'
go run ./cmd/api
```

如需在可访问 Kubernetes 的环境中同步真实 RBAC：

```powershell
$env:K8S_RBAC_SYNC_ENABLED='true'
$env:KUBECONFIG='C:\Users\you\.kube\config'
go run ./cmd/api
```

后续开发建议顺序：

1. 接入真实 PostgreSQL migration 和 repository。
2. 接入 Keycloak JWT 校验和 Admin API。
3. 实现 MCP Client。
4. 实现 OpenAI/Anthropic adapter。
5. 将前端静态页面接入真实 API。

## MCP Server 扩展方式

新增 MCP 工具时需要同步完成：

1. 在 `mcp-server/internal/tools/` 增加工具参数和结果结构。
2. 在 MCP Server 路由中注册工具端点。
3. 在 Backend Chat 授权映射中登记工具对应的 `namespace/resource/verb`。
4. 更新 [Chat 与 MCP 流程](../architecture/chat-mcp-flow.md) 的工具表。
5. 更新 [API 设计](../reference/api-design.md) 或相关接口文档。
6. 增加单元测试，覆盖正常调用和越权拒绝。

## LLM Provider 扩展方式

新增 Provider 协议时需要：

1. 在 `backend/internal/llm` 增加协议类型。
2. 实现统一 Chat Request/Response 转换。
3. 实现工具调用解析。
4. 增加 Provider 配置字段校验。
5. 更新 LLM 管理 API 和文档。
6. 增加 mock 测试，避免依赖真实外部模型。

## 权限开发规则

- 任何 K8S 工具调用都必须先映射到 `namespace/resource/verb`。
- 任何工具调用都必须先经过业务权限校验。
- 任何实际 Kubernetes API 调用都必须使用操作员 ServiceAccount。
- 不允许因为管理员登录态而让操作员请求复用管理员 ServiceAccount。

## Kubernetes RBAC Manager

`backend/internal/k8s.RBACManager` 负责将业务权限落到 Kubernetes RBAC 对象中。

输入：

```go
UserNamespacePermissions{
    UserID: "u123",
    Namespace: "dev",
    Rules: []PermissionSpec{
        {APIGroup: "", Resource: "pods", Verbs: []string{"get", "list"}},
    },
}
```

输出到 Kubernetes：

- `ServiceAccount`: `k8s-ai-operator-u123`
- `Role`: `k8s-ai-role-u123-dev`
- `RoleBinding`: `k8s-ai-binding-u123-dev`

测试使用 `client-go` fake client，避免依赖真实集群。

HTTP 层通过 `SetRBACApplier` 注入 RBAC Manager，单元测试使用 fake applier 验证：

- 权限按 namespace 分组下发。
- RBAC 同步失败时返回 `K8S_RBAC_APPLY_FAILED`。

## 日志规则

程序日志使用英文结构化格式：

```text
level=INFO component=backend-api event=server_start addr=:8080
level=ERROR component=mcp-server event=server_exit error="listen tcp ..."
```

日志中禁止输出：

- LLM API Key。
- ServiceAccount token。
- Kubernetes Secret 明文。
- 用户密码。

## 文档同步规则

- 改 API，更新 `docs/reference/api-design.md`。
- 改架构，更新 `docs/architecture/system-architecture.md`。
- 改权限，更新 `docs/architecture/permission-model.md` 和 `docs/security/security-design.md`。
- 改部署脚本或 Helm values，更新 `docs/operations/deployment-guide.md`。
- 改用户流程，更新 `docs/product/user-journeys.md`。

## 验证命令

```bash
cd backend && go test ./...
cd mcp-server && go test ./...
cd frontend && npm run build
bash -n scripts/bootstrap-local.sh scripts/helm-install.sh scripts/helm-upgrade.sh scripts/uninstall.sh scripts/build-images.sh
```
