#!/usr/bin/env python3
"""DP-1419 R5h controls: prove the per-item replay green can fail.

Two mutations, each applied through a Go build overlay so the tree is never
touched:

A. no check on restore — the pre-DP-1419 behaviour. The menu entry never calls
   ReplayCrashLoadBadStats, so a returning character with saved zero-stat gear
   receives nothing and keeps full HP.
B. a single check on the final stats — the tempting shortcut. The replay runs
   once over all worn items instead of once per equip, so the repeated DEX
   firings collapse into one (HP 160 instead of 40, one line instead of four).

The tests must fail under each mutation and pass on the restored tree.
"""
import json
from pathlib import Path
import subprocess
import tempfile

MENU = Path("pkg/session/menu.go").resolve()
BAD_STATS = Path("pkg/game/bad_stats.go").resolve()

SESSION_TESTS = ["^TestCrashLoadBadStats"]
GAME_TESTS = ["^TestReplayCrashLoadBadStatsFiresPerEquippedItem$"]


def run(arguments, fail=None):
    result = subprocess.run(arguments, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end="")
    if fail is None:
        assert result.returncode == 0, result.returncode
        return
    assert result.returncode == 1 and "[build failed]" not in result.stdout
    for name in fail:
        assert "--- FAIL: " + name in result.stdout, name


def overlay(source, mutation):
    directory = tempfile.mkdtemp()
    replacement = Path(directory) / source.name
    replacement.write_text(mutation)
    path = Path(directory) / "overlay.json"
    path.write_text(json.dumps({"Replace": {str(source): str(replacement)}}))
    return "-overlay=" + str(path)


def session_command(extra=None):
    return ["go", "test", "./pkg/session", "-run", "|".join(SESSION_TESTS), "-count=1"] + (extra or [])


def game_command(extra=None):
    return ["go", "test", "./pkg/game", "-run", "|".join(GAME_TESTS), "-count=1"] + (extra or [])


# Green first.
run(session_command())
run(game_command())

# A. No check on restore: the entry never replays the per-item equips.
menu = MENU.read_text()
anchor = "\tif loaded {\n\t\ts.manager.world.ReplayCrashLoadBadStats(s.player, s.sendPlayerText)\n\t}\n"
assert anchor in menu, anchor
menu_mutation = menu.replace(anchor, "\tif false && loaded {\n\t\ts.manager.world.ReplayCrashLoadBadStats(s.player, s.sendPlayerText)\n\t}\n", 1)
run(session_command([overlay(MENU, menu_mutation)]), fail=[
    "TestCrashLoadBadStatsDexFiresPerRestoredItem",
    "TestCrashLoadBadStatsSingleItemArms",
    "TestCrashLoadBadStatsCharismaStartsHunting",
])

# B. One check on the final stats: the per-equip replay collapses to one firing.
bad_stats = BAD_STATS.read_text()
loop = (
    "\titems := COrderedWorn(p)\n"
    "\tfor i := range items {\n"
    "\t\tp.mu.RLock()\n"
    "\t\tstats := p.affectTotalForItemsLocked(items[:i+1])\n"
    "\t\tp.mu.RUnlock()\n"
    "\t\tw.applyBadStatDeliver(p, badStat(stats), deliver)\n"
    "\t}\n"
)
assert loop in bad_stats, loop
once = (
    "\titems := COrderedWorn(p)\n"
    "\tif len(items) > 0 {\n"
    "\t\tp.mu.RLock()\n"
    "\t\tstats := p.affectTotalForItemsLocked(items)\n"
    "\t\tp.mu.RUnlock()\n"
    "\t\tw.applyBadStatDeliver(p, badStat(stats), deliver)\n"
    "\t}\n"
)
bad_stats_mutation = bad_stats.replace(loop, once, 1)
run(session_command([overlay(BAD_STATS, bad_stats_mutation)]), fail=["TestCrashLoadBadStatsDexFiresPerRestoredItem"])
run(game_command([overlay(BAD_STATS, bad_stats_mutation)]), fail=["TestReplayCrashLoadBadStatsFiresPerEquippedItem"])

# Restored tree is green again.
run(session_command())
run(game_command())
