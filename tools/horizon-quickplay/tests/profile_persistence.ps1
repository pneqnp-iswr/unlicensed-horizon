# Windows PowerShell 5.1 regression: a chosen client executable must survive
# every subsequent profile save, including the calls made by Play/Prepare.
param(
    [Parameter(Mandatory = $true)][string]$Cli,
    [Parameter(Mandatory = $true)][string]$Source
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if (-not (Test-Path -LiteralPath $Cli -PathType Leaf)) { throw "Missing built CLI: $Cli" }
if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) { throw "Missing UI script: $Source" }

# Load only the selected production function definitions; do not open WinForms.
# This runs the actual Save-Profile, Get-Profile and Use-Local-Horizon code.
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile(
    $Source, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -gt 0) {
    throw ("QuickPlay.ps1 parse errors: " + (($parseErrors | ForEach-Object { $_.Message }) -join '; '))
}
$names = @('Run-Cli', 'Get-Profile', 'Save-Profile', 'Use-Local-Horizon')
$defs = @($ast.FindAll({
    param($node)
    $node -is [System.Management.Automation.Language.FunctionDefinitionAst]
}, $true))
foreach ($name in $names) {
    $matches = @($defs | Where-Object { $_.Name -eq $name })
    if ($matches.Count -ne 1) { throw "Expected one function $name, got $($matches.Count)" }
    Invoke-Expression $matches[0].Extent.Text
}

# Replace only GUI-only helpers with stubs; leave persistence functions real.
function Save-Preferences { }
function Set-Status([string]$message) { }

$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('qp-profile-test-' + [guid]::NewGuid().ToString('N'))
[void][IO.Directory]::CreateDirectory($testRoot)
try {
    $script:appDir = $testRoot
    $env:UHC_DATA_DIR = $testRoot
    $script:selectedEnginePath = ''
    $script:nickname = [pscustomobject]@{ Text = 'azetech_' }
    $script:joinServer = [pscustomobject]@{ Checked = $false }
    $script:server = [pscustomobject]@{ Text = '' }
    $script:resourcesPath = ''
    $script:minecraftPath = [pscustomobject]@{ Text = '' }

    $engineDir = Join-Path $testRoot 'official Horizon with spaces'
    [void][IO.Directory]::CreateDirectory($engineDir)
    $enginePath = Join-Path $engineDir 'horizon-client.exe'
    [IO.File]::WriteAllBytes($enginePath, [byte[]](0x4d, 0x5a))

    Save-Profile # Initial profile, no engine yet
    Use-Local-Horizon $enginePath
    for ($step = 1; $step -le 3; $step++) {
        Save-Profile # Same sequence used before Play/Prepare/doctor
        $stored = Get-Profile
        if (-not [string]::Equals([string]$stored['client'], $enginePath,
                [StringComparison]::OrdinalIgnoreCase)) {
            throw "Save-Profile dropped the selected engine at step $step; stored='$($stored['client'])'"
        }
        if (-not [IO.File]::Exists([string]$stored['client'])) {
            throw "Stored engine path is no longer a file at step $step"
        }
    }
    Write-Output 'PASS: official local engine survives three profile saves with a spaced path'
} finally {
    Remove-Item Env:UHC_DATA_DIR -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $testRoot -Recurse -Force -ErrorAction SilentlyContinue
}
