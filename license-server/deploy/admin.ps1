# Owner page (ADR-0072) from Windows, over the deploy SSH key. Nothing secret is saved on this PC.
#   admin-setup.cmd   first time: password, authenticator key, recovery codes, server settings and
#                     the Cloudflare gateway Worker for https://scadabench.com/<owner path>/
#   reset-admin.cmd   lost the password or the phone: new password, new authenticator key and new
#                     recovery codes; every signed-in browser is signed out
#   powershell -File license-server\deploy\admin.ps1 -Action gateway   new gateway secret + Worker
#   powershell -File license-server\deploy\admin.ps1 -Action status    is it set up and reachable?
# The owner path is remembered in %LOCALAPPDATA%\ScadaBench-deploy\owner-page.json (not in the repo).
param(
    [ValidateSet("setup", "reset", "gateway", "status")][string]$Action = "setup",
    [string]$Path,
    [string]$NotifyEmail
)
$ErrorActionPreference = "Stop"
# Native ssh reads UTF-8 without a byte-order mark.
$OutputEncoding = New-Object System.Text.UTF8Encoding $false
$state = Join-Path $env:LOCALAPPDATA "ScadaBench-deploy"
$server = (Get-Content (Join-Path $state "server.txt") -Raw).Trim()
$key = Join-Path $env:USERPROFILE ".ssh\scadabench_deploy"
$ssh = @("-i", $key, "-o", "IdentitiesOnly=yes", "-o", "BatchMode=yes", "-o", "ConnectTimeout=15")
$admin = "sudo /usr/local/bin/license-server-admin"
$saved = Join-Path $state "owner-page.json"
if (-not $Path -and (Test-Path $saved)) { $Path = (Get-Content $saved -Raw | ConvertFrom-Json).path }
if ($Path -notmatch '^/[a-z0-9][a-z0-9-]{2,39}$') {
    throw "Give the owner page address once, for example: -Path /yourname (lowercase letters, digits or -)"
}
@{ path = $Path } | ConvertTo-Json | Set-Content -Encoding ascii $saved
$pageUrl = "https://scadabench.com$Path/"

function Plain($secure) {
    $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    try { return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer) }
    finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer) }
}

function Read-NewPassword {
    while ($true) {
        $first = Plain (Read-Host "New owner password (at least 8 characters; typing is hidden)" -AsSecureString)
        if ($first.Length -lt 8) { Write-Host "Too short. Use at least 8 characters." -ForegroundColor Yellow; continue }
        $second = Plain (Read-Host "Type the same password again" -AsSecureString)
        if ($first -ne $second) { Write-Host "The two passwords were different. Try again." -ForegroundColor Yellow; continue }
        return $first
    }
}

function Set-Login {
    $password = Read-NewPassword
    Write-Host "== Saving the new login on the server"
    $output = $password | ssh @ssh $server "$admin admin-setup --password-stdin --json"
    $code = $LASTEXITCODE
    Remove-Variable password
    if ($code -ne 0) { throw "the server did not accept the new login" }
    $enrollment = ($output | Select-Object -Last 1) | ConvertFrom-Json
    Write-Host ""
    Write-Host "== Step 1: put ScadaBench in your authenticator app" -ForegroundColor Cyan
    Write-Host "   Google Authenticator: tap + > 'Enter a setup key'."
    Write-Host "   Account name:  ScadaBench owner"
    Write-Host "   Your key:      $($enrollment.secret_grouped)" -ForegroundColor Green
    Write-Host "   Type of key:   Time based"
    Write-Host ""
    Write-Host "== Step 2: write these recovery codes on paper and keep them safe" -ForegroundColor Cyan
    Write-Host "   Each one signs you in once if you lose your phone. They are not saved anywhere else."
    $number = 1
    foreach ($recovery in $enrollment.recovery_codes) { Write-Host ("   {0,2}. {1}" -f $number, $recovery) -ForegroundColor Green; $number++ }
    Write-Host ""
    Write-Host "Every browser that was signed in is now signed out."
}

function Set-Gateway {
    $bytes = New-Object byte[] 32
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    $secret = -join ($bytes | ForEach-Object { $_.ToString("x2") })
    $settings = "$admin set --admin-path $Path --admin-origin https://scadabench.com"
    if ($NotifyEmail) { $settings += " --notify-email $NotifyEmail" }
    Write-Host "== Saving the owner page settings on the server"
    "admin_gateway_secret=$secret" | ssh @ssh $server "$settings --secrets-stdin && sudo /usr/bin/systemctl restart license-server"
    if ($LASTEXITCODE -ne 0) { throw "the server did not accept the owner page settings" }
    $node = Join-Path $env:LOCALAPPDATA "pi-node\current"
    if (Test-Path $node) { $env:PATH = "$node;$env:PATH" }
    Push-Location (Join-Path (Split-Path -Parent $PSScriptRoot) "gateway")
    try {
        Write-Host "== Publishing the Cloudflare gateway for scadabench.com$Path (a browser may ask you to allow Wrangler once)"
        npx --yes wrangler@4 deploy --routes "scadabench.com$Path*"
        if ($LASTEXITCODE -ne 0) { throw "the Cloudflare Worker could not be published" }
        $secret | npx --yes wrangler@4 secret put ADMIN_GATEWAY_SECRET
        if ($LASTEXITCODE -ne 0) { throw "the gateway secret could not be stored in Cloudflare" }
    } finally { Pop-Location }
    Remove-Variable secret
}

function Show-Status {
    ssh @ssh $server "$admin admin-status"
    try {
        $response = Invoke-WebRequest $pageUrl -UseBasicParsing -MaximumRedirection 0 -ErrorAction Stop
        Write-Host "Owner page answers: $($response.StatusCode) at $pageUrl"
    } catch {
        $status = $_.Exception.Response.StatusCode.value__
        if ($status -eq 302 -or $status -eq 303) { Write-Host "Owner page is behind Cloudflare Access (sign-in code by email): $pageUrl" }
        else { Write-Host "Owner page answered $status at $pageUrl" -ForegroundColor Yellow }
    }
}

switch ($Action) {
    "setup" { Set-Gateway; Set-Login }
    "reset" { Set-Login }
    "gateway" { Set-Gateway }
    "status" { Show-Status; return }
}
Write-Host ""
Write-Host "Open $pageUrl on your phone or PC and sign in." -ForegroundColor Cyan
if ($Action -eq "setup") {
    Write-Host "Extra lock (recommended, once): follow docs\specs\owner-page.md, section Cloudflare Access, so Cloudflare"
    Write-Host "emails you a code before anyone even sees the sign-in page."
}
