[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$results = [Collections.Generic.List[object]]::new()
$goCommand = Get-Command go -ErrorAction SilentlyContinue
$goExecutable = if ($goCommand) { $goCommand.Source } else {
    @(
        (Join-Path $env:ProgramFiles 'Go\bin\go.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\Go\bin\go.exe')
    ) | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
}
function Invoke-TestStep([string]$Name, [scriptblock]$Action, [bool]$Required = $true) {
    $started = [DateTime]::UtcNow
    & $Action
    $exitCode = $LASTEXITCODE
    if ($null -eq $exitCode) { $exitCode = 0 }
    $results.Add([pscustomobject]@{
        name = $Name
        required = $Required
        exitCode = $exitCode
        status = $(if ($exitCode -eq 0) { 'passed' } else { 'failed' })
        startedAt = $started.ToString('o')
        durationMs = [int]([DateTime]::UtcNow - $started).TotalMilliseconds
    })
}

Push-Location $projectRoot
try {
    if (Get-Command node -ErrorAction SilentlyContinue) {
        Invoke-TestStep 'extension-js-syntax' { & node scripts/check-js.js }
        Invoke-TestStep 'extension-node-tests' { & node --test extension/tests/*.test.js }
    } else {
        $results.Add([pscustomobject]@{ name = 'extension-node-tests'; required = $true; exitCode = -1; status = 'blocked'; details = 'node not found' })
    }

    $pwshExecutable = Join-Path $PSHOME 'pwsh.exe'
    Invoke-TestStep 'powershell-installer-checks' { & $pwshExecutable -NoProfile -File scripts/check-powershell.ps1 }

    if ($goExecutable) {
        Invoke-TestStep 'go-test' { & $goExecutable test ./... }
        Invoke-TestStep 'go-race' { & $goExecutable test -race ./... } $false
        Invoke-TestStep 'go-vet' { & $goExecutable vet ./... }
    } else {
        foreach ($name in @('go-test', 'go-race', 'go-vet')) {
            $results.Add([pscustomobject]@{ name = $name; required = ($name -ne 'go-race'); exitCode = -1; status = 'blocked'; details = 'go not found' })
        }
    }
} finally {
    Pop-Location
}

$distRoot = Join-Path $projectRoot 'dist'
New-Item -ItemType Directory -Path $distRoot -Force | Out-Null
$resultPath = Join-Path $distRoot 'test-results.json'
$results | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $resultPath -Encoding UTF8
$results | Format-Table -AutoSize

$failedRequired = @($results | Where-Object { $_.required -and $_.status -ne 'passed' })
if ($failedRequired.Count -gt 0) { exit 1 }
exit 0
