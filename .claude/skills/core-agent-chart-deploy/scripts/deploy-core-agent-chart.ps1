[CmdletBinding()]
param(
    [string]$ChartPath = "D:\code\v9_work\core_agent_service_chart",
    [string]$RemoteHost = "124.174.9.249",
    [string]$RemoteUser = "root",
    [string]$RemotePassword = "eisoo.com123",
    [string]$ReleaseName = "core-agent-service",
    [string]$Namespace = "anybackup-ai",
    [string]$Image = "core-agent-service:local",
    [string]$KweaverBaseUrl = "https://115.190.186.186/",
    [string]$KweaverDecisionAgentId = "01KQ187V2TFPYMZACY8VZTMQ4Y",
    [string]$KweaverBusinessDomain = "",
    [string]$KweaverChatTimeout = "",
    [string]$KweaverTlsInsecure = "",
    [string]$DatabaseUrl = "postgresql+psycopg://kweaver:V9_KILL_POLICY@172.31.12.93:5432/postgres",
    [string]$RabbitmqUrl = "amqp://kweaver:V9_KILL_POLICY@rbtmq-a1d7abfa1faf.rabbitmq.ivolces.com:5672/",
    [string]$RabbitmqExchange = "",
    [string]$RabbitmqExchangeType = "",
    [string]$RabbitmqQueue = "",
    [string]$RabbitmqConsumerCount = "",
    [string]$KweaverUsername = "",
    [string]$KweaverPassword = "",
    [string]$KweaverToken = "",
    [string]$SecretsCreate = "",
    [string]$SecretsName = "",
    [string]$KweaverHostPath = "",
    [string]$KweaverMountPath = "",
    [string]$EnvFileEnabled = "",
    [string]$EnvFileMountPath = "",
    [string[]]$ExtraDeployArgs = @(),
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-Log {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Host "[core-agent-chart-deploy] $Message"
}

function Invoke-ExternalCommand {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [string[]]$Arguments = @()
    )

    $display = if ($Arguments.Count -gt 0) { "$FilePath " + ($Arguments -join " ") } else { $FilePath }
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
    param([Parameter(Mandatory = $true)][string]$Name)
    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($null -eq $command) {
        return $null
    }
    return $command.Source
}

function Convert-ToBashLiteral {
    param([Parameter(Mandatory = $true)][string]$Value)
    $escaped = $Value.Replace('\', '\\').Replace('"', '\"').Replace('$', '\$').Replace('`', '\`')
    return '"' + $escaped + '"'
}

function Add-DeployArg {
    param(
        [System.Collections.Generic.List[string]]$ArgsList,
        [Parameter(Mandatory = $true)][string]$Name,
        [string]$Value
    )

    if (-not [string]::IsNullOrWhiteSpace($Value)) {
        $ArgsList.Add($Name)
        $ArgsList.Add($Value)
    }
}

if (-not (Test-Path -LiteralPath $ChartPath)) {
    throw "Chart path does not exist: $ChartPath"
}

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$skillRoot = Split-Path -Parent $scriptRoot
$tempDirectory = Join-Path ([System.IO.Path]::GetTempPath()) 'core-agent-chart-deploy'
$runId = Get-Date -Format 'yyyyMMdd-HHmmss'
$archiveName = "core-agent-service-chart-$runId.tar.gz"
$localArchivePath = Join-Path $tempDirectory $archiveName
$remoteBaseDir = "/tmp/core-agent-service-chart-$runId"
$remoteArchivePath = "$remoteBaseDir/$archiveName"
$remoteExtractDir = "$remoteBaseDir/chart"

New-Item -ItemType Directory -Path $tempDirectory -Force | Out-Null
if (Test-Path -LiteralPath $localArchivePath) {
    Remove-Item -LiteralPath $localArchivePath -Force
}

$tarPath = (Get-Command 'tar' -ErrorAction Stop).Source
$pscpPath = Get-OptionalCommand -Name 'pscp'
$plinkPath = Get-OptionalCommand -Name 'plink'
$scpPath = (Get-Command 'scp' -ErrorAction Stop).Source
$sshPath = (Get-Command 'ssh' -ErrorAction Stop).Source

$deployArgs = [System.Collections.Generic.List[string]]::new()
Add-DeployArg -ArgsList $deployArgs -Name '--release-name' -Value $ReleaseName
Add-DeployArg -ArgsList $deployArgs -Name '--namespace' -Value $Namespace
Add-DeployArg -ArgsList $deployArgs -Name '--image' -Value $Image
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-base-url' -Value $KweaverBaseUrl
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-decision-agent-id' -Value $KweaverDecisionAgentId
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-business-domain' -Value $KweaverBusinessDomain
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-chat-timeout' -Value $KweaverChatTimeout
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-tls-insecure' -Value $KweaverTlsInsecure
Add-DeployArg -ArgsList $deployArgs -Name '--database-url' -Value $DatabaseUrl
Add-DeployArg -ArgsList $deployArgs -Name '--rabbitmq-url' -Value $RabbitmqUrl
Add-DeployArg -ArgsList $deployArgs -Name '--rabbitmq-exchange' -Value $RabbitmqExchange
Add-DeployArg -ArgsList $deployArgs -Name '--rabbitmq-exchange-type' -Value $RabbitmqExchangeType
Add-DeployArg -ArgsList $deployArgs -Name '--rabbitmq-queue' -Value $RabbitmqQueue
Add-DeployArg -ArgsList $deployArgs -Name '--rabbitmq-consumer-count' -Value $RabbitmqConsumerCount
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-username' -Value $KweaverUsername
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-password' -Value $KweaverPassword
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-token' -Value $KweaverToken
Add-DeployArg -ArgsList $deployArgs -Name '--secrets-create' -Value $SecretsCreate
Add-DeployArg -ArgsList $deployArgs -Name '--secrets-name' -Value $SecretsName
if ([string]::IsNullOrWhiteSpace($KweaverUsername)) {
    Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-host-path' -Value $KweaverHostPath
}
Add-DeployArg -ArgsList $deployArgs -Name '--kweaver-mount-path' -Value $KweaverMountPath
Add-DeployArg -ArgsList $deployArgs -Name '--env-file-enabled' -Value $EnvFileEnabled
Add-DeployArg -ArgsList $deployArgs -Name '--env-file-mount-path' -Value $EnvFileMountPath
foreach ($extraArg in $ExtraDeployArgs) {
    if (-not [string]::IsNullOrWhiteSpace($extraArg)) {
        $deployArgs.Add($extraArg)
    }
}

$quotedDeployArgs = ($deployArgs | ForEach-Object { Convert-ToBashLiteral -Value $_ }) -join ' '
$remoteTarget = "$RemoteUser@${RemoteHost}:$remoteArchivePath"
$remoteCommand = @(
    'set -euo pipefail'
    'cleanup() {'
    "  rm -rf $(Convert-ToBashLiteral -Value $remoteBaseDir)"
    '}'
    'trap cleanup EXIT'
    "mkdir -p $(Convert-ToBashLiteral -Value $remoteExtractDir)"
    "tar -xzf $(Convert-ToBashLiteral -Value $remoteArchivePath) -C $(Convert-ToBashLiteral -Value $remoteExtractDir) --strip-components=1"
    "cd $(Convert-ToBashLiteral -Value $remoteExtractDir)"
    "bash scripts/deploy.sh $quotedDeployArgs"
) -join "`n"

Write-Log "Preparing chart archive"
Invoke-ExternalCommand -FilePath $tarPath -Arguments @(
    '--exclude=.git',
    '--exclude=.pytest_cache',
    '--exclude=*.tar',
    '-czf',
    $localArchivePath,
    '-C',
    $ChartPath,
    '.'
)

if (-not $DryRun -and -not (Test-Path -LiteralPath $localArchivePath)) {
    throw "Chart archive was not created: $localArchivePath"
}

if ($pscpPath -and $plinkPath) {
    Write-Log 'Using PuTTY tools for non-interactive password-based transfer'
    Invoke-ExternalCommand -FilePath $plinkPath -Arguments @('-ssh', '-batch', '-pw', $RemotePassword, "$RemoteUser@$RemoteHost", "mkdir -p $(Convert-ToBashLiteral -Value $remoteBaseDir)")
    Invoke-ExternalCommand -FilePath $pscpPath -Arguments @('-pw', $RemotePassword, $localArchivePath, $remoteTarget)
    Invoke-ExternalCommand -FilePath $plinkPath -Arguments @('-ssh', '-batch', '-pw', $RemotePassword, "$RemoteUser@$RemoteHost", $remoteCommand)
}
else {
    Write-Log 'PuTTY tools not found, falling back to interactive OpenSSH commands'
    Write-Log 'If key-based SSH is not configured, the terminal may prompt for the remote password'
    Invoke-ExternalCommand -FilePath $sshPath -Arguments @("$RemoteUser@$RemoteHost", "mkdir -p $(Convert-ToBashLiteral -Value $remoteBaseDir)")
    Invoke-ExternalCommand -FilePath $scpPath -Arguments @($localArchivePath, $remoteTarget)
    Invoke-ExternalCommand -FilePath $sshPath -Arguments @("$RemoteUser@$RemoteHost", $remoteCommand)
}

if (Test-Path -LiteralPath $localArchivePath) {
    Remove-Item -LiteralPath $localArchivePath -Force
}

Write-Log 'Chart deployment workflow completed'
