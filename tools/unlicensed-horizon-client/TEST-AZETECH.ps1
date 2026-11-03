# Manual join attempt for a trusted offline-mode server.
# Needs built unlicensed-horizon-client.exe and prepared Horizon resources.
param(
    [string]$Server = '127.0.0.1:25565',
    [string]$Nickname = 'azetech_'
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$cli = Join-Path $PSScriptRoot 'unlicensed-horizon-client.exe'
if (-not (Test-Path -LiteralPath $cli -PathType Leaf)) {
    Write-Host 'Missing unlicensed-horizon-client.exe next to this script.' -ForegroundColor Red
    Write-Host 'Build from tools\unlicensed-horizon-client: go build -o unlicensed-horizon-client.exe ./cmd/uhc'
    exit 2
}
if ($Nickname -cnotmatch '^[A-Za-z0-9_]{3,16}$') {
    Write-Error 'Nickname must contain 3-16 ASCII letters, digits, or underscores.'
    exit 2
}

# Preserve current configuration before changing nickname/server.
$data = $env:UHC_DATA_DIR
if ([string]::IsNullOrWhiteSpace($data)) { $data = Join-Path $env:APPDATA 'unlicensed-horizon-client' }
$configPath = Join-Path $data 'config.json'
if (Test-Path -LiteralPath $configPath -PathType Leaf) {
    $backup = "$configPath.$([DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss-fff')).bak"
    Copy-Item -LiteralPath $configPath -Destination $backup -ErrorAction Stop
    Write-Host "Previous configuration backed up to: $backup"
}

Write-Host "Configuring local Horizon nickname '$Nickname' and target '$Server'..."
& $cli config --name $Nickname --world= --server $Server
if ($LASTEXITCODE -ne 0) { Write-Error 'Configuration rejected by launcher.'; exit 1 }

Write-Host 'Checking Minecraft status endpoint (does NOT authenticate a player)...'
& $cli probe-server $Server
if ($LASTEXITCODE -ne 0) {
    Write-Warning 'Status probe failed. Network trouble or disabled status endpoint; this is not proof login cannot work.'
}

Write-Host 'Checking game executable and prepared resources...'
& $cli doctor
if ($LASTEXITCODE -ne 0) {
    Write-Warning 'Launch blocked. Install Horizon and provide legitimately prepared Minecraft 26.3 resources.'
    & $cli show
    exit 3
}

Write-Host 'Starting Horizon. Inspect actual in-game login; a TCP/status response does not prove a successful join.'
& $cli play --wait
$gameExit = $LASTEXITCODE
Write-Host 'Game log locations:'
& $cli logs
if ($gameExit -ne 0) {
    Write-Warning 'Horizon process reported failure. Check the game/server logs.'
    exit 4
}
Write-Host 'Horizon process finished. Confirm multiplayer spawn from the game or server console.'
