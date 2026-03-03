@echo off
REM Start OpenClaw Gateway for AmmanGate AI Agent

echo ========================================
echo   Starting OpenClaw Gateway
echo   for AmmanGate AI Agent
echo ========================================
echo.

REM Set environment variables
set TELEGRAM_BOT_TOKEN=8724885465:AAFT0n7MMBgKfMUUYstNfPwblFqDEhWgAIA

REM Check if OpenClaw is installed
where openclaw >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo OpenClaw not found in PATH, using direct path...
    node "C:/Users/PC/AppData/Roaming/npm/node_modules/openclaw/openclaw.mjs" gateway --port 18789
) else (
    openclaw gateway --port 18789
)

pause
