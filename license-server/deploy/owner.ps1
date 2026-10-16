# Everyday owner actions on the live license server, over the deploy SSH key.
#   set-price.cmd 149        change the price everywhere (buy page, website, PayPal) in seconds
#   payments.cmd             list every recorded payment and refund with buyer details
#   powershell -File license-server\deploy\owner.ps1 -Action notify -Value you@example.com
param(
    [ValidateSet("price", "payments", "notify", "status")][string]$Action = "status",
    [string]$Value
)
$ErrorActionPreference = "Stop"
$state = Join-Path $env:LOCALAPPDATA "ScadaBench-deploy"
$server = (Get-Content (Join-Path $state "server.txt") -Raw).Trim()
$key = Join-Path $env:USERPROFILE ".ssh\scadabench_deploy"
$ssh = @("-i", $key, "-o", "IdentitiesOnly=yes", "-o", "BatchMode=yes", "-o", "ConnectTimeout=15")
$admin = "sudo /usr/local/bin/license-server-admin"

switch ($Action) {
    "price" {
        if (-not $Value) { $Value = Read-Host "New price in USD (for example 149 or 0.10)" }
        if ($Value -notmatch '^\$?[0-9]+(\.[0-9]{1,2})?$') { throw "Write the price as a number, for example 149 or 149.00" }
        $remote = "$admin set --price $($Value.TrimStart('$')) && sudo /usr/bin/systemctl restart license-server"
    }
    "notify" {
        if ($Value -notmatch '^[^@\s]+@[^@\s]+\.[^@\s]+$') { throw "Give the email address that should get sale alerts" }
        $remote = "$admin set --notify-email $Value && sudo /usr/bin/systemctl restart license-server"
    }
    "payments" { $remote = "$admin payments" }
    "status" { $remote = "$admin status" }
}
ssh @ssh $server $remote
if ($LASTEXITCODE -ne 0) { throw "the license server did not accept the command" }
if ($Action -eq "price") {
    Start-Sleep 2
    $product = Invoke-RestMethod "https://license.scadabench.com/api/v1/product"
    Write-Host "Live price is now $($product.amount). The buy page and scadabench.com show it within a minute."
}
