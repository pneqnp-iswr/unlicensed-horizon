# No elevation, no account token, no proprietary assets. Windows PowerShell 5.1+.
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

$ErrorActionPreference = 'Stop'
$exe = Join-Path $PSScriptRoot 'unlicensed-horizon-client.exe'
if (-not (Test-Path -LiteralPath $exe)) {
    [System.Windows.Forms.MessageBox]::Show("Missing executable: $exe", 'Horizon Local Launcher', 'OK', 'Error') | Out-Null
    exit 1
}

function Get-Profile {
    $result = @{}
    $out = & $exe show 2>&1
    foreach ($line in @($out)) {
        $s = [string]$line
        $parts = $s.Split(@('='), 2)
        if ($parts.Count -eq 2) { $result[$parts[0]] = $parts[1] }
    }
    return $result
}
function New-TextBox($x, $y, $w, $value) {
    $tb = New-Object System.Windows.Forms.TextBox
    $tb.Location = New-Object System.Drawing.Point($x, $y)
    $tb.Size = New-Object System.Drawing.Size($w, 27)
    $tb.Text = [string]$value
    $tb.Font = New-Object System.Drawing.Font('Segoe UI', 10)
    $tb.BackColor = [System.Drawing.Color]::FromArgb(34, 41, 52)
    $tb.ForeColor = [System.Drawing.Color]::White
    $tb.BorderStyle = 'FixedSingle'
    $form.Controls.Add($tb)
    return $tb
}
function New-Label($text, $x, $y, $w) {
    $lbl = New-Object System.Windows.Forms.Label
    $lbl.Location = New-Object System.Drawing.Point($x, $y)
    $lbl.Size = New-Object System.Drawing.Size($w, 23)
    $lbl.Text = $text
    $lbl.ForeColor = [System.Drawing.Color]::FromArgb(186, 200, 215)
    $lbl.Font = New-Object System.Drawing.Font('Segoe UI', 9)
    $form.Controls.Add($lbl)
    return $lbl
}
function New-Button($text, $x, $y, $w, $color) {
    $btn = New-Object System.Windows.Forms.Button
    $btn.Text = $text
    $btn.Location = New-Object System.Drawing.Point($x, $y)
    $btn.Size = New-Object System.Drawing.Size($w, 34)
    $btn.BackColor = $color
    $btn.ForeColor = [System.Drawing.Color]::White
    $btn.FlatStyle = 'Flat'
    $btn.FlatAppearance.BorderSize = 0
    $btn.Font = New-Object System.Drawing.Font('Segoe UI Semibold', 9)
    $form.Controls.Add($btn)
    return $btn
}
function Pop-Error($msg) { [System.Windows.Forms.MessageBox]::Show([string]$msg,'Launcher','OK','Error') | Out-Null }

$form = New-Object System.Windows.Forms.Form
$form.Text = 'unlicensed-horizon-client · Local Launcher'
$form.Size = New-Object System.Drawing.Size(695, 905)
$form.MinimumSize = $form.Size
$form.MaximumSize = $form.Size
$form.BackColor = [System.Drawing.Color]::FromArgb(20, 26, 35)
$form.StartPosition = 'CenterScreen'

$heading = New-Label 'HORIZON / LOCAL' 26 16 580
$heading.Font = New-Object System.Drawing.Font('Segoe UI', 20, [System.Drawing.FontStyle]::Bold)
$heading.ForeColor = [System.Drawing.Color]::FromArgb(116, 202, 182)
$heading.Size = New-Object System.Drawing.Size(550, 45)
$sub = New-Label 'Independent offline launcher · no Minecraft files distributed' 28 65 640
$profile = Get-Profile

New-Label 'Offline nickname' 28 108 190 | Out-Null
$nickname = New-TextBox 28 130 250 $profile['name']
New-Label 'Render distance' 308 108 130 | Out-Null
$render = New-TextBox 308 130 85 $profile['distance']
$creative = New-Object System.Windows.Forms.CheckBox
$creative.Text = 'Creative (new world)'
$creative.ForeColor = [System.Drawing.Color]::White
$creative.Location = New-Object System.Drawing.Point(433, 132)
$creative.Size = New-Object System.Drawing.Size(210, 28)
$creative.Checked = ($profile['creative'] -eq 'true')
$form.Controls.Add($creative)

New-Label 'World (optional; blank = title screen)' 28 178 420 | Out-Null
$world = New-TextBox 28 200 624 $profile['world']
New-Label 'Server (optional, offline-mode only; leave World blank)' 28 246 600 | Out-Null
$server = New-TextBox 28 270 624 $profile['server']
New-Label 'Horizon engine executable (horizon-client.exe)' 28 317 560 | Out-Null
$client = New-TextBox 28 341 548 $profile['client']
$bBrowseClient = New-Button 'Browse' 587 338 65 ([System.Drawing.Color]::FromArgb(61, 74, 91))
New-Label 'Prepared Horizon resources (26.3 folder, with horizon-resources.json)' 28 388 620 | Out-Null
$resources = New-TextBox 28 412 548 $profile['resources']
$bBrowseResources = New-Button 'Browse' 587 409 65 ([System.Drawing.Color]::FromArgb(61, 74, 91))

$info = New-Label 'No Minecraft license is checked by this launcher. Game data must be supplied legally.' 28 460 636
$info.Size = New-Object System.Drawing.Size(636, 44)
$info.ForeColor = [System.Drawing.Color]::FromArgb(203, 180, 115)

$bFind = New-Button 'Find Prepared Data' 28 514 202 ([System.Drawing.Color]::FromArgb(61, 74, 91))
$bAudit = New-Button 'Audit Resources' 244 514 185 ([System.Drawing.Color]::FromArgb(61, 74, 91))
$bLogs = New-Button 'View Game Logs' 443 514 209 ([System.Drawing.Color]::FromArgb(61, 74, 91))
$bInstall = New-Button '1. Install Engine' 28 568 185 ([System.Drawing.Color]::FromArgb(59, 79, 111))
$bCheck = New-Button '2. Check Setup' 226 568 180 ([System.Drawing.Color]::FromArgb(57, 79, 100))
$bPlay = New-Button '3. PLAY' 420 568 232 ([System.Drawing.Color]::FromArgb(35, 123, 105))
$status = New-Label 'Ready to configure.' 28 634 625
$status.Size = New-Object System.Drawing.Size(625, 65)
$status.ForeColor = [System.Drawing.Color]::FromArgb(174, 215, 213)
$bLab = New-Button '4. Create Original Content Lab (experimental, not playable)' 28 719 624 ([System.Drawing.Color]::FromArgb(89, 82, 120))

$bContract = New-Button '5. Inspect Original Lab Contracts (not a game boot)' 28 768 624 ([System.Drawing.Color]::FromArgb(61, 74, 91))
$bContract.Add_Click({
    try {
        $root = Join-Path $env:APPDATA 'unlicensed-horizon-client\open-content-lab-v0.7'
        $result = & $exe lab-contract $root 2>&1
        $jsonLine = ($result | Where-Object { $_ -is [string] -and $_.Trim().StartsWith('{') } | Select-Object -First 1)
        if (-not $jsonLine) { throw [string]($result -join [Environment]::NewLine) }
        $audit = $jsonLine | ConvertFrom-Json
        $lines = @(
            "Subset structurally consistent: $($audit.subset_consistent)",
            "Registry states: $($audit.registry_states)",
            "Registry blocks: $($audit.registry_blocks)",
            "Resolved tags: $($audit.resolved_tags)",
            "Model bindings: $($audit.model_bindings)",
            "Decoded PNG textures: $($audit.png_textures)"
        )
        foreach ($issue in $audit.issues) { $lines += "ERROR: $issue" }
        $lines += 'Remaining known blockers:'
        foreach ($item in $audit.next_known_blockers) { $lines += "- $item" }
        $lines += $audit.note
        [System.Windows.Forms.MessageBox]::Show(($lines -join [Environment]::NewLine),'Open-content lab contract','OK','Information') | Out-Null
        $status.Text = if ($audit.subset_consistent) { 'Subset structurally consistent. Actual game launch NOT verified.' } else { 'Structural contract failed. See findings.' }
    } catch { Pop-Error $_.Exception.Message }
})

$bLab.Add_Click({
    try {
        $output = & $exe lab-init 2>&1
        if ($LASTEXITCODE -eq 0) {
            $status.Text = 'Generated original experimental content. It is NOT playable in Horizon yet.'
            [System.Windows.Forms.MessageBox]::Show(($output -join [Environment]::NewLine),'Original-content lab','OK','Information') | Out-Null
        } else {
            $status.Text = 'Lab already exists or creation failed. No existing files overwritten.'
            Pop-Error ($output -join [Environment]::NewLine)
        }
    } catch { Pop-Error $_.Exception.Message }
})

$bProbe = New-Button '6. Run isolated Horizon laboratory probe (experimental)' 28 816 624 ([System.Drawing.Color]::FromArgb(91, 80, 110))
$bProbe.Add_Click({
    try {
        $script:probeProcess = Start-Process -FilePath (Join-Path $PSScriptRoot 'PROBE-LAB-WINDOWS.bat') -PassThru
        $status.Text = 'Isolated lab probe started. The test runs separately from normal Play.'
    } catch { Pop-Error $_.Exception.Message }
})

$bBrowseClient.Add_Click({
    $dlg = New-Object System.Windows.Forms.OpenFileDialog
    $dlg.Filter = 'Executables (*.exe)|*.exe|All files (*.*)|*.*'
    if ($dlg.ShowDialog() -eq 'OK') { $client.Text = $dlg.FileName }
})
$bBrowseResources.Add_Click({
    $dlg = New-Object System.Windows.Forms.FolderBrowserDialog
    $dlg.Description = 'Choose the 26.3 prepared Horizon resource directory'
    if ($dlg.ShowDialog() -eq 'OK') { $resources.Text = $dlg.SelectedPath }
})
function Save-Profile {
    # PowerShell 5.1's legacy native argument quoting can drop empty string values.
    # Send one UTF-8 JSON payload instead, preserving blank world/server fields.
    $prior = Get-Profile
    $settings = @{
        nickname = [string]$nickname.Text
        client_path = [string]$client.Text
        resources_path = [string]$resources.Text
        home_path = [string]$prior['home']
        world = [string]$world.Text
        server = [string]$server.Text
        render_distance = [int]$render.Text
        creative = [bool]$creative.Checked
    }
    $json = $settings | ConvertTo-Json -Compress -Depth 4
    $encoded = [System.Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($json))
    $out = & $exe config-json $encoded 2>&1
    if ($LASTEXITCODE -ne 0) { throw [string]($out -join [Environment]::NewLine) }
}

$bCheck.Add_Click({
    try {
        Save-Profile
        $out = & $exe doctor 2>&1
        if ($LASTEXITCODE -eq 0) {
            $status.Text = 'READY · Files configured for an offline launch.'
        } else {
            $status.Text = 'Missing requirements. See details in the dialog.'
            Pop-Error ($out -join [Environment]::NewLine)
        }
    } catch { Pop-Error $_.Exception.Message }
})
$bFind.Add_Click({
    try {
        $output = & $exe find-resources 2>&1
        if ($LASTEXITCODE -eq 0) {
            $resources.Text = ([string]($output -join '')).Trim()
            $status.Text = 'Found prepared 26.3 data. Click Check Setup.'
        } else { Pop-Error ($output -join [Environment]::NewLine) }
    } catch { Pop-Error $_.Exception.Message }
})
$bAudit.Add_Click({
    try {
        Save-Profile
        $output = & $exe audit-resources 2>&1
        $report = ($output | Where-Object { $_ -is [string] -and $_.Trim().StartsWith('{') } | Select-Object -First 1)
        if ($report) {
            $audit = $report | ConvertFrom-Json
            $lines = @("Preflight: $($audit.ready_for_attempt)")
            foreach ($item in $audit.files) {
                $lines += "$($item.name): present=$($item.present), JSON=$($item.json_valid), bytes=$($item.bytes)"
            }
            foreach ($issue in $audit.issues) { $lines += "BLOCKER: $issue" }
            $lines += "NOTE: $($audit.note)"
            [System.Windows.Forms.MessageBox]::Show(($lines -join [Environment]::NewLine), 'Horizon resource audit', 'OK', 'Information') | Out-Null
            $status.Text = if ($audit.ready_for_attempt) { 'Resource preflight passed, full game compatibility unverified.' } else { 'Resource files incomplete; see audit findings.' }
        } else { Pop-Error ($output -join [Environment]::NewLine) }
    } catch { Pop-Error $_.Exception.Message }
})
$bLogs.Add_Click({
    try {
        Save-Profile
        $home = (Get-Profile)['home']
        $folder = Join-Path $home 'logs'
        if (-not (Test-Path -LiteralPath $folder)) { New-Item -ItemType Directory -Path $folder -Force | Out-Null }
        Invoke-Item -LiteralPath $folder
        $status.Text = 'Opened game logs. latest.log is written by Horizon itself.'
    } catch { Pop-Error $_.Exception.Message }
})
$bPlay.Add_Click({
    try {
        if ($null -ne $script:gameProcess -and -not $script:gameProcess.HasExited) {
            $status.Text = 'A game session is already running.'
            return
        }
        Save-Profile
        $out = & $exe doctor 2>&1
        if ($LASTEXITCODE -ne 0) { Pop-Error ($out -join [Environment]::NewLine); return }
        $script:gameOutput = Join-Path $env:TEMP 'uhc-game-output.txt'
        $script:gameErrors = Join-Path $env:TEMP 'uhc-game-errors.txt'
        $script:gameProcess = Start-Process -FilePath $exe -ArgumentList @('play', '--wait') -PassThru `
            -RedirectStandardOutput $script:gameOutput -RedirectStandardError $script:gameErrors
        $bPlay.Enabled = $false
        $status.Text = 'Horizon started. The launcher will report if the game exits with an error.'
    } catch { Pop-Error $_.Exception.Message }
})
$bInstall.Add_Click({
    try {
        Save-Profile
        $bInstall.Enabled = $false
        $script:installProcess = Start-Process -FilePath $exe -ArgumentList @('install') -PassThru
        $status.Text = 'Downloading Horizon engine from official GitHub (separate console)...'
    } catch { $bInstall.Enabled = $true; Pop-Error $_.Exception.Message }
})
$timer = New-Object System.Windows.Forms.Timer
$timer.Interval = 500
$timer.Add_Tick({
    if ($null -ne $script:gameProcess -and $script:gameProcess.HasExited) {
        $exitCode = $script:gameProcess.ExitCode
        $script:gameProcess = $null
        $bPlay.Enabled = $true
        if ($exitCode -eq 0) {
            $status.Text = 'Game closed normally. Ready to play again.'
        } else {
            $status.Text = "Horizon exited with error $exitCode. Check View Game Logs."
            if (Test-Path -LiteralPath $script:gameErrors) {
                $errLog = (Get-Content -LiteralPath $script:gameErrors -Tail 15 -ErrorAction SilentlyContinue) -join [Environment]::NewLine
                if ($errLog) { Pop-Error $errLog }
            }
        }
    }
    if ($null -ne $script:installProcess -and $script:installProcess.HasExited) {
        $code = $script:installProcess.ExitCode
        $script:installProcess = $null
        $bInstall.Enabled = $true
        if ($code -eq 0) {
            $updated = Get-Profile
            $client.Text = [string]$updated['client']
            $status.Text = 'Engine installed. Specify prepared game data, then press PLAY.'
        } else { $status.Text = "Engine installation failed (exit $code); see console output." }
    }
})
$timer.Start()
[void]$form.ShowDialog()
$timer.Stop()
$form.Dispose()
