#!/usr/bin/env python3
"""R5h: build the raw-tick mutation, require assertion failures, restore."""
import json
from pathlib import Path
import subprocess
import tempfile

source = Path('pkg/game/limits_condition.go').resolve()
original = source.read_text()
removed = original
for body in ('p', 'm'):
    for damage, kind in ((10, '33'), (13, '143'), (1, 'combat.TYPE_SUFFERING'), (2, 'combat.TYPE_SUFFERING')):
        old = f'w.selfDamage({body}, {damage}, {kind})'
        assert old in removed, old
        removed = removed.replace(old, f'{body}.TakeDamage({damage}); w.DamageBeforeMessage({body}, {body}, {damage})')
removed += '\nvar _ = combat.TYPE_SUFFERING\n'
command = ['go', 'test', './pkg/game', './pkg/session', '-run', '^TestPointUpdate(PoisonDamageTail|CutthroatDamageTail|WoundedDamageTail|MountedPoisonDamageTail|PoisonWireMatchesC|PoisonShopkeeperWire)$', '-count=1']

def run(arguments, fail=False):
    result = subprocess.run(arguments, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end='')
    if fail:
        assert result.returncode == 1 and '[build failed]' not in result.stdout
        for name in ('PoisonDamageTail', 'CutthroatDamageTail', 'WoundedDamageTail', 'MountedPoisonDamageTail', 'PoisonWireMatchesC', 'PoisonShopkeeperWire'):
            assert '--- FAIL: TestPointUpdate' + name in result.stdout, name
    else:
        assert result.returncode == 0

run(command)
with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    mutated = root / source.name
    mutated.write_text(removed)
    overlay = root / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(source): str(mutated)}}))
    run(command[:2] + ['-overlay=' + str(overlay)] + command[2:], fail=True)
run(command)

# R5h census finding: raw-only AFF readers miss spell-installed poison.
for filename, test in (("limits_condition.go", "TestPointUpdatePoisonDamageTail/spell-affect"), ("limits_gain.go", "TestRegenActiveSpellFlags")):
    path = Path("pkg/game") / filename
    path = path.resolve()
    text = path.read_text()
    mutation = text
    for flag in ("Poison", "Flaming", "Cutthroat"):
        mutation = mutation.replace(f"p.isAffectedLocked(Aff{flag})", f"p.Affects&(1<<Aff{flag}) != 0")
    assert mutation != text
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        replacement = root / filename
        replacement.write_text(mutation)
        overlay = root / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(path): str(replacement)}}))
        result = subprocess.run(["go", "test", "-overlay=" + str(overlay), "./pkg/game", "-run", "^" + test + "$", "-count=1"], text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        print(result.stdout, end="")
        assert result.returncode == 1 and "[build failed]" not in result.stdout
        assert "--- FAIL: " + test.split("/")[0] in result.stdout
    run(["go", "test", "./pkg/game", "-run", "^" + test + "$", "-count=1"])
