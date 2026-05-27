# 系统架构

## 组件视图

```mermaid
flowchart LR
  subgraph Browser["用户浏览器"]
    UI["React Web UI"]
  end

  subgraph Platform["k8s-ai-system namespace"]
    FE["frontend Service"]
    API["backend-api Deployment"]
    Agent["agent-server Deployment"]
    MCP["mcp-server Deployment"]
    KC["Keycloak"]
    PG["PostgreSQL"]
    Redis["Redis"]
  end

  subgraph External["外部服务"]
    OpenAI["OpenAI Compatible API"]
    Anthropic["Anthropic API"]
  end

  subgraph Cluster["Kubernetes 集群"]
    K8S["Kubernetes API Server"]
    RBAC["ServiceAccount / Role / RoleBinding"]
  end

  UI --> FE
  FE --> API
  API --> KC
  API --> PG
  API --> Redis
  API -->|"gRPC AgentService.Run"| Agent
  Agent --> OpenAI
  Agent --> Anthropic
  Agent --> MCP
  MCP --> K8S
  API --> RBAC
  RBAC --> K8S
```

## 运行时调用链

```mermaid
flowchart TD
  Login["用户登录 Keycloak"] --> JWT["获得 JWT"]
  JWT --> APIAuth["Backend 校验 JWT"]
  APIAuth --> LocalUser["映射本地 userId"]
  LocalUser --> Perm["加载业务权限和 LLM 模型"]
  Perm --> Chat["构造 Chat 历史、runtimeContext 和工具 allowlist"]
  Chat --> Agent["gRPC 调用 Agent Server"]
  Agent --> LLM["Eino 调用 LLM"]
  LLM --> Tool{"是否需要工具？"}
  Tool -- "否" --> Answer["返回回答"]
  Tool -- "是" --> Authz["工具调用前授权校验"]
  Authz --> MCP["Agent 调用 MCP Server"]
  MCP --> ClientGo["使用用户 ServiceAccount 创建 client-go"]
  ClientGo --> K8S["访问 Kubernetes API"]
  K8S --> Result["返回工具结果"]
  Result --> LLM
  Answer --> Audit["保存 Chat 和审计日志"]
```

## 部署拓扑

```mermaid
flowchart TB
  subgraph Local["本地开发/演示"]
    Kind["Kind Cluster"]
    Tar["image-tars/*.tar"]
    HelmLocal["Helm install"]
    Tar --> Kind
    Kind --> HelmLocal
  end

  subgraph Existing["已有集群"]
    Registry["镜像仓库"]
    HelmUpgrade["Helm upgrade"]
    Registry --> HelmUpgrade
  end

  HelmLocal --> NS["k8s-ai-system namespace"]
  HelmUpgrade --> NS
  NS --> Pods["frontend / backend-api / agent-server / mcp-server / keycloak / postgresql / redis"]
```

## 公有云部署拓扑

```mermaid
flowchart TD
  User["用户浏览器"] --> Entry["Ingress / LoadBalancer / PortForward"]
  Entry --> FE["Frontend"]

  subgraph CloudK8S["公有云 Kubernetes 集群"]
    subgraph SystemNS["k8s-ai-system namespace"]
      FE
      Backend["Backend API"]
      Agent["Agent Server"]
      MCP["MCP Server"]
      KC["Keycloak"]
      PG["PostgreSQL"]
      Redis["Redis"]
    end

    subgraph BizNS["业务 namespace，例如 dev/test"]
      SA["操作员 ServiceAccount"]
      Role["Role / RoleBinding"]
      Workloads["业务工作负载"]
    end
  end

  FE --> Backend
  Backend --> KC
  Backend --> PG
  Backend --> Redis
  Backend --> Agent
  Agent --> MCP
  Agent --> LLM["外部 LLM Provider"]
  Backend --> Role
  MCP --> SA
  SA --> Workloads
```

公有云测试时，PostgreSQL、Redis、Keycloak 可以先使用 Helm Chart 内置部署；如果云上已有托管服务或统一身份源，可以通过 values 关闭内置组件并接入外部地址。

## 服务职责

| 服务 | 职责 | 不负责 |
| --- | --- | --- |
| Frontend | 页面展示、登录跳转、调用 Backend API | 不保存权限，不直接调用 K8S |
| Backend API | 认证校验、业务授权、用户管理、LLM 管理、Chat 编排、审计 | 不直接暴露 K8S 凭据给前端 |
| Agent Server | 使用 Eino 执行无状态 agent loop，通过 gRPC 接收 Backend 上下文 | 不保存 Chat 历史，不决定业务权限 |
| MCP Server | 将 Kubernetes 能力封装为工具 | 不决定业务权限，不管理用户 |
| Keycloak | 身份认证和平台角色 | 不保存 K8S 资源权限 |
| PostgreSQL | 持久化业务状态 | 不保存明文 token 和 API Key |
| Redis | 短期缓存和流式状态 | 不作为最终数据源 |

## 当前代码实现状态

- Backend 已支持 MemoryStore 和 PostgresStore。
- Backend 已支持 Redis 连通性检查和基础 `PING/SET/GET`。
- Backend 已实现 Kubernetes RBAC Manager，可通过 `client-go` 创建/更新 ServiceAccount、Role、RoleBinding。
- HTTP 权限更新接口已支持在 `K8S_RBAC_SYNC_ENABLED=true` 时调用 RBAC Manager，并按 namespace 分组同步 ServiceAccount、Role、RoleBinding。
- Helm Chart 已改为 namespace 级 RBAC 授权：通过 `rbac.managedNamespaces` 指定 Backend 可管理的目标 namespace，不默认创建 ClusterRole。

## 架构约束

- 操作员只允许 namespace 级权限。
- LLM 不直接访问 Kubernetes。
- MCP Server 只接受 Backend 调用。
- Backend 工具调用前必须做授权校验。
- Kubernetes RBAC 是最终权限边界。
