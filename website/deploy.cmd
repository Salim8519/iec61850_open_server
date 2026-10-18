@echo off
rem Publish scadabench.com: double-click, or run from any terminal.
rem First time only: a browser tab asks you to allow Wrangler; click Allow.
setlocal
cd /d "%~dp0"
call npm ci --no-audit --no-fund || goto :failed
call npm run build || goto :failed
rem Ship the newest real installer (never the sandbox build) under its versioned name; links to the
rem old unversioned names redirect to it. Pages allows files up to 25 MiB.
if exist dist\_redirects del dist\_redirects
for /f "delims=" %%F in ('dir /b /o-d "..\studio\dist-protected\ScadaBench-Setup-*.exe" 2^>nul') do (
  if not exist dist\download mkdir dist\download
  copy /y "..\studio\dist-protected\%%F" "dist\download\%%F" >nul
  findstr /c:"/download/%%F" dist\download\index.html >nul || (echo The download page does not link %%F: set the version in src\data\site.ts. & goto :failed)
  >>dist\_redirects echo /download/ScadaBench-Setup.exe /download/%%F 302
  echo Installer included: %%F
  goto :installer_done
)
echo No release installer found in studio\dist-protected; the download link will not work yet.
:installer_done
rem With a saved API token (set-cloudflare-token.cmd) uploads use only api.cloudflare.com.
set "TOKEN_FILE=%LOCALAPPDATA%\ScadaBench-deploy\cloudflare.token"
if exist "%TOKEN_FILE%" (
  set /p CLOUDFLARE_API_TOKEN=<"%TOKEN_FILE%"
  set "CLOUDFLARE_ACCOUNT_ID=526f27c5e0f7387a16bc955c18425c7f"
) else (
  call npx --yes wrangler@4 whoami >nul 2>&1 || call npx --yes wrangler@4 login || goto :failed
)
rem The portable exe that runs directly (ADR-0063) is over the 25 MiB Pages limit: it goes to the
rem scadabench-downloads R2 bucket (download.scadabench.com); old portable links redirect to it.
set "PORTABLE="
for /f "delims=" %%F in ('dir /b /o-d "..\studio\dist-protected\ScadaBench-Portable-*.exe" 2^>nul') do if not defined PORTABLE set "PORTABLE=%%F"
if not defined PORTABLE (echo No portable exe found in studio\dist-protected; the portable link will not work yet. & goto :portable_done)
findstr /c:"download.scadabench.com/download/%PORTABLE%" dist\download\index.html >nul || (echo The download page does not link %PORTABLE%: set the version in src\data\site.ts. & goto :failed)
call npx --yes wrangler@4 r2 object put "scadabench-downloads/download/%PORTABLE%" --file "..\studio\dist-protected\%PORTABLE%" --content-type application/vnd.microsoft.portable-executable --content-disposition "attachment; filename=%PORTABLE%" --remote || goto :failed
>>dist\_redirects echo /download/ScadaBench-Portable.exe https://download.scadabench.com/download/%PORTABLE% 302
>>dist\_redirects echo /download/ScadaBench-Portable-* https://download.scadabench.com/download/%PORTABLE% 302
echo Portable exe uploaded: %PORTABLE%
:portable_done
rem Tutorial videos: the light 720p copies from marketing\videos\web_encode.py when present,
rem otherwise the 1080p masters rendered by marketing\videos\make.py (never the raw captures).
set "VIDEO_DIR=..\marketing\videos\out\web"
if not exist "%VIDEO_DIR%\*.mp4" set "VIDEO_DIR=..\marketing\videos\out"
if not exist "%VIDEO_DIR%\*.mp4" goto :no_videos
if not exist dist\videos mkdir dist\videos
copy /y "%VIDEO_DIR%\*.mp4" dist\videos\ >nul
del /q dist\videos\*.capture.mp4 2>nul
copy /y "%VIDEO_DIR%\*.jpg" dist\videos\ >nul
echo Tutorial videos included from %VIDEO_DIR%.
goto :videos_done
:no_videos
echo No tutorial videos found; run marketing\videos\make.py, then web_encode.py.
:videos_done
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
