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
    $latestTar = Get-ChildItem -Path $outputDir -Filter "*.tar" -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending | Select-Object -First 1
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
