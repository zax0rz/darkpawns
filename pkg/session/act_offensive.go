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

// cmdAssist — assist a target in their combat (LVL_IMMORT).
func cmdAssist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Assist whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Find the player being assisted
	for _, p := range s.manager.world.GetPlayersInRoom(room.VNum) {
		if p.Name == s.player.Name {
			continue
		}
		if !strings.Contains(strings.ToLower(p.Name), targetName) {
			continue
		}
		// Find who they're fighting
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(p.Name)
		if !fighting {
			s.Send(fmt.Sprintf("%s is not in combat.", p.Name))
			return nil
		}
		if s.manager.combatEngine.IsFighting(s.player.Name) {
			s.Send("You're already fighting someone!")
			return nil
		}
		if err := s.manager.combatEngine.StartCombat(s.player, opponent); err != nil {
			s.Send(err.Error())
			return nil
		}
		s.Send(fmt.Sprintf("You jump to the aid of %s!", p.Name))
		broadcastCombatMsg(s, room.VNum, "assist",
			fmt.Sprintf("%s jumps to the aid of %s!", s.player.Name, p.Name))
		s.markDirty(VarFighting)
		return nil
	}

	s.Send("They aren't here.")
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
func cmdDisembowel(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Disembowel who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You drive your blade deep into %s's gut, spilling entrails everywhere!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "disembowel",
				fmt.Sprintf("%s disembowels %s in a shower of gore!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdRescue — rescue another player from combat.
func cmdRescue(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Rescue who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	for _, p := range s.manager.world.GetPlayersInRoom(room.VNum) {
		if p.Name == s.player.Name {
			continue
		}
		if !strings.Contains(strings.ToLower(p.Name), targetName) {
			continue
		}
		// Take their combat opponent
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(p.Name)
		if !fighting {
			s.Send(fmt.Sprintf("%s doesn't need rescuing!", p.Name))
			return nil
		}
		s.manager.combatEngine.StopCombat(p.Name)
		if err := s.manager.combatEngine.StartCombat(s.player, opponent); err != nil {
			s.Send(err.Error())
			return nil
		}
		s.Send(fmt.Sprintf("You valiantly rescue %s!", p.Name))
		if target, ok := s.manager.GetSession(p.Name); ok {
			target.Send(fmt.Sprintf("%s rescues you!", s.player.Name))
		}
		broadcastCombatMsg(s, room.VNum, "rescue",
			fmt.Sprintf("%s rescues %s!", s.player.Name, p.Name))
		s.markDirty(VarFighting)
		return nil
	}

	s.Send("They aren't here.")
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
func cmdDragonKick(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Dragon kick whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You unleash a devastating dragon kick against %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "dragon_kick",
				fmt.Sprintf("%s dragon kicks %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdTigerPunch — tiger-style punch attack.
func cmdTigerPunch(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Tiger punch whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You snap a lightning-fast tiger punch into %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "tiger_punch",
				fmt.Sprintf("%s tiger punches %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdShoot — ranged attack with bow or gun.
func cmdShoot(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Shoot who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You fire at %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "shoot",
				fmt.Sprintf("%s shoots at %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdSubdue — non-lethal subduing attack.
func cmdSubdue(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Subdue who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You attempt to subdue %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "subdue",
				fmt.Sprintf("%s tries to subdue %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdSleeper — put target to sleep.
func cmdSleeper(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Use a sleeper hold on who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You apply a sleeper hold to %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "sleeper",
				fmt.Sprintf("%s applies a sleeper hold to %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdNeckbreak — lethal stealth attack.
func cmdNeckbreak(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Neckbreak who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You snap %s's neck with a sickening crack!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "neckbreak",
				fmt.Sprintf("%s breaks %s's neck!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdAmbush — ambush attack from hiding.
func cmdAmbush(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Ambush who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You can't ambush while you're already fighting!")
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
			s.Send(fmt.Sprintf("You spring from the shadows and ambush %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "ambush",
				fmt.Sprintf("%s leaps from the shadows to ambush %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
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
