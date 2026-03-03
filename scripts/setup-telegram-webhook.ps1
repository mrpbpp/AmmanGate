# Telegram Bot Webhook Setup Script for AmmanGate
# Run this script to configure your Telegram bot webhook

Write-Host "🛡️ AmmanGate - Telegram Bot Webhook Setup" -ForegroundColor Cyan
Write-Host ""

# Check if .env file exists (try claw-gateway first, then root)
$envPath = "apps\claw-gateway\.env"
if (-not (Test-Path $envPath)) {
    $envPath = ".\.env"
}
if (-not (Test-Path $envPath)) {
    Write-Host "❌ Error: .env file not found!" -ForegroundColor Red
    Write-Host "Please create a .env file with your Telegram bot configuration:" -ForegroundColor Yellow
    Write-Host "  - At apps\claw-gateway\.env (recommended)" -ForegroundColor White
    Write-Host "  - Or at .\.env (root directory)" -ForegroundColor White
    Write-Host ""
    Write-Host "Example:" -ForegroundColor Yellow
    Write-Host "  TELEGRAM_BOT_TOKEN=your_bot_token_here" -ForegroundColor White
    Write-Host "  TELEGRAM_ALLOWED_USER_IDS=your_telegram_user_id" -ForegroundColor White
    exit 1
}

# Read configuration from .env
$envContent = Get-Content $envPath
$botToken = ($envContent | Select-String "TELEGRAM_BOT_TOKEN=").ToString().Replace("TELEGRAM_BOT_TOKEN=", "")
$allowedUsers = ($envContent | Select-String "TELEGRAM_ALLOWED_USER_IDS=").ToString().Replace("TELEGRAM_ALLOWED_USER_IDS=", "")

if ([string]::IsNullOrEmpty($botToken) -or $botToken -eq "your_telegram_bot_token_here") {
    Write-Host "❌ Error: TELEGRAM_BOT_TOKEN not configured in .env file!" -ForegroundColor Red
    Write-Host "Get your bot token from https://t.me/BotFather" -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Configuration found:" -ForegroundColor Green
Write-Host "  Bot Token: ${botToken.Substring(0, 10)}..." -ForegroundColor White
Write-Host "  Allowed Users: $allowedUsers" -ForegroundColor White
Write-Host ""

# Get the public URL for webhook
Write-Host "📝 Enter your public URL for the webhook:" -ForegroundColor Cyan
Write-Host "  - For local testing, use a service like ngrok: https://ngrok.com" -ForegroundColor Yellow
Write-Host "  - Example: https://your-domain.com or https://abc123.ngrok.io" -ForegroundColor Yellow
Write-Host ""

$publicUrl = Read-Host "Public URL"

if ([string]::IsNullOrEmpty($publicUrl)) {
    Write-Host "❌ Error: Public URL is required!" -ForegroundColor Red
    exit 1
}

# Ensure URL doesn't have trailing slash
$publicUrl = $publicUrl.TrimEnd('/')

Write-Host ""
Write-Host "🔧 Setting up Telegram webhook..." -ForegroundColor Cyan

# Build webhook URL
$webhookUrl = "$publicUrl/webhook/telegram"

try {
    # Call the setup endpoint
    $response = Invoke-RestMethod -Uri "http://localhost:3001/webhook/telegram/setup" -Method POST -ContentType "application/json" -Body (@{external_url=$publicUrl} | ConvertTo-Json) -ErrorAction Stop

    Write-Host "✅ Webhook configured successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Webhook URL: $webhookUrl" -ForegroundColor White
    Write-Host ""
    Write-Host "📱 Test your bot by sending a message on Telegram:" -ForegroundColor Cyan
    Write-Host "  - Send: /start" -ForegroundColor White
    Write-Host "  - Send: help" -ForegroundColor White
    Write-Host "  - Send: status" -ForegroundColor White
    Write-Host ""
} catch {
    Write-Host "❌ Error setting up webhook: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "Make sure the claw-gateway service is running on port 3001" -ForegroundColor Yellow
    Write-Host "Run: cd apps/claw-gateway && npm start" -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Available commands:" -ForegroundColor Cyan
Write-Host "  status    - Show system status" -ForegroundColor White
Write-Host "  devices   - List all devices" -ForegroundColor White
Write-Host "  alerts    - Show active alerts" -ForegroundColor White
Write-Host "  help      - Show all commands" -ForegroundColor White
