# Frontend 完整 MVP 设计

## 背景

当前 Frontend 只有单文件静态演示页面，尚未接入 Backend API。Backend 已提供完整 MVP 所需接口，包括当前用户、操作员权限、操作员可用模型、Chat 会话与消息、管理员用户管理、权限配置、LLM Provider/Model 管理和审计日志查询。

本设计目标是在现有 React + Vite 项目上实现完整 MVP 前端，使系统可以演示“管理员配置平台能力”和“操作员通过 Chat 巡检 Kubernetes 异常资源”的核心闭环。

## 目标

- 实现多页面 TO B 控制台体验。
- 对接 Backend 已实现的真实 HTTP API 和 Chat SSE 流。
- 实现 Keycloak 前端登录流程，在生产认证模式下获取 Access Token 并随 API 请求发送。
- 按前端 DDD 风格拆分领域类型、应用用例、基础设施 API 客户端和界面组件。
- 支持管理员完整 MVP 操作：用户创建、权限编辑、Provider 创建/更新、Model 创建/更新、审计查看。
- 支持操作员完整 MVP 操作：查看权限、选择授权模型、创建 Chat 会话、发送自然语言运维指令、展示 AI 总结和结构化资源结果。
- 保持文档、命令、接口路径和字段名与现有 Backend 契约一致。

## 非目标

- 不新增全局状态管理库。
- 不引入大型 UI 组件库。
- 不实现复杂路由权限守卫；本阶段使用内置导航和角色可见性控制。
- 不修改 Backend API 契约。
- 不实现 Keycloak Admin API 用户同步；管理员创建用户仍调用现有 Backend 用户接口。

## 用户体验设计

整体采用 TO B 运维控制台风格：

- 左侧深色导航栏展示产品名、角色分组和页面入口。
- 顶部工具栏展示当前用户、角色、数据刷新入口和请求状态。
- 主内容区使用紧凑表格、筛选栏、表单面板、状态标签、空态和错误提示。
- 页面色彩克制，强调可读性和密度，避免营销式 hero、装饰性大卡片和单一色系。
- 移动端保留可用性，导航和表格通过响应式布局折叠或横向滚动。

页面划分：

- 认证：
  - `登录页`：展示产品名、Keycloak 登录按钮、开发模式演示登录入口和登录错误。
  - `认证回调`：处理 Keycloak Authorization Code + PKCE 回调，交换 Token，保存会话后进入控制台。
  - `退出登录`：清理本地会话，并在 Keycloak 模式下跳转到 Keycloak logout endpoint。
- 操作员：
  - `Chat 运维`：模型选择、消息列表、输入框、流式响应状态、资源结果表。
  - `我的权限`：展示 namespace、apiGroup、resource、verbs 和启用状态。
  - `可用模型`：展示授权模型、Provider、工具调用和流式能力。
- 管理员：
  - `用户管理`：用户列表和创建用户表单。
  - `权限配置`：选择用户，编辑权限项并提交到 `/api/admin/users/:userId/permissions`。
  - `LLM Provider`：Provider 列表、创建和更新表单，隐藏 API Key 明文。
  - `LLM Model`：Model 列表、创建和更新表单。
  - `审计日志`：展示审计动作、操作者、目标、命名空间、资源、动词、是否允许、原因和时间。

## 前端分层设计

Frontend 采用轻量 DDD 风格，目录建议如下：

```text
frontend/src/
  config.ts
  domain/
    auth.ts
    audit.ts
    chat.ts
    llm.ts
    permission.ts
    user.ts
  infrastructure/
    api/
      client.ts
      authApi.ts
      auditApi.ts
      chatApi.ts
      llmApi.ts
      permissionApi.ts
      userApi.ts
  application/
    useAdminData.ts
    useAuth.ts
    useChatOps.ts
    useOperatorData.ts
  interfaces/
    components/
    layout/
    pages/
  App.tsx
  main.tsx
  styles.css
```

分层职责：

- `domain`：只定义业务类型、枚举和值转换，不依赖 React 和浏览器 API。
- `infrastructure/api`：封装 `fetch`、错误解析、JSON 请求、SSE 解析、Backend 路径和认证 header 注入。
- `application`：组合 API 调用和页面状态，暴露用例级 hook。
- `interfaces`：负责 React 组件、表单、表格、导航和视觉样式。

依赖方向：

```text
interfaces -> application -> infrastructure/api -> domain
application -> domain
interfaces -> domain
```

其中 `domain` 不依赖外层。`infrastructure/api` 依赖领域类型来描述输入输出，但不包含页面状态。

## 认证设计

Frontend 支持两种认证模式：

- `dev`：本地演示模式。前端不跳转 Keycloak，请求 API 时可附带 `X-Demo-User` 和 `X-Demo-Role`，与 Backend `AUTH_MODE=dev` 对齐。
- `keycloak`：生产模式。前端使用 Keycloak OIDC Authorization Code + PKCE 登录，拿到 `access_token` 后通过 `Authorization: Bearer <token>` 调用 Backend，与 Backend `AUTH_MODE=jwt` 对齐。

认证配置通过 Vite 环境变量注入：

```text
VITE_AUTH_MODE=dev|keycloak
VITE_KEYCLOAK_URL=http://localhost:8089
VITE_KEYCLOAK_REALM=k8s-ai
VITE_KEYCLOAK_CLIENT_ID=k8s-ai-frontend
VITE_KEYCLOAK_REDIRECT_URI=http://localhost:5173/auth/callback
```

Keycloak 登录流程：

1. 用户点击登录。
2. 前端生成 `code_verifier`、`code_challenge` 和 `state`，暂存到 `sessionStorage`。
3. 浏览器跳转到 Keycloak authorization endpoint。
4. Keycloak 回调 `/auth/callback?code=...&state=...`。
5. 前端校验 `state`，使用 `code_verifier` 调用 token endpoint。
6. 前端保存 `access_token`、`refresh_token`、过期时间和基础用户声明。
7. 前端调用 `GET /api/me` 获取 Backend 认可的当前用户与角色。

Token 存储策略：

- 本阶段使用 `sessionStorage` 保存 Token，刷新页面仍保留当前标签页会话。
- 不把 Token 写入日志、审计、URL 持久化参数或 UI 明文。
- 请求前如 Access Token 接近过期，优先使用 Refresh Token 刷新；刷新失败则回到登录页。
- 登出时清理本地 Token，并在 Keycloak 模式下跳转到 logout endpoint。

## API 对接

统一请求客户端：

- 基础路径默认为 `/api`。
- `keycloak` 模式下所有 API 请求附带 `Authorization: Bearer <access_token>`。
- `dev` 模式下可附带 `X-Demo-User` 和 `X-Demo-Role`，默认使用操作员演示身份。
- HTTP 错误解析 Backend 统一错误结构：

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid request body.",
    "requestId": "..."
  }
}
```

对接接口：

- `GET /api/me`
- `GET /api/operator/permissions`
- `GET /api/operator/llm-models`
- `POST /api/operator/chat/sessions`
- `POST /api/operator/chat/sessions/:sessionId/messages`
- `GET /api/admin/users`
- `POST /api/admin/users`
- `PUT /api/admin/users/:userId/permissions`
- `GET /api/admin/llm/providers`
- `POST /api/admin/llm/providers`
- `PUT /api/admin/llm/providers/:id`
- `GET /api/admin/llm/models`
- `POST /api/admin/llm/models`
- `PUT /api/admin/llm/models/:id`
- `GET /api/admin/audit-logs`

Chat 消息接口按 SSE 处理。前端发送 `modelId` 和 `content` 后，从响应流逐行解析 `data: ...`，兼容以下事件形态：

- 包含 `summary` 和 `resources` 的最终业务结果。
- 包含 `error` 的错误事件。
- Agent 透传的中间 JSON 事件；前端保留原始事件摘要用于调试展示。

## 状态与错误处理

- 未登录或 `GET /api/me` 返回 `UNAUTHENTICATED` 时进入登录页。
- 当前用户角色为 `admin` 时展示管理员和操作员页面；角色为 `operator` 时只展示操作员页面。
- 页面级数据加载使用 `loading`、`error`、`data` 三态。
- 表单提交时禁用提交按钮，成功后刷新对应列表。
- 错误提示展示 Backend `error.message`，并在可用时展示 `requestId`。
- Chat SSE 中途失败时将错误作为 assistant 消息展示，并保留用户输入。
- Provider 的 `apiKey` 仅在创建或更新时输入，不在列表中展示明文。
- 空列表使用明确空态，提示下一步操作。

## 表单设计

用户创建表单字段：

- `username`
- `displayName`
- `email`
- `role`

权限编辑表单字段：

- `namespace`
- `apiGroup`
- `resource`
- `verbs`

Provider 表单字段：

- `name`
- `protocol`
- `baseUrl`
- `apiKey`
- `enabled`

Model 表单字段：

- `providerId`
- `modelName`
- `displayName`
- `supportsTools`
- `supportsStreaming`
- `enabled`

所有新增代码注释如有必要必须使用中文。普通 UI 文案使用中文，接口字段、命令和协议名保持英文。

## 测试策略

先补前端测试工具链，再以 TDD 方式实现关键逻辑：

- Auth 客户端能生成 PKCE 参数、构造 Keycloak 授权 URL，并校验回调 `state`。
- API 客户端能正确解析成功响应和 Backend 错误响应。
- API 客户端能在 Keycloak 模式下注入 `Authorization` header，在 dev 模式下注入演示用户 header。
- SSE 解析器能解析 `data: ...` 行、忽略空行，并识别错误事件。
- 权限表单能组装 Backend 需要的 `permissions` payload。
- Chat 用例能创建会话、发送消息并追加 assistant 结果。

构建验证：

```bash
cd frontend
npm run build
```

如新增测试命令，应同步更新 `frontend/package.json` 和 `README.md` 的本地验证命令。

## 文档影响

本次实现会影响 Frontend 当前实现状态，应同步检查并更新：

- `README.md`
- `docs/INDEX.md`
- `docs/product/requirements.md`
- `docs/reference/api-design.md`
- `docs/developer/developer-guide.md`
- `docs/operations/deployment-guide.md`

如果接口路径或字段不变，仅需更新“当前实现状态”、前端验证说明和 Keycloak 前端配置说明。

## 验收标准

- Frontend 构建通过。
- Keycloak 模式下可以完成登录、回调、Token 保存、调用 `/api/me` 和退出登录。
- Dev 模式下可以使用演示身份进入控制台。
- 页面可以展示当前用户和角色。
- 操作员可以查看自己的权限和可用模型。
- 操作员可以创建 Chat 会话，发送消息，并看到流式或最终响应。
- 管理员可以创建用户。
- 管理员可以为用户配置 namespace 级 Kubernetes 权限。
- 管理员可以创建和更新 LLM Provider。
- 管理员可以创建和更新 LLM Model。
- 管理员可以查看审计日志。
- 页面风格符合 TO B 产品：信息密度合理、层级清晰、表格和表单易扫描。
- 代码按 `domain`、`application`、`infrastructure`、`interfaces` 分层组织。
