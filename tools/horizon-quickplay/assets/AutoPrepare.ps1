# Automatic local preparation for Horizon QuickPlay.
# Downloads only an official, SHA-1-verified Minecraft SERVER jar after a
# locally installed Minecraft 26.3 client has been found. Does not download
# proprietary client jars/assets or bypass Microsoft ownership checks.
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Report([string]$message) {
    [Console]::Out.WriteLine($message)
    [Console]::Out.Flush()
}
function ExistingFile([string]$path) {
    return [System.IO.File]::Exists($path)
}
function Ensure-Path([string]$path) {
    [void][System.IO.Directory]::CreateDirectory($path)
}

# Write UTF-8 structured diagnostics instead of requiring the GUI to decode
# Russian PowerShell 5.1 stderr through a separate process/codepage.
$script:prepareDiagnostic = [Environment]::GetEnvironmentVariable('HORIZON_QUICKPLAY_PREPARE_DIAG')
function Stop-Preparation([string]$code, [string]$message, [string[]]$missingPaths = @()) {
    if (-not [string]::IsNullOrWhiteSpace($script:prepareDiagnostic)) {
        $report = [ordered]@{
            status = 'error'
            code = $code
            message = $message
            missing_paths = @($missingPaths)
        }
        $json = $report | ConvertTo-Json -Depth 4
        $target = [System.IO.Path]::GetFullPath($script:prepareDiagnostic)
        [void][System.IO.Directory]::CreateDirectory([System.IO.Path]::GetDirectoryName($target))
        [System.IO.File]::WriteAllText($target, $json, (New-Object System.Text.UTF8Encoding($false)))
    }
    throw "[$code] $message"
}

$engine = [Environment]::GetEnvironmentVariable('HORIZON_QUICKPLAY_ENGINE')
$output = [Environment]::GetEnvironmentVariable('HORIZON_QUICKPLAY_PREPARED')
if ([string]::IsNullOrWhiteSpace($engine) -or -not (ExistingFile $engine)) {
    throw 'Движок Horizon не найден. Сначала установи движок кнопкой Play.'
}
if ([string]::IsNullOrWhiteSpace($output)) {
    throw 'Не задан каталог подготовленных ресурсов.'
}
if (-not $env:APPDATA) {
    throw 'Не удалось определить папку APPDATA.'
}
$mc = [Environment]::GetEnvironmentVariable('HORIZON_QUICKPLAY_MC_DIR')
if ([string]::IsNullOrWhiteSpace($mc)) { $mc = Join-Path $env:APPDATA '.minecraft' }
$mc = [System.IO.Path]::GetFullPath($mc)
$versionRoot = Join-Path (Join-Path $mc 'versions') '26.3'
$versionJson = Join-Path $versionRoot '26.3.json'
$clientJar = Join-Path $versionRoot '26.3.jar'
$assetsIndex = Join-Path $mc 'assets\indexes'
$assetsObjects = Join-Path $mc 'assets\objects'
$libraries = Join-Path $mc 'libraries'
# Report the exact absent paths, rather than a generic install failure.
$requiredFiles = @($versionJson, $clientJar)
$requiredDirs = @($assetsIndex, $assetsObjects, $libraries)
$missing = @()
foreach ($candidate in $requiredFiles) {
    if (-not (ExistingFile $candidate)) { $missing += $candidate }
}
foreach ($candidate in $requiredDirs) {
    if (-not [System.IO.Directory]::Exists($candidate)) { $missing += $candidate }
}
if ($missing.Count -gt 0) {
    Stop-Preparation 'MINECRAFT_26_3_INPUTS_MISSING' (
        'В выбранной папке нет полного набора игровых файлов Minecraft Java Edition 26.3. ' +
        'Если Minecraft установлена в другой папке, нажми «Обзор…» и выбери её каталог с versions, assets и libraries. ' +
        'Если локальной установленной копии игры нет, Horizon не сможет подготовить её ресурсы.'
    ) $missing
}

Report 'Найдена локальная Minecraft Java Edition 26.3. Проверяю исходные файлы...'
# Reject incomplete or modified client data before running a third-party JVM.
# Only the local, explicitly selected installation is read; never fetch client assets.
$manifestBeforeDownload = Get-Content -LiteralPath $versionJson -Raw | ConvertFrom-Json
if (-not $manifestBeforeDownload.downloads -or -not $manifestBeforeDownload.downloads.client) {
    throw 'Не найдена информация для проверки установленного Minecraft client.jar.'
}
$clientInfo = $manifestBeforeDownload.downloads.client
$clientSha1 = [string]$clientInfo.sha1
$clientSize = [long]$clientInfo.size
if ($clientSha1 -cnotmatch '^[a-fA-F0-9]{40}$' -or $clientSize -le 0) {
    throw 'Некорректная контрольная сумма client.jar в локальном manifest.'
}
if ((Get-Item -LiteralPath $clientJar).Length -ne $clientSize -or
    (Get-FileHash -LiteralPath $clientJar -Algorithm SHA1).Hash -ine $clientSha1) {
    Stop-Preparation 'MINECRAFT_CLIENT_CHECKSUM_MISMATCH' 'Установленный client.jar не прошёл проверку SHA-1. Восстанови Minecraft 26.3 через официальный лаунчер.' @($clientJar)
}
Report 'Проверка client.jar по SHA-1 пройдена.'

$metadata = Get-Content -LiteralPath $versionJson -Raw | ConvertFrom-Json
if (-not $metadata.downloads -or -not $metadata.downloads.server) {
    throw 'Локальный файл версии не содержит проверяемой ссылки на сервер Minecraft 26.3.'
}
$serverInfo = $metadata.downloads.server
$url = [string]$serverInfo.url
$expectedSha1 = [string]$serverInfo.sha1
$expectedSize = [long]$serverInfo.size
$uri = New-Object System.Uri($url, [System.UriKind]::Absolute)
$approvedHosts = @('piston-data.mojang.com','launcher.mojang.com','piston-meta.mojang.com','launchermeta.mojang.com')
if ($uri.Scheme -ne 'https' -or $approvedHosts -cnotcontains $uri.Host -or
    $expectedSha1 -cnotmatch '^[a-fA-F0-9]{40}$' -or $expectedSize -le 1024 -or
    $expectedSize -gt 125829120) {
    throw 'Источник server.jar или его контрольная сумма не прошли проверку.'
}
$cache = Join-Path $env:LOCALAPPDATA 'Horizon-QuickPlay\downloads'
Ensure-Path $cache
$serverJar = Join-Path $cache ('minecraft-server-26.3-' + $expectedSha1.ToLowerInvariant() + '.jar')
$valid = $false
if (ExistingFile $serverJar) {
    $serverFile = Get-Item -LiteralPath $serverJar
    if ($serverFile.Length -eq $expectedSize) {
        $valid = ((Get-FileHash -LiteralPath $serverJar -Algorithm SHA1).Hash -ieq $expectedSha1)
    }
}
if (-not $valid) {
    Report 'Скачиваю server.jar с официального сервера Mojang (только серверный файл)...'
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12
    $temporary = $serverJar + '.partial-' + [Guid]::NewGuid().ToString('N')
    try {
        Invoke-WebRequest -UseBasicParsing -Uri $uri.AbsoluteUri -OutFile $temporary
        $actualSize = (Get-Item -LiteralPath $temporary).Length
        if ($actualSize -ne $expectedSize) {
            throw "Неверный размер скачанного server.jar: $actualSize вместо $expectedSize"
        }
        $actualHash = (Get-FileHash -LiteralPath $temporary -Algorithm SHA1).Hash
        if ($actualHash -ine $expectedSha1) {
            throw 'SHA-1 скачанного server.jar не совпадает с версией Mojang.'
        }
        Move-Item -LiteralPath $temporary -Destination $serverJar -Force
    } finally {
        if (ExistingFile $temporary) { Remove-Item -LiteralPath $temporary -Force }
    }
    Report 'Скачанный server.jar проверен по SHA-1.'
} else {
    Report 'Проверенный server.jar уже скачан. Использую кэш.'
}

$engineDir = Split-Path -Parent $engine
$prepareExe = Join-Path $engineDir 'horizon-prepare.exe'
if (-not (ExistingFile $prepareExe)) {
    throw ('В установленном Horizon отсутствует horizon-prepare.exe: ' + $prepareExe + '. Обнови движок.')
}
$root = $engineDir
$foundRoot = $false
for ($i = 0; $i -lt 4; $i++) {
    if (ExistingFile (Join-Path $root 'research-lock.json')) {
        $foundRoot = $true
        break
    }
    $parent = Split-Path -Parent $root
    if (-not $parent -or $parent -eq $root) { break }
    $root = $parent
}
if (-not $foundRoot) {
    throw 'В папке движка не найден research-lock.json. Переустанови Horizon из официального релиза.'
}
Ensure-Path $output
Report 'Подготавливаю игровые ресурсы Horizon из локальной Minecraft (может потребоваться Java 25)...'
& $prepareExe --prepare --minecraft-dir $mc --server-jar $serverJar --root $root --output $output --complete --progress json
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    throw "horizon-prepare завершился с ошибкой $exitCode. Проверь наличие Java 25 и лог подготовки."
}
Report 'Проверяю готовность подготовленных файлов...'
& $prepareExe --check --root $root --output $output
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    throw 'horizon-prepare не подтвердил полную готовность ресурсов. Проверь лог подготовки.'
}
Report ('PREPARED_PATH=' + $output)