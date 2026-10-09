#!/usr/bin/env python3
"""R5h: each teleport/view gate correction must independently fail."""
import json
from pathlib import Path
import subprocess
import tempfile

command = ["go", "test", "./pkg/session", "-run", "^TestTeleportMortallyWoundedRoomView$", "-count=1"]
def run(args, fail=False):
    result = subprocess.run(args, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end="")
    if fail:
        assert result.returncode == 1 and "[build failed]" not in result.stdout
        assert "--- FAIL: TestTeleportMortallyWoundedRoomView" in result.stdout
        assert "missing direct C room view" in result.stdout
    else:
        assert result.returncode == 0

run(command)
for name, old, new in (
    ("pkg/session/wiz_movement.go", "cmdMovementLook(targetSession)", "cmdLook(targetSession, nil)"),
    ("pkg/game/look.go", "ObservationResult{roomDelivery: true}", "ObservationResult{roomDelivery: false}"),
):
    source = Path(name).resolve()
    original = source.read_text()
    assert old in original
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        mutation = root / source.name
        mutation.write_text(original.replace(old, new, 1))
        overlay = root / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(source): str(mutation)}}))
        run(command[:2] + ["-overlay=" + str(overlay)] + command[2:], True)
    run(command)
