@echo off
rem Save the Cloudflare API token used by website\deploy.cmd (typing is hidden).
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0set-cloudflare-token.ps1"
pause
