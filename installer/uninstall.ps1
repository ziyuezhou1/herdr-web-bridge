[CmdletBinding()]
param(
    [string]$InstallRoot = (Join-Path $env:LOCALAPPDATA 'Programs\HerdrWebBridge'),
    [switch]$RemoveBindings,
    [switch]$RemoveGeneratedQuickActions
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'process-control.ps1')

if ($env:OS -ne 'Windows_NT') { throw 'Windows is required.' }

$resolvedInstallRoot = [IO.Path]::GetFullPath($InstallRoot)
$allowedInstallParent = [IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs'))
if (-not $resolvedInstallRoot.StartsWith($allowedInstallParent + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
    throw "InstallRoot must remain under $allowedInstallParent"
}

$configRoot = Join-Path $env:LOCALAPPDATA 'HerdrWebBridge'
$bindingsPath = Join-Path $configRoot 'bindings.json'
$manifestPath = Join-Path $configRoot 'native-host-manifest.json'
$registryPath = 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.herdr_web_bridge'
$logPath = Join-Path $configRoot 'install.log.jsonl'
function Write-UninstallLog([string]$Action, [string]$Target, [string]$Status) {
    New-Item -ItemType Directory -Path $configRoot -Force | Out-Null
    [ordered]@{ time = [DateTime]::UtcNow.ToString('o'); action = $Action; target = $Target; status = $Status } |
        ConvertTo-Json -Compress | Add-Content -LiteralPath $logPath -Encoding utf8
}

if ($RemoveGeneratedQuickActions -and (Test-Path -LiteralPath $bindingsPath -PathType Leaf)) {
    $data = Get-Content -Raw -LiteralPath $bindingsPath | ConvertFrom-Json
    foreach ($binding in $data.bindings) {
        if (-not $binding.quickActionFile) { continue }
        $projectRoot = [IO.Path]::GetFullPath([string]$binding.projectPath)
        $expectedRoot = [IO.Path]::GetFullPath((Join-Path $projectRoot '.herdr-plus\quick-actions'))
        $candidate = [IO.Path]::GetFullPath([string]$binding.quickActionFile)
        if (-not $candidate.StartsWith($expectedRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
            Write-UninstallLog 'preserve-quick-action' $candidate 'outside-expected-root'
            continue
        }
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            $marker = "# generated-by = `"herdr-web-bridge:$($binding.id)`""
            if ((Get-Content -Raw -LiteralPath $candidate).Contains($marker)) {
                Remove-Item -LiteralPath $candidate -Force
                Write-UninstallLog 'remove-quick-action' $candidate 'ok'
            } else {
                Write-UninstallLog 'preserve-quick-action' $candidate 'ownership-marker-missing'
            }
        }
    }
}

if (Test-Path -LiteralPath $registryPath) {
    $registeredManifest = (Get-Item -LiteralPath $registryPath).GetValue('')
    if ([string]::Equals([string]$registeredManifest, $manifestPath, [StringComparison]::OrdinalIgnoreCase)) {
        Remove-Item -LiteralPath $registryPath -Force
        Write-UninstallLog 'unregister-native-host' $registryPath 'ok'
    } else {
        Write-UninstallLog 'preserve-registry-key' $registryPath 'points-elsewhere'
    }
}

foreach ($ownedFile in @($manifestPath, (Join-Path $configRoot 'install.json'))) {
    if (Test-Path -LiteralPath $ownedFile -PathType Leaf) {
        Remove-Item -LiteralPath $ownedFile -Force
        Write-UninstallLog 'remove-file' $ownedFile 'ok'
    }
}

if (Test-Path -LiteralPath $resolvedInstallRoot) {
    $installedExecutable = Join-Path $resolvedInstallRoot 'herdr-web-bridge.exe'
    $stopped = Stop-HerdrWebBridgeProcess -ExecutablePath $installedExecutable
    Write-UninstallLog 'stop-running-bridge' $installedExecutable "stopped-$($stopped.StoppedCount)"
    Remove-Item -LiteralPath $resolvedInstallRoot -Recurse -Force
    Write-UninstallLog 'remove-install-directory' $resolvedInstallRoot 'ok'
}

if ($RemoveBindings) {
    foreach ($bindingFile in @($bindingsPath, ($bindingsPath + '.bak'))) {
        if (Test-Path -LiteralPath $bindingFile -PathType Leaf) {
            Remove-Item -LiteralPath $bindingFile -Force
            Write-UninstallLog 'remove-bindings' $bindingFile 'ok'
        }
    }
} else {
    Write-UninstallLog 'preserve-bindings' $bindingsPath 'default'
}

[pscustomobject]@{
    Uninstalled = $true
    BindingsPreserved = -not $RemoveBindings
    GeneratedQuickActionsRemoved = [bool]$RemoveGeneratedQuickActions
    HerdrModified = $false
    HerdrPlusModified = $false
}
