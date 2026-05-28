# 开发者文档中心

这组文档面向准备接手 k8s-agent 代码、扩展能力或维护交付流程的开发者。

## 推荐阅读顺序

1. [快速接手](./getting-started.md) — 理解目录、跑通命令、找到修改入口
2. [架构总览](./architecture-overview.md) — 理解模块边界、分层架构和执行链路
3. [技术原理](./technical-principles.md) — 理解 Eino/ReAct、MCP 工具发现、SSE 流式、权限纵深防线等核心机制

按需阅读：

- [扩展开发指南](./extension-guide.md) — 新增 MCP 工具、LLM Provider、API 端点的方法
- [交付流程](./delivery-workflow.md) — 从代码到镜像到部署的完整交付链路

## 这组文档解决什么问题

它不是面向使用者的操作手册，而是面向开发者回答这些问题：

- 这套系统为什么存在、解决什么问题
- 四个服务（Backend、Agent Server、MCP Server、Frontend）分别负责什么
- 核心模块之间如何通信（HTTP / gRPC / SSE）
- Eino ReAct agent loop 如何工作
- MCP 工具如何注册、发现和权限校验
- 新开发者应该从哪个文件开始读
- 新增功能（工具、模型、API）应该改哪些模块
- 如何从本地开发推进到镜像构建和部署

## 当前实现概览

截至当前仓库状态，各服务已实现的核心能力：

| 服务 | 已实现 | 尚未实现 |
|------|--------|----------|
| **Backend** | HTTP API 14 个端点；MemoryStore + PostgresStore；K8s RBAC Manager (ServiceAccount/Role/RoleBinding 动态同步)；gRPC IdentityService server + AgentService client；SSE 事件中继；审计日志 | Keycloak JWT audience 校验；Keycloak Admin API 集成；PostgreSQL migration 版本管理；Redis 业务缓存 |
| **Agent Server** | 基于 Eino ADK ChatModelAgent 的 ReAct loop；MCP SSE client 工具发现与注册；Skills 系统（`SKILLS_DIR`）；server-streaming gRPC `RunStream` | Skills 按需加载机制；更丰富的 prompt 模板 |
| **MCP Server** | 8 个 K8s 运维工具；per-user K8s client（通过 IdentityService gRPC）；SSE transport；工具调用前权限校验 | 更多工具类型（如 ConfigMap/Secret 管理） |
| **Frontend** | React Web UI；管理员/操作员双控制台；Chat SSE 流式渲染（markdown） | 真实 Keycloak 登录流程对接；完整 API 集成 |

## 关键验证命令

```bash
cd backend && go test ./...
cd agent-server && go test ./...
cd mcp-server && go test ./...
cd proto && go test ./...
cd frontend && npm install && npm run build
bash -n scripts/bootstrap-local.sh scripts/helm-install.sh scripts/helm-upgrade.sh scripts/uninstall.sh scripts/build-images.sh
```

## 与现有文档的关系

- 面向使用者：`docs/guides/` 和 `docs/reference/`
- 面向架构和技术负责人：`docs/architecture/`
- 面向运维人员：`docs/operations/`
- 历史设计与计划：`docs/superpowers/`
- 本目录专门服务"接手代码和继续开发"的人
