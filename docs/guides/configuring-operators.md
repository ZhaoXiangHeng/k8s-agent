# 配置操作员

这篇文档面向管理员，说明如何完成"创建操作员 → 分配 K8s 权限 → 绑定 LLM 模型 → 验证"的完整流程。

## 1. 先理解权限模型

在开始配置之前，先简要了解三层权限：

1. **Keycloak JWT**：确认用户是谁，是什么平台角色（admin / operator）
2. **业务权限（PostgreSQL）**：管理员分配给操作员的 namespace 级 K8s 资源权限和 LLM 模型绑定
3. **Kubernetes RBAC**：Backend 根据业务权限动态创建的 ServiceAccount、Role、RoleBinding

三层权限形成一个纵深防线：即使某一层有缺陷，下一层仍会拦截越权访问。

完整权限设计见 [权限模型](../architecture/permission-model.md)。

## 2. 创建操作员

管理员登录 Admin Console，填写操作员信息：

```text
POST /api/admin/users
```

系统行为：

1. Backend 调用 Keycloak Admin API 创建用户并赋予 `operator` 角色
2. Backend 在本地 `users` 表创建映射记录
3. 事件写入审计日志

当前限制：Keycloak Admin API 集成尚未实现，用户创建仅写入本地 `users` 表。

## 3. 分配 K8s 权限

```text
PUT /api/admin/users/:id/permissions
```

请求示例：

```json
{
  "permissions": [
    {
      "namespace": "dev",
      "apiGroup": "",
      "resource": "pods",
      "verbs": ["get", "list", "watch"]
    },
    {
      "namespace": "dev",
      "apiGroup": "apps",
      "resource": "deployments",
      "verbs": ["get", "list", "watch", "patch"]
    }
  ]
}
```

系统行为（`K8S_RBAC_SYNC_ENABLED=true` 时）：

1. Backend 在 PostgreSQL 中保存 `k8s_permissions`
2. Backend 调用 RBAC Manager，按 namespace 分组创建：
   - `ServiceAccount`: `k8s-ai-operator-{userId}`
   - `Role`: `k8s-ai-role-{userId}-{namespace}`
   - `RoleBinding`: `k8s-ai-binding-{userId}-{namespace}`
3. RBAC 同步失败时返回 `K8S_RBAC_APPLY_FAILED` 错误

这些 K8s 对象都带有托管标签 `app.kubernetes.io/managed-by=k8s-ai-ops-backend`。

## 4. 配置 LLM Provider 和模型

### 4.1 创建 Provider

```text
POST /api/admin/llm/providers
```

```json
{
  "name": "OpenAI",
  "protocol": "openai",
  "baseUrl": "https://api.openai.com/v1",
  "apiKey": "sk-...",
  "enabled": true
}
```

`apiKey` 创建后即被加密存储，查询接口不返回明文。

### 4.2 创建 Model

```text
POST /api/admin/llm/models
```

```json
{
  "providerId": "prov-001",
  "modelName": "gpt-4.1",
  "displayName": "GPT-4.1",
  "supportsTools": true,
  "supportsStreaming": true
}
```

### 4.3 绑定模型给操作员

管理员在后台为操作员绑定可用模型并设置默认模型。绑定后，操作员的 `GET /api/operator/llm-models` 只返回已绑定且启用的模型。

## 5. 验证配置

### 5.1 管理员验证

```bash
# 查看操作员权限
curl -H "Authorization: Bearer <admin-jwt>" \
  http://localhost:8080/api/admin/users/:id/permissions

# 查看审计日志
curl -H "Authorization: Bearer <admin-jwt>" \
  "http://localhost:8080/api/admin/audit-logs?actorUserId=:userId"
```

### 5.2 Kubernetes 侧验证

```bash
# 确认 ServiceAccount 已创建
kubectl get sa -n dev k8s-ai-operator-{userId}

# 确认 Role 已创建
kubectl get role -n dev k8s-ai-role-{userId}-dev

# 确认 RoleBinding 已绑定
kubectl get rolebinding -n dev k8s-ai-binding-{userId}-dev

# 模拟操作员权限
kubectl auth can-i list pods -n dev --as=system:serviceaccount:dev:k8s-ai-operator-{userId}
kubectl auth can-i list pods -n prod --as=system:serviceaccount:dev:k8s-ai-operator-{userId}
# 预期：dev 返回 yes，prod 返回 no
```

### 5.3 操作员 Chat 验证

操作员登录后，输入自然语言进行巡检：

```text
帮我看看现在集群里有什么异常吗？
```

预期行为：

1. 系统只巡检操作员授权的 namespace
2. 未授权 namespace 的相关查询会被拒绝
3. 拒绝事件写入审计日志
4. UI 返回 AI 总结和异常 Pod 表格

## 6. 常见问题

### 操作员看不到某个 namespace

按顺序检查：
1. `k8s_permissions` 表中是否有该 namespace 的授权记录
2. ServiceAccount 是否已在目标 namespace 创建
3. RoleBinding 是否正确绑定
4. `K8S_RBAC_SYNC_ENABLED` 是否为 `true`
5. 用 `kubectl auth can-i` 验证实际权限

### 操作员无法使用某个模型

按顺序检查：
1. Provider 是否 `enabled=true`
2. Model 是否 `enabled=true`
3. `user_llm_bindings` 表中是否有该用户与该模型的绑定
4. 申请的 `modelId` 是否属于当前用户

### RBAC 同步失败

- 检查 kubeconfig 是否有目标 namespace 的 ServiceAccount/Role/RoleBinding 创建权限
- 确认 `rbac.managedNamespaces` 包含目标 namespace
- Helm 不会默认创建 ClusterRole，需确保使用了 namespace 级权限
