# AmmanGate - Start All Services Script for Windows
# This script starts all AmmanGate services in separate windows

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = $ScriptDir

# Colors for output
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

Write-ColorOutput Cyan "🛡️ AmmanGate - Starting All Services"
Write-ColorOutput White ""
Write-ColorOutput Gray "Project Root: $ProjectRoot"
Write-ColorOutput White ""

# Check if required directories exist
$requiredDirs = @(
    "$ProjectRoot\apps\bodyguard-core",
    "$ProjectRoot\apps\bodyguard-ui",
    "$ProjectRoot\apps\claw-gateway",
    "$ProjectRoot\apps\clawbot"
)

foreach ($dir in $requiredDirs) {
    if (-not (Test-Path $dir)) {
        Write-ColorOutput Red "❌ Required directory not found: $dir"
        Write-ColorOutput Yellow "Please run installation first"
        exit 1
    }
}

# Check if Go is installed
try {
    $goVersion = go version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-ColorOutput Green "✅ Go found: $goVersion"
    } else {
        throw "Go not found"
    }
} catch {
    Write-ColorOutput Red "❌ Go not found. Please install Go 1.22+ from https://golang.org/dl/"
    exit 1
}

# Check if Node.js is installed
try {
    $nodeVersion = node --version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-ColorOutput Green "✅ Node.js found: $nodeVersion"
    } else {
        throw "Node.js not found"
    }
} catch {
    Write-ColorOutput Red "❌ Node.js not found. Please install Node.js 20+ from https://nodejs.org/"
    exit 1
}

Write-ColorOutput White ""

# Function to kill existing processes on ports
function Kill-PortProcess($port) {
    $process = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty OwningProcess -ErrorAction SilentlyContinue
    if ($process) {
        Write-ColorOutput Yellow "⚠️  Process found on port $port, terminating..."
        Stop-Process -Id $process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 1
    }
}

# Kill existing processes on our ports
Write-ColorOutput Cyan "Checking for existing services..."
Kill-PortProcess 8787  # bodyguard-core
Kill-PortProcess 3000   # bodyguard-ui
Kill-PortProcess 3001   # claw-gateway
Kill-PortProcess 8080   # clawbot
Write-ColorOutput White ""

# Load environment variables
$envFile = "$ProjectRoot\.env"
if (Test-Path $envFile) {
    Write-ColorOutput Green "✅ Loading environment from .env"
    Get-Content $envFile | ForEach-Object {
        if ($_ -match "^([^#].+?)=(.+)$") {
            $name = $matches[1]
            $value = $matches[2]
            [Environment]::SetEnvironmentVariable($name, $value, "Process")
        }
    }
} else {
    Write-ColorOutput Yellow "⚠️  .env file not found, using defaults"
    Write-ColorOutput Gray "   Copy .env.example to .env and configure for production"
}
Write-ColorOutput White ""

# Array to store process info for cleanup
$script:services = @()

# Function to start a service in new window
function Start-ServiceWindow($serviceName, $scriptBlock, $workingDir, $args) {
    $powershellArgs = @(
        "-NoExit"
        "-Command"
        "cd '$workingDir'; $scriptBlock"
    )

    if ($args) {
        $powershellArgs += $args
    }

    $process = Start-Process powershell.exe -ArgumentList $powershellArgs -PassThru
    $script:services += @{
        Name = $serviceName
        Process = $process
    }

    Write-ColorOutput Green "✅ Started $serviceName (PID: $($process.Id))"
}

# Start services
Write-ColorOutput Cyan "🚀 Starting services..."
Write-ColorOutput White ""

# Service 1: Bodyguard Core (Go Backend)
Start-ServiceWindow `
    "AmmanGate Core (Port 8787)" `
    "Write-Host '🛡️ Starting AmmanGate Core...' -ForegroundColor Cyan; go run main.go" `
    "$ProjectRoot\apps\bodyguard-core"

Start-Sleep -Seconds 3

# Service 2: Clawbot
if (Test-Path "$ProjectRoot\apps\clawbot\bin\clawbot.exe") {
    Start-ServiceWindow `
        "Clawbot AI (Port 8080)" `
        "Write-Host '🤖 Starting Clawbot AI...' -ForegroundColor Cyan; .\bin\clawbot.exe server --config=.\config\clawbot.yaml" `
        "$ProjectRoot\apps\clawbot"

    Start-Sleep -Seconds 2
} else {
    Write-ColorOutput Yellow "⚠️  Clawbot not found, skipping..."
}

# Service 3: Claw Gateway
if (Test-Path "$ProjectRoot\apps\claw-gateway\node_modules") {
    Start-ServiceWindow `
        "Claw Gateway (Port 3001)" `
        "Write-Host '🔗 Starting Claw Gateway...' -ForegroundColor Cyan; npm start" `
        "$ProjectRoot\apps\claw-gateway"

    Start-Sleep -Seconds 2
} else {
    Write-ColorOutput Yellow "⚠️  Claw Gateway dependencies not found. Run: cd apps\claw-gateway; npm install"
}

# Service 4: Bodyguard UI (Dashboard)
if (Test-Path "$ProjectRoot\apps\bodyguard-ui\node_modules") {
    Start-ServiceWindow `
        "Dashboard UI (Port 3000)" `
        "Write-Host '📊 Starting Dashboard UI...' -ForegroundColor Cyan; npm run dev" `
        "$ProjectRoot\apps\bodyguard-ui"
} else {
    Write-ColorOutput Yellow "⚠️  Dashboard dependencies not found. Run: cd apps\bodyguard-ui; npm install"
}

Write-ColorOutput White ""
Write-ColorOutput Green "✅ All services started!"
Write-ColorOutput White ""

# Display service URLs
Write-ColorOutput Cyan "📌 Service URLs:"
Write-ColorOutput White ""
Write-Host "   🏠 Dashboard:     " -NoNewline
Write-ColorOutput Cyan "http://localhost:3000"
Write-Host "   🛡️  Core API:      " -NoNewline
Write-ColorOutput Cyan "http://localhost:8787"
Write-Host "   🤖 Clawbot AI:    " -NoNewline
Write-ColorOutput Cyan "http://localhost:8080"
Write-Host "   🔗 Claw Gateway:  " -NoNewline
Write-ColorOutput Cyan "http://localhost:3001"
Write-ColorOutput White ""

# Display default credentials
Write-ColorOutput Cyan "🔑 Default Credentials:"
Write-ColorOutput White ""
Write-Host "   Username: " -NoNewline
Write-ColorOutput Yellow "admin"
Write-Host "   Password: " -NoNewline
Write-ColorOutput Yellow "admin123"
Write-Host "   Action PIN: " -NoNewline
Write-ColorOutput Yellow "1234"
Write-ColorOutput White ""

Write-ColorOutput Red "⚠️  IMPORTANT: Change default credentials before production use!"
Write-ColorOutput White ""

# Wait for user input before closing (if run directly)
if ($MyInvocation.InvocationName -ne "&") {
    Write-ColorOutput Gray "Press Ctrl+C to stop all services, or close this window to leave them running..."
    Write-ColorOutput White ""

    # Wait for interrupt signal
    try {
        while ($true) {
            Start-Sleep -Seconds 1
        }
    } finally {
        # Cleanup: kill all started services
        Write-ColorOutput Yellow "Stopping all services..."
        foreach ($service in $script:services) {
            if ($service.Process -and !$service.Process.HasExited) {
                Write-ColorOutput Gray "   Stopping $($service.Name)..."
                Stop-Process -Id $service.Process.Id -Force -ErrorAction SilentlyContinue
            }
        }
        Write-ColorOutput Green "All services stopped."
    }
}

# Export service info for external use
$script:services
