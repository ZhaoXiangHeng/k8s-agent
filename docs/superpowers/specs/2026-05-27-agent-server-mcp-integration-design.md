# Agent Server MCP 集成 & 流式 ReAct 设计

## 1. 概述

将 agent-server 从硬编码 mock 改造成具备真实智能的 Agent 服务：通过 Eino ReAct 循环自主调用 MCP 工具，通过 server-streaming gRPC 实时推送中间事件，由 LLM 分析 K8s 资源数据并给出最终结论。

## 2. 架构

```
 Frontend ←SSE→ Backend ←gRPC Stream→ Agent Server ←SSE (MCP)→ MCP Server ←client-go→ K8s API
                                                  ↓
                                                LLM (OpenAI / Anthropic / ...)
```

### 2.1 组件职责

| 组件 | 职责 | 不管 |
|---|---|---|
| **mcp-server** | 使用 `mark3labs/mcp-go` 实现标准 MCP 协议，提供 K8s 资源 CRUD 工具，返回原始数据 | 不判断 Pod 异常、不分析原因 |
| **agent-server** | 接收 ModelRuntimeConfig 动态创建 LLM ChatModel，Eino ReAct 循环决定调用哪些 MCP 工具，LLM 分析 K8s 数据并总结，通过 server-streaming gRPC 推送事件 | 不直接调用 K8s API、不写 K8s 资源的判断逻辑 |
| **后端** | gRPC stream → SSE 纯中转，持久化对话消息到 chat_messages 表 | 不解析 StreamEvent 内容、不做决策、不调 MCP |
| **前端** | 渲染 StreamEvent（思考/工具调用/工具结果/资源/完成） | — |

## 3. Proto 变更

**文件：** `proto/agent/v1/agent.proto`

将 `Run(AgentRunRequest) returns (AgentRunResponse)` 替换为：

```protobuf
service AgentService {
  rpc RunStream(AgentRunRequest) returns (stream StreamEvent);
}
```

### 3.1 StreamEvent

```protobuf
message StreamEvent {
  string event_id = 1;
  int64 timestamp = 2;

  oneof event {
    ThinkingEvent   thinking    = 10;  // LLM 思考/流式文本
    ToolCallEvent   tool_call   = 11;  // Agent 决定调用 MCP 工具
    ToolResultEvent tool_result = 12;  // MCP 工具返回结果
    ResourceEvent   resource    = 13;  // 发现的 K8s 资源
    CompleteEvent   complete    = 14;  // 最终总结
    ErrorEvent      error       = 15;  // 错误事件
  }
}

message ThinkingEvent {
  string content = 1;
}

message ToolCallEvent {
  string tool_name = 1;
  string arguments_json = 2;   // JSON string，与 proto 解耦
}

message ToolResultEvent {
  string tool_name = 1;
  bool success = 2;
  string result_json = 3;      // MCP 返回的原始 JSON
}

message ResourceEvent {
  ResourceResult resource = 1; // 复用现有 ResourceResult
}

message CompleteEvent {
  string summary = 1;
  repeated ResourceResult resources = 2;
}

message ErrorEvent {
  string code = 1;
  string message = 2;
}
```

### 3.2 现有消息保持不变

`AgentRunRequest`、`ModelRuntimeConfig`、`Permission`、`ToolRef`、`RuntimeContext`、`ResourceResult` 保持现有定义不变。

## 4. MCP Server 改造

**文件范围：** `mcp-server/`

### 4.1 使用 `mark3labs/mcp-go` 替代自定 REST API

替换现有的 `GET /mcp/tools` + `POST /mcp/tools/call`，改为标准 MCP 协议：

- `GET /sse` — SSE endpoint（Eino 客户端连接点）
- `POST /messages` — 客户端到服务端的消息通道
- `GET /healthz` — 保留

### 4.2 工具定义

使用 `mcp.NewTool()` 定义工具：

| 工具 | 参数 | 返回值 |
|---|---|---|
| `list_namespaces` | — | `[{name, status}]` |
| `list_pods` | namespace, label_selector | `[{name, namespace, phase, containers, ...}]` |
| `get_pod` | namespace, name | 单个 Pod 完整对象 |
| `get_pod_logs` | namespace, name, container, tail_lines | Pod 日志字符串 |
| `get_pod_events` | namespace, name | `[{type, reason, message, timestamp}]` |
| `list_deployments` | namespace | `[{name, namespace, replicas, ...}]` |
| `restart_deployment` | namespace, name | `{success, message}` |

### 4.3 K8s 客户端

使用 `client-go` 实现真实的 K8s API 调用。每个工具 handler 只做权限范围内 API 调用并原样返回数据，不包含任何分析判断逻辑。

### 4.4 删除项

- 删除 `internal/mcp/types.go`（ToolDefinition/CallRequest/CallResponse — 已被 mcp-go 替代）
- 删除 `internal/mcp/registry.go`（工具注册 — 由 mcp-go Server.AddTool 替代）
- 删除 `internal/tools/pods.go` 中的判断逻辑（IsAbnormalPodPhase/IsAbnormalPodReason — LLM 在 ReAct 循环中自主判断 Pod 状态，不需要代码层面分析）

## 5. Agent Server 核心

**文件范围：** `agent-server/`

### 5.1 目录结构

```
agent-server/internal/
├── eino/
│   ├── runner.go              ← Eino ReAct runner（重构核心）
│   ├── runner_test.go
│   ├── config.go              ← MCP_SERVER_URL env 读取
│   ├── llm/
│   │   └── factory.go          ← 根据 ModelRuntimeConfig 创建 ChatModel
│   └── mcp/
│       └── client.go           ← SSE MCP 客户端初始化 + GetTools()
└── server/
    └── server.go               ← 改为实现 RunStream handler
```

### 5.2 启动流程

1. 从 `MCP_SERVER_URL` 环境变量读取 mcp-server 地址
2. `mcp.NewSSEMCPClient(ctx, url)` 建立 SSE 连接
3. `mcp.GetTools(ctx, config)` 发现并注册所有 MCP 工具
4. 等待 gRPC 请求

### 5.3 RunStream 核心流程

```
1. 从 AgentRunRequest.ModelRuntimeConfig 创建对应 ChatModel
2. 构建系统提示词 + messages
3. 创建 ReAct agent，注入 MCP 工具列表
4. agent.Stream() 启动流式循环
5. for each stream message:
     - schema.Assistant → 发送 ThinkingEvent
     - schema.Tool       → 发送 ToolCallEvent → 等待工具执行 → 发送 ToolResultEvent
6. 循环结束 → 发送 CompleteEvent(summary + resources)
7. 任何错误 → 发送 ErrorEvent
```

### 5.4 LLM Factory

根据 `ModelRuntimeConfig.ProviderId` 动态创建 ChatModel：

| ProviderId | ChatModel | 备注 |
|---|---|---|
| `openai` | `openai.NewChatModel(cfg)` | 兼容 OpenAI API 的服务（DeepSeek、通义千问等） |
| `anthropic` | `anthropic.NewChatModel(cfg)` | Anthropic Claude |

- `baseUrl`、`apiKey`、`modelName` 全部来自 `ModelRuntimeConfig`
- 如果 Eino 没有直接的 provider adapter，使用 OpenAIChatModel 统配（通过 baseUrl 指向不同服务）

### 5.5 删除项

- 删除 mock 数据（当前 runner.go 中硬编码的 Pod/namespace）
- 删除 `systemContext()` 中硬编码的指令（保留基础的 system prompt 构建，但由 LLM 自主推理）

## 6. 后端改造

**文件范围：** `backend/internal/agent/client.go`、`backend/internal/http/router.go`

### 6.1 gRPC 客户端升级

- 从 `Run()` 改为 `RunStream()`
- 实现 streaming 消费逻辑：逐事件接收 → 写入 channel

### 6.2 HTTP handler 改造

- `createChatMessage` handler：改为 SSE 输出
- 从 agent stream channel 读取 StreamEvent → 序列化 JSON → `data: {...}\n\n` 推给前端
- Stream 结束时：保存最终消息到 `chat_messages` 表

### 6.3 删除项

- 删除 `backend/internal/agent/model.go` 中 Client interface 的 `Run()` → 改为 `RunStream()`
- 删除 `backend/internal/config/config.go` 中的 `MCPServerURL`（后端不再直连 MCP）

## 7. 数据流示例

以"检查 dev 命名空间异常"为例：

```
1. 用户: "检查 dev 命名空间"
2. 后端 → agent-server: AgentRunRequest { message: "检查 dev 命名空间", model: {...}, permissions: [...], messages: [...] }
3. agent-server → 后端 StreamEvent:
   - ThinkingEvent:     "用户想检查 dev 命名空间，我先列出所有 Pod"
   - ToolCallEvent:     tool=list_pods(args={"namespace":"dev"})
   - ToolResultEvent:   tool=list_pods success=true result_json=[{name:"worker-x1y2",phase:"Pending",...}]
   - ThinkingEvent:     "发现 worker-x1y2 状态 Pending，容器 CrashLoopBackOff，拉取事件和日志"
   - ToolCallEvent:     tool=get_pod_events(args={"namespace":"dev","name":"worker-x1y2"})
   - ToolCallEvent:     tool=get_pod_logs(args={"namespace":"dev","name":"worker-x1y2","tail_lines":50})
   - ToolResultEvent:   tool=get_pod_events success=true result_json=[...]
   - ToolResultEvent:   tool=get_pod_logs success=true result_json="panic: db connection failed"
   - ThinkingEvent:     "根因是数据库连接失败导致容器反复重启"
   - ResourceEvent:     worker-x1y2 Pod 详情
   - CompleteEvent:     summary="dev 中有 1 个异常 Pod..." resources=[worker-x1y2]
4. 后端 → 前端: SSE 推送每个事件
5. 后端: 保存最终消息到 chat_messages
```

## 8. 错误处理

| 场景 | 行为 |
|---|---|
| MCP server 不可达 | StartUp 时记录并重试，RunStream 返回 ErrorEvent |
| LLM API 调用失败 | 返回 ErrorEvent，agent 循环中止 |
| MCP 工具执行错误 | ToolResultEvent(success=false)，LLM 根据错误重试或跳过 |
| gRPC stream 中断 | 后端检测 ctx.Done()，关闭 HTTP 连接 |
| LLM 返回非标准格式 | ErrorEvent，记录原始响应 |

## 9. 测试策略

- **mcp-server**: 集成测试验证 MCP 工具定义和 SSE 端点 + K8s fake client
- **agent-server**: 单元测试 mock MCP + mock LLM 验证 ReAct 流程，集成测试验证动态 LLM 创建
- **后端**: 单元测试验证 gRPC stream → SSE 转换，集成测试端到端
- **Proto**: 生成代码编译通过

## 10. 迁移影响

- **proto 不兼容变更**: `Run` RPC 被 `RunStream` 替换，需要重新生成代码
- **后端**: agent client 需要从 unary 调用改为 streaming
- **前端**: 解析从单次响应改为 SSE 事件流
- **mcp-server**: REST API 完全被 MCP 协议替代
- **agent-server**: runner 完全重写
