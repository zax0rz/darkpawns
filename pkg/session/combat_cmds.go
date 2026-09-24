package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// cmdHit initiates combat with a target.
// Combat is self-rate-limited by the 2s engine tick.
// StartCombat enrolls the player; PerformRound fires autonomously.
func cmdKill(s *Session, args []string) error {
	// Mortals delegate to hit before argument parsing. C therefore answers a
	// mortal bare "kill" with do_hit's "Hit who?", not "Kill who?".
	if s.player.GetLevel() < LVL_IMPL-1 {
		return cmdHit(s, args)
	}

	if len(args) == 0 {
		s.Send("Kill who?\r\n")
		return nil
	}

	// Immortal instakill (C: src/act.offensive.c do_kill())
	// C gates this to GET_LEVEL(ch) >= LVL_IMPL-1 (level 39+), NOT LVL_IMMORT
	// (31+). Every immortal used to have implementor-grade instakill power.
	// (DP-1041)
	// Resolve via the canonical in-room resolver (DP-907) so `kill X`
	// agrees with consider/kick/... on what "X" is.
	tgt, found := s.manager.world.ResolveCharInRoom(s.player, strings.Join(args, " "))
	if !found {
		s.Send("They aren't here.\r\n")
		return nil
	}

	victim := combatTargetActor(tgt)
	if victim == nil {
		s.Send("They aren't here.\r\n")
		return nil
	}
	if victim == s.player {
		s.Send("Your mother would be so sad.. :(\r\n")
		return nil
	}
	if tgt.Combatant.GetLevel() == s.player.GetLevel() {
		s.Send("No can do, buddy.. \r\n")
		return nil
	}

	game.Act(nil, false, s.player, victim, nil, nil,
		"You chop $M to pieces!  Ah!  The blood!", "", game.ToChar)
	game.Act(nil, false, victim, s.player, nil, nil,
		"$N chops you to pieces!", "", game.ToChar)
	game.Act(s.manager.world, false, s.player, victim, nil, nil,
		"$n brutally slays $N!", "", game.ToNotVict)
	s.manager.world.Instakill(tgt.Combatant, s.player, 0)
	return nil
}

func cmdHit(s *Session, args []string) error {
	// C do_hit calls one_argument(), not a whole-argument lookup: leading fill
	// words are discarded and only the first non-fill token is passed to
	// get_char_room_vis. This also makes `hit target trailing words` address
	// target, exactly as the C command does (R1/R2/R5e).
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	if targetName == "" {
		s.Send("Hit who?\r\n")
		return nil
	}

	if _, ok := s.manager.world.GetRoom(s.player.GetRoom()); !ok {
		return fmt.Errorf("invalid room")
	}

	// Resolve target via the canonical in-room resolver (DP-907): keyword-list
	// abbreviation matching, ordinals, self/me, visibility — identical to
	// consider/kick/backstab/...
	tgt, found := s.manager.world.ResolveCharInRoom(s.player, targetName)
	if !found {
		s.Send("They don't seem to be here.\r\n")
		return nil
	}
	victim := combatTargetActor(tgt)
	if victim == nil {
		s.Send("They don't seem to be here.\r\n")
		return nil
	}

	// do_hit handles self and a charmed character's master before position,
	// combat-state, dismount, and damage gates.
	if victim == s.player {
		game.Act(nil, false, s.player, victim, nil, nil,
			"You hit yourself...OUCH!.", "", game.ToChar)
		game.Act(s.manager.world, false, s.player, victim, nil, nil,
			"$n hits $mself, and says OUCH!", "", game.ToRoom)
		return nil
	}
	if s.manager.world.IsCharmedI(s.player) &&
		strings.EqualFold(s.player.GetFollowing(), victim.GetName()) {
		game.Act(nil, false, s.player, victim, nil, nil,
			"$N is just such a good friend, you simply can't hit $M.", "", game.ToChar)
		return nil
	}

	// C only starts a hit while standing and when the requested victim is not
	// already FIGHTING(ch). It has no separate "already fighting" message.
	if s.player.GetPosition() != combat.PosStanding ||
		strings.EqualFold(s.player.GetFighting(), victim.GetName()) {
		s.Send("You do the best you can!\r\n")
		return nil
	}

	// C dismounts only on the branch that is about to call hit().
	if s.player.IsMounted() {
		s.manager.world.ExecDismount(s.player, "")
	}

	// C do_hit calls WAIT_STATE(ch, PULSE_VIOLENCE+2) after hit() returns,
	// unconditionally on this branch — even when damage()'s peaceful/newbie/
	// shopkeeper gates block the swing inside hit() (act.offensive.c:126-127).
	// Set the wait before those gates so every blocked swing still costs the
	// attacker the round, exactly like C.
	s.player.SetWaitState(3) // C: WAIT_STATE(ch, PULSE_VIOLENCE+2)

	// damage()'s protection block (fight.c:1318-1368): peaceful rooms, the
	// level-10 PK protections and shopkeepers, with C's refusal bytes. The
	// shared gate is the same one every other damage seam asks (DP-1327).
	var victimCombatant combat.Combatant
	if tgt.Player != nil {
		victimCombatant = tgt.Player
	} else if tgt.Mob != nil {
		victimCombatant = tgt.Mob
	}
	if victimCombatant != nil && s.manager.world.DamageRefused(s.player, victimCombatant) {
		return nil
	}

	if tgt.Mob != nil {
		mob := tgt.Mob
		err := s.manager.combatEngine.StartCombat(s.player, mob)
		if err != nil {
			s.Send(err.Error())
			return nil
		}
		s.markDirty(VarFighting)
		if err := s.manager.combatEngine.PerformInitialAttack(s.player, mob); err != nil {
			return err
		}
		return nil
	}

	if tgt.Player != nil {
		p := tgt.Player
		err := s.manager.combatEngine.StartCombat(s.player, p)
		if err != nil {
			s.Send(err.Error())
			return nil
		}
		s.markDirty(VarFighting)
		if err := s.manager.combatEngine.PerformInitialAttack(s.player, p); err != nil {
			return err
		}
		return nil
	}

	s.Send("They don't seem to be here.\r\n")
	return nil
}

// cmdParry implements do_parry from src/new_cmds.c:2340-2389. The command
// argument is intentionally ignored: C checks the caller's current FIGHTING
// pointer, not the text supplied after "parry".
func cmdParry(s *Session, _ []string) error {
	if s.player.GetSkill(game.SkillParry) == 0 {
		s.Send("You're not good enough at swordplay to parry!\r\n")
		return nil
	}

	if s.player.GetFighting() == "" {
		s.Send("But you aren't fighting anyone!\r\n")
		return nil
	}

	target, ok := s.manager.world.ResolveFightingTarget(s.player)
	if !ok || target.Combatant == nil {
		s.Send("But you aren't fighting anyone!\r\n")
		return nil
	}
	victim := combatTargetActor(target)
	if victim == nil || target.Combatant.GetFighting() != s.player.GetName() {
		s.Send("But noone's attacking you!\r\n")
		return nil
	}

	if s.player.Equipment == nil {
		s.Send("Parry with what? You're unarmed!\r\n")
		return nil
	}
	if weapon, wielded := s.player.Equipment.GetItemInSlot(game.SlotWield); !wielded || weapon == nil {
		s.Send("Parry with what? You're unarmed!\r\n")
		return nil
	}

	// C draws number(1, 101) before choosing the manual skill probability.
	percent := dprng.Number(1, 101)
	if percent > s.player.GetSkill(game.SkillParry) {
		s.Send("With a dazzling show of swordplay, you attempt to parry...but are outmaneuvered!\r\n")
		game.ImproveSkill(s.player, game.SkillParry)
		s.player.SetWaitState(3) // C: WAIT_STATE(ch, PULSE_VIOLENCE * 3)
		return nil
	}

	s.Send("With a dazzling show of swordplay, you move into defensive position...\r\n")
	game.Act(s.manager.world, true, s.player, victim, nil, nil,
		"$n displays a dazzling show of swordplay, fending off $N's every blow!", "", game.ToRoom)
	game.Act(nil, true, s.player, victim, nil, nil,
		"$n displays a dazzling show of swordplay, fending off your every blow!", "", game.ToVict)
	s.manager.combatEngine.MarkParried(victim.GetName(), "parry")
	s.player.SetWaitState(2) // C: WAIT_STATE(ch, PULSE_VIOLENCE * 2)
	return nil
}

func combatTargetActor(target game.CharTarget) game.Actor {
	if target.Player != nil {
		return target.Player
	}
	if target.Mob != nil {
		return target.Mob
	}
	return nil
}

// cmdFlee attempts to flee from combat.
// Port of do_flee() from src/fight.c: loops up to 6 random directions,
// checks each exit is open and the destination isn't a DEATH room.
func cmdFlee(s *Session) error {
	if s.player.GetPosition() < combat.PosFighting {
		s.Send("Get on your feet first!")
		return nil
	}
	if (s.player.GetClass() == game.ClassThief || s.player.GetClass() == game.ClassAssassin) && s.player.GetWaitState() > 1 {
		s.Send("You attempt to flee but cannot!")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Capture opponent info for XP penalty before any move changes state.
	var xpLoss int
	if opponent, ok := s.manager.combatEngine.GetCombatTarget(s.player.Name); ok {
		loss := opponent.GetMaxHP() - opponent.GetHP()
		if loss < 0 {
			loss = 0
		}
		xpLoss = loss * opponent.GetLevel()
	}
	level := s.player.GetLevel()
	if level > 10 {
		xpLoss += int(500 * (float64(level) / 2.6))
	}

	// C makes up to six independent number(0, NUM_OF_DIRS-1) attempts. Sampling
	// a shuffled list consumes a different number and shape of shared RNG draws.
	allDirs := []string{"north", "east", "south", "west", "up", "down"}

	newRoomVNum := -1
	for range 6 {
		// #nosec G404 — game RNG, not cryptographic
		dir := allDirs[dprng.Number(0, len(allDirs)-1)]
		exit, hasExit := room.Exits[dir]
		if !hasExit || exit.ToRoom == -1 {
			continue
		}
		if exit.ExitInfo&parser.ExitClosed != 0 {
			continue
		}
		// Skip DEATH rooms (ROOM_DEATH = bit 1 in CircleMUD room flags).
		if dest, ok := s.manager.world.GetRoom(exit.ToRoom); ok && dest.HasFlag(1) {
			continue
		}
		leaveMsg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "flee",
				From: s.player.Name,
				Text: fmt.Sprintf("%s panics, and attempts to flee!", s.player.Name),
			},
		})
		if err != nil {
			slog.Error("json.Marshal error", "error", err)
			return nil
		}
		s.manager.BroadcastToRoom(s.player.GetRoom(), leaveMsg, s.player.Name)

		if !s.manager.world.DoFleeMove(s.player, dir) {
			broadcastToRoom(s, fmt.Sprintf("%s tries to flee, but can't!", s.player.Name))
			return nil
		}
		newRoomVNum = s.player.GetRoom()
		break
	}

	if newRoomVNum == -1 {
		s.Send("PANIC!  You couldn't escape!")
		return nil
	}

	s.manager.combatEngine.StopCombat(s.player.Name)

	// Apply XP loss to all levels; level > 10 already included extra above.
	// LoseExp caps at max_exp_loss and returns the actual amount subtracted.
	if xpLoss > 0 {
		s.player.LoseExp(xpLoss)
	}

	s.markDirty(VarFighting, VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems)

	s.Send("You flee head over heels.")
	return nil
}

// cmdRetreat implements do_retreat from src/act.offensive.c:1001-1075.
// `escape` is the Ninja spelling; every other class uses the same handler but
// the retreat skill and messages. Its movement leg deliberately uses the
// canonical do_simple_move path without stopping combat, matching C.
func cmdRetreat(s *Session) error {
	capmsg, lowmsg := "Retreat", "retreat"
	skill := game.SkillRetreat
	if s.player.GetClass() == game.ClassNinja {
		capmsg, lowmsg, skill = "Escape", "escape", game.SkillEscape
	}

	if s.player.GetPosition() < combat.PosFighting {
		s.Send("Get on your feet first!\r\n")
		return nil
	}
	if s.player.GetSkill(game.SkillEscape) == 0 && s.player.GetSkill(game.SkillRetreat) == 0 {
		s.Send("Huh?\r\n")
		return nil
	}
	if s.player.GetFighting() == "" {
		s.Send(fmt.Sprintf("%s from what? You aren't fighting!\n\r", capmsg))
		return nil
	}

	// C draws number(1, 101) before selecting the class-specific probability.
	// #nosec G404 — game RNG, not cryptographic
	if dprng.Number(1, 101) > s.player.GetSkill(skill) {
		game.ImproveSkill(s.player, skill)
		s.Send(fmt.Sprintf("You try to %s but get cornered in the process!\r\n", lowmsg))
		s.player.SetWaitState(3) // C: WAIT_STATE(ch, PULSE_VIOLENCE+2)
		return nil
	}

	for range 6 {
		// C uses six independent number(0, NUM_OF_DIRS-1) draws, including
		// repeats; do not replace this with a shuffled direction list.
		// #nosec G404 — game RNG, not cryptographic
		attempt := dprng.Number(0, len(retreatDirections)-1)
		direction := retreatDirections[attempt]
		room, ok := s.manager.world.GetRoom(s.player.GetRoom())
		if !ok {
			return fmt.Errorf("invalid room")
		}
		exit, ok := room.Exits[direction]
		if !ok || exit.ToRoom == -1 {
			continue
		}
		destination, ok := s.manager.world.GetRoom(exit.ToRoom)
		if !ok || destination.HasFlag(1) { // C: ROOM_DEATH
			continue
		}

		game.Act(s.manager.world, true, s.player, nil, nil, nil,
			fmt.Sprintf("$n realizes it's a losing cause and gracefully attempts to %s.", lowmsg),
			"", game.ToRoom)
		if s.manager.world.DoFleeMove(s.player, direction) {
			s.Send(fmt.Sprintf("You make a hasty %s.\r\n", lowmsg))
		} else {
			game.Act(s.manager.world, true, s.player, nil, nil, nil,
				fmt.Sprintf("$n is cornered and fails to %s!", lowmsg), "", game.ToRoom)
		}
		return nil
	}

	s.Send(fmt.Sprintf("You are cornered and fail to %s!\r\n", lowmsg))
	game.Act(s.manager.world, true, s.player, nil, nil, nil,
		fmt.Sprintf("$n is cornered and fails to %s!", lowmsg), "", game.ToRoom)
	return nil
}

var retreatDirections = []string{"north", "east", "south", "west", "up", "down"}
