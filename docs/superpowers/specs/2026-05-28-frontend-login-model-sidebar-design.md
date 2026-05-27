# 前端三大功能改造：登录、模型分配、侧栏收起

**日期**: 2026-05-28
**状态**: 设计中

## 概述

在不引入新依赖（零路由库、零 UI 框架、零图标库、零状态管理库）的前提下，改造现有前端实现三个功能：
1. 登录与密码管理
2. 模型分配双列表穿梭框
3. 侧栏收起/展开

## 现状

- `App.tsx` 147 行，包含所有 UI 和 mock 数据
- 纯 `useState` 控制 `view: "admin"|"operator"` 切换两个视图
- 无登录、无 API 调用、无状态管理
- 后端已有完整 CRUD API（用户、模型、审计），但缺少 binding 端点

## 一、登录与密码管理

### 登录页

- 新增 `LoginPage` 组件：用户名 + 密码表单
- 登录校验逻辑在 `store/auth.ts` 中（纯前端 mock）
- Mock 默认账号：
  - `admin / admin123` → 角色 admin → 显示 Admin Console
  - `operator / operator123` → 角色 operator → 显示 Operator Chat
- 用户名或密码错误 → 表单下方显示红色错误提示
- 登录成功 → 设置 `auth` state，不再显示登录页

### 状态机

```
App 顶层 state:
  auth: "none" | "admin" | "operator"
  view:  "operator" | "admin"      (仅 admin 使用)
  sidebarCollapsed: boolean
```

- `auth === "none"` → 渲染 `<LoginPage />`
- `auth === "operator"` → 渲染 `<OperatorConsole />`
- `auth === "admin"` → 渲染 `<AdminConsole />`（包含子 tab 切换）

### 用户管理

- 用户列表包含：用户名、角色、创建时间、操作列
- 新建用户表单增加**密码**字段（用户名、角色、密码三个必填项）
- 操作列增加**重置密码**按钮
- 点击重置密码 → 右侧滑入抽屉/弹窗，输入新密码并确认
- 后端预留接口（本期不实现，仅 mock）：
  - `POST /api/auth/login`
  - `POST /api/admin/users`
  - `PUT /api/admin/users/:id/password`

## 二、模型分配双列表穿梭框

### 交互流程

```
选择用户 → 左侧(可分配) ←→ 右侧(已分配) → 保存变更
```

1. 顶部下拉框选择目标用户
2. 左侧列表显示该用户**未绑定**的模型（可多选）
3. 右侧列表显示该用户**已绑定**的模型（可多选）
4. 选中左侧 + 点击「分配 >」→ 模型从左侧移到右侧
5. 选中右侧 + 点击「< 回收」→ 模型从右侧移回左侧
6. 右侧模型可点击「设为默认」
7. 底部「保存变更」→ 批量提交所有变更
8. 「取消」→ 撤销所有未保存操作，恢复原始状态

### 状态管理

穿梭框内部维护三个状态：
- `availableModels`: 左侧可分配列表
- `assignedModels`: 右侧已分配列表
- `dirtyModels`: { added: string[], removed: string[], defaultModelId?: string }

保存时只提交 diff（dirtyModels），不提交全量列表。

### 后端绑定 API（需新增）

- `GET /api/admin/users/:userId/bindings` — 获取用户绑定的模型列表
- `PUT /api/admin/users/:userId/bindings` — 批量更新绑定（add + remove + setDefault）
  - Body: `{ add: string[], remove: string[], defaultModelId?: string }`

## 三、侧栏收起/展开

### 行为

- 展开状态（默认）：280px，显示图标 + 文字
- 收起状态：64px，仅显示图标
- 切换按钮在侧栏底部
- 收起时鼠标悬浮图标 → tooltip 显示菜单文字
- 内容区域 CSS Grid `1fr` 自动填充剩余空间

### 实现

- 侧栏组件从 App.tsx 提取为 `components/Sidebar.tsx`
- 宽度由 `sidebarCollapsed` state 控制
- CSS 过渡动画：`transition: width 0.2s ease`
- tooltip 用纯 CSS（`:hover` 伪类 + `::after` 伪元素），不引入外部库
- 图标初期使用 emoji/CSS 字符，后续替换为 lucide-react

### 侧栏菜单项

| 图标 | 名称 | 路由(view) | 谁可见 |
|------|------|-----------|--------|
| 💬 | Chat 运维 | operator | 所有人 |
| 👥 | 用户与权限 | admin | 仅 admin |
| 🔗 | 模型分配 | admin | 仅 admin |

## 文件拆分

```
frontend/src/
├── App.tsx                    # 顶层状态 + 布局壳 (~80行)
├── main.tsx                   # 入口 (不变)
├── styles.css                 # 全局样式 (追加穿梭框/侧栏/登录)
├── store/
│   └── auth.ts                # Mock 用户数据 + 登录/CRUD 逻辑
├── pages/
│   ├── LoginPage.tsx          # 登录表单
│   ├── OperatorConsole.tsx    # Chat + Pod 表 (从 App.tsx 抽出)
│   ├── AdminConsole.tsx       # Admin 主页：tab 切换 + 子页路由
│   ├── UserManagement.tsx     # 用户 CRUD + 密码重置抽屉
│   └── ModelAssignment.tsx    # 双列表穿梭框
└── components/
    └── Sidebar.tsx            # 侧栏组件：logo + nav + 收展开关
```

## 不变约束

- 不引入 react-router-dom
- 不引入 UI 框架（MUI/Ant Design/等）
- 不引入图标库（lucide-react/等）
- 不引入状态管理库（Redux/Zustand/等）
- 所有 mock 数据在前端 store/auth.ts 中以 JavaScript 对象维护
- API 调用暂不实现，预留接口注释
