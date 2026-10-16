@echo off
rem Change the ScadaBench price everywhere:  set-price.cmd 149   (or double-click and type it)
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0license-server\deploy\owner.ps1" -Action price -Value "%~1"
if errorlevel 1 echo. & echo Failed. Read the message above.
pause
