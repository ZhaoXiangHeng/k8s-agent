---
name: k8s-ai-ops-push
description: 用于扫描 `image-tars/` 目录中的镜像 tar 包，通过 `remote_exec.py` 上传到 `120.55.84.39` 并通过 `ctr -n k8s.io images import` 导入到 containerd 运行时。当用户要求"推送镜像"、"导入 containerd"、"把镜像传到 120.55.84.39"或执行同类部署动作时使用。
---

# K8s AI Ops - Push Images

## 概述

将 `build-images.ps1` 构建的 tar 包推送到远程 K8s 节点的 containerd 运行时中，使镜像在 `k8s.io` namespace 下可用。

## 工作流

1. 扫描 `image-tars/` 匹配指定 tag 的 tar 文件
2. 通过 `remote_exec.py upload` 逐服务上传 tar 到 `/tmp/`
3. 远端 `ctr -n k8s.io images rm` 容错删除旧镜像
4. 远端 `ctr -n k8s.io images import` 导入新镜像
5. 远端 `ctr -n k8s.io images list | grep` 验证导入结果
6. 清理远端临时 tar 文件

## 使用方式

推送所有服务（自动匹配最新 tag）：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\push-images.ps1
```

指定服务和 tag：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\push-images.ps1 -Services "backend,frontend" -Tag "20260527-143000"
```

## 参数约定

- 默认目标主机：`120.55.84.39`
- 默认用户：`root`
- 默认密码：从环境变量 `REMOTE_PASSWORD` 读取
- 默认 containerd namespace：`k8s.io`
- 远端临时路径：`/tmp/`

## 连接策略

- 使用 Python `remote_exec.py`（paramiko）进行非交互式密码认证
- 连接信息通过环境变量传递：`REMOTE_HOST`、`REMOTE_USER`、`REMOTE_PASSWORD`

## 依赖

- Python 3 + paramiko
- `scripts/remote_exec.py`

## 资源

- 主脚本：`scripts/push-images.ps1`
- 远程执行工具：`scripts/remote_exec.py`
- 说明文档：`docs/superpowers/specs/2026-05-27-k8s-ai-ops-deploy-automation-design.md`
