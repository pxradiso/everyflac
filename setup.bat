@echo off
setlocal EnableExtensions EnableDelayedExpansion
chcp 65001 >nul
title everyFLAC by pxradise Setup

echo.
echo ===========================================================
echo       everyFLAC by pxradise - Windows Setup ^& Installer
echo ===========================================================
echo.

set "INSTALL_DIR=%USERPROFILE%\.local\bin"
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%" >nul 2>&1
if not exist "%INSTALL_DIR%" (
    echo   [ERROR] Could not create %INSTALL_DIR%
    goto fail
)

rem --- Pick the right precompiled binary for this machine ---------------
set "ARCH=amd64"
if /I "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"
if /I "%PROCESSOR_ARCHITEW6432%"=="ARM64" set "ARCH=arm64"

set "SRC=%~dp0bin\everyflac-tui-windows-%ARCH%.exe"
if not exist "%SRC%" (
    rem Fall back to the x64 build, which also runs on ARM64 via emulation.
    set "SRC=%~dp0bin\everyflac-tui-windows-amd64.exe"
)
if not exist "%SRC%" (
    echo   [ERROR] Precompiled binary not found.
    echo           Expected: %~dp0bin\everyflac-tui-windows-%ARCH%.exe
    echo           Keep setup.bat inside the everyflac folder, next to bin\
    goto fail
)

rem --- 1/3 Install the application --------------------------------------
echo   [1/3] Installing everyflac.exe...
if exist "%INSTALL_DIR%\everyflac.exe" (
    del /F /Q "%INSTALL_DIR%\everyflac.exe" >nul 2>&1
)
copy /Y "%SRC%" "%INSTALL_DIR%\everyflac.exe" >nul
if errorlevel 1 (
    echo   [ERROR] Could not copy the binary.
    echo           Close any running everyflac window and try again.
    goto fail
)
del /F /Q "%INSTALL_DIR%\everyflac-tui.exe" >nul 2>&1
del /F /Q "%INSTALL_DIR%\pxradise-everyflac.exe" >nul 2>&1
echo         installed: %INSTALL_DIR%\everyflac.exe

rem --- Icon + Start Menu shortcut (opens everyFLAC in Command Prompt) ---
if exist "%~dp0assets\everyflac.ico" (
    copy /Y "%~dp0assets\everyflac.ico" "%INSTALL_DIR%\everyflac.ico" >nul 2>&1
)
where powershell >nul 2>&1
if not errorlevel 1 (
    powershell -NoProfile -ExecutionPolicy Bypass -Command "$q=[char]34; $d=Join-Path $env:USERPROFILE '.local\bin'; $exe=Join-Path $d 'everyflac.exe'; $ico=Join-Path $d 'everyflac.ico'; $lnk=Join-Path ([Environment]::GetFolderPath('Programs')) 'everyFLAC.lnk'; $s=(New-Object -ComObject WScript.Shell).CreateShortcut($lnk); $s.TargetPath=$env:ComSpec; $s.Arguments='/k '+$q+$exe+$q; if(Test-Path $ico){$s.IconLocation=$ico}; $s.Description='everyFLAC'; $s.Save()" >nul 2>&1
    echo         created Start Menu shortcut: everyFLAC
)

rem --- 2/3 yt-dlp -------------------------------------------------------
echo   [2/3] Checking yt-dlp...
if exist "%INSTALL_DIR%\yt-dlp.exe" (
    echo         found: %INSTALL_DIR%\yt-dlp.exe
) else (
    call :download "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe" "%INSTALL_DIR%\yt-dlp.exe" "yt-dlp"
    if errorlevel 1 goto fail
)

rem --- 3/3 ffmpeg -------------------------------------------------------
echo   [3/3] Checking ffmpeg...
if exist "%INSTALL_DIR%\ffmpeg.exe" (
    echo         found: %INSTALL_DIR%\ffmpeg.exe
    goto ffmpeg_done
)
where ffmpeg >nul 2>&1
if not errorlevel 1 (
    for /f "delims=" %%P in ('where ffmpeg') do (
        copy /Y "%%P" "%INSTALL_DIR%\ffmpeg.exe" >nul 2>&1
        goto ffmpeg_done
    )
)
call :download "https://github.com/eugeneware/ffmpeg-static/releases/latest/download/ffmpeg-win32-x64" "%INSTALL_DIR%\ffmpeg.exe" "ffmpeg"
if errorlevel 1 goto fail
:ffmpeg_done
echo         ready: %INSTALL_DIR%\ffmpeg.exe

rem --- Add install dir to the User PATH ---------------------------------
echo   Adding %INSTALL_DIR% to your User PATH...
where powershell >nul 2>&1
if not errorlevel 1 (
    powershell -NoProfile -ExecutionPolicy Bypass -Command "$d=Join-Path $env:USERPROFILE '.local\bin'; $p=[Environment]::GetEnvironmentVariable('Path','User'); if($null -eq $p){$p=''}; if($p -notlike '*'+$d+'*'){[Environment]::SetEnvironmentVariable('Path',($d+';'+$p).TrimEnd(';'),'User')}" >nul 2>&1
) else (
    set "USER_PATH="
    for /f "tokens=2,*" %%A in ('reg query "HKCU\Environment" /v Path 2^>nul ^| findstr /i "Path"') do set "USER_PATH=%%B"
    if defined USER_PATH (
        reg add "HKCU\Environment" /v Path /t REG_EXPAND_SZ /d "%INSTALL_DIR%;!USER_PATH!" /f >nul 2>&1
    ) else (
        reg add "HKCU\Environment" /v Path /t REG_EXPAND_SZ /d "%INSTALL_DIR%" /f >nul 2>&1
    )
)

rem --- Smoke test -------------------------------------------------------
echo.
echo   Verifying installation...
"%INSTALL_DIR%\everyflac.exe" --version
if errorlevel 1 (
    echo   [WARNING] The binary was installed but did not start correctly.
    goto fail
)

echo.
echo ===========================================================
echo   Installation complete!
echo ===========================================================
echo.
echo   Open a NEW Command Prompt or PowerShell window, then run:
echo.
echo       everyflac
echo.
echo   Installed to: %INSTALL_DIR%
echo.
if /I not "%~1"=="/quiet" pause
exit /b 0

:download
set "DL_URL=%~1"
set "DL_DEST=%~2"
set "DL_NAME=%~3"
echo         downloading %DL_NAME%...
if exist "%DL_DEST%" del /F /Q "%DL_DEST%" >nul 2>&1
where curl >nul 2>&1
if not errorlevel 1 (
    curl.exe -fL --retry 3 --progress-bar "%DL_URL%" -o "%DL_DEST%"
) else (
    powershell -NoProfile -ExecutionPolicy Bypass -Command "[Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; Invoke-WebRequest -UseBasicParsing -Uri '%DL_URL%' -OutFile '%DL_DEST%'" 
)
if not exist "%DL_DEST%" (
    echo   [ERROR] Could not download %DL_NAME%.
    exit /b 1
)
for %%F in ("%DL_DEST%") do if %%~zF LSS 1024 (
    del /F /Q "%DL_DEST%" >nul 2>&1
    echo   [ERROR] Download of %DL_NAME% looks invalid.
    exit /b 1
)
echo         installed: %DL_DEST%
exit /b 0

:fail
echo.
echo   Setup did not complete. Fix the error above and run setup.bat again.
echo.
if /I not "%~1"=="/quiet" pause
exit /b 1
