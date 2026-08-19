$ErrorActionPreference = "Stop"

$projectPrefix = Join-Path $PSScriptRoot ""
$nginxExe = Join-Path $PSScriptRoot "nginx-local.exe"
if (-not (Test-Path -LiteralPath $nginxExe)) {
    throw "nginx-local.exe not found under nginx/. Place a local nginx binary there or edit this script."
}

# Stop any previous instance started with this prefix.
& $nginxExe -p $projectPrefix -c "nginx.conf" -s stop 2>$null
Start-Sleep -Seconds 2

& $nginxExe -p $projectPrefix -c "nginx.conf" -t
if ($LASTEXITCODE -ne 0) {
    throw "Nginx configuration test failed."
}

& $nginxExe -p $projectPrefix -c "nginx.conf"
