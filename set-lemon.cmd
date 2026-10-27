@echo off
rem Connect Lemon Squeezy to the license server (typing of the API key is hidden).
rem   set-lemon.cmd          test mode
rem   set-lemon.cmd -Live    real sales
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0license-server\deploy\lemon.ps1" %*
set "RESULT=%errorlevel%"
if not "%RESULT%"=="0" echo. & echo Failed. Read the message above.
pause
exit /b %RESULT%
