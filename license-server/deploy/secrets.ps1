# Send PayPal and Mailgun settings to the license server. Secrets are typed hidden, sent over SSH
# standard input to "license-server set --secrets-stdin", and never saved on this PC or in the repo.
#   set-payments.cmd                      (repository root)
#   powershell -File license-server\deploy\secrets.ps1 [-Mode live] [-ClientId ...] [-WebhookId ...]
param(
    [ValidateSet("sandbox", "live")][string]$Mode,
    [string]$ClientId,
    [string]$WebhookId,
    [switch]$SkipMailgun
)
$ErrorActionPreference = "Stop"
$OutputEncoding = New-Object System.Text.UTF8Encoding $false
$state = Join-Path $env:LOCALAPPDATA "ScadaBench-deploy"
$saved = Join-Path $state "payments.json"
$server = (Get-Content (Join-Path $state "server.txt") -Raw).Trim()
$key = Join-Path $env:USERPROFILE ".ssh\scadabench_deploy"
$ssh = @("-i", $key, "-o", "IdentitiesOnly=yes", "-o", "BatchMode=yes", "-o", "ConnectTimeout=15")
$known = if (Test-Path $saved) { Get-Content $saved -Raw | ConvertFrom-Json } else { [pscustomobject]@{} }

function Ask($label, $current) {
    $answer = Read-Host "$label [$current]"
    if ($answer) { return $answer.Trim() } else { return $current }
}
function AskSecret($label) {
    $secure = Read-Host "$label (typing is hidden; Enter to keep the current one)" -AsSecureString
    $plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure))
    return $plain.Trim()
}

if (-not $Mode) { $Mode = Ask "PayPal mode (sandbox or live)" ($(if ($known.mode) { $known.mode } else { "sandbox" })) }
if (-not $ClientId) { $ClientId = Ask "PayPal Client ID ($Mode)" $known."client_id_$Mode" }
if (-not $WebhookId) { $WebhookId = Ask "PayPal Webhook ID ($Mode)" $known."webhook_id_$Mode" }
if (-not $ClientId -or -not $WebhookId) {
    throw "PayPal $Mode Client ID and Webhook ID are needed first (developer.paypal.com > Apps & Credentials, $Mode tab)"
}
$paypalSecret = AskSecret "PayPal Secret ($Mode) from developer.paypal.com > Apps & Credentials"
$mailgunKey = ""
if (-not $SkipMailgun) { $mailgunKey = AskSecret "Mailgun Sending API key for mg.scadabench.com" }

$known | Add-Member -Force NoteProperty mode $Mode
$known | Add-Member -Force NoteProperty "client_id_$Mode" $ClientId
$known | Add-Member -Force NoteProperty "webhook_id_$Mode" $WebhookId
$known | ConvertTo-Json | Set-Content -Encoding utf8 $saved

$lines = @()
if ($paypalSecret) { $lines += "paypal_client_secret=$paypalSecret" }
if ($mailgunKey) { $lines += "mailgun_api_key=$mailgunKey" }
# Single quotes: Windows PowerShell drops inner double quotes, and bash would read <...> as a redirect.
$from = "'ScadaBench <licenses@mg.scadabench.com>'"
$remote = "sudo /usr/local/bin/license-server-admin set --paypal-mode $Mode --paypal-client-id $ClientId " +
          "--paypal-webhook-id $WebhookId --mailgun-domain mg.scadabench.com --mailgun-region us " +
          "--mailgun-from $from --secrets-stdin && sudo /usr/bin/systemctl restart license-server"
Write-Host "== Sending to $server"
($lines -join "`n") | ssh @ssh $server $remote
if ($LASTEXITCODE -ne 0) { throw "could not update the license server" }
Remove-Variable paypalSecret, mailgunKey, lines
Start-Sleep 3
$health = Invoke-RestMethod "https://license.scadabench.com/api/v1/health"
Write-Host "PayPal checkout ready: $($health.checkout). Test page: https://license.scadabench.com/buy"
