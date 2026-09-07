@echo off
chcp 65001 >nul
title everyFLAC by pxradise Setup

where powershell >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup.ps1"
    goto finish
)

echo.
echo ===========================================================
echo       everyFLAC by pxradise - Windows Setup ^& Installer
echo ===========================================================
echo.
echo PowerShell not detected. Running standard CMD installer...
set "INSTALL_DIR=%USERPROFILE%\.local\bin"
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

set "SRC=%~dp0bin\everyflac-tui-windows-amd64.exe"
if exist "%SRC%" (
    copy /Y "%SRC%" "%INSTALL_DIR%\everyflac.exe" >nul
    if exist "%INSTALL_DIR%\everyflac-tui.exe" del /F /Q "%INSTALL_DIR%\everyflac-tui.exe" >nul 2>&1
    if exist "%INSTALL_DIR%\pxradise-everyflac.exe" del /F /Q "%INSTALL_DIR%\pxradise-everyflac.exe" >nul 2>&1
    echo   [OK] Installed everyflac.exe to %INSTALL_DIR%
) else (
    echo   [ERROR] Precompiled binary not found at %SRC%
    pause
    exit /b 1
)

setx PATH "%INSTALL_DIR%;%PATH%" >nul 2>&1
echo   [OK] Added %INSTALL_DIR% to User PATH

echo.
echo Installation complete! Run 'everyflac' in a new Command Prompt window.

:finish
pause
