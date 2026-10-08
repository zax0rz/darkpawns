#!/usr/bin/env python3
"""R5h controls for the PR 1a engine-diagnostic producers.

Each case removes exactly its producer (or its call sites), requires the named
test's assertion failure -- never a build failure -- then restores and re-proves
the test. Source is always restored, even on failure.

    python3 docs/fidelity/depth/handoff/2026-10-08-dp-1371-mudlog-engine-diagnostics-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

CASES = [
    {
        "case": "act-no-valid-target",
        "test": "TestActNoValidTargetMudlog",
        "package": "./pkg/game",
        "assertion": "file payload count=0 want 1",
        "patches": [{
            "path": "pkg/game/act.go",
            "new": "\t\tMudLog(\"SYSERR: no valid target to act()!\", MudlogComplete, LVL_IMMORT, true)\n\t\treturn\n\t}\n\n\t// Determine room VNum from ch or obj",
            "old": "\t\treturn\n\t}\n\n\t// Determine room VNum from ch or obj",
        }, {
            "path": "pkg/game/act.go",
            "new": "\t\tMudLog(\"SYSERR: no valid target to act()!\", MudlogComplete, LVL_IMMORT, true)\n\t\treturn\n\t}\n\n\t// Get all actors in the room",
            "old": "\t\treturn\n\t}\n\n\t// Get all actors in the room",
        }],
    },
    {
        "case": "zone-error-arms",
        "test": "TestZoneErrorMudlogBranches",
        "package": "./pkg/game",
        "assertion": "file payload count=0 want 1",
        "patches": [{
            "path": "pkg/game/spawner.go",
            "new": "\t\t\t\tzoneError(\"attempt to give obj to non-existant mob\")\n",
            "old": "",
        }, {
            "path": "pkg/game/spawner.go",
            "new": "\t\t\t\tzoneError(\"trying to equip non-existant mob\")\n",
            "old": "",
        }, {
            "path": "pkg/game/spawner.go",
            "new": "\t\t\t\tzoneError(\"invalid equipment pos number\")\n",
            "old": "",
        }, {
            "path": "pkg/game/spawner.go",
            "new": "\t\t\t\tzoneError(\"target obj not found\")\n",
            "old": "",
        }, {
            "path": "pkg/game/spawner.go",
            "new": "\t\t\t\tzoneError(\"door does not exist\")\n",
            "old": "",
        }, {
            "path": "pkg/game/spawner.go",
            "new": "\t\t\tzoneError(\"unknown cmd in reset table; cmd disabled\")\n",
            "old": "",
        }],
    },
    {
        "case": "lua-skip-spaces",
        "test": "TestLuaSkipSpacesInvalidArgumentProducer",
        "package": "./pkg/scripting",
        "assertion": "producer calls=0 want 1",
        "patches": [{
            "path": "pkg/scripting/engine.go",
            "new": "\tif L.Get(1).Type() != lua.LTString {\n\t\tif b := e.activeBridge; b != nil {\n\t\t\tb.Log(\"[Lua] Invalid argument passed to lua_skip_spaces.\")\n\t\t}\n\t}\n",
            "old": "",
        }],
    },
    {
        "case": "lua-load-failure",
        "test": "TestLuaRunScriptLoadFailureProducers",
        "package": "./pkg/scripting",
        "assertion": "producers=[] want",
        "patches": [{
            "path": "pkg/scripting/engine.go",
            "new": "\t\tscriptMudLogLoadFailure(bridge, fname, \"No such file.\")\n",
            "old": "",
        }, {
            "path": "pkg/scripting/engine.go",
            "new": "\t\t\tscriptMudLogLoadFailure(bridge, fname, luaLoadErrorKind(err))\n",
            "old": "",
        }],
    },
    {
        "case": "lua-call-failure",
        "test": "TestLuaRunScriptCallFailureProducer",
        "package": "./pkg/scripting",
        "assertion": "producers=[] want",
        "patches": [{
            "path": "pkg/scripting/engine.go",
            "new": "\t\t\tscriptMudLogCallFailure(bridge, fname, triggerName)\n",
            "old": "",
        }, {
            "path": "pkg/scripting/engine.go",
            "new": "\t\tscriptMudLogCallFailure(bridge, fname, triggerName)\n",
            "old": "",
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
            assert reverted[patch["path"]].count(patch["new"]) >= 1, (case["case"], patch["path"])
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
