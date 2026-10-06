#!/usr/bin/env python3
"""Reproducible R5h controls: assertion failures, never build failures."""
import argparse
import pathlib
import subprocess

CASES = [{'case': 'helpbody', 'file': 'pkg/session/cmd_info.go', 'new': '\t\ts.manager.mu.RLock()\n\t\tname := s.player.GetName()\n\t\tif s.isSwitched && s.switchedMob != nil {\n\t\t\tname = s.switchedMob.GetName()\n\t\t}\n\t\ts.manager.mu.RUnlock()\n\t\tgame.MudLog(fmt.Sprintf("HELP: %s attempted to get help on %s", name, argument), game.MudlogNormal, game.LVL_IMMORT, true)\n', 'old': '\t\tgame.MudLog(fmt.Sprintf("HELP: %s attempted to get help on %s", s.playerName, argument), game.MudlogNormal, game.LVL_IMMORT, true)\n', 'anchor': '', 'position': 'replace', 'imports': False, 'test': 'TestHelpMudlogActingBody', 'pkg': './pkg/session', 'wrapper': 'pkg/session/help_body_mudlog_test.go'}, {'case': 'advance', 'file': 'pkg/game/level.go', 'new': '\tMudLog(fmt.Sprintf("%s advanced to level %d", name, level), MudlogBrief, max(LVL_IMMORT, p.GetInvisLevel()), true)\n', 'old': '\tslog.Info("advanced to level", "name", name, "level", level)\n', 'anchor': '', 'position': 'replace', 'imports': True, 'test': 'TestMilestoneMudlogAdvance', 'pkg': './pkg/game', 'wrapper': 'pkg/game/milestone_mudlog_advance_test.go'}, {'case': 'stable', 'file': 'pkg/game/spec_procs2.go', 'new': '\t\t\tMudLog("Mount not loaded in stable", MudlogBrief, LVL_GRGOD, true)\n', 'old': '', 'anchor': '\t\t\ttellFromMob(me, ch, "Sorry, we are unable to gather your mount, try back later.")\n', 'position': 'after', 'imports': False, 'test': 'TestMilestoneMudlogStable', 'pkg': './pkg/game', 'wrapper': 'pkg/game/milestone_mudlog_stable_test.go'}, {'case': 'remort', 'file': 'pkg/game/spec_procs2.go', 'new': '\tMudLog("Due to remorting:", MudlogBrief, LVL_IMMORT, true)\n', 'old': '', 'anchor': '\tch.AffectTotal() // src/spec_procs2.c:942\n', 'position': 'after', 'imports': False, 'test': 'TestMilestoneMudlogRemort', 'pkg': './pkg/game', 'wrapper': 'pkg/game/milestone_mudlog_remort_test.go'}, {'case': 'assassinpc', 'file': 'pkg/game/spec_procs2.go', 'new': '\t\t\tMudLog(fmt.Sprintf("%s is in the assassin store room.", player.GetName()), MudlogBrief, LVL_IMMORT, true)\n', 'old': '\t\t\tslog.Info("player found in assassin store room", "player", player.GetName())\n', 'anchor': '', 'position': 'replace', 'imports': False, 'test': 'TestMilestoneMudlogAssassinPC', 'pkg': './pkg/game', 'wrapper': 'pkg/game/milestone_mudlog_assassinpc_test.go'}, {'case': 'assassinhire', 'file': 'pkg/game/spec_procs2.go', 'new': '\t\tMudLog(fmt.Sprintf("%s hires %s to kill %s.\\r\\n", ch.GetName(), hired.GetName(), victim.GetName()), MudlogBrief, LVL_IMMORT, true)\n', 'old': '', 'anchor': '\t\tAct(w, false, ch, hired, nil, nil, "$n hires $N for a job.", "", ToRoom)\n', 'position': 'after', 'imports': False, 'test': 'TestMilestoneMudlogAssassinHire', 'pkg': './pkg/game', 'wrapper': 'pkg/game/milestone_mudlog_assassinhire_test.go'}, {'case': 'medusa', 'file': 'pkg/game/spec_procs2.go', 'new': '\tMudLog(fmt.Sprintf("%s killed by Medusa special at %s", ch.GetName(), w.GetRoomInWorld(ch.GetRoom()).Name), MudlogBrief, LVL_IMMORT, true)\n', 'old': '', 'anchor': '\tch.Deaths++\n', 'position': 'before', 'imports': False, 'test': 'TestMilestoneMudlogMedusa', 'pkg': './pkg/game', 'wrapper': 'pkg/game/milestone_mudlog_medusa_test.go'}, {'case': 'loot', 'file': 'pkg/game/attitude_loot.go', 'new': '\tif !victim.IsNPC() {\n\t\tMudLog(fmt.Sprintf("(LOOT) %s attitude looted %s.", killer.GetName(), victim.GetName()), MudlogComplete, LVL_IMMORT, true)\n\t}\n', 'old': '', 'anchor': '}\n\nfunc (w *World) findAttitudeLootCorpse', 'position': 'before', 'imports': True, 'test': 'TestMilestoneMudlogLoot', 'pkg': './pkg/game', 'wrapper': 'pkg/game/milestone_mudlog_loot_test.go'}]
parser=argparse.ArgumentParser()
parser.add_argument("--output",required=True)
args=parser.parse_args()
root=pathlib.Path(args.output);root.mkdir(parents=True,exist_ok=True)
for case in CASES:
    path=pathlib.Path(case["file"]);original=path.read_text()
    assert original.count(case["new"])==1,case
    reverted=original.replace(case["new"],case["old"])
    if case["imports"]:reverted=reverted.replace('\t"fmt"\n','')
    try:
        for stage,source in [("green",original),("revert",reverted),("restore",original)]:
            path.write_text(source)
            result=subprocess.run(["go","test",case["pkg"],"-run","^"+case["test"]+"$","-count=1"],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
            (root/(case["case"]+"-"+stage+".txt")).write_text(result.stdout+"\nEXIT="+str(result.returncode)+"\n")
            assert "[build failed]" not in result.stdout,result.stdout
            if stage=="revert":assert result.returncode!=0 and "--- FAIL: "+case["test"] in result.stdout,result.stdout
            else:assert result.returncode==0,result.stdout
        print(case["case"]+": 1 -> 0 -> 1",flush=True)
    finally:path.write_text(original)
# NPC classifier: retaining the producer but deleting only the C !IS_NPC gate
# must fail the NPC-silence assertion, with the positive PC proof still intact.
case=CASES[-1];path=pathlib.Path(case["file"]);original=path.read_text()
line=[l for l in case["new"].splitlines(True) if "MudLog(" in l][0]
reverted=original.replace(case["new"],line[1:])
try:
    for stage,source in [("green",original),("revert",reverted),("restore",original)]:
        path.write_text(source)
        result=subprocess.run(["go","test",case["pkg"],"-run","^"+case["test"]+"$","-count=1"],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
        (root/("loot-npc-"+stage+".txt")).write_text(result.stdout+"\nEXIT="+str(result.returncode)+"\n")
        assert "[build failed]" not in result.stdout,result.stdout
        if stage=="revert":assert result.returncode!=0 and "NPC-victim loot must not log" in result.stdout,result.stdout
        else:assert result.returncode==0,result.stdout
    print("loot-npc: 1 -> 0 -> 1",flush=True)
finally:path.write_text(original)
