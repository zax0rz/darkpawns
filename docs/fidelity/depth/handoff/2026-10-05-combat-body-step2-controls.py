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
wire='pkg/game/combat_wire.go'
for field,test in [('GetHP','TestCombatBodyCallbackIdentity'),('HasAffect','TestCombatBodyCallbackAffectsFlags'),('HasMobVNum','TestCombatBodyCallbackAffectsFlags')]:
 s=pathlib.Path(wire).read_text();a=s.index('cb.'+field+' =');b=s.index('{',a)+1;needle=s[a:b]
 add(field,wire,needle,needle+'\n if name.IsNPC() { name=w.GetMobByName(name.GetName()) }','./pkg/game',test)
s=pathlib.Path(wire).read_text();a=s.index('cb.GetKills =');b=s.index('{',a)+1;needle=s[a:b]
add('retired-player',wire,needle,needle+'\n if p,ok:=w.GetPlayer(name.GetName());ok {name=p}','./pkg/game','TestCombatBodyCallbackPlayerState')
s=pathlib.Path(wire).read_text();a=s.index('cb.GetWeaponInfo =');b=s.index('{',a)+1;needle=s[a:b]
add('weapon',wire,needle,needle+'\n if body.IsNPC(){body=w.GetMobByName(body.GetName())}','./pkg/game','TestCombatBodyWeaponCallback')
add('room-order',wire,'sort.SliceStable(chars, func(i, j int) bool { return combatRoomSequence(chars[i]) > combatRoomSequence(chars[j]) })','sort.SliceStable(chars, func(i, j int) bool { return combatRoomSequence(chars[i]) < combatRoomSequence(chars[j]) })','./pkg/game','TestCombatBodyFollowingAndRoomOrder')
add('player-default','pkg/game/target.go','body := ch.GetFightingBody()','body := ch.GetFightingBody();if body!=nil && body.IsNPC(){body=w.GetMobByName(body.GetName())}','./pkg/game','TestCombatBodyDefaultTargets/player-default')
add('mob-default','pkg/game/spec_procs.go','return me.GetFightingBody()','if me.GetTarget()!=nil{return me.GetTarget()};return me.GetFightingBody()','./pkg/game','TestCombatBodyDefaultTargets/mobile-default')
add('script','pkg/game/world_scriptable.go','mob := combatMob(body)','mob := w.GetMobByName(body.GetName())','./pkg/game','TestCombatBodyScriptDispatch')
add('self-protection','pkg/game/damage_gate.go','self := ch == victim','self := ch.GetName() == victim.GetName()','./pkg/game','TestCombatBodyProtectionIsNotNameSelf')
add('look','pkg/game/look.go','func fightingPresence(target combat.Combatant, viewer *Player) string {','func fightingPresence(target combat.Combatant, viewer *Player) string { if target!=nil && target.GetName()==viewer.GetName(){target=viewer}','./pkg/game','TestCombatBodyLookTarget')
add('lua-handle','pkg/scripting/engine.go','if ref, ok := charRefOf(tbl); ok {','if ref, ok := charRefOf(tbl); ok && !ref.NPC {','./pkg/game','TestCombatBodyLuaTarget')
add('npc-descriptor','pkg/session/combat_body_routing.go','s.switchedMob == mob','s.switchedMob.GetName() == mob.GetName()','./pkg/session','TestCombatBodyMessageRouting')
add('audience-name','pkg/session/combat_body_routing.go','if body == excluded {','if body.GetName() == excluded.GetName() {','./pkg/session','TestCombatBodyMessageRouting')
add('pc-descriptor','pkg/session/combat_body_routing.go','return m.attachedBodyLocked(p)','return m.sessions[p.GetName()]','./pkg/session','TestCombatBodyPCSwitchRouting')
add('target-notification','pkg/session/manager.go','fighting && target == victim','fighting && target.GetName() == victim.GetName()','./pkg/session','TestCombatBodyDamageNotification')
add('defense-name','pkg/combat/engine.go','observer == fighter || observer == opponent','observer.GetName() == fighter.GetName() || observer.GetName() == opponent.GetName()','./pkg/combat','TestCombatBodyDefenseAudience')
add('defense-sleep','pkg/combat/engine.go',' || observer.GetPosition() <= PosSleeping','','./pkg/combat','TestCombatBodyDefenseAudience')
add('group-recipient','pkg/combat/fight_core.go','PerformGroupGain(memberName, victim, base)','PerformGroupGain(leader, victim, base)','./pkg/game','TestCombatBodyGroupRecipients')
add('direct-text','pkg/combat/callbacks.go','callbacks != nil && callbacks.SendText != nil','false','./pkg/combat','TestCombatBodyDirectText')
add('follower-detach','pkg/game/follow.go','leader := asActor(w.combatFollowingBody(ch))','leader := w.followingActor(ch.GetFollowing())','./pkg/session','TestCombatBodyFollowerDetachMessages')
add('npc-follow','pkg/game/combat_wire.go','case *MobInstance:\n\t\tbody.mu.RLock()','case *MobInstance:\n return nil;body.mu.RLock()','./pkg/game','TestCombatBodyNPCFollowing')
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
