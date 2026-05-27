# 部署链路说明

## 固定目标

- 本地仓库目录：`D:\code\v9_work\Anybackup\Agent\service\core_agent`
- 本地构建方式：在 WSL 中执行 `docker build`
- 本地镜像标签：默认自动生成 `core-agent-service:yyyyMMdd-HHmmss`
- 远端主机：`124.174.9.249`
- 远端登录用户：`root`
- 远端登录密码：`eisoo.com123`
- 远端导入命令：`ctr -n k8s.io images import`

## 固定步骤

1. 在 `D:\code\v9_work\Anybackup\Agent\service\core_agent` 目录执行 `docker build -t core-agent-service:<timestamp> .`
2. 执行 `docker save -o <tar路径> core-agent-service:<timestamp>`
3. 将 tar 包传到远端 `/tmp/core-agent-service-image.tar`
4. 远端执行 `ctr -n k8s.io images rm core-agent-service:<timestamp>`，如果同 tag 镜像不存在则忽略错误
5. 远端执行 `ctr -n k8s.io images import /tmp/core-agent-service-image.tar`
6. 远端执行 `ctr -n k8s.io images list` 查询，确认当前 tag 的 `core-agent-service` 已出现
7. 删除远端临时 tar 文件

## 连接兼容策略

- 如果本机安装了 `pscp` 与 `plink`，脚本使用密码参数执行非交互式上传与远程命令。
- 如果本机没有 `pscp` 与 `plink`，脚本回退到 `scp` 与 `ssh`。
- 回退模式下通常需要人工输入密码；如果已配置 SSH 免密，则可直接执行。

## 已知优化点

- 远端删除旧镜像不再依赖 `crictl inspecti` 预检查，避免输出帮助信息。
- 远端导入完成后会立即执行 `ctr -n k8s.io images list` 并打印镜像查询结果，便于确认导入是否成功。
- 本地默认自动生成时间戳 tag，构建、导出、远端删除与验收都使用同一个 tag。
- 远端临时 tar 文件通过 `trap` 做兜底清理，减少失败后残留。
- 当前受控终端中执行 `wsl` 可能需要提权；这属于环境限制，不属于技能逻辑异常。

## 受限环境说明

- 某些受控终端中调用 `wsl` 可能会返回访问拒绝。
- 遇到该情况时，应在用户自己的终端中运行脚本，或在获得授权后提升执行权限重试。
