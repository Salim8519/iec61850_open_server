@echo off
rem Deploy or update the ScadaBench license server: double-click this file.
rem   deploy-server.cmd                 update with the current settings
rem   deploy-server.cmd -Configure      also answer the PayPal / Mailgun questions on the server
rem   deploy-server.cmd -Price 0.10     change the price (for example a real-money test)
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0license-server\deploy\push.ps1" %*
set "RESULT=%errorlevel%"
if not "%RESULT%"=="0" echo. & echo Deploy failed. Read the message above.
pause
exit /b %RESULT%
