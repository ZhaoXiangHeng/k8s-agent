# 术语表

## 核心概念

| 术语 | 说明 |
|------|------|
| **MCP** | Model Context Protocol，LLM 通过标准化工具接口访问外部能力的协议 |
| **Eino** | CloudWeGo 的 Go LLM 应用框架，本项目用于 Agent Server 内部 ReAct agent loop 编排 |
| **ReAct** | Reasoning + Acting，LLM 交替进行推理和行动的 agent 模式 |
| **gRPC** | Google Remote Procedure Call，Backend 与 Agent Server、MCP Server 与 Backend 之间的通信协议 |
| **SSE** | Server-Sent Events，MCP Server 传输和 Frontend 流式渲染使用的协议 |
| **Traefik** | 内置 API 网关，通过 hostPort 暴露 HTTP 入口，按路径路由到前端、API 和 Keycloak |

## 服务

| 术语 | 说明 |
|------|------|
| **Backend API** | 系统控制面，负责认证授权、用户管理、权限管理、LLM Provider/Model 管理、Chat 会话编排和审计日志。通过 gRPC AgentService.RunStream 将上下文传递给 Agent Server。不直接调用 LLM 或 MCP Server |
| **Agent Server** | 独立的 Go 服务，使用 Eino ADK ChatModelAgent 执行 ReAct agent loop，通过 gRPC AgentService.RunStream 接收 Backend 的 Chat 请求，调用 LLM 和 MCP Server 完成 K8s 运维操作。支持 Skills 系统 |
| **MCP Server** | 将 Kubernetes API 封装为标准 MCP 协议工具的独立服务。每次工具调用时动态创建 per-user K8s client（通过 gRPC IdentityService 获取 ServiceAccount 凭据），确保租户隔离。SSE transport |
| **Frontend** | React Web UI，包含管理员控制台和操作员 Chat 页面 |

## Agent 相关

| 术语 | 说明 |
|------|------|
| **AgentService.RunStream** | Backend 与 Agent Server 之间的服务端流式 gRPC 方法，由 `proto/agent/v1/agent.proto` 定义 |
| **StreamEvent** | Agent Server 通过 gRPC 流式返回的事件，包含 Thinking、ToolCall、ToolResult、Resource、Complete、Error 六种类型 |
| **MCP 内置工具** | Agent Server 从 MCP Server 发现并注册的 K8s 工具集合，不由 Backend 逐请求下发 |
| **Runtime Context** | Backend 传给 Agent Server 的最小必要上下文：当前用户、授权 namespace、最近资源引用 |
| **Skills** | 存放在 `SKILLS_DIR` 目录下的运维知识单元，通过渐进式披露按需加载到 Agent 上下文 |
| **IdentityService.GetServiceAccount** | MCP Server 调用 Backend 获取用户 ServiceAccount 凭据的 unary gRPC 方法，由 `proto/identity/v1/identity.proto` 定义 |

## Kubernetes

| 术语 | 说明 |
|------|------|
| **ServiceAccount** | Kubernetes 中为工作负载提供身份的对象。系统为每个操作员创建 `k8s-ai-operator-{userId}` |
| **Role** | Kubernetes namespace 级权限集合。系统创建 `k8s-ai-role-{userId}-{namespace}` |
| **RoleBinding** | 将 Role 绑定到 ServiceAccount。系统创建 `k8s-ai-binding-{userId}-{namespace}` |
| **托管标签** | `app.kubernetes.io/managed-by=k8s-ai-ops-backend`，标识由 Backend RBAC Manager 管理的对象 |
| **rbac.managedNamespaces** | Helm values 中声明的 namespace 白名单，限制 Backend 的 RBAC 操作范围 |

## LLM

| 术语 | 说明 |
|------|------|
| **Provider** | LLM 服务提供方，例如 OpenAI 或 Anthropic |
| **Model** | Provider 下的具体模型，例如 `gpt-4.1` 或 `claude-3-5-sonnet` |
| **Tool Calling** | LLM 请求外部工具执行操作的机制 |
| **API Key** | LLM 认证凭据，使用 AES-256-GCM 加密存储，不在日志和审计中暴露 |

## 安全与审计

| 术语 | 说明 |
|------|------|
| **JWT** | JSON Web Token，Keycloak 签发的身份凭据 |
| **Keycloak** | 身份认证服务，负责登录、用户身份和平台角色（admin / operator） |
| **三层权限防线** | prompt 限制 + MCP Server 校验 + Kubernetes RBAC 兜底 |
| **审计日志** | 记录用户操作、工具调用、权限变更和 K8s 操作的追踪日志，请求与响应脱敏 |
