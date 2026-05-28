# 扩展开发指南

这篇文档面向准备在 k8s-agent 现有框架上扩展新能力的开发者。

## 1. 扩展前先判断问题属于哪一类

在当前架构里，扩展需求通常落在这几类：

- 新增 MCP 工具（如 ConfigMap 管理、Secret 巡检）
- 新增 LLM Provider 协议
- 新增 Backend API 端点
- 新增 Kubernetes RBAC 管理能力
- 新增 Skills（运维知识单元）
- 新增 Helm Chart 组件

先把问题分类清楚，再改代码，会少走很多弯路。

## 2. 新增 MCP 工具

这是当前最常见、也最现实的扩展方向。以新增 `get_configmap` 工具为例：

### 2.1 在 MCP Server 实现工具处理器

1. 在 `mcp-server/internal/handler/` 中新增工具处理器
2. 定义工具的输入参数和返回结果结构
3. 实现工具逻辑，包含权限校验和 K8s API 调用

### 2.2 注册工具端点

在 MCP Server 的路由中注册新工具，供 Agent Server 通过 MCP 协议发现。

### 2.3 更新 Backend 权限映射

在 Backend 的权限配置中登记新工具的 `namespace/resource/verb` 映射，确保权限控制覆盖新工具。

### 2.4 更新文档

- [Chat 与 MCP 流程](../architecture/chat-mcp-flow.md) — 更新工具表
- [技术原理](technical-principles.md) — 更新已注册工具列表

### 2.5 增加测试

- 正常调用测试
- 越权拒绝测试
- K8s API 不可达时的错误处理测试

## 3. 新增 LLM Provider 协议

### 3.1 在 Agent Server 增加协议支持

1. 在 `agent-server/internal/eino/llm/factory.go` 增加新协议类型
2. 实现对应协议的 ChatModel 创建逻辑
3. 利用 Eino 框架统一 Chat Request/Response 和工具调用解析

### 3.2 在 Backend 增加 Provider 校验

1. 在 Backend 的 LLM 管理 API 中增加新协议的校验
2. 更新 Provider 创建接口的字段验证规则

### 3.3 增加 mock 测试

新增 Provider 时不依赖真实外部模型。

## 4. 新增 Backend API 端点

### 4.1 遵循 DDD 分层

按现有分层结构逐步修改：

1. `domain/`：定义新的实体、值对象或仓储接口
2. `app/`：新增应用服务和 DTO
3. `infra/postgres/`：实现新的仓储方法
4. `interfaces/http/`：新增 handler 并在 routes.go 注册路由

### 4.2 同步更新

- `docs/reference/api-design.md` — 新增端点文档
- `docs/architecture/permission-model.md` — 如涉及权限变更
- `docs/security/security-design.md` — 如涉及安全变更
- Backend 测试 — 覆盖正常调用和权限拒绝

## 5. 新增 Kubernetes RBAC 管理能力

### 5.1 在 Backend RBAC Manager 扩展

`backend/internal/infra/k8s/` 中的 RBAC Manager 可按需扩展：
- 新增 K8s 资源类型管理（如 NetworkPolicy）
- 增强 Role 的 API Group 覆盖
- 支持更多 verb 组合

### 5.2 更新 Helm Chart

- 在 `deploy/helm/k8s-ai-ops/templates/rbac.yaml` 中声明新增的权限
- 在 `values.yaml` 中暴露配置项

## 6. 新增 Skills

### 6.1 创建 Skill 文件

在 `SKILLS_DIR` 目录下创建子目录，包含：

- `SKILL.md`：技能定义文件，描述技能用途、参数和用法
- 可选脚本、模板等辅助文件

### 6.2 Skill 结构示例

```
SKILLS_DIR/
├── k8s-troubleshoot/
│   └── SKILL.md
├── deploy-rollout/
│   └── SKILL.md
└── config-audit/
    ├── SKILL.md
    └── audit-policy.yaml
```

### 6.3 Skill 加载机制

1. Agent Server 启动时加载 skill 元数据索引（名称、描述）
2. ReAct loop 中，当 LLM 判断需要使用某 skill 时，动态加载对应 `SKILL.md`
3. Skill 中引用的 MCP 工具映射实际 K8s 操作

## 7. 什么时候应该新增服务

可以用这个经验规则判断：

- 如果功能有独立的运行时边界、独立的扩缩需求、独立的故障域 → 考虑新增服务
- 如果功能需要访问新类型的外部系统（如 Prometheus、Grafana API）→ 考虑新增 MCP 工具或新 MCP Server
- 如果功能只是现有服务的逻辑扩展 → 在现有服务内扩展

## 8. 测试策略建议

扩展时优先补这几类测试：

| 测试类型 | 示例 |
|----------|------|
| 工具正常调用 | `TestListPods_Success` |
| 越权拒绝 | `TestListPods_UnauthorizedNamespace` |
| K8s API 错误处理 | `TestListPods_K8sAPIError` |
| gRPC 契约兼容性 | `proto/` 的生成代码验证 |
| 权限同步 | `TestRBACManager_ApplyPermissions` |

当前测试使用 `client-go` fake client 避免依赖真实 K8s 集群。

## 9. 文档同步要求

每次扩展功能后，至少同步这些文档：

- [API 设计](../reference/api-design.md) — 新增或修改的接口
- [Chat 与 MCP 流程](../architecture/chat-mcp-flow.md) — 新增的 MCP 工具
- [权限模型](../architecture/permission-model.md) — 权限变更
- [安全设计](../security/security-design.md) — 安全影响
- [开发者文档中心](./README.md) — 当前实现概览
- [配置参考](../reference/config-reference.md) — 新增的配置项

如果只改代码不改文档，后续维护成本会非常高。

## 10. 扩展时的设计原则

继续扩展时，建议始终守住这几个原则：

- **无状态 Agent**：Agent Server 不持久化业务数据
- **纵深防线**：prompt 限制 + MCP 校验 + K8s RBAC 三层权限
- **契约优先**：proto/ 是服务间通信的唯一来源
- **Per-user 隔离**：K8s 操作使用操作员 ServiceAccount
- **审计不可少**：管理员操作和操作员 K8s 操作都要写入审计日志
- **默认路径简单**：复杂能力渐进增加
