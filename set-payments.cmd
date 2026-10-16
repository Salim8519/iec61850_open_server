@echo off
rem Paste your PayPal and Mailgun secrets into the license server (typing is hidden).
rem   set-payments.cmd              asks for mode, IDs and secrets (IDs are remembered)
rem   set-payments.cmd -Mode live   switch to real payments
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0license-server\deploy\secrets.ps1" %*
set "RESULT=%errorlevel%"
if not "%RESULT%"=="0" echo. & echo Failed. Read the message above.
pause
exit /b %RESULT%
