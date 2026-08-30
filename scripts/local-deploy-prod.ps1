$ErrorActionPreference = "Stop"
$workspace = Split-Path -Parent $PSScriptRoot
$key = "C:\Users\86186\Key\AIX.pem"
$sshTarget = "ubuntu@56.69.184.203"

Write-Host "=== build backend (linux) ==="
Push-Location $workspace
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o bin/server-linux ./cmd/server

Write-Host "=== build web ==="
Push-Location (Join-Path $workspace "web")
npm run build
Pop-Location

Write-Host "=== build admin ==="
Push-Location (Join-Path $workspace "admin")
$env:NODE_OPTIONS = "--openssl-legacy-provider"
npx vue-cli-service build
Pop-Location

Write-Host "=== pack web dist ==="
tar -czf web-dist-deploy.tar.gz -C web dist

Write-Host "=== upload artifacts ==="
scp -i $key -o StrictHostKeyChecking=no bin/server-linux "${sshTarget}:/tmp/server-linux.new"
scp -i $key -o StrictHostKeyChecking=no web-dist-deploy.tar.gz "${sshTarget}:/tmp/web-dist-deploy.tar.gz"
# scp -r 会把源目录塞进已存在的目标目录，必须先清掉旧目录
ssh -i $key -o StrictHostKeyChecking=no $sshTarget "rm -rf /tmp/admin-dist-new"
scp -i $key -o StrictHostKeyChecking=no -r admin/dist "${sshTarget}:/tmp/admin-dist-new"
scp -i $key -o StrictHostKeyChecking=no scripts/remote-sync-www.sh scripts/cron/aix-chain-jobs.sh scripts/remote-apply-uploaded.sh "${sshTarget}:/tmp/"

Write-Host "=== apply on server ==="
ssh -i $key -o StrictHostKeyChecking=no $sshTarget "chmod +x /tmp/remote-apply-uploaded.sh && bash /tmp/remote-apply-uploaded.sh"

Pop-Location
Write-Host "DEPLOY_DONE"
