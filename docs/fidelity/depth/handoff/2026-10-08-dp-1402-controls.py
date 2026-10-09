#!/usr/bin/env python3
"""R5h: a compiling removal of the shared pre-message damage block."""
import json
from pathlib import Path
import subprocess
import tempfile

source = Path("pkg/combat/damage_before_message.go").resolve()
command = ["go", "test", "./pkg/game", "./pkg/command", "-run", "^(TestSpellNeutralRescueBeforeMessage|TestMeleeNeutralRescueBeforeMessage|TestDamageNewbieExperienceBeforeMessage|TestSpellLowLevelStopBeforeMessage|TestNeutralRescueDamageTailClass|TestNeutralRescueActorAndMount|TestSkillCommandNeutralRescueBeforeMessage)$", "-count=1"]

def run(cmd, red=False):
    result = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end="")
    if red:
        assert result.returncode == 1
        assert "[build failed]" not in result.stdout
        for name in ["TestSpellNeutralRescueBeforeMessage", "TestMeleeNeutralRescueBeforeMessage", "TestDamageNewbieExperienceBeforeMessage", "TestSpellLowLevelStopBeforeMessage", "TestNeutralRescueDamageTailClass", "TestNeutralRescueActorAndMount", "TestSkillCommandNeutralRescueBeforeMessage"]:
            assert "--- FAIL: " + name in result.stdout, name
    else:
        assert result.returncode == 0

run(command)
with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    replacement = root / source.name
    replacement.write_text("package combat\nfunc DamageBeforeMessage(ch, victim Combatant, dam int, cb *GameCallbacks) bool { return false }\n")
    overlay = root / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source): str(replacement)}}))
    run(command[:2] + ["-overlay=" + str(overlay)] + command[2:], red=True)
run(command)
