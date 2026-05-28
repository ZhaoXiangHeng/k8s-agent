# 面试交付索引 — K8S AI Ops

本文档对照面试任务（`golang挑战任务.pdf`）的交付要求，逐一索引本仓库中的交付物。

## 任务要求概览

> 开发一个云原生 AI 运维系统，AI 引擎通过 MCP 协议获取 K8s 集群状态，用户通过 Web 页面与 AI 助手交互，实现"用自然语言管理集群"。

- MVP 功能：集群巡检（Read 动作）—— 用户输入"帮我看看现在集群里有什么异常吗？"
- 技术栈：Docker + K8s + MCP + LLM Tool Calling + 全栈 Web
- 交付物：MCP Server 源码 / 全栈前后端源码 / K8s 部署文件 / 架构文档 / 安装文档 / AI 协同研发报告

---

## 一、交付物索引

### 1. 源码与配置

#### MCP Server 源码

> 任务要求：实现与 K8s API 交互的 MCP Tools（如 list_pods、get_pod_logs、delete_pod）

| 路径 | 说明 |
|------|------|
| [mcp-server/](mcp-server/) | MCP Server 完整源码，Go 语言，基于 `mark3labs/mcp-go` 标准 MCP 协议 |
| [mcp-server/internal/handler/](mcp-server/internal/handler/) | 8 个 MCP 工具处理器（详见下方工具表） |
| [mcp-server/internal/k8s/](mcp-server/internal/k8s/) | Per-user K8s client 工厂，通过 gRPC IdentityService 获取操作员 ServiceAccount |
| [mcp-server/internal/identity/](mcp-server/internal/identity/) | gRPC IdentityService 客户端，连接 Backend API 获取用户 K8s 凭据 |

**已实现的 MCP 工具（超出任务要求）**：

| 工具 | K8s 资源 | 用途 |
|------|----------|------|
| `list_namespaces` | 权限摘要 | 返回当前用户可见 namespace |
| `list_pods` | pods | 查询 Pod 列表和异常状态 |
| `get_pod` | pods | 查询 Pod 详情 |
| `get_pod_logs` | pods/log | 查询 Pod 日志 |
| `list_events` | events | 查询集群事件 |
| `get_pod_events` | events | 查询特定 Pod 的事件 |
| `list_deployments` | deployments.apps | 查询 Deployment |
| `restart_deployment` | deployments.apps | 通过 patch annotation 触发滚动重启 |

#### 全栈前后端源码

> 任务要求：前端对话界面及后端中转服务

| 路径 | 说明 |
|------|------|
| [frontend/](frontend/) | React + TypeScript 前端，Vite 构建 |
| [frontend/src/pages/](frontend/src/pages/) | 操作员 Chat 界面、管理员控制台（用户管理/权限管理/LLM 配置/模型分配） |
| [frontend/src/infrastructure/api/](frontend/src/infrastructure/api/) | Backend API 调用封装 |
| [backend/](backend/) | Go Backend API，DDD 四层架构 |
| [backend/internal/interfaces/http/](backend/internal/interfaces/http/) | HTTP 路由、中间件、SSE 流式事件中继 |
| [backend/internal/infra/agent/](backend/internal/infra/agent/) | gRPC AgentService 客户端（连接 Agent Server） |
| [agent-server/](agent-server/) | Go Agent Server，基于 Eino ADK ChatModelAgent 的 ReAct agent loop |
| [agent-server/internal/eino/](agent-server/internal/eino/) | Eino agent 配置、ReAct runner、LLM 工厂、MCP 客户端 |
| [proto/](proto/) | gRPC 契约定义（agent/v1 + identity/v1），服务间通信的唯一来源 |

#### K8s 部署文件

> 任务要求：一套可以 kubectl apply -f 一键部署到集群中的 YAML（含 RBAC），amd64 架构离线镜像包

| 路径 | 说明 |
|------|------|
| [deploy/helm/k8s-ai-ops/](deploy/helm/k8s-ai-ops/) | Helm Chart，15 个模板文件 |
| [deploy/helm/k8s-ai-ops/templates/rbac.yaml](deploy/helm/k8s-ai-ops/templates/rbac.yaml) | RBAC 配置：ServiceAccount、Role、RoleBinding、ClusterRoleBinding |
| [deploy/helm/k8s-ai-ops/templates/traefik.yaml](deploy/helm/k8s-ai-ops/templates/traefik.yaml) | Traefik API 网关（hostPort:80） |
| [deploy/helm/k8s-ai-ops/templates/ingressroute.yaml](deploy/helm/k8s-ai-ops/templates/ingressroute.yaml) | IngressRoute 路由规则 |
| [deploy/helm/k8s-ai-ops/templates/keycloak-realm-configmap.yaml](deploy/helm/k8s-ai-ops/templates/keycloak-realm-configmap.yaml) | Keycloak Realm 预配置（admin/operator 角色，PKCE OIDC client） |
| [deploy/helm/k8s-ai-ops/values.yaml](deploy/helm/k8s-ai-ops/values.yaml) | 默认 Helm values |
| [deploy/helm/k8s-ai-ops/values-local.yaml](deploy/helm/k8s-ai-ops/values-local.yaml) | 本地 Kind 集群 values |
| [scripts/build-images.sh](scripts/build-images.sh) | 构建 4 个服务 amd64 镜像 + 导出 tar 包 |
| [scripts/bootstrap-local.sh](scripts/bootstrap-local.sh) | Kind 集群创建 + 镜像加载 + Helm 安装（一键部署） |
| [scripts/helm-install.sh](scripts/helm-install.sh) | Helm-only 安装脚本 |
| [scripts/uninstall.sh](scripts/uninstall.sh) | 卸载脚本 |

镜像交付物示例（运行 `scripts/build-images.sh --tag local --output-dir image-tars` 生成）：

```text
image-tars/backend-api-amd64.tar
image-tars/agent-server-amd64.tar
image-tars/mcp-server-amd64.tar
image-tars/frontend-amd64.tar
```

---

### 2. 架构说明文档 (ARCHITECTURE.md)

> 任务要求：架构拓扑图（UI、Backend、MCP Server、LLM、K8s API 调用链路），安全设计（ServiceAccount 的设计）

| 文档 | 覆盖内容 |
|------|----------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | 核心架构图（6 个子图、14 条通信连线）、核心原则、文档入口 |
| [docs/architecture/system-architecture.md](docs/architecture/system-architecture.md) | 完整部署拓扑、组件清单、服务间通信矩阵、Traefik 路由规则、RBAC 设计、Keycloak Realm 配置、健康检查 |
| [docs/architecture/permission-model.md](docs/architecture/permission-model.md) | 三层权限防线：prompt 限制 + MCP Server 校验 + K8s RBAC 兜底；ServiceAccount 命名规则和托管标签；工具调用授权流程 |
| [docs/architecture/chat-mcp-flow.md](docs/architecture/chat-mcp-flow.md) | Chat→Backend→Agent→MCP→K8s 完整序列图，MCP→Backend IdentityService 身份/权限交互流程 |
| [docs/developer/architecture-overview.md](docs/developer/architecture-overview.md) | 模块划分、DDD 分层、代码入口文件索引 |
| [docs/security/security-design.md](docs/security/security-design.md) | 认证/授权/LLM 安全/审计/威胁与控制表 |

**安全设计要点（直接回应任务要求）**：

- MCP Server 每次工具调用前通过 Backend IdentityService gRPC 获取操作员 K8s ServiceAccount token
- 每个操作员拥有独立的 K8s ServiceAccount（`k8s-ai-operator-{userId}`）
- MCP Server 工具执行前校验 namespace/resource/verb 是否在用户业务权限内
- Kubernetes RBAC 作为最终权限边界，即使上层校验有缺陷，K8s API Server 也会拒绝越权访问
- LLM 不直接访问 K8s API，LLM API Key 不写日志，ServiceAccount token 加密存储

---

### 3. 安装部署文档 (README.md)

> 任务要求：如何在本地集群一键安装，如何配置 LLM 的 API Key

| 文档 | 覆盖内容 |
|------|----------|
| [README.md](README.md) | 项目入口、一键部署命令、构建命令、本地验证命令 |
| [docs/guides/quickstart.md](docs/guides/quickstart.md) | 5 分钟快速上手 |
| [docs/operations/deployment-guide.md](docs/operations/deployment-guide.md) | 完整部署指南（Kind/Helm/公有云三种模式）、values 配置项、卸载 |
| [docs/reference/config-reference.md](docs/reference/config-reference.md) | Helm values 完整参考、环境变量、端口约定、Traefik 路由规则 |

**一键部署命令**：

```bash
scripts/bootstrap-local.sh --image-source tar --image-dir image-tars --cluster-name k8s-ai
```

**LLM API Key 配置方式**：通过 Helm values 或 Backend 管理 API（`POST /api/admin/llm/providers`），API Key 使用 AES-256-GCM 加密存储。

---

### 4. AI 协同研发报告 (AI_PROMPTS.md) — 重点考察点

> 任务要求：记录如何使用 AI 辅助编程和调试，包含：你使用了什么 Prompt？AI 帮你写了哪部分代码？遇到了什么 Bug，AI 是如何帮你定位和修复的？

| 文档 | 覆盖内容 |
|------|----------|
| [AI_PROMPTS.md](AI_PROMPTS.md) | 完整 AI 协同研发记录（~300 行），按 7 个开发阶段组织 |
| [问题记录.md](问题记录.md) | 24 个问题的详细记录（问题描述 → 根因 → 修复 → 验证） |

**AI_PROMPTS.md 中包含的内容**：

1. **实际提示词**：初始需求 4 条、架构打磨 3 条、远程运维 7 条、部署触发 1 条、文档迭代 4 条（全部附原文）
2. **AI 写的代码**：Backend API、Agent Server、MCP Server、Frontend 全部四服务源码均由 AI 生成，人工做决策确认和安全把关
3. **Bug 与 AI 修复**：24 个线上 bug 的分层定位和修复过程（部署配置缺失/网关路由冲突/环境适配/启动依赖/AI 协作流程）
4. **Git 执行记录**：96 分钟 20 个 commit 的 session 日志
5. **方法论**：Superpowers 标准化流程（Brainstorming → Spec → Plan → Execute → Verify → Deploy）

---

## 二、任务要求 vs 实际交付对照

| 任务要求 | 完成情况 | 超出部分 |
|----------|----------|----------|
| MCP Server (list_pods, get_pod_logs, delete_pod) | 8 个 MCP 工具，基于 mark3labs/mcp-go | +5 个工具，per-user K8s client 隔离 |
| 前端对话界面 + 后端中转 | React Chat UI + Go Backend + gRPC agent 服务 | 管理员控制台（用户/权限/LLM/模型/审计管理） |
| K8s 部署文件 + RBAC + amd64 离线镜像 | Helm Chart (15 模板) + build-images.sh 生成 amd64.tar | Traefik 网关 + Keycloak 认证 + initContainers 启动依赖 |
| 架构拓扑图 | 6-subgraph mermaid 架构图 + 14 条通信连线 | 服务间通信矩阵 + 公有云拓扑 + 健康检查 |
| 安全设计 (ServiceAccount) | namespace 级 SA + 三层权限防线 + 托管标签 | AES-256-GCM 加密存储 + 审计日志脱敏 + managedNamespaces 白名单 |
| 一键安装 | bootstrap-local.sh (Kind + Helm) | tar 和 registry 双模式 + 公有云部署指南 |
| LLM API Key 配置 | 管理 API 配置 + Helm values | 加密存储 + 查询接口不返回明文 |
| AI 协同研发报告 | 7 阶段 + 19 条提示词原文 + 24 个 bug 记录 | git session 实录 + Superpowers 方法论 + 角色分工表 |

## 三、阅读路径建议（给面试官）

如果是第一次看这个仓库，建议按以下顺序阅读：

1. **[README.md](README.md)** — 了解项目是什么
2. **[ARCHITECTURE.md](ARCHITECTURE.md)** — 看懂架构拓扑
3. **[docs/architecture/system-architecture.md](docs/architecture/system-architecture.md)** — 深度理解部署架构
4. **[docs/architecture/chat-mcp-flow.md](docs/architecture/chat-mcp-flow.md)** — 理解从自然语言到 K8s 的完整链路
5. **[docs/security/security-design.md](docs/security/security-design.md)** — 理解安全设计（重点：ServiceAccount 隔离）
6. **[AI_PROMPTS.md](AI_PROMPTS.md)** — 了解 AI 协同研发全流程
7. **[问题记录.md](问题记录.md)** — 了解工程化问题和解决过程
