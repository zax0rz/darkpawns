#!/usr/bin/env python3
"""R5h controls for DP-1417: the oedit corrupt-ex_desc producer and the extra
description presence it depends on.

Each case removes one piece -- the load-time presence flag, the writer's skip,
the producer, the re-edit presence fold, or the look guard -- and requires the
named test's assertion failure (never a build failure), then restores and
re-proves green.

    python3 docs/fidelity/depth/handoff/2026-10-10-dp-1417-oedit-presence-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

PARSER = "pkg/parser/obj.go"
OEDIT = "pkg/session/oedit.go"
LOOK = "pkg/game/look.go"

# The fread_string load rule (src/db.c:2736-2740): an empty tilde field is NULL.
FLAG_OLD = "\t\tNullKeyword:     keywords == \"\",\n"
FLAG_NEW = "\t\tNullKeyword:     false,\n"

# src/oedit.c:425 -- the "continue" that skips the corrupt record.
SKIP_OLD = (
    "\t\t\tgame.MudLog(\"SYSERR: OLC: oedit_save_to_disk: Corrupt ex_desc!\", game.MudlogBrief, game.LVL_IMMORT, true)\n"
    "\t\t\tcontinue\n"
)
SKIP_NEW = "\t\t\tgame.MudLog(\"SYSERR: OLC: oedit_save_to_disk: Corrupt ex_desc!\", game.MudlogBrief, game.LVL_IMMORT, true)\n"

# The producer itself (src/oedit.c:421-424).
PRODUCER_OLD = "\t\t\tgame.MudLog(\"SYSERR: OLC: oedit_save_to_disk: Corrupt ex_desc!\", game.MudlogBrief, game.LVL_IMMORT, true)\n"
PRODUCER_NEW = ""

# The presence fold that keeps an editor-allocated empty description "set".
FOLD_OLD = "\t\t\tkeywordSet:     !extra.NullKeyword,\n\t\t\tdescriptionSet: !extra.NullDescription,\n"
FOLD_NEW = "\t\t\tkeywordSet:     extra.Keywords != \"\",\n\t\t\tdescriptionSet: extra.Description != \"\",\n"

# find_exdesc returns a pointer; a NULL description is "not found".
GUARD_OLD = "\t\t\tif description.NullDescription {\n\t\t\t\treturn \"\", false\n\t\t\t}\n"
GUARD_NEW = ""

CASES = [
    {
        "case": "presence-flag",
        "test": "TestOeditSaveLoadedNullExtraLogsAndSkips",
        "package": "./pkg/session",
        "assertion": "want the BRF producer",
        "patches": [{"path": PARSER, "new": FLAG_NEW, "old": FLAG_OLD}],
    },
    {
        "case": "writer-skip",
        "test": "TestOeditSaveLoadedNullExtraLogsAndSkips",
        "package": "./pkg/session",
        "assertion": "the corrupt record must be skipped",
        "patches": [{"path": OEDIT, "new": SKIP_NEW, "old": SKIP_OLD}],
    },
    {
        "case": "producer",
        "test": "TestOeditSaveLoadedNullExtraLogsAndSkips",
        "package": "./pkg/session",
        "assertion": "want the BRF producer",
        "patches": [{"path": OEDIT, "new": PRODUCER_NEW, "old": PRODUCER_OLD}],
    },
    {
        "case": "reedit-text-inference",
        "test": "TestOeditReeditAllocatedEmptyIsSetNotNone",
        "package": "./pkg/session",
        "assertion": "must render as set but empty",
        "patches": [{"path": OEDIT, "new": FOLD_NEW, "old": FOLD_OLD}],
    },
    {
        "case": "look-guard",
        "test": "TestLookNullDescriptionFallsThroughToObjectView",
        "package": "./pkg/game",
        "assertion": "must fall through to the generic object view",
        "patches": [{"path": LOOK, "new": GUARD_NEW, "old": GUARD_OLD}],
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
