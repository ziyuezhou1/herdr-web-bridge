[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$files = @(
    Get-ChildItem -LiteralPath (Join-Path $projectRoot 'installer') -Filter '*.ps1' -File
    Get-ChildItem -LiteralPath (Join-Path $projectRoot 'scripts') -Filter '*.ps1' -File
)
$parseErrors = [Collections.Generic.List[object]]::new()
foreach ($file in $files) {
    $tokens = $null
    $errors = $null
    [void][Management.Automation.Language.Parser]::ParseFile($file.FullName, [ref]$tokens, [ref]$errors)
    foreach ($error in @($errors)) { $parseErrors.Add($error) }
}
if ($parseErrors.Count -gt 0) {
    throw (($parseErrors | ForEach-Object { $_.Message }) -join [Environment]::NewLine)
}

. (Join-Path $projectRoot 'installer\process-control.ps1')
$probePath = Join-Path $projectRoot 'does-not-exist\herdr-web-bridge.exe'
$probe = Stop-HerdrWebBridgeProcess -ExecutablePath $probePath
if ($probe.StoppedCount -ne 0 -or $probe.ProcessIds.Count -ne 0) {
    throw 'Process control matched a process whose executable path was not the requested path.'
}

Write-Output "PowerShell validation passed for $($files.Count) scripts."
