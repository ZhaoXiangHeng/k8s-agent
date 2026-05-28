# 快速开始

这篇文档面向第一次接触 K8S AI Ops 的使用者，目标是在 5 分钟内了解系统是什么、能做什么，以及最小的本地体验路径。

## 1. 系统是什么

K8S AI Ops 是一个基于 MCP 的 Kubernetes AI 运维助手。管理员通过 Web 控制台管理用户、Kubernetes namespace 级权限和 LLM 模型，操作员通过自然语言 Chat 巡检和操作授权范围内的 Kubernetes 资源。

核心组件：

| 组件 | 用途 |
|------|------|
| **Frontend** | React Web UI，管理员和操作员两个控制台 |
| **Backend API** | 认证授权、用户管理、权限管理、Chat 编排、审计 |
| **Agent Server** | 基于 Eino 的 ReAct agent loop，gRPC server-streaming |
| **MCP Server** | 将 Kubernetes API 封装为标准 MCP 工具，per-user K8s client 隔离 |
| **Keycloak** | 统一身份认证和平台角色 |
| **PostgreSQL / Redis** | 业务状态持久化和缓存 |

调用链路：

```mermaid
flowchart LR
    UI["React Web UI"] --> API["Backend API"]
    API --> KC["Keycloak"]
    API --> PG["PostgreSQL"]
    API --> Agent["Agent Server"]
    Agent --> LLM["LLM Provider"]
    Agent --> MCP["MCP Server"]
    MCP --> K8S["Kubernetes API"]
```

## 2. 第一次本地体验

### 2.1 前置条件

- Docker 可用（用于 Kind 集群和 infra 容器）
- 或已有可访问的 Kubernetes 集群（Kind / 云上均可）
- kubectl、Helm 已安装

### 2.2 启动本地 Kind 集群并部署

```bash
scripts/bootstrap-local.sh \
  --image-source tar \
  --image-dir image-tars \
  --cluster-name k8s-ai
```

这个脚本会完成：创建 Kind 集群 → 创建 dev/test 演示 namespace → 加载镜像 → Helm 安装系统。

### 2.3 仅部署 Helm（已有集群）

```bash
scripts/helm-install.sh \
  --image-source tar \
  --image-dir image-tars \
  --values deploy/helm/k8s-ai-ops/values-local.yaml
```

### 2.4 访问系统

```bash
kubectl port-forward -n k8s-ai-system svc/frontend 8088:80
kubectl port-forward -n k8s-ai-system svc/keycloak 8089:8080
```

浏览器打开：

- Frontend: http://localhost:8088
- Keycloak: http://localhost:8089

### 2.5 构建新镜像

修改代码后需要重新构建镜像：

```bash
scripts/build-images.sh --tag local --output-dir image-tars
```

生成：

```text
image-tars/backend-api-amd64.tar
image-tars/agent-server-amd64.tar
image-tars/mcp-server-amd64.tar
image-tars/frontend-amd64.tar
```

## 3. 开发环境（Windows + WSL Docker）

如果需要在 Windows 本地开发，可以启动 WSL Docker 中的 PostgreSQL 和 Redis：

```bash
wsl bash /mnt/e/k8s-agent/scripts/dev-infra-wsl.sh
```

然后启动 Backend（接入真实 DB/Redis）：

```powershell
$env:STORE_DRIVER='postgres'
$env:CACHE_DRIVER='redis'
$env:DATABASE_URL='postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable'
$env:REDIS_ADDR='localhost:56379'
go run ./backend/cmd/api
```

运行测试：

```powershell
cd backend
$env:K8S_AI_TEST_DATABASE_URL='postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable'
$env:K8S_AI_TEST_REDIS_ADDR='localhost:56379'
go test ./internal/store ./internal/cache -count=1 -v
```

## 4. 需要 Keycloak 真实 JWT 和 RBAC 同步

```powershell
$env:K8S_RBAC_SYNC_ENABLED='true'
$env:KUBECONFIG='C:\Users\you\.kube\config'
go run ./backend/cmd/api
```

## 5. 下一步

- 管理员想要配置操作员：[配置操作员完整流程](configuring-operators.md)
- 想要理解系统设计：[产品概览](../product/overview.md)
- 想要部署到公有云：[部署指南](../operations/deployment-guide.md)
- 想要修改代码：[开发者文档中心](../developer/README.md)
