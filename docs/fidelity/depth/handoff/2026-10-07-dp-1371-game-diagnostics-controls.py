#!/usr/bin/env python3
"""R5h controls using Go overlays; leaves the checkout unchanged."""
import argparse, json, os, pathlib, subprocess, tempfile
p=argparse.ArgumentParser();p.add_argument('--output', required=True);p.add_argument('--case');a=p.parse_args()
root=pathlib.Path(a.output).expanduser();root.mkdir(parents=True,exist_ok=True)
(root/'HEAD.txt').write_bytes(subprocess.check_output(['git','rev-parse','HEAD']))
controls=[('milestone','pkg/game/death.go','MudLog(fmt.Sprintf("%s hit %d kills.", ch.GetName(), kills), MudlogNormal, LVL_IMMORT, false)','', 'TestGameDiagnosticKillMilestone')]
controls += [('cross-room','pkg/game/damage_gate.go','MudLog("Attempt to assign damage when ch and vict are in different rooms.", MudlogNormal, LVL_IMMORT, false)','','TestGameDiagnosticCrossRoomDamage')]
controls += [('protection-evil','pkg/spells/affect_spells.go','logProtectionKill(world, ch, "Evil")','','TestGameDiagnosticProtectionEvil')]
controls += [('protection-good','pkg/spells/affect_spells.go','logProtectionKill(world, ch, "Good")','','TestGameDiagnosticProtectionGood')]
controls += [('death-trap','pkg/game/death.go','LogDeathTrap(player.GetName(), room.VNum, room.Name)','','TestGameDiagnosticDeathTrapMovement')]
for name,file,before,after,test in controls:
 if a.case and a.case!=name:continue
 source=pathlib.Path(file).read_text();assert source.count(before)==1,(name,source.count(before))
 folder=root/name;folder.mkdir(exist_ok=True)
 with tempfile.TemporaryDirectory() as tmp:
  replacement=pathlib.Path(tmp)/pathlib.Path(file).name
  changed = source.replace(before,after)
  if name == 'zone-open-order':changed=changed.replace('tmp.Write(render())','tmp.Write(data)')
  replacement.write_text(changed)
  overlay=pathlib.Path(tmp)/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(pathlib.Path(file).resolve()):str(replacement)}}))
  for stage in ['green','revert','restore']:
   cmd=['go','test','-p','2']+(['-overlay='+str(overlay)] if stage=='revert' else [])+['./pkg/game','-run','^'+test+'$','-count=1']
   r=subprocess.run(cmd,env=dict(os.environ,GOMAXPROCS='2'),text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
   (folder/(stage+'.txt')).write_text(r.stdout+'\nEXIT='+str(r.returncode)+'\n')
   if stage=='revert':assert r.returncode!=0 and '[build failed]' not in r.stdout and '--- FAIL: '+test in r.stdout,r.stdout
   else:assert r.returncode==0,r.stdout
 print(name,'PASS 0/1/0',flush=True)
