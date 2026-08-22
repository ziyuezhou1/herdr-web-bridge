[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$distRoot = Join-Path $projectRoot 'dist'
$executable = Join-Path $distRoot 'herdr-web-bridge.exe'
$extension = Join-Path $distRoot 'edge-extension'
if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) { throw 'dist\herdr-web-bridge.exe is missing. Run scripts\build.ps1 first.' }
if (-not (Test-Path -LiteralPath (Join-Path $extension 'manifest.json') -PathType Leaf)) { throw 'dist\edge-extension is missing.' }
& (Join-Path $PSScriptRoot 'validate-edge-extension.ps1') -ExtensionDirectory $extension | Out-Host

$requiredDocs = @('README.md', 'docs\QUICKSTART_WINDOWS.md', 'docs\SECURITY.md', 'docs\KNOWN_LIMITATIONS.md', 'docs\TEST_REPORT.md')
foreach ($relative in $requiredDocs) {
    if (-not (Test-Path -LiteralPath (Join-Path $projectRoot $relative) -PathType Leaf)) { throw "Required release file missing: $relative" }
}

$staging = Join-Path ([IO.Path]::GetTempPath()) ("HerdrWebBridge-" + [Guid]::NewGuid().ToString('N'))
$stagingRoot = [IO.Path]::GetFullPath($staging)
$tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
if (-not $stagingRoot.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Invalid temporary staging path.' }
New-Item -ItemType Directory -Path $stagingRoot -Force | Out-Null
try {
    Copy-Item -LiteralPath $executable -Destination $stagingRoot
    Copy-Item -LiteralPath $extension -Destination (Join-Path $stagingRoot 'edge-extension') -Recurse
    Copy-Item -LiteralPath (Join-Path $projectRoot 'installer') -Destination (Join-Path $stagingRoot 'installer') -Recurse
    New-Item -ItemType Directory -Path (Join-Path $stagingRoot 'scripts') | Out-Null
    Copy-Item -LiteralPath (Join-Path $projectRoot 'scripts\validate-edge-extension.ps1') -Destination (Join-Path $stagingRoot 'scripts')
    Copy-Item -LiteralPath (Join-Path $projectRoot 'README.md') -Destination $stagingRoot
    New-Item -ItemType Directory -Path (Join-Path $stagingRoot 'docs') | Out-Null
    foreach ($doc in @('QUICKSTART_WINDOWS.md', 'SECURITY.md', 'KNOWN_LIMITATIONS.md', 'TEST_REPORT.md')) {
        Copy-Item -LiteralPath (Join-Path $projectRoot "docs\$doc") -Destination (Join-Path $stagingRoot 'docs')
    }

    $releaseRoot = Join-Path $projectRoot 'release'
    New-Item -ItemType Directory -Path $releaseRoot -Force | Out-Null
    $zipPath = Join-Path $releaseRoot 'Herdr_Web_Bridge_Windows_MVP.zip'
    if (Test-Path -LiteralPath $zipPath) { Remove-Item -LiteralPath $zipPath -Force }
    Compress-Archive -Path (Join-Path $stagingRoot '*') -DestinationPath $zipPath -CompressionLevel Optimal
    Get-FileHash -Algorithm SHA256 -LiteralPath $zipPath | Select-Object Path, Hash
} finally {
    if (Test-Path -LiteralPath $stagingRoot) { Remove-Item -LiteralPath $stagingRoot -Recurse -Force }
}
