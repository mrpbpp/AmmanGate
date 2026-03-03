# AmmanGate - Clawbot Installation Script for Windows
# This script installs Clawbot locally as part of AmmanGate

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $ScriptDir)
$ClawbotDir = Join-Path $ProjectRoot "apps\clawbot"
$ClawbotBin = Join-Path $ClawbotDir "bin"

Write-Host "🛡️ Installing Clawbot for AmmanGate..." -ForegroundColor Cyan
Write-Host "   Project root: $ProjectRoot"
Write-Host "   Clawbot directory: $ClawbotDir"
Write-Host ""

# Create directory structure
New-Item -ItemType Directory -Force -Path $ClawbotBin | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $ClawbotDir "data") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $ClawbotDir "config") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $ClawbotDir "logs") | Out-Null

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Clawbot download URL
$ClawbotVersion = "1.0.0"
$ClawbotUrl = "https://github.com/clawbot/clawbot/releases/download/v${ClawbotVersion}/clawbot-windows-${Arch}.exe"

Write-Host "📥 Downloading Clawbot for Windows-${Arch}..." -ForegroundColor Yellow

# Download using Invoke-WebRequest
Invoke-WebRequest -Uri $ClawbotUrl -OutFile (Join-Path $ClawbotBin "clawbot.exe") -UseBasicParsing

Write-Host "✅ Clawbot binary downloaded" -ForegroundColor Green

# Create configuration file
$ConfigPath = Join-Path $ClawbotDir "config\clawbot.yaml"
$ConfigContent = @"
# Clawbot Configuration for AmmanGate

server:
  host: 0.0.0.0
  port: 8080

# Database configuration
database:
  type: sqlite
  path: ../data/clawbot.db

# Logging
logging:
  level: info
  file: ../logs/clawbot.log

# API Configuration
api:
  enabled: true
  api_key: `$env:CLAWBOT_API_KEY

# Webhook Configuration
webhook:
  secret: `$env:CLAWBOT_WEBHOOK_SECRET
  allowed_ips:
    - 127.0.0.1
    - ::1

# Message Handlers
handlers:
  # AmmanGate Claw Gateway
  - name: ammangate
    type: webhook
    url: http://localhost:3001/webhook/clawbot
    secret: `$env:CLAWBOT_WEBHOOK_SECRET
    enabled: true

# Rate Limiting
rate_limit:
  messages_per_minute: 60
  commands_per_minute: 30

# Session Configuration
session:
  timeout: 24h
  cleanup_interval: 1h
"@

Set-Content -Path $ConfigPath -Value $ConfigContent

Write-Host "✅ Configuration file created" -ForegroundColor Green

# Create start script
$StartScript = @"
@echo off
cd /d "%~dp0"
echo Starting Clawbot...
bin\clawbot.exe server --config=config\clawbot.yaml
pause
"@

Set-Content -Path (Join-Path $ClawbotDir "start.bat") -Value $StartScript

Write-Host "✅ Start script created" -ForegroundColor Green

# Create environment file template
$EnvFile = @"
# Clawbot Environment Variables
# Generate secure keys with: [System.Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])

CLAWBOT_API_KEY=change_me_in_production_generate_with_powershell
CLAWBOT_WEBHOOK_SECRET=change_me_in_production_generate_with_powershell

# AmmanGate Integration
AMMANGATE_CORE_URL=http://127.0.0.1:8787
AMMANGATE_CLAW_GATEWAY_URL=http://127.0.0.1:3001
"@

Set-Content -Path (Join-Path $ClawbotDir ".env.example") -Value $EnvFile

Write-Host ""
Write-Host "✅ Clawbot installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Generate secure keys:" -ForegroundColor White
Write-Host "   `$apiKey = [System.Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])" -ForegroundColor Gray
Write-Host "   `$webhookSecret = [System.Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])" -ForegroundColor Gray
Write-Host ""
Write-Host "2. Set environment variables:" -ForegroundColor White
Write-Host "   `$env:CLAWBOT_API_KEY = 'your_api_key'" -ForegroundColor Gray
Write-Host "   `$env:CLAWBOT_WEBHOOK_SECRET = 'your_webhook_secret'" -ForegroundColor Gray
Write-Host ""
Write-Host "3. Start Clawbot:" -ForegroundColor White
Write-Host "   .\start.bat" -ForegroundColor Gray
Write-Host ""
Write-Host "4. Or install as Windows service (requires NSSM):" -ForegroundColor White
Write-Host "   nssm install Clawbot `"$ClawbotBin\clawbot.exe`" server --config=`"$ConfigPath`"" -ForegroundColor Gray
Write-Host ""
