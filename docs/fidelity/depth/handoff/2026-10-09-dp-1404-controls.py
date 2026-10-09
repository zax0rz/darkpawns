#!/usr/bin/env python3
"""R5h controls for DP-1404: the stored last-exit rent code and Crash_load's
four live entry producers.

Each case removes one piece of behaviour (an arm's payload bytes, the cryo
write, or the producer itself) and requires the named test's assertion failure --
never a build failure -- then restores and re-proves green.

    python3 docs/fidelity/depth/handoff/2026-10-09-dp-1404-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

CRASH_LOAD = "pkg/session/crash_load.go"
CMD_INVENTORY = "pkg/session/cmd_inventory.go"

ENTRY_TEST = "TestCrashLoadEntryProducers"
QUIT_TEST = "TestQuitStoresLastExitRentCode"

# src/objsave.c:488-489, :519, :524, :528 -- the four live payloads.
NO_EQUIP_OLD = '"%s entering game with no equipment."'
NO_EQUIP_NEW = '"%s entering game with equipment."'
UNRENT_OLD = '"%s un-renting and entering game."'
UNRENT_NEW = '"%s renting and entering game."'
CRASH_OLD = '"%s retrieving crash-saved items and entering game."'
CRASH_NEW = '"%s retrieving saved items and entering game."'
CRYO_OLD = '"%s un-cryo\'ing and entering game."'
CRYO_NEW = '"%s cryo\'ing and entering game."'

# src/act.other.c:164-165 -- PLR_NODELETE takes RENT_CRYO.
CRYO_WRITE_OLD = "\t\t\tkind = rentCryo\n"
CRYO_WRITE_NEW = "\t\t\tkind = rentRented\n"

# The producer call itself. The mutation must still compile: `game` stays
# referenced (the package is otherwise imported only for this call).
PRODUCER_OLD = (
    "\t\t\tgame.MudLog(payload, game.MudlogNormal, max(game.LVL_IMMORT, "
    "s.player.GetInvisLevel()), true)\n"
)
PRODUCER_NEW = "\t\t\t_ = payload\n\t\t\t_ = game.MudlogNormal\n"

CASES = [
    {
        "case": "no-equipment",
        "test": ENTRY_TEST,
        "assertion": "Enterer entering game with no equipment.",
        "patches": [{"path": CRASH_LOAD, "new": NO_EQUIP_NEW, "old": NO_EQUIP_OLD}],
    },
    {
        "case": "unrenting",
        "test": ENTRY_TEST,
        "assertion": "Enterer un-renting and entering game.",
        "patches": [{"path": CRASH_LOAD, "new": UNRENT_NEW, "old": UNRENT_OLD}],
    },
    {
        "case": "crash-saved",
        "test": ENTRY_TEST,
        "assertion": "Enterer retrieving crash-saved items and entering game.",
        "patches": [{"path": CRASH_LOAD, "new": CRASH_NEW, "old": CRASH_OLD}],
    },
    {
        "case": "cryo",
        "test": ENTRY_TEST,
        "assertion": "Enterer un-cryo'ing and entering game.",
        "patches": [{"path": CRASH_LOAD, "new": CRYO_NEW, "old": CRYO_OLD}],
    },
    {
        "case": "rent-code",
        "test": QUIT_TEST,
        "assertion": "want kind 3",
        "patches": [{"path": CMD_INVENTORY, "new": CRYO_WRITE_NEW, "old": CRYO_WRITE_OLD}],
    },
    {
        "case": "producer-off",
        "test": ENTRY_TEST,
        "assertion": 'observer saw ""',
        "patches": [{"path": CRASH_LOAD, "new": PRODUCER_NEW, "old": PRODUCER_OLD}],
    },
]


def run(case, stage, root):
    result = subprocess.run(
        ["go", "test", "./pkg/session", "-run", "^" + case["test"] + "$", "-count=1"],
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
