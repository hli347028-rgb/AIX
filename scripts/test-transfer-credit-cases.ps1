# 逐个验证对接文档 §6 的错误码分支（本地开发库）。
param(
    [string]$GoodAddress = "0xFFc948606F1fB3a3B889973e2De7e6740e47CCa7",
    [string]$FrozenAddress = "",
    [string]$ReplayNonce = ""
)

$ErrorActionPreference = "Continue"
$script:runner = Join-Path $PSScriptRoot "test-transfer-credit.ps1"

function Invoke-Case {
    param([string]$Name, [string]$ExpectCode, [hashtable]$CaseArgs)
    # 合作方限速 10/s，用间隔避免把正常用例误判成 5001
    Start-Sleep -Milliseconds 400
    $out = & $script:runner @CaseArgs 2>&1 | Out-String
    $code = if ($out -match '"code":"(\d{4})"') { $Matches[1] } else { "none" }
    $status = if ($out -match 'HTTP_STATUS=(\d+)') { $Matches[1] } else { "?" }
    $ok = if ($code -eq $ExpectCode) { "PASS" } else { "FAIL" }
    "{0,-6} {1,-28} expect={2} got={3} http={4}" -f $ok, $Name, $ExpectCode, $code, $status
}

$results = @()
$results += Invoke-Case -Name "bad signature" -ExpectCode "1001" -CaseArgs @{ Address = $GoodAddress; BreakSign = $true }
$results += Invoke-Case -Name "unknown partner" -ExpectCode "1002" -CaseArgs @{ Address = $GoodAddress; PartnerId = "AIX99999" }
$results += Invoke-Case -Name "stale timestamp" -ExpectCode "1003" -CaseArgs @{ Address = $GoodAddress; TimestampMs = ([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() - 600000) }
if ($ReplayNonce) {
    $results += Invoke-Case -Name "replayed nonce" -ExpectCode "1004" -CaseArgs @{ Address = $GoodAddress; Nonce = $ReplayNonce }
}
$results += Invoke-Case -Name "missing sign" -ExpectCode "3002" -CaseArgs @{ Address = $GoodAddress; OmitSign = $true }
$results += Invoke-Case -Name "bad address format" -ExpectCode "3003" -CaseArgs @{ Address = "0x1234" }
$results += Invoke-Case -Name "below minimum" -ExpectCode "2003" -CaseArgs @{ Address = $GoodAddress; Amount = "1" }
$results += Invoke-Case -Name "above per-tx limit" -ExpectCode "2004" -CaseArgs @{ Address = $GoodAddress; Amount = "200000" }
$results += Invoke-Case -Name "address not found" -ExpectCode "2001" -CaseArgs @{ Address = "0x1111111111111111111111111111111111111111" }
if ($FrozenAddress) {
    $results += Invoke-Case -Name "frozen account" -ExpectCode "2002" -CaseArgs @{ Address = $FrozenAddress }
}

Write-Host ""
Write-Host "=== transfer credit error-code matrix ==="
$results | ForEach-Object { Write-Host $_ }
