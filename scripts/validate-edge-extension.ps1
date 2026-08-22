[CmdletBinding()]
param(
    [string]$ExtensionDirectory = (Join-Path ([IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))) 'dist\edge-extension')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if ($env:OS -ne 'Windows_NT') { throw 'Microsoft Edge validation requires Windows.' }

$resolvedExtension = [IO.Path]::GetFullPath($ExtensionDirectory)
$manifestPath = Join-Path $resolvedExtension 'manifest.json'
if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
    throw "Select the extension leaf directory containing manifest.json: $resolvedExtension"
}

$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
if ($manifest.manifest_version -ne 3) { throw 'The Edge extension must use Manifest V3.' }
if (-not $manifest.key) { throw 'The Edge extension manifest is missing its stable public key.' }
[void][Convert]::FromBase64String([string]$manifest.key)

$referencedFiles = [Collections.Generic.List[string]]::new()
if ($manifest.background.service_worker) { $referencedFiles.Add([string]$manifest.background.service_worker) }
if ($manifest.action.default_popup) { $referencedFiles.Add([string]$manifest.action.default_popup) }
if ($manifest.options_page) { $referencedFiles.Add([string]$manifest.options_page) }
foreach ($contentScript in @($manifest.content_scripts)) {
    foreach ($script in @($contentScript.js)) { $referencedFiles.Add([string]$script) }
}
foreach ($relativePath in $referencedFiles) {
    if (-not (Test-Path -LiteralPath (Join-Path $resolvedExtension $relativePath) -PathType Leaf)) {
        throw "Extension manifest references a missing file: $relativePath"
    }
}

$edgeCandidates = @(
    (Join-Path ${env:ProgramFiles(x86)} 'Microsoft\Edge\Application\msedge.exe'),
    (Join-Path $env:ProgramFiles 'Microsoft\Edge\Application\msedge.exe')
)
$edge = $edgeCandidates | Where-Object { $_ -and (Test-Path -LiteralPath $_ -PathType Leaf) } | Select-Object -First 1
if (-not $edge) { throw 'Microsoft Edge was not found.' }

$tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$tempRoot = Join-Path $tempBase ('hwb-edge-validation-' + [Guid]::NewGuid().ToString('N'))
$tempExtension = Join-Path $tempRoot 'edge-extension'
$tempProfile = Join-Path $tempRoot 'profile'
try {
    New-Item -ItemType Directory -Path $tempRoot, $tempProfile | Out-Null
    Copy-Item -LiteralPath $resolvedExtension -Destination $tempExtension -Recurse
    $arguments = @(
        ('--user-data-dir=' + $tempProfile),
        ('--pack-extension=' + $tempExtension),
        '--no-first-run',
        '--no-default-browser-check'
    )
    $process = Start-Process -FilePath $edge -ArgumentList $arguments -PassThru -Wait -WindowStyle Hidden
    $packagePath = $tempExtension + '.crx'
    if (-not (Test-Path -LiteralPath $packagePath -PathType Leaf)) {
        throw "Edge rejected the extension directory (exit code $($process.ExitCode))."
    }

    [pscustomobject]@{
        Status = 'passed'
        ExtensionDirectory = $resolvedExtension
        Manifest = $manifestPath
        EdgeExecutable = $edge
        ReferencedFiles = $referencedFiles.Count
    }
} finally {
    $resolvedTemp = [IO.Path]::GetFullPath($tempRoot)
    if ($resolvedTemp.StartsWith($tempBase, [StringComparison]::OrdinalIgnoreCase) -and
        [IO.Path]::GetFileName($resolvedTemp).StartsWith('hwb-edge-validation-', [StringComparison]::Ordinal)) {
        Remove-Item -LiteralPath $resolvedTemp -Recurse -Force -ErrorAction SilentlyContinue
    }
}
