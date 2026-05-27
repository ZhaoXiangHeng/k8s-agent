---
name: core-agent-kweaver-integration
description: 用于指导 `D:\code\v9_work\Anybackup\Agent\service\core_agent` 下的 core_agent_service 通过 pip 安装的 kweaver-sdk 对接 KWeaver Core Decision Agent。 当任务涉及 Python SDK 初始化、平台会话创建与复用、taskId 到 conversation 的映射、结构化响应解析或适配层封装时使用。
---

# Core Agent KWeaver 接入

## 概述

这个技能用于把 `core_agent_service` 对 `KWeaver Decision Agent` 的调用方式固定下来。
源码实施目录固定为 `D:\code\v9_work\Anybackup\Agent\service\core_agent`。
它强调 SDK 只通过 pip 包接入，并且只通过统一适配层调用。

## 硬约束

- 依赖来源必须是 `pip install kweaver-sdk`
- 代码、测试与文档变更必须落在 `D:\code\v9_work\Anybackup\Agent\service\core_agent`
- 不允许直接引用工作区中的本地 SDK 源码目录
- 首次消息允许创建平台会话
- 后续续跑必须复用同一个平台 `conversation_id`
- 非结构化响应必须走兼容路径，不能直接作为稳定契约返回

## 推荐封装点

- `create_client`
- `send_task_payload`
- `extract_conversation_id`
- `parse_agent_response`

## 参考资料

- Python SDK 接入说明：`references/python-sdk-guide.md`
