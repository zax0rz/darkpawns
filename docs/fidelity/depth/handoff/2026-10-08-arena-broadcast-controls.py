#!/usr/bin/env python3
"""R5h: every arena proof fails on a compiling removal of the shared path."""
import json
from pathlib import Path
import subprocess
import tempfile

source = Path("pkg/game/act.go").resolve()
text = source.read_text()
start = text.index("\t// src/comm.c:2529-2530: arena broadcast")
end = text.index("\t// Get all actors in the room", start)
command = ["go", "test", "./pkg/game", "-run", "^TestArenaAct", "-count=1"]
names = ["RemoteAndLocal", "NoBroadcastToggle", "InvisibleActor", "NonArenaGate", "SleepWritingAndFullText"]

def run(cmd, red=False):
    result = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end="")
    if red:
        assert result.returncode == 1 and "[build failed]" not in result.stdout
        for name in names:
            assert "--- FAIL: TestArenaAct" + name in result.stdout, name
    else:
        assert result.returncode == 0

run(command)
with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    replacement = root / "act.go"
    replacement.write_text(text[:start] + text[end:])
    overlay = root / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source): str(replacement)}}))
    run(command[:2] + ["-overlay=" + str(overlay)] + command[2:], red=True)
run(command)
