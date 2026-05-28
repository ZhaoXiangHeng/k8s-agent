# K8s AI Ops 部署自动化 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一套完整的部署自动化工具链（镜像构建 → 推送 containerd → Helm 部署），并封装为 Claude Code Skills

**Architecture:** PowerShell 负责本地编排（WSL Docker 构建、文件管理），Python (paramiko) 负责所有远程操作（SSH/SFTP）。3 个独立脚本各自可独立运行，按流程串联：build → push → deploy

**Tech Stack:** PowerShell 5+, Python 3 + paramiko, WSL + Docker, Helm 3, containerd (ctr)

**Spec:** `docs/superpowers/specs/2026-05-27-k8s-ai-ops-deploy-automation-design.md`

---

### Task 1: 更新 Helm Chart — 修复服务依赖关系和配置

**Files:**
- Modify: `deploy/helm/k8s-ai-ops/values.yaml`
- Modify: `deploy/helm/k8s-ai-ops/templates/backend.yaml`
- Modify: `deploy/helm/k8s-ai-ops/templates/agent-server.yaml`
- Modify: `deploy/helm/k8s-ai-ops/templates/mcp-server.yaml`
- Modify: `deploy/helm/k8s-ai-ops/templates/frontend.yaml`

- [ ] **Step 1: values.yaml — 添加 backend gRPC 端口配置**

在 `backend.service.port` 下添加 gRPC 端口：

```yaml
backend:
  replicas: 1
  image:
    repository: k8s-ai-backend
  service:
    port: 8080
    grpcPort: 8082
```

- [ ] **Step 2: backend.yaml — 添加 gRPC Service 端口、initContainer、健康检查、env 修正**

完整重写 `deploy/helm/k8s-ai-ops/templates/backend.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-api
  namespace: {{ .Values.global.namespace }}
spec:
  replicas: {{ .Values.backend.replicas }}
  selector:
    matchLabels:
      app: backend-api
  template:
    metadata:
      labels:
        app: backend-api
    spec:
      serviceAccountName: k8s-ai-backend
      {{- if .Values.postgresql.enabled }}
      initContainers:
        - name: wait-postgresql
          image: busybox:1.36
          command:
            - sh
            - -c
            - |
              until nc -z postgresql {{ .Values.postgresql.service.port }}; do
                echo "waiting for postgresql..."
                sleep 2
              done
        {{- if .Values.redis.enabled }}
        - name: wait-redis
          image: busybox:1.36
          command:
            - sh
            - -c
            - |
              until nc -z redis {{ .Values.redis.service.port }}; do
                echo "waiting for redis..."
                sleep 2
              done
        {{- end }}
      {{- end }}
      containers:
        - name: backend-api
          image: {{ include "k8s-ai-ops.image" (list . .Values.backend.image.repository) }}
          imagePullPolicy: {{ .Values.images.pullPolicy }}
          ports:
            - name: http
              containerPort: 8080
            - name: grpc
              containerPort: 8082
          env:
            - name: HTTP_ADDR
              value: ":8080"
            - name: GRPC_ADDR
              value: ":8082"
            - name: AGENT_SERVER_ADDR
              value: "agent-server:8082"
            - name: STORE_DRIVER
              value: {{ .Values.backend.storeDriver | quote }}
            - name: CACHE_DRIVER
              value: {{ .Values.backend.cacheDriver | quote }}
            - name: K8S_RBAC_SYNC_ENABLED
              value: {{ .Values.backend.rbacSyncEnabled | quote }}
            - name: DATABASE_URL
              value: "postgres://{{ .Values.postgresql.username }}:{{ .Values.postgresql.password }}@postgresql:5432/{{ .Values.postgresql.database }}?sslmode=disable"
            - name: REDIS_ADDR
              value: "redis:6379"
            - name: APP_ENCRYPTION_KEY
              valueFrom:
                secretKeyRef:
                  name: k8s-ai-secrets
                  key: APP_ENCRYPTION_KEY
          livenessProbe:
            tcpSocket:
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            tcpSocket:
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: backend-api
  namespace: {{ .Values.global.namespace }}
spec:
  selector:
    app: backend-api
  ports:
    - name: http
      port: {{ .Values.backend.service.port }}
      targetPort: 8080
    - name: grpc
      port: {{ .Values.backend.service.grpcPort }}
      targetPort: 8082
```

- [ ] **Step 3: agent-server.yaml — 添加 initContainer、健康检查、MCP_SERVER_URL 修正**

完整重写 `deploy/helm/k8s-ai-ops/templates/agent-server.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-server
  namespace: {{ .Values.global.namespace }}
spec:
  replicas: {{ .Values.agentServer.replicas }}
  selector:
    matchLabels:
      app: agent-server
  template:
    metadata:
      labels:
        app: agent-server
    spec:
      initContainers:
        - name: wait-mcp-server
          image: busybox:1.36
          command:
            - sh
            - -c
            - |
              until nc -z mcp-server {{ .Values.mcpServer.service.port }}; do
                echo "waiting for mcp-server..."
                sleep 2
              done
      containers:
        - name: agent-server
          image: {{ include "k8s-ai-ops.image" (list . .Values.agentServer.image.repository) }}
          imagePullPolicy: {{ .Values.images.pullPolicy }}
          ports:
            - containerPort: 8082
          env:
            - name: GRPC_ADDR
              value: ":8082"
            - name: MCP_SERVER_URL
              value: "http://mcp-server:8081/sse"
          livenessProbe:
            tcpSocket:
              port: 8082
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            tcpSocket:
              port: 8082
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: agent-server
  namespace: {{ .Values.global.namespace }}
spec:
  selector:
    app: agent-server
  ports:
    - port: {{ .Values.agentServer.service.port }}
      targetPort: 8082
```

- [ ] **Step 4: mcp-server.yaml — 添加 initContainer、IDENTITY_SERVER_ADDR env、健康检查**

完整重写 `deploy/helm/k8s-ai-ops/templates/mcp-server.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-server
  namespace: {{ .Values.global.namespace }}
spec:
  replicas: {{ .Values.mcpServer.replicas }}
  selector:
    matchLabels:
      app: mcp-server
  template:
    metadata:
      labels:
        app: mcp-server
    spec:
      initContainers:
        - name: wait-backend
          image: busybox:1.36
          command:
            - sh
            - -c
            - |
              until nc -z backend-api {{ .Values.backend.service.grpcPort }}; do
                echo "waiting for backend gRPC..."
                sleep 2
              done
      containers:
        - name: mcp-server
          image: {{ include "k8s-ai-ops.image" (list . .Values.mcpServer.image.repository) }}
          imagePullPolicy: {{ .Values.images.pullPolicy }}
          ports:
            - containerPort: 8081
          env:
            - name: HTTP_ADDR
              value: ":8081"
            - name: IDENTITY_SERVER_ADDR
              value: "backend-api:8082"
          livenessProbe:
            tcpSocket:
              port: 8081
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            tcpSocket:
              port: 8081
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: mcp-server
  namespace: {{ .Values.global.namespace }}
spec:
  selector:
    app: mcp-server
  ports:
    - port: {{ .Values.mcpServer.service.port }}
      targetPort: 8081
```

- [ ] **Step 5: frontend.yaml — 添加健康检查**

完整重写 `deploy/helm/k8s-ai-ops/templates/frontend.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: {{ .Values.global.namespace }}
spec:
  replicas: {{ .Values.frontend.replicas }}
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
        - name: frontend
          image: {{ include "k8s-ai-ops.image" (list . .Values.frontend.image.repository) }}
          imagePullPolicy: {{ .Values.images.pullPolicy }}
          ports:
            - containerPort: 80
          livenessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 5
            periodSeconds: 15
          readinessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 3
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: frontend
  namespace: {{ .Values.global.namespace }}
spec:
  selector:
    app: frontend
  ports:
    - port: {{ .Values.frontend.service.port }}
      targetPort: 80
```

- [ ] **Step 6: 验证 Helm Chart 渲染**

运行: `helm template k8s-ai-ops deploy/helm/k8s-ai-ops/`
预期: 渲染成功，无错误

- [ ] **Step 7: Commit**

```bash
git add deploy/helm/k8s-ai-ops/
git commit -m "fix(chart): add initContainers, health checks, fix service ports and env vars"
```

---

### Task 2: 更新 Dockerfiles — 支持 LDFLAGS 构建参数

**Files:**
- Modify: `backend/Dockerfile`
- Modify: `agent-server/Dockerfile`
- Modify: `mcp-server/Dockerfile`

- [ ] **Step 1: 更新 backend/Dockerfile**

```dockerfile
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.26-alpine AS build
ARG LDFLAGS=""
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./... && go build -ldflags="${LDFLAGS}" -o /out/backend-api ./cmd/api

FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/gcr.io/iguazio/alpine:3.20
WORKDIR /app
COPY --from=build /out/backend-api /app/backend-api
EXPOSE 8080 8082
ENTRYPOINT ["/app/backend-api"]
```

- [ ] **Step 2: 更新 agent-server/Dockerfile**

```dockerfile
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.26-alpine AS build
ARG LDFLAGS=""
WORKDIR /src
COPY proto /src/proto
COPY agent-server /src/agent-server
WORKDIR /src/agent-server
RUN go build -ldflags="${LDFLAGS}" -o /out/agent-server ./cmd/server

FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/gcr.io/iguazio/alpine:3.20
COPY --from=build /out/agent-server /usr/local/bin/agent-server
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/agent-server"]
```

- [ ] **Step 3: 更新 mcp-server/Dockerfile**

```dockerfile
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.26-alpine AS build
ARG LDFLAGS=""
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./... && go build -ldflags="${LDFLAGS}" -o /out/mcp-server ./cmd/server

FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/gcr.io/iguazio/alpine:3.20
WORKDIR /app
COPY --from=build /out/mcp-server /app/mcp-server
EXPOSE 8081
ENTRYPOINT ["/app/mcp-server"]
```

- [ ] **Step 4: Commit**

```bash
git add backend/Dockerfile agent-server/Dockerfile mcp-server/Dockerfile
git commit -m "feat(docker): add LDFLAGS build arg for stripping debug symbols"
```

---

### Task 3: 创建 Python 远程执行工具 `remote_exec.py`

**Files:**
- Create: `scripts/remote_exec.py`

- [ ] **Step 1: 创建 `scripts/remote_exec.py`**

```python
#!/usr/bin/env python3
"""
Remote executor via SSH/SFTP using paramiko.

Usage as CLI:
    python remote_exec.py exec "helm list -A"
    python remote_exec.py upload ./local.tar /tmp/remote.tar
    python remote_exec.py download /tmp/remote.log ./local.log

Usage as module:
    from remote_exec import RemoteExecutor

    with RemoteExecutor(host="120.55.84.39", user="root", password="...") as e:
        exit_code, stdout, stderr = e.exec("ls /tmp")
        e.upload("local.tar", "/tmp/remote.tar")
        e.download("/tmp/remote.tar", "local.tar")

Connection info priority: CLI args > env vars (REMOTE_HOST, REMOTE_USER, REMOTE_PASSWORD) > defaults
"""

import os
import sys
import argparse
import stat
from contextlib import contextmanager

try:
    import paramiko
except ImportError:
    sys.exit("paramiko is required. Install with: pip install paramiko")


class RemoteExecutorError(Exception):
    """Raised when a remote operation fails."""
    pass


class RemoteExecutor:
    """SSH/SFTP remote command executor and file transfer client."""

    def __init__(
        self,
        host="120.55.84.39",
        port=22,
        user="root",
        password="",
        connect_timeout=15,
    ):
        self.host = host
        self.port = port
        self.user = user
        self.password = password
        self.connect_timeout = connect_timeout
        self._client = None
        self._sftp = None

    def _ensure_connected(self):
        if self._client is not None:
            return
        client = paramiko.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        try:
            client.connect(
                hostname=self.host,
                port=self.port,
                username=self.user,
                password=self.password,
                timeout=self.connect_timeout,
                allow_agent=False,
                look_for_keys=False,
            )
        except Exception as e:
            raise RemoteExecutorError(
                f"Failed to connect to {self.user}@{self.host}:{self.port}: {e}"
            ) from e
        self._client = client

    def _ensure_sftp(self):
        self._ensure_connected()
        if self._sftp is not None:
            return self._sftp
        self._sftp = self._client.open_sftp()
        return self._sftp

    def exec(self, command, stream=False, timeout=None):
        """
        Execute a command on the remote host.

        Args:
            command: Shell command to execute.
            stream: If True, yield (channel, line) tuples for real-time output.
                    If False, return (exit_code, stdout, stderr).
            timeout: Command timeout in seconds. None = no timeout.

        Returns:
            If stream=False: (exit_code: int, stdout: str, stderr: str)
            If stream=True: generator yielding (channel, line: str)
        """
        self._ensure_connected()
        chan = self._client.get_transport().open_session(timeout=timeout)
        chan.set_combine_stderr(False)
        chan.exec_command(command)

        if stream:
            return self._stream_output(chan)
        else:
            return self._collect_output(chan, timeout)

    def _stream_output(self, chan):
        """Generator that yields (channel, line) for real-time output."""
        import select as _select
        stdout_buf = b""
        stderr_buf = b""

        while not chan.exit_status_ready():
            if chan.recv_ready():
                data = chan.recv(4096)
                if data:
                    stdout_buf += data
                    while b"\n" in stdout_buf:
                        line, stdout_buf = stdout_buf.split(b"\n", 1)
                        yield (chan, "stdout", line.decode("utf-8", errors="replace"))
            if chan.recv_stderr_ready():
                data = chan.recv_stderr(4096)
                if data:
                    stderr_buf += data
                    while b"\n" in stderr_buf:
                        line, stderr_buf = stderr_buf.split(b"\n", 1)
                        yield (chan, "stderr", line.decode("utf-8", errors="replace"))

        # Drain remaining
        while chan.recv_ready():
            stdout_buf += chan.recv(4096)
        while chan.recv_stderr_ready():
            stderr_buf += chan.recv_stderr(4096)

        if stdout_buf:
            yield (chan, "stdout", stdout_buf.decode("utf-8", errors="replace"))
        if stderr_buf:
            yield (chan, "stderr", stderr_buf.decode("utf-8", errors="replace"))

    def _collect_output(self, chan, timeout):
        """Collect all output, return (exit_code, stdout, stderr)."""
        stdout_lines = []
        stderr_lines = []
        for _, stream_name, line in self._stream_output(chan):
            if stream_name == "stdout":
                stdout_lines.append(line)
            else:
                stderr_lines.append(line)
        exit_code = chan.recv_exit_status()
        return (exit_code, "\n".join(stdout_lines), "\n".join(stderr_lines))

    def upload(self, local_path, remote_path):
        """Upload a file to the remote host via SFTP."""
        sftp = self._ensure_sftp()
        try:
            sftp.put(local_path, remote_path)
        except Exception as e:
            raise RemoteExecutorError(
                f"Failed to upload {local_path} -> {remote_path}: {e}"
            ) from e

    def download(self, remote_path, local_path):
        """Download a file from the remote host via SFTP."""
        sftp = self._ensure_sftp()
        try:
            sftp.get(remote_path, local_path)
        except Exception as e:
            raise RemoteExecutorError(
                f"Failed to download {remote_path} -> {local_path}: {e}"
            ) from e

    def close(self):
        if self._sftp is not None:
            self._sftp.close()
            self._sftp = None
        if self._client is not None:
            self._client.close()
            self._client = None

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
        return False


def _get_executor_from_env_or_args(args):
    """Build RemoteExecutor from CLI args merged with env vars."""
    host = args.host or os.environ.get("REMOTE_HOST", "120.55.84.39")
    port = args.port or int(os.environ.get("REMOTE_PORT", "22"))
    user = args.user or os.environ.get("REMOTE_USER", "root")
    password = args.password or os.environ.get("REMOTE_PASSWORD", "")
    return RemoteExecutor(host=host, port=port, user=user, password=password)


def cmd_exec(args):
    executor = _get_executor_from_env_or_args(args)
    with executor:
        if args.stream:
            for _, stream_name, line in executor.exec(args.command, stream=True):
                if stream_name == "stdout":
                    print(line)
                else:
                    print(line, file=sys.stderr)
        else:
            exit_code, stdout, stderr = executor.exec(args.command)
            if stdout:
                print(stdout)
            if stderr:
                print(stderr, file=sys.stderr)
            sys.exit(exit_code)


def cmd_upload(args):
    executor = _get_executor_from_env_or_args(args)
    with executor:
        executor.upload(args.local_path, args.remote_path)
        print(f"Uploaded {args.local_path} -> {args.remote_path}")


def cmd_download(args):
    executor = _get_executor_from_env_or_args(args)
    with executor:
        executor.download(args.remote_path, args.local_path)
        print(f"Downloaded {args.remote_path} -> {args.local_path}")


def main():
    parser = argparse.ArgumentParser(description="Remote executor via SSH/SFTP")
    parser.add_argument("--host", help="Remote host (env: REMOTE_HOST)")
    parser.add_argument("--port", type=int, help="SSH port (env: REMOTE_PORT, default: 22)")
    parser.add_argument("--user", help="SSH user (env: REMOTE_USER, default: root)")
    parser.add_argument("--password", help="SSH password (env: REMOTE_PASSWORD)")

    subparsers = parser.add_subparsers(dest="action", required=True)

    exec_parser = subparsers.add_parser("exec", help="Execute remote command")
    exec_parser.add_argument("command", help="Command to execute")
    exec_parser.add_argument("--stream", action="store_true", help="Stream output in real-time")
    exec_parser.set_defaults(func=cmd_exec)

    upload_parser = subparsers.add_parser("upload", help="Upload a file")
    upload_parser.add_argument("local_path", help="Local file path")
    upload_parser.add_argument("remote_path", help="Remote destination path")
    upload_parser.set_defaults(func=cmd_upload)

    download_parser = subparsers.add_parser("download", help="Download a file")
    download_parser.add_argument("remote_path", help="Remote file path")
    download_parser.add_argument("local_path", help="Local destination path")
    download_parser.set_defaults(func=cmd_download)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
```

- [ ] **Step 2: 验证 remote_exec.py 语法**

```bash
python -c "import py_compile; py_compile.compile('scripts/remote_exec.py', doraise=True)"
```

- [ ] **Step 3: Commit**

```bash
git add scripts/remote_exec.py
git commit -m "feat: add Python remote executor via paramiko SSH/SFTP"
```

---

### Task 4: 创建 `build-images.ps1` — Windows 端 WSL Docker 构建脚本

**Files:**
- Create: `scripts/build-images.ps1`

- [ ] **Step 1: 创建 `scripts/build-images.ps1`**

```powershell
[CmdletBinding()]
param(
    [string]$Services = "all",
    [string]$Tag = "",
    [switch]$SkipProto,
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptRoot
$outputDir = Join-Path $repoRoot "image-tars"

function Write-Log {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Host "[build-images] $Message"
}

function Convert-ToWslPath {
    param([Parameter(Mandatory = $true)][string]$WindowsPath)
    $fullPath = [System.IO.Path]::GetFullPath($WindowsPath)
    $normalized = $fullPath.Replace("\", "/")
    if ($normalized -match "^([A-Za-z]):/(.*)$") {
        $drive = $Matches[1].ToLowerInvariant()
        $rest = $Matches[2]
        return "/mnt/$drive/$rest"
    }
    throw "Unsupported Windows path for WSL conversion: $WindowsPath"
}

function Invoke-WslCommand {
    param(
        [Parameter(Mandatory = $true)][string]$Command,
        [string]$Description = ""
    )
    if ($DryRun) {
        Write-Log "DRY RUN: wsl bash -lc '$Command'"
        return
    }
    if ($Description) {
        Write-Log $Description
    }
    $wslPath = (Get-Command "wsl" -ErrorAction Stop).Source
    & $wslPath bash -lc $Command
    if ($LASTEXITCODE -ne 0) {
        throw "WSL command failed with exit code ${LASTEXITCODE}: $Command"
    }
}

if ([string]::IsNullOrWhiteSpace($Tag)) {
    $Tag = Get-Date -Format "yyyyMMdd-HHmmss"
}
Write-Log "Using image tag: $Tag"

$wslRepoRoot = Convert-ToWslPath $repoRoot
$wslOutputDir = Convert-ToWslPath $outputDir

# Ensure output directory exists
if (-not $DryRun) {
    New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
}

$allServices = @("mcp-server", "agent-server", "backend", "frontend")
$servicesToBuild = if ($Services -eq "all") { $allServices } else { $Services.Split(",") | ForEach-Object { $_.Trim() } }

$serviceConfig = @{
    "mcp-server" = @{
        Dockerfile = "mcp-server/Dockerfile"
        Context = "mcp-server"
        Image = "k8s-ai-mcp-server"
        LdFlags = $true
    }
    "agent-server" = @{
        Dockerfile = "agent-server/Dockerfile"
        Context = "."
        Image = "k8s-ai-agent-server"
        LdFlags = $true
    }
    "backend" = @{
        Dockerfile = "backend/Dockerfile"
        Context = "backend"
        Image = "k8s-ai-backend"
        LdFlags = $true
    }
    "frontend" = @{
        Dockerfile = "frontend/Dockerfile"
        Context = "frontend"
        Image = "k8s-ai-frontend"
        LdFlags = $false
    }
}

Write-Log "Starting builds for services: $($servicesToBuild -join ', ')"

foreach ($svc in $servicesToBuild) {
    $cfg = $serviceConfig[$svc]
    $imageTag = "$($cfg.Image):$Tag"
    $tarName = "$($cfg.Image)-$Tag.tar"
    $tarPath = Join-Path $outputDir $tarName
    $wslTarPath = Convert-ToWslPath $tarPath
    $wslContextPath = if ($cfg.Context -eq ".") { $wslRepoRoot } else { "$wslRepoRoot/$($cfg.Context)" }

    Write-Log "Building $svc ($imageTag)"

    # Clean up old tar if exists
    if (-not $DryRun -and (Test-Path -LiteralPath $tarPath)) {
        Remove-Item -LiteralPath $tarPath -Force
    }

    # Build docker image
    $buildArgs = ""
    if ($cfg.LdFlags) {
        $buildArgs = "--build-arg LDFLAGS='-s -w'"
    }
    $buildCmd = "cd '$wslRepoRoot' && docker build $buildArgs -f '$wslRepoRoot/$($cfg.Dockerfile)' -t '$imageTag' '$wslContextPath'"
    Invoke-WslCommand -Command $buildCmd -Description "docker build $svc"

    # Save image as tar
    $saveCmd = "docker save -o '$wslTarPath' '$imageTag'"
    Invoke-WslCommand -Command $saveCmd -Description "docker save $svc"

    # Report file size
    if (-not $DryRun) {
        $tarInfo = Get-Item -LiteralPath $tarPath
        $sizeMB = [math]::Round($tarInfo.Length / 1MB, 2)
        Write-Log "$svc tar: $tarName ($sizeMB MB)"
    }
}

Write-Log "Build complete. Tag: $Tag"
Write-Log "Tars in: $outputDir"
foreach ($svc in $servicesToBuild) {
    $cfg = $serviceConfig[$svc]
    $tarName = "$($cfg.Image)-$Tag.tar"
    Write-Log "  $svc -> $tarName"
}
```

- [ ] **Step 2: Commit**

```bash
git add scripts/build-images.ps1
git commit -m "feat: add Windows PowerShell build script via WSL Docker"
```

---

### Task 5: 创建 `push-images.ps1` — 镜像推送到 containerd

**Files:**
- Create: `scripts/push-images.ps1`

- [ ] **Step 1: 创建 `scripts/push-images.ps1`**

```powershell
[CmdletBinding()]
param(
    [string]$Services = "all",
    [string]$Tag = "",
    [switch]$SkipCleanup,
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptRoot
$outputDir = Join-Path $repoRoot "image-tars"
$remoteExecScript = Join-Path $scriptRoot "remote_exec.py"

function Write-Log {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Host "[push-images] $Message"
}

function Invoke-RemotePython {
    param(
        [Parameter(Mandatory = $true)][string]$Action,
        [string[]]$ExtraArgs = @()
    )
    $args = @($remoteExecScript, $Action) + $ExtraArgs
    if ($DryRun) {
        Write-Log "DRY RUN: python $($args -join ' ')"
        return ""
    }
    $result = & python $args 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Log "Remote python command failed: $result"
        throw "Remote command failed"
    }
    return $result
}

$allServices = @("mcp-server", "agent-server", "backend", "frontend")
$servicesToPush = if ($Services -eq "all") { $allServices } else { $Services.Split(",") | ForEach-Object { $_.Trim() } }

$serviceConfig = @{
    "mcp-server" = @{ Image = "k8s-ai-mcp-server" }
    "agent-server" = @{ Image = "k8s-ai-agent-server" }
    "backend" = @{ Image = "k8s-ai-backend" }
    "frontend" = @{ Image = "k8s-ai-frontend" }
}

# Auto-detect tag if not specified
if ([string]::IsNullOrWhiteSpace($Tag)) {
    $latestTar = Get-ChildItem -Path $outputDir -Filter "*.tar" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($latestTar) {
        # Extract tag from filename like "k8s-ai-backend-yyyyMMdd-HHmmss.tar"
        if ($latestTar.Name -match "-(\d{8}-\d{6})\.tar$") {
            $Tag = $Matches[1]
        }
    }
    if ([string]::IsNullOrWhiteSpace($Tag)) {
        throw "No tar files found in $outputDir and no -Tag specified"
    }
}
Write-Log "Using image tag: $Tag"

foreach ($svc in $servicesToPush) {
    $cfg = $serviceConfig[$svc]
    $imageTag = "$($cfg.Image):$Tag"
    $tarName = "$($cfg.Image)-$Tag.tar"
    $tarPath = Join-Path $outputDir $tarName
    $remoteTarPath = "/tmp/$tarName"

    if (-not $DryRun -and -not (Test-Path -LiteralPath $tarPath)) {
        throw "Tar file not found: $tarPath"
    }

    Write-Log "Pushing $svc ($imageTag)"

    # Step 1: Upload tar
    Write-Log "  Uploading $tarName..."
    Invoke-RemotePython -Action "upload" -ExtraArgs @($tarPath, $remoteTarPath)

    # Step 2: Remove old image (best-effort)
    Write-Log "  Removing old image (if exists)..."
    try {
        Invoke-RemotePython -Action "exec" -ExtraArgs @("ctr -n k8s.io images rm '$imageTag' 2>/dev/null || true")
    } catch {
        Write-Log "  (old image removal skipped or failed, continuing)"
    }

    # Step 3: Import new image
    Write-Log "  Importing into containerd..."
    $result = Invoke-RemotePython -Action "exec" -ExtraArgs @("ctr -n k8s.io images import '$remoteTarPath'")
    Write-Log "  Import output: $result"

    # Step 4: Verify
    Write-Log "  Verifying..."
    $verifyResult = Invoke-RemotePython -Action "exec" -ExtraArgs @("ctr -n k8s.io images list | grep '$($cfg.Image)' || echo 'WARNING: image not found'")
    Write-Log "  Verify: $verifyResult"

    # Step 5: Cleanup
    if (-not $SkipCleanup) {
        Write-Log "  Cleaning up remote tar..."
        Invoke-RemotePython -Action "exec" -ExtraArgs @("rm -f '$remoteTarPath'")
    }
}

Write-Log "Push complete. Tag: $Tag"
```

- [ ] **Step 2: Commit**

```bash
git add scripts/push-images.ps1
git commit -m "feat: add image push script for containerd via remote_exec.py"
```

---

### Task 6: 创建 `deploy-chart.ps1` — Helm 部署脚本

**Files:**
- Create: `scripts/deploy-chart.ps1`

- [ ] **Step 1: 创建 `scripts/deploy-chart.ps1`**

```powershell
[CmdletBinding()]
param(
    [ValidateSet("guided", "auto")]
    [string]$Mode = "guided",
    [string]$Tag = "",
    [string]$Namespace = "k8s-ai-system",
    [string]$ReleaseName = "k8s-ai-ops",
    [string]$KeycloakEnabled = "",
    [string]$AuthMode = "",
    [string]$StoreDriver = "",
    [string]$CacheDriver = "",
    [string]$RbacSyncEnabled = "",
    [string]$DatabaseUrl = "",
    [string]$RedisAddr = "",
    [string]$EncryptionKey = "",
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptRoot
$chartPath = Join-Path $repoRoot "deploy\helm\k8s-ai-ops"
$remoteExecScript = Join-Path $scriptRoot "remote_exec.py"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "k8s-ai-ops-deploy"

function Write-Log {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Host "[deploy-chart] $Message"
}

function Invoke-RemotePython {
    param(
        [Parameter(Mandatory = $true)][string]$Action,
        [string[]]$ExtraArgs = @()
    )
    $args = @($remoteExecScript, $Action) + $ExtraArgs
    if ($DryRun) {
        Write-Log "DRY RUN: python $($args -join ' ')"
        return ""
    }
    $result = & python $args 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Log "Remote command failed: $result"
        throw "Remote command failed"
    }
    return $result
}

function Read-UserInput {
    param(
        [Parameter(Mandatory = $true)][string]$Prompt,
        [string]$Default = ""
    )
    if ($Default) {
        $response = Read-Host "$Prompt [$Default]"
        if ([string]::IsNullOrWhiteSpace($response)) {
            return $Default
        }
        return $response
    }
    else {
        return Read-Host "$Prompt"
    }
}

# Guided mode: interactive prompts
if ($Mode -eq "guided") {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  K8s AI Ops - Helm Chart Deployment" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""

    $Namespace = Read-UserInput "Namespace" $Namespace
    $ReleaseName = Read-UserInput "Release name" $ReleaseName

    if ([string]::IsNullOrWhiteSpace($Tag)) {
        $Tag = Read-UserInput "Image tag" (Get-Date -Format "yyyyMMdd-HHmmss")
    }

    $keycloakChoice = Read-UserInput "Enable Keycloak? (y/n)" "y"
    $KeycloakEnabled = if ($keycloakChoice -eq "y") { "true" } else { "false" }

    $AuthMode = Read-UserInput "Auth mode (dev/jwt)" "dev"

    $dbChoice = Read-UserInput "PostgreSQL: built-in (b) or external URL? (b/url)" "b"
    if ($dbChoice -ne "b") {
        $DatabaseUrl = Read-UserInput "  Database URL"
    }

    $redisChoice = Read-UserInput "Redis: built-in (b) or external address? (b/addr)" "b"
    if ($redisChoice -ne "b") {
        $RedisAddr = Read-UserInput "  Redis address"
    }

    $rbacChoice = Read-UserInput "Enable K8s RBAC sync? (y/n)" "y"
    $RbacSyncEnabled = if ($rbacChoice -eq "y") { "true" } else { "false" }

    if ($AuthMode -eq "dev") {
        $EncryptionKey = "dev-32-byte-key-not-for-production"
    }
    else {
        $EncryptionKey = Read-UserInput "Encryption key (32 bytes)" "change-me-32-byte-development-key"
    }

    Write-Host ""
    Write-Host "--- Configuration Summary ---" -ForegroundColor Yellow
    Write-Host "  Namespace:        $Namespace"
    Write-Host "  Release:          $ReleaseName"
    Write-Host "  Image Tag:        $Tag"
    Write-Host "  Keycloak:         $KeycloakEnabled"
    Write-Host "  Auth Mode:        $AuthMode"
    Write-Host "  Store Driver:     $(if ($DatabaseUrl) { 'external' } else { 'postgres (built-in)' })"
    Write-Host "  Cache Driver:     $(if ($RedisAddr) { 'external' } else { 'redis (built-in)' })"
    Write-Host "  RBAC Sync:        $RbacSyncEnabled"
    Write-Host ""

    $confirm = Read-UserInput "Proceed with deployment? (y/n)" "y"
    if ($confirm -ne "y") {
        Write-Log "Deployment cancelled."
        exit 0
    }
}

# Defaults for auto mode
if ([string]::IsNullOrWhiteSpace($Tag)) {
    $Tag = Get-Date -Format "yyyyMMdd-HHmmss"
}
if ([string]::IsNullOrWhiteSpace($KeycloakEnabled)) {
    $KeycloakEnabled = "true"
}
if ([string]::IsNullOrWhiteSpace($AuthMode)) {
    $AuthMode = "dev"
}
if ([string]::IsNullOrWhiteSpace($StoreDriver)) {
    $StoreDriver = "postgres"
}
if ([string]::IsNullOrWhiteSpace($CacheDriver)) {
    $CacheDriver = "redis"
}
if ([string]::IsNullOrWhiteSpace($RbacSyncEnabled)) {
    $RbacSyncEnabled = "true"
}

Write-Log "Deploying chart with tag=$Tag, namespace=$Namespace, release=$ReleaseName"

# Prepare temp directory
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

# Generate override values.yaml
$runId = Get-Date -Format "yyyyMMdd-HHmmss"
$valuesContent = @"
global:
  namespace: $Namespace

images:
  source: registry
  tag: "$Tag"
  pullPolicy: IfNotPresent

keycloak:
  enabled: $KeycloakEnabled

backend:
  storeDriver: $StoreDriver
  cacheDriver: $CacheDriver
  rbacSyncEnabled: $RbacSyncEnabled
"@

if ($DatabaseUrl) {
    $valuesContent += "`n  databaseUrl: `"$DatabaseUrl`"`n"
}
if ($RedisAddr) {
    $valuesContent += "`n  redisAddr: `"$RedisAddr`"`n"
}
if ($EncryptionKey) {
    $valuesContent += "`n  encryptionKey: `"$EncryptionKey`"`n"
}

$valuesPath = Join-Path $tempDir "values-override.yaml"
$valuesContent | Out-File -FilePath $valuesPath -Encoding utf8 -NoNewline

# Package chart
$archiveName = "k8s-ai-ops-chart-$runId.tar.gz"
$localArchivePath = Join-Path $tempDir $archiveName
$remoteBaseDir = "/tmp/k8s-ai-ops-deploy-$runId"
$remoteArchivePath = "$remoteBaseDir/$archiveName"
$remoteExtractDir = "$remoteBaseDir/chart"

Write-Log "Packaging chart from $chartPath"
$tarPath = (Get-Command "tar" -ErrorAction Stop).Source
& $tarPath --exclude='.git' --exclude='*.tar' -czf $localArchivePath -C $chartPath "."

if (-not $DryRun -and -not (Test-Path -LiteralPath $localArchivePath)) {
    throw "Chart archive was not created: $localArchivePath"
}

# Upload chart and values to remote
Write-Log "Uploading chart to remote..."
Invoke-RemotePython -Action "exec" -ExtraArgs @("mkdir -p $remoteExtractDir")
Invoke-RemotePython -Action "upload" -ExtraArgs @($localArchivePath, $remoteArchivePath)
Invoke-RemotePython -Action "upload" -ExtraArgs @($valuesPath, "$remoteExtractDir/values-override.yaml")

# Deploy via helm
Write-Log "Running helm upgrade --install..."
$helmCmd = @(
    "cd $remoteExtractDir",
    "tar -xzf $remoteArchivePath --strip-components=0",
    "helm upgrade --install $ReleaseName .",
    "-n $Namespace",
    "--create-namespace",
    "-f values-override.yaml",
    "--wait",
    "--timeout 5m"
) -join " && "

$deployResult = Invoke-RemotePython -Action "exec" -ExtraArgs @($helmCmd)
Write-Log "Helm output: $deployResult"

# Show pod status
Write-Log "Checking deployment status..."
$podResult = Invoke-RemotePython -Action "exec" -ExtraArgs @("kubectl get pods -n $Namespace")
Write-Log "Pods: $podResult"

# Cleanup local temp
Remove-Item -Path $localArchivePath -Force -ErrorAction SilentlyContinue
Remove-Item -Path $valuesPath -Force -ErrorAction SilentlyContinue

# Cleanup remote temp
Invoke-RemotePython -Action "exec" -ExtraArgs @("rm -rf $remoteBaseDir")

Write-Log "Deployment complete."
```

- [ ] **Step 2: Commit**

```bash
git add scripts/deploy-chart.ps1
git commit -m "feat: add guided/auto Helm deploy script via remote_exec.py"
```

---

### Task 7: 创建 Skills（3 个 SKILL.md）

**Files:**
- Create: `.claude/skills/k8s-ai-ops-build/SKILL.md`
- Create: `.claude/skills/k8s-ai-ops-push/SKILL.md`
- Create: `.claude/skills/k8s-ai-ops-deploy/SKILL.md`

- [ ] **Step 1: 创建 `.claude/skills/k8s-ai-ops-build/SKILL.md`**

```markdown
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
```

- [ ] **Step 2: 创建 `.claude/skills/k8s-ai-ops-push/SKILL.md`**

```markdown
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
```

- [ ] **Step 3: 创建 `.claude/skills/k8s-ai-ops-deploy/SKILL.md`**

```markdown
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
```

- [ ] **Step 4: Commit**

```bash
git add .claude/skills/k8s-ai-ops-build/ .claude/skills/k8s-ai-ops-push/ .claude/skills/k8s-ai-ops-deploy/
git commit -m "feat: add k8s-ai-ops build/push/deploy skills"
```

---

## Plan Review Checklist

在实现完成前，确认以下验收条件：

1. **Chart 更新验证**: `helm lint deploy/helm/k8s-ai-ops/` 通过，`helm template` 渲染无错误
2. **remote_exec.py**: `python -c "import scripts.remote_exec"` 无语法错误
3. **build-images.ps1**: `-DryRun` 模式输出所有 wsl 命令不报错
4. **push-images.ps1**: `-DryRun` 模式输出所有命令不报错
5. **deploy-chart.ps1**: `-DryRun -Mode auto` 模式输出所有命令不报错
6. **Skills**: 3 个 SKILL.md 文件存在且 frontmatter 格式正确
