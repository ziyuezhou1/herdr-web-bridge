[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'process-control.ps1')
if ($env:OS -ne 'Windows_NT') { throw 'Windows is required.' }

$configRoot = Join-Path $env:LOCALAPPDATA 'HerdrWebBridge'
$statePath = Join-Path $configRoot 'install-state.json'
if (-not (Test-Path -LiteralPath $statePath -PathType Leaf)) { throw 'No installation rollback state was found.' }
$state = Get-Content -Raw -LiteralPath $statePath | ConvertFrom-Json
if ($state.schemaVersion -ne 1) { throw 'Unsupported rollback state.' }

$installRoot = [IO.Path]::GetFullPath([string]$state.installRoot)
$allowedParent = [IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs'))
if (-not $installRoot.StartsWith($allowedParent + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Rollback install root is outside the per-user Programs directory.'
}

$logPath = Join-Path $configRoot 'install.log.jsonl'
function Write-RollbackLog([string]$Action, [string]$Target, [string]$Status) {
    [ordered]@{ time = [DateTime]::UtcNow.ToString('o'); action = $Action; target = $Target; status = $Status } |
        ConvertTo-Json -Compress | Add-Content -LiteralPath $logPath -Encoding UTF8
}

$registryPath = 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.herdr_web_bridge'
$currentRegistryExisted = Test-Path -LiteralPath $registryPath
$currentRegistryValue = $null
if ($currentRegistryExisted) {
    $currentRegistryValue = (Get-Item -LiteralPath $registryPath).GetValue('')
    Remove-Item -LiteralPath $registryPath -Force
    Write-RollbackLog 'suspend-native-host-registration' $registryPath 'ok'
}

$rollbackCompleted = $false
try {
$exe = Join-Path $installRoot 'herdr-web-bridge.exe'
$stopped = Stop-HerdrWebBridgeProcess -ExecutablePath $exe
Write-RollbackLog 'stop-running-bridge' $exe "stopped-$($stopped.StoppedCount)"
$previousExe = $exe + '.previous'
if ($state.hadExecutable -and (Test-Path -LiteralPath $previousExe -PathType Leaf)) {
    Copy-Item -LiteralPath $previousExe -Destination $exe -Force
    Write-RollbackLog 'restore-executable' $exe 'ok'
} elseif (-not $state.hadExecutable -and (Test-Path -LiteralPath $exe -PathType Leaf)) {
    Remove-Item -LiteralPath $exe -Force
    Write-RollbackLog 'remove-new-executable' $exe 'ok'
}

$extension = Join-Path $installRoot 'edge-extension'
$previousExtension = Join-Path $installRoot 'edge-extension.previous'
if (Test-Path -LiteralPath $extension) { Remove-Item -LiteralPath $extension -Recurse -Force }
if ($state.hadExtension -and (Test-Path -LiteralPath $previousExtension -PathType Container)) {
    Copy-Item -LiteralPath $previousExtension -Destination $extension -Recurse -Force
    Write-RollbackLog 'restore-extension' $extension 'ok'
} else {
    Write-RollbackLog 'remove-new-extension' $extension 'ok'
}

foreach ($entry in @(
    @{ Path = (Join-Path $configRoot 'install.json'); Had = [bool]$state.hadInstallConfig },
    @{ Path = (Join-Path $configRoot 'native-host-manifest.json'); Had = [bool]$state.hadNativeManifest }
)) {
    $previous = $entry.Path + '.previous'
    if ($entry.Had -and (Test-Path -LiteralPath $previous -PathType Leaf)) {
        Copy-Item -LiteralPath $previous -Destination $entry.Path -Force
        Write-RollbackLog 'restore-file' $entry.Path 'ok'
    } elseif (-not $entry.Had -and (Test-Path -LiteralPath $entry.Path -PathType Leaf)) {
        Remove-Item -LiteralPath $entry.Path -Force
        Write-RollbackLog 'remove-new-file' $entry.Path 'ok'
    }
}

if ($state.registryExisted) {
    New-Item -Path $registryPath -Force | Out-Null
    Set-Item -LiteralPath $registryPath -Value ([string]$state.registryValue)
    Write-RollbackLog 'restore-registry' $registryPath 'ok'
} elseif (Test-Path -LiteralPath $registryPath) {
    Remove-Item -LiteralPath $registryPath -Force
    Write-RollbackLog 'remove-new-registry' $registryPath 'ok'
}

Move-Item -LiteralPath $statePath -Destination ($statePath + '.rolled-back') -Force
$rollbackCompleted = $true
} finally {
    if (-not $rollbackCompleted) {
        if ($currentRegistryExisted) {
            New-Item -Path $registryPath -Force | Out-Null
            Set-Item -LiteralPath $registryPath -Value ([string]$currentRegistryValue)
            Write-RollbackLog 'restore-registration-after-failure' $registryPath 'ok'
        } elseif (Test-Path -LiteralPath $registryPath) {
            Remove-Item -LiteralPath $registryPath -Force
        }
    }
}
[pscustomobject]@{ RolledBack = $true; BindingsPreserved = $true; ProjectFilesModified = $false }
