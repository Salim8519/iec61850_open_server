@echo off
rem Publish scadabench.com: double-click, or run from any terminal.
rem First time only: a browser tab asks you to allow Wrangler; click Allow.
setlocal
cd /d "%~dp0"
call npm ci --no-audit --no-fund || goto :failed
call npm run build || goto :failed
rem Ship the installer of the version the site names (src\data\site.ts), not simply the newest file:
rem test builds land in dist-protected too. Old unversioned names redirect to it. Pages allows 25 MiB.
set "VERSION="
for /f "tokens=2 delims='" %%V in ('findstr /c:"const VERSION" src\data\site.ts') do set "VERSION=%%V"
if not defined VERSION (echo Could not read VERSION from src\data\site.ts. & goto :failed)
if exist dist\_redirects del dist\_redirects
set "SETUP=ScadaBench-Setup-%VERSION%.exe"
if not exist "..\studio\dist-protected\%SETUP%" (echo No release installer %SETUP% in studio\dist-protected; the download link will not work yet. & goto :installer_done)
if not exist dist\download mkdir dist\download
copy /y "..\studio\dist-protected\%SETUP%" "dist\download\%SETUP%" >nul
findstr /c:"/download/%SETUP%" dist\download\index.html >nul || (echo The download page does not link %SETUP%. & goto :failed)
>>dist\_redirects echo /download/ScadaBench-Setup.exe /download/%SETUP% 302
echo Installer included: %SETUP%
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
set "PORTABLE=ScadaBench-Portable-%VERSION%.exe"
if not exist "..\studio\dist-protected\%PORTABLE%" set "PORTABLE="
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
rem Tell installed apps (daily, quiet update check) which version the site now offers.
if exist "..\studio\dist-protected\%SETUP%" powershell -NoProfile -ExecutionPolicy Bypass -File "..\license-server\deploy\owner.ps1" -Action release || echo The website is live, but installed apps were not told about %VERSION%. Run it again with: powershell -File license-server\deploy\owner.ps1 -Action release
echo.
echo Published. Live at https://scadabench.com (preview: https://scadabench.pages.dev)
pause
exit /b 0
:failed
echo.
echo Publishing failed. Read the message above, fix it, then run deploy.cmd again.
pause
exit /b 1
