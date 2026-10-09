#!/usr/bin/env python3
"""R5h: all four level confirmations fail on compiling original plain output."""
import json
from pathlib import Path
import subprocess
import tempfile

source=Path('pkg/session/act_informative.go').resolve()
text=source.read_text()
a=text.index('\t// src/act.informative.c:2494-2495:')
b=text.index('\treturn nil\n}',a)
removed=text[:a]+'\ts.Send(fmt.Sprintf("Your color is now %s.", levels[tp]))\n'+text[b:]
command=['go','test','./pkg/session','-run','^TestColorConfirmationNewLevelANSI$','-count=1']
def run(cmd, red=False):
 result=subprocess.run(cmd,text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
 print(result.stdout,end='')
 if red:
  assert result.returncode==1 and '[build failed]' not in result.stdout
  for level in ['off','sparse','normal','complete']:
   for old in ['off','complete']:
    assert '--- FAIL: TestColorConfirmationNewLevelANSI/'+level+'-from-'+old in result.stdout
 else: assert result.returncode==0
run(command)
with tempfile.TemporaryDirectory() as directory:
 root=Path(directory)
 file=root/source.name
 file.write_text(removed)
 overlay=root/'overlay.json'
 overlay.write_text(json.dumps({'Replace':{str(source):str(file)}}))
 run(command[:2]+['-overlay='+str(overlay)]+command[2:],True)
run(command)
