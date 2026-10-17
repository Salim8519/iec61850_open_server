# Save a Cloudflare API token for website uploads (typing is hidden). Uploads then use only
# api.cloudflare.com, so they work even where dash.cloudflare.com is filtered.
# The token is stored only in %LOCALAPPDATA%\ScadaBench-deploy\cloudflare.token (never in the repo).
$ErrorActionPreference = "Stop"
$state = Join-Path $env:LOCALAPPDATA "ScadaBench-deploy"
New-Item -ItemType Directory -Force $state | Out-Null
$secure = Read-Host "Paste the Cloudflare API token (typing is hidden)" -AsSecureString
$token = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)).Trim()
if ($token.Length -lt 30) { throw "That does not look like a Cloudflare API token." }
$check = Invoke-RestMethod -Uri "https://api.cloudflare.com/client/v4/user/tokens/verify" -Headers @{ Authorization = "Bearer $token" }
if (-not $check.success) { throw "Cloudflare did not accept the token." }
$file = Join-Path $state "cloudflare.token"
Set-Content -Path $file -Value $token -NoNewline -Encoding ascii
icacls $file /inheritance:r /grant:r "$($env:USERNAME):F" | Out-Null
Remove-Variable token, secure
Write-Host "Saved. website\deploy.cmd will now upload with this token."
