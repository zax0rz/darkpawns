#!/usr/bin/env python3
"""R5h controls for PR 4's 57 Lua seam proofs (DP-1371 D7).

These are **family-level** controls, not 57 independent mutation controls: the
survey's group column splits the 57 rows into eight D7 families
(lua-proof-01..08), and each case removes the producer of one representative
binding from that family and requires its named subtest to fail on the
tightened assertion. A green here does not claim each of the 57 was mutated;
it claims every family can fail.

Each case runs green -> reverted (assertion red, never a build failure) ->
restored green:

    python3 docs/fidelity/depth/handoff/2026-10-11-dp-1371-pr4-lua-proofs-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

BINDINGS = "pkg/scripting/bindings_bridge.go"
PACKAGE = "./pkg/scripting"
TEST = "TestLuaProofBadArgumentProducers"

# One representative per family: its payload literal is unique in the file, so
# the patch asserts exactly one occurrence.
CASES = [
    ("lua-proof-01", "src/scripts.c:137", '\tb.Log("[Lua] Invalid argument passed to lua_action.")\n'),
    ("lua-proof-02", "src/scripts.c:451", '\tb.Log("[Lua] Invalid argument passed to lua_exit_flags.")\n'),
    ("lua-proof-03", "src/scripts.c:692", '\tb.Log("[Lua] Invalid argument passed to lua_ishunt.")\n'),
    ("lua-proof-04", "src/scripts.c:838", '\tb.Log("[Lua] Invalid argument passed to lua_mob_flagged.")\n'),
    ("lua-proof-05", "src/scripts.c:990", '\tb.Log("[Lua] lua_oload returned an unknown object.")\n'),
    ("lua-proof-06", "src/scripts.c:1255", '\tb.Log("[Lua] Invalid arguments passed to lua_raw_kill.")\n'),
    ("lua-proof-07", "src/scripts.c:1419", '\tb.Log("[Lua] Unknown command passed to lua_social.")\n'),
    ("lua-proof-08", "src/scripts.c:1604", '\tb.Log("[Lua] Invalid argument passed to lua_unaffect.")\n'),
]

ASSERTION = "log sequence ="


def run(family, site, stage, root):
    selector = "^" + TEST + "$/^" + site.replace(".", "\\.") + "$"
    result = subprocess.run(
        ["go", "test", PACKAGE, "-run", selector, "-count=1"],
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
    )
    (root / f"{family}-{stage}.txt").write_text(result.stdout + "\nEXIT=" + str(result.returncode) + "\n")
    assert "[build failed]" not in result.stdout, result.stdout
    if stage == "revert":
        assert result.returncode != 0, result.stdout
        assert "--- FAIL: " + TEST + "/" + site in result.stdout, result.stdout
        assert ASSERTION in result.stdout, result.stdout
    else:
        assert result.returncode == 0, result.stdout


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    root = pathlib.Path(parser.parse_args().output)
    root.mkdir(parents=True, exist_ok=True)
    original = pathlib.Path(BINDINGS).read_text()
    for family, site, line in CASES:
        assert original.count(line) == 1, (family, site)
        reverted = original.replace(line, "")
        try:
            for stage, source in (("green", original), ("revert", reverted), ("restore", original)):
                pathlib.Path(BINDINGS).write_text(source)
                run(family, site, stage, root)
            print(f"{family} ({site}): 1 -> 0 -> 1", flush=True)
        finally:
            pathlib.Path(BINDINGS).write_text(original)


if __name__ == "__main__":
    main()
