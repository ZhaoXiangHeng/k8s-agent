# 文档重构设计

## 目标

参考 `deploy-descision-agent` 仓库的文档组织风格，对 `docs/` 目录进行重构，解决当前文档的重复、层次混乱和受众不明确问题。

## 背景问题

1. **"当前实现状态"章节在 7 个文档中重复**，内容几乎相同，维护成本高且容易不一致
2. **开发者文档缺少索引入口**，没有 `developer/README.md`
3. **缺少 guides/ 分层**，教程式和参考式文档混在一起
4. **大部分文档没有开头说明受众**，"谁该读这篇"不明确
5. **根目录文档臃肿**，`ARCHITECTURE.md`、`AI_PROMPTS.md` 与 `docs/` 内容重叠
6. **`docs/operations/public-cloud-test-plan.md`** 是远期计划，不应放在当前运维文档中
7. **缺少开发者快速上手路径**（接手指南、技术原理、扩展指南、交付流程）

## 参考风格（来自 deploy-descision-agent）

- 每篇文档第一段说明受众和目的："这篇文档面向 XX，重点回答 YY"
- 编号章节 `## 1.` / `## 2.`
- 技术原理类使用"原则 + 落地方式"模式
- 严格区分：已实现 / 当前限制 / 后续方向
- 使用者文档(guides/) vs 开发者文档(developer/) vs 参考手册(reference/) 严格分层
- 每个子目录的 README.md 解释这组文档"解决什么问题"
- 中文文档，技术术语保留英文

## 新目录结构

```
docs/
├── README.md                          # 文档中心（重写）
├── guides/                            # [新增] 教程式文档
│   ├── quickstart.md                  # 5 分钟快速体验
│   └── configuring-operators.md       # 管理员配置操作员完整流程
├── product/                           # [保留]
│   ├── overview.md
│   ├── requirements.md
│   └── user-journeys.md
├── architecture/                      # [保留]
│   ├── system-architecture.md
│   ├── permission-model.md
│   ├── chat-mcp-flow.md
│   └── data-model.md
├── developer/                         # [重构]
│   ├── README.md                      # [新增] 开发者文档索引
│   ├── getting-started.md            # 快速接手
│   ├── architecture-overview.md      # 架构总览（模块划分 + 执行链路）
│   ├── technical-principles.md       # [新增] 技术原理
│   ├── extension-guide.md            # [新增] 扩展指南
│   └── delivery-workflow.md          # [新增] 交付流程
├── operations/                        # [精简]
│   ├── deployment-guide.md
│   └── observability-and-troubleshooting.md
├── reference/                         # [保留]
│   ├── api-design.md
│   ├── config-reference.md           # [新增] 配置参考
│   └── glossary.md
├── security/                          # [保留]
│   └── security-design.md
└── superpowers/                       # [保留]
    ├── specs/
    └── plans/
```

## 核心改动

### 1. docs/README.md 重写
按"使用者路径"和"开发者路径"两条主线组织，参考 deploy-descision-agent 风格。

### 2. 新增 docs/guides/
提取教程式内容，面向第一次使用者：
- `quickstart.md`：最小路径体验
- `configuring-operators.md`：管理员配置操作员完整流程

### 3. 重构 docs/developer/
拆分原 `developer-guide.md` + 补充新内容：
- `README.md`：开发者文档索引
- `getting-started.md`：快速接手（目录结构、入口文件、验证命令）
- `architecture-overview.md`：模块划分、分层架构、执行链路
- `technical-principles.md`：Eino/ReAct、MCP 工具发现、SSE 流式、幂等等核心机制
- `extension-guide.md`：如何新增 MCP 工具、LLM Provider、API 端点
- `delivery-workflow.md`：构建→推送→部署→验证完整链路

### 4. 删除所有文档中重复的"当前实现状态"章节
统一收口到 `developer/README.md` 中，以单一来源维护。

### 5. 移动 public-cloud-test-plan.md
从 `docs/operations/` 移动到 `docs/superpowers/plans/`。

### 6. 精简根目录文档
- `ARCHITECTURE.md`：精简为指向 docs/ 的入口，删除重复内容
- `AI_PROMPTS.md`：移到 `docs/superpowers/`，保持原有内容

### 7. 新增 reference/config-reference.md
汇总环境变量、Helm values、服务端口等配置参考。

### 8. 统一文档风格
所有文档：开头说明受众 + 编号章节 + 区分已实现/未实现/后续方向。

## 实施顺序

1. 先创建新文件（guides/, developer/ 新文档, config-reference.md）
2. 重写现有文档（docs/README.md, 各 product/architecture/ 文档的"当前实现状态"清理）
3. 移动文件（public-cloud-test-plan.md, AI_PROMPTS.md）
4. 更新入口链接（README.md, ARCHITECTURE.md, docs/README.md, AGENTS.md）
5. 最终验证所有文档链接有效
