#!/usr/bin/env python3
"""R5h controls for the DP-1416 unassigned-script producer and its proofs.

Each case removes one site's behaviour (the producer's bytes, or the empty-name
branch of one owner) and requires the named test's assertion failure -- never a
build failure -- then restores and re-proves green.

    python3 docs/fidelity/depth/handoff/2026-10-09-dp-1416-mudlog-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

SCRIPTS = "pkg/game/scripts.go"
ROOMOBJ = "pkg/game/room_obj_scripts.go"

# scripts.c:1763-1767: the producer's exact payload.
PAYLOAD_OLD = '"SYSERR: Attempting to call unassigned script for %s (#%d)."'
PAYLOAD_NEW = '"SYSERR: Attempting to call assigned script for %s (#%d)."'

# The mobile's empty-name arm (RunScript).
MOB_ARM_OLD = '\tif m.Proto().ScriptName == "" {\n'
MOB_ARM_NEW = '\tif false {\n'

# The room owner's empty-name arm (runRoomScript).
ROOM_ARM_OLD = '\tif room.ScriptName == "" {\n'
ROOM_ARM_NEW = '\tif false {\n'

# The object oncmd arm, disambiguated from the onpulse arm by its first comment
# line.
OBJ_ARM_OLD = '\tif obj.Prototype.ScriptName == "" {\n\t\t// run_script\'s !*script_name arm for an object owner\n'
OBJ_ARM_NEW = '\tif false {\n\t\t// run_script\'s !*script_name arm for an object owner\n'

# The object onpulse divergence guard.
ONPULSE_OLD = '\tif obj.Prototype.ScriptName == "" {\n\t\t// DP-1416 approved divergence: C\'s object-onpulse caller passes both\n'
ONPULSE_NEW = '\tif false {\n\t\t// DP-1416 approved divergence: C\'s object-onpulse caller passes both\n'

CASES = [
    {
        "case": "producer-bytes",
        "test": "TestUnassignedScriptMobProducer",
        "package": "./pkg/game",
        "assertion": "unassigned script for a goblin guard (#2001).",
        "patches": [{"path": SCRIPTS, "new": PAYLOAD_NEW, "old": PAYLOAD_OLD}],
    },
    {
        "case": "mob-arm",
        "test": "TestUnassignedScriptMobProducer",
        "package": "./pkg/game",
        "assertion": "the engine ran on an unassigned script",
        "patches": [{"path": SCRIPTS, "new": MOB_ARM_NEW, "old": MOB_ARM_OLD}],
    },
    {
        "case": "room-arm",
        "test": "TestUnassignedScriptRoomOnCmdConsumes",
        "package": "./pkg/game",
        "assertion": "the engine ran on an unassigned script",
        "patches": [{"path": ROOMOBJ, "new": ROOM_ARM_NEW, "old": ROOM_ARM_OLD}],
    },
    {
        "case": "obj-oncmd-arm",
        "test": "TestUnassignedScriptObjOnCmdConsumes",
        "package": "./pkg/game",
        "assertion": "the engine ran on an unassigned script",
        "patches": [{"path": ROOMOBJ, "new": OBJ_ARM_NEW, "old": OBJ_ARM_OLD}],
    },
    {
        "case": "onpulse-divergence",
        "test": "TestUnassignedScriptObjOnPulseIsSilentDivergence",
        "package": "./pkg/game",
        "assertion": "the engine ran on an unassigned script",
        "patches": [{"path": ROOMOBJ, "new": ONPULSE_NEW, "old": ONPULSE_OLD}],
    },
]


def run(case, stage, root):
    result = subprocess.run(
        ["go", "test", case["package"], "-run", "^" + case["test"] + "$", "-count=1"],
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
    )
    (root / f"{case['case']}-{stage}.txt").write_text(result.stdout + "\nEXIT=" + str(result.returncode) + "\n")
    assert "[build failed]" not in result.stdout, result.stdout
    if stage == "revert":
        assert result.returncode != 0, result.stdout
        assert "--- FAIL: " + case["test"] in result.stdout, result.stdout
        assert case["assertion"] in result.stdout, result.stdout
    else:
        assert result.returncode == 0, result.stdout


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    root = pathlib.Path(parser.parse_args().output)
    root.mkdir(parents=True, exist_ok=True)
    for case in CASES:
        originals = {p["path"]: pathlib.Path(p["path"]).read_text() for p in case["patches"]}
        reverted = dict(originals)
        for patch in case["patches"]:
            assert reverted[patch["path"]].count(patch["old"]) == 1, (case["case"], patch["path"])
            reverted[patch["path"]] = reverted[patch["path"]].replace(patch["old"], patch["new"])
        try:
            for stage, sources in (("green", originals), ("revert", reverted), ("restore", originals)):
                for path, source in sources.items():
                    pathlib.Path(path).write_text(source)
                run(case, stage, root)
            print(case["case"] + ": 1 -> 0 -> 1", flush=True)
        finally:
            for path, source in originals.items():
                pathlib.Path(path).write_text(source)


if __name__ == "__main__":
    main()
