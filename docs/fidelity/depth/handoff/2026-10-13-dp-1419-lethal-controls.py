#!/usr/bin/env python3
"""DP-1419 lethal-replay R5h controls: each green must be able to fail.

Four mutations, each applied through a Go build overlay so the tree is never
touched:

A. drop raw_kill's NOWHERE guard — the pre-ruling behaviour: the death pipeline
   runs to death_cry, make_corpse and extract_char, so the character gets a
   corpse and is queued for extraction instead of staying POS_DEAD in place.
B. drop the CON-loss block — the RNG-gated decrement never happens.
C. restore the old inline exp computation — the -max_exp_loss clamp is lost.
D. drop the entering player's descriptor sink — C's death bytes never reach the
   descriptor, because the character has no world reference yet.

The tests must fail under each mutation and pass on the restored tree.
"""
import json
from pathlib import Path
import subprocess
import tempfile

DEATH = Path("pkg/game/death.go").resolve()
BAD_STATS = Path("pkg/game/bad_stats.go").resolve()

GAME_TESTS = "^TestReplayCrashLoadBadStatsLethal(StopsAtRawKill|ConLossDraw|ExpLossClamp)$"
SESSION_TESTS = "^TestCrashLoadLethalEntryLeavesCharacterDead$"


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


def game(extra=None):
    return ["go", "test", "./pkg/game", "-run", GAME_TESTS, "-count=1"] + (extra or [])


def session(extra=None):
    return ["go", "test", "./pkg/session", "-run", SESSION_TESTS, "-count=1"] + (extra or [])


run(game())
run(session())

death = DEATH.read_text()
bad_stats = BAD_STATS.read_text()

# A. No raw_kill NOWHERE guard.
guard = (
    "\tif roomVNum == roomNowhere {\n"
    "\t\tplayer.StopFighting()\n"
    "\t\tplayer.SetPosition(combat.PosDead)\n"
    "\t\treturn\n"
    "\t}\n"
)
assert guard in death, "guard anchor"
death_a = death.replace(guard, "", 1)
run(game([overlay(DEATH, death_a)]), fail=["TestReplayCrashLoadBadStatsLethalStopsAtRawKill"])
run(session([overlay(DEATH, death_a)]), fail=["TestCrashLoadLethalEntryLeavesCharacterDead"])

# B. No CON-loss block.
con_gate = "\t\tif playerLevel > ConLossMinLevel-1 {\n"
assert con_gate in death, "con anchor"
death_b = death.replace(con_gate, "\t\tif false {\n", 1)
run(game([overlay(DEATH, death_b)]), fail=["TestReplayCrashLoadBadStatsLethalConLossDraw"])

# C. Old inline exp computation (no gain_exp clamp).
gain = (
    "\tif isCombatDeath {\n"
    "\t\tw.GainExp(player, -(player.GetExp() / 37))\n"
    "\t} else {\n"
    "\t\tw.GainExp(player, -(player.GetExp() / 3))\n"
    "\t}\n"
)
assert gain in death, "gain anchor"
inline = (
    "\texpLoss := player.GetExp() / 37\n"
    "\tif !isCombatDeath {\n"
    "\t\texpLoss = player.GetExp() / 3\n"
    "\t}\n"
    "\tnewExp := player.GetExp() - expLoss\n"
    "\tif newExp < 0 {\n"
    "\t\tnewExp = 0\n"
    "\t}\n"
    "\tplayer.SetExp(newExp)\n"
)
death_c = death.replace(gain, inline, 1)
run(game([overlay(DEATH, death_c)]), fail=["TestReplayCrashLoadBadStatsLethalExpLossClamp"])

# D. No descriptor sink for the entering player.
sink = (
    "\tif deliver != nil {\n"
    "\t\tp.SetEntryTextSink(deliver)\n"
    "\t\tdefer p.SetEntryTextSink(nil)\n"
    "\t}\n"
)
assert sink in bad_stats, "sink anchor"
bad_stats_d = bad_stats.replace(sink, "", 1)
run(game([overlay(BAD_STATS, bad_stats_d)]), fail=["TestReplayCrashLoadBadStatsLethalStopsAtRawKill"])
run(session([overlay(BAD_STATS, bad_stats_d)]), fail=["TestCrashLoadLethalEntryLeavesCharacterDead"])

# Restored tree is green again.
run(game())
run(session())
