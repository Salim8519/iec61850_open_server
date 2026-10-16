# Lazy deploy from Windows: build the Linux program here, upload it, install or update it.
#   deploy-server.cmd                       (repository root; remembers the server)
#   powershell -File license-server\deploy\push.ps1 [-Server user@host] [-Configure] [-Price 0.10]
# Settings that are not secret are applied without questions. Secrets (PayPal, Mailgun) are typed by
# you on the server with -Configure. Machine-local deploy settings live in %LOCALAPPDATA%\ScadaBench-deploy.
param(
    [string]$Server,
    [switch]$Configure,
    [string]$Price,
    [string]$PublicUrl = "https://license.scadabench.com",
    [string]$Listen = "127.0.0.1:18090"
)
$ErrorActionPreference = "Stop"
# Pipe text to ssh without a UTF-8 byte-order mark (a BOM breaks systemd environment files).
$OutputEncoding = New-Object System.Text.UTF8Encoding $false
$source = Split-Path -Parent $PSScriptRoot
$state = Join-Path $env:LOCALAPPDATA "ScadaBench-deploy"
New-Item -ItemType Directory -Force $state | Out-Null
$serverFile = Join-Path $state "server.txt"
$tokenFile = Join-Path $state "tunnel.token"
$key = Join-Path $env:USERPROFILE ".ssh\scadabench_deploy"
if ($Server) { Set-Content -Encoding ascii $serverFile $Server }
elseif (Test-Path $serverFile) { $Server = (Get-Content $serverFile -Raw).Trim() }
else { throw "First run: pass -Server user@host (it is remembered afterwards)." }
$ssh = @("-i", $key, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes", "-o", "ConnectTimeout=15")

Write-Host "== Building the Linux program (amd64 + arm64)"
$bin = Join-Path $source "bin"
New-Item -ItemType Directory -Force $bin | Out-Null
Push-Location $source
try {
    foreach ($arch in "amd64", "arm64") {
        $env:CGO_ENABLED = "0"; $env:GOOS = "linux"; $env:GOARCH = $arch
        go build -trimpath -ldflags "-s -w" -o (Join-Path $bin "license-server-linux-$arch") ./cmd/license-server
        if ($LASTEXITCODE -ne 0) { throw "go build failed for $arch" }
    }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}

Write-Host "== Packing"
$archive = Join-Path $env:TEMP "scadabench-license-server.tar.gz"
if (Test-Path $archive) { Remove-Item $archive -Force }
tar -czf $archive -C $source deploy bin
if ($LASTEXITCODE -ne 0) { throw "tar failed" }

Write-Host "== Uploading to $Server"
scp @ssh $archive "${Server}:/tmp/scadabench-license-server.tar.gz"
if ($LASTEXITCODE -ne 0) { throw "upload failed (is the deploy key authorised on the server?)" }

if (Test-Path $tokenFile) {
    Write-Host "== Sending the tunnel token (kept root-only on the server)"
    $token = (Get-Content $tokenFile -Raw).Trim()
    "TUNNEL_TOKEN=$token" | ssh @ssh $Server "sudo install -d -m 0700 /etc/scadabench && sudo tee /etc/scadabench/tunnel.env >/dev/null && sudo chmod 0600 /etc/scadabench/tunnel.env"
    if ($LASTEXITCODE -ne 0) { throw "could not store the tunnel token" }
}

$setArgs = "--listen $Listen --public-url $PublicUrl --support-email support@scadabench.com"
if ($Price) { $setArgs += " --price $Price" }
$flag = ""
if ($Configure) { $flag = "--configure" }
$remote = "rm -rf /tmp/scadabench-ls && mkdir -p /tmp/scadabench-ls && " +
          "tar -xzf /tmp/scadabench-license-server.tar.gz -C /tmp/scadabench-ls && " +
          "sudo SET_ARGS='$setArgs' bash /tmp/scadabench-ls/deploy/setup.sh $flag && " +
          "rm -rf /tmp/scadabench-ls /tmp/scadabench-license-server.tar.gz"
Write-Host "== Installing on $Server"
ssh -t @ssh $Server $remote
if ($LASTEXITCODE -ne 0) { throw "remote setup failed" }
Remove-Item $archive -Force
Write-Host "Done. Check: https://license.scadabench.com/api/v1/health"
