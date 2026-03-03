@echo off
REM Start Suricata with User-Writable Log Directory
REM Run as Administrator

echo ========================================
echo Suricata User-Mode Startup
echo ========================================
echo.

REM Check admin
net session >nul 2>&1
if "%ERRORLEVEL%" neq 0 (
    echo [ERROR] Run as Administrator!
    pause
    exit /b 1
)

REM Stop existing Suricata
taskkill /F /IM suricata.exe >nul 2>&1

REM Create log directory in user profile
set LOG_DIR=d:\AmmanGate\suricata-logs
if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"

echo Starting Suricata with log directory: %LOG_DIR%
cd /D "C:\Program Files\Suricata"
start "" suricata.exe -i 192.168.1.106 -c suricata.yaml -l "%LOG_DIR%"

echo Waiting 5 seconds for initialization...
timeout /t 5 /nobreak >nul

echo.
echo Checking if eve.json was created...
if exist "%LOG_DIR%\eve.json" (
    echo [SUCCESS] eve.json created at %LOG_DIR%\eve.json
) else (
    echo [INFO] eve.json not yet created - may need traffic
)

echo.
echo Update d:\AmmanGate\apps\bodyguard-core\.env with:
echo SURICATA_EVE_LOG=%LOG_DIR%\eve.json
echo.

pause
