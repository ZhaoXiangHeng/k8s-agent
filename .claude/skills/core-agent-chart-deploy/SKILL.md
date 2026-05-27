---
name: core-agent-chart-deploy
description: 用于把 `D:\code\v9_work\core_agent_service_chart` 打包传到 `124.174.9.249`，并在远端执行 `bash scripts/deploy.sh` 安装 Helm Chart。 当用户要求“安装 core_agent_service_chart”、“在 124.174.9.249 部署 chart”、“执行 deploy.sh 安装 core-agent-service chart”或要求按需覆写 `--image`、`--namespace`、`--database-url`、`--rabbitmq-url`、`--kweaver-base-url`、`--kweaver-decision-agent-id` 等参数时使用。
---

# Core Agent Chart Deploy

## 概述

这个技能把 `core_agent_service_chart` 的最小部署流程固定为一条可重复执行的链路。
默认会把本地 Chart 打包后传到 `124.174.9.249` 的临时目录，再在远端执行 `scripts/deploy.sh`。

## 工作流

1. 读取本地 `D:\code\v9_work\core_agent_service_chart`
2. 打包为临时 `tar.gz`
3. 通过 `scp` 或 `pscp` 传到 `124.174.9.249`
4. 远端解压到临时目录
5. 远端执行 `bash scripts/deploy.sh`
6. 把用户显式提供的参数动态拼到安装命令后
7. 安装完成后清理系统临时目录中的本地压缩包和远端临时文件

## 使用方式

直接运行脚本：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy-core-agent-chart.ps1
```

如果要覆写部分安装参数，例如镜像和命名空间：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy-core-agent-chart.ps1 `
  -Image 'core-agent-service:20260424-170127' `
  -Namespace 'anybackup-ai'
```

如果要显式使用 KWeaver 用户名密码鉴权，可额外传入：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy-core-agent-chart.ps1 `
  -Image 'core-agent-service:20260424-170127' `
  -KWeaverUsername 'service_account' `
  -KWeaverPassword '***'
```

如果只想先检查将要执行的命令，不实际传输或安装，使用：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy-core-agent-chart.ps1 -DryRun
```

## 默认参数

- 远端主机：`124.174.9.249`
- 远端用户：`root`
- 远端密码：`eisoo.com123`
- 镜像：`core-agent-service:local`
- PostgreSQL：`postgresql+psycopg://kweaver:V9_KILL_POLICY@172.31.12.93:5432/postgres`
- RabbitMQ：`amqp://kweaver:V9_KILL_POLICY@rbtmq-a1d7abfa1faf.rabbitmq.ivolces.com:5672/`
- KWeaver Base URL：`https://115.190.186.186/`
- KWeaver Decision Agent ID：`01KQ187V2TFPYMZACY8VZTMQ4Y`
- RabbitMQ consumer worker 数：`2`
- KWeaver 启动探针：`false`
- Namespace：`anybackup-ai`
- Release Name：`core-agent-service`
- KWeaver 鉴权优先级：用户名密码 > token > `~/.kweaver` 挂载

## 动态覆写规则

- 用户显式提供的参数优先于脚本默认值。
- 未提供的参数不强行覆盖，保持 `deploy.sh` 的既有默认行为。
- 如果需要传入脚本暂未单独暴露的参数，可使用 `-ExtraDeployArgs` 追加原始参数片段。
- 如果已经提供用户名密码，就不再需要挂载 `/root/.kweaver`。
- 默认关闭 KWeaver 启动探针，避免外部 OAuth 注册接口临时 500 时导致 Pod CrashLoop。

## 连接策略

- 优先使用 `pscp` 和 `plink`，因为它们支持通过密码参数完成非交互式传输与远程执行。
- 如果本机没有 `pscp/plink`，脚本会回退到系统自带的 `scp/ssh`。
- 回退到 `scp/ssh` 时，密码通常需要在终端中手动输入；如果已经配置免密登录，则可以直接执行。
- 当前受控环境通常没有 `pscp/plink`，需要全自动执行时可使用 Python `paramiko` 按默认主机、用户和密码上传 chart 包并执行远端 `bash scripts/deploy.sh`。
- 远端需要具备 `bash`、`tar`、`helm`。

## 资源

- 主脚本：`scripts/deploy-core-agent-chart.ps1`
- 参考说明：`references/deploy-chart-flow.md`

## 执行要求

- 程序日志、异常文本和运行时输出保持英文
- 说明文字与注释保持中文
- 只封装 chart 传输与安装链路，不扩展额外部署语义
