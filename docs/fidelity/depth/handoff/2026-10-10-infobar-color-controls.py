#!/usr/bin/env python3
"""R5h: compile removed prompt fields and assert the corresponding failures."""
import json
from pathlib import Path
import subprocess
import tempfile

source=Path('pkg/session/session_send.go').resolve()
original=source.read_text()
def run(args, fail=False, name=''):
 r=subprocess.run(args,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 print(r.stdout,end='')
 if fail:
  assert r.returncode==1 and '[build failed]' not in r.stdout
  assert '--- FAIL: '+name in r.stdout
 else: assert r.returncode==0
command=['go','test','./pkg/session','-run','^TestInfobarColorLevels$','-count=1']
source=Path('pkg/session/display_cmds.go').resolve()
original=source.read_text()
run(command)
for replacement in ('if false {', 'if ch.colorNormal {'):
 assert 'if !ch.colorNormal {' in original
 with tempfile.TemporaryDirectory() as d:
  root=Path(d);mutation=root/source.name
  mutation.write_text(original.replace('if !ch.colorNormal {',replacement,1))
  overlay=root/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(source):str(mutation)}}))
  run(['go','test','-overlay='+str(overlay),'./pkg/session','-run','^TestInfobarColorLevels$','-count=1'],True,'TestInfobarColorLevels')
 run(command)
