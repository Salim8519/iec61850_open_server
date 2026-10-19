@echo off
rem Set up the ScadaBench owner page (https://scadabench.com/<owner path>/): password, authenticator,
rem recovery codes and the Cloudflare gateway. Run deploy-server.cmd first after code changes.
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0license-server\deploy\admin.ps1" -Action setup %*
if errorlevel 1 echo. & echo Failed. Read the message above.
pause
