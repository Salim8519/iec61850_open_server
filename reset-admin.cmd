@echo off
rem Lost the owner password or your phone? This sets a new password, a new authenticator key and
rem new recovery codes, and signs every browser out of the owner page.
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0license-server\deploy\admin.ps1" -Action reset %*
if errorlevel 1 echo. & echo Failed. Read the message above.
pause
