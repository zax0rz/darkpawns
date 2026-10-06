import pathlib, subprocess, sys
root=pathlib.Path(sys.argv[1]); root.mkdir(parents=True, exist_ok=True)
controls=[
 ('dex-missile','pkg/command/shoot.go','missile*10 - reaction*10','0*missile - reaction*10','./pkg/command','TestShootDexProbabilityAndStrictComparison'),
 ('dex-reaction','pkg/command/shoot.go','missile*10 - reaction*10','missile*10 - 0*reaction','./pkg/command','TestShootDexProbabilityAndStrictComparison'),
 ('strict-comparison','pkg/command/shoot.go','percent >= probability','percent > probability','./pkg/command','TestShootDexProbabilityAndStrictComparison'),
 ('projectile-dice','pkg/command/shoot.go','dprng.Dice(projectile.GetValue(1), projectile.GetValue(2))','dprng.Dice(1, 6)','./pkg/command','TestShootMobOutcomes'),
 ('bow-dice','pkg/command/shoot.go','dprng.Dice(bow.GetValue(1), bow.GetValue(2))','dprng.Dice(1, 4)','./pkg/command','TestShootPlayerOutcomes'),
 ('improve','pkg/command/shoot.go','game.ImproveSkill(ch, game.SkillShoot)','// omitted improvement','./pkg/command','TestShootPlayerOutcomes'),
 ('player-relocation','pkg/command/shoot.go','if !target.IsNPC() {','if !target.IsNPC() {\n world.MovePlayerToRoom(target.(*game.Player),ch.GetRoom())','./pkg/command','TestShootPlayerOutcomes'),
 ('retaliation','pkg/command/shoot.go','return engine.PerformRangedRetaliation(target, ch)','_ = engine\n return nil','./pkg/command','TestShootMobOutcomes'),
 ('actor-preamble','pkg/command/shoot.go','Twang... your projectile flies into the distance.','Twang... you miss!','./pkg/command','TestShootActorOutput'),
 ('audience','pkg/command/shoot.go','and narrowly misses $n!','and misses $n!','./pkg/command','TestShootAudienceOutput'),
 ('extract','pkg/command/shoot.go','world.ExtractObject(projectile, ch.GetRoom())','// omitted extraction','./pkg/command','TestShootProjectileAndZeroWait'),
 ('drop','pkg/command/shoot.go','world.MoveObjectToRoomFront(projectile, target.GetRoom())','world.MoveObjectToRoomFront(projectile, ch.GetRoom())','./pkg/command','TestShootProjectileAndZeroWait'),
 ('wait','pkg/command/shoot.go','percent := dprng.Number(1, 101)','ch.SetWaitStatePulses(20)\n percent := dprng.Number(1, 101)','./pkg/command','TestShootProjectileAndZeroWait'),
 ('death-counter','pkg/game/shoot.go','v.mu.Lock()\n\t\tif v.Level','v.mu.Lock()\n v.Deaths++\n\t\tif v.Level','./pkg/game','TestRangedDeathHasNoKillerBookkeeping'),
 ('death-exp','pkg/game/shoot.go','v.Exp/3','v.Exp/37','./pkg/game','TestRangedDeathHasNoKillerBookkeeping'),
 ('rawkill','pkg/game/shoot.go','w.RawKillCombatant(victim, combat.TYPE_UNDEFINED)','// omitted raw kill','./pkg/game','TestRangedDeathHasNoKillerBookkeeping'),
 ('eager-retaliation','pkg/combat/engine.go','ce.performOneHit(&CombatPair{Attacker: attacker, Defender: defender, RangedRetaliation: true, DeferDefenderEnrollment: true})','if err:=ce.StartCombatFromMob(attacker,defender);err!=nil{return err}\n ce.PerformInitialAttack(attacker,defender)','./pkg/combat','TestRangedRetaliationEnrollmentAndProtections'),
]
controls += [
 ('kill-credit','pkg/command/shoot.go','world.ApplyRangedProjectileDamage(target, damage)','ch.Kills++\n ch.SetExp(ch.GetExp()+100)\n world.ApplyRangedProjectileDamage(target, damage)','./pkg/command','TestShootLethalCommandBoundary'),
 ('pk-flags','pkg/command/shoot.go','world.ApplyRangedProjectileDamage(target, damage)','ch.PKs++\n ch.SetPLRFlag(game.PlrOutlaw)\n world.ApplyRangedProjectileDamage(target, damage)','./pkg/command','TestShootLethalCommandBoundary'),
 ('con-loss','pkg/game/shoot.go','if v.Level >= 1','v.effectiveAttributes.Con--\n if v.Level >= 1','./pkg/game','TestRangedDeathHasNoKillerBookkeeping'),
 ('retaliation-posture','pkg/combat/engine.go','ce.performOneHit(&CombatPair{Attacker: attacker, Defender: defender, RangedRetaliation: true, DeferDefenderEnrollment: true})','defender.SetPosition(PosFighting)\n ce.performOneHit(&CombatPair{Attacker: attacker, Defender: defender, RangedRetaliation: true, DeferDefenderEnrollment: true})','./pkg/combat','TestRangedRetaliationReadsSleepingPostureBeforeDamage'),
 ('hunter','pkg/combat/engine.go','cb.RangedHunt(attacker, defender)','// omitted hunter assignment','./pkg/game','TestRangedRetaliationHunterCallback'),
 ('bare-transfer','pkg/game/shoot.go','func (w *World) TransferRangedVictim(mob *MobInstance, to int) error {','func (w *World) TransferRangedVictim(mob *MobInstance, to int) error {\n return w.MobTransfer(mob,to)\n','./pkg/game','TestRangedTransferPreservesPostureAndCircle'),
]
controls += [
 ('improve-before-extract','pkg/command/shoot.go','world.ExtractObject(projectile, ch.GetRoom())\n\tgame.ImproveSkill(ch, game.SkillShoot)','game.ImproveSkill(ch, game.SkillShoot)\n world.ExtractObject(projectile,ch.GetRoom())','./pkg/command','TestShootImprovementFollowsExtractionBeforeHP'),
 ('circle','pkg/game/shoot.go','obj.SetTimer(obj.GetTimer() - 1)','// omitted circle check','./pkg/game','TestRangedTransferPreservesPostureAndCircle'),
 ('atomic-hp','pkg/game/shoot.go','v.mu.Lock()\n\t\tv.Health -= damage\n\t\tv.mu.Unlock()','v.SetHP(v.GetHP()-damage)','./pkg/game','TestRangedDamageConcurrentHPUpdates'),
]
controls = [c for c in controls if c[0] != 'atomic-hp']
controls += [
 ('no-skill-message','pkg/command/shoot.go','world.ApplyRangedProjectileDamage(target, damage)','combat.GetCallbacks().SkillMessage(damage,ch,target,148,ch.GetRoom())\n world.ApplyRangedProjectileDamage(target, damage)','./pkg/command','TestShootNoSkillMessagePath'),
 ('duplicate-retaliation','pkg/combat/engine.go','ce.performOneHit(&CombatPair{Attacker: attacker, Defender: defender, RangedRetaliation: true, DeferDefenderEnrollment: true})','return nil','./pkg/game','TestShootRealDuplicateMobRetaliation'),
]
controls.append(('projectile-before-direction', 'pkg/command/skill_commands.go', '\tprojectile, found := world.ResolveObjectInInventory(ch, projectileName)\n\tif !found {\n\t\treturn s.SendMessage(fmt.Sprintf("You don\'t seem to have any %ss.\\r\\n", projectileName))\n\t}\n\tif projectile.GetTypeFlag() != int(game.ItemMissile) {\n\t\treturn s.SendMessage(game.CapitalizeSentence(projectile.GetShortDesc()+" is not a projectile!") + "\\r\\n")\n\t}\n\n\tdirections := map[string]string{\n\t\t"north": "north", "n": "north",\n\t\t"east": "east", "e": "east",\n\t\t"south": "south", "s": "south",\n\t\t"west": "west", "w": "west",\n\t\t"up": "up", "u": "up",\n\t\t"down": "down", "d": "down",\n\t}\n\tdirection, validDirection := directions[directionName]\n\tif !validDirection {\n\t\treturn s.SendMessage("Interesting direction.\\r\\n")\n\t}\n\n', '\tdirections := map[string]string{\n\t\t"north": "north", "n": "north",\n\t\t"east": "east", "e": "east",\n\t\t"south": "south", "s": "south",\n\t\t"west": "west", "w": "west",\n\t\t"up": "up", "u": "up",\n\t\t"down": "down", "d": "down",\n\t}\n\tdirection, validDirection := directions[directionName]\n\tif !validDirection {\n\t\treturn s.SendMessage("Interesting direction.\\r\\n")\n\t}\n\n\tprojectile, found := world.ResolveObjectInInventory(ch, projectileName)\n\tif !found {\n\t\treturn s.SendMessage(fmt.Sprintf("You don\'t seem to have any %ss.\\r\\n", projectileName))\n\t}\n\tif projectile.GetTypeFlag() != int(game.ItemMissile) {\n\t\treturn s.SendMessage(game.CapitalizeSentence(projectile.GetShortDesc()+" is not a projectile!") + "\\r\\n")\n\t}\n\n', './pkg/command', 'TestShootProjectileBeforeDirection'))
for name,path,before,after,pkg,test in controls:
 p=pathlib.Path(path);original=p.read_text()
 if before not in original:sys.exit('missing mutation '+name)
 r=subprocess.run(['go','test',pkg,'-run','^'+test+'$','-count=1'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 (root/(name+'-green.txt')).write_text(r.stdout)
 if r.returncode:sys.exit('initial green failed '+name+'\n'+r.stdout)
 try:
  p.write_text(original.replace(before,after,1))
  r=subprocess.run(['go','test',pkg,'-run','^'+test+'$','-count=1'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
  (root/(name+'-revert.txt')).write_text(r.stdout)
  if r.returncode==0 or '--- FAIL:' not in r.stdout or '[build failed]' in r.stdout:sys.exit('invalid red '+name+'\n'+r.stdout)
 finally:p.write_text(original)
 r=subprocess.run(['go','test',pkg,'-run','^'+test+'$','-count=1'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 (root/(name+'-restore.txt')).write_text(r.stdout)
 if r.returncode:sys.exit('restore failed '+name+'\n'+r.stdout)
 print(name,'1 -> 0',flush=True)

# Restore the actual main shoot route (and its removed DoShoot callee), rather
# than a guessed surrogate. Shared body-identity prerequisites remain in place.
paths = ['pkg/command/skill_commands.go', 'pkg/game/skill_c10_combat.go']
originals = {path: pathlib.Path(path).read_bytes() for path in paths}
try:
 for path in paths:
  pathlib.Path(path).write_bytes(subprocess.check_output(['git', 'show', 'dcea4e13b:'+path]))
 r = subprocess.run(['go', 'test', './pkg/command', '-run', '^TestShoot(MobOutcomes|PlayerOutcomes|ActorOutput|AudienceOutput|ProjectileAndZeroWait|NoSkillMessagePath|LethalCommandBoundary)$', '-count=1'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
 (root/'main-route-revert.txt').write_text(r.stdout)
 if r.returncode == 0 or '--- FAIL:' not in r.stdout or '[build failed]' in r.stdout:
  sys.exit('invalid main route red\n'+r.stdout)
finally:
 for path, content in originals.items():
  pathlib.Path(path).write_bytes(content)
r = subprocess.run(['go', 'test', './pkg/command', '-run', '^TestShoot', '-count=1'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
(root/'main-route-restore.txt').write_text(r.stdout)
if r.returncode:
 sys.exit('main route restore failed\n'+r.stdout)
print('main-route 1 -> 0', flush=True)
