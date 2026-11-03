# Bundled by Horizon-QuickPlay.exe. No Microsoft credentials or game assets stored here.
# Windows PowerShell 5.1 / Windows Forms.
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

$cli = Join-Path $PSScriptRoot 'unlicensed-horizon-client.exe'
if (-not (Test-Path -LiteralPath $cli -PathType Leaf)) {
    [System.Windows.Forms.MessageBox]::Show('The bundled Horizon launcher is missing.','Horizon QuickPlay','OK','Error') | Out-Null
    exit 1
}
$script:installProcess = $null
$script:prepareProcess = $null
$script:playProcess = $null
$script:pendingPlay = $false
$script:busy = $false
$script:appDir = Join-Path $env:APPDATA 'unlicensed-horizon-client'
$script:logDir = Join-Path $script:appDir 'quickplay-logs'
$script:preferencesPath = Join-Path $script:appDir 'quickplay-preferences.json'
function Get-Preferences {
    $primary = [string]$script:preferencesPath
    $backup = $primary + '.bak'
    # Recover from an interrupted write without discarding the last valid configuration.
    foreach ($candidate in @($primary, $backup)) {
        if (-not [System.IO.File]::Exists($candidate)) { continue }
        try {
            $prefs = Get-Content -LiteralPath $candidate -Raw -ErrorAction Stop | ConvertFrom-Json -ErrorAction Stop
            return @{ minecraft_dir = [string]$prefs.minecraft_dir }
        } catch { }
    }
    return @{}
}
function Save-Preferences {
    # PowerShell 5.1/.NET Framework can reject File.Replace(source, destination, $null)
    # with an invalid-path exception. Use validated absolute paths and Copy(overwrite)
    # instead; never use the three-argument File.Replace API here.
    $destination = [System.IO.Path]::GetFullPath([string]$script:preferencesPath)
    $directory = [System.IO.Path]::GetDirectoryName($destination)
    [void][System.IO.Directory]::CreateDirectory($directory)
    $content = @{ minecraft_dir = $script:minecraftPath.Text.Trim() } | ConvertTo-Json -Compress
    $temp = [System.IO.Path]::Combine($directory, ('quickplay-preferences-' + [Guid]::NewGuid().ToString('N') + '.tmp'))
    $backup = $destination + '.bak'
    try {
        [System.IO.File]::WriteAllText($temp, $content, (New-Object System.Text.UTF8Encoding($false)))
        if ([System.IO.File]::Exists($destination)) {
            [System.IO.File]::Copy($destination, $backup, $true)
        }
        [System.IO.File]::Copy($temp, $destination, $true)
    } catch {
        # If the overwrite was interrupted or refused, recover the previous copy.
        if ([System.IO.File]::Exists($backup)) {
            try { [System.IO.File]::Copy($backup, $destination, $true) } catch { }
        }
        throw
    } finally {
        if ([System.IO.File]::Exists($temp)) { [System.IO.File]::Delete($temp) }
    }
}


function Get-Profile {
    $map = @{}
    try {
        $lines = & $cli show 2>&1
        if ($LASTEXITCODE -ne 0) { return $map }
        foreach ($line in @($lines)) {
            $part = ([string]$line).Split(@('='), 2)
            if ($part.Count -eq 2) { $map[$part[0]] = $part[1] }
        }
    } catch { }
    return $map
}
function UI-Color([int]$r, [int]$g, [int]$b) {
    return [System.Drawing.Color]::FromArgb($r, $g, $b)
}
function Label([string]$text, [int]$x, [int]$y, [int]$w, [int]$h = 24) {
    $c = New-Object System.Windows.Forms.Label
    $c.Text = $text
    $c.Location = New-Object System.Drawing.Point($x, $y)
    $c.Size = New-Object System.Drawing.Size($w, $h)
    $c.Font = New-Object System.Drawing.Font('Segoe UI', 9)
    $c.ForeColor = UI-Color 190 203 219
    $script:form.Controls.Add($c)
    return $c
}
function Field([string]$value, [int]$x, [int]$y, [int]$w) {
    $c = New-Object System.Windows.Forms.TextBox
    $c.Text = $value
    $c.Location = New-Object System.Drawing.Point($x, $y)
    $c.Size = New-Object System.Drawing.Size($w, 34)
    $c.Font = New-Object System.Drawing.Font('Segoe UI', 12)
    $c.BackColor = UI-Color 32 43 59
    $c.ForeColor = [System.Drawing.Color]::White
    $c.BorderStyle = 'FixedSingle'
    $script:form.Controls.Add($c)
    return $c
}
function Button([string]$text, [int]$x, [int]$y, [int]$w, [int]$h, [int[]]$rgb) {
    $b = New-Object System.Windows.Forms.Button
    $b.Text = $text
    $b.Location = New-Object System.Drawing.Point($x, $y)
    $b.Size = New-Object System.Drawing.Size($w, $h)
    $b.BackColor = UI-Color $rgb[0] $rgb[1] $rgb[2]
    $b.ForeColor = [System.Drawing.Color]::White
    $b.FlatStyle = 'Flat'
    $b.FlatAppearance.BorderSize = 0
    $b.Font = New-Object System.Drawing.Font('Segoe UI Semibold', 10)
    $script:form.Controls.Add($b)
    return $b
}
function Alert([string]$text, [string]$title = 'Horizon QuickPlay') {
    [System.Windows.Forms.MessageBox]::Show($text, $title, 'OK', 'Information') | Out-Null
}
function Fail([string]$text) {
    $script:status.Text = $text
    $script:status.ForeColor = UI-Color 245 159 148
    [System.Windows.Forms.MessageBox]::Show($text,'Horizon QuickPlay','OK','Warning') | Out-Null
}
function Set-Status([string]$text) {
    $script:status.Text = $text
    $script:status.ForeColor = UI-Color 163 222 199
}
function Run-Cli([string[]]$values) {
    # In Windows PowerShell 5.1, native stderr can become an ErrorRecord. Keep
    # nonzero exit-code diagnostics rather than turning them into terminating GUI errors.
    $previousPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $output = & $cli @values 2>&1
        $code = $LASTEXITCODE
    } finally { $ErrorActionPreference = $previousPreference }
    return @{ Code = $code; Text = (($output | ForEach-Object { [string]$_ }) -join [Environment]::NewLine) }
}
function Save-Profile {
    $nick = $script:nickname.Text.Trim()
    if ($nick -cnotmatch '^[A-Za-z0-9_]{3,16}$') {
        throw 'Ник должен содержать от 3 до 16 символов: английские буквы, цифры или _.'
    }
    $destination = ''
    if ($script:joinServer.Checked) {
        $destination = $script:server.Text.Trim()
        if ($destination.Length -eq 0) { throw 'Введи адрес сервера или выключи режим подключения к серверу.' }
    }
    # Missing Minecraft data does not prevent saving a nickname or installing the
    # open-source engine. The preparation stage reports missing data explicitly.
    Save-Preferences
    $previous = Get-Profile
    # PowerShell variable names are case-insensitive; $home would collide with
    # the protected built-in $HOME variable and fail whenever Play is clicked.
    $gameHomeDir = [string]$previous['home']
    if (-not $gameHomeDir) { $gameHomeDir = Join-Path $script:appDir 'game-home' }
    $settings = @{
        nickname = $nick
        client_path = [string]$previous['client']
        resources_path = [string]$script:resourcesPath
        home_path = $gameHomeDir
        world = ''
        server = $destination
        render_distance = 12
        creative = $false
    }
    $json = $settings | ConvertTo-Json -Compress -Depth 4
    $b64 = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($json))
    $r = Run-Cli @('config-json', $b64)
    if ($r.Code -ne 0) { throw ('Не удалось сохранить профиль: ' + $r.Text) }
}
function Is-Prepared([string]$folder) {
    if (-not $folder -or -not (Test-Path -LiteralPath $folder -PathType Container)) { return $false }
    $saved = $script:resourcesPath
    try {
        # The audit returns structured JSON even when incomplete; this test is
        # deliberately stricter than simply finding a folder on disk.
        $script:resourcesPath = $folder
        $r = Run-Cli @('audit-resources')
        $report = $r.Text | ConvertFrom-Json
        return ($r.Code -eq 0 -and $report.ready_for_attempt -eq $true)
    } catch { return $false }
    finally { $script:resourcesPath = $saved }
}
function Find-Resources {
    # Run a non-mutating audit against any candidate, rather than accepting an
    # empty folder or an incomplete original-content experiment as playable.
    $current = [string]$script:resourcesPath
    if (Is-Prepared $current) { return $true }
    $result = Run-Cli @('find-resources')
    if ($result.Code -ne 0) { return $false }
    $path = $result.Text.Trim()
    if (-not $path -or -not (Test-Path -LiteralPath $path -PathType Container)) { return $false }
    $script:resourcesPath = $path
    # Persist before auditing: the CLI reads its own on-disk config.
    Save-Profile
    if (-not (Is-Prepared $path)) { return $false }
    $script:resourcesLabel.Text = 'Игровые ресурсы: найдены и проверены'
    $script:resourcesLabel.ForeColor = UI-Color 154 229 188
    return $true
}
function Prepare-Game {
    try {
        $profile = Get-Profile
        $engine = [string]$profile['client']
        if (-not $engine -or -not (Test-Path -LiteralPath $engine -PathType Leaf)) {
            throw 'Сначала нужно установить Horizon.'
        }
        New-Item -ItemType Directory -Path $script:logDir -Force | Out-Null
        $script:preparedTarget = Join-Path $script:appDir 'prepared-resources\26.3'
        $env:HORIZON_QUICKPLAY_ENGINE = $engine
        $env:HORIZON_QUICKPLAY_PREPARED = $script:preparedTarget
        $env:HORIZON_QUICKPLAY_MC_DIR = $script:minecraftPath.Text.Trim()
        $helper = Join-Path $PSScriptRoot 'AutoPrepare.ps1'
        if (-not (Test-Path -LiteralPath $helper -PathType Leaf)) {
            throw 'Не найден встроенный AutoPrepare.ps1.'
        }
        $stdout = Join-Path $script:logDir 'prepare-stdout.log'
        $stderr = Join-Path $script:logDir 'prepare-stderr.log'
        $args = @('-NoLogo', '-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass',
            '-File', ('"' + $helper + '"'))
        $script:prepareProcess = Start-Process -FilePath 'powershell.exe' -ArgumentList $args `
            -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
        $script:busy = $true
        $script:playButton.Enabled = $false
        Set-Status 'Проверяю Minecraft 26.3 и автоматически подготавливаю ресурсы…'
    } catch {
        $script:busy = $false
        $script:playButton.Enabled = $true
        Fail $_.Exception.Message
    }
}
function Start-Verified-Game {
    Save-Profile
    $check = Run-Cli @('doctor')
    if ($check.Code -ne 0) { throw ('Не удалось запустить Horizon: ' + $check.Text) }
    New-Item -ItemType Directory -Path $script:logDir -Force | Out-Null
    $stdout = Join-Path $script:logDir 'play-stdout.log'
    $stderr = Join-Path $script:logDir 'play-stderr.log'
    $script:playProcess = Start-Process -FilePath $cli -ArgumentList @('play','--wait') -PassThru `
        -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    $script:busy = $true
    $script:playButton.Enabled = $false
    Set-Status 'Horizon запущен. Если окно игры не появилось, открой логи.'
}
function Launch-Game {
    try {
        Save-Profile
        if (-not (Find-Resources)) {
            # The Play button handles both the engine download and preparation
            # of resources from the player's own installed Minecraft copy.
            Prepare-Game
            return
        }
        Start-Verified-Game
    } catch {
        $script:busy = $false
        $script:playButton.Enabled = $true
        Fail $_.Exception.Message
    }
}
function Find-Local-Horizon {
    $roots = @()
    if ($env:APPDATA) {
        $roots += (Join-Path $env:APPDATA 'horizon-launcher\versions')
        $roots += (Join-Path $env:APPDATA 'unlicensed-horizon-client\engines')
    }
    if ($env:LOCALAPPDATA) {
        $roots += (Join-Path $env:LOCALAPPDATA 'Horizon\current')
        $roots += (Join-Path $env:LOCALAPPDATA 'Horizon\releases')
    }
    foreach ($root in $roots) {
        if (-not (Test-Path -LiteralPath $root -PathType Container)) { continue }
        $direct = Join-Path $root 'horizon-client.exe'
        if (Test-Path -LiteralPath $direct -PathType Leaf) { return $direct }
        $children = @(Get-ChildItem -LiteralPath $root -Directory -ErrorAction SilentlyContinue |
            Sort-Object LastWriteTime -Descending | Select-Object -First 30)
        foreach ($child in $children) {
            foreach ($candidate in @((Join-Path $child.FullName 'horizon-client.exe'),
                    (Join-Path $child.FullName 'horizon\horizon-client.exe'))) {
                if (Test-Path -LiteralPath $candidate -PathType Leaf) { return $candidate }
            }
        }
    }
    return $null
}
function Use-Local-Horizon([string]$path) {
    if ([IO.Path]::GetFileName($path) -ine 'horizon-client.exe' -or
            -not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw 'Выбери horizon-client.exe из официального распакованного архива Horizon.'
    }
    $result = Run-Cli @('config', '--client', $path)
    if ($result.Code -ne 0) { throw ('Не удалось выбрать движок: ' + $result.Text) }
    Set-Status ('Локальный движок: ' + $path)
}
function Play-Clicked {
    if ($script:busy) { return }
    try {
        Save-Profile
        $profile = Get-Profile
        $engine = [string]$profile['client']
        if ($engine -and (Test-Path -LiteralPath $engine -PathType Leaf)) {
            Launch-Game
            return
        }
        $localEngine = Find-Local-Horizon
        if ($localEngine) {
            Use-Local-Horizon $localEngine
            Launch-Game
            return
        }
        # Download only open-source engine binary. Do not fetch proprietary assets.
        New-Item -ItemType Directory -Path $script:logDir -Force | Out-Null
        $out = Join-Path $script:logDir 'install-stdout.log'
        $err = Join-Path $script:logDir 'install-stderr.log'
        $script:installProcess = Start-Process -FilePath $cli -ArgumentList @('install') -PassThru `
            -WindowStyle Hidden -RedirectStandardOutput $out -RedirectStandardError $err
        $script:pendingPlay = $true
        $script:busy = $true
        $script:playButton.Enabled = $false
        Set-Status 'Устанавливаю движок Horizon из официального GitHub…'
    } catch { Fail $_.Exception.Message }
}

$form = New-Object System.Windows.Forms.Form
$form.Text = 'Horizon QuickPlay'
$form.ClientSize = New-Object System.Drawing.Size(602, 707)
$form.MinimumSize = $form.Size
$form.MaximumSize = $form.Size
$form.BackColor = UI-Color 19 27 39
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedSingle'
$form.MaximizeBox = $false
$script:form = $form

$heading = Label 'HORIZON' 28 18 400 47
$heading.Font = New-Object System.Drawing.Font('Segoe UI', 27, [System.Drawing.FontStyle]::Bold)
$heading.ForeColor = UI-Color 131 235 192
$tagline = Label 'QUICKPLAY  /  LOCAL PROFILE' 32 72 510 26
$tagline.ForeColor = UI-Color 140 168 193

Label 'Никнейм' 32 126 240 | Out-Null
$profile = Get-Profile
$defaultNick = [string]$profile['name']
if (-not $defaultNick -or $defaultNick -eq 'Player') { $defaultNick = 'azetech_' }
$script:nickname = Field $defaultNick 32 155 537

$script:joinServer = New-Object System.Windows.Forms.CheckBox
$script:joinServer.Text = 'Подключиться к серверу (offline-mode)'
$script:joinServer.Location = New-Object System.Drawing.Point(32, 210)
$script:joinServer.Size = New-Object System.Drawing.Size(529, 30)
$script:joinServer.Font = New-Object System.Drawing.Font('Segoe UI', 10)
$script:joinServer.ForeColor = [System.Drawing.Color]::White
$script:joinServer.Checked = [bool]$profile['server']
$form.Controls.Add($script:joinServer)

$script:server = Field ([string]$profile['server']) 32 246 537
$script:server.Enabled = $script:joinServer.Checked
$script:joinServer.Add_CheckedChanged({
    $script:server.Enabled = $script:joinServer.Checked
})
$hint = Label 'Оставь выключенным, чтобы открыть главное меню игры.' 32 291 535 28
$hint.ForeColor = UI-Color 156 171 189

$preferences = Get-Preferences
$defaultMinecraft = [string]$preferences['minecraft_dir']
if (-not $defaultMinecraft -and $env:APPDATA) { $defaultMinecraft = Join-Path $env:APPDATA '.minecraft' }
Label 'Папка Minecraft Java Edition 26.3 (для подготовки ресурсов)' 32 324 530 | Out-Null
$script:minecraftPath = Field $defaultMinecraft 32 354 420
$browseMinecraft = Button 'Обзор…' 463 354 106 34 @(64, 79, 102)
$browseMinecraft.Add_Click({
    $dlg = New-Object System.Windows.Forms.FolderBrowserDialog
    $dlg.Description = 'Выбери папку .minecraft (в ней должны быть versions, libraries, assets)'
    if ((Test-Path -LiteralPath $script:minecraftPath.Text -PathType Container)) {
        $dlg.SelectedPath = $script:minecraftPath.Text
    }
    if ($dlg.ShowDialog() -eq 'OK') { $script:minecraftPath.Text = $dlg.SelectedPath }
})
$script:resourcesPath = [string]$profile['resources']
$script:resourcesLabel = Label 'Игровые ресурсы: проверка при запуске' 32 407 530 28
$script:resourcesLabel.ForeColor = UI-Color 200 186 142
$choose = Button 'Выбрать ресурсы…' 32 446 210 38 @(64, 79, 102)
$logs = Button 'Логи' 254 446 123 38 @(64, 79, 102)
$checkServer = Button 'Проверить сервер' 389 446 180 38 @(64, 79, 102)
$checkServer.Enabled = $script:joinServer.Checked
$script:joinServer.Add_CheckedChanged({ $checkServer.Enabled = $script:joinServer.Checked })
$choose.Add_Click({
    $dlg = New-Object System.Windows.Forms.FolderBrowserDialog
    $dlg.Description = 'Папка подготовленных ресурсов Horizon 26.3 (horizon-resources.json)'
    if ($dlg.ShowDialog() -eq 'OK') {
        $script:resourcesPath = $dlg.SelectedPath
        $script:resourcesLabel.Text = 'Папка выбрана; проверка пройдёт при запуске.'
        $script:resourcesLabel.ForeColor = UI-Color 200 186 142
    }
})
$logs.Add_Click({
    New-Item -ItemType Directory -Path $script:logDir -Force | Out-Null
    Invoke-Item -LiteralPath $script:logDir
})
$checkServer.Add_Click({
    try {
        Save-Profile
        $result = Run-Cli @('probe-server', $script:server.Text.Trim())
        if ($result.Code -eq 0) {
            Alert ('Сервер ответил на status-ping.' + [Environment]::NewLine +
                'Это НЕ означает успешный вход и НЕ определяет online-mode.' +
                [Environment]::NewLine + $result.Text) 'Проверка сервера'
        } else { Fail ('Не удалось получить статус сервера:' + [Environment]::NewLine + $result.Text) }
    } catch { Fail $_.Exception.Message }
})
$script:playButton = Button '▶   PLAY HORIZON' 32 507 537 55 @(38, 135, 110)
$script:playButton.Font = New-Object System.Drawing.Font('Segoe UI Semibold', 14)
$script:playButton.Add_Click({ Play-Clicked })
$script:status = Label 'PLAY скачает Horizon и проверит игровые файлы.' 32 572 537 42
$script:status.ForeColor = UI-Color 163 222 199
$script:progress = New-Object System.Windows.Forms.ProgressBar
$script:progress.Location = New-Object System.Drawing.Point(32, 625)
$script:progress.Size = New-Object System.Drawing.Size(424, 18)
$script:progress.Style = 'Marquee'
$script:progress.MarqueeAnimationSpeed = 0
$form.Controls.Add($script:progress)
$cancel = Button 'Отмена' 469 616 100 37 @(108, 69, 72)
$cancel.Enabled = $false
$cancel.Add_Click({
    foreach ($task in @('installProcess', 'prepareProcess')) {
        $process = Get-Variable -Name $task -Scope Script -ValueOnly -ErrorAction SilentlyContinue
        if ($process -and -not $process.HasExited) {
            # Kill the helper's process tree, including an active preparer child.
            try { & taskkill.exe /PID $process.Id /T /F 2>$null | Out-Null } catch {
                try { $process.Kill() } catch { }
            }
        }
        Set-Variable -Name $task -Scope Script -Value $null
    }
    $script:pendingPlay = $false
    $script:busy = $false
    $script:playButton.Enabled = $true
    $cancel.Enabled = $false
    $script:progress.MarqueeAnimationSpeed = 0
    Set-Status 'Установка отменена. Частично скачанные файлы можно будет проверить при следующем запуске.'
})
$footer = Label 'Локальные ники работают только на offline-mode серверах. Для online-mode нужна авторизация.' 32 660 542 40
$footer.ForeColor = UI-Color 140 159 180

$script:timer = New-Object System.Windows.Forms.Timer
$script:timer.Interval = 650
$script:timer.Add_Tick({
    $cancel.Enabled = [bool]($script:installProcess -or $script:prepareProcess)
    $script:progress.MarqueeAnimationSpeed = $(if ($script:installProcess -or $script:prepareProcess) { 28 } else { 0 })
    if ($script:installProcess) {
        if ($script:installProcess.HasExited) {
        $code = $script:installProcess.ExitCode
        $script:installProcess = $null
        $script:busy = $false
        if ($code -ne 0) {
            $script:pendingPlay = $false
            $script:playButton.Enabled = $true
            $details = @()
            foreach ($fileName in @('install-stderr.log', 'install-stdout.log')) {
                $filePath = Join-Path $script:logDir $fileName
                if (Test-Path -LiteralPath $filePath -PathType Leaf) {
                    $details += @(Get-Content -LiteralPath $filePath -Tail 12 -ErrorAction SilentlyContinue)
                }
            }
            $message = ('Не удалось установить Horizon. Код: ' + $code +
                [Environment]::NewLine + (($details | Select-Object -Last 20) -join [Environment]::NewLine) +
                [Environment]::NewLine + 'Логи: ' + $script:logDir +
                [Environment]::NewLine + 'Загрузка вручную: https://github.com/horizon-rust/horizon/releases/tag/nightly' +
                [Environment]::NewLine + 'Распакуй официальный ZIP и выбери horizon-client.exe через «Локальный Horizon…».')
            Fail $message
        } else {
            Set-Status 'Движок Horizon установлен.'
            if ($script:pendingPlay) { $script:pendingPlay = $false; Launch-Game }
        }
        } else {
            $installLog = Join-Path $script:logDir 'install-stdout.log'
            if (Test-Path -LiteralPath $installLog) {
                $last = Get-Content -LiteralPath $installLog -Tail 1 -ErrorAction SilentlyContinue
                if ($last) { Set-Status ('Установка Horizon: ' + [string]$last) }
            }
        }
    }
    if ($script:prepareProcess) {
        if ($script:prepareProcess.HasExited) {
            $code = $script:prepareProcess.ExitCode
            $script:prepareProcess = $null
            $script:busy = $false
            if ($code -ne 0) {
                $script:playButton.Enabled = $true
                $details = ''
                $errFile = Join-Path $script:logDir 'prepare-stderr.log'
                if (Test-Path -LiteralPath $errFile) {
                    $details = ((Get-Content -LiteralPath $errFile -Tail 9) -join [Environment]::NewLine)
                }
                Fail ('Не удалось автоматически подготовить игру.' + [Environment]::NewLine +
                    $details + [Environment]::NewLine +
                    'Нужна установленная Minecraft Java Edition 26.3 из твоей копии игры. ' +
                    'Для лицензированных ресурсов используй официальный Minecraft Launcher.')
            } else {
                $script:resourcesPath = $script:preparedTarget
                $script:resourcesLabel.Text = 'Игровые ресурсы: подготовлены автоматически'
                $script:resourcesLabel.ForeColor = UI-Color 154 229 188
                try { Start-Verified-Game } catch { $script:busy = $false; $script:playButton.Enabled = $true; Fail $_.Exception.Message }
            }
        } else {
            # Update progress without blocking the Windows Forms event loop.
            $progressFile = Join-Path $script:logDir 'prepare-stdout.log'
            if (Test-Path -LiteralPath $progressFile) {
                $tail = Get-Content -LiteralPath $progressFile -Tail 1 -ErrorAction SilentlyContinue
                if ($tail -and ([string]$tail).Length -lt 250) {
                    try {
                        $progress = ([string]$tail) | ConvertFrom-Json
                        if ($null -ne $progress.fraction) {
                            $percent = [math]::Round(100 * [double]$progress.fraction)
                            Set-Status "Подготовка Horizon: $percent%"
                        }
                    } catch { }
                }
            }
        }
    }
    if ($script:playProcess -and $script:playProcess.HasExited) {
        $code = $script:playProcess.ExitCode
        $script:playProcess = $null
        $script:busy = $false
        $script:playButton.Enabled = $true
        if ($code -eq 0) { Set-Status 'Игра завершилась. Можно запустить снова.' }
        else { Set-Status ('Игра завершилась с кодом ' + $code + '. Открой «Логи».') }
    }
})
$script:timer.Start()
[void]$form.ShowDialog()
$script:timer.Stop()
$form.Dispose()