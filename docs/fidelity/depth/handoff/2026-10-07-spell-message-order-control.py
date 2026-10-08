#!/usr/bin/env python3
"""R5h compiling revert control; overlay leaves checkout unchanged."""
import json
from pathlib import Path
import subprocess
import tempfile
source = Path("pkg/spells/damage_spells.go").resolve()
text = source.read_text()
start = text.index("\t\t// R1: damage() subtracts HP")
end = text.index("\t\t// C damage() still calls", start)
reverted = text[:start] + text[end:]
needle = "\n\t\t// Enter the wounded band"
assert reverted.count(needle) == 1
reverted = reverted.replace(needle, "\n\t\tvictCombat.TakeDamage(dam)\n" + needle)
command = ["go", "test", "./pkg/spells", "-run", "^TestSpellMessage", "-count=1"]
def run(cmd, red=False):
    result = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end="")
    if red:
        assert result.returncode != 0 and "--- FAIL: TestSpellMessage" in result.stdout
        assert "[build failed]" not in result.stdout
    else:
        assert result.returncode == 0
run(command)
with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    replacement = root / "damage_spells.go"
    replacement.write_text(reverted)
    overlay = root / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source): str(replacement)}}))
    run(command[:2] + ["-overlay=" + str(overlay)] + command[2:], red=True)
run(command)
