#!/usr/bin/env python3
"""Replay step-4 compiled assertion controls, restoring every file in finally.
Run in a disposable worktree; never concurrently with another editor/test process.
"""
import argparse
import pathlib
import subprocess

parser=argparse.ArgumentParser()
parser.add_argument('--output',type=pathlib.Path,required=True)
parser.add_argument('--only',default='')
parser.add_argument('--oracle',action='store_true')
args=parser.parse_args();args.output.mkdir(parents=True,exist_ok=True)
controls=[]
def add(label,file,before,after,package,test):
 controls.append((label,file,before,after,package,test))

s=pathlib.Path('pkg/game/world.go').read_text()
a=s.index('func (w *World) MovePlayer(');b=s.index('// sectorMoveCost',a)
old=subprocess.check_output(['git','show','0d7a0bad2:pkg/game/world.go'],text=True)
c=old.index('func (w *World) MovePlayer(');d=old.index('// sectorMoveCost',c)
add('movement-window','pkg/game/world.go',s[a:b],old[c:d],'./pkg/game','TestCombatBodyMovementWindow')
add('ordinal','pkg/game/target.go','sort.SliceStable(candidates, func(i, j int) bool { return combatRoomSequence(candidates[i]) > combatRoomSequence(candidates[j]) })','// restored prototype/ID ordinal order','./pkg/game','TestCombatBodyOrdinalSelection')
add('kill-type','pkg/session/combat_cmds.go','Instakill(tgt.Combatant, s.player, combat.TYPE_SLASH)','Instakill(tgt.Combatant, s.player, 0)','./pkg/session','TestCombatBodyKillCorpseSlash')
add('breath-identity','pkg/spells/damage_spells.go','chCombat != victCombat','chCombat.GetName() != victCombat.GetName()','./pkg/spells','TestCombatBodyBreathDuplicateEnrollment')
add('loot-identity','pkg/game/death.go','killer != victim &&','killerName != victim.GetName() &&','./pkg/game','TestCombatBodyLootNameCollision')
add('skill-audience','pkg/command/skill_commands.go','p == target','p.Name == target.GetName()','./pkg/command','TestCombatBodySkillAudienceCollision')
add('group-leaders','pkg/spells/affect_spells.go','if len(worlds) != 0 {','if false && len(worlds) != 0 {','./pkg/game','TestCombatBodyLiveGroupSpellLeaders')
add('death-credit','pkg/game/death.go','if kp, ok := killer.(*Player); ok && kp != nil {','if kp, ok := w.GetPlayer(killer.GetName()); ok {','./pkg/game','TestCombatBodyDeathCreditNameCollision')
add('pk-credit','pkg/game/death.go','if len(killerBodies) != 0 {','if false && len(killerBodies) != 0 {','./pkg/game','TestCombatBodyDeathCreditNameCollision')
(args.output/'HEAD.txt').write_text(subprocess.check_output(['git','rev-parse','HEAD'],text=True))
for label,file,before,after,package,test in controls:
 if args.only and label not in args.only.split(','):continue
 p=pathlib.Path(file);original=p.read_text();assert before in original,(label,before)
 cmd=['go','test',package,'-run','^'+test+'$','-count=1']
 (args.output/(label+'-command.txt')).write_text(' '.join(cmd)+'\n')
 try:
  for phase in ['green','revert','restore']:
   p.write_text(original.replace(before,after) if phase=='revert' else original)
   r=subprocess.run(cmd,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
   (args.output/(label+'-'+phase+'.txt')).write_text(r.stdout)
   if phase=='revert':assert r.returncode and '--- FAIL:' in r.stdout and '[build failed]' not in r.stdout,r.stdout
   else:assert r.returncode==0,r.stdout
   print(label,phase,r.returncode,flush=True)
 finally:p.write_text(original)

if args.oracle:
 p=pathlib.Path('pkg/game/target.go');original=p.read_text()
 before='sort.SliceStable(candidates, func(i, j int) bool { return combatRoomSequence(candidates[i]) > combatRoomSequence(candidates[j]) })'
 assert before in original
 cmd=['go','run','./cmd/dp-oracle-diff','--scenario','combat-duplicate-bodies-depth','--seed','1','--show-oracle']
 (args.output/'oracle-command.txt').write_text(' '.join(cmd)+'\n')
 try:
  for phase in ['green','revert','restore']:
   p.write_text(original.replace(before,'// restored prototype/ID ordinal order',1) if phase=='revert' else original)
   r=subprocess.run(cmd,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
   (args.output/('oracle-'+phase+'.txt')).write_text(r.stdout)
   if phase=='revert':assert r.returncode and 'normalized divergence detected' in r.stdout and 'build failed' not in r.stdout,r.stdout
   else:assert r.returncode==0,r.stdout
   print('oracle',phase,r.returncode,flush=True)
 finally:p.write_text(original)
