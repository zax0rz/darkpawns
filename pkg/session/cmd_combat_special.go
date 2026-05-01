package session

import (
	"fmt"
	"math/rand"
	"strings"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

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
