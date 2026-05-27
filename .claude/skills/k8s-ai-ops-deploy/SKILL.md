---
name: k8s-ai-ops-deploy
description: 用于将 `deploy/helm/k8s-ai-ops/` Helm Chart 部署到 `120.55.84.39` 的 K8s 集群。支持引导式（guided）和非引导式（auto）两种模式。引导式逐个询问配置项，非引导式通过参数一键部署（供 AI 调用）。当用户要求"部署"、"helm install"、"deploy chart"、"部署到 K8s"时使用。
---

# K8s AI Ops - Deploy Chart

## 概述

将本地 Helm Chart 打包上传到远端 K8s 节点，通过 `helm upgrade --install` 部署或更新 k8s-ai-ops 全部服务。

## 工作流

1. 收集配置参数（guided 模式交互式询问，auto 模式从参数/环境变量读取）
2. 生成临时 `values-override.yaml`
3. `tar` 打包 `deploy/helm/k8s-ai-ops/` （排除 .git）
4. 通过 `remote_exec.py upload` 上传 chart 包和 values 到远端
5. 远端解压并执行 `helm upgrade --install`
6. `--wait --timeout 5m` 等待 rollout 完成
7. 输出 pod 状态
8. 清理本地和远端临时文件

## 使用方式

### 引导式（默认）

逐步询问 namespace、tag、Keycloak、数据库、Redis 等配置：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy-chart.ps1
```

### 非引导式（AI 调用）

通过参数一键部署，无需交互：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy-chart.ps1 -Mode auto `
  -Tag "20260527-143000" `
  -Namespace "k8s-ai-system" `
  -KeycloakEnabled "true" `
  -AuthMode "dev"
```

预览不执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy-chart.ps1 -DryRun
```

## 参数约定

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-Mode` | guided | `guided` 或 `auto` |
| `-Tag` | yyyyMMdd-HHmmss | 镜像 tag |
| `-Namespace` | k8s-ai-system | K8s namespace |
| `-ReleaseName` | k8s-ai-ops | Helm release 名称 |
| `-KeycloakEnabled` | true | 是否启用 Keycloak |
| `-AuthMode` | dev | 认证模式 |
| `-StoreDriver` | postgres | 存储驱动 |
| `-CacheDriver` | redis | 缓存驱动 |
| `-RbacSyncEnabled` | true | RBAC 同步 |

## 连接策略

- 使用 Python `remote_exec.py`（paramiko）上传 chart 和执行 helm
- 连接信息通过环境变量传递

## 依赖

- Python 3 + paramiko
- `scripts/remote_exec.py`
- 远端需具备 `helm`、`kubectl`

## 资源

- 主脚本：`scripts/deploy-chart.ps1`
- 远程执行工具：`scripts/remote_exec.py`
- Helm Chart：`deploy/helm/k8s-ai-ops/`
- 说明文档：`docs/superpowers/specs/2026-05-27-k8s-ai-ops-deploy-automation-design.md`
