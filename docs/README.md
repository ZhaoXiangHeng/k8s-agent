# 文档中心

这组文档按受众和关注层次组织，避免把产品、架构、开发、运维、安全信息混在同一篇文档里。

文档分成三条主线：

- **使用者路径**：适合第一次接触系统，按步骤完成部署和配置
- **开发者路径**：适合接手代码、理解架构、扩展功能
- **参考手册路径**：适合日常运维、查字段、查 API、排错

## 使用者路径

- [快速开始](guides/quickstart.md) — 5 分钟了解系统并体验最小路径
- [配置操作员](guides/configuring-operators.md) — 管理员创建操作员、分配权限、绑定模型的完整流程
- [产品概览](product/overview.md) — 项目背景、核心价值、MVP 演示闭环
- [业务需求](product/requirements.md) — 角色定义、功能需求、非功能需求和验收标准
- [用户流程](product/user-journeys.md) — 管理员和操作员的核心业务时序

## 开发者路径

- [开发者文档中心](developer/README.md) — 开发者入口，包含当前实现概览
- [快速接手](developer/getting-started.md) — 理解目录结构、代码入口、本地开发环境
- [架构总览](developer/architecture-overview.md) — 模块边界、DDD 分层、核心执行链路
- [技术原理](developer/technical-principles.md) — 无状态 Agent、ReAct loop、MCP 工具发现、三层权限防线
- [扩展开发指南](developer/extension-guide.md) — 新增 MCP 工具、LLM Provider、API 端点
- [交付流程](developer/delivery-workflow.md) — 从代码到镜像到部署的完整链路

## 架构与设计

- [系统架构](architecture/system-architecture.md) — 组件拓扑、Traefik 网关、服务职责、部署拓扑
- [权限模型](architecture/permission-model.md) — Keycloak、业务权限、Kubernetes RBAC 三层授权
- [Chat 与 MCP 流程](architecture/chat-mcp-flow.md) — 自然语言、Eino Agent、gRPC、MCP 工具执行时序
- [数据模型](architecture/data-model.md) — 核心表、关系和数据边界

## 部署运维

- [部署指南](operations/deployment-guide.md) — Kind、Helm、本地 tar 镜像、registry 镜像部署
- [日志、审计与排错](operations/observability-and-troubleshooting.md) — 日志规范、审计事件、排错路径

## 参考手册

- [API 设计](reference/api-design.md) — Backend HTTP API 分组、gRPC 契约、错误码
- [配置参考](reference/config-reference.md) — 环境变量、Helm values、端口约定
- [术语表](reference/glossary.md) — 项目关键概念

## 安全

- [安全设计](security/security-design.md) — 认证、授权、LLM 安全、威胁与控制

## 根目录文档

- [README](../README.md) — 项目首页和快速命令
- [ARCHITECTURE](../ARCHITECTURE.md) — 架构摘要入口
- [AGENTS](../AGENTS.md) — 项目协作规则

## 历史设计与计划

设计规格和实现计划沉淀在 `docs/superpowers/specs/` 和 `docs/superpowers/plans/` 中，它们记录阶段性设计决策，不一定完全等于当前代码实现。
