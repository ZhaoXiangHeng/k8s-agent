---
name: core-agent-image-push
description: 用于在 `D:\code\v9_work\Anybackup\Agent\service\core_agent` 目录中通过 WSL 自动生成 `core-agent-service:timestamp` 镜像 tag，并将对应镜像传输到 `124.174.9.249` 后使用 `ctr` 删除同 tag 旧镜像并导入新镜像。 当用户要求“构建镜像并推送到 124.174.9.249”、“把本地镜像导入远端 CRI”、“用 crictl 上传 core-agent-service 镜像”或执行同类部署动作时使用。
---

# Core Agent Image Push

## 概述

这个技能把核心智能体服务的最小镜像构建和远端导入流程固定为一条可重复执行的链路。
源码与 Docker build context 固定使用 `D:\code\v9_work\Anybackup\Agent\service\core_agent`。
默认目标是 `124.174.9.249`，默认会生成 `core-agent-service:yyyyMMdd-HHmmss` 形式的镜像 tag，并基于该 tag 生成本地导出 tar 文件名。

## 工作流

1. 进入 `D:\code\v9_work\Anybackup\Agent\service\core_agent`
2. 通过 WSL 执行 `docker build -t core-agent-service:timestamp .`
3. 通过 WSL 执行 `docker save`，把同 tag 镜像导出为 tar 文件
4. 通过 `scp` 或 `pscp` 把 tar 文件传到 `124.174.9.249`
5. 通过 `ssh` 或 `plink` 连接远端主机
6. 如果远端已存在同一个 tag 的镜像，先执行删除
7. 执行 `ctr -n k8s.io images import` 导入新镜像
8. 远端通过 `ctr -n k8s.io images list` 打印镜像查询结果，确认导入后的同 tag 镜像可见
9. 删除远端和本地的临时 tar 文件

## 使用方式

优先运行脚本：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\push-core-agent-image.ps1
```

如果只想先检查将要执行的命令，不实际构建或传输，使用：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\push-core-agent-image.ps1 -DryRun
```

## 参数约定

- 默认仓库路径：`D:\code\v9_work\Anybackup\Agent\service\core_agent`
- 默认镜像标签：自动生成 `core-agent-service:yyyyMMdd-HHmmss`
- 默认远端主机：`124.174.9.249`
- 默认远端用户：`root`
- 默认远端密码：`eisoo.com123`
- 默认远端 tar 路径：`/tmp/core-agent-service-image.tar`

如果需要覆盖默认值，也可以在执行脚本时显式传入 `-ImageTag`。

## 连接策略

- 优先使用 `pscp` 和 `plink`，因为它们支持通过密码参数完成非交互式传输与远程执行。
- 如果本机没有 `pscp/plink`，脚本会回退到系统自带的 `scp/ssh`。
- 回退到 `scp/ssh` 时，密码通常需要在终端中手动输入；如果已经配置免密登录，则可以直接执行。
- 当前受控环境通常没有 `pscp/plink`，需要全自动执行时可使用 Python `paramiko` 按默认主机、用户和密码上传 tar 并执行远端 `ctr` 导入。
- 如果 `wsl` 在受限环境中返回访问拒绝，应在用户终端或获批后的高权限会话中重试。
- 远端删旧镜像采用 `ctr -n k8s.io images rm` 容错删除，不再依赖额外的预检查命令。
- 导入完成后会打印远端镜像查询结果，便于快速验收。

## 资源

- 主脚本：`scripts/push-core-agent-image.ps1`
- 参考说明：`references/deploy-flow.md`

## 执行要求

- 程序日志、异常文本和运行时输出保持英文
- 说明文字与注释保持中文
- 不在业务仓库里引入额外部署语义，只固化镜像构建和导入动作
