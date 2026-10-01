#!/usr/bin/env python3
"""Focused concurrent-claim and TCP-boundary assertion triples."""
from pathlib import Path
import subprocess
import sys

candidate, proof, logs = map(lambda p: Path(p).resolve(), sys.argv[1:4])
logs.mkdir(parents=True, exist_ok=True)
(proof / 'pkg/session/entry_name_test.go').write_bytes((candidate / 'pkg/session/entry_name_test.go').read_bytes())
(proof / 'pkg/telnet/entry_name_test.go').write_bytes((candidate / 'pkg/telnet/entry_name_test.go').read_bytes())
rows = ['proof\tbefore\tmutant\tafter\tfailure\n']
for name, file, old, new, package, symbol in [
    ('concurrent-claim', 'pkg/session/entry_name.go', 'if owner != s &&', 'if false && owner != s &&', './pkg/session', 'TestEntryNameConcurrentClaims'),
    ('security-list', 'pkg/validation/validation.go', '"zax0rz",', '"otherreserved",', './pkg/session', 'TestEntrySecurityReservedNames'),
    ('telnet-boundary', 'pkg/session/terminal.go', 'return s.terminalName(rawLine)', 'return s.terminalName(line)', './pkg/telnet', 'TestEntryNameTelnetBoundary'),
]:
    path = proof / file
    original = path.read_text()
    assert old in original
    codes = []
    try:
        for stage in ['before', 'mutant', 'after']:
            path.write_text(original.replace(old, new, 1) if stage == 'mutant' else original)
            result = subprocess.run(['go', 'test', package, '-run', '^' + symbol + '$', '-count=1'], cwd=proof, capture_output=True, text=True)
            output = result.stdout + result.stderr
            (logs / f'{name}-{stage}.log').write_text(output)
            codes.append(result.returncode)
            if stage == 'mutant':
                assert f'--- FAIL: {symbol}' in output and '[build failed]' not in output, output
    finally:
        path.write_text(original)
    assert codes == [0, 1, 0], codes
    rows.append(f'{name}\t0\t1\t0\tassertion\n')
    print(f'{name}: 0/1/0 assertion', flush=True)
(candidate / 'docs/fidelity/depth/evidence/2026-10-01-entry-name/boundary-revert-triples.tsv').write_text(''.join(rows))
