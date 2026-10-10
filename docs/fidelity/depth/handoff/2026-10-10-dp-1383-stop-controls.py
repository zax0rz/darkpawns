#!/usr/bin/env python3
"""R5h compiling overlays; never modify the checkout."""
import json
from pathlib import Path
import subprocess
import tempfile

source=Path('pkg/game/follow.go').resolve()
original=source.read_text()
tests='TestStopFollowerMobClearsCharmAndGroup|TestDamageBreaksMountCharmAndGroup'
def run(args, failure=False):
 r=subprocess.run(args,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 print(r.stdout,end='')
 if failure:
  assert r.returncode==1 and '[build failed]' not in r.stdout
  assert '--- FAIL: TestStopFollowerMobClearsCharmAndGroup' in r.stdout
 else: assert r.returncode==0
command=['go','test','./pkg/game','-run','^('+tests+')$','-count=1']
run(command)
for old in ('mob.RemoveAffected(affCharm)','mob.RemoveAffected(affGroup)','mob.RemoveAffectBySpell(7) // src/utils.c:414-415, SPELL_CHARM'):
 assert old in original
 with tempfile.TemporaryDirectory() as d:
  root=Path(d);mutation=root/source.name
  mutation.write_text(original.replace(old,'// removed by R5h control',1))
  overlay=root/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(source):str(mutation)}}))
  run(['go','test','-overlay='+str(overlay),'./pkg/game','-run','^('+tests+')$','-count=1'],True)
 run(command)
