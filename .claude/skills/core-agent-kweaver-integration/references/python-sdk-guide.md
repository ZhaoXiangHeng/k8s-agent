# Python SDK 接入指南

## 依赖安装

```bash
pip install kweaver-sdk
```

## 导入方式

```python
from kweaver import KWeaverClient, TokenAuth
```

或：

```python
import kweaver
```

## 初始化建议

```python
from kweaver import KWeaverClient, TokenAuth

client = KWeaverClient(
    base_url="https://kweaver.example.com",
    auth=TokenAuth("token"),
    business_domain="bd_public",
    timeout=30.0,
)
```

## 会话发送建议

`Decision Agent` 的会话通常通过 `client.conversations.send_message(...)` 发起。

首次发送时可以传入空 `conversation_id`：

```python
message = client.conversations.send_message(
    "",
    content='{"task":{"taskId":"task_123"}}',
    agent_id="agent_001",
    stream=False,
)
```

返回结果里如果带有平台 `conversation_id`，要立即保存到本地数据库，供后续同一个 `taskId` 复用。

## 适配层建议

不要在应用层直接写：

- `client.conversations.send_message(...)`
- `client.agents.get(...)`

而是封装为：

- `KWeaverAgentGateway.send_task_payload(...)`
- `KWeaverAgentGateway.resolve_agent(...)`

## 错误处理建议

- 网络错误映射为基础设施错误
- SDK 认证错误映射为配置错误
- 非 JSON 文本响应映射为协议错误
- 空响应映射为上游不可用错误

## 禁止事项

- 禁止通过修改 `PYTHONPATH` 直接引用工作区里的 `kweaver-sdk`
- 禁止把 SDK 返回结构直接透传给上层 gRPC 契约
- 禁止在多个模块里重复初始化 `KWeaverClient`
