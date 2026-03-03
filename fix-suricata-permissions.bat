@echo off
REM Fix Suricata Log Permissions and Restart
REM Run as Administrator

echo ========================================
echo Suricata Permission Fix
echo ========================================
echo.

REM Check admin
net session >nul 2>&1
if "%ERRORLEVEL%" neq 0 (
    echo [ERROR] Run as Administrator!
    pause
    exit /b 1
)

echo [1/3] Stopping Suricata...
taskkill /F /IM suricata.exe >nul 2>&1

echo [2/3] Fixing permissions on log directory...
icacls "C:\Program Files\Suricata\log" /grant Users:F /T >nul 2>&1
icacls "C:\Program Files\Suricata\log" /grant Everyone:F /T >nul 2>&1

REM Make sure directory exists
if not exist "C:\Program Files\Suricata\log" mkdir "C:\Program Files\Suricata\log"

echo [3/3] Starting Suricata...
cd /D "C:\Program Files\Suricata"
start "" suricata.exe -i 192.168.1.106 -c suricata.yaml -l log

echo.
echo ========================================
echo Suricata restarted!
echo ========================================
echo.
echo Waiting 5 seconds for initialization...
timeout /t 5 /nobreak >nul

echo Checking if eve.json was created...
if exist "C:\Program Files\Suricata\log\eve.json" (
    echo [SUCCESS] eve.json created!
    dir "C:\Program Files\Suricata\log\eve.json"
) else (
    echo [INFO] eve.json not yet created - may need traffic
)

pause
