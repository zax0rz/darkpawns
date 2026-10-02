#!/usr/bin/env python3
"""Assertion-only 0/1/0 proofs in a disposable checkout; never edits the oracle."""
from pathlib import Path
import subprocess
import sys
candidate, proof, logs = [Path(p).resolve() for p in sys.argv[1:4]]
logs.mkdir(parents=True, exist_ok=True)
files = ['pkg/db/convert.go', 'pkg/game/character_data.go', 'pkg/session/entry_load_failure_test.go', 'pkg/telnet/entry_load_failure_test.go']
for name in files:
    (proof / name).write_bytes((candidate / name).read_bytes())
mutations = [
    ('object-preflight', 'pkg/db/convert.go', 'if err := game.ValidateCharacterData(r.CharacterData); err != nil {', 'if err := game.ValidateCharacterData(r.CharacterData); false && err != nil {', './pkg/session', 'TestEntryRestoreFailureNoObjects'),
    ('typed-schema', 'pkg/game/character_data.go', 'if err := json.Unmarshal(raw, &data); err != nil {', 'if err := json.Unmarshal(raw, &data); false && err != nil {', './pkg/session', 'TestEntryRestoreFailureFailsClosed'),
    ('restore-abort', 'pkg/session/session_login.go', 'return s.abortEntry(fmt.Errorf("restore character: %w", err))', 'return nil', './pkg/session', 'TestEntryRestoreFailureFailsClosed'),
    ('missing-password-record', 'pkg/session/session_login.go', 'return s.abortEntry(fmt.Errorf("character is no longer available"))', 'return nil', './pkg/session', 'TestEntryRecordDisappearsAtPassword'),
    ('legacy-control', 'pkg/game/character_data.go', 'return nil, nil', 'return nil, fmt.Errorf("mutated legacy rejection")', './pkg/session', 'TestEntryCharacterDataLegacyAndMissingControls'),
    ('transport-error', 'pkg/session/char_creation.go', 's.sendError(err.Error())', 's.sendError("mutated error")', './pkg/session ./pkg/telnet', 'TestEntryRestoreFailureWebSocket|TestEntryRestoreFailureTelnet'),
]
rows = ['mutation\tbefore\tmutant\tafter\tfailure']
for name, path, old, new, package, symbol in mutations:
    target = proof / path
    original = target.read_text()
    assert old in original, name
    codes = []
    try:
        for stage in ['before', 'mutant', 'after']:
            target.write_text(original.replace(old, new, 1) if stage == 'mutant' else original)
            result = subprocess.run(['go', 'test', *package.split(), '-run', '^(' + symbol + ')$', '-count=1'], cwd=proof, capture_output=True, text=True)
            text = result.stdout + result.stderr
            (logs / (name + '-' + stage + '.log')).write_text(text)
            codes.append(result.returncode)
            if stage == 'mutant':
                assert '--- FAIL: Test' in text and '[build failed]' not in text and 'panic:' not in text and 'i/o timeout' not in text, text
    finally:
        target.write_text(original)
    assert codes == [0, 1, 0], (name, codes)
    rows.append(name + '\t0\t1\t0\tassertion')
    print(name, '0/1/0 assertion', flush=True)
(candidate / 'docs/fidelity/depth/evidence/2026-10-01-entry-load-failure/revert-triples.tsv').write_text('\n'.join(rows) + '\n')
