#!/usr/bin/env python3
"""R5h controls for PR 4's Lua seam proofs (DP-1371 D7).

Two groups:

* **Family producer removals (8).** The survey's group column splits the 57 rows
  into eight D7 families (lua-proof-01..08); each case removes one
  representative binding's producer and requires that family's subtest to fail
  on the tightened assertion. These are family-level, **not** 57 independent
  mutation controls: a green claims every family can fail.
* **Companion argument mutations (3).** The adapter's type, level and file
  arguments are mutated one at a time (BRF->CMP, LVL_IMMORT->LVL_IMMORT-1,
  FALSE->TRUE), each required to trip its own assertion in
  TestScriptAdapterLogIsBriefImmortalNoFile.

Every case runs green -> reverted (assertion red, never a build failure) ->
restored green:

    python3 docs/fidelity/depth/handoff/2026-10-11-dp-1371-pr4-lua-proofs-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

BINDINGS = "pkg/scripting/bindings_bridge.go"
ADAPTER = "pkg/game/world_bridge.go"
SCRIPTING_TEST = "TestLuaProofBadArgumentProducers"
ADAPTER_TEST = "TestScriptAdapterLogIsBriefImmortalNoFile"
ADAPTER_LINE = "\tMudLog(msg, MudlogBrief, lvlImmort, false)\n"

CASES = [
    # One representative per family: its payload literal is unique in the file,
    # so the patch asserts exactly one occurrence.
    {"case": "lua-proof-01", "file": BINDINGS, "package": "./pkg/scripting", "test": SCRIPTING_TEST,
     "site": "src/scripts.c:137", "old": '\tb.Log("[Lua] Invalid argument passed to lua_action.")\n',
     "new": "", "assertion": "log sequence ="},
    {"case": "lua-proof-02", "file": BINDINGS, "package": "./pkg/scripting", "test": SCRIPTING_TEST,
     "site": "src/scripts.c:451", "old": '\tb.Log("[Lua] Invalid argument passed to lua_exit_flags.")\n',
     "new": "", "assertion": "log sequence ="},
    {"case": "lua-proof-03", "file": BINDINGS, "package": "./pkg/scripting", "test": SCRIPTING_TEST,
     "site": "src/scripts.c:692", "old": '\tb.Log("[Lua] Invalid argument passed to lua_ishunt.")\n',
     "new": "", "assertion": "log sequence ="},
    {"case": "lua-proof-04", "file": BINDINGS, "package": "./pkg/scripting", "test": SCRIPTING_TEST,
     "site": "src/scripts.c:838", "old": '\tb.Log("[Lua] Invalid argument passed to lua_mob_flagged.")\n',
     "new": "", "assertion": "log sequence ="},
    {"case": "lua-proof-05", "file": BINDINGS, "package": "./pkg/scripting", "test": SCRIPTING_TEST,
     "site": "src/scripts.c:990", "old": '\tb.Log("[Lua] lua_oload returned an unknown object.")\n',
     "new": "", "assertion": "log sequence ="},
    {"case": "lua-proof-06", "file": BINDINGS, "package": "./pkg/scripting", "test": SCRIPTING_TEST,
     "site": "src/scripts.c:1255", "old": '\tb.Log("[Lua] Invalid arguments passed to lua_raw_kill.")\n',
     "new": "", "assertion": "log sequence ="},
    {"case": "lua-proof-07", "file": BINDINGS, "package": "./pkg/scripting", "test": SCRIPTING_TEST,
     "site": "src/scripts.c:1419", "old": '\tb.Log("[Lua] Unknown command passed to lua_social.")\n',
     "new": "", "assertion": "log sequence ="},
    {"case": "lua-proof-08", "file": BINDINGS, "package": "./pkg/scripting", "test": SCRIPTING_TEST,
     "site": "src/scripts.c:1604", "old": '\tb.Log("[Lua] Invalid argument passed to lua_unaffect.")\n',
     "new": "", "assertion": "log sequence ="},

    # The adapter's three arguments, one mutation each. A wrong type silences
    # the LOG1 observer, a wrong level lights up the level-30 observer, and a
    # wrong file flag writes the sink.
    {"case": "companion-type", "file": ADAPTER, "package": "./pkg/game", "test": ADAPTER_TEST,
     "site": "", "old": ADAPTER_LINE, "new": "\tMudLog(msg, MudlogComplete, lvlImmort, false)\n",
     "assertion": "BRF observer bytes"},
    {"case": "companion-level", "file": ADAPTER, "package": "./pkg/game", "test": ADAPTER_TEST,
     "site": "", "old": ADAPTER_LINE, "new": "\tMudLog(msg, MudlogBrief, lvlImmort-1, false)\n",
     "assertion": "observer saw"},
    {"case": "companion-file", "file": ADAPTER, "package": "./pkg/game", "test": ADAPTER_TEST,
     "site": "", "old": ADAPTER_LINE, "new": "\tMudLog(msg, MudlogBrief, lvlImmort, true)\n",
     "assertion": "file sink must be entirely empty"},
]


def run(case, stage, root):
    selector = "^" + case["test"] + "$"
    if case["site"]:
        selector += "/^" + case["site"].replace(".", "\\.") + "$"
    result = subprocess.run(
        ["go", "test", case["package"], "-run", selector, "-count=1"],
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
        original = pathlib.Path(case["file"]).read_text()
        assert original.count(case["old"]) == 1, (case["case"], case["file"])
        reverted = original.replace(case["old"], case["new"])
        try:
            for stage, source in (("green", original), ("revert", reverted), ("restore", original)):
                pathlib.Path(case["file"]).write_text(source)
                run(case, stage, root)
            suffix = f" ({case['site']})" if case["site"] else ""
            print(f"{case['case']}{suffix}: 1 -> 0 -> 1", flush=True)
        finally:
            pathlib.Path(case["file"]).write_text(original)


if __name__ == "__main__":
    main()
