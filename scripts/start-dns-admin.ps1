# AmmanGate DNS Server Startup Script (Run as Administrator)
# This script starts the bodyguard-core with DNS server enabled on port 53

# Check if running as Administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "❌ This script requires Administrator privileges!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Port 53 requires Administrator rights to bind." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Please run this script as Administrator:" -ForegroundColor Cyan
    Write-Host "  1. Right-click on PowerShell" -ForegroundColor White
    Write-Host "  2. Select 'Run as Administrator'" -ForegroundColor White
    Write-Host "  3. Navigate to the AmmanGate directory" -ForegroundColor White
    Write-Host "  4. Run: .\scripts\start-dns-admin.ps1" -ForegroundColor White
    Write-Host ""
    Write-Host "Or right-click this script and select 'Run as Administrator'" -ForegroundColor Yellow
    exit 1
}

Write-Host "🛡️ AmmanGate DNS Server" -ForegroundColor Cyan
Write-Host ""
Write-Host "✅ Running with Administrator privileges" -ForegroundColor Green
Write-Host ""

# Check if .env exists
$envPath = "apps\bodyguard-core\.env"
if (-not (Test-Path $envPath)) {
    Write-Host "❌ Error: .env file not found at $envPath" -ForegroundColor Red
    exit 1
}

# Check if DNS is enabled
$envContent = Get-Content $envPath
$dnsEnabled = ($envContent | Select-String "BG_DNS_ENABLED=true")

if (-not $dnsEnabled) {
    Write-Host "⚠️  Warning: BG_DNS_ENABLED is not set to true in .env" -ForegroundColor Yellow
    Write-Host "   DNS server may not start properly." -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "🔧 Starting DNS Server on port 53..." -ForegroundColor Cyan
Write-Host ""
Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
Write-Host ""

# Start the bodyguard-core
Set-Location apps\bodyguard-core
go run .
