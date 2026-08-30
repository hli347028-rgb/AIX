# 合作方转账加款接口的本地联调工具。
#
# 用法：
#   .\scripts\test-transfer-credit.ps1 -Address 0xAbC... -Amount 100.50
#   .\scripts\test-transfer-credit.ps1 -Address 0xAbC... -Amount 100.50 -BreakSign
#
# 密钥从 configs/secrets 读取，脚本不会把密钥打印出来。
param(
    [Parameter(Mandatory = $true)][string]$Address,
    [string]$Amount = "100.50",
    [string]$PartnerId = "AIX10001",
    [string]$BaseUrl = "http://127.0.0.1:9000",
    [string]$KeyFile = "configs/secrets/partner-AIX10001.test.key",
    [string]$Nonce = "",
    [int64]$TimestampMs = 0,
    [switch]$BreakSign,
    [switch]$OmitSign
)

$ErrorActionPreference = "Stop"
$workspace = Split-Path -Parent $PSScriptRoot
$secret = (Get-Content (Join-Path $workspace $KeyFile) -Raw).Trim()

if ($TimestampMs -eq 0) {
    $TimestampMs = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
}
if ([string]::IsNullOrEmpty($Nonce)) {
    $Nonce = -join ((1..12) | ForEach-Object { "0123456789abcdef"[(Get-Random -Max 16)] })
}

# 文档 §4：排除 sign 与空值，字段名按 ASCII 升序，key=value 用 & 连接，值取原样
$fields = [ordered]@{
    address    = $Address
    amount     = $Amount
    nonce      = $Nonce
    partner_id = $PartnerId
    timestamp  = "$TimestampMs"
}
$payload = (($fields.Keys | Sort-Object) | ForEach-Object { "$_=$($fields[$_])" }) -join "&"

$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($secret)
$sig = ($hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($payload)) | ForEach-Object { $_.ToString("x2") }) -join ''

if ($BreakSign) { $sig = "0" * 64 }

$body = [ordered]@{
    address    = $Address
    amount     = $Amount
    partner_id = $PartnerId
    timestamp  = $TimestampMs
    nonce      = $Nonce
}
if (-not $OmitSign) { $body.sign = $sig }

$json = $body | ConvertTo-Json -Compress
Write-Host "signed payload: $payload"
Write-Host "request       : $json"

$tmp = [System.IO.Path]::GetTempFileName()
[System.IO.File]::WriteAllText($tmp, $json, (New-Object System.Text.UTF8Encoding($false)))
try {
    & curl.exe -s -w "`nHTTP_STATUS=%{http_code}`n" -X POST "$BaseUrl/v1/transfer/credit" `
        -H "Content-Type: application/json; charset=utf-8" `
        --data-binary "@$tmp"
}
finally {
    Remove-Item $tmp -ErrorAction SilentlyContinue
}
