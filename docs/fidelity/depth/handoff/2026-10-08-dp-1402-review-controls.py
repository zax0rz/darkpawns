#!/usr/bin/env python3
"""Compiling removal controls for Claude's movement and nil-attacker findings."""
import json
from pathlib import Path
import subprocess
import tempfile

def run(name, source=None, replacement=None):
    command = ['go', 'test', './pkg/game', '-run', '^' + name + '$', '-count=1']
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        if source:
            file = root / source.name
            file.write_text(replacement)
            overlay = root / 'overlay.json'
            overlay.write_text(json.dumps({'Replace': {str(source): str(file)}}))
            command[2:2] = ['-overlay=' + str(overlay)]
        result = subprocess.run(command, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        print(result.stdout, end='')
        if source:
            assert result.returncode == 1 and '[build failed]' not in result.stdout
            assert '--- FAIL: ' + name in result.stdout
        else:
            assert result.returncode == 0

movement = Path('pkg/game/damage_before_message.go').resolve()
text = movement.read_text()
start = text.index('\tif err := w.transferBody(p, 8004, false);')
end = text.index('\tw.lookAtRoom', start)
old = (text[:start] + '\tp.SetRoom(8004)\n' + text[end:]).replace('\t"log/slog"\n', '')
skill = Path('pkg/game/damage_stubs.go').resolve()
old_skill = skill.read_text().replace('if v := combatantFromInterface(victim); v != nil {', 'if v := combatantFromInterface(victim); killer != nil && v != nil {')
for name, source, removed in [('TestNeutralRescueRoomTransfer', movement, old), ('TestSkillDamageNilAttacker', skill, old_skill)]:
    run(name)
    run(name, source, removed)
    run(name)
