#!/usr/bin/env python3
"""R5h: compiling named assertion failures, never build failures."""
import argparse,pathlib,subprocess
CASES = [{'case': 'inventory', 'test': 'TestStealZoneMudlogInventory', 'wrapper': 'pkg/session/steal_zone_mudlog_inventory_test.go', 'patches': [{'path': 'pkg/game/skill_stealth.go', 'old': '\titem.Location = LocInventoryPlayer(ch.Name)\n\tapplyRobbedAffect(target)\n\tmessage := appendImprovementMessage("Got it!"', 'new': '\titem.Location = LocInventoryPlayer(ch.Name)\n\tif !target.IsNPC() {\n\t\tMudLog(fmt.Sprintf("(PS) %s stole %s from %s.", ch.GetName(), item.GetShortDesc(), target.GetName()), MudlogComplete, LVL_IMMORT, true)\n\t}\n\tapplyRobbedAffect(target)\n\tmessage := appendImprovementMessage("Got it!"'}]}, {'case': 'equipment', 'test': 'TestStealZoneMudlogEquipment', 'wrapper': 'pkg/session/steal_zone_mudlog_equipment_test.go', 'patches': [{'path': 'pkg/game/skill_stealth.go', 'old': '\titem.Location = LocInventoryPlayer(ch.Name)\n\tapplyRobbedAffect(target)\n\tmessage := appendImprovementMessage(\n', 'new': '\titem.Location = LocInventoryPlayer(ch.Name)\n\tif !target.IsNPC() {\n\t\tMudLog(fmt.Sprintf("(PS) %s stole %s from %s.", ch.GetName(), item.GetShortDesc(), target.GetName()), MudlogComplete, LVL_IMMORT, true)\n\t}\n\tapplyRobbedAffect(target)\n\tmessage := appendImprovementMessage(\n'}]}, {'case': 'failure', 'test': 'TestStealZoneMudlogFailure', 'wrapper': 'pkg/session/steal_zone_mudlog_failure_test.go', 'patches': [{'path': 'pkg/game/skill_stealth.go', 'old': '\tch.SetPlrFlag(PlrOutlaw, true)\n}\n\nfunc applyRobbedAffect', 'new': '\tch.SetPlrFlag(PlrOutlaw, true)\n\tresult.StealCaughtPlayer = true\n}\n\nfunc applyRobbedAffect'}, {'path': 'pkg/game/skills.go', 'old': '\tStartCombat bool\n', 'new': "\tStartCombat bool\n\t// StealCaughtPlayer identifies only ordinary steal's caught PC arm. Its\n\t// CMP producer follows all caught messages and precedes WAIT_STATE\n\t// (src/act.other.c:531-539); it is not a generic deferred log callback.\n\tStealCaughtPlayer bool\n"}, {'path': 'pkg/command/skill_commands.go', 'old': '\t// Apply WAIT_STATE (C-10: cooldown in PULSE_VIOLENCE ticks)\n', 'new': '\tif result.StealCaughtPlayer && target != nil && !target.IsNPC() {\n\t\tgame.MudLog(fmt.Sprintf("(PS) %s unsuccessfuly tried to steal from %s.", ch.GetName(), target.GetName()), game.MudlogComplete, game.LVL_IMMORT, true)\n\t}\n\n\t// Apply WAIT_STATE (C-10: cooldown in PULSE_VIOLENCE ticks)\n'}]}, {'case': 'zonearg3', 'test': 'TestStealZoneMudlogZoneArg3', 'wrapper': 'pkg/session/steal_zone_mudlog_zonearg3_test.go', 'patches': [{'path': 'pkg/session/zedit.go', 'old': '\t\ts.zeditShowMenuLocked()\n\t}\n}\n\nfunc zeditRemoveSaveZone', 'new': '\t\ts.zeditShowMenuLocked()\n\tdefault:\n\t\ts.finishZeditLocked(false)\n\t\tgame.MudLog("SYSERR: OLC: zedit_parse(): case ARG3: Ack!", game.MudlogBrief, game.LVL_IMMORT, true)\n\t}\n}\n\nfunc zeditRemoveSaveZone'}]}, {'case': 'zonedefault', 'test': 'TestStealZoneMudlogZoneDefault', 'wrapper': 'pkg/session/steal_zone_mudlog_zonedefault_test.go', 'patches': [{'path': 'pkg/session/zedit.go', 'old': '\t\ts.zeditShowMenuLocked()\n\t}\n}\n\nfunc (s *Session) parseZeditMainLocked', 'new': '\t\ts.zeditShowMenuLocked()\n\tdefault:\n\t\ts.finishZeditLocked(false)\n\t\tgame.MudLog("SYSERR: OLC: zedit_parse(): Reached default case!", game.MudlogBrief, game.LVL_IMMORT, true)\n\t}\n}\n\nfunc (s *Session) parseZeditMainLocked'}]}]
a=argparse.ArgumentParser();a.add_argument('--output',required=True);root=pathlib.Path(a.parse_args().output);root.mkdir(parents=True,exist_ok=True)
for c in CASES:
 originals={p['path']:pathlib.Path(p['path']).read_text() for p in c['patches']}
 reverted=dict(originals)
 for p in c['patches']:
  assert reverted[p['path']].count(p['new'])==1,(c['case'],p['path'])
  reverted[p['path']]=reverted[p['path']].replace(p['new'],p['old'])
 try:
  for stage,sources in [('green',originals),('revert',reverted),('restore',originals)]:
   for path,source in sources.items():pathlib.Path(path).write_text(source)
   result=subprocess.run(['go','test','./pkg/session','-run','^'+c['test']+'$','-count=1'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
   (root/(c['case']+'-'+stage+'.txt')).write_text(result.stdout+'\nEXIT='+str(result.returncode)+'\n')
   assert '[build failed]' not in result.stdout,result.stdout
   if stage=='revert':assert result.returncode!=0 and '--- FAIL: '+c['test'] in result.stdout,result.stdout
   else:assert result.returncode==0,result.stdout
  print(c['case']+': 1 -> 0 -> 1',flush=True)
 finally:
  for path,source in originals.items():pathlib.Path(path).write_text(source)

# Orthogonal classifier/order controls retain the positive producer.
import re
extras=[]
for c in CASES[:2]:
 p=c['patches'][0];original=pathlib.Path(p['path']).read_text()
 changed=re.sub(r'\tif !target.IsNPC\(\) {\n(\t\tMudLog[^\n]+\n)\t}\n',lambda m:m.group(1)[1:],p['new'])
 assert changed!=p['new']
 extras.append((c['case']+'-npc',c['test'],p['path'],original.replace(p['new'],changed),'NPC steal path logged'))
c=CASES[2];p=c['patches'][-1];original=pathlib.Path(p['path']).read_text()
block=p['new'].replace(p['old'],'')
changed=original.replace(block,'')
anchor='\t// Send to victim\n'
assert changed.count(anchor)==1
extras.append(('failure-before-audiences',c['test'],p['path'],changed.replace(anchor,block+anchor),'message must precede log'))
for c in CASES[3:]:
 p=c['patches'][0];original=pathlib.Path(p['path']).read_text()
 changed=p['new'].replace('\t\ts.finishZeditLocked(false)\n','')
 extras.append((c['case']+'-without-cleanup',c['test'],p['path'],original.replace(p['new'],changed),'zone log must follow full cleanup'))
for name,test,path,reverted,assertion in extras:
 original=pathlib.Path(path).read_text()
 try:
  for stage,source in [('green',original),('revert',reverted),('restore',original)]:
   pathlib.Path(path).write_text(source)
   result=subprocess.run(['go','test','./pkg/session','-run','^'+test+'$','-count=1'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
   (root/(name+'-'+stage+'.txt')).write_text(result.stdout+'\nEXIT='+str(result.returncode)+'\n')
   assert '[build failed]' not in result.stdout,result.stdout
   if stage=='revert':assert result.returncode!=0 and assertion in result.stdout,result.stdout
   else:assert result.returncode==0,result.stdout
  print(name+': 1 -> 0 -> 1',flush=True)
 finally:pathlib.Path(path).write_text(original)
