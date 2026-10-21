@echo off
setlocal
set "ROOT=%~dp0"
set "PY=%ROOT%studio\.venv\Scripts\python.exe"
if not exist "%PY%" (
    echo Development Python environment not found: "%PY%"
    exit /b 1
)
rem Go engine like the released app (ADR-0083); add --engine python for the frozen Python engine.
"%PY%" "%ROOT%studio\tools\run_dev_engine.py" %*
set "RESULT=%errorlevel%"
if not "%RESULT%"=="0" pause
exit /b %RESULT%
