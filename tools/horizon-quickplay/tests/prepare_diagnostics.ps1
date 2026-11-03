# Run on Windows PowerShell 5.1. Verify structured UTF-8 missing-Minecraft
# diagnostics without fetching Mojang files, using a fake existing engine file.
param([Parameter(Mandatory = $true)][string]$Source)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) { throw 'AutoPrepare.ps1 is missing' }

$testDir = Join-Path ([IO.Path]::GetTempPath()) ('quickplay-preflight-' + [Guid]::NewGuid().ToString('N'))
[void][IO.Directory]::CreateDirectory($testDir)
try {
    $engine = Join-Path $testDir 'horizon-client.exe'
    [IO.File]::WriteAllBytes($engine, [byte[]]@(77, 90))
    $minecraft = Join-Path $testDir 'empty minecraft installation'
    [void][IO.Directory]::CreateDirectory($minecraft)
    $diagnostic = Join-Path $testDir 'prepare-diagnostic.json'
    $env:HORIZON_QUICKPLAY_ENGINE = $engine
    $env:HORIZON_QUICKPLAY_PREPARED = Join-Path $testDir 'prepared'
    $env:HORIZON_QUICKPLAY_MC_DIR = $minecraft
    $env:HORIZON_QUICKPLAY_PREPARE_DIAG = $diagnostic

    $windowsPowerShell = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
    & $windowsPowerShell -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $Source 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) { throw 'Preparation incorrectly accepted missing files' }
    if (-not (Test-Path -LiteralPath $diagnostic -PathType Leaf)) {
        throw 'No structured diagnostic written'
    }
    $json = Get-Content -LiteralPath $diagnostic -Raw -Encoding UTF8 | ConvertFrom-Json
    if ($json.status -ne 'error' -or $json.code -ne 'MINECRAFT_26_3_INPUTS_MISSING') {
        throw ('Wrong diagnostic status or code: ' + ($json | ConvertTo-Json -Compress))
    }
    $expected = @(
        (Join-Path $minecraft 'versions\26.3\26.3.json'),
        (Join-Path $minecraft 'versions\26.3\26.3.jar'),
        (Join-Path $minecraft 'assets\indexes'),
        (Join-Path $minecraft 'assets\objects'),
        (Join-Path $minecraft 'libraries')
    )
    if (@($json.missing_paths).Count -ne $expected.Count) {
        throw 'Wrong number of missing paths'
    }
    foreach ($p in $expected) {
        if (@($json.missing_paths) -cnotcontains $p) {
            throw "Missing path not reported: $p"
        }
    }
    if ($json.message -notmatch 'Minecraft' -or $json.message -notmatch 'Обзор') {
        throw 'Missing Russian recovery guidance or encoding corruption'
    }
    Write-Output 'PASS: missing Minecraft 26.3 inputs are explicitly reported in UTF-8 JSON'
} finally {
    Remove-Item Env:HORIZON_QUICKPLAY_ENGINE -ErrorAction SilentlyContinue
    Remove-Item Env:HORIZON_QUICKPLAY_PREPARED -ErrorAction SilentlyContinue
    Remove-Item Env:HORIZON_QUICKPLAY_MC_DIR -ErrorAction SilentlyContinue
    Remove-Item Env:HORIZON_QUICKPLAY_PREPARE_DIAG -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $testDir -Recurse -Force -ErrorAction SilentlyContinue
}
