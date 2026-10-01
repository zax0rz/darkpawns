#!/usr/bin/env python3
"""Assert-only 0/1/0 proofs in an isolated checkout; no census-source mutations."""
import subprocess
from pathlib import Path
work=Path('/home/zach/dp-p4-item-sleep-proof')
out=Path('/home/zach/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-item-sleep-proofs')
out.mkdir(parents=True,exist_ok=True)
mutations=[
 ('self-target','pkg/session/eat_cmds.go','spells.CallMagic(s.player, s.player, nil,','spells.CallMagic(s.player, nil, nil,'),
 ('potion-save-type','pkg/spells/call_magic.go','case CastWand, CastStaff, CastScroll, CastPotion:','case CastWand, CastStaff, CastScroll:'),
 ('outlaw-refusal','pkg/spells/affect_spells.go','if !hasOutlawFlag(ch) {','if false {'),
 ('outlaw-success','pkg/spells/affect_spells.go','if !hasOutlawFlag(ch) {','if true {'),
 ('reagent-vnum','pkg/spells/affect_spells.go','consumeSpellReagentVNum(ch, 1226)','consumeSpellReagentVNum(ch, 1227)'),
 ('reagent-before-outlaw','pkg/spells/affect_spells.go','if consumeSpellReagentVNum(ch, 1226) {','if hasOutlawFlag(ch) && consumeSpellReagentVNum(ch, 1226) {'),
 ('reagent-narration','pkg/spells/affect_spells.go','Pulling a bit of sand from a pocket, you cast it about the room...','Pulling a bit of sand from a pocket, you cast it about the room.'),
 ('self-refusal-message','pkg/spells/affect_spells.go','%s tried to cast a spell on you but failed because %s is not an Outlaw!','%s tried to cast a spell on you but failed because %s is not an Outlaw.'),
 ('save-draw','pkg/spells/affect_spells.go','saved = magAffectsSaveRoll(victim, savetype)','saved = false'),
 ('duration-caster','pkg/spells/affect_spells.go','4+getLevel(ch)/4+reag, 0, engine.AFFSleep','4+level/4+reag, 0, engine.AFFSleep'),
 ('duration-reagent','pkg/spells/affect_spells.go','4+getLevel(ch)/4+reag, 0, engine.AFFSleep','4+getLevel(ch)/4, 0, engine.AFFSleep'),
 ('sleep-flag','pkg/spells/affect_spells.go','4+getLevel(ch)/4+reag, 0, engine.AFFSleep','4+getLevel(ch)/4+reag, 0, engine.AFFCharm'),
 ('position','pkg/spells/affect_spells.go','p.SetPosition(int(PosSleeping))','p.SetPosition(int(PosStanding))'),
 ('sleep-message','pkg/spells/affect_spells.go','You feel very sleepy...  Zzzz......','You feel very sleepy...  Zzzz.....'),
 ('wait-pulses','pkg/session/eat_cmds.go','s.player.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)','s.player.SetWaitState(2) // mutation'),
 ('potion-consumption','pkg/session/eat_cmds.go','s.player.Inventory.RemoveItem(item)','// mutation: retain potion'),
]
rows=['mutation\tbefore\tmutant\tafter\tfailure']
def run(name,stage):
 r=subprocess.run(['go','test','./pkg/session','-run','^TestItemSleepEntryDepth$','-count=1'],cwd=work,text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
 (out/f'{name}-{stage}.log').write_text(r.stdout)
 return r
for name,file,old,new in mutations:
 p=work/file;original=p.read_text();assert old in original,name
 try:
  before=run(name,'before');p.write_text(original.replace(old,new,1));mutant=run(name,'mutant')
 finally:p.write_text(original)
 after=run(name,'after')
 assert before.returncode==0 and mutant.returncode==1 and after.returncode==0,(name,mutant.stdout)
 assert '--- FAIL: TestItemSleepEntryDepth' in mutant.stdout and 'build failed' not in mutant.stdout,name
 rows.append(f'{name}\t0\t1\t0\tassertion');print(name,'0 -> 1 -> 0',flush=True)
Path('docs/fidelity/depth/evidence/2026-10-01-item-sleep/revert-triples.tsv').write_text('\n'.join(rows)+'\n')
