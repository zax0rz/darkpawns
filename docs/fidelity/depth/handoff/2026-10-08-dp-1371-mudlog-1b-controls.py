#!/usr/bin/env python3
"""R5h controls for the PR 1b sites.

Each case removes exactly its producer or behaviour, requires the named test's
assertion failure -- never a build failure -- then restores and re-proves green.

    python3 docs/fidelity/depth/handoff/2026-10-08-dp-1371-mudlog-1b-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

# Reverting DP-1405 restores both halves: dominate leaves the charm case, and
# the invented default line comes back. With the case present the default arm is
# unreachable, so the invented text can only be provoked by reverting both.
DOMINATE_REVERT = [
    {
        "path": "pkg/spells/affect_spells.go",
        "old": "\tcase SpellCharm:\n",
        "new": "\tcase SpellCharm, SpellDominate:\n",
    },
    {
        "path": "pkg/spells/affect_spells.go",
        "new": (
            "\tdefault:\n"
            "\t\t// C's manual switch has no default clause at all (src/spell_parser.c:\n"
            "\t\t// 502-536): every MAG_MANUAL spell in the spell table has a case, so\n"
            "\t\t// this arm is unreachable and C emits nothing here. The port used to\n"
            '\t\t// send "Spell not yet implemented.\\r\\n", a string that appears in no\n'
            "\t\t// C file, which a live dominate hit (R4, DP-1405).\n"
            "\t}\n"
        ),
        "old": '\tdefault:\n\t\tsendToCaster(ch, "Spell not yet implemented.\\r\\n")\n\t}\n',
    },
]

CASES = [
    {
        "case": "dominate-class-audit",
        "test": "TestManualSpellsNeverPrintInventedText",
        "package": "./pkg/spells",
        "assertion": "printed the invented line",
    },
    {
        "case": "dominate-routing",
        "test": "TestDominateRoutesToCharm",
        "package": "./pkg/spells",
        "assertion": "charm=",
    },
]
for _case in CASES:
    _case["patches"] = DOMINATE_REVERT


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
