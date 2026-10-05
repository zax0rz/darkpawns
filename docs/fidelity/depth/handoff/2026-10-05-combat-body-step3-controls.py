#!/usr/bin/env python3
"""Replay step-2 compiled assertion controls, restoring every file in finally.
Run in a disposable worktree; never concurrently with another editor/test process.
"""
import argparse
import pathlib
import subprocess

parser=argparse.ArgumentParser()
parser.add_argument('--output',type=pathlib.Path,required=True)
parser.add_argument('--only',default='')
args=parser.parse_args();args.output.mkdir(parents=True,exist_ok=True)
controls=[]
def add(label,file,before,after,package,test):
 controls.append((label,file,before,after,package,test))

add('one-way','pkg/combat/engine.go','body.StopFighting()\n\tbody.SetPosition','if target!=nil {target.StopFighting()}\n\tbody.StopFighting()\n\tbody.SetPosition','./pkg/combat','TestCombatBodyStopOneWay')
add('retarget','pkg/combat/engine.go','if !force && target != nil','if false && !force && target != nil','./pkg/combat','TestCombatBodyStopRetarget')
add('retire-name','pkg/combat/engine.go','if fighter.GetFightingBody() == body {','if fighter.GetFightingBody()!=nil && fighter.GetFightingBody().GetName()==body.GetName() {','./pkg/combat','TestCombatBodyRetirementIsolation')
add('extract-mob','pkg/game/world.go','w.retireCombatBody(mob)','_ = mob','./pkg/game','TestCombatBodyWorldCleanup/extract-mob')
add('pending-player','pkg/game/char_mgmt.go','retired = append(retired, p)','_ = p','./pkg/game','TestCombatBodyWorldCleanup/pending-player')
add('death','pkg/game/death.go','w.retireCombatBody(deadMob)','_ = deadMob','./pkg/game','TestCombatBodyWorldCleanup/death')
add('room-stop','pkg/game/combat_cleanup.go','if body.GetFightingBody() == nil {','if true {','./pkg/game','TestCombatBodyWorldCleanup/transfer-mob')
add('retreat-room-stop','pkg/game/combat_cleanup.go','if body.GetFightingBody() == nil {','if true {','./pkg/session','TestCmdRetreat_CharFromRoomStopsCombatAfterSuccess')
add('transfer-name','pkg/game/world_movement.go','func (w *World) transferBody(body combat.Combatant, toRoomVNum int, moveMount bool) error {','func (w *World) transferBody(body combat.Combatant, toRoomVNum int, moveMount bool) error { if body.IsNPC(){for _,m:=range w.GetMobsInRoom(body.GetRoom()){if m!=body && m.GetName()==body.GetName(){body=m;break}}}','./pkg/game','TestCombatBodyTeleportDuplicate')
add('stale-enrollment','pkg/combat/engine.go','func (ce *CombatEngine) startCombat(attacker, defender Combatant, deferDefenderEnrollment, afterDamage bool) error {','func (ce *CombatEngine) startCombat(attacker, defender Combatant, deferDefenderEnrollment, afterDamage bool) error { if m,ok:=attacker.(interface{SetCombatRetired(bool)});ok{m.SetCombatRetired(false)}','./pkg/game','TestCombatBodyRoundRetirement')
add('session-removal','pkg/session/manager.go','m.world.RemovePlayerBody(s.player)','// omitted selected body removal','./pkg/session','TestCombatBodySessionCleanup/remove')
add('retained-leader','pkg/game/combat_cleanup.go','w.combatFollowingBody(m) == body','m.GetFollowing() == body.GetName()','./pkg/game','TestCombatBodyRetiredLeader')
add('retire-round','pkg/game/combat_cleanup.go','ce.RetireCombatant(body)','_ = ce // omitted engine retirement','./pkg/game','TestCombatBodyConcurrentRetirement')
(args.output/'HEAD.txt').write_text(subprocess.check_output(['git','rev-parse','HEAD'],text=True))
for label,file,before,after,package,test in controls:
 if args.only and label not in args.only.split(','):continue
 p=pathlib.Path(file);original=p.read_text();assert before in original,(label,before)
 cmd=['go','test',package,'-run','^'+test+'$','-count=1']
 (args.output/(label+'-command.txt')).write_text(' '.join(cmd)+'\n')
 try:
  for phase in ['green','revert','restore']:
   p.write_text(original.replace(before,after,1) if phase=='revert' else original)
   r=subprocess.run(cmd,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
   (args.output/(label+'-'+phase+'.txt')).write_text(r.stdout)
   if phase=='revert':assert r.returncode and '--- FAIL:' in r.stdout and '[build failed]' not in r.stdout,r.stdout
   else:assert r.returncode==0,r.stdout
   print(label,phase,r.returncode,flush=True)
 finally:p.write_text(original)
