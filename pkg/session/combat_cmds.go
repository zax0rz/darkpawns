package session

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
)

// cmdHit initiates combat with a target.
func cmdHit(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Hit whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])

	// Check if already fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.sendText("You're already fighting!")
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
				s.sendText(err.Error())
				return nil
			}

			// Notify player
			s.sendText(fmt.Sprintf("You attack %s!", mob.GetShortDesc()))

			// Notify room
			msg, _ := json.Marshal(ServerMessage{
				Type: MsgEvent,
				Data: EventData{
					Type: "combat",
					From: s.player.Name,
					Text: fmt.Sprintf("%s attacks %s!", s.player.Name, mob.GetShortDesc()),
				},
			})
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
				s.sendText(err.Error())
				return nil
			}

			// Notify both players
			s.sendText(fmt.Sprintf("You attack %s!", p.Name))

			// Notify target
			if targetSession, ok := s.manager.GetSession(p.Name); ok {
				targetSession.sendText(fmt.Sprintf("%s attacks you!", s.player.Name))
			}

			// Notify room
			msg, _ := json.Marshal(ServerMessage{
				Type: MsgEvent,
				Data: EventData{
					Type: "combat",
					From: s.player.Name,
					Text: fmt.Sprintf("%s attacks %s!", s.player.Name, p.Name),
				},
			})
			s.manager.BroadcastToRoom(room.VNum, msg, s.player.Name)

			return nil
		}
	}

	s.sendText("They aren't here.")
	return nil
}

// cmdFlee attempts to flee from combat.
// Implements do_flee() from act.offensive.c lines 360-420
func cmdFlee(s *Session) error {
	// Check if in combat
	if !s.manager.combatEngine.IsFighting(s.player.Name) {
		s.sendText("You're not fighting anyone!")
		return nil
	}

	// Get current room
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Get available exits
	if len(room.Exits) == 0 {
		s.sendText("There's nowhere to flee!")
		return nil
	}

	// Pick random exit
	var directions []string
	for dir := range room.Exits {
		directions = append(directions, dir)
	}

	// 50% chance to flee successfully
	if rand.Intn(100) > 50 {
		s.sendText("You attempt to flee but fail!")
		return nil
	}

	// Calculate XP loss before stopping combat
	// Source: act.offensive.c do_flee() lines 367-371
	var xpLoss int
	if opponent, ok := s.manager.combatEngine.GetCombatTarget(s.player.Name); ok {
		loss := opponent.GetMaxHP() - opponent.GetHP()
		loss *= opponent.GetLevel()
		xpLoss = loss
	}

	// Apply XP loss for players level > 10
	// Source: act.offensive.c do_flee() lines 398-401
	level := s.player.GetLevel()
	if level > 10 {
		// 500 * (level / 2.6) — original uses float division, cast to int
		// Source: act.offensive.c do_flee() line 400
		xpLoss += int(500 * (float64(level) / 2.6))
		s.player.LoseExp(xpLoss)
		if xpLoss > 0 {
			s.sendText(fmt.Sprintf("You lose %d experience points for fleeing.", xpLoss))
		}
	}

	// Flee successfully
	s.manager.combatEngine.StopCombat(s.player.Name)

	// Pick random direction
	direction := directions[rand.Intn(len(directions))]

	// Move player
	oldRoom := s.player.GetRoom()
	newRoom, err := s.manager.world.MovePlayer(s.player, direction)
	if err != nil {
		s.sendText("You panic and can't find an exit!")
		return nil
	}

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

	s.sendText("You flee head over heels.")

	// Send new room state
	return cmdLook(s, nil)
}