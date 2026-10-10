#!/usr/bin/env python3
"""R5h: remove death unmount and require immediate mounted-state assertion red."""
import json
from pathlib import Path
import subprocess
import tempfile

command = ['go', 'test', './pkg/game', '-run', '^TestMountDeathUnmountsImmediately$', '-count=1']
def run(args):
    result = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print(result.stdout, end='')
    return result
assert run(command).returncode == 0
source = Path('pkg/game/raw_kill.go').resolve()
original = source.read_text()
old = '\tcase *MobInstance:\n\t\trider, _ := w.combatFollowingBody(victim).(*Player)\n\t\tif rider == nil {\n\t\t\trider, _ = w.GetPlayer(victim.GetMountRider())\n\t\t}\n\t\tw.clearMountedPair(rider, victim)\n\t\tvictim.RemoveAffected(affMounted)\n'
assert old in original
with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    mutation = root / 'death.go'
    mutation.write_text(original.replace(old, '\tcase *MobInstance:\n\t\t// removed death unmount\n', 1))
    overlay = root / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(source): str(mutation)}}))
    result = run(command[:2] + ['-overlay=' + str(overlay)] + command[2:])
    assert result.returncode == 1 and '[build failed]' not in result.stdout
    assert '--- FAIL: TestMountDeathUnmountsImmediately' in result.stdout
    assert 'mount death left mounted state before extraction' in result.stdout
assert run(command).returncode == 0
