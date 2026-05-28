# Chat 与 MCP 流程

这篇文档面向想理解自然语言到 K8s 操作完整链路的架构师和开发者，说明从操作员输入到 Kubernetes API 调用的每一步。

## 1. 流程概览

```mermaid
sequenceDiagram
  actor User as 操作员
  participant UI as Frontend
  participant API as Backend API
  participant Agent as Agent Server
  participant DB as PostgreSQL
  participant LLM as LLM Provider
  participant MCP as MCP Server
  participant K8S as Kubernetes API

  User->>UI: 输入自然语言
  UI->>API: 发送 Chat 消息
  API->>DB: 校验会话归属、模型绑定，加载权限和历史消息
  API->>API: 组装 context_messages、current_input 和权限快照
  API->>Agent: gRPC AgentService.RunStream
  Agent->>LLM: Eino 发送系统提示词和多轮消息
  LLM-->>Agent: 返回工具调用请求
  Agent->>MCP: 调用 Kubernetes 工具
  MCP->>API: gRPC IdentityService.GetServiceAccount（获取用户凭据）
  API-->>MCP: 返回 ServiceAccount token/ca/apiserver
  MCP->>MCP: 校验 namespace/resource/verb 是否在用户权限内
  MCP->>K8S: 使用用户 ServiceAccount 查询
  K8S-->>MCP: 返回资源数据
  MCP-->>Agent: 返回工具结果
  Agent->>LLM: 发送工具结果
  LLM-->>Agent: 返回最终总结
  Agent-->>API: 返回总结、资源和工具事件
  API->>DB: 保存消息和审计
  API-->>UI: 返回总结和结构化结果
```

## 2. MCP Server 与 Backend 的身份/权限交互

MCP Server 每次工具调用需要两步与 Backend 交互：

### 2.1 获取用户 K8s 凭据（IdentityService）

```
MCP Server ──gRPC──▶ Backend API (IdentityService.GetServiceAccount)
                     │
                     ├── 查询 service_account_bindings 表
                     ├── 查询 service_account_tokens 表（解密 token）
                     └── 返回: ServiceAccount name, namespace, token, ca_cert, api_server
```

MCP Server 拿到凭据后构建 per-user `client-go` Kubernetes client，确保每个操作员的 K8s 请求使用自己的 ServiceAccount。

### 2.2 校验业务权限

```
MCP Server ──gRPC──▶ Backend API (IdentityService)
                     │
                     ├── 传入: userId + namespace + apiGroup + resource + verb
                     ├── 查询 k8s_permissions 表
                     └── 返回: allowed / denied + 可访问范围建议
```

### 2.3 两层校验的协作关系

```mermaid
flowchart TD
  ToolCall["Agent 调用 MCP 工具<br/>参数: namespace=prod, resource=pods"] --> GetSA["MCP → Backend IdentityService<br/>获取用户 ServiceAccount 凭据"]
  GetSA --> PermCheck["MCP → Backend 校验业务权限<br/>userId + namespace + resource + verb"]
  PermCheck --> Allowed{"允许？"}
  Allowed -- "否" --> Deny["拒绝 + 返回可访问范围提示<br/>写入 denied 审计"]
  Allowed -- "是" --> BuildClient["使用 ServiceAccount 凭据<br/>构建 per-user K8s client"]
  BuildClient --> K8SCall["调用 Kubernetes API"]
  K8SCall --> K8SRBAC{"K8s RBAC 允许？"}
  K8SRBAC -- "否" --> K8SDeny["K8s 拒绝 + 返回权限不足"]
  K8SRBAC -- "是" --> Return["返回工具结果"]
```

这种设计确保即使业务权限校验（MCP → Backend）有缺陷，Kubernetes RBAC 仍会在 API Server 层做最终拦截。

## 3. 系统提示词内容

Agent Server 通过 Eino 构造 LLM prompt 时必须包含 Backend 传入的最小必要上下文：

- 当前用户身份和角色
- 当前用户允许访问的 namespace
- 每个 namespace 下允许访问的 resource 和 verb
- 最近多轮 `context_messages`
- 当前输入 `current_input`
- 内置 MCP 工具能力说明
- 禁止越权访问说明
- 输出格式要求：自然语言总结 + 结构化资源结果

## 4. MCP 工具映射

| 工具 | Kubernetes 资源 | verb | 用途 |
|------|-----------------|------|------|
| `list_namespaces` | 业务权限摘要 | read | 返回当前用户可见 namespace |
| `list_pods` | `pods` | `list` | 查询 Pod 列表和异常状态 |
| `get_pod` | `pods` | `get` | 查询 Pod 详情 |
| `get_pod_logs` | `pods/log` | `get` | 查询 Pod 日志 |
| `list_events` | `events` | `list` | 查询事件 |
| `get_pod_events` | `events` | `get` | 查询特定 Pod 的事件 |
| `list_deployments` | `deployments.apps` | `list` | 查询 Deployment |
| `restart_deployment` | `deployments.apps` | `patch` | 通过 patch annotation 触发滚动重启 |

## 5. 异常 Pod 巡检细节

```mermaid
flowchart TD
  Ask["用户询问集群异常"] --> NS["获取授权 namespace"]
  NS --> Pods["list_pods"]
  Pods --> Filter["筛选 Failed/Pending/CrashLoopBackOff/ImagePullBackOff"]
  Filter --> Events["对异常 Pod 查询 events"]
  Filter --> Logs["必要时查询最近日志"]
  Events --> Summary["LLM 汇总原因"]
  Logs --> Summary
  Summary --> UI["UI 展示总结和表格"]
```

## 6. 错误处理

- LLM Provider 不可用：返回模型不可用提示，写入错误审计
- Chat 会话不属于当前用户：拒绝请求，写入错误审计
- `modelId` 未绑定到当前用户：拒绝请求，写入错误审计
- MCP → Backend 身份查询失败：返回"无法获取操作员凭据"
- MCP → Backend 权限拒绝：拒绝调用，返回可访问范围提示，写入 denied 审计
- Kubernetes RBAC 拒绝：返回权限不足提示，写入 Kubernetes denied 审计
- MCP Server 不可用：返回工具服务不可用提示，建议稍后重试
- Pod 日志过大：只读取 tail 行数，并在响应中说明截断

## 7. gRPC 契约

系统内部两条 gRPC 通道：

| 契约 | 方向 | 方法 | 用途 |
|------|------|------|------|
| `proto/agent/v1/agent.proto` | Backend → Agent Server | `RunStream` (server-streaming) | Backend 发起 Chat，Agent Server 流式返回事件 |
| `proto/identity/v1/identity.proto` | MCP Server → Backend | `GetServiceAccount` (unary) | MCP Server 获取操作员 K8s 凭据和校验业务权限 |

## 8. 开发扩展

新增 MCP 工具时需要同步完成：

1. 在 `mcp-server/internal/handler/` 实现工具处理器
2. 在 MCP Server 路由中注册工具端点
3. 在 Backend 权限映射中登记工具对应的 `namespace/resource/verb`
4. 在 Backend IdentityService 中补全新工具的权限校验逻辑
5. 更新本文档的工具映射表
6. 更新 [API 设计](../reference/api-design.md)
7. 增加单元测试，覆盖正常调用和越权拒绝

详见 [扩展开发指南](../developer/extension-guide.md)。
