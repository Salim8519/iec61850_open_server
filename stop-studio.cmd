@echo off
setlocal
set "ROOT=%~dp0"
set "PY=%ROOT%studio\.venv\Scripts\python.exe"
if not exist "%PY%" set "PY=python"
rem Stops every ScadaBench process (app, engine, workers, dev launchers); add --dry-run to only list them.
"%PY%" "%ROOT%studio\tools\stop_all.py" %*
set "RESULT=%errorlevel%"
if not "%RESULT%"=="0" pause
exit /b %RESULT%
