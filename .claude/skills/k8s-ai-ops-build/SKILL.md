---
name: k8s-ai-ops-build
description: 用于在 `e:\k8s-agent` 仓库中通过 WSL Docker 构建全部 4 个服务镜像（mcp-server、agent-server、backend、frontend），Go 服务构建时自动移除调试符号（-ldflags="-s -w"），并将镜像导出为 tar 包到 `image-tars/` 目录。当用户要求"构建镜像"、"编译并导出 tar"、"docker build"或执行同类构建动作时使用。
---

# K8s AI Ops - Build Images

## 概述

在 Windows 端通过 WSL 调用 Docker，构建 k8s-ai-ops 全部 4 个服务镜像，Go 服务自动 strip debug symbols，最终导出为 containerd 可直接导入的 tar 包。

## 工作流

1. 检查 WSL 和 Docker 是否可用
2. 按依赖顺序构建：mcp-server → agent-server → backend → frontend
3. Go 服务注入 `--build-arg LDFLAGS="-s -w"` 减小二进制体积
4. 每个服务 `docker save` 导出 tar 到 `image-tars/` 目录
5. 打印构建摘要（每个 tar 的大小）

## 使用方式

优先运行脚本：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-images.ps1
```

构建部分服务：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-images.ps1 -Services "backend,frontend"
```

自定义 tag：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-images.ps1 -Tag "v1.0.0"
```

只预览不执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-images.ps1 -DryRun
```

## 参数约定

- 默认构建全部 4 个服务
- 默认 tag 自动生成为 `yyyyMMdd-HHmmss`
- 产物输出到 `image-tars/` 目录
- Go 服务默认 `LDFLAGS="-s -w"`
- 构建上下文为仓库根目录（agent-server 需要访问 `proto/`）

## 依赖

- Windows WSL（需安装 Docker）
- WSL 内 Docker daemon 需运行中

## 资源

- 主脚本：`scripts/build-images.ps1`
- 说明文档：`docs/superpowers/specs/2026-05-27-k8s-ai-ops-deploy-automation-design.md`
