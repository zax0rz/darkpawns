//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Ensure slog is used

func cmdStand(s *Session) error {
	s.manager.world.DoStand(s.player)
	return nil
}

func cmdSit(s *Session) error {
	s.manager.world.DoSit(s.player)
	return nil
}

func cmdRest(s *Session) error {
	s.manager.world.DoRest(s.player)
	return nil
}

func cmdSleep(s *Session) error {
	s.manager.world.DoSleep(s.player)
	return nil
}

func cmdWake(s *Session, args []string) error {
	s.manager.world.DoWake(s.player, strings.Join(args, " "))
	return nil
}

// cmdFleeMovement is a movement-phase variant of cmdFlee.
// The canonical implementation lives in combat_cmds.go; this alias is kept
// for any routing that needs to reference it from the movement package.
// Source: act.offensive.c do_flee() lines 360–420
func cmdFleeMovement(s *Session) error {
	// Must be fighting
	if !s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're not fighting anyone!")
		return nil
	}

	// Must be on feet
	if s.player.GetPosition() < combat.PosFighting {
		s.Send("Get on your feet first!")
		return nil
	}

	// Get current room
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Get available exits
	if len(room.Exits) == 0 {
		s.Send("There's nowhere to flee!")
		return nil
	}

	// Calculate XP loss before stopping combat
	// Source: act.offensive.c do_flee() lines 367–371
	var xpLoss int
	if opponent, ok := s.manager.combatEngine.GetCombatTarget(s.player.Name); ok {
		loss := opponent.GetMaxHP() - opponent.GetHP()
		loss *= opponent.GetLevel()
		xpLoss = loss
	}

	// Try up to 6 random directions
	var directions []string
	for dir := range room.Exits {
		directions = append(directions, dir)
	}

	// Broadcast panic attempt to room
	broadcastToRoom(s, fmt.Sprintf("%s panics, and attempts to flee!", s.player.Name))

	// Try random exits
	fled := false
	for i := 0; i < 6 && len(directions) > 0; i++ {
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		idx := dprng.Number(0, len(directions)-1)
		direction := directions[idx]

		// Closed exits cannot be used to flee.
		if exit, ok := room.Exits[direction]; !ok || exit.ExitInfo&parser.ExitClosed != 0 {
			continue
		}

		// Try to move
		oldRoom := s.player.GetRoom()
		newRoom, err := s.manager.world.MovePlayer(s.player, direction)
		if err != nil {
			continue
		}

		// Successful flee

		// Apply XP loss. Base loss (opponent missing HP * opponent level) is
		// computed at ALL levels. Bonus loss is gated to level > 10.
		// Source: act.offensive.c:365-369; limits.c:319 caps at max_exp_loss.
		level := s.player.GetLevel()
		if level > 10 {
			xpLoss += int(500 * (float64(level) / 2.6))
		}

		if xpLoss > 0 {
			actualLoss := s.player.LoseExp(xpLoss)
			s.Send(fmt.Sprintf("You lose %d experience points for fleeing.", actualLoss))
		}

		s.manager.combatEngine.StopCombat(s.player.Name)

		// Notify old room
		leaveMsg, _ := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "flee",
				From: s.player.Name,
				Text: fmt.Sprintf("%s panics, and attempts to flee!", s.player.Name),
			},
		})
		s.manager.BroadcastToRoom(oldRoom, leaveMsg, s.player.Name)

		// Notify new room
		enterMsg, _ := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "enter",
				From: s.player.Name,
				Text: fmt.Sprintf("%s has arrived, fleeing from combat!", s.player.Name),
			},
		})
		s.manager.BroadcastToRoom(newRoom.VNum, enterMsg, s.player.Name)

		s.Send("You flee head over heels.")
		s.markDirty(VarFighting, VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems, VarMove)

		// Send new room state
		return cmdMovementLook(s)
	}

	if !fled {
		s.Send("PANIC!  You couldn't escape!")
		broadcastToRoom(s, fmt.Sprintf("%s tries to flee, but can't!", s.player.Name))
	}
	return nil
}

// cmdSneak handles the 'sneak' command.
// This is a wrapper around the skill-based sneak in pkg/command.
// The actual skill implementation lives in pkg/game/skills.go.
// Source: act.movement.c — sneak is handled via skill system in Dark Pawns.
func cmdSneak(s *Session) error {
	// Sneak is already registered as a skill command in commands.go init()
	// via wrapSkill(command.CmdSneak). This stub exists for any direct
	// routing needs but the skill path handles it.
	s.Send("You attempt to move silently.")
	return nil
}

// broadcastToRoom sends a plain text event to all players in the room except the sender.
func broadcastToRoom(s *Session, text string) {
	if s.player == nil {
		return
	}
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "position",
			From: s.player.Name,
			Text: text,
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
}
