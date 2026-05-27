# 术语表

## Agent 相关术语

- Eino：CloudWeGo 的 Go LLM 应用框架，本项目用于 Agent Server 内部 agent loop 编排。
- Agent Server：独立 Go 服务，通过 gRPC 接收 Backend 的 Chat Run 请求，使用 Eino 调用 LLM 和 MCP 工具。
- AgentService.Run：Backend 与 Agent Server 之间的 gRPC 方法，由 `proto/agent/v1/agent.proto` 定义。
- Tool Allowlist：Backend 根据当前用户权限生成的本轮允许工具列表。
- Runtime Context：Backend 传给 Agent Server 的最小必要上下文，包括当前用户、授权 namespace 和最近资源引用。
- Recent Resources：最近几轮 Chat 返回的轻量 Kubernetes 资源引用，用于多轮指代理解，不作为授权凭证。

| 术语 | 说明 |
| --- | --- |
| MCP | Model Context Protocol，用于让 AI 通过标准化工具接口访问外部能力。 |
| Backend API | 系统控制面，负责认证、授权、Chat 编排、LLM 调用、MCP 调用和审计。 |
| MCP Server | 暴露 Kubernetes 工具的服务。 |
| Keycloak | 身份认证服务，负责登录、用户身份和平台角色。 |
| ServiceAccount | Kubernetes 中用于工作负载访问 API Server 的身份。 |
| Role | Kubernetes namespace 级权限集合。 |
| RoleBinding | 将 Role 绑定到 ServiceAccount 或用户。 |
| Provider | LLM 服务提供方，例如 OpenAI 或 Anthropic。 |
| Model | Provider 下的具体模型，例如 `gpt-4.1` 或 `claude-3-5-sonnet`。 |
| Tool Calling | LLM 请求外部工具执行操作的机制。 |
| 审计日志 | 记录用户、工具调用、权限变更和 Kubernetes 操作的追踪日志。 |
