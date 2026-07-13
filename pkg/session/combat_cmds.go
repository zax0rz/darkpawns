package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// cmdHit initiates combat with a target.
// Combat is self-rate-limited by the 2s engine tick.
// StartCombat enrolls the player; PerformRound fires autonomously.
func cmdKill(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Kill who?")
		return nil
	}

	// Immortal instakill (C: src/act.offensive.c do_kill())
	// C gates this to GET_LEVEL(ch) >= LVL_IMPL-1 (level 39+), NOT LVL_IMMORT
	// (31+). Every immortal used to have implementor-grade instakill power.
	// (DP-1041)
	if s.player.GetLevel() >= LVL_IMPL-1 {
		// Resolve via the canonical in-room resolver (DP-907) so `kill X`
		// agrees with consider/kick/... on what "X" is.
		tgt, found := s.manager.world.ResolveCharInRoom(s.player, strings.Join(args, " "))
		if !found {
			s.Send("They aren't here.")
			return nil
		}
		// C: else if (GET_LEVEL(vict) == GET_LEVEL(ch)) — blocks equal-level
		// kills regardless of PC/NPC (act.offensive.c:149).
		victLevel := 0
		switch {
		case tgt.Player != nil:
			victLevel = tgt.Player.GetLevel()
		case tgt.Mob != nil:
			victLevel = tgt.Mob.GetLevel()
		}
		if victLevel == s.player.GetLevel() {
			s.Send("No can do, buddy..\r\n")
			return nil
		}
		switch {
		case tgt.Mob != nil:
			s.manager.world.Instakill(tgt.Mob, s.player, 0)
			s.Send(fmt.Sprintf("You chop %s to pieces! Ah! The blood!", tgt.Mob.GetShortDesc()))
		case tgt.Player != nil:
			// C: act("$N chops you to pieces!", ...) sent to the victim
			// before raw_kill (act.offensive.c:152-154).
			tgt.Player.SendMessage(fmt.Sprintf("%s chops you to pieces!\r\n", s.player.GetName()))
			s.manager.world.Instakill(tgt.Player, s.player, 0)
			s.Send(fmt.Sprintf("You chop %s to pieces! Ah! The blood!", tgt.Player.Name))
		default:
			s.Send("They aren't here.")
		}
		return nil
	}

	// Mortals delegate to hit
	return cmdHit(s, args)
}

func cmdHit(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Hit whom?")
		return nil
	}

	// Auto-dismount before attacking (C: do_hit calls do_dismount if mounted).
	if s.player.IsMounted() {
		s.manager.world.ExecDismount(s.player, "")
	}

	// Check if already fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're already fighting!")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Resolve target via the canonical in-room resolver (DP-907): keyword-list
	// abbreviation matching, ordinals, self/me, visibility — identical to
	// consider/kick/backstab/...
	tgt, found := s.manager.world.ResolveCharInRoom(s.player, strings.Join(args, " "))
	if !found {
		s.Send("They aren't here.")
		return nil
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
			s.Send(fmt.Sprintf("You are not experienced enough to attack %s!\r\n", tgt.Player.Name))
			return nil
		}
		if victimLevel <= 10 && !victimIsOutlaw {
			s.Send(fmt.Sprintf("Ancient forces protect %s from your wrath!\r\n", tgt.Player.Name))
			return nil
		}
	}

	if tgt.Mob != nil {
		mob := tgt.Mob
		// Start combat
		err := s.manager.combatEngine.StartCombat(s.player, mob)
		if err != nil {
			s.Send(err.Error())
			return nil
		}

		// Notify player
		s.Send(fmt.Sprintf("You attack %s!", mob.GetShortDesc()))
		s.markDirty(VarFighting)

		// Notify room
		msg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "combat",
				From: s.player.Name,
				Text: fmt.Sprintf("%s attacks %s!", s.player.Name, mob.GetShortDesc()),
			},
		})
		if err != nil {
			slog.Error("json.Marshal error", "error", err)
			return nil
		}
		s.manager.BroadcastToRoom(room.VNum, msg, s.player.Name)
		return nil
	}

	if tgt.Player != nil {
		p := tgt.Player
		// Start combat with player
		err := s.manager.combatEngine.StartCombat(s.player, p)
		if err != nil {
			s.Send(err.Error())
			return nil
		}

		// Notify both players
		s.Send(fmt.Sprintf("You attack %s!", p.Name))
		s.markDirty(VarFighting)

		// Notify target
		if targetSession, ok := s.manager.GetSession(p.Name); ok {
			targetSession.Send(fmt.Sprintf("%s attacks you!", s.player.Name))
		}

		// Notify room
		msg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "combat",
				From: s.player.Name,
				Text: fmt.Sprintf("%s attacks %s!", s.player.Name, p.Name),
			},
		})
		if err != nil {
			slog.Error("json.Marshal error", "error", err)
			return nil
		}
		s.manager.BroadcastToRoom(room.VNum, msg, s.player.Name)
		return nil
	}

	s.Send("They aren't here.")
	return nil
}

// cmdFlee attempts to flee from combat.
// Port of do_flee() from src/fight.c: loops up to 6 random directions,
// checks each exit is open and the destination isn't a DEATH room.
func cmdFlee(s *Session) error {
	if !s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're not fighting anyone!")
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

	// Try up to 6 shuffled directions (C: do_flee loops up to 6 random dirs).
	// #nosec G404 — game RNG, not cryptographic
	allDirs := []string{"north", "east", "south", "west", "up", "down"}
	for i := len(allDirs) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		allDirs[i], allDirs[j] = allDirs[j], allDirs[i]
	}

	oldRoom := s.player.GetRoom()
	newRoomVNum := -1
	for _, dir := range allDirs {
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
		nr, err := s.manager.world.MovePlayer(s.player, dir)
		if err != nil {
			continue
		}
		newRoomVNum = nr.VNum
		break
	}

	if newRoomVNum == -1 {
		s.Send("You panic but can't find a way out!")
		return nil
	}

	s.manager.combatEngine.StopCombat(s.player.Name)

	// Apply XP loss to all levels; level > 10 already included extra above.
	// LoseExp caps at max_exp_loss and returns the actual amount subtracted.
	if xpLoss > 0 {
		actualLoss := s.player.LoseExp(xpLoss)
		s.Send(fmt.Sprintf("You lose %d experience points for fleeing.", actualLoss))
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
	s.manager.BroadcastToRoom(oldRoom, leaveMsg, s.player.Name)

	enterMsg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "enter",
			From: s.player.Name,
			Text: fmt.Sprintf("%s has arrived, fleeing from combat!", s.player.Name),
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
	s.manager.BroadcastToRoom(newRoomVNum, enterMsg, s.player.Name)

	s.Send("You flee head over heels.")
	s.markDirty(VarFighting, VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems)

	return cmdMovementLook(s)
}
