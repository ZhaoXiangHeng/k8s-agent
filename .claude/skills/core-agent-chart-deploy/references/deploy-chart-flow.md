# Chart 部署链路说明

## 固定目标

- 本地 Chart 目录：`D:\code\v9_work\core_agent_service_chart`
- 远端主机：`124.174.9.249`
- 远端登录用户：`root`
- 远端登录密码：`eisoo.com123`
- 远端执行脚本：`bash scripts/deploy.sh`
- 默认镜像参数：`core-agent-service:local`
- 默认命名空间：`anybackup-ai`

## 固定步骤

1. 将本地 chart 目录打包成临时 `tar.gz`
2. 把压缩包传到远端临时目录
3. 远端解压 chart
4. 远端执行 `bash scripts/deploy.sh`
5. 动态追加用户显式传入的安装参数
6. 清理系统临时目录中的本地压缩包和远端临时文件

## 默认参数

- `--image core-agent-service:local`
- `--database-url 'postgresql+psycopg://kweaver:V9_KILL_POLICY@172.31.12.93:5432/postgres'`
- `--rabbitmq-url 'amqp://kweaver:V9_KILL_POLICY@rbtmq-a1d7abfa1faf.rabbitmq.ivolces.com:5672/'`
- `--rabbitmq-consumer-count 2`
- `--kweaver-base-url 'https://115.190.186.186/'`
- `--kweaver-decision-agent-id '01KQ187V2TFPYMZACY8VZTMQ4Y'`
- `--kweaver-probe-on-startup false`
- `--namespace anybackup-ai`

## 覆写方式

- 若用户指定 `-Image`、`-Namespace`、`-KweaverBaseUrl`、`-KweaverDecisionAgentId` 等参数，脚本应使用用户值。
- 若用户未指定其余可选参数，则保持 `deploy.sh` 默认值。
- 若用户要求传入脚本未显式建模的参数，可通过 `-ExtraDeployArgs` 透传。

## 环境要求

- 远端主机需要可执行 `bash`、`tar`、`helm`
- 本机需要可执行 `ssh/scp`，若有 `plink/pscp` 则优先使用
