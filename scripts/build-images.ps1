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
