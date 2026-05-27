[CmdletBinding()]
param(
    [string]$RepoPath = "D:\code\v9_work\Anybackup\Agent\service\core_agent",
    [string]$ImageTag = "",
    [string]$RemoteHost = "124.174.9.249",
    [string]$RemoteUser = "root",
    [string]$RemotePassword = "eisoo.com123",
    [string]$RemoteTarPath = "/tmp/core-agent-service-image.tar",
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-Log {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    Write-Host "[core-agent-image-push] $Message"
}

function Convert-ToWslPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$WindowsPath
    )

    $fullPath = [System.IO.Path]::GetFullPath($WindowsPath)
    $normalized = $fullPath.Replace("\", "/")
    if ($normalized -match "^([A-Za-z]):/(.*)$") {
        $drive = $Matches[1].ToLowerInvariant()
        $rest = $Matches[2]
        return "/mnt/$drive/$rest"
    }

    throw "Unsupported Windows path for WSL conversion: $WindowsPath"
}

function Invoke-ExternalCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$Arguments = @()
    )

    $display = if ($Arguments.Count -gt 0) {
        "$FilePath " + ($Arguments -join " ")
    }
    else {
        $FilePath
    }

    if ($DryRun) {
        Write-Log "DRY RUN: $display"
        return
    }

    Write-Log "Running command: $display"
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $display"
    }
}

function Get-OptionalCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($null -eq $command) {
        return $null
    }

    return $command.Source
}

function Get-RemoteVerificationPattern {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ImageTag
    )

    return [regex]::Escape($ImageTag)
}

# 统一把临时产物放在技能目录下，避免污染业务目录。
$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$skillRoot = Split-Path -Parent $scriptRoot
$tempDirectory = Join-Path $skillRoot ".tmp"

if ([string]::IsNullOrWhiteSpace($ImageTag)) {
    $ImageTag = "core-agent-service:{0}" -f (Get-Date -Format "yyyyMMdd-HHmmss")
}

$tagSuffix = $ImageTag.Replace(":", "_")
$localTarPath = Join-Path $tempDirectory ("{0}.tar" -f $tagSuffix)
Write-Log "Using image tag: $ImageTag"

if (-not (Test-Path -LiteralPath $RepoPath)) {
    throw "Repository path does not exist: $RepoPath"
}

New-Item -ItemType Directory -Path $tempDirectory -Force | Out-Null
if (Test-Path -LiteralPath $localTarPath) {
    Remove-Item -LiteralPath $localTarPath -Force
}

$wslRepoPath = Convert-ToWslPath -WindowsPath $RepoPath
$wslTarPath = Convert-ToWslPath -WindowsPath $localTarPath

$wslBuildScript = @(
    "set -euo pipefail"
    "cd '$wslRepoPath'"
    "docker build -t '$ImageTag' ."
    "docker save -o '$wslTarPath' '$ImageTag'"
) -join "; "

$pscpPath = Get-OptionalCommand -Name "pscp"
$plinkPath = Get-OptionalCommand -Name "plink"
$scpPath = (Get-Command "scp" -ErrorAction Stop).Source
$sshPath = (Get-Command "ssh" -ErrorAction Stop).Source
$wslPath = (Get-Command "wsl" -ErrorAction Stop).Source

$remoteTarget = "$RemoteUser@${RemoteHost}:$RemoteTarPath"
$remoteVerifyPattern = Get-RemoteVerificationPattern -ImageTag $ImageTag
$remoteShellScript = @(
    "set -e"
    "cleanup() {"
    "  rm -f '$RemoteTarPath'"
    "}"
    "trap cleanup EXIT"
    "ctr -n k8s.io images rm '$ImageTag' >/dev/null 2>&1 || true"
    "ctr -n k8s.io images import '$RemoteTarPath'"
    "echo 'Imported image summary:'"
    "ctr -n k8s.io images list | grep -E '$remoteVerifyPattern'"
    "rm -f '$RemoteTarPath'"
) -join "`n"

Write-Log "Starting image build workflow"
Invoke-ExternalCommand -FilePath $wslPath -Arguments @("bash", "-lc", $wslBuildScript)

if (-not $DryRun -and -not (Test-Path -LiteralPath $localTarPath)) {
    throw "Local tar file was not created: $localTarPath"
}

if ($pscpPath -and $plinkPath) {
    Write-Log "Using PuTTY tools for non-interactive password-based transfer"
    Invoke-ExternalCommand -FilePath $pscpPath -Arguments @("-pw", $RemotePassword, $localTarPath, $remoteTarget)
    Invoke-ExternalCommand -FilePath $plinkPath -Arguments @("-ssh", "-batch", "-pw", $RemotePassword, "$RemoteUser@$RemoteHost", $remoteShellScript)
}
else {
    Write-Log "PuTTY tools not found, falling back to interactive OpenSSH commands"
    Write-Log "If key-based SSH is not configured, the terminal may prompt for the remote password"
    Invoke-ExternalCommand -FilePath $scpPath -Arguments @($localTarPath, $remoteTarget)
    Invoke-ExternalCommand -FilePath $sshPath -Arguments @("$RemoteUser@$RemoteHost", $remoteShellScript)
}

if (Test-Path -LiteralPath $localTarPath) {
    Remove-Item -LiteralPath $localTarPath -Force
}

Write-Log "Image transfer workflow completed"
