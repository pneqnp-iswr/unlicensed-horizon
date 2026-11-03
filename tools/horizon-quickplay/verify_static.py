"""Static safety regressions for the embedded Windows PowerShell scripts.

Not a replacement for exercising WinForms on Windows or parsing via PowerShell's
own AST parser on Windows.
"""
import re
from pathlib import Path

scripts = [Path('assets/QuickPlay.ps1'), Path('assets/AutoPrepare.ps1')]
for script in scripts:
    raw = script.read_bytes()
    assert raw.startswith(b'\xef\xbb\xbf'), f'{script}: missing BOM for PowerShell 5.1 Cyrillic'
    source = raw.decode('utf-8-sig')
    forbidden = re.compile(r'^\s*\$(?:global:|local:|script:|private:)?(?:home|host|pid|true|false|null|psversiontable|psculture|psuiculture)\s*(?:=|\+=|\?=)', re.I | re.M)
    # Ignore full-line comments but not executable statements.
    live_code = '\n'.join(line for line in source.splitlines() if not line.lstrip().startswith('#'))
    offenders = forbidden.findall(live_code)
    assert not offenders, f'{script}: assignment to read-only built-in variable: {offenders}'
    assert live_code.count('{') >= live_code.count('}'), f'{script}: unexpected brace balance'
qp = scripts[0].read_text(encoding='utf-8-sig')
assert '$gameHomeDir = [string]$previous[\'home\']' in qp
assert 'home_path = $gameHomeDir' in qp
assert not re.search(r'\$home\s*=', qp, re.I), 'legacy HOME assignment'
assert 'File]::Replace(' not in qp, 'v1.3 preferences fix regressed'
assert "Find-Local-Horizon" in qp, "no local engine autodetection"
assert "Use-Local-Horizon" in qp, "manual engine fallback absent"
assert "Get-Content -LiteralPath $filePath -Tail 12" in qp, "download stderr/stdout diagnostic absent"
assert "Проверь интернет. " not in qp, "misleading generic install error still present"
assert "Локальный Horizon" in qp, "manual engine picker button absent"
assert "$separator = $record.IndexOf('=')" in qp, "Windows PowerShell 5.1 profile parser regressed"
assert "client_path = $clientPath" in qp, "client_path no longer uses selected engine precedence"
assert "$script:selectedEnginePath = $absolutePath" in qp, "manual selection is not cached"
assert "$script:selectedEnginePath = $installedEngine" in qp, "installer output is not cached"
assert "Save-Profile" in qp[qp.index("function Use-Local-Horizon"):qp.index("function Play-Clicked")], "engine selection not persisted"
assert "Путь к движку не сохранился" in qp, "manual engine persistence not checked"
assert "Сначала нужно установить Horizon." not in qp, "misleading old engine error still present"
print('STATIC VALIDATION PASSED: PowerShell 5.1 BOM; protected variables; preferences; HOME; local engine fallback; accurate install logs; persistent selected engine')