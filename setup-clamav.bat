@echo off
REM AmmanGate ClamAV Setup Script
REM Run this as Administrator FIRST before using ClamAV

echo ========================================
echo AmmanGate ClamAV Setup
echo ========================================
echo.

REM Check for Administrator privileges
net session >nul 2>&1
if "%ERRORLEVEL%" neq 0 (
    echo [ERROR] This script must be run as Administrator!
    echo        Right-click and select "Run as administrator"
    pause
    exit /b 1
)

echo [1/3] Copying ClamAV configuration files...
copy /Y "C:\Program Files\ClamAV\conf_examples\clamd.conf.sample" "C:\Program Files\ClamAV\clamd.conf"
echo.

echo [2/3] Fixing configuration (commenting out Example line)...
powershell -Command "(Get-Content 'C:\Program Files\ClamAV\clamd.conf') -replace '^Example$', '#Example' | Set-Content 'C:\Program Files\ClamAV\clamd.conf'"
echo.

echo [3/3] Downloading ClamAV virus database (freshclam)...
cd /D "C:\Program Files\ClamAV"
freshclam.exe
echo.

if "%ERRORLEVEL%"=="0" (
    echo ========================================
    echo [SUCCESS] ClamAV setup complete!
    echo ========================================
    echo.
    echo Now you can start ClamAV daemon with:
    echo   C:\Program Files\ClamAV\clamd.exe
    echo.
    echo Or run: start-services.bat
) else (
    echo ========================================
    echo [WARN] freshclam had issues
    echo ========================================
    echo Check your internet connection
)

pause
