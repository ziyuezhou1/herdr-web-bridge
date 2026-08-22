[CmdletBinding()]
param(
    [switch]$SkipTests,
    [switch]$ExtensionOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if ($env:OS -ne 'Windows_NT') { throw 'Windows is required.' }
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$distRoot = Join-Path $projectRoot 'dist'
$extensionOutput = Join-Path $distRoot 'edge-extension'
$goCommand = Get-Command go -ErrorAction SilentlyContinue
$goExecutable = if ($goCommand) { $goCommand.Source } else {
    @(
        (Join-Path $env:ProgramFiles 'Go\bin\go.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\Go\bin\go.exe')
    ) | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
}

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    throw 'Node.js is not installed or not on PATH.'
}

if ($ExtensionOnly -and -not $SkipTests) {
    Push-Location $projectRoot
    try {
        & node scripts/check-js.js
        if ($LASTEXITCODE -ne 0) { throw 'Extension syntax checks failed.' }
        & node --test extension/tests/*.test.js
        if ($LASTEXITCODE -ne 0) { throw 'Extension tests failed.' }
    } finally {
        Pop-Location
    }
} elseif (-not $SkipTests) {
    & (Join-Path $PSScriptRoot 'test.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed; build stopped.' }
}

New-Item -ItemType Directory -Path $distRoot -Force | Out-Null
if (Test-Path -LiteralPath $extensionOutput) {
    $resolvedOutput = [IO.Path]::GetFullPath($extensionOutput)
    if (-not $resolvedOutput.StartsWith([IO.Path]::GetFullPath($distRoot) + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Refusing to replace an extension output outside dist.'
    }
    Remove-Item -LiteralPath $resolvedOutput -Recurse -Force
}
New-Item -ItemType Directory -Path $extensionOutput -Force | Out-Null
foreach ($item in @('manifest.json', 'popup.html', 'options.html', 'src', 'icons')) {
    Copy-Item -LiteralPath (Join-Path $projectRoot "extension\$item") -Destination $extensionOutput -Recurse -Force
}

if (-not $ExtensionOnly) {
    if (-not $goExecutable) {
        throw 'Go is not installed or not on PATH. The Edge extension was built, but no executable was produced.'
    }
    Push-Location $projectRoot
    try {
        & $goExecutable build -trimpath -ldflags '-s -w' -o (Join-Path $distRoot 'herdr-web-bridge.exe') .\cmd\herdr-web-bridge
        if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
    } finally {
        Pop-Location
    }
}

[pscustomobject]@{
    Executable = $(if ($ExtensionOnly) { $null } else { Join-Path $distRoot 'herdr-web-bridge.exe' })
    EdgeExtension = $extensionOutput
}
