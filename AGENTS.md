# 项目协作规则

## 文档语言

- 本项目所有 Markdown 文档必须使用中文编写
- 已存在的英文文档需要改为中文
- 代码、命令、API 路径、环境变量、协议名、镜像名、包名、表名、字段名可以保留英文，因为它们属于接口或运行时契约

## 文档目录

- 根目录文档：
  - `README.md`：项目入口、快速开始、验证命令、部署命令
  - `ARCHITECTURE.md`：架构摘要和文档入口
- `docs/` 目录：
  - `docs/README.md`：文档中心，按使用者/开发者/参考三条主线组织
  - `docs/guides/`：教程式文档，面向第一次使用者
  - `docs/product/`：产品概览、业务需求、用户流程
  - `docs/architecture/`：系统架构、权限模型、Chat/MCP 流程、数据模型
  - `docs/developer/`：开发者文档（接手、架构、原理、扩展、交付）
  - `docs/operations/`：部署、日志、审计、排错
  - `docs/security/`：安全设计
  - `docs/reference/`：API 设计、配置参考、术语表等参考资料
- `docs/superpowers/specs/`：阶段性设计规格
- `docs/superpowers/plans/`：实现计划和任务拆分
- `docs/superpowers/` 中也存放 AI 协同研发记录（`AI_PROMPTS.md`、`问题记录.md`）和远期计划

## 文档和代码一致性

- 修改需求、架构、API、部署、安全策略时，必须同步更新对应文档
- 修改代码行为时，必须检查是否影响 `README.md`、`docs/README.md` 和对应分层文档
- 修改 Helm values、脚本参数、镜像名称、端口、环境变量时，必须同步更新 `docs/operations/deployment-guide.md` 和 `docs/reference/config-reference.md`
- 修改用户角色、权限模型、LLM 管理模型、MCP 工具列表时，必须同步更新 `docs/product/requirements.md`、`docs/architecture/permission-model.md`、`docs/architecture/chat-mcp-flow.md`、`docs/reference/api-design.md` 和 `docs/security/security-design.md`
- 文档示例必须能对应到代码中的真实路径、命令、端口和配置项

## 文档写作风格

文档写作遵循 `docs-style-guide` skill 中定义的规范（`.claude/skills/docs-style-guide/SKILL.md`），核心要点：

- 每篇文档第一段说明受众和目的
- 使用编号章节（`## 1.` / `## 2.`）
- 严格区分"已实现"和"未实现"
- 文档正文中文，技术术语保留英文

## 代码注释

- 所有新增代码注释必须使用中文
- 只在复杂逻辑、权限边界、安全边界、错误处理原因不明显时添加注释
- 不添加解释显而易见代码的空洞注释

## 代码架构与设计

- 代码必须按照 DDD（Domain-Driven Design，领域驱动设计）风格进行设计和开发
- 分层结构遵循：
  - **领域层（domain）**：包含实体（entity）、值对象（value object）、聚合根（aggregate root）、领域服务（domain service）和仓储接口（repository interface），不依赖任何外部框架
  - **应用层（application/app）**：包含应用服务（application service）、DTO 和用例编排，负责协调领域对象完成业务用例
  - **基础设施层（infrastructure）**：包含仓储实现（repository implementation）、外部 API 客户端、消息队列、持久化等，实现领域层定义的接口
  - **接口层（interface/http/grpc）**：包含 HTTP handler、gRPC server、中间件等，负责协议转换和请求响应处理
- 各层依赖方向必须是单向的：接口层 → 应用层 → 领域层 ← 基础设施层
- 领域层作为核心，不依赖任何外层，基础设施层通过依赖反转实现领域层接口
- 每个服务（backend、agent-server、mcp-server）内部按此分层组织代码

## 程序日志

- 程序日志使用英文，便于运行环境、Kubernetes、CI/CD 和第三方日志平台统一检索
- 日志必须尽可能清晰、利于排错，并包含明确级别
- 日志格式建议：
  - `level=INFO component=backend event=server_start addr=:8080`
  - `level=ERROR component=mcp-server event=server_exit error="..."`
- 错误日志必须包含错误原因和关键上下文，不记录敏感信息
- 禁止在日志中输出 LLM API Key、ServiceAccount token、Kubernetes Secret 内容、用户密码

## AI 协同研发问题记录

- 在用 AI 实现代码过程中遇到的任何问题、bug、踩坑，都必须记录到 `docs/superpowers/问题记录.md` 中
- 每条记录应简短包含：**问题描述** + **原因** + **解决方案**
- 典型问题类型包括但不限于：
  - AI 生成代码质量不符合预期（缺少框架选型、非企业级风格、未使用 DDD 分层等）
  - AI 自行选择技术方案导致偏离预期（需人把关纠正）
  - 需求/架构未提前对齐导致 AI 反复修改、token 消耗大
  - 部署环境问题（镜像源、网络等）导致 AI 操作失败
  - 前端与后端脱节，AI 只关注后端逻辑忽略用户操作流程
- 每次遇到问题并解决后，立即追加记录，避免遗忘

## 安全和审计

- LLM 工具调用参数视为不可信输入，执行前必须做业务权限校验
- K8S API 调用必须使用当前用户绑定的 ServiceAccount，不能使用前端传入的角色声明
- 管理员操作、权限变更、LLM 配置变更、K8S 工具调用和拒绝访问都要写审计日志
