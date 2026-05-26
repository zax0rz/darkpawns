package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
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
	if s.player.GetLevel() >= LVL_IMMORT {
		targetName := strings.ToLower(args[0])
		room, ok := s.manager.world.GetRoom(s.player.GetRoom())
		if !ok {
			return fmt.Errorf("invalid room")
		}

		// Check mobs
		mobs := s.manager.world.GetMobsInRoom(room.VNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
				s.manager.world.HandleDeath(mob, s.player, 0)
				s.Send(fmt.Sprintf("You chop %s to pieces! Ah! The blood!", mob.GetShortDesc()))
				return nil
			}
		}

		// Check players
		players := s.manager.world.GetPlayersInRoom(room.VNum)
		for _, p := range players {
			if p.Name != s.player.Name && strings.Contains(strings.ToLower(p.Name), targetName) {
				s.manager.world.HandleDeath(p, s.player, 0)
				s.Send(fmt.Sprintf("You chop %s to pieces! Ah! The blood!", p.Name))
				return nil
			}
		}

		s.Send("They aren't here.")
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

	targetName := strings.ToLower(args[0])

	// Auto-dismount before attacking (C: do_hit calls do_dismount if mounted).
	if s.player.IsMounted() {
		s.manager.world.ExecDismount(s.player, "")
	}

	// Check if already fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're already fighting!")
		return nil
	}

	// Find target in room
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Check for mobs in room
	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
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
	}

	// Check for players in room
	players := s.manager.world.GetPlayersInRoom(room.VNum)
	for _, p := range players {
		if p.Name != s.player.Name && strings.Contains(strings.ToLower(p.Name), targetName) {
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
		j := rand.Intn(i + 1)
		allDirs[i], allDirs[j] = allDirs[j], allDirs[i]
	}

	oldRoom := s.player.GetRoom()
	newRoomVNum := -1
	for _, dir := range allDirs {
		exit, hasExit := room.Exits[dir]
		if !hasExit || exit.ToRoom == -1 {
			continue
		}
		if exit.DoorState != 0 { // closed or locked
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
	s.player.LoseExp(xpLoss)
	if xpLoss > 0 {
		s.Send(fmt.Sprintf("You lose %d experience points for fleeing.", xpLoss))
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

	return cmdLook(s, nil)
}

