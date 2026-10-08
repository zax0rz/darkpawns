#!/usr/bin/env python3
"""R5h: house floor/load proofs must fail against #1861's nesting code."""
import json
from pathlib import Path
import subprocess
import tempfile

source = Path('pkg/game/house_save.go').resolve()
command = ['go', 'test', './pkg/game', '-run', '^TestHouse(SaveLoadFloors|LoadExtractsUnrentable)', '-count=1']
names = ['TestHouseSaveLoadFloorsContainerContents', 'TestHouseSaveLoadFloorsDepthTwo', 'TestHouseLoadExtractsUnrentable']

def run(cmd, names_to_fail=()):
    result = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end='')
    if names_to_fail:
        assert result.returncode == 1 and '[build failed]' not in result.stdout
        for name in names_to_fail:
            assert '--- FAIL: ' + name in result.stdout, name
    else:
        assert result.returncode == 0

run(command)
old = subprocess.check_output(['git', 'show', '631fa67ee:pkg/game/house_save.go'], text=True)
no_extract = source.read_text().replace('\t\t\tw.ExtractObject(obj, vnum)\n', '')
for label, removed, expected in [('original nesting', old, names), ('extraction removed', no_extract, [names[-1]])]:
    print(label)
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        replacement = root / source.name
        replacement.write_text(removed)
        overlay = root / 'overlay.json'
        overlay.write_text(json.dumps({'Replace': {str(source): str(replacement)}}))
        run(command[:2] + ['-overlay=' + str(overlay)] + command[2:], expected)
    run(command)
