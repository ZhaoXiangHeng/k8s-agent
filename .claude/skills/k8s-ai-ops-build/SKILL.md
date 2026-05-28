---
name: k8s-ai-ops-build
description: 用于将 `e:\k8s-agent` 仓库源码上传到远端服务器 `120.55.84.39` 并通过服务器端 Docker 构建全部 4 个服务镜像（mcp-server、agent-server、backend、frontend），Go 服务构建时自动移除调试符号（-ldflags="-s -w"），构建完成后直接导入到 containerd 运行时。当用户要求"构建镜像"、"docker build"或执行同类构建动作时使用。
---

# K8s AI Ops - Build Images

## 概述

将本地源码打包上传到远端 K8s 节点（120.55.84.39），在服务器端使用 Docker 构建全部 4 个服务镜像，Go 服务自动 strip debug symbols，构建完成后直接导入到 containerd 的 `k8s.io` namespace。

## 工作流

1. 在本地将仓库打包为 tar.gz（排除 `.git`、`node_modules`、`image-tars` 等不需要的目录）
2. 通过 `remote_exec.py upload` 上传 tar.gz 到远端 `/tmp/`
3. 远端解压到 `/tmp/k8s-agent-build/`
4. 按依赖顺序构建：mcp-server → agent-server → backend → frontend
   - Go 服务注入 `--build-arg LDFLAGS="-s -w"` 减小二进制体积
   - 构建上下文为 `/tmp/k8s-agent-build/`（agent-server 需要访问 `proto/`）
5. 每个服务构建完成后 `docker save` 并通过 `ctr -n k8s.io images import` 导入 containerd
6. 验证导入结果：`ctr -n k8s.io images list | grep k8s-ai`
7. 清理远端临时文件

## 使用方式

构建全部 4 个服务（默认 tag=local）：

```bash
python scripts/remote_exec.py exec "bash /tmp/k8s-agent-build/scripts/build-on-server.sh --tag local"
```

指定 tag：

```bash
python scripts/remote_exec.py exec "bash /tmp/k8s-agent-build/scripts/build-on-server.sh --tag v1.0.0"
```

构建部分服务：

```bash
python scripts/remote_exec.py exec "bash /tmp/k8s-agent-build/scripts/build-on-server.sh --tag local --services backend,frontend"
```

## 参数约定

- 默认构建全部 4 个服务
- 默认 tag 为 `local`
- Go 服务默认 `LDFLAGS="-s -w"`
- 构建上下文为仓库根目录（agent-server 需要访问 `proto/`）
- 远端构建目录：`/tmp/k8s-agent-build/`

## 依赖

- 本地：Python 3 + paramiko（`scripts/remote_exec.py`）
- 远端：Docker daemon、containerd（`ctr` 命令）

## 资源

- 远程执行工具：`scripts/remote_exec.py`
- 远端构建脚本：`scripts/build-on-server.sh`
- 说明文档：`docs/superpowers/specs/2026-05-27-k8s-ai-ops-deploy-automation-design.md`
