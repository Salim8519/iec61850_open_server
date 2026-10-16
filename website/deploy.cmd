@echo off
rem Publish scadabench.com: double-click, or run from any terminal.
rem First time only: a browser tab asks you to allow Wrangler; click Allow.
setlocal
cd /d "%~dp0"
call npm ci --no-audit --no-fund || goto :failed
call npm run build || goto :failed
rem Ship the newest real installer (never the sandbox build). Pages allows files up to 25 MiB.
for /f "delims=" %%F in ('dir /b /o-d "..\studio\dist-protected\ScadaBench-Setup-*.exe" 2^>nul') do (
  if not exist dist\download mkdir dist\download
  copy /y "..\studio\dist-protected\%%F" dist\download\ScadaBench-Setup.exe >nul
  echo Installer included: %%F
  goto :installer_done
)
echo No release installer found in studio\dist-protected; the download link will not work yet.
:installer_done
rem Tutorial videos rendered by marketing\videos\make.py (MP4 and poster per lesson).
if not exist "..\marketing\videos\out\*.mp4" goto :no_videos
if not exist dist\videos mkdir dist\videos
copy /y "..\marketing\videos\out\*.mp4" dist\videos\ >nul
copy /y "..\marketing\videos\out\*.jpg" dist\videos\ >nul
echo Tutorial videos included.
goto :videos_done
:no_videos
echo No tutorial videos found; run marketing\videos\make.py first.
:videos_done
call npx --yes wrangler@4 whoami >nul 2>&1 || call npx --yes wrangler@4 login || goto :failed
call npx --yes wrangler@4 pages deploy dist --project-name scadabench --branch main --commit-dirty=true || goto :failed
echo.
echo Published. Live at https://scadabench.com (preview: https://scadabench.pages.dev)
pause
exit /b 0
:failed
echo.
echo Publishing failed. Read the message above, fix it, then run deploy.cmd again.
pause
exit /b 1
