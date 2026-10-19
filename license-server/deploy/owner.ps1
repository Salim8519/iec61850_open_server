# Everyday owner actions on the live license server, over the deploy SSH key.
#   set-price.cmd 149        change the price everywhere (buy page, website, PayPal) in seconds
#   payments.cmd             list every recorded payment and refund with buyer details
#   powershell -File license-server\deploy\owner.ps1 -Action notify -Value you@example.com
#   ... -Action release      tell installed apps about the version on scadabench.com (website\deploy.cmd runs it)
param(
    [ValidateSet("price", "payments", "notify", "status", "release")][string]$Action = "status",
    [string]$Value
)
$ErrorActionPreference = "Stop"
$state = Join-Path $env:LOCALAPPDATA "ScadaBench-deploy"
$server = (Get-Content (Join-Path $state "server.txt") -Raw).Trim()
$key = Join-Path $env:USERPROFILE ".ssh\scadabench_deploy"
$ssh = @("-i", $key, "-o", "IdentitiesOnly=yes", "-o", "BatchMode=yes", "-o", "ConnectTimeout=15")
$admin = "sudo /usr/local/bin/license-server-admin"
# Native ssh reads UTF-8; Windows PowerShell would otherwise pipe ASCII and lose "→" in notes.
$OutputEncoding = New-Object System.Text.UTF8Encoding $false

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
    "release" {
        # Sent as base64 on the command line: Windows PowerShell puts a BOM in front of piped text.
        $record = (& (Join-Path $PSScriptRoot "release-record.ps1")) -join ""
        $encoded = [Convert]::ToBase64String((New-Object System.Text.UTF8Encoding $false).GetBytes($record))
        $remote = "echo $encoded | base64 -d | $admin release && sudo /usr/bin/systemctl restart license-server"
    }
}
ssh @ssh $server $remote
if ($LASTEXITCODE -ne 0) { throw "the license server did not accept the command" }
if ($Action -eq "price") {
    Start-Sleep 2
    $product = Invoke-RestMethod "https://license.scadabench.com/api/v1/product"
    Write-Host "Live price is now $($product.amount). The buy page and scadabench.com show it within a minute."
}
if ($Action -eq "release") {
    Start-Sleep 2
    $live = Invoke-RestMethod "https://license.scadabench.com/api/v1/release" -Headers @{ "Cache-Control" = "no-cache" }
    $version = ($live.payload | ConvertFrom-Json).version
    Write-Host "Installed apps now see version $version as the newest (they check about once a day)."
}
