@echo off
REM AmmanGate Services Startup Script
REM Run this as Administrator to start ClamAV and Suricata

echo ========================================
echo AmmanGate Services - Starting as Admin
echo ========================================
echo.

REM Start ClamAV daemon
echo [1/2] Starting ClamAV daemon...
start "" "C:\Program Files\ClamAV\clamd.exe"
timeout /t 2 /nobreak >nul

REM Check if ClamAV started
tasklist /FI "IMAGENAME eq clamd.exe" 2>nul | find /I /N "clamd.exe">nul
if "%ERRORLEVEL%"=="0" (
    echo [OK] ClamAV daemon started
) else (
    echo [WARN] ClamAV may not have started properly
)
echo.

REM Start Suricata IDS
echo [2/2] Starting Suricata IDS...
cd /D "C:\Program Files\Suricata"

REM Ensure log directory exists
if not exist "log" mkdir log

start "" suricata.exe -i 1 -c suricata.yaml -l log
timeout /t 2 /nobreak >nul

REM Check if Suricata started
tasklist /FI "IMAGENAME eq suricata.exe" 2>nul | find /I /N "suricata.exe">nul
if "%ERRORLEVEL%"=="0" (
    echo [OK] Suricata IDS started
) else (
    echo [WARN] Suricata may not have started properly
)
echo.

echo ========================================
echo Services Status Check
echo ========================================
echo Checking AmmanGate Core API...
curl -s http://127.0.0.1:8787/v1/health
echo.
echo.

echo Checking ClamAV status...
curl -s http://127.0.0.1:8787/v1/clamav/status
echo.
echo.

echo Checking Suricata status...
curl -s http://127.0.0.1:8787/v1/suricata/status
echo.
echo.

pause
