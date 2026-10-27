# Connect the Lemon Squeezy store to the license server (ADR-0126) in one go:
#   set-lemon.cmd          test mode (use a Lemon Squeezy API key made while "Test mode" is on)
#   set-lemon.cmd -Live    real sales (API key made with Test mode off)
# The API key is typed hidden, used only on this PC to read the store and create the webhook,
# and never saved. A new random webhook signing secret goes to the server over SSH standard input.
param([switch]$Live)
$ErrorActionPreference = "Stop"
$OutputEncoding = New-Object System.Text.UTF8Encoding $false
$state = Join-Path $env:LOCALAPPDATA "ScadaBench-deploy"
$server = (Get-Content (Join-Path $state "server.txt") -Raw).Trim()
$key = Join-Path $env:USERPROFILE ".ssh\scadabench_deploy"
$ssh = @("-i", $key, "-o", "IdentitiesOnly=yes", "-o", "BatchMode=yes", "-o", "ConnectTimeout=15")
$hook = "https://license.scadabench.com/api/v1/webhooks/lemonsqueezy"
$mode = if ($Live) { "LIVE (real money)" } else { "TEST" }

Write-Host "Lemon Squeezy setup, $mode mode."
Write-Host "Make the key at app.lemonsqueezy.com > Settings > API > + (Test mode switch $(if ($Live) {'OFF'} else {'ON'}))."
$secure = Read-Host "Paste the Lemon Squeezy API key (typing is hidden)" -AsSecureString
$apiKey = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)).Trim()
if (-not $apiKey) { throw "No API key was pasted." }
$headers = @{ Authorization = "Bearer $apiKey"; Accept = "application/vnd.api+json"; "Content-Type" = "application/vnd.api+json" }
function Api($method, $path, $body) {
    $arguments = @{ Method = $method; Uri = "https://api.lemonsqueezy.com/v1/$path"; Headers = $headers; UseBasicParsing = $true }
    if ($body) { $arguments.Body = ($body | ConvertTo-Json -Depth 8 -Compress) }
    return Invoke-RestMethod @arguments
}

$store = (Api GET "stores").data | Select-Object -First 1
if (-not $store) { throw "This API key sees no store." }
$product = (Api GET "products?filter[store_id]=$($store.id)").data |
    Where-Object { $_.attributes.name -like "*ScadaBench*" -and $_.attributes.status -eq "published" } | Select-Object -First 1
if (-not $product) { throw "No published product with ScadaBench in its name in store $($store.attributes.name) ($mode mode)." }
$variant = (Api GET "variants?filter[product_id]=$($product.id)").data |
    Where-Object { $_.attributes.status -ne "draft" } | Select-Object -First 1
$price = "{0:0.00}" -f ($variant.attributes.price / 100)
Write-Host "Store: $($store.attributes.name) | Product: $($product.attributes.name) | Variant $($variant.id) | Price $price $($store.attributes.currency)"

$bytes = New-Object byte[] 32
[System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
$secret = -join ($bytes | ForEach-Object { "{0:x2}" -f $_ })
$existing = (Api GET "webhooks?filter[store_id]=$($store.id)").data | Where-Object { $_.attributes.url -eq $hook }
$attributes = @{ url = $hook; events = @("order_created", "order_refunded"); secret = $secret }
if ($existing) {
    Api PATCH "webhooks/$($existing.id)" @{ data = @{ type = "webhooks"; id = "$($existing.id)"; attributes = $attributes } } | Out-Null
    Write-Host "Webhook updated with a new signing secret."
} else {
    $attributes.test_mode = (-not $Live)
    Api POST "webhooks" @{ data = @{ type = "webhooks"; attributes = $attributes
        relationships = @{ store = @{ data = @{ type = "stores"; id = "$($store.id)" } } } } } | Out-Null
    Write-Host "Webhook created."
}
Remove-Variable apiKey, headers

$testFlag = if ($Live) { "off" } else { "on" }
$remote = "sudo /usr/local/bin/license-server-admin set --lemon-checkout-url $($product.attributes.buy_now_url) " +
          "--lemon-variant-id $($variant.id) --lemon-test-mode $testFlag --price $price --secrets-stdin " +
          "&& sudo /usr/bin/systemctl restart license-server"
Write-Host "== Sending to $server"
"lemon_signing_secret=$secret" | ssh @ssh $server $remote
if ($LASTEXITCODE -ne 0) { throw "could not update the license server" }
Remove-Variable secret
Start-Sleep 3
$health = Invoke-RestMethod "https://license.scadabench.com/api/v1/health"
Write-Host "Lemon Squeezy checkout ready: $($health.lemon). Buy page: https://license.scadabench.com/buy"
