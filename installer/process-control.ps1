Set-StrictMode -Version Latest

function Stop-HerdrWebBridgeProcess {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$ExecutablePath,

        [ValidateRange(1, 30)]
        [int]$TimeoutSeconds = 5
    )

    $resolvedExecutable = [IO.Path]::GetFullPath($ExecutablePath)
    $matches = @(
        Get-Process -Name 'herdr-web-bridge' -ErrorAction SilentlyContinue | ForEach-Object {
            $processPath = $null
            try { $processPath = $_.Path } catch { $processPath = $null }
            if ($processPath -and [string]::Equals($processPath, $resolvedExecutable, [StringComparison]::OrdinalIgnoreCase)) {
                $_
            }
        }
    )

    foreach ($process in $matches) {
        Stop-Process -Id $process.Id -Force -ErrorAction Stop
    }

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    $remaining = @($matches)
    while ($remaining.Count -gt 0 -and [DateTime]::UtcNow -lt $deadline) {
        Start-Sleep -Milliseconds 100
        $remaining = @(
            foreach ($process in $remaining) {
                if (Get-Process -Id $process.Id -ErrorAction SilentlyContinue) { $process }
            }
        )
    }
    if ($remaining.Count -gt 0) {
        $remainingIds = @($remaining | ForEach-Object { $_.Id })
        throw "Timed out stopping Herdr Web Bridge process IDs: $($remainingIds -join ', ')"
    }

    [pscustomobject]@{
        ExecutablePath = $resolvedExecutable
        StoppedCount = $matches.Count
        ProcessIds = @($matches | ForEach-Object { $_.Id })
    }
}
