package session

import (
	"fmt"
	"math/rand"
	"strings"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

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
