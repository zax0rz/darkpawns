#!/usr/bin/env python3
"""R5h controls for the DP-1371 PR 3 producers and proofs.

Each case removes exactly its producer (or restores the pre-fix behaviour or
breaks the guard its proof pins), requires the named test's assertion failure --
never a build failure -- then restores and re-proves green.

    python3 docs/fidelity/depth/handoff/2026-10-09-dp-1371-mudlog-closeout-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

LIMITS = "pkg/game/limits_exp.go"
BRIDGE = "pkg/scripting/bindings_bridge.go"
WIZ_ZONE = "pkg/session/wiz_zone.go"
DEATH = "pkg/game/death.go"

# limits.c:269-283: the producer itself.
AUTOWIZ_LOG_OLD = "\tMudLog(\"Initiating autowiz.\", MudlogComplete, LVL_IMMORT, false)\n"
AUTOWIZ_LOG_NEW = ""

# src/limits.c:357-360: C calls check_autowiz inside gain_exp_regardless whenever
# a level was gained, not only when the rise message is printed. Gating it on the
# announcement is the pre-fix shape that leaves do_advance's path silent.
AUTOWIZ_CALL_OLD = "\t\tCheckAutowiz(p)\n"
AUTOWIZ_CALL_NEW = "\t\tif announce {\n\t\t\tCheckAutowiz(p)\n\t\t}\n"

# scripts.c:791: the success arm writes the file.
LUA_LOG_OLD = "\tscriptMudLogFile(b, text, scriptMudlogBrief)\n"
LUA_LOG_NEW = "\tb.Log(text)\n"

# utils.c:721-723 at act.wizard.c:3471's call site.
HUNT_OLD = "\tif prey, isPlayer := victim.Combatant.(*game.Player); isPlayer {\n\t\tgame.LogHuntingStart(hunter.Mob, prey)\n\t}\n"
HUNT_NEW = ""

# The bounded caller proof's guard: Go re-resolves the mount in the rider's room
# and returns early. Crying for a mount that is not there is the C shape the
# proof excludes.
DEATH_GUARD_OLD = "\tif mount == nil {\n\t\tplayer.MountName = \"\"\n"
DEATH_GUARD_NEW = "\tif mount == nil {\n\t\tw.roomMessage(roomVNum, \"Your blood freezes as you hear a death cry.\\r\\n\")\n\t\tplayer.MountName = \"\"\n"

CASES = [
    {
        "case": "autowiz-producer",
        "test": "TestAutowizMudlogProducer",
        "package": "./pkg/session",
        "assertion": "Initiating autowiz.",
        "patches": [{"path": LIMITS, "new": AUTOWIZ_LOG_NEW, "old": AUTOWIZ_LOG_OLD}],
    },
    {
        "case": "autowiz-announce-gate",
        "test": "TestAutowizMudlogProducer",
        "package": "./pkg/session",
        "assertion": "Initiating autowiz.",
        "patches": [{"path": LIMITS, "new": AUTOWIZ_CALL_NEW, "old": AUTOWIZ_CALL_OLD}],
    },
    {
        "case": "lua-log-file-flag",
        "test": "TestLuaLogProducerFileFlag",
        "package": "./pkg/scripting",
        "assertion": "producers=[]",
        "patches": [{"path": BRIDGE, "new": LUA_LOG_NEW, "old": LUA_LOG_OLD}],
    },
    {
        "case": "hunting-producer",
        "test": "TestHuntingMudlogProducer",
        "package": "./pkg/session",
        "assertion": "started hunting Victim",
        "patches": [{"path": WIZ_ZONE, "new": HUNT_NEW, "old": HUNT_OLD}],
    },
    {
        "case": "deathcry-mount-guard",
        "test": "TestDeathCryBoundedCallers",
        "package": "./pkg/game",
        "assertion": "want only the rider's cry",
        "patches": [{"path": DEATH, "new": DEATH_GUARD_NEW, "old": DEATH_GUARD_OLD}],
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
