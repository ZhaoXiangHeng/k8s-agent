# 用户流程

## 管理员创建操作员

```mermaid
sequenceDiagram
  actor Admin as 管理员
  participant UI as Admin Console
  participant API as Backend API
  participant KC as Keycloak
  participant DB as PostgreSQL
  participant K8S as Kubernetes API

  Admin->>UI: 填写操作员信息
  UI->>API: POST /api/admin/users
  API->>KC: 创建 Keycloak 用户并赋予 operator 角色
  API->>DB: 创建本地 users 记录
  API-->>UI: 返回操作员信息

  Admin->>UI: 分配 namespace/resource/verb 权限
  UI->>API: PUT /api/admin/users/:id/permissions
  API->>DB: 保存 k8s_permissions
  API->>K8S: 创建 ServiceAccount
  API->>K8S: 创建 Role
  API->>K8S: 创建 RoleBinding
  API->>DB: 保存 ServiceAccount 绑定
  API-->>UI: 返回权限分配结果
```

## 管理员配置 LLM

```mermaid
flowchart TD
  A["创建 Provider"] --> B["填写协议 openai/anthropic"]
  B --> C["配置 base_url 和 api_key"]
  C --> D["api_key 加密入库"]
  D --> E["创建 Provider 下的 Model"]
  E --> F["标记 supports_tools / supports_streaming"]
  F --> G["绑定模型给操作员"]
  G --> H["设置默认模型"]
```

## 操作员 Chat 巡检

```mermaid
sequenceDiagram
  actor Operator as 操作员
  participant UI as Frontend
  participant API as Backend API
  participant Agent as Agent Server
  participant MCP as MCP Server

  Operator->>UI: 输入多轮 Chat 问题
  UI->>API: POST /api/operator/chat/sessions/:id/messages
  API->>API: 读取历史、权限和最近资源引用
  API->>Agent: gRPC AgentService.Run(messages, runtimeContext, tools)
  Agent->>MCP: 调用授权工具
  Agent-->>API: 返回总结、资源结果和工具事件
  API->>API: 保存消息和审计
  API-->>UI: 返回总结和结构化资源
```

当操作员追问“看看这个 Pod 的日志”时，Backend 从最近资源引用中填充 `runtimeContext.recentResources`，Agent Server 用它理解指代，但实际 `get_pod_logs` 仍必须通过工具 allowlist 和 MCP 权限校验。

```mermaid
sequenceDiagram
  actor Operator as 操作员
  participant UI as Operator Console
  participant API as Backend API
  participant LLM as LLM Provider
  participant MCP as MCP Server
  participant K8S as Kubernetes API
  participant DB as PostgreSQL

  Operator->>UI: 输入自然语言巡检请求
  UI->>API: POST /api/operator/chat/sessions/:id/messages
  API->>DB: 读取用户权限和可用模型
  API->>LLM: 发送受权限约束的 prompt
  LLM-->>API: 请求 list_pods 工具
  API->>API: 校验 namespace/resource/verb
  API->>MCP: 调用 list_pods
  MCP->>K8S: 使用操作员 ServiceAccount 查询 Pod
  K8S-->>MCP: 返回 Pod 列表
  MCP-->>API: 返回结构化工具结果
  API->>LLM: 发送工具结果生成总结
  LLM-->>API: 返回 AI 总结
  API->>DB: 保存 Chat 和审计日志
  API-->>UI: 返回总结和异常 Pod 表格
```

## 越权请求拦截

```mermaid
flowchart TD
  Ask["操作员请求：查看 prod namespace"] --> API["Backend API"]
  API --> Check["查询业务权限"]
  Check --> Deny{"是否有 prod 权限？"}
  Deny -- "否" --> Audit["写入 denied 审计日志"]
  Audit --> Reply["返回：无权访问 prod，可访问 dev/test"]
  Deny -- "是" --> MCP["调用 MCP"]
  MCP --> K8S["K8S RBAC 再次校验"]
```
