# K8S AI Ops 实现计划

> **给自动化开发代理的要求：** 如果继续执行本计划，应按任务逐项推进，并在每个任务完成后运行对应验证命令。任务使用 checkbox 语法记录进度。

**目标：** 构建 Go + React + Helm 的 Kubernetes AI 运维 MVP，包含 Keycloak 认证、动态 namespace 级 RBAC、LLM Provider 管理、MCP Kubernetes 工具和基于 Chat 的异常 Pod 巡检。

**架构：** Go Backend API 作为控制面和 Chat 编排服务。独立 Go MCP Server 暴露 Kubernetes 工具，并使用操作员 ServiceAccount 调用 Kubernetes API。React UI 提供管理员和操作员控制台，Helm 默认使用本地 tar 镜像包部署整套系统。

**技术栈：** Go、标准库 HTTP、client-go、PostgreSQL、Redis、Keycloak、React、Vite、TypeScript、Helm、Kind。

---

## 文件结构

```text
backend/
  go.mod
  cmd/api/main.go
  internal/config/config.go
  internal/http/router.go
  internal/store/models.go
  internal/k8s/rbac.go
  internal/llm/provider.go
  internal/chat/service.go

mcp-server/
  go.mod
  cmd/server/main.go
  internal/tools/pods.go
  internal/k8s/client.go

frontend/
  package.json
  index.html
  src/App.tsx
  src/main.tsx
  src/styles.css

deploy/helm/k8s-ai-ops/
  Chart.yaml
  values.yaml
  templates/*.yaml

scripts/
  bootstrap-local.sh
  helm-install.sh
  helm-upgrade.sh
  uninstall.sh
  build-images.sh
```

## 任务

### 任务 1：Backend API 骨架

**文件：**

- 创建：`backend/go.mod`
- 创建：`backend/cmd/api/main.go`
- 创建：`backend/internal/config/config.go`
- 创建：`backend/internal/http/router.go`

- [x] 创建 Go module：`k8s-ai-ops/backend`。
- [x] 从环境变量加载配置。
- [x] 实现 `/healthz`、`/api/me` 和操作员占位接口。
- [x] 在 `backend` 目录运行 `go test ./...`。

### 任务 2：Backend 领域模型

**文件：**

- 创建：`backend/internal/store/models.go`
- 创建：`backend/internal/store/models_test.go`

- [x] 增加用户、权限、ServiceAccount、LLM Provider、LLM Model、Chat、审计模型。
- [x] 增加 namespace、apiGroup、resource、verb 权限匹配测试。
- [x] 在 `backend` 目录运行 `go test ./...`。

### 任务 3：Kubernetes RBAC 服务

**文件：**

- 创建：`backend/internal/k8s/rbac.go`
- 创建：`backend/internal/k8s/rbac_test.go`

- [x] 实现 ServiceAccount、Role、RoleBinding 确定性命名。
- [x] 实现权限记录到 RoleRule 的转换。
- [x] 测试 namespace 级 RBAC 对象生成逻辑。
- [x] 在 `backend` 目录运行 `go test ./...`。

### 任务 4：LLM Provider 抽象

**文件：**

- 创建：`backend/internal/llm/provider.go`
- 创建：`backend/internal/llm/provider_test.go`

- [x] 定义 OpenAI、Anthropic、Mock Provider 协议类型。
- [x] 定义通用 Chat 请求和工具调用响应结构。
- [x] 实现默认模型选择逻辑。
- [x] 测试模型绑定和默认模型选择。
- [x] 在 `backend` 目录运行 `go test ./...`。

### 任务 5：Chat 编排

**文件：**

- 创建：`backend/internal/chat/service.go`
- 创建：`backend/internal/chat/service_test.go`

- [x] 构造包含权限摘要的系统提示词。
- [x] 在 MCP 执行前校验工具请求。
- [x] 定义异常 Pod 巡检响应结构。
- [x] 测试未授权 namespace/resource/verb 会被拒绝。
- [x] 在 `backend` 目录运行 `go test ./...`。

### 任务 6：MCP Server 骨架

**文件：**

- 创建：`mcp-server/go.mod`
- 创建：`mcp-server/cmd/server/main.go`
- 创建：`mcp-server/internal/tools/pods.go`
- 创建：`mcp-server/internal/k8s/client.go`

- [x] 实现 `/healthz`。
- [x] 实现 MVP 阶段 JSON-over-HTTP 风格工具端点。
- [x] 增加异常 Pod phase/reason 判断。
- [x] 在 `mcp-server` 目录运行 `go test ./...`。

### 任务 7：Frontend 骨架

**文件：**

- 创建：`frontend/package.json`
- 创建：`frontend/index.html`
- 创建：`frontend/src/main.tsx`
- 创建：`frontend/src/App.tsx`
- 创建：`frontend/src/styles.css`

- [x] 实现 Admin 和 Operator 导航。
- [x] 实现用户、权限、LLM 模型的静态 Admin 面板。
- [x] 实现 Operator Chat 面板和异常 Pod 表格。
- [x] 在 `frontend` 目录运行 `npm install` 和 `npm run build`。

### 任务 8：Helm Chart

**文件：**

- 创建：`deploy/helm/k8s-ai-ops/Chart.yaml`
- 创建：`deploy/helm/k8s-ai-ops/values.yaml`
- 创建：`deploy/helm/k8s-ai-ops/templates/*.yaml`

- [x] 增加 frontend、backend、mcp-server、Keycloak、PostgreSQL、Redis 的 Deployment 和 Service。
- [x] 增加 Backend 管理 namespace 级 RBAC 的权限。
- [x] 支持 tar 和 registry 镜像来源。
- [ ] 运行 `helm template k8s-ai-ops deploy/helm/k8s-ai-ops`，当前本机缺少 Helm，待安装后验证。

### 任务 9：部署脚本

**文件：**

- 创建：`scripts/bootstrap-local.sh`
- 创建：`scripts/helm-install.sh`
- 创建：`scripts/helm-upgrade.sh`
- 创建：`scripts/uninstall.sh`
- 创建：`scripts/build-images.sh`

- [x] 实现完整 Kind 启动脚本。
- [x] 实现 Helm-only 安装脚本。
- [x] 实现 Helm-only 升级脚本。
- [x] 实现默认不删除 PVC 的安全卸载脚本。
- [x] 运行 shell 语法检查。

### 任务 10：最终验证

**文件：**

- 修改：`README.md`
- 修改：`AI_PROMPTS.md`

- [x] 运行 Backend 测试。
- [x] 运行 MCP Server 测试。
- [x] 运行 Frontend 构建。
- [x] 更新 README 中的实际命令。
- [x] 更新 AI_PROMPTS 中的实现和调试记录。
- [x] 新增 `AGENTS.md`，记录文档、注释、日志和一致性规则。
