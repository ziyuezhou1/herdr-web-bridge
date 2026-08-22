[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ExecutablePath,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[a-p]{32}$')]
    [string]$ExtensionId,

    [string]$ConfigRoot = (Join-Path $env:LOCALAPPDATA 'HerdrWebBridge')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if ($env:OS -ne 'Windows_NT') {
    throw 'Edge Native Messaging registration is supported only on Windows.'
}

$resolvedExecutable = [IO.Path]::GetFullPath($ExecutablePath)
if (-not (Test-Path -LiteralPath $resolvedExecutable -PathType Leaf)) {
    throw "Native host executable not found: $resolvedExecutable"
}

$resolvedConfig = [IO.Path]::GetFullPath($ConfigRoot)
New-Item -ItemType Directory -Path $resolvedConfig -Force | Out-Null
$manifestPath = Join-Path $resolvedConfig 'native-host-manifest.json'
$manifest = [ordered]@{
    name = 'com.herdr_web_bridge'
    description = 'Herdr Web Bridge Native Messaging Host'
    path = $resolvedExecutable
    type = 'stdio'
    allowed_origins = @("chrome-extension://$ExtensionId/")
}
$manifestJson = $manifest | ConvertTo-Json -Depth 4
[IO.File]::WriteAllText($manifestPath, $manifestJson, (New-Object Text.UTF8Encoding($false)))

$registryPath = 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.herdr_web_bridge'
New-Item -Path $registryPath -Force | Out-Null
Set-Item -LiteralPath $registryPath -Value $manifestPath

[pscustomobject]@{
    ManifestPath = $manifestPath
    RegistryPath = $registryPath
    AllowedOrigin = "chrome-extension://$ExtensionId/"
}
