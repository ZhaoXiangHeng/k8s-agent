# K8s AI Ops 部署自动化设计

**日期**: 2026-05-27
**状态**: 设计中

## 概述

为 k8s-ai-ops 项目构建一套完整的 CI/CD 部署自动化工具链，包括：
1. 利用 WSL Docker 构建多服务镜像并导出 tar
2. 基于 paramiko 的 Python 远程执行工具
3. 镜像推送到远端 containerd 运行时
4. 引导式 Helm Chart 部署脚本
5. 更新 Helm Chart 以匹配当前服务依赖关系
6. 封装为可复用的 Claude Code Skills

目标机器：`120.55.84.39`（root / containerd 运行时 / K8s 集群已部署）

## 架构

```
Windows (开发机)
├── build-images.ps1        ← wsl docker build → WSL (Docker)
├── remote_exec.py          ← paramiko SSH → 120.55.84.39 (containerd + helm)
├── push-images.ps1         ← 调用 remote_exec.py 上传 tar + ctr import
└── deploy-chart.ps1        ← 调用 remote_exec.py 上传 chart + helm install
```

关键决策：采用**混合方案** — PowerShell 负责本地编排（WSL 调用、文件管理），Python (paramiko) 负责所有远程操作（SSH 执行命令、上传/下载文件）。

## 文件结构

```
e:\k8s-agent\
├── scripts/
│   ├── build-images.ps1          # 镜像构建
│   ├── remote_exec.py            # Python 远程执行工具
│   ├── push-images.ps1           # 镜像推送
│   └── deploy-chart.ps1          # Helm 部署
├── image-tars/                   # 构建产物（gitignore）
├── deploy/helm/k8s-ai-ops/       # 更新后的 Helm Chart
└── .claude/skills/
    ├── k8s-ai-ops-build/         # 构建 skill
    │   └── SKILL.md
    ├── k8s-ai-ops-push/          # 推送 skill
    │   └── SKILL.md
    └── k8s-ai-ops-deploy/        # 部署 skill
        └── SKILL.md
```

## 组件 1：build-images.ps1

**语言**: PowerShell
**功能**: 调用 WSL Docker 构建所有服务镜像，导出为 tar 包

**构建顺序**（按依赖关系）:
proto → mcp-server → agent-server → backend → frontend

**参数**:

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-Services` | all | 指定构建部分服务（逗号分隔） |
| `-Tag` | `yyyyMMdd-HHmmss` | 镜像 tag |
| `-SkipProto` | false | 跳过 proto 目录的复制 |
| `-DryRun` | false | 只打印命令，不执行 |

**行为细节**:
- Go 服务构建时注入 `--build-arg LDFLAGS="-s -w"` 移除调试符号，减小 tar 体积
- Docker build context 为仓库根目录（因为需要 `proto/` 目录）
- 产物输出到 `image-tars/` 目录，命名格式：`{service}-{tag}.tar`
- 构建前检查 wsl 和 docker 是否可用
- 构建后打印每个镜像的大小

## 组件 2：remote_exec.py

**语言**: Python 3
**依赖**: paramiko

**CLI 用法**:
```bash
# 执行命令
python remote_exec.py exec "helm list -A"

# 上传文件
python remote_exec.py upload ./local.tar /tmp/remote.tar

# 下载文件
python remote_exec.py download /tmp/remote.log ./local.log
```

**模块用法**:
```python
from remote_exec import RemoteExecutor

executor = RemoteExecutor(host="120.55.84.39", user="root", password="xxxx")

# 执行命令，实时输出
result = executor.exec("ls /tmp")

# 上传/下载
executor.upload("local.tar", "/tmp/remote.tar")
executor.download("/tmp/remote.tar", "local.tar")

# 作为 context manager
with RemoteExecutor(...) as e:
    e.exec("...")
```

**连接信息传递优先级**:
1. 命令行参数 `--host` / `--user` / `--password`
2. 环境变量 `REMOTE_HOST` / `REMOTE_USER` / `REMOTE_PASSWORD`
3. 模块调用时直接传入

**功能**:
- `exec(command, stream=True)` — 执行命令，stream 模式下逐行 yield 输出，非 stream 模式返回 (stdout, stderr, exit_code)
- `upload(local_path, remote_path)` — 通过 SFTP 上传文件
- `download(remote_path, local_path)` — 通过 SFTP 下载文件
- 自动处理 SSH host key（首次连接自动接受）
- 支持连接超时（默认 15s）
- 执行超时控制（默认无超时，可通过参数指定）

**错误处理**:
- 连接失败：打印明确错误信息并返回非 0 退出码
- 命令执行失败：返回 exit_code，由调用方决定处理方式
- 文件传输失败：抛出 RemoteExecutorError 异常

## 组件 3：push-images.ps1

**语言**: PowerShell
**依赖**: `remote_exec.py`、`scripts/remote_exec.py`

**参数**:

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-Services` | all | 指定推送部分服务 |
| `-Tag` | 自动匹配最新 | 镜像 tag |
| `-SkipCleanup` | false | 保留远端临时 tar 文件 |
| `-DryRun` | false | 只打印不执行 |

**流程**:
1. 扫描 `image-tars/` 目录，匹配指定 tag 的 tar 文件
2. 对每个服务的 tar 文件：
   a. 调用 `python remote_exec.py upload` 上传到 `/tmp/`
   b. 调用 `python remote_exec.py exec "ctr -n k8s.io images rm {image_tag}"` 容错删除旧镜像
   c. 调用 `python remote_exec.py exec "ctr -n k8s.io images import /tmp/{tar}"` 导入
   d. 调用 `python remote_exec.py exec "ctr -n k8s.io images list | grep {service}"` 验证
   e. 调用 `python remote_exec.py exec "rm -f /tmp/{tar}"` 清理
3. 输出推送摘要

**需要推送到所有 worker 节点吗？** 当前设计只推送到 master/控制节点。如果 `ctr import` 的镜像需要分发到各 worker，可在后续版本中添加。

## 组件 4：deploy-chart.ps1

**语言**: PowerShell
**依赖**: `remote_exec.py`

**运行模式**:

| 模式 | 参数 | 行为 |
|------|------|------|
| `guided`（默认） | `-Mode guided` | 交互式引导 |
| `auto` | `-Mode auto` | 非交互一键部署 |

**参数**:

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-Mode` | guided | `guided` 或 `auto` |
| `-Tag` | 自动取最新 | 镜像 tag |
| `-Namespace` | k8s-ai-system | K8s namespace |
| `-ReleaseName` | k8s-ai-ops | Helm release 名称 |
| `-KeycloakEnabled` | true | 是否启用 Keycloak |
| `-AuthMode` | dev | 认证模式 (dev/jwt) |
| `-StoreDriver` | postgres | 存储驱动 |
| `-CacheDriver` | redis | 缓存驱动 |
| `-RbacSyncEnabled` | true | K8s RBAC 同步开关 |
| `-DatabaseUrl` | (内置) | 外部 PostgreSQL 地址 |
| `-RedisAddr` | (内置) | 外部 Redis 地址 |
| `-EncryptionKey` | (生成) | 应用加密密钥 |
| `-DryRun` | false | 只打印不执行 |

**引导式交互流程**（guided 模式按顺序询问）:
1. Namespace（默认 `k8s-ai-system`）
2. 镜像 Tag
3. 是否启用 Keycloak
4. 认证模式（dev / jwt）
5. 数据库（内置 PostgreSQL / 外部地址）
6. Redis（内置 / 外部地址）
7. RBAC 同步开关
8. 展示配置摘要，确认后执行

**执行流程**:
1. 收集/解析所有配置参数
2. 生成临时 `values.yaml`（合并用户配置到默认 values）
3. `tar` 打包 `deploy/helm/k8s-ai-ops/`（排除 .git）
4. 调用 `python remote_exec.py upload` 上传到 `/tmp/`
5. 调用 `python remote_exec.py exec` 在远端：
   a. 解压 chart 包到临时目录
   b. 将 values.yaml 上传到远端
   c. 执行 `helm upgrade --install {release} {chart_dir} -n {namespace} --create-namespace -f values.yaml --wait --timeout 5m`
6. 清理本地和远端临时文件
7. 输出部署状态（pod 列表、service 端点）

## 组件 5：Helm Chart 更新

基于当前代码状态，更新 `deploy/helm/k8s-ai-ops/` 中的以下内容：

### a) 启动依赖 (initContainers)
- `backend` Deployment 添加 initContainer，TCP 探测 `postgresql:5432` 和 `redis:6379` 就绪
- `agent-server` Deployment 添加 initContainer，TCP 探测 `mcp-server:8081` 就绪
- `mcp-server` Deployment 添加 initContainer，TCP 探测 `backend:8082`（gRPC IdentityService）就绪

### b) Service 端口修正
- `backend` Service 加上 gRPC 端口 8082（当前只暴露了 8080）

### c) 环境变量默认值修正
- `agent-server` 的 `MCP_SERVER_URL` 默认值改为 `http://mcp-server:8081/sse`
- `mcp-server` 的 `IDENTITY_SERVER_ADDR` 默认值改为 `backend:8082`

### d) 健康检查
- 为所有 4 个服务添加 `livenessProbe` 和 `readinessProbe`
  - Go 服务（backend/agent-server/mcp-server）：TCP 探测各自服务端口
  - frontend：HTTP GET `/` 探测端口 80

### e) 镜像拉取策略
- `images.pullPolicy` 默认值 `IfNotPresent`（适配 `ctr import` 场景）

## 组件 6：Skills 封装

3 个 skill 的 SKILL.md 结构（与现有 `core-agent-*` 模式一致）：

### k8s-ai-ops-build

| 字段 | 值 |
|------|-----|
| 触发 | "构建镜像"、"编译并导出 tar"、"build images" |
| 脚本 | `scripts/build-images.ps1` |
| 目标机器 | 无（纯本地操作） |

工作流：
1. 检查 WSL 和 Docker 可用性
2. 按依赖顺序构建：proto → mcp-server → agent-server → backend → frontend
3. 每个服务 docker build + docker save 导出 tar
4. 输出构建摘要

### k8s-ai-ops-push

| 字段 | 值 |
|------|-----|
| 触发 | "推送镜像"、"导入 containerd"、"push images to 120.55.84.39" |
| 脚本 | `scripts/push-images.ps1` |
| 目标机器 | `120.55.84.39` |

工作流：
1. 扫描 image-tars/ 匹配 tar 包
2. 通过 remote_exec.py 逐服务上传 tar 到远端
3. 远端 ctr images rm（容错）→ ctr images import → 验证
4. 清理远端临时文件

### k8s-ai-ops-deploy

| 字段 | 值 |
|------|-----|
| 触发 | "部署"、"helm install"、"deploy to K8s" |
| 脚本 | `scripts/deploy-chart.ps1` |
| 目标机器 | `120.55.84.39` |

工作流：
1. 引导式或自动模式收集配置
2. 生成 values.yaml → 打包 chart → 上传远端
3. 远端 helm upgrade --install
4. 等待 rollout → 输出部署状态

## 连接策略

远程连接优先级（与现有 skill 一致）：
1. `remote_exec.py`（paramiko）— 推荐，非交互式密码认证
2. `pscp` / `plink`（PuTTY 工具）— PowerShell 脚本直接调用
3. `scp` / `ssh`（OpenSSH）— 回退方案，可能需要手动输入密码

## 实现注意事项

- 所有脚本日志和错误信息用英文
- 脚本注释和说明用中文
- `image-tars/` 目录加入 `.gitignore`
- 密码不硬编码在脚本中，通过环境变量或 CLI 参数传入
- 现有 `scripts/build-images.sh` 保留不动，`build-images.ps1` 是新的 Windows 端脚本
