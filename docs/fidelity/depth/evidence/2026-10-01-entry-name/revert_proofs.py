#!/usr/bin/env python3
"""Assertion-only unit revert triples in an isolated worktree (R5h)."""
from pathlib import Path
import subprocess
import sys

candidate = Path(sys.argv[1]).resolve()
proof = Path(sys.argv[2]).resolve()
logs = Path(sys.argv[3]).resolve()
logs.mkdir(parents=True, exist_ok=True)
files = subprocess.check_output(['git', 'diff', '--name-only'], cwd=candidate, text=True).splitlines()
files += subprocess.check_output(['git', 'ls-files', '--others', '--exclude-standard'], cwd=candidate, text=True).splitlines()
for file in files:
    if file.endswith('.go'):
        target = proof / file
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes((candidate / file).read_bytes())

mutations = [
 ('leading', 'pkg/session/entry_name.go', 'strings.TrimLeft(raw, " \\t\\n\\r\\v\\f")', 'raw'),
 ('trailing', 'pkg/session/entry_name.go', 'strings.TrimLeft(raw, " \\t\\n\\r\\v\\f")', 'strings.TrimSpace(raw)'),
 ('min-length', 'pkg/session/entry_name.go', 'len(name) < 2', 'len(name) < 3'),
 ('max-length', 'pkg/session/entry_name.go', 'len(name) > 20', 'len(name) > 19'),
 ('alphabetic', 'pkg/session/entry_name.go', "if (letter < 'a'", "if false && (letter < 'a'"),
 ('fill-reserved', 'pkg/session/entry_name.go', 'switch strings.ToLower(name)', 'switch strings.ToLower(name + "z")'),
 ('empty-close', 'pkg/session/entry_name.go', 's.CloseSend()', '// omit close'),
 ('held-name', 'pkg/session/entry_name.go', 'if owner != s &&', 'if false && owner != s &&'),
 ('nonplaying', 'pkg/session/entry_name.go', 'if candidate.charCreating || candidate.menuActive || candidate.inOLCEditorState()', 'if false && (candidate.charCreating || candidate.menuActive || candidate.inOLCEditorState())'),
 ('banned', 'pkg/session/entry_name.go', 'if !allowed && !game.ValidNameNoActive(name)', 'if false && !allowed && !game.ValidNameNoActive(name)'),
 ('playing-override', 'pkg/session/entry_name.go', 'if !allowed && !game.ValidNameNoActive(name)', 'if !game.ValidNameNoActive(name) && !allowed || !game.ValidNameNoActive(name)'),
 ('terminal-bytes', 'pkg/session/terminal.go', 'return s.terminalName(rawLine)', 'return s.terminalName(line)'),
 ('retry-bytes', 'pkg/session/char_creation.go', 's.charStage == "get_name" || s.charStage == "create_password"', 's.charStage == "create_password"'),
 ('N-release', 'pkg/session/char_creation.go', 's.releaseEntryName()', '// omit N release'),
 ('close-release', 'pkg/session/session_player.go', 's.releaseEntryName()', '// omit close release'),
 ('world-release', 'pkg/session/char_creation.go', '// Clear char creation state\n\ts.releaseEntryName()', '// Clear char creation state'),
 ('security-overlay', 'pkg/session/entry_name.go', 'validation.IsReservedPlayerName(name)', '(false && validation.IsReservedPlayerName(name))'),
 ('password-identity', 'pkg/session/session_login.go', 'login.PlayerName = s.charName', 'login.PlayerName = strings.TrimSpace(login.PlayerName)'),
 ('guest-handoff', 'pkg/session/session_login.go', 's.releaseEntryName()', '// omit guest handoff release'),
 ('guest-prefix', 'pkg/session/session_login.go', 'if strings.HasPrefix(strings.ToLower(login.PlayerName), "guest")', 'if false && strings.HasPrefix(strings.ToLower(login.PlayerName), "guest")'),
]
pattern = '^(TestEntryName|TestEntryGuestPrefixApproved|TestEntryGuestHandoffReleasesName|TestEntrySecurityReservedNames)'

def run(name, stage):
    result = subprocess.run(['go', 'test', './pkg/session', '-run', pattern, '-count=1'], cwd=proof, capture_output=True, text=True)
    output = result.stdout + result.stderr
    (logs / f'{name}-{stage}.log').write_text(output)
    return result.returncode, output

rows = ['mutation\tbefore\tmutant\tafter\tfailure\n']
for name, file, old, new in mutations:
    path = proof / file
    original = path.read_text()
    if old not in original:
        raise SystemExit(f'missing mutation anchor: {name}')
    before, _ = run(name, 'before')
    try:
        path.write_text(original.replace(old, new, 1))
        mutant, output = run(name, 'mutant')
    finally:
        path.write_text(original)
    after, _ = run(name, 'after')
    if (before, mutant, after) != (0, 1, 0) or '--- FAIL: TestEntry' not in output or '[build failed]' in output:
        raise SystemExit(f'rejected triple {name}: {before}/{mutant}/{after}')
    rows.append(f'{name}\t0\t1\t0\tassertion\n')
    print(f'{name}: 0/1/0 assertion', flush=True)
(candidate / 'docs/fidelity/depth/evidence/2026-10-01-entry-name/revert-triples.tsv').write_text(''.join(rows))
