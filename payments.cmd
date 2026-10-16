@echo off
rem Show every recorded ScadaBench payment and refund, with buyer details.
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0license-server\deploy\owner.ps1" -Action payments
if errorlevel 1 echo. & echo Failed. Read the message above.
pause
