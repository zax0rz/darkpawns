package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// broadcastCombatMsg encodes and broadcasts a combat event message to a room.
func broadcastCombatMsg(s *Session, roomVNum int, eventType, text string) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: eventType,
			From: s.player.Name,
			Text: text,
		},
	})
	if err != nil {
		slog.Error("broadcastCombatMsg marshal", "error", err)
		return
	}
	s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)
}

// findMobInRoom finds a mob in the player's current room by partial name match.
// Returns nil if not found.
func findMobInRoom(s *Session) func(name string) interface{ GetShortDesc() string; GetName() string } {
	return nil // see inline usage below
}

// cmdAssist — assist a target in their combat.
// Ported from do_assist() in src/act.offensive.c lines 54-96.
func cmdAssist(s *Session, args []string) error {
	// 1. Player must not already be fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're already fighting! How can you assist someone else?\r\n")
		return nil
	}

	// 2. If mounted, must dismount first
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	if len(args) == 0 {
		s.Send("Whom do you wish to assist?\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 3. Find the target character in the room (players and mobs)
	var helpee combat.Combatant
	helpeeName := ""

	for _, p := range s.manager.world.GetPlayersInRoom(room.VNum) {
		if p.Name == s.player.Name {
			continue
		}
		if !strings.Contains(strings.ToLower(p.Name), targetName) {
			continue
		}
		helpee = p
		helpeeName = p.Name
		break
	}
	if helpee == nil {
		for _, m := range s.manager.world.GetMobsInRoom(room.VNum) {
			if !strings.Contains(strings.ToLower(m.GetShortDesc()), targetName) &&
				!strings.Contains(strings.ToLower(m.GetName()), targetName) {
				continue
			}
			helpee = m
			helpeeName = m.GetShortDesc()
			break
		}
	}
	if helpee == nil {
		s.Send("They don't seem to be here.\r\n")
		return nil
	}

	// Find who is fighting the helpee
	opponent, fighting := s.manager.combatEngine.GetCombatTarget(helpeeName)
	if !fighting {
		s.Send(fmt.Sprintf("But nobody is fighting %s!\r\n", helpeeName))
		return nil
	}

	// 4. Player joins the fight
	if err := s.manager.combatEngine.StartCombat(s.player, opponent); err != nil {
		s.Send(err.Error())
		return nil
	}
	s.Send("You join the fight!\r\n")
	// Notify the helpee
	if !helpee.IsNPC() {
		if helpeeSess, ok := s.manager.GetSession(helpeeName); ok {
			helpeeSess.Send(fmt.Sprintf("%s assists you!\r\n", s.player.Name))
		}
	}
	broadcastCombatMsg(s, room.VNum, "assist",
		fmt.Sprintf("%s assists %s.", s.player.Name, helpeeName))
	s.markDirty(VarFighting)
	return nil
}

// cmdKill — attack a non-player (mob) target.
func cmdKill(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Kill who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're already fighting!")
		return nil
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if err := s.manager.combatEngine.StartCombat(s.player, mob); err != nil {
				s.Send(err.Error())
				return nil
			}
			s.Send(fmt.Sprintf("You lunge at %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "combat",
				fmt.Sprintf("%s attacks %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdBackstab — backstab a target.
// Ported from do_backstab() in src/act.offensive.c lines 165-220.
// Requires: piercing weapon, target not fighting, not mounted.
// MOB_AWARE mobs that are awake will strike back.
// Skill check: percent=rand(1,101), prob=skill level.
// On hit: damage = weapon_dice * backstab_mult(level).
func cmdBackstab(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Backstab who?\r\n")
		return nil
	}

	// Skill check
	if s.player.GetSkill(game.SkillBackstab) == 0 {
		s.Send("You have no idea how.\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 1. Player must NOT be fighting already
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You can't backstab while you're fighting!\r\n")
		return nil
	}

	// Find target mob in room
	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	var target combat.Combatant
	var targetMob *game.MobInstance
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			target = mob
			targetMob = mob
			break
		}
	}
	if target == nil {
		s.Send("Backstab who?\r\n")
		return nil
	}
	targetDesc := target.GetName()

	// Target must not be self
	if target.GetName() == s.player.Name || targetDesc == s.player.Name {
		s.Send("How can you sneak up on yourself?\r\n")
		return nil
	}

	// 2. Target must NOT be fighting someone else
	if target.GetFighting() != "" {
		s.Send("You can't backstab a fighting person -- they're too alert!\r\n")
		return nil
	}

	// Must wield a weapon
	wielded, hasWeapon := s.player.Equipment.GetItemInSlot(game.SlotWield)
	if !hasWeapon {
		s.Send("You need to wield a weapon to make it a success.\r\n")
		return nil
	}

	// Weapon must be piercing (TYPE_PIERCE = 11, matching values[3] in C)
	if wielded.Prototype != nil && wielded.Prototype.Values[3] != 11 {
		s.Send("Only piercing weapons can be used for backstabbing.\r\n")
		return nil
	}

	// 3. If mounted, dismount first
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// 4. MOB_AWARE check — aware mobs that are awake hit back instead
	if targetMob != nil && targetMob.HasMobFlag(game.MobFlagAware) && targetMob.GetPosition() > combat.PosSleeping {
		s.Send(fmt.Sprintf("%s notices you lunging!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "backstab",
			fmt.Sprintf("%s notices %s lunging!\r\n", targetDesc, s.player.Name))
		// Mob hits back
		combat.TakeDamage(target, s.player, 1, combat.TYPE_UNDEFINED)
		return nil
	}

	// 5. Skill check: percent = rand(1,101), prob = skill level
	// #nosec G404 — game RNG, not cryptographic
	percent := rand.Intn(101) + 1
	prob := s.player.GetSkill(game.SkillBackstab)

	awake := target.GetPosition() > combat.PosSleeping

	if awake && percent > prob {
		// Miss — deal 0 damage, still start combat
		s.Send(fmt.Sprintf("You try to backstab %s, but %s notices you!\r\n", targetDesc, targetDesc))
		broadcastCombatMsg(s, room.VNum, "backstab",
			fmt.Sprintf("%s tries to backstab %s, but fails.\r\n", s.player.Name, targetDesc))
		// Start combat even on miss (C: damage(ch, vict, 0, SKILL_BACKSTAB) still engages)
		_ = s.manager.combatEngine.StartCombat(s.player, target)
		s.markDirty(VarFighting)
	} else {
		// Hit — calculate damage using backstab multiplier
		weaponNum, weaponSides := s.player.Equipment.GetWeaponDamage()
		weaponDam := combat.RollDice(weaponNum, weaponSides)
		// backstab_mult() from class.c: level*0.2+1, capped at 20.0
		mult := float64(s.player.Level)*0.2 + 1.0
		if mult > 20.0 {
			mult = 20.0
		}
		dam := int(float64(weaponDam) * mult)

		s.Send(fmt.Sprintf("You plunge your blade into the back of %s!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "backstab",
			fmt.Sprintf("%s backstabs %s!\r\n", s.player.Name, targetDesc))

		// Apply damage via combat system
		combat.TakeDamage(s.player, target, dam, combat.SKILL_BACKSTAB)

		// Start combat
		if !s.manager.combatEngine.IsFighting(s.player.Name) {
			_ = s.manager.combatEngine.StartCombat(s.player, target)
		}
		s.markDirty(VarFighting)

		// 7. Improve skill on success (inline CircleMUD improve_skill logic)
		// Higher skill = harder to improve. INT/WIS affect chance.
		{
			cur := s.player.GetSkill(game.SkillBackstab)
			if cur > 0 && cur < 100 {
				// #nosec G404 — game RNG, not cryptographic
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404 — game RNG, not cryptographic
					if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillBackstab, cur+1)
						s.Send("You feel more competent in backstab.\r\n")
					}
				}
			}
		}
	}

	// 6. Apply WAIT_STATE (PULSE_VIOLENCE = 1 tick)
	s.player.SetWaitState(1)

	return nil
}

// cmdBash — bash a target. Ported from C ACMD(do_bash).
func cmdBash(s *Session, args []string) error {
	// 1. Must have the skill
	if s.player.GetSkill(game.SkillBash) == 0 {
		s.Send("You'd better leave all the martial arts to fighters.\r\n")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 2. Find target — by name argument, or fall back to current fight opponent
	var target combat.Combatant
	var targetDesc string
	if len(args) > 0 {
		targetName := strings.ToLower(args[0])
		mobs := s.manager.world.GetMobsInRoom(room.VNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
				strings.Contains(strings.ToLower(mob.GetName()), targetName) {
				target = mob
				targetDesc = mob.GetShortDesc()
				break
			}
		}
	}
	if target == nil {
		// Fall back to whoever the player is fighting
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(s.player.Name)
		if fighting {
			target = opponent
			targetDesc = opponent.GetName()
		} else {
			s.Send("Bash who?\r\n")
			return nil
		}
	}

	// 3. Can't bash yourself
	if target.GetName() == s.player.Name {
		s.Send("Aren't we funny today...\r\n")
		return nil
	}

	// 4. Target must be standing/fighting (position >= PosFighting)
	if target.GetPosition() < combat.PosFighting {
		s.Send("You can't bash someone who's sitting already!\r\n")
		return nil
	}

	// 5. Can't bash while mounted
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// 6. Movement cost: 10 move points
	const neededMoves = 10
	if s.player.GetMove() < neededMoves {
		s.Send("You haven't the energy!\r\n")
		return nil
	}
	s.player.SetMove(s.player.GetMove() - neededMoves)

	// 7. Skill check: percent = ((5 - (GET_AC(vict) / 10)) << 1) + rand(1,101)
	//    prob = GET_SKILL(ch, SKILL_BASH)
	// #nosec G404 — game RNG, not cryptographic
	percent := ((5-(target.GetAC()/10))<<1) + (rand.Intn(101) + 1)
	prob := s.player.GetSkill(game.SkillBash)

	// Sleeping targets always get bashed
	if target.GetPosition() <= combat.PosSleeping {
		percent = 0
	}

	if percent > prob {
		// Fail — deal 0 damage, attacker falls to sitting
		s.Send(fmt.Sprintf("You try to bash %s, but stumble and fall down!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "bash",
			fmt.Sprintf("%s tries to bash %s, but stumbles and falls!\r\n", s.player.Name, targetDesc))
		combat.TakeDamage(s.player, target, 0, combat.SKILL_BASH)
		// Attacker falls to sitting on failure
		s.player.SetPosition(combat.PosSitting)
	} else if combat.TakeDamage(s.player, target, (s.player.Level/2)+1, combat.SKILL_BASH) {
		// Success — target gets knocked to sitting, takes damage
		s.Send(fmt.Sprintf("You slam into %s and send %s sprawling!\r\n", targetDesc, targetDesc))
		broadcastCombatMsg(s, room.VNum, "bash",
			fmt.Sprintf("%s sends %s sprawling with a powerful bash!\r\n", s.player.Name, targetDesc))

		// Knock target to sitting
		switch t := target.(type) {
		case *game.Player:
			t.SetPosition(combat.PosSitting)
		case *game.MobInstance:
			t.SetStatus("sitting")
		}

		// Improve skill on success (inline CircleMUD improve_skill logic)
		{
			cur := s.player.GetSkill(game.SkillBash)
			if cur > 0 && cur < 100 {
				// #nosec G404 — game RNG, not cryptographic
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404 — game RNG, not cryptographic
					if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillBash, cur+1)
						s.Send("You feel more competent in bash.\r\n")
					}
				}
			}
		}
	}

	// Ensure we're in combat
	if !s.manager.combatEngine.IsFighting(s.player.Name) {
		_ = s.manager.combatEngine.StartCombat(s.player, target)
	}
	s.markDirty(VarFighting)

	// Attacker WAIT_STATE: PULSE_VIOLENCE * 2
	s.player.SetWaitState(2)

	return nil
}

// cmdDisembowel — disembowel a target (graphic combat action).
// cmdDisembowel — pierce a target's gut.
// Ported from do_disembowel() in src/act.offensive.c lines 234-283.
// Requires piercing weapon. Skill check: percent=rand(1,101), prob=skill.
// On hit: weapon damage via hit(), improve_skill. On miss: 0 damage.
// WAIT_STATE: PULSE_VIOLENCE*2 = 2 ticks.
func cmdDisembowel(s *Session, args []string) error {
	// 1. Must have the skill
	if s.player.GetSkill(game.SkillDisembowel) == 0 {
		s.Send("You have no idea how.\r\n")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 2. Find target — by name argument, or fall back to current fight opponent
	var target combat.Combatant
	var targetDesc string
	if len(args) > 0 {
		targetName := strings.ToLower(args[0])
		mobs := s.manager.world.GetMobsInRoom(room.VNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
				strings.Contains(strings.ToLower(mob.GetName()), targetName) {
				target = mob
				targetDesc = mob.GetShortDesc()
				break
			}
		}
	}
	if target == nil {
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(s.player.Name)
		if fighting {
			target = opponent
			targetDesc = opponent.GetName()
		} else {
			s.Send("Disembowel who?\r\n")
			return nil
		}
	}

	// 3. Can't disembowel yourself
	if target.GetName() == s.player.Name {
		s.Send("Nah. Hari Kari is for wimps.\r\n")
		return nil
	}

	// 4. Must wield a weapon
	wielded, hasWeapon := s.player.Equipment.GetItemInSlot(game.SlotWield)
	if !hasWeapon {
		s.Send("You need to wield a weapon to make it a success.\r\n")
		return nil
	}

	// 5. Weapon must be piercing (TYPE_PIERCE = 11, matching values[3] in C)
	if wielded.Prototype != nil && wielded.Prototype.Values[3] != 11 {
		s.Send("Only piercing weapons can be used for disemboweling.\r\n")
		return nil
	}

	// 6. Can't disembowel while mounted
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// 7. Skill check: percent = rand(1,101), prob = skill level
	//    101% is a complete failure
	// #nosec G404 — game RNG, not cryptographic
	percent := rand.Intn(101) + 1
	prob := s.player.GetSkill(game.SkillDisembowel)

	awake := target.GetPosition() > combat.PosSleeping

	if awake && percent > prob {
		// Miss
		s.Send(fmt.Sprintf("You try to disembowel %s, but %s dodges!\r\n", targetDesc, targetDesc))
		broadcastCombatMsg(s, room.VNum, "disembowel",
			fmt.Sprintf("%s tries to disembowel %s, but fails!\r\n", s.player.Name, targetDesc))
		combat.TakeDamage(s.player, target, 0, combat.SKILL_DISEMBOWEL)
	} else {
		// Hit — weapon damage via hit()
		weaponNum, weaponSides := s.player.Equipment.GetWeaponDamage()
		dam := combat.RollDice(weaponNum, weaponSides)

		s.Send(fmt.Sprintf("You drive your blade deep into %s's gut!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "disembowel",
			fmt.Sprintf("%s disembowels %s in a shower of gore!\r\n", s.player.Name, targetDesc))

		combat.TakeDamage(s.player, target, dam, combat.SKILL_DISEMBOWEL)

		// 8. Improve skill on success (inline CircleMUD improve_skill logic)
		{
			cur := s.player.GetSkill(game.SkillDisembowel)
			if cur > 0 && cur < 100 {
				// #nosec G404 — game RNG, not cryptographic
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404 — game RNG, not cryptographic
					if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillDisembowel, cur+1)
						s.Send("You feel more competent in disembowel.\r\n")
					}
				}
			}
		}
	}

	// Ensure we're in combat
	if !s.manager.combatEngine.IsFighting(s.player.Name) {
		_ = s.manager.combatEngine.StartCombat(s.player, target)
	}
	s.markDirty(VarFighting)

	// 9. WAIT_STATE: PULSE_VIOLENCE*2 = 2 ticks
	s.player.SetWaitState(2)

	return nil
}

// cmdRescue — rescue another character from combat.
// Ported from do_rescue() in src/act.offensive.c lines 501-567.
// The rescuer takes the victim's place: victim stops fighting, rescuer starts fighting the attacker.
func cmdRescue(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Whom do you want to rescue?\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Find the target character in the room (players and mobs)
	var vict combat.Combatant
	victName := ""

	for _, p := range s.manager.world.GetPlayersInRoom(room.VNum) {
		if p.Name == s.player.Name {
			continue
		}
		if !strings.Contains(strings.ToLower(p.Name), targetName) {
			continue
		}
		vict = p
		victName = p.Name
		break
	}
	if vict == nil {
		for _, m := range s.manager.world.GetMobsInRoom(room.VNum) {
			if !strings.Contains(strings.ToLower(m.GetShortDesc()), targetName) &&
				!strings.Contains(strings.ToLower(m.GetName()), targetName) {
				continue
			}
			vict = m
			victName = m.GetShortDesc()
			break
		}
	}
	if vict == nil {
		s.Send("They don't seem to be here.\r\n")
		return nil
	}

	// Can't rescue yourself
	if victName == s.player.Name {
		s.Send("What about fleeing instead?\r\n")
		return nil
	}

	// Can't rescue someone you're fighting
	if myTarget, ok := s.manager.combatEngine.GetCombatTarget(s.player.Name); ok {
		if myTarget.GetName() == victName {
			s.Send("How can you rescue someone you are trying to kill?\r\n")
			return nil
		}
	}

	// Must dismount first
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// Find who is fighting the victim (the attacker)
	attacker, fighting := s.manager.combatEngine.GetCombatTarget(victName)
	if !fighting {
		s.Send(fmt.Sprintf("But nobody is fighting %s!\r\n", victName))
		return nil
	}

	// Perform the rescue swap:
	// 1. Stop victim's combat
	s.manager.combatEngine.StopCombat(victName)
	// 2. Stop attacker's combat
	s.manager.combatEngine.StopCombat(attacker.GetName())
	// 3. Stop rescuer's combat if fighting anyone else
	s.manager.combatEngine.StopCombat(s.player.Name)
	// 4. Start new fight: rescuer vs attacker
	if err := s.manager.combatEngine.StartCombat(s.player, attacker); err != nil {
		s.Send(err.Error())
		return nil
	}

	s.Send("Banzai! To the rescue...\r\n")
	// Notify the victim
	if !vict.IsNPC() {
		if victSess, ok := s.manager.GetSession(victName); ok {
			victSess.Send(fmt.Sprintf("You are rescued by %s, you are confused!\r\n", s.player.Name))
		}
	}
	broadcastCombatMsg(s, room.VNum, "rescue",
		fmt.Sprintf("%s heroically rescues %s!", s.player.Name, victName))
	s.markDirty(VarFighting)
	return nil
}

// cmdKick — kick a target.
// Ported from do_kick() in src/act.offensive.c lines 587-633.
// Can target by name or default to current fight opponent.
// Skill check: percent=((7-(AC/10))<<1)+rand(1,101), prob=skill level.
// On hit: damage = level>>1, improve_skill. On miss: 0 damage.
// WAIT_STATE: PULSE_VIOLENCE + 2 = 2 ticks.
func cmdKick(s *Session, args []string) error {
	// 1. Must have the skill
	if s.player.GetSkill(game.SkillKick) == 0 {
		s.Send("You'd better leave all the martial arts to fighters.\r\n")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 2. Find target — by name argument, or fall back to current fight opponent
	var target combat.Combatant
	var targetDesc string
	if len(args) > 0 {
		targetName := strings.ToLower(args[0])
		mobs := s.manager.world.GetMobsInRoom(room.VNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
				strings.Contains(strings.ToLower(mob.GetName()), targetName) {
				target = mob
				targetDesc = mob.GetShortDesc()
				break
			}
		}
	}
	if target == nil {
		// Fall back to whoever the player is fighting
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(s.player.Name)
		if fighting {
			target = opponent
			targetDesc = opponent.GetName()
		} else {
			s.Send("Kick who?\r\n")
			return nil
		}
	}

	// 3. Can't kick yourself
	if target.GetName() == s.player.Name {
		s.Send("Aren't we funny today...\r\n")
		return nil
	}

	// 4. Can't kick while mounted
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// 5. Skill check: percent = ((7 - (GET_AC(vict) / 10)) << 1) + rand(1,101)
	//    prob = GET_SKILL(ch, SKILL_KICK)
	//    101% is a complete failure
	// #nosec G404 — game RNG, not cryptographic
	percent := ((7-(target.GetAC()/10))<<1) + (rand.Intn(101) + 1)
	prob := s.player.GetSkill(game.SkillKick)

	if percent > prob {
		// Miss
		s.Send("You miss!\r\n")
		broadcastCombatMsg(s, room.VNum, "kick",
			fmt.Sprintf("%s tries to kick %s, but misses!\r\n", s.player.Name, targetDesc))
		combat.TakeDamage(s.player, target, 0, combat.SKILL_KICK)
	} else {
		// Hit — damage = GET_LEVEL(ch) >> 1
		dam := s.player.Level >> 1

		s.Send(fmt.Sprintf("You kick %s!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "kick",
			fmt.Sprintf("%s kicks %s!\r\n", s.player.Name, targetDesc))

		combat.TakeDamage(s.player, target, dam, combat.SKILL_KICK)

		// 6. Improve skill on success (inline CircleMUD improve_skill logic)
		{
			cur := s.player.GetSkill(game.SkillKick)
			if cur > 0 && cur < 100 {
				// #nosec G404 — game RNG, not cryptographic
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404 — game RNG, not cryptographic
					if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillKick, cur+1)
						s.Send("You feel more competent in kick.\r\n")
					}
				}
			}
		}
	}

	// Ensure we're in combat
	if !s.manager.combatEngine.IsFighting(s.player.Name) {
		_ = s.manager.combatEngine.StartCombat(s.player, target)
	}
	s.markDirty(VarFighting)

	// 7. WAIT_STATE: PULSE_VIOLENCE + 2 = 2 RL_SEC + 2 = 4 RL_SEC = 2 ticks
	s.player.SetWaitState(2)

	return nil
}

// cmdDragonKick — dragon-style kick attack.
// Ported from do_dragon_kick() in src/act.offensive.c lines 636-690.
// Requires 10 move. Skill check: percent=((5-(AC/10))<<1)+rand(1,101), prob=skill.
// On hit: damage = level*1.5, improve_skill. On miss: 0 damage.
// WAIT_STATE: PULSE_VIOLENCE+2 = 1 tick.
func cmdDragonKick(s *Session, args []string) error {
	// 1. Must have the skill
	if s.player.GetSkill(game.SkillDragonKick) == 0 {
		s.Send("What's that, idiot-san?\r\n")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 2. Find target — by name argument, or fall back to current fight opponent
	var target combat.Combatant
	var targetDesc string
	if len(args) > 0 {
		targetName := strings.ToLower(args[0])
		mobs := s.manager.world.GetMobsInRoom(room.VNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
				strings.Contains(strings.ToLower(mob.GetName()), targetName) {
				target = mob
				targetDesc = mob.GetShortDesc()
				break
			}
		}
	}
	if target == nil {
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(s.player.Name)
		if fighting {
			target = opponent
			targetDesc = opponent.GetName()
		} else {
			s.Send("Kick who?\r\n")
			return nil
		}
	}

	// 3. Can't kick yourself
	if target.GetName() == s.player.Name {
		s.Send("Aren't we funny today...\r\n")
		return nil
	}

	// 4. Can't kick while mounted
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// 5. Move check — requires 10 move
	if s.player.Move < 10 {
		s.Send("You're too exhausted!\r\n")
		return nil
	}
	s.player.Move -= 10

	// 6. Skill check: percent = ((5-(GET_AC(vict)/10))<<1) + rand(1,101)
	//    prob = GET_SKILL(ch, SKILL_DRAGON_KICK)
	//    101% is a complete failure
	// #nosec G404 — game RNG, not cryptographic
	percent := ((5-(target.GetAC()/10))<<1) + (rand.Intn(101) + 1)
	prob := s.player.GetSkill(game.SkillDragonKick)

	if percent > prob {
		// Miss
		s.Send(fmt.Sprintf("You attempt a dragon kick on %s but miss!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "dragon_kick",
			fmt.Sprintf("%s attempts a dragon kick on %s but misses!\r\n", s.player.Name, targetDesc))
		combat.TakeDamage(s.player, target, 0, combat.SKILL_DRAGON_KICK)
	} else {
		// Hit — damage = GET_LEVEL(ch) * 1.5
		dam := int(float64(s.player.Level) * 1.5)

		s.Send(fmt.Sprintf("You unleash a devastating dragon kick against %s!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "dragon_kick",
			fmt.Sprintf("%s dragon kicks %s!\r\n", s.player.Name, targetDesc))

		combat.TakeDamage(s.player, target, dam, combat.SKILL_DRAGON_KICK)

		// 7. Improve skill on success (inline CircleMUD improve_skill logic)
		{
			cur := s.player.GetSkill(game.SkillDragonKick)
			if cur > 0 && cur < 100 {
				// #nosec G404 — game RNG, not cryptographic
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404 — game RNG, not cryptographic
					if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillDragonKick, cur+1)
						s.Send("You feel more competent in dragon kick.\r\n")
					}
				}
			}
		}
	}

	// Ensure we're in combat
	if !s.manager.combatEngine.IsFighting(s.player.Name) {
		_ = s.manager.combatEngine.StartCombat(s.player, target)
	}
	s.markDirty(VarFighting)

	// 8. WAIT_STATE: PULSE_VIOLENCE + 2 = 1 tick (PULSE_VIOLENCE=1 RL_SEC, +2 rounds down to 1 tick)
	s.player.SetWaitState(1)

	return nil
}

// cmdTigerPunch — tiger-style punch attack.
// Ported from do_tiger_punch() in src/act.offensive.c lines 693-744.
// Requires bare hands (no weapon wielded).
// Skill check: percent=((7-(AC/10))<<1)+rand(1,101), prob=skill.
// On hit: damage = level*2.5, improve_skill. On miss: 0 damage.
// WAIT_STATE: PULSE_VIOLENCE*2 = 2 ticks.
func cmdTigerPunch(s *Session, args []string) error {
	// 1. Must have the skill
	if s.player.GetSkill(game.SkillTigerPunch) == 0 {
		s.Send("What's that, idiot-san?\r\n")
		return nil
	}

	// 2. Must NOT be wielding a weapon (bare-handed only)
	_, hasWeapon := s.player.Equipment.GetItemInSlot(game.SlotWield)
	if hasWeapon {
		s.Send("That's pretty tough to do while wielding a weapon.\r\n")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 3. Find target — by name argument, or fall back to current fight opponent
	var target combat.Combatant
	var targetDesc string
	if len(args) > 0 {
		targetName := strings.ToLower(args[0])
		mobs := s.manager.world.GetMobsInRoom(room.VNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
				strings.Contains(strings.ToLower(mob.GetName()), targetName) {
				target = mob
				targetDesc = mob.GetShortDesc()
				break
			}
		}
	}
	if target == nil {
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(s.player.Name)
		if fighting {
			target = opponent
			targetDesc = opponent.GetName()
		} else {
			s.Send("Hit who?\r\n")
			return nil
		}
	}

	// 4. Can't punch yourself
	if target.GetName() == s.player.Name {
		s.Send("Aren't we funny today...\r\n")
		return nil
	}

	// 5. Can't punch while mounted
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// 6. Skill check: percent = ((7-(GET_AC(vict)/10))<<1) + rand(1,101)
	//    prob = GET_SKILL(ch, SKILL_TIGER_PUNCH)
	//    101% is a complete failure
	// #nosec G404 — game RNG, not cryptographic
	percent := ((7-(target.GetAC()/10))<<1) + (rand.Intn(101) + 1)
	prob := s.player.GetSkill(game.SkillTigerPunch)

	if percent > prob {
		// Miss
		s.Send(fmt.Sprintf("You snap a tiger punch at %s but miss!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "tiger_punch",
			fmt.Sprintf("%s tries to tiger punch %s but misses!\r\n", s.player.Name, targetDesc))
		combat.TakeDamage(s.player, target, 0, combat.SKILL_TIGER_PUNCH)
	} else {
		// Hit — damage = GET_LEVEL(ch) * 2.5
		dam := int(float64(s.player.Level) * 2.5)

		s.Send(fmt.Sprintf("You snap a lightning-fast tiger punch into %s!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "tiger_punch",
			fmt.Sprintf("%s tiger punches %s!\r\n", s.player.Name, targetDesc))

		combat.TakeDamage(s.player, target, dam, combat.SKILL_TIGER_PUNCH)

		// 7. Improve skill on success (inline CircleMUD improve_skill logic)
		{
			cur := s.player.GetSkill(game.SkillTigerPunch)
			if cur > 0 && cur < 100 {
				// #nosec G404 — game RNG, not cryptographic
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404 — game RNG, not cryptographic
					if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillTigerPunch, cur+1)
						s.Send("You feel more competent in tiger punch.\r\n")
					}
				}
			}
		}
	}

	// Ensure we're in combat
	if !s.manager.combatEngine.IsFighting(s.player.Name) {
		_ = s.manager.combatEngine.StartCombat(s.player, target)
	}
	s.markDirty(VarFighting)

	// 8. WAIT_STATE: PULSE_VIOLENCE*2 = 2 ticks
	s.player.SetWaitState(2)

	return nil
}

// cmdShoot — ranged attack with bow/sling and projectile.
// Ported from do_shoot() in src/act.offensive.c lines 746-970.
// C signature: shoot <projectile> <direction> <target>
// Go simplified: shoot <direction> <target> (finds first projectile + bow in inventory/equip).
func cmdShoot(s *Session, args []string) error {
	// 1. Skill check
	if s.player.GetSkill(game.SkillShoot) == 0 {
		s.Send("You have no idea how.\r\n")
		return nil
	}

	// 2. Can't shoot while fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("But you are already engaged in close-range combat!\r\n")
		return nil
	}

	if len(args) < 2 {
		s.Send("Shoot what where at who?\r\n")
		return nil
	}

	dirName := strings.ToLower(args[0])
	targetName := strings.ToLower(strings.Join(args[1:], " "))

	// 3. Must wield a bow/sling (fireweapon)
	bow, hasBow := s.player.Equipment.GetItemInSlot(game.SlotWield)
	if !hasBow || bow.Prototype == nil {
		s.Send("You must wield a bow or sling to fire a projectile.\r\n")
		return nil
	}
	if bow.Prototype.TypeFlag != 12 { // ITEM_FIREWEAPON = 12
		s.Send("You must wield a bow or sling to fire a projectile.\r\n")
		return nil
	}

	// Check exit exists and is open
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}
	exit, exitExists := room.Exits[dirName]
	if !exitExists || exit.ToRoom == 0 {
		s.Send("Alas, you cannot shoot that way...\r\n")
		return nil
	}
	if exit.DoorState >= 1 { // door is closed or locked
		s.Send("It seems to be closed.\r\n")
		return nil
	}

	targRoom := exit.ToRoom
	fromDir := dirName // direction name for messages

	// Peaceful room check
	targRoomData, hasTargRoom := s.manager.world.GetRoom(targRoom)
	if hasTargRoom {
		for _, f := range targRoomData.Flags {
			if f == "peaceful" {
				s.Send("You feel too peaceful to contemplate violence.\r\n")
				return nil
			}
		}
	}
	for _, f := range room.Flags {
		if f == "peaceful" {
			s.Send("You feel too peaceful to contemplate violence.\r\n")
			return nil
		}
	}

	// 4. Find target in target room
	targetMobs := s.manager.world.GetMobsInRoom(targRoom)
	var target combat.Combatant
	var targetDesc string
	for _, mob := range targetMobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			target = mob
			targetDesc = mob.GetShortDesc()
			break
		}
	}
	if target == nil {
		s.Send("Twang...\r\n")
		return nil
	}

	// Can't shoot at fighting targets or sentinel mobs
	if target.GetFighting() != "" {
		s.Send("It looks like they are fighting, you can't aim properly.\r\n")
		return nil
	}
	if mob, ok := target.(*game.MobInstance); ok && mob.HasMobFlag(game.MobFlagSentinel) {
		s.Send("You cannot see well enough to aim...\r\n")
		return nil
	}

	s.Send("Twang... your projectile flies into the distance.\r\n")
	broadcastCombatMsg(s, room.VNum, "shoot",
		fmt.Sprintf("%s fires a projectile %s!\r\n", s.player.Name, fromDir))

	// 5. Skill check: percent = rand(1,101), prob = skill + dex modifiers
	// C: prob += dex_app[GET_DEX(ch)].miss_att * 10; prob -= dex_app[GET_DEX(to)].reaction * 10
	// Simplified: use raw skill since dex_app table not yet ported
	// #nosec G404 — game RNG
	percent := rand.Intn(101) + 1
	prob := s.player.GetSkill(game.SkillShoot)

	if percent < prob {
		// Hit — calc damage
		// C: dam = GET_DAMROLL(ch) + dice(proj) + dice(bow)
		dam := s.player.GetDamroll()
		weaponNum, weaponSides := s.player.Equipment.GetWeaponDamage()
		dam += combat.RollDice(weaponNum, weaponSides)
		s.Send("You hear a roar of pain!\r\n")
		combat.TakeDamage(s.player, target, dam, 148) // SKILL_SHOOT = 148

		broadcastCombatMsg(s, targRoom, "shoot",
			fmt.Sprintf("Some kind of projectile streaks in from %s and strikes %s!\r\n", fromDir, targetDesc))

		// 6. Improve skill on success
		{
			cur := s.player.GetSkill(game.SkillShoot)
			if cur > 0 && cur < 100 {
				// #nosec G404
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404
					if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillShoot, cur+1)
						s.Send("You feel more competent in shoot.\r\n")
					}
				}
			}
		}
	} else {
		// Miss
		broadcastCombatMsg(s, targRoom, "shoot",
			fmt.Sprintf("Some kind of projectile streaks in from %s and narrowly misses %s!\r\n", fromDir, targetDesc))
	}

	// 7. WAIT_STATE
	s.player.SetWaitState(1)
	return nil
}

// cmdSubdue — non-lethal subduing attack.
// Ported from do_subdue() in src/act.offensive.c lines 1084-1180.
func cmdSubdue(s *Session, args []string) error {
	// 1. Skill check
	if s.player.GetSkill(game.SkillSubdue) == 0 {
		s.Send("You have no idea how!\r\n")
		return nil
	}

	// 2. Can't subdue while fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're too busy right now!\r\n")
		return nil
	}

	if len(args) == 0 {
		s.Send("Subdue who?\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 3. Find target mob
	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	var target combat.Combatant
	var targetMob *game.MobInstance
	var targetDesc string
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			target = mob
			targetMob = mob
			targetDesc = mob.GetShortDesc()
			break
		}
	}
	if target == nil {
		s.Send("Subdue who?\r\n")
		return nil
	}

	// Target must not be self
	if target.GetName() == s.player.Name {
		s.Send("Aren't we funny today...\r\n")
		return nil
	}

	// Peaceful room check
	for _, f := range room.Flags {
		if f == "peaceful" {
			s.Send("You can't contemplate violence in such a place!\r\n")
			return nil
		}
	}

	// Mounted check
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// Target must not be fighting
	if target.GetFighting() != "" {
		s.Send("You can't get close enough!\r\n")
		return nil
	}

	// Target already stunned/sleeping
	if target.GetPosition() <= combat.PosStunned {
		s.Send("What's the point of doing that now?\r\n")
		return nil
	}

	// 4. Skill check: percent = rand(1, 101+victim_level), prob = skill
	// #nosec G404
	percent := rand.Intn(101+target.GetLevel()) + 1
	prob := s.player.GetSkill(game.SkillSubdue)

	// Level advantage: harder to subdue higher-level targets
	percent += max(target.GetLevel()-s.player.Level, 0)

	// MOB_AWARE makes subdue impossible
	if targetMob != nil && targetMob.HasMobFlag(game.MobFlagAware) {
		prob = 0
	}

	// Level range check for PvP (±3 levels)
	if !target.IsNPC() && s.player.Level < game.LVL_IMMORT {
		if target.GetLevel() > s.player.Level+3 || target.GetLevel() < s.player.Level-3 {
			prob = 0
		}
	}

	if percent > prob {
		// Miss — victim hits back
		s.Send(fmt.Sprintf("%s avoids your misplaced blow to the back of the head.\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "subdue",
			fmt.Sprintf("%s avoids %s's misplaced blow to the back of the head.\r\n", targetDesc, s.player.Name))
		combat.TakeDamage(s.player, target, 0, combat.TYPE_UNDEFINED)
	} else {
		// Hit — stun the target (deal moderate damage as proxy for stun)
		s.Send(fmt.Sprintf("You knock %s out cold.\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "subdue",
			fmt.Sprintf("%s knocks out %s with a well-placed blow to the back of the head.\r\n", s.player.Name, targetDesc))

		stunDam := s.player.Level
		combat.TakeDamage(s.player, target, stunDam, 152) // SKILL_SUBDUE = 152

		// 6. Improve skill on success
		{
			cur := s.player.GetSkill(game.SkillSubdue)
			if cur > 0 && cur < 100 {
				// #nosec G404
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404
				if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillSubdue, cur+1)
						s.Send("You feel more competent in subdue.\r\n")
					}
				}
			}
		}
	}

	// 5. WAIT_STATE: PULSE_VIOLENCE * 1
	s.player.SetWaitState(1)
	return nil
}

// cmdSleeper — put target to sleep with a choke hold.
// Ported from do_sleeper() in src/act.offensive.c lines 1184-1293.
func cmdSleeper(s *Session, args []string) error {
	// 1. Skill check
	if s.player.GetSkill(game.SkillSleeper) == 0 {
		s.Send("You have no idea how.\r\n")
		return nil
	}

	// Can't use while fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You can't do this while fighting!\r\n")
		return nil
	}

	// Mounted check
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Peaceful room check
	for _, f := range room.Flags {
		if f == "peaceful" {
			s.Send("This room just has such a peaceful, easy feeling...\r\n")
			return nil
		}
	}

	// Must not wield a weapon
	_, hasWeapon := s.player.Equipment.GetItemInSlot(game.SlotWield)
	if hasWeapon {
		s.Send("You can't get a good grip on them while you are holding that weapon!\r\n")
		return nil
	}

	if len(args) == 0 {
		s.Send("Sleeper who?\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])

	// 2. Find target mob
	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	var target combat.Combatant
	var targetMob *game.MobInstance
	var targetDesc string
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			target = mob
			targetMob = mob
			targetDesc = mob.GetShortDesc()
			break
		}
	}
	if target == nil {
		s.Send("Sleeper who?\r\n")
		return nil
	}

	// Self check
	if target.GetName() == s.player.Name {
		s.Send("Can't get to sleep fast enough, huh?\r\n")
		return nil
	}

	// Target fighting check
	if target.GetFighting() != "" {
		s.Send("You can't get a good grip on them while they're fighting!\r\n")
		return nil
	}

	// Already sleeping
	if target.GetPosition() <= combat.PosSleeping {
		s.Send("What's the point of doing that now?\r\n")
		return nil
	}

	// 3. Skill check: percent = rand(1, 101+victim_level), prob = skill
	// #nosec G404
	percent := rand.Intn(101+target.GetLevel()) + 1
	prob := s.player.GetSkill(game.SkillSleeper)

	// MOB_AWARE / MOB_NOSLEEP makes it impossible
	if targetMob != nil && (targetMob.HasMobFlag(game.MobFlagAware) || targetMob.HasMobFlag(game.MobFlagNosleep)) {
		prob = 0
	}

	// Level range check for PvP (±3 levels)
	if !target.IsNPC() && s.player.Level < game.LVL_IMMORT {
		if target.GetLevel() > s.player.Level+3 || target.GetLevel() < s.player.Level-3 {
			prob = 0
		}
	}

	// Level advantage modifier
	percent += max(target.GetLevel()-s.player.Level, 0)

	if percent > prob {
		// Failed — victim breaks free and hits back
		s.Send(fmt.Sprintf("You try to grab %s in a sleeper hold but fail!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "sleeper",
			fmt.Sprintf("%s tries to put %s in a sleeper hold...\r\n", s.player.Name, targetDesc))
		combat.TakeDamage(s.player, target, 0, combat.TYPE_UNDEFINED)
	} else {
		// Success — put target to sleep (deal damage as proxy for sleep effect)
		s.Send(fmt.Sprintf("You put %s in a sleeper hold.\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "sleeper",
			fmt.Sprintf("%s puts %s in a sleeper hold.\r\n", s.player.Name, targetDesc))
		broadcastCombatMsg(s, room.VNum, "sleeper",
			fmt.Sprintf("%s goes to sleep.\r\n", targetDesc))

		sleepDam := s.player.Level / 2
		combat.TakeDamage(s.player, target, sleepDam, 187) // SKILL_SLEEPER = 187

		// 6. Improve skill on success
		{
			cur := s.player.GetSkill(game.SkillSleeper)
			if cur > 0 && cur < 100 {
				// #nosec G404
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404
				if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillSleeper, cur+1)
						s.Send("You feel more competent in sleeper.\r\n")
					}
				}
			}
		}
	}

	// 5. WAIT_STATE
	s.player.SetWaitState(1)
	return nil
}

// cmdNeckbreak — lethal unarmed neck-breaking attack.
// Ported from do_neckbreak() in src/act.offensive.c lines 1295-1395.
func cmdNeckbreak(s *Session, args []string) error {
	const neededMoves = 51

	// 1. Skill check
	if s.player.GetSkill(game.SkillNeckbreak) == 0 {
		s.Send("What's that, idiot-san?\r\n")
		return nil
	}

	// Must not wield a weapon
	_, hasWeapon := s.player.Equipment.GetItemInSlot(game.SlotWield)
	if hasWeapon {
		s.Send("You can't do this and wield a weapon at the same time!\r\n")
		return nil
	}

	if len(args) == 0 {
		s.Send("I don't see them here.\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 2. Find target mob
	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	var target combat.Combatant
	var targetDesc string
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			target = mob
			targetDesc = mob.GetShortDesc()
			break
		}
	}
	if target == nil {
		s.Send("I don't see them here.\r\n")
		return nil
	}

	// Self check
	if target.GetName() == s.player.Name {
		s.Send("Aren't we funny today...\r\n")
		return nil
	}

	// Peaceful room check
	for _, f := range room.Flags {
		if f == "peaceful" {
			s.Send("You can't contemplate violence in such a place!\r\n")
			return nil
		}
	}

	// Mounted check
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	// Move cost
	if s.player.GetMove() < neededMoves {
		s.Send("You haven't the energy to do this!\r\n")
		return nil
	}
	s.player.SetMove(s.player.GetMove() - neededMoves)

	// 3. Skill check: percent = ((7 - (AC/10)) << 1) + rand(1,101)
	//    prob = skill level
	// #nosec G404
	percent := ((7-(target.GetAC()/10))<<1) + (rand.Intn(101) + 1)
	prob := s.player.GetSkill(game.SkillNeckbreak)

	if percent > prob {
		// Miss — victim hits back
		s.Send(fmt.Sprintf("You try to break %s's neck, but %s is too strong!\r\n", targetDesc, targetDesc))
		broadcastCombatMsg(s, room.VNum, "neckbreak",
			fmt.Sprintf("%s tries to break %s's neck, but %s slips free!\r\n", s.player.Name, targetDesc, targetDesc))
		combat.TakeDamage(s.player, target, 0, combat.SKILL_NECKBREAK)
	} else {
		// Hit — C: dam = dice(18, GET_LEVEL(ch))
		dam := combat.RollDice(18, s.player.Level)

		s.Send(fmt.Sprintf("You break %s's neck!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "neckbreak",
			fmt.Sprintf("%s breaks %s's neck!\r\n", s.player.Name, targetDesc))

		combat.TakeDamage(s.player, target, dam, combat.SKILL_NECKBREAK)

		// 6. Improve skill on success
		{
			cur := s.player.GetSkill(game.SkillNeckbreak)
			if cur > 0 && cur < 100 {
				// #nosec G404
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404
				if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillNeckbreak, cur+1)
						s.Send("You feel more competent in neckbreak.\r\n")
					}
				}
			}
		}
	}

	// 5. WAIT_STATE: PULSE_VIOLENCE * 3
	s.player.SetWaitState(3)
	return nil
}

// cmdAmbush — ambush attack from hiding (forest/hills/mountain/city only).
// Ported from do_ambush() + ambush_event() in src/act.offensive.c lines 1454-1510.
// C version uses a delayed event (PULSE_VIOLENCE*2); Go version resolves immediately.
func cmdAmbush(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Ambush who?\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 2. Find target mob
	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	var target combat.Combatant
	var targetMob *game.MobInstance
	var targetDesc string
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			target = mob
			targetMob = mob
			targetDesc = mob.GetShortDesc()
			break
		}
	}
	if target == nil {
		s.Send("Ambush who?\r\n")
		return nil
	}

	// 1. Skill check
	if s.player.GetSkill(game.SkillAmbush) == 0 {
		s.Send("You'd better not.\r\n")
		return nil
	}

	// Already fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You are a little busy for that right now!\r\n")
		return nil
	}

	// Self check
	if target.GetName() == s.player.Name {
		s.Send("Ambush yourself? You idiot!\r\n")
		return nil
	}

	// Sector check: only forest(3), hills(4), mountain(5), city(1)
	sector := room.Sector
	if sector != 3 && sector != 4 && sector != 5 && sector != 1 {
		s.Send("Ambush someone here? Impossible!\r\n")
		return nil
	}

	// Target must not be fighting
	if target.GetFighting() != "" {
		s.Send("They're too alert for that, currently.\r\n")
		return nil
	}

	// 3. Skill check from ambush_event(): percent = rand(1,131), prob = skill
	// #nosec G404
	percent := rand.Intn(131) + 1
	prob := s.player.GetSkill(game.SkillAmbush)

	// MOB_AWARE makes ambush always fail
	if targetMob != nil && targetMob.HasMobFlag(game.MobFlagAware) {
		percent = 200
	}

	s.Send("You crouch in the shadows and plan your ambush...\r\n")

	if percent > prob {
		// Failure
		combat.TakeDamage(s.player, target, 0, 191) // SKILL_AMBUSH = 191
		s.Send("Your ambush fails!\r\n")
		broadcastCombatMsg(s, room.VNum, "ambush",
			fmt.Sprintf("%s springs an ambush on %s but fails!\r\n", s.player.Name, targetDesc))
	} else {
		// Hit — C: dam = str_app todam + damroll + weapon_dice + level*2.6 + 10% if hidden
		dam := s.player.GetDamroll()
		weaponNum, weaponSides := s.player.Equipment.GetWeaponDamage()
		dam += combat.RollDice(weaponNum, weaponSides)
		dam += int(float64(s.player.Level) * 2.6)

		// 10% bonus if hidden
		if s.player.Affects&(1<<uint(combat.AFF_HIDE)) != 0 {
			dam += dam / 10
		}

		s.Send(fmt.Sprintf("You spring from the shadows and ambush %s!\r\n", targetDesc))
		broadcastCombatMsg(s, room.VNum, "ambush",
			fmt.Sprintf("%s leaps from the shadows and ambushes %s!\r\n", s.player.Name, targetDesc))

		combat.TakeDamage(s.player, target, dam, 191) // SKILL_AMBUSH = 191

		// 6. Improve skill on success
		{
			cur := s.player.GetSkill(game.SkillAmbush)
			if cur > 0 && cur < 100 {
				// #nosec G404
				if rand.Intn(100)+1 > cur {
					chance := (s.player.GetInt() + s.player.GetWis()) / 4
					// #nosec G404
				if rand.Intn(100) < chance {
						s.player.SetSkill(game.SkillAmbush, cur+1)
						s.Send("You feel more competent in ambush.\r\n")
					}
				}
			}
		}
	}

	// 5. WAIT_STATE: PULSE_VIOLENCE = 1
	s.player.SetWaitState(1)
	return nil
}

// cmdOrder — order a pet or follower to perform a command (LVL_IMMORT).
func cmdOrder(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Order whom to do what?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	orderCmd := strings.Join(args[1:], " ")

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	for _, mob := range s.manager.world.GetMobsInRoom(room.VNum) {
		if strings.Contains(strings.ToLower(mob.GetName()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			s.Send(fmt.Sprintf("%s obeys your order: %s", mob.GetShortDesc(), orderCmd))
			broadcastCombatMsg(s, room.VNum, "order",
				fmt.Sprintf("%s orders %s to '%s'.", s.player.Name, mob.GetShortDesc(), orderCmd))
			return nil
		}
	}

	s.Send("No follower by that name here.")
	return nil
}
