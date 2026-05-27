# 术语表

## Agent 相关术语

- Eino：CloudWeGo 的 Go LLM 应用框架，本项目用于 Agent Server 内部 ReAct agent loop 编排。
- Agent Server：独立的 Go 服务，使用 Eino ADK ChatModelAgent 执行 ReAct agent loop，通过 gRPC AgentService.RunStream 接收 Backend 的 Chat 请求，调用 LLM 和 MCP Server 完成 K8s 运维操作。支持 Skills 系统（渐进式披露的运维知识库）。
- AgentService.RunStream：Backend 与 Agent Server 之间的服务端流式 gRPC 方法，由 `proto/agent/v1/agent.proto` 定义。Backend 发送 AgentRunRequest，Agent Server 流式返回 StreamEvent。
- MCP 内置工具：Agent Server 从 MCP Server 发现并注册的 Kubernetes 工具集合，不由 Backend 逐请求下发。
- Runtime Context：Backend 传给 Agent Server 的最小必要上下文，包括当前用户、授权 namespace 和最近资源引用。
- Recent Resources：最近几轮 Chat 返回的轻量 Kubernetes 资源引用，用于多轮指代理解，不作为授权凭证。
- Skills：存放在 SKILLS_DIR 目录下的运维知识单元。每个 skill 是一个包含 SKILL.md 的子目录，通过渐进式披露机制按需加载到 Agent 上下文中。
- StreamEvent：Agent Server 通过 gRPC 流式返回的事件类型，包含 Thinking（思考过程）、ToolCall（工具调用）、ToolResult（工具结果）、Resource（资源引用）、Complete（完成）、Error（错误）六种类型。

| 术语 | 说明 |
| --- | --- |
| MCP | Model Context Protocol，用于让 AI 通过标准化工具接口访问外部能力。 |
| Backend API | 系统控制面，负责认证授权、用户管理、权限管理、LLM Provider/Model 管理、Chat 会话编排和审计日志。通过 gRPC AgentService.RunStream 将上下文传递给 Agent Server 执行 AI agent loop。不直接调用 LLM 或 MCP Server。 |
| MCP Server | 将 Kubernetes API 封装为标准 MCP 协议工具的独立服务。每个工具调用时动态创建用户级 K8s client，确保租户隔离。通过 gRPC IdentityService 获取用户 ServiceAccount 凭据。 |
| Keycloak | 身份认证服务，负责登录、用户身份和平台角色。 |
| ServiceAccount | Kubernetes 中用于工作负载访问 API Server 的身份。 |
| Role | Kubernetes namespace 级权限集合。 |
| RoleBinding | 将 Role 绑定到 ServiceAccount 或用户。 |
| Provider | LLM 服务提供方，例如 OpenAI 或 Anthropic。 |
| Model | Provider 下的具体模型，例如 `gpt-4.1` 或 `claude-3-5-sonnet`。 |
| Tool Calling | LLM 请求外部工具执行操作的机制。 |
| 审计日志 | 记录用户、工具调用、权限变更和 Kubernetes 操作的追踪日志。 |
