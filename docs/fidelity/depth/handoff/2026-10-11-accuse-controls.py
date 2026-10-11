#!/usr/bin/env python3
"""R5h compiling controls for all four accuse cases; checkout never mutated."""
import json
from pathlib import Path
import subprocess
import tempfile
checks = (
 ('socCharNoArg   = 0', 'socCharNoArg   = 1', 'TestAccuseNoArgBytesPinCSource'),
)
command = ['go', 'test', './pkg/session', '-run', '^TestAccuse', '-count=1']
def run(args):
 r=subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
 print(r.stdout, end='')
 return r
assert run(command).returncode == 0
source=Path('pkg/game/act_social.go').resolve()
original=source.read_text()
for old,new,test in checks:
 assert old in original
 with tempfile.TemporaryDirectory() as directory:
  root=Path(directory); mutation=root/'act_social.go'
  mutation.write_text(original.replace(old,new,1))
  overlay=root/'overlay.json'
  overlay.write_text(json.dumps({'Replace':{str(source):str(mutation)}}))
  result=run(['go','test','-overlay='+str(overlay),'./pkg/session','-run','^'+test+'$','-count=1'])
  assert result.returncode==1 and '[build failed]' not in result.stdout
  assert '--- FAIL: '+test in result.stdout
 assert run(command).returncode==0
