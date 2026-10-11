#!/usr/bin/env python3
"""R5h: restore invented instakill POS_DEAD and require a wire assertion red."""
import json
from pathlib import Path
import subprocess
import tempfile

command = ['go', 'test', './pkg/session', '-run', '^TestInstakillFollowerNoticeBeforeMenuWire$', '-count=1']
def run(args):
    result = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end='')
    return result
assert run(command).returncode == 0
source = Path('pkg/game/death.go').resolve()
original = source.read_text()
old = '\tw.QueuePlayerExtraction(player)\n'
assert old in original
with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    mutation = root / 'death.go'
    mutation.write_text(original.replace(old, old + '\tplayer.SetPosition(combat.PosDead)\n', 1))
    overlay = root / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(source): str(mutation)}}))
    result = run(command[:2] + ['-overlay=' + str(overlay)] + command[2:])
    assert result.returncode == 1 and '[build failed]' not in result.stdout
    assert '--- FAIL: TestInstakillFollowerNoticeBeforeMenuWire/awake' in result.stdout
    assert 'victim extraction wire=' in result.stdout
assert run(command).returncode == 0
