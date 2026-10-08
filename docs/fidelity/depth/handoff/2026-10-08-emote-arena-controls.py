#!/usr/bin/env python3
"""R5h: live emote proof fails without either the shared hook or the routing fix."""
import json
from pathlib import Path
import subprocess
import tempfile

command = ["go", "test", "./pkg/session", "-run", "^TestEmoteCommandArenaBroadcast$", "-count=1"]

def run(cmd, red=False):
    result = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end="")
    if red:
        assert result.returncode == 1 and "[build failed]" not in result.stdout
        assert "--- FAIL: TestEmoteCommandArenaBroadcast" in result.stdout
        assert "remote command output" in result.stdout
    else:
        assert result.returncode == 0

run(command)
act = Path("pkg/game/act.go").resolve()
text = act.read_text()
start = text.index("\t// src/comm.c:2529-2530: arena broadcast")
end = text.index("\t// Get all actors in the room", start)
emote = Path("pkg/session/comm_cmds.go").resolve()
old = subprocess.check_output(["git", "show", "99901e06b:pkg/session/comm_cmds.go"], text=True)
for source, removed in [(act, text[:start] + text[end:]), (emote, old)]:
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        replacement = root / source.name
        replacement.write_text(removed)
        overlay = root / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(source): str(replacement)}}))
        print("removal:", source)
        run(command[:2] + ["-overlay=" + str(overlay)] + command[2:], red=True)
    run(command)
