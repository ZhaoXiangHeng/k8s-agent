# AI 协同研发记录

这个项目（K8S AI Ops）是我用 AI 辅助完整开发的一个作品。从最初的 PDF 面试题需求，到最终部署在云主机上可以跑，全程由 Claude Code + DeepSeek 配合完成。这里记录一下整个过程中我是怎么用 AI 的、写了哪些提示词、遇到了什么问题、以及摸索出来的一套方法论。

## 用的 AI 工具

试过三种，最终稳定在 Claude Code + DeepSeek：

- **Claude Code**：主力。上下文够长（200K），有 Superpowers 技能体系，该有的工具都有。从需求分析到部署全部用它。
- **DeepSeek V4**：作为 Claude Code 的后端推理模型，成本低，中文理解也还行。
- **Codex (OpenAI)**：一开始用过，设计质量不错，但 token 很快就用完了，长任务容易断。后来切到 Claude+DeepSeek。

当时在 `问题记录.md` 里记过：

> "codex 写代码的质量高 但是 token 不够用 很快就用完了  后来改用 claude+deepseek 也还可以"

> "一开始使用阿里云自带的 agent 帮我安装 k8s...因为免费、好奇心也想试试他们的效果...后来还是通过我本地的 claude+deepseek 远程操作云主机修复"

---

## 我实际写的提示词和 AI 给的结果

### 一、初始需求

刚开始的需求很简单，就是根据面试题 PDF 把项目做出来：

```
按照 docs/golang挑战任务.pdf 这个文档中的面试任务要求，帮我开发这个 k8s 运行 AI。
先梳理需求，生成需求文档；完成技术选型、架构设计、服务设计、实现设计；文档完善后进行代码开发。
```

AI 从 PDF 里提取了 MVP 需求，生成了产品需求文档和初始技术选型。

---

然后我追加了产品需求，把系统从单角色扩展成多角色平台：

```
产品需要多角色用户体系，包含管理员和操作员。
管理员创建管理员和操作员，分配操作员可以操作的 Kubernetes namespace/resource/verb 权限。
系统动态生成 ServiceAccount、Role、RoleBinding 并绑定到操作员 ID。
管理员管理 OpenAI 和 Anthropic 协议的 LLM provider/model，并给操作员绑定可用模型。
操作员通过 chat 自然语言管理自己有权限的 Kubernetes 资源。
```

AI 给出了基于 Keycloak 的多角色方案、namespace 级动态 K8s RBAC 和 ServiceAccount 隔离、以及 Provider/Model 管理模型。

---

部署方面：

```
安装部署脚本需要支持一键同时安装本地 Kubernetes 和 Helm 部署，也支持只部署和更新 Helm。
部署镜像支持本地 tar 包或者镜像仓库地址，默认使用本地 tar 包。
```

AI 给出了 Helm Chart（最终 15 个模板）、Kind 一键启动脚本、tar 和 registry 两种镜像来源的方案。

---

文档和代码规范方面，我把约束写清楚：

```
生成文件写入项目规则。
项目文档路径需要明确。
更新代码或文档时需要保持两者一致。
所有文档需要中文，已经是英文的需要改成中文。
所有代码注释是中文。
所有程序日志使用英文，并尽可能清晰、利于排错、具备分级。
```

AI 生成了 `AGENTS.md`，建立了文档-代码一致性约束、DDD 分层要求、日志规范等。

### 二、架构设计和代码质量的反复磨合

这个阶段花了不少时间。AI 生成的代码一开始质量不够，需要我反复给反馈纠正。

**代码不够企业级**：

```
开发的代码不是企业级的，后来改成 DDD 领域驱动开发。
开发代码的时候缺少一些代码风格的约束，生成的代码不够企业级代码。
```

AI 把所有服务（Backend、Agent Server、MCP Server）重构成了 DDD 四层架构（domain → app → infra → interfaces）。后续新代码都按这个分层来。

---

**技术选型没提前对齐**：

```
继续选型的时候 没有提及到需要使用 orm 框架和 redis 的框架的库
再写代码的时候 agent 就没有自己决策出要用第三方库（人的把关还是很重要的）
```

```
mcp server 技术栈，agent-service 技术栈和集成 mcp server 的方式 一开始没有说明
AI 乱搞 没有用工程化的第三方框架
（经验：还是需要在服务拆分的时候和 AI 对齐）
```

这两次踩坑之后我意识到——AI 不会主动帮你选第三方框架。必须我在 Spec 阶段就明确指定 GORM、go-redis、Eino、mcp-go 这些。后来每个 Spec 文档都有明确的技术选型章节。

---

**前端 UX 被忽略了**：

```
只考虑了后端实现业务，没有考虑页面的呈现，导致前端代码边写边思考操作流程。
```

补了一个独立的 Frontend MVP Spec + Plan，完整设计了管理员/操作员双控制台的页面流程、组件树、SSE 流式渲染和 Keycloak PKCE 登录。

### 三、远程环境搭建

部署目标是一台公网云主机。环境搭建也是 AI 帮忙弄的。

**K8s 排错**：

```
帮我远程看一下 云主机 这台机器上的 k8s 是否正常安装后运行了
如果没有帮我修复。
```

AI 通过 SSH 连上去诊断，定位到阿里云 agent 用的外网镜像源有问题，切到国内源重新安装。

---

**写了个远程操作工具**：

```
你帮我写一个 python 脚本去支持远程对云主机操作
```

AI 生成了 `scripts/remote_exec.py`（基于 paramiko），支持 SSH exec、SFTP 上传下载。这个脚本后来成了所有构建/部署 Skill 的基础。

---

**后续各种运维操作**：

```
重新远程执行 kubeadm 重置安装 并用本地已经拉取的镜像使用 kubeadm init 初始化安装 k8s
```

```
安装 helm
```

```
在 k8s 中安装 Harbor 并暴露端口可以通过公网 ip 访问
```

```
在 win 的 wsl 里帮我使用 kind 部署一个 k8s
```

```
远端机器的磁盘满了 我又新增加一个磁盘 可以扩容到之前的磁盘吗
```

```
可以把 /var/lib/containerd 的数据复制出来再将新盘挂载到 /var/lib/containerd 目录吗
```

AI 逐个完成了：K8s 集群重置、Helm 安装、Harbor 私有镜像仓库、本地 Kind 集群、磁盘扩容、containerd 数据迁移。

### 四、编码执行

架构定好之后，每个功能模块按照 Superpowers 流程来：

```
Spec（设计规格）→ Plan（带 TDD 步骤的任务列表）→ Execute（AI 按任务逐个实现并 commit）
```

有一次 coding session 的 git 记录比较典型，大概 96 分钟完成 20 个 commit，平均 5 分钟一个：

```
92d5581 fix(docker): use repo root context for all services, add GOPROXY mirror for China network
41eb10f feat(chart): add Traefik API Gateway with hostPort:80, IngressRoutes for frontend/API/Keycloak
8954dbb feat: implement new API endpoints — delete user/model/session, reset password, model bindings
83d3352 feat: adapt Chat SSE to backend proto StreamEvent format, add markdown rendering
34a5957 fix: sidebar collapse now properly resizes content area with CSS grid auto 1fr
5339923 style: redesign chat UI — bubbles, panel layout, composer, Enter to send
698205f feat: redesign session history as dropdown with search, rename, and delete
37c329d feat: add edit model drawer from ... menu, with right-slide overlay
816517e feat: replace model action buttons with ... dropdown menu
9e4565f feat: add provider filter dropdown to models page for filtering by provider
9c71de4 refactor: replace provider action buttons with ... dropdown menu
927fdca fix: add missing drawerOverlay CSS for right-slide drawer mask
52da8d3 refactor: replace inline create forms with + button and slide drawer in LLM config pages
7bbb816 feat: add edit provider drawer and edit-models navigation in LLM config page
7813577 fix: remove max-width constraint, use fixed positioning for context menu
a472dea feat: add notes column to user table and edit drawer
eb9a88a feat: add edit user drawer — username, role, displayName, email all editable
1357e23 feat: add '...' action menu in user list with cross-tab navigation
c4a7d1a feat: add PermissionsManagement tab for K8s resource access permissions
a2c12c8 feat: merge frontend-mvp worktree — API layer, auth hooks, new pages, nav groups
```

严格按 Plan 的任务顺序走，commit message 用 conventional-commits（feat/fix/refactor/style）。

### 五、Bug 修复和问题沉淀

部署上线后修了不少 bug，我把每个问题都记在 `问题记录.md` 里，一共 24 条：

- **部署配置缺失（8 个）**：Backend 忘了加 AUTH_MODE 和 KEYCLOAK_ISSUER 环境变量、Keycloak Issuer 和前端 Realm 对不上...
- **网关路由冲突（3 个）**：OIDC 回调 `/auth/callback` 被 Keycloak 的 `/auth/` 路由抢走了、没配 `proxy-headers` 导致 issuer 不确定...
- **环境适配（4 个）**：Go 依赖拉不下来要配国内 GOPROXY、`crypto.subtle.digest` 要 HTTPS 安全上下文...
- **启动依赖顺序（2 个）**：MCP Server 启动需要 Backend IdentityService 就绪，缺 initContainers...
- **AI 协作流程（7 个）**：技术选型没提前对齐、代码不够企业级、前端 UX 被忽略等等...

我的习惯是：遇到问题 → AI 分层定位根因 → 出修复方案 → 验证 → 记到 `问题记录.md`。后续新 session 的 AI 会自动加载这些问题记录，不会重复踩坑。

### 六、部署自动化

部署流程封装成了 Skill，说一句话就行：

```
构建镜像并部署
```

AI 的执行链路：

```
→ 加载 k8s-ai-ops-build Skill → 上传源码到云主机
→ 服务器端 Docker build (Go -ldflags="-s -w") → ctr images import
→ 加载 k8s-ai-ops-deploy Skill → 打包 Helm chart 上传
→ helm upgrade --install --wait --timeout 5m → 验证 Pod 状态
```

### 七、文档重构

做了好几轮迭代：

```
扫描现在的代码 然后重构 docs 下的文档。
文档的结构和风格可以参考 deploy-descision-agent 仓库。
```

AI 扫描了全部代码和 Helm Chart，分析了参考仓库的风格，蒸馏出一个 `docs-style-guide` Skill，然后重构了 24 个文档文件（新增 10 个、重写 14 个）。

后面又补了几轮：

```
docs 部署架构相关的 md 文档好像不完整，你根据代码和 deploy 中 chart 的部署结构重新梳理一下
```

```
架构和流程相关的可以补充 mcp-server 会和 backend api 交互获取权限
```

```
核心架构图需要再完善一下
```

三次迭代下来，架构图从 8 个节点扩展到了 6 个子图、14 条通信连线。

---

## 总结一下我摸索出来的方法

### 标准流程

```
我的提示词
  → Brainstorming（AI 逐个提问澄清，一次一个问题）
  → Write Spec（AI 出设计规格文档，含架构图 + 验收标准）
  → Write Plan（AI 把 Spec 拆成编号任务，每个任务带 TDD 三步）
  → Execute Plan（AI 按 Plan 逐个实现，完成一个 commit 一个）
  → Verify（AI 跑测试 / 构建 / Helm template 验证）
  → Deploy（AI 通过 Skills 自动构建镜像 + 部署）
  → 问题反馈循环（bug → 定位 → 修复 → 记到问题记录 → AI 后续自动加载）
```

### 我在每个环节做什么

| 环节 | 我 | AI |
|------|-----|-----|
| 需求 | 写初始需求、追加约束 | 逐个提问澄清、出结构化需求文档 |
| 技术选型 | 指定关键框架（GORM、Eino、Keycloak） | 设计方案、对比选项 |
| 架构设计 | 确认最终方案、安全审查 | 拆分服务、设计协议、画架构图 |
| 编码 | 审查关键逻辑 | 全量编码，按 Plan + TDD 流程走 |
| 测试 | 确认覆盖率要求 | 写单元测试、集成测试 |
| Bug 修复 | 描述现象 | 分层定位 → 出方案 → 修 → 验证 → 记录 |
| 部署 | 给服务器和凭据 | 构建镜像 → 推送 → Helm 部署 → 验证 |
| 文档 | 确认风格方向 | 全量编写、风格统一、交叉链接 |

整体 AI 参与度大概 90%，我主要做决策确认和安全把关。

### 沉淀下来的基础设施

| 能力 | 工具 | 产出 |
|------|------|------|
| 远程操作 | `remote_exec.py`（paramiko SSH） | 构建/推送/部署 Skill 的底层 |
| 构建自动化 | `k8s-ai-ops-build` Skill | 源码上传 → Docker build → ctr import |
| 部署自动化 | `k8s-ai-ops-deploy` Skill | Helm chart 部署，guided/auto 双模式 |
| 知识沉淀 | `问题记录.md`（24 条） | 踩坑就记，后续 AI 自动加载 |
| 流程约束 | `AGENTS.md` | 文档语言、DDD 分层、日志规范、一致性规则 |
| 文档规范 | `docs-style-guide` Skill | 受众定位 → 编号章节 → 已实现/未实现区分 |

### 几个关键经验

1. **先 Spec 再 Plan 再写代码**。一次性把需求、架构、模块边界聊清楚，比边写边改省 token。
2. **技术选型必须提前指定**。AI 不会主动帮你选 GORM、go-redis 这类框架，你不说它就随便写。
3. **AI 容易只关注后端**。前端 UX 流程得单独开 Plan，不然 AI 做完后端 API 就觉得完事了。
4. **非企业级的代码需要约束**。DDD 分层、结构化日志、错误处理规范这些，不明确要求 AI 就给你"能跑就行"的代码。
5. **中国网络环境要显式处理**。Go 依赖、Docker 镜像都得配国内源。
6. **把重复操作用 Skill 封装起来**。构建、部署、推送这些，封装成 Skill 后一句话就能触发。
