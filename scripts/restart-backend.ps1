$ErrorActionPreference = "Stop"

$workspace = Split-Path -Parent $PSScriptRoot
$serverExe = Join-Path $workspace "bin\server.exe"
$expectedPath = (Resolve-Path -LiteralPath $serverExe).Path

$listenerPids = Get-NetTCPConnection -State Listen |
    Where-Object { $_.LocalPort -in 9000, 9100 } |
    Select-Object -ExpandProperty OwningProcess -Unique

foreach ($listenerPid in $listenerPids) {
    $process = Get-Process -Id $listenerPid -ErrorAction SilentlyContinue
    if ($process -and $process.Path -eq $expectedPath) {
        Stop-Process -Id $listenerPid -Force
    }
}

Start-Sleep -Seconds 2
$logsDir = Join-Path $workspace "logs"
New-Item -ItemType Directory -Force -Path $logsDir | Out-Null
Start-Process -FilePath $serverExe `
    -ArgumentList "-conf", "configs" `
    -WorkingDirectory $workspace `
    -RedirectStandardOutput (Join-Path $logsDir "backend.out.log") `
    -RedirectStandardError (Join-Path $logsDir "backend.err.log") `
    -WindowStyle Hidden
