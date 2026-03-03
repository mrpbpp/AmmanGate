# Fix Suricata config to disable fast.log
$configPath = "C:\Program Files\Suricata\suricata.yaml"

Write-Host "Fixing Suricata configuration..."

# Read the config
$content = Get-Content $configPath

# Disable fast.log (line 90: enabled: yes -> enabled: no)
$content = $content -replace '(\s+-\s+fast:.*?enabled:\s)yes', '$1no'

# Also make sure eve-log is enabled
$content = $content -replace '(\s+-\s+eve-log:.*?enabled:\s)no', '$1yes'

# Write back
$content | Set-Content $configPath

Write-Host "Configuration updated!"
Write-Host "Restarting Suricata..."

# Stop Suricata
Stop-Process -Name "suricata" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Start Suricata
$logDir = "d:\AmmanGate\suricata-logs"
Start-Process -FilePath "C:\Program Files\Suricata\suricata.exe" -ArgumentList "-i 192.168.1.106 -c C:\Program Files\Suricata\suricata.yaml -l $logDir"

Write-Host "Suricata restarted!"
Start-Sleep -Seconds 5

# Check if eve.json exists
if (Test-Path "$logDir\eve.json") {
    Write-Host "[SUCCESS] eve.json created!"
    Get-Item "$logDir\eve.json" | Select-Object Name, Length, LastWriteTime
} else {
    Write-Host "[INFO] eve.json not found - checking logs..."
    if (Test-Path "$logDir\suricata.log") {
        Write-Host "Last 10 lines of suricata.log:"
        Get-Content "$logDir\suricata.log" -Tail 10
    }
}
