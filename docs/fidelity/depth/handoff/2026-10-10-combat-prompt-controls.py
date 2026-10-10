#!/usr/bin/env python3
"""R5h: compile removed prompt fields and assert the corresponding failures."""
import json
from pathlib import Path
import subprocess
import tempfile

source=Path('pkg/session/session_send.go').resolve()
original=source.read_text()
checks=(
 ('flags&(1<<uint(game.PrfDispTarget)) != 0 && target != nil','false && target != nil','TestCombatPromptTargetAndTank'),
 ('flags&(1<<uint(game.PrfDispTank)) != 0 && target != nil','false && target != nil','TestCombatPromptTargetAndTank'),
 ('return "\\x1b[31m" + text + "\\x1b[0m "','return text + " "','TestPromptStatusColorLevels'),
)
def run(args, fail=False, name=''):
 r=subprocess.run(args,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 print(r.stdout,end='')
 if fail:
  assert r.returncode==1 and '[build failed]' not in r.stdout
  assert '--- FAIL: '+name in r.stdout
 else: assert r.returncode==0
command=['go','test','./pkg/session','-run','^(TestCombatPrompt|TestPromptStatusColorLevels)','-count=1']
run(command)
for old,new,test in checks:
 assert old in original
 with tempfile.TemporaryDirectory() as d:
  root=Path(d);mutation=root/source.name;mutation.write_text(original.replace(old,new,1))
  overlay=root/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(source):str(mutation)}}))
  run(['go','test','-overlay='+str(overlay),'./pkg/session','-run','^'+test+'$','-count=1'],True,test)
 run(command)
