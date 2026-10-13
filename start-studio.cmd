@echo off
setlocal
set "ROOT=%~dp0"
set "PY=%ROOT%studio\.venv\Scripts\python.exe"
if not exist "%PY%" (
    echo Development Python environment not found: "%PY%"
    exit /b 1
)
"%PY%" "%ROOT%studio\tools\run_dev.py" %*
set "RESULT=%errorlevel%"
if not "%RESULT%"=="0" pause
exit /b %RESULT%
