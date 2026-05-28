# 交付流程

这篇文档说明开发者如何把当前 k8s-agent 从本地代码开发推进到可部署、可验证的交付状态。

## 1. 当前交付目标

不是单纯"把代码写完"，而是形成一条完整链路：

1. 本地开发 + 测试通过
2. 构建镜像（四个服务）
3. 推送镜像到目标节点（或镜像仓库）
4. Helm 部署
5. 端到端验证

## 2. 当前交付物组成

交付面由四部分组成：

### 2.1 四个服务镜像

| 镜像 | Dockerfile | Go 构建参数 |
|------|------------|-------------|
| `backend-api` | `backend/Dockerfile` | `-ldflags="-s -w"` |
| `agent-server` | `agent-server/Dockerfile` | `-ldflags="-s -w"` |
| `mcp-server` | `mcp-server/Dockerfile` | `-ldflags="-s -w"` |
| `frontend` | `frontend/Dockerfile` | Vite 构建 + nginx |

Go 服务构建上下文为仓库根目录（agent-server 需要访问 `proto/`），`-ldflags="-s -w"` 自动去除调试符号。

### 2.2 Helm Chart

```text
deploy/helm/k8s-ai-ops/
├── Chart.yaml
├── values.yaml
├── values-local.yaml
├── values-prod-example.yaml
└── templates/
    ├── namespace.yaml
    ├── backend.yaml
    ├── agent-server.yaml
    ├── frontend.yaml
    ├── mcp-server.yaml
    ├── keycloak.yaml
    ├── postgresql.yaml
    ├── redis.yaml
    ├── rbac.yaml
    ├── secret.yaml
    ├── traefik.yaml
    ├── ingressroute.yaml
    └── keycloak-realm-configmap.yaml
```

### 2.3 部署脚本

| 脚本 | 用途 |
|------|------|
| `scripts/build-images.sh` | 本地构建 4 个服务镜像 + 导出 tar |
| `scripts/bootstrap-local.sh` | 创建 Kind 集群 + 加载镜像 + Helm 安装 |
| `scripts/helm-install.sh` | Helm-only 安装（已有集群） |
| `scripts/helm-upgrade.sh` | Helm-only 升级 |
| `scripts/uninstall.sh` | 卸载 Helm release |
| `scripts/dev-infra-wsl.sh` | 启动 WSL Docker 中的 PostgreSQL/Redis |
| `scripts/build-on-server.sh` | 在远端服务器上构建 |

### 2.4 文档

- 使用者文档：`docs/guides/`、`docs/reference/`
- 开发者文档：`docs/developer/`
- 架构与安全：`docs/architecture/`、`docs/security/`
- 历史设计与计划：`docs/superpowers/`

## 3. 本地开发阶段流程

日常迭代建议按这个顺序：

1. 修改代码
2. 跑测试：
   ```bash
   cd backend && go test ./...
   cd agent-server && go test ./...
   cd mcp-server && go test ./...
   cd proto && go test ./...
   cd frontend && npm run build
   ```
3. 验证 Shell 脚本语法：
   ```bash
   bash -n scripts/bootstrap-local.sh scripts/helm-install.sh scripts/helm-upgrade.sh scripts/uninstall.sh scripts/build-images.sh
   ```
4. 如有功能变更，同步更新对应文档

## 4. 构建阶段

### 4.1 本地构建

```bash
scripts/build-images.sh --tag local --output-dir image-tars
```

生成：

```text
image-tars/backend-api-amd64.tar
image-tars/agent-server-amd64.tar
image-tars/mcp-server-amd64.tar
image-tars/frontend-amd64.tar
```

### 4.2 远端构建（用于推到 120.55.84.39）

```bash
python scripts/remote_exec.py exec "bash /tmp/k8s-agent-build/scripts/build-on-server.sh --tag local"
```

远端构建流程：上传源码 tar.gz → 远端解压 → Docker build → `docker save` → `ctr -n k8s.io images import` 导入 containerd。

## 5. 部署阶段

### 5.1 本地 Kind 集群（从零启动）

```bash
scripts/bootstrap-local.sh \
  --image-source tar \
  --image-dir image-tars \
  --cluster-name k8s-ai
```

脚本职责：
- 检查 docker、kind、kubectl、helm
- 创建 Kind 集群
- 创建 dev、test 演示 namespace
- 加载本地 tar 镜像包
- 调用 Helm 安装系统

### 5.2 已有集群首次部署

```bash
scripts/helm-install.sh \
  --image-source tar \
  --image-dir image-tars \
  --values deploy/helm/k8s-ai-ops/values-local.yaml
```

### 5.3 已有集群升级（镜像仓库模式）

```bash
scripts/helm-upgrade.sh \
  --image-source registry \
  --registry registry.example.com/k8s-ai \
  --tag v1.0.0 \
  --values deploy/helm/k8s-ai-ops/values-prod-example.yaml
```

## 6. 验证阶段

### 6.1 部署后验证

```bash
# Pod 状态
kubectl get pods -n k8s-ai-system

# 服务状态
kubectl get svc -n k8s-ai-system

# 检查各服务日志
kubectl logs -n k8s-ai-system deploy/backend-api
kubectl logs -n k8s-ai-system deploy/agent-server
kubectl logs -n k8s-ai-system deploy/mcp-server
kubectl logs -n k8s-ai-system deploy/frontend
```

### 6.2 访问验证

```bash
kubectl port-forward -n k8s-ai-system svc/frontend 8088:80
kubectl port-forward -n k8s-ai-system svc/keycloak 8089:8080
```

### 6.3 RBAC 验证

```bash
# 确认 Backend 可以在 managedNamespaces 内创建 RBAC 对象
kubectl auth can-i create serviceaccounts -n dev --as=system:serviceaccount:k8s-ai-system:backend-api
kubectl auth can-i create roles -n dev --as=system:serviceaccount:k8s-ai-system:backend-api
kubectl auth can-i create rolebindings -n dev --as=system:serviceaccount:k8s-ai-system:backend-api

# 确认 Backend 不能在未授权 namespace 创建 RBAC 对象
kubectl auth can-i create serviceaccounts -n kube-system --as=system:serviceaccount:k8s-ai-system:backend-api
# 预期：no
```

### 6.4 业务验证

1. 管理员登录 → 创建操作员 → 分配 namespace 权限 → 绑定 LLM 模型
2. 操作员登录 → 发起 Chat 巡检 → 确认只看到授权 namespace 的资源
3. 尝试越权访问 → 确认被拒绝 → 审计日志记录 denied

## 7. 发布前检查清单

每次准备交付前，建议至少检查：

- [ ] 所有服务 `go test ./...` 通过
- [ ] Frontend `npm run build` 通过
- [ ] Shell 脚本语法检查通过
- [ ] `helm template` 无错误
- [ ] 镜像能成功构建
- [ ] 相关文档已更新
- [ ] 如果有 proto 变更，gRPC 代码已重新生成
- [ ] 如果有 API 变更，api-design.md 已更新
- [ ] 如果有配置变更，config-reference.md 已更新
- [ ] 如果涉及安全变更，security-design.md 已更新

## 8. 当前交付流程的边界

当前系统已经能完成"代码 → 镜像 → 部署 → 验证"这条链路，但还没有完全产品化：

- 没有 CI/CD 流水线驱动的自动化构建
- 没有镜像仓库版本管理策略
- 没有 Helm Chart 版本发布流程
- 没有生产环境的自动化验收测试
- 没有自动化的文档与代码一致性检查

## 9. 后续建议

如果继续把交付流程做强，建议优先考虑：

1. CI 中自动化构建 + Helm template 验证
2. 镜像 tag 版本管理 + changelog
3. 文档与 CLI 输出一致性自动检查
4. 端到端冒烟测试（真实 Kind 集群）
