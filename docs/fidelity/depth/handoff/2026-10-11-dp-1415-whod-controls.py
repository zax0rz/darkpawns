#!/usr/bin/env python3
"""R5h controls for DP-1415: the WHOD formatter, the listener filters and the
socket-lifecycle producers.

Each case removes one piece and requires the named test's assertion failure
(never a build failure), then restores and re-proves green.

    python3 docs/fidelity/depth/handoff/2026-10-11-dp-1415-whod-controls.py \
        --output /absolute/evidence/path
"""
import argparse
import pathlib
import subprocess

WHOD = "pkg/game/whod.go"
LISTEN = "pkg/session/whod_listener.go"

# SHORT_GREETINGS (src/config.c:221-231): drop one space from the banner line.
GREETING_OLD = '"        /                         |||\\n\\r" +\n'
GREETING_NEW = '"        /                        |||\\n\\r" +\n'

# The list header (src/whod.c:341).
HEADER_OLD = "\tsb.WriteString(WhodHeader())\n\n"
HEADER_NEW = "\n"

# The IS_INVIS gate (src/whod.c:345).
INVIS_OLD = "\t\tif entry.IsInvisible && mode&WhodShowInvis == 0 {\n"
INVIS_NEW = "\t\tif false {\n"

# The served producer (src/whod.c:335), logged before any byte is written.
SERVED_OLD = '\tgame.MudLog(fmt.Sprintf("WHO request from %s served.", host), game.MudlogBrief, game.LVL_GOD, true)\n'
SERVED_NEW = ""

# DISPLAY_LINKDEAD is 0: C requires ch->desc (src/whod.c:45,344).
LINKLESS_OLD = "\t\tif p == nil || p.IsLinkless() {\n"
LINKLESS_NEW = "\t\tif p == nil {\n"

# The close producer (src/whod.c:267) on the SHOW_OFF reconcile.
CLOSE_OLD = (
    "\t\tif l.closeListener() {\n"
    '\t\t\tgame.MudLog("WHOD port closed.", game.MudlogBrief, game.LVL_GOD, true)\n'
    "\t\t}\n"
)
CLOSE_NEW = "\t\tl.closeListener()\n"

CASES = [
    {
        "case": "greeting-bytes",
        "test": "TestWhodGreetingMatchesConfigBytes",
        "package": "./pkg/game",
        "assertion": "greeting length",
        "patches": [{"path": WHOD, "new": GREETING_NEW, "old": GREETING_OLD}],
    },
    {
        "case": "list-header",
        "test": "TestWhodBuildWhoListDefaultMode",
        "package": "./pkg/game",
        "assertion": "default-mode bytes",
        "patches": [{"path": WHOD, "new": HEADER_NEW, "old": HEADER_OLD}],
    },
    {
        "case": "invisible-gate",
        "test": "TestWhodBuildWhoListInvisibleAndSite",
        "package": "./pkg/game",
        "assertion": "an invisible player leaked without SHOW_INVIS",
        "patches": [{"path": WHOD, "new": INVIS_NEW, "old": INVIS_OLD}],
    },
    {
        "case": "linkless-filter",
        "test": "TestWhodListenerHidesInvisibleAndLinkless",
        "package": "./pkg/session",
        "assertion": "linkless player leaked",
        "patches": [{"path": LISTEN, "new": LINKLESS_NEW, "old": LINKLESS_OLD}],
    },
    {
        "case": "served-producer",
        "test": "TestWhodListenerProducers",
        "package": "./pkg/session",
        "assertion": "missing file payload",
        "patches": [{"path": LISTEN, "new": SERVED_NEW, "old": SERVED_OLD}],
    },
    {
        "case": "close-producer",
        "test": "TestWhodListenerOffClosesAndOnReopens",
        "package": "./pkg/session",
        "assertion": "closed producer count",
        "patches": [{"path": LISTEN, "new": CLOSE_NEW, "old": CLOSE_OLD}],
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
