#!/usr/bin/env python3
"""R5h: remove the player equip-stat boundary; require assertion failures."""
import json
from pathlib import Path
import subprocess
import tempfile
source=Path('pkg/game/bad_stats.go').resolve()
original=source.read_text()
a=original.index('func (p *Player) checkEquipmentStats() {')
b=original.index('\nfunc (w *World) applyBadStat',a)
removed=original[:a]+'func (p *Player) checkEquipmentStats() {}\n'+original[b:]
command=['go','test','./pkg/game','-run','^TestPlayerEquipment(BadStats|BadStatPriority|DexUsesDamageTail|CharismaHunting|BadStatEntryPoints)$','-count=1']
def run(args,fail=False):
 result=subprocess.run(args,text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
 print(result.stdout,end='')
 if fail:
  assert result.returncode==1 and '[build failed]' not in result.stdout
  for name in ('BadStats','BadStatPriority','DexUsesDamageTail','CharismaHunting','BadStatEntryPoints'):
   assert '--- FAIL: TestPlayerEquipment'+name in result.stdout,name
 else:assert result.returncode==0
run(command)
with tempfile.TemporaryDirectory() as directory:
 root=Path(directory)
 mutated=root/source.name;mutated.write_text(removed)
 overlay=root/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(source):str(mutated)}}))
 run(command[:2]+['-overlay='+str(overlay)]+command[2:],True)
run(command)
