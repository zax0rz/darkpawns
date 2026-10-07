#!/usr/bin/env python3
"""Compiling R5h report controls; use Go overlays to keep checkout unchanged."""
import json, os, pathlib, subprocess, sys, tempfile
root = pathlib.Path(sys.argv[1]).expanduser()
root.mkdir(parents=True, exist_ok=True)
files = ['pkg/game/other_settings.go', 'pkg/session/commands.go', 'pkg/session/cmd_misc.go', 'pkg/session/redit.go', 'pkg/game/world_redit.go', 'pkg/session/olc_control_reachability_test.go']
original = {name:pathlib.Path(name).read_text() for name in files}
producer = 'MudLog(fmt.Sprintf("%s %s: %s", ch.GetName(), cmd, arg), MudlogComplete, LVL_IMMORT, false)'
raw = '''\tif (cmd == "bug" || cmd == "typo" || cmd == "idea" || cmd == "todo") && rawArgs != "" {
\t\ts.manager.world.ExecGenWrite(s.player, cmd, strings.ReplaceAll(rawArgs, "$", "$$"))
\t\treturn nil
\t}
'''
controls = [
 ('producer','pkg/game/other_settings.go',producer,'', 'TestReportMudlogRawBytesAndOrder'),
 ('type','pkg/game/other_settings.go',producer,producer.replace('MudlogComplete','MudlogNormal'), 'TestReportMudlogRawBytesAndOrder'),
 ('level','pkg/game/other_settings.go',producer,producer.replace('LVL_IMMORT','LVL_IMMORT + 1'), 'TestReportMudlogRawBytesAndOrder'),
 ('file','pkg/game/other_settings.go',producer,producer.replace('false','true'), 'TestReportMudlogRawBytesAndOrder'),
 ('raw-spacing','pkg/session/commands.go',raw,'', 'TestReportMudlogRawBytesAndOrder'),
 ('raw-dollar','pkg/session/commands.go','strings.ReplaceAll(rawArgs, "$", "$$")','rawArgs', 'TestReportMudlogRawBytesAndOrder'),
 ('tokenized-dollar','pkg/session/cmd_misc.go','strings.ReplaceAll(strings.Join(args, " "), "$", "$$")','strings.Join(args, " ")','TestReportMudlogTokenizedDollars'),
]
controls += [
 ('room-producer','pkg/session/redit.go','game.MudLog("SYSERR: OLC: redit_save_internally: Unknown comand", game.MudlogBrief, LVL_IMMORT, true)','', 'TestReditInsertionMudlogBoundary'),
 ('room-loop-omission','pkg/game/world_redit.go','case "M", "O", "D", "R", "G", "P", "E", "*":','case "M", "O", "D", "R", "G", "P", "E", "*", "L":','TestReditInsertionMudlogBoundary'),
 ('room-replacement-gate','pkg/game/world_redit.go','if existed {','if false && existed {','TestReditInsertionMudlogBoundary'),
]
controls += [('parse-action-audit','pkg/session/olc_control_reachability_test.go','if !covered[argument] {','if false && !covered[argument] {','TestCImprovedEditorDefaultUnreachable')]
cmd=['go','test','-p','2','./pkg/session','-run','^Test(ReportMudlog.*|ReditInsertionMudlogBoundary|CImprovedEditorDefaultUnreachable)$','-count=1']
def run(path, overlay=None):
 args=cmd[:2]+(['-overlay='+str(overlay)] if overlay else [])+cmd[2:]
 result=subprocess.run(args, env=dict(os.environ,GOMAXPROCS='2'),text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
 path.write_text(result.stdout)
 return result
for name, file, before, after, test in controls:
 if len(sys.argv) > 2 and not name.startswith(sys.argv[2]): continue
 folder=root/name;folder.mkdir(exist_ok=True)
 assert run(folder/'green.txt').returncode == 0
 assert before in original[file], name
 with tempfile.TemporaryDirectory() as tmp:
  replacement=pathlib.Path(tmp)/pathlib.Path(file).name
  replacement.write_text(original[file].replace(before,after))
  overlay=pathlib.Path(tmp)/'overlay.json'
  overlay.write_text(json.dumps({'Replace':{str(pathlib.Path(file).resolve()):str(replacement)}}))
  result=run(folder/'revert.txt',overlay)
  assert result.returncode != 0 and '[build failed]' not in result.stdout and '--- FAIL: '+test in result.stdout, result.stdout
 assert run(folder/'restore.txt').returncode == 0
 print(name,'PASS 0/1/0',flush=True)
