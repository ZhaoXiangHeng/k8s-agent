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
