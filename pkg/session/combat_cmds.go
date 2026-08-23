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
	if len(args) == 0 {
		s.Send("Hit who?\r\n")
		return nil
	}

	if _, ok := s.manager.world.GetRoom(s.player.GetRoom()); !ok {
		return fmt.Errorf("invalid room")
	}

	// Resolve target via the canonical in-room resolver (DP-907): keyword-list
	// abbreviation matching, ordinals, self/me, visibility — identical to
	// consider/kick/backstab/...
	tgt, found := s.manager.world.ResolveCharInRoom(s.player, strings.Join(args, " "))
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

	// --- Combat-entry gates (fight.c:1336-1357, DP-1045 partial) ---
	// These mirror C's damage() pre-swing checks at the command layer — the
	// faithful, non-spammy home for player-initiated melee in Go.

	// Resolve victim facts uniformly across mob/player targets.
	victimFighting := ""
	victimIsOutlaw := false
	victimIsPlayer := tgt.Player != nil
	victimLevel := 0
	switch {
	case tgt.Player != nil:
		victimFighting = tgt.Player.GetFighting()
		victimIsOutlaw = tgt.Player.GetFlags()&(1<<uint(game.PlrOutlaw)) != 0
		victimLevel = tgt.Player.GetLevel()
	case tgt.Mob != nil:
		victimFighting = tgt.Mob.GetFighting()
		victimLevel = tgt.Mob.GetLevel()
	}

	// Peaceful room — fight.c:1336. Outlaws and already-engaged retaliation
	// are exempt. Applies to both mob and player targets (mobs are never
	// outlaws, so they're always protected here).
	if !victimIsOutlaw && victimFighting != s.player.Name &&
		s.manager.world.RoomHasFlag(s.player.GetRoom(), "peaceful") {
		s.Send("This room just has such a peaceful, easy feeling...\r\n")
		return nil
	}

	// Low-level PC protection — fight.c:1344 (PC vs PC only). The attacker is
	// always a player here (cmdHit is a player command), so we only need the
	// victim-is-PC test for the IS_NPC(victim) half of the C condition.
	if victimIsPlayer {
		if s.player.GetLevel() <= 10 {
			game.Act(nil, false, s.player, victim, nil, nil,
				"You are not experienced enough to attack $N!", "", game.ToChar)
			return nil
		}
		if victimLevel <= 10 && !victimIsOutlaw {
			game.Act(nil, false, s.player, victim, nil, nil,
				"Ancient forces protect $N from your wrath!", "", game.ToChar)
			return nil
		}
	}

	// C's damage() protects shopkeepers after peaceful/newbie gates and stops
	// any existing combat involving either participant.
	if tgt.Mob != nil && isShopkeeper(s.manager.world, tgt.Mob) {
		s.Send("Ha ha... Don't think so.\r\n")
		s.manager.combatEngine.StopCombat(s.player.GetName())
		s.manager.combatEngine.StopCombat(tgt.Mob.GetName())
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
		s.player.SetWaitState(3) // C: WAIT_STATE(ch, PULSE_VIOLENCE+2)
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
		s.player.SetWaitState(3) // C: WAIT_STATE(ch, PULSE_VIOLENCE+2)
		return nil
	}

	s.Send("They don't seem to be here.\r\n")
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

func isShopkeeper(world *game.World, mob *game.MobInstance) bool {
	if world == nil || mob == nil {
		return false
	}
	manager := world.GetShopManager()
	if manager == nil {
		return false
	}
	_, ok := manager.GetShopByNPC(mob.GetVNum())
	return ok
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
