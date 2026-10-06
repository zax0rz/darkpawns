#!/usr/bin/env python3
"""R5h: each producer removal must compile and fail its named assertion."""
import argparse
import pathlib
import subprocess

CASES = [{'case': 'oedit', 'file': 'pkg/session/oedit.go', 'line': '\t\tgame.MudLog("SYSERR: OLC: Reached default case in oedit_parse()!", game.MudlogBrief, game.LVL_IMMORT, true)\n', 'test': 'TestMudlogLocalOeditDefault'}, {'case': 'sedit', 'file': 'pkg/session/sedit.go', 'line': '\t\tgame.MudLog("SYSERR: OLC: sedit_parse(): Reached default case!", game.MudlogBrief, game.LVL_IMMORT, true)\n', 'test': 'TestMudlogLocalSeditDefault'}, {'case': 'help', 'file': 'pkg/session/cmd_info.go', 'line': '\t\tgame.MudLog(fmt.Sprintf("HELP: %s attempted to get help on %s", s.playerName, argument), game.MudlogNormal, game.LVL_IMMORT, true)\n', 'test': 'TestMudlogLocalHelp'}, {'case': 'withdraw', 'file': 'pkg/game/clan_bank.go', 'line': '\t\tMudLog(fmt.Sprintf("%s withdraws %d coins from %s clan account.", ch.GetName(), amount, hshr(ch)), MudlogBrief, LVL_IMMORT, true)\n', 'test': 'TestMudlogLocalClanWithdraw'}, {'case': 'deposit', 'file': 'pkg/game/clan_bank.go', 'line': '\t\tMudLog(fmt.Sprintf("%s adds %d coins to %s clan account.", ch.GetName(), amount, hshr(ch)), MudlogBrief, LVL_IMMORT, true)\n', 'test': 'TestMudlogLocalClanDeposit'}]

parser = argparse.ArgumentParser()
parser.add_argument("--output", required=True)
args = parser.parse_args()
root = pathlib.Path(args.output)
root.mkdir(parents=True, exist_ok=True)
for case in CASES:
    path = pathlib.Path(case["file"])
    original = path.read_text()
    assert original.count(case["line"]) == 1, case
    try:
        for stage in ("green", "revert", "restore"):
            path.write_text(original.replace(case["line"], "") if stage == "revert" else original)
            result = subprocess.run(["go", "test", "./pkg/session", "-run", "^" + case["test"] + "$", "-count=1"], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            (root / (case["case"] + "-" + stage + ".txt")).write_text(result.stdout + "\nEXIT=" + str(result.returncode) + "\n")
            assert "[build failed]" not in result.stdout, result.stdout
            if stage == "revert":
                assert result.returncode != 0 and "--- FAIL: " + case["test"] in result.stdout, result.stdout
            else:
                assert result.returncode == 0, result.stdout
        print(case["case"] + ": 1 -> 0 -> 1", flush=True)
    finally:
        path.write_text(original)
