# Fix ClamAV DatabaseDirectory configuration
$configPath = "C:\Program Files\ClamAV\clamd.conf"

Write-Host "Reading config from $configPath..."
$content = Get-Content $configPath

# Fix the DatabaseDirectory line
$newContent = $content -replace '#DatabaseDirectory "C:\\Program Files\\ClamAV\\database"', 'DatabaseDirectory C:\Program Files\ClamAV\database'

# Write back
$newContent | Set-Content $configPath

Write-Host "Config updated successfully!"
Write-Host "Restarting ClamAV..."

# Kill existing clamd
Stop-Process -Name "clamd" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Start clamd
Start-Process -FilePath "C:\Program Files\ClamAV\clamd.exe"

Write-Host "ClamAV restarted!"
Start-Sleep -Seconds 3
