[CmdletBinding()]
param(
    [ValidatePattern('^[a-p]{32}$')]
    [string]$ExtensionId = 'pphgcjjepkodhghpncncnmikafkdjdjd',

    [string]$InstallRoot = (Join-Path $env:LOCALAPPDATA 'Programs\HerdrWebBridge'),

    [switch]$SkipConnectionTest
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'process-control.ps1')

if ($env:OS -ne 'Windows_NT') {
    throw 'Herdr Web Bridge supports Windows 10/11 only.'
}

$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$resolvedInstallRoot = [IO.Path]::GetFullPath($InstallRoot)
$allowedInstallParent = [IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs'))
if (-not $resolvedInstallRoot.StartsWith($allowedInstallParent + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
    throw "InstallRoot must remain under $allowedInstallParent"
}

$exeCandidates = @(
    (Join-Path $projectRoot 'dist\herdr-web-bridge.exe'),
    (Join-Path $projectRoot 'herdr-web-bridge.exe')
)
$sourceExe = $exeCandidates | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
if (-not $sourceExe) {
    throw 'herdr-web-bridge.exe is missing. Install Go, then run .\scripts\build.ps1 first.'
}

$extensionCandidates = @(
    (Join-Path $projectRoot 'dist\edge-extension'),
    (Join-Path $projectRoot 'edge-extension'),
    (Join-Path $projectRoot 'extension')
)
$sourceExtension = $extensionCandidates | Where-Object { Test-Path -LiteralPath (Join-Path $_ 'manifest.json') -PathType Leaf } | Select-Object -First 1
if (-not $sourceExtension) {
    throw 'Edge extension files were not found.'
}

$extensionManifest = Get-Content -Raw -LiteralPath (Join-Path $sourceExtension 'manifest.json') | ConvertFrom-Json
if (-not $extensionManifest.key) { throw 'Extension manifest is missing its stable public key.' }
& (Join-Path $projectRoot 'scripts\validate-edge-extension.ps1') -ExtensionDirectory $sourceExtension | Out-Host
$publicKey = [Convert]::FromBase64String([string]$extensionManifest.key)
$sha256 = [Security.Cryptography.SHA256]::Create()
try { $digest = $sha256.ComputeHash($publicKey) } finally { $sha256.Dispose() }
$alphabet = 'abcdefghijklmnop'
$derivedExtensionId = -join ($digest[0..15] | ForEach-Object { $alphabet[($_ -shr 4)] + $alphabet[($_ -band 15)] })
if ($derivedExtensionId -ne $ExtensionId) {
    if ($PSBoundParameters.ContainsKey('ExtensionId')) {
        Write-Warning "Manifest key derives $derivedExtensionId, but explicit -ExtensionId is $ExtensionId. Continue only if Edge actually displays the explicit ID."
    } else {
        throw "Stable extension ID verification failed: expected $ExtensionId, derived $derivedExtensionId"
    }
}

$edgeCandidates = @(
    (Join-Path ${env:ProgramFiles(x86)} 'Microsoft\Edge\Application\msedge.exe'),
    (Join-Path $env:ProgramFiles 'Microsoft\Edge\Application\msedge.exe')
)
if (-not ($edgeCandidates | Where-Object { $_ -and (Test-Path -LiteralPath $_ -PathType Leaf) } | Select-Object -First 1)) {
    throw 'Microsoft Edge was not found.'
}

$configRoot = Join-Path $env:LOCALAPPDATA 'HerdrWebBridge'
New-Item -ItemType Directory -Path $configRoot -Force | Out-Null
$logPath = Join-Path $configRoot 'install.log.jsonl'
function Write-InstallLog([string]$Action, [string]$Target, [string]$Status) {
    [ordered]@{ time = [DateTime]::UtcNow.ToString('o'); action = $Action; target = $Target; status = $Status } |
        ConvertTo-Json -Compress | Add-Content -LiteralPath $logPath -Encoding utf8
}

$destinationExe = Join-Path $resolvedInstallRoot 'herdr-web-bridge.exe'
$destinationExtension = Join-Path $resolvedInstallRoot 'edge-extension'
$previousExtension = Join-Path $resolvedInstallRoot 'edge-extension.previous'
$installConfigPath = Join-Path $configRoot 'install.json'
$manifestPath = Join-Path $configRoot 'native-host-manifest.json'
$registryPath = 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.herdr_web_bridge'
$registryExisted = Test-Path -LiteralPath $registryPath
$registryValue = $null
if ($registryExisted) { $registryValue = (Get-Item -LiteralPath $registryPath).GetValue('') }
$rollbackState = [ordered]@{
    schemaVersion = 1
    recordedAt = [DateTime]::UtcNow.ToString('o')
    installRoot = $resolvedInstallRoot
    hadExecutable = (Test-Path -LiteralPath $destinationExe -PathType Leaf)
    hadExtension = (Test-Path -LiteralPath $destinationExtension -PathType Container)
    hadInstallConfig = (Test-Path -LiteralPath $installConfigPath -PathType Leaf)
    hadNativeManifest = (Test-Path -LiteralPath $manifestPath -PathType Leaf)
    registryExisted = $registryExisted
    registryValue = $registryValue
}
$rollbackStatePath = Join-Path $configRoot 'install-state.json'
$rollbackStateJson = $rollbackState | ConvertTo-Json -Depth 4
[IO.File]::WriteAllText($rollbackStatePath, $rollbackStateJson, (New-Object Text.UTF8Encoding($false)))
Write-InstallLog 'write-rollback-state' $rollbackStatePath 'ok'

if (-not (Get-Command herdr -ErrorAction SilentlyContinue)) {
    Write-Warning 'Herdr CLI is not on PATH. Files will be installed, but bridge operations will remain unavailable.'
    Write-InstallLog 'check' 'herdr' 'not-found'
} else {
    Write-InstallLog 'check' 'herdr' 'found'
}

New-Item -ItemType Directory -Path $resolvedInstallRoot -Force | Out-Null
Write-InstallLog 'create-directory' $resolvedInstallRoot 'ok'
$stagedExe = $destinationExe + '.new'
Copy-Item -LiteralPath $sourceExe -Destination $stagedExe -Force
if ((Get-FileHash -LiteralPath $sourceExe -Algorithm SHA256).Hash -ne (Get-FileHash -LiteralPath $stagedExe -Algorithm SHA256).Hash) {
    Remove-Item -LiteralPath $stagedExe -Force -ErrorAction SilentlyContinue
    throw 'Staged executable hash verification failed.'
}
Write-InstallLog 'stage-executable' $stagedExe 'ok'

$registrationSuspended = $false
try {
    if (Test-Path -LiteralPath $registryPath) {
        Remove-Item -LiteralPath $registryPath -Force
        $registrationSuspended = $true
        Write-InstallLog 'suspend-native-host-registration' $registryPath 'ok'
    }

    $stopped = Stop-HerdrWebBridgeProcess -ExecutablePath $destinationExe
    Write-InstallLog 'stop-running-bridge' $destinationExe "stopped-$($stopped.StoppedCount)"

    $previousExe = $destinationExe + '.previous'
    if (Test-Path -LiteralPath $destinationExe -PathType Leaf) {
        Copy-Item -LiteralPath $destinationExe -Destination $previousExe -Force
        Write-InstallLog 'backup' $destinationExe 'ok'
        Remove-Item -LiteralPath $destinationExe -Force
    }
    try {
        Move-Item -LiteralPath $stagedExe -Destination $destinationExe -Force
    } catch {
        if (-not (Test-Path -LiteralPath $destinationExe -PathType Leaf) -and (Test-Path -LiteralPath $previousExe -PathType Leaf)) {
            Copy-Item -LiteralPath $previousExe -Destination $destinationExe -Force
            Write-InstallLog 'restore-executable-after-failure' $destinationExe 'ok'
        }
        throw
    }
    Write-InstallLog 'replace-executable' $destinationExe 'ok'
} finally {
    if (Test-Path -LiteralPath $stagedExe -PathType Leaf) {
        Remove-Item -LiteralPath $stagedExe -Force -ErrorAction SilentlyContinue
    }
    if ($registrationSuspended -and $registryExisted) {
        New-Item -Path $registryPath -Force | Out-Null
        Set-Item -LiteralPath $registryPath -Value ([string]$registryValue)
        Write-InstallLog 'resume-prior-native-host-registration' $registryPath 'ok'
    }
}

if (Test-Path -LiteralPath $destinationExtension) {
    $resolvedExtension = [IO.Path]::GetFullPath($destinationExtension)
    if (-not $resolvedExtension.StartsWith($resolvedInstallRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Refusing to replace an extension directory outside InstallRoot.'
    }
    if (Test-Path -LiteralPath $previousExtension) {
        $resolvedPrevious = [IO.Path]::GetFullPath($previousExtension)
        if (-not $resolvedPrevious.StartsWith($resolvedInstallRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Refusing to replace an extension backup outside InstallRoot.'
        }
        Remove-Item -LiteralPath $resolvedPrevious -Recurse -Force
    }
    Copy-Item -LiteralPath $resolvedExtension -Destination $previousExtension -Recurse -Force
    Write-InstallLog 'backup-extension' $resolvedExtension 'ok'
    Remove-Item -LiteralPath $resolvedExtension -Recurse -Force
    Write-InstallLog 'remove-old-extension' $resolvedExtension 'ok'
}
Copy-Item -LiteralPath $sourceExtension -Destination $destinationExtension -Recurse -Force
if (-not (Test-Path -LiteralPath (Join-Path $destinationExtension 'manifest.json') -PathType Leaf)) {
    throw "Installed extension layout is invalid: manifest.json is not at $destinationExtension"
}
Write-InstallLog 'copy-extension' $destinationExtension 'ok'

$installConfig = [ordered]@{ schemaVersion = 1; extensionId = $ExtensionId; hostName = 'com.herdr_web_bridge'; executablePath = $destinationExe }
if (Test-Path -LiteralPath $installConfigPath -PathType Leaf) {
    Copy-Item -LiteralPath $installConfigPath -Destination ($installConfigPath + '.previous') -Force
    Write-InstallLog 'backup' $installConfigPath 'ok'
}
$installConfigJson = $installConfig | ConvertTo-Json
[IO.File]::WriteAllText($installConfigPath, $installConfigJson, (New-Object Text.UTF8Encoding($false)))
Write-InstallLog 'write-config' $installConfigPath 'ok'

if (Test-Path -LiteralPath $manifestPath -PathType Leaf) {
    Copy-Item -LiteralPath $manifestPath -Destination ($manifestPath + '.previous') -Force
    Write-InstallLog 'backup' $manifestPath 'ok'
}
$registration = & (Join-Path $PSScriptRoot 'register-native-host.ps1') -ExecutablePath $destinationExe -ExtensionId $ExtensionId -ConfigRoot $configRoot
Write-InstallLog 'register-native-host' $registration.ManifestPath 'ok'

$statusExit = 0
& $destinationExe install-status
$statusExit = $LASTEXITCODE
Write-InstallLog 'install-status' $destinationExe $(if ($statusExit -eq 0) { 'ok' } else { "exit-$statusExit" })

if (-not $SkipConnectionTest) {
    & $destinationExe doctor
    $doctorExit = $LASTEXITCODE
    Write-InstallLog 'doctor' $destinationExe $(if ($doctorExit -eq 0) { 'ok' } else { "exit-$doctorExit" })
    if ($doctorExit -ne 0) {
        Write-Warning "End-to-end doctor is not yet green. Load the unpacked extension in Edge, keep Herdr running, then run: & `"$destinationExe`" doctor"
    }
}

Write-Host ''
Write-Host 'Edge: choose Load unpacked and select this exact folder (the folder containing manifest.json):'
Write-Host "  $destinationExtension"
Write-Host 'After Edge loads the extension, run:'
Write-Host "  & `"$destinationExe`" doctor"
Write-Host ''

[pscustomobject]@{
    InstalledExecutable = $destinationExe
    ExtensionDirectory = $destinationExtension
    ExpectedExtensionId = $ExtensionId
    NativeHostManifest = $registration.ManifestPath
    InstallLog = $logPath
    NextStep = "Load ExtensionDirectory in edge://extensions, then run: & `"$destinationExe`" doctor"
}
