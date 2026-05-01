package session

import (
	"fmt"
	"math/rand"
	"strings"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

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
