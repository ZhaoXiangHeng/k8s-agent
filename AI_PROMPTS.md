# AI 协同研发报告

本文件记录 K8S AI Ops 在设计、实现、调试过程中如何使用 AI 协助研发。

## 规划阶段 Prompt

初始需求：

```text
按照 docs/golang挑战任务.pdf 这个文档中的面试任务要求，帮我开发这个 k8s 运行 AI。
先梳理需求，生成需求文档；完成技术选型、架构设计、服务设计、实现设计；文档完善后进行代码开发。
```

追加的产品需求：

```text
产品需要多角色用户体系，包含管理员和操作员。
管理员创建管理员和操作员，分配操作员可以操作的 Kubernetes namespace/resource/verb 权限。
系统动态生成 ServiceAccount、Role、RoleBinding 并绑定到操作员 ID。
管理员管理 OpenAI 和 Anthropic 协议的 LLM provider/model，并给操作员绑定可用模型。
操作员通过 chat 自然语言管理自己有权限的 Kubernetes 资源。
```

部署需求：

```text
安装部署脚本需要支持一键同时安装本地 Kubernetes 和 Helm 部署，也支持只部署和更新 Helm。
部署镜像支持本地 tar 包或者镜像仓库地址，默认使用本地 tar 包。
```

文档和日志规范需求：

```text
生成文件写入项目规则。
项目文档路径需要明确。
更新代码或文档时需要保持两者一致。
所有文档需要中文，已经是英文的需要改成中文。
所有代码注释是中文。
所有程序日志使用英文，并尽可能清晰、利于排错、具备分级。
```

## AI 参与内容

- 从 PDF 面试题中提取 MVP 需求。
- 将系统扩展为基于 Keycloak 的多角色平台。
- 提出 Go Backend API、Go MCP Server、React UI 的架构方案。
- 设计 namespace 级动态 Kubernetes RBAC 和 ServiceAccount 隔离方案。
- 设计 OpenAI 与 Anthropic Provider/Model 管理模型。
- 设计 Helm 部署、本地 Kind 启动、tar 镜像包和镜像仓库两种镜像来源。
- 实现第一版 Backend、MCP Server、Frontend、Helm Chart、Dockerfile 和部署脚本骨架。
- 补充项目协作规则，并将文档统一改为中文。
- 根据企业级分层文档继续完善 Backend API，补充内存版管理员用户管理、权限更新、LLM Provider/Model 管理、操作员 Chat 巡检接口和统一错误响应。
- 引入 Agent Server 服务拆分方案：Backend 通过 proto 生成的 gRPC `AgentService.Run` 调用 Agent Server，Agent Server 使用 Eino 执行无状态 agent loop，MCP Server 提供 Kubernetes 工具边界。

## 调试记录

### Frontend TypeScript 构建

问题：

```text
tsc 只打印帮助信息，没有编译项目。
```

根因：

```text
frontend 项目缺少 tsconfig.json。
```

修复：

```text
新增 Vite 兼容的 tsconfig.json。
```

后续问题：

```text
TypeScript 找不到 React 类型声明。
```

根因：

```text
项目缺少 @types/react 和 @types/react-dom。
```

修复：

```text
安装 @types/react 和 @types/react-dom 到 devDependencies。
```

验证：

```text
npm run build 通过。
```

### Helm 验证

问题：

```text
本机未安装 Helm 或 Helm 不在 PATH 中，无法执行 helm template。
```

后续验证命令：

```bash
helm template k8s-ai-ops deploy/helm/k8s-ai-ops
```

### Backend API 骨架完善

实现内容：

```text
新增 HTTP 层测试，覆盖 /api/me、操作员权限、操作员模型、Chat 会话、Chat 消息、管理员创建用户、更新权限、创建 LLM Provider 和创建 LLM Model。
```

当前边界：

```text
这些接口使用内存数据实现，用于先稳定 API 契约和前后端联调。Keycloak、PostgreSQL、Redis、Kubernetes client-go、真实 LLM 调用和生产级 API Key 加密仍是后续任务。
```

验证：

```text
backend: go test ./... 通过。
```

### Store 边界重构和审计接口

实现内容：

```text
将 HTTP 层中的内存状态下沉到 backend/internal/store.MemoryStore，新增用户、权限、LLM Provider/Model、审计日志的内存读写方法。
新增 GET /api/admin/audit-logs，并在创建用户、更新权限、创建 LLM Provider、创建 LLM Model、Chat 消息时写入审计日志。
```

验证：

```text
backend/internal/store: go test ./internal/store 通过。
backend/internal/http: go test ./internal/http 通过。
```

### PostgreSQL 和 Redis 真实接口

实现内容：

```text
新增 Store 接口，HTTP 层依赖接口而不是具体 MemoryStore。
新增 PostgresStore，使用 pgx/database/sql 连接 PostgreSQL，支持 schema 初始化、Demo 数据初始化、用户、权限、LLM Provider/Model、审计日志持久化。
新增 RedisClient，使用 RESP 协议实现 PING、SET、GET。
新增 WSL Docker 本地依赖脚本 scripts/dev-infra-wsl.sh。
```

验证：

```text
使用 WSL Docker 启动 k8s-ai-pg 和 k8s-ai-redis。
K8S_AI_TEST_DATABASE_URL=postgres://k8s_ai:k8s_ai@localhost:55432/k8s_ai?sslmode=disable
K8S_AI_TEST_REDIS_ADDR=localhost:56379
go test ./internal/store ./internal/cache -count=1 -v 通过。
```

### Kubernetes RBAC Manager

实现内容：

```text
新增 backend/internal/k8s.RBACManager，使用 client-go 创建或更新操作员 ServiceAccount、Role、RoleBinding。
使用 client-go fake client 编写单元测试，验证首次创建和重复更新 Role 的行为。
```

验证：

```text
go test ./internal/k8s -v 通过。
```

### Eino Agent Server 与 gRPC 契约

实现内容：

```text
新增 proto/agent/v1/agent.proto，并提交 protoc 生成的 Go/gRPC 代码。
新增 agent-server 服务，使用 gRPC 暴露 AgentService.Run。
Agent Server 内部引入 CloudWeGo Eino，将 Backend 传入的 messages 和 runtimeContext 转换为 Eino message 上下文。
Backend 新增 gRPC Agent client，并在 Chat 消息处理中调用 Agent Server。
Backend 继续负责多轮历史、最近资源引用、权限 allowlist 和审计，Agent Server 不持久化历史。
```

验证：

```text
proto: go test ./... 通过。
agent-server: go test ./... 通过。
backend/internal/agent backend/internal/config backend/internal/http: go test 通过。
mcp-server: go test ./... 通过。
Helm: 当前 PowerShell 环境未找到 helm，需要在安装 Helm 后执行 helm template k8s-ai deploy/helm/k8s-ai-ops。
```
