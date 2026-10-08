#!/usr/bin/env python3
"""R5h controls for the PR 1c sites.

Each case removes exactly its producer (or restores the pre-fix behaviour),
requires the named test's assertion failure -- never a build failure -- then
restores and re-proves green.

    python3 docs/fidelity/depth/handoff/2026-10-08-dp-1371-mudlog-1c-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

# comm.c:773-787. The producer is the file's only fmt use, so the revert drops
# the import too, and the whole repair body goes with it.
WEIGHT_BLOCK = "\n".join([
    "\tif obj.GetWeight() == obj.Prototype.Weight {",
    "\t\treturn",
    "\t}",
    "\tif len(obj.Contains) != 0 {",
    "\t\treturn",
    "\t}",
    '\tlocation := "unknown"',
    "\tswitch obj.Location.Kind {",
    "\tcase ObjInInventory:",
    '\t\tlocation = "carried"',
    "\tcase ObjInRoom:",
    '\t\tlocation = "in room"',
    "\tcase ObjEquipped:",
    '\t\tlocation = "worn by"',
    "\t}",
    '\tMudLog(fmt.Sprintf("SYSERR: Object \'%s\' weight incorrect, location \'%s\'",',
    "\t\tobj.GetShortDesc(), location), MudlogBrief, LVL_IMMORT, true)",
    "\tobj.SetWeight(obj.Prototype.Weight)",
]) + "\n"

CANGET_NIL = "\n".join([
    "\t\t// scripts.c:211-212: CAN_GET_OBJ failing pushes nil, not a number.",
    "\t\t// Lua treats 0 as true, so the port's 0 made a script's",
    '\t\t// "if canget(obj) then" take the wrong branch.',
    "\t\tL.Push(lua.LNil)",
]) + "\n"

CASES = [
    {
        "case": "object-weight",
        "test": "TestObjectActivityWeightMudlog",
        "package": "./pkg/game",
        "assertion": "file payload count=0 want 1",
        "patches": [{
            "path": "pkg/game/room_obj_scripts.go",
            "new": '\t"fmt"\n',
            "old": "",
        }, {
            "path": "pkg/game/room_obj_scripts.go",
            "new": WEIGHT_BLOCK,
            "old": "",
        }],
    },
    {
        "case": "canget-nil",
        "test": "TestLuaCanGetValidPathPushesNil",
        "package": "./pkg/scripting",
        "assertion": "disagreed with C",
        "patches": [{
            "path": "pkg/scripting/engine.go",
            "new": CANGET_NIL,
            "old": "\t\tL.Push(lua.LNumber(0))\n",
        }],
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
            assert reverted[patch["path"]].count(patch["new"]) == 1, (case["case"], patch["path"])
            reverted[patch["path"]] = reverted[patch["path"]].replace(patch["new"], patch["old"])
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
