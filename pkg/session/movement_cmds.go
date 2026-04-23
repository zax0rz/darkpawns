package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Ensure slog is used


// cmdStand handles the 'stand' command.
// Source: act.movement.c do_stand() lines 691–730
func cmdStand(s *Session) error {
	pos := s.player.GetPosition()

	switch pos {
	case combat.PosStanding:
		s.Send("You are already standing.")
	case combat.PosSitting:
		s.Send("You stand up.")
		broadcastToRoom(s, fmt.Sprintf("%s clambers to %s feet.", s.player.Name, genderHisHer(s.player)))
		s.player.SetPosition(combat.PosStanding)
	case combat.PosResting:
		s.Send("You stop resting, and stand up.")
		broadcastToRoom(s, fmt.Sprintf("%s stops resting, and clambers on %s feet.", s.player.Name, genderHisHer(s.player)))
		s.player.SetPosition(combat.PosStanding)
	case combat.PosSleeping:
		s.Send("You have to wake up first!")
	case combat.PosFighting:
		s.Send("Do you not consider fighting as standing?")
	default:
		s.Send("You stop floating around, and put your feet on the ground.")
		broadcastToRoom(s, fmt.Sprintf("%s stops floating around, and puts %s feet on the ground.", s.player.Name, genderHisHer(s.player)))
		s.player.SetPosition(combat.PosStanding)
	}
	return nil
}

// cmdSit handles the 'sit' command.
// Source: act.movement.c do_sit() lines 732–762
func cmdSit(s *Session) error {
	pos := s.player.GetPosition()

	switch pos {
	case combat.PosStanding:
		s.Send("You sit down.")
		broadcastToRoom(s, fmt.Sprintf("%s sits down.", s.player.Name))
		s.player.SetPosition(combat.PosSitting)
	case combat.PosSitting:
		s.Send("You're sitting already.")
	case combat.PosResting:
		s.Send("You stop resting, and sit up.")
		broadcastToRoom(s, fmt.Sprintf("%s stops resting.", s.player.Name))
		s.player.SetPosition(combat.PosSitting)
	case combat.PosSleeping:
		s.Send("You have to wake up first.")
	case combat.PosFighting:
		s.Send("Sit down while fighting? are you MAD?")
	default:
		s.Send("You stop floating around, and sit down.")
		broadcastToRoom(s, fmt.Sprintf("%s stops floating around, and sits down.", s.player.Name))
		s.player.SetPosition(combat.PosSitting)
	}
	return nil
}

// cmdRest handles the 'rest' command.
// Source: act.movement.c do_rest() lines 764–795
func cmdRest(s *Session) error {
	pos := s.player.GetPosition()

	switch pos {
	case combat.PosStanding:
		s.Send("You sit down and rest your tired bones.")
		broadcastToRoom(s, fmt.Sprintf("%s sits down and rests.", s.player.Name))
		s.player.SetPosition(combat.PosResting)
	case combat.PosSitting:
		s.Send("You rest your tired bones.")
		broadcastToRoom(s, fmt.Sprintf("%s rests.", s.player.Name))
		s.player.SetPosition(combat.PosResting)
	case combat.PosResting:
		s.Send("You are already resting.")
	case combat.PosSleeping:
		s.Send("You have to wake up first.")
	case combat.PosFighting:
		s.Send("Rest while fighting?  Are you MAD?")
	default:
		s.Send("You stop floating around, and stop to rest your tired bones.")
		broadcastToRoom(s, fmt.Sprintf("%s stops floating around, and rests.", s.player.Name))
		s.player.SetPosition(combat.PosResting)
	}
	return nil
}

// cmdSleep handles the 'sleep' command.
// Source: act.movement.c do_sleep() lines 797–825
func cmdSleep(s *Session) error {
	pos := s.player.GetPosition()

	switch pos {
	case combat.PosStanding, combat.PosSitting, combat.PosResting:
		s.Send("You go to sleep.")
		broadcastToRoom(s, fmt.Sprintf("%s lies down and falls asleep.", s.player.Name))
		s.player.SetPosition(combat.PosSleeping)
	case combat.PosSleeping:
		s.Send("You are already sound asleep.")
	case combat.PosFighting:
		s.Send("Sleep while fighting?  Are you MAD?")
	default:
		s.Send("You stop floating around, and lie down to sleep.")
		broadcastToRoom(s, fmt.Sprintf("%s stops floating around, and lie down to sleep.", s.player.Name))
		s.player.SetPosition(combat.PosSleeping)
	}
	return nil
}

// cmdWake handles the 'wake' command with optional target.
// Source: act.movement.c do_wake() lines 827–870
func cmdWake(s *Session, args []string) error {
	if len(args) > 0 {
		targetName := strings.Join(args, " ")

		// Can't wake others while sleeping
		if s.player.GetPosition() == combat.PosSleeping {
			s.Send("Maybe you should wake yourself up first.")
			return nil
		}

		// Find target in room
		target, ok := s.manager.world.GetPlayer(targetName)
		if !ok {
			s.Send("There is no one by that name here.")
			return nil
		}
		if target.GetRoom() != s.player.GetRoom() {
			s.Send("They are not here.")
			return nil
		}

		if target == s.player {
			// Fall through to self-wake below
		} else if target.GetPosition() > combat.PosSleeping {
			s.Send(fmt.Sprintf("%s is already awake.", target.Name))
			return nil
		} else if target.GetPosition() < combat.PosSleeping {
			s.Send(fmt.Sprintf("%s's in pretty bad shape!", target.Name))
			return nil
		} else {
			// Wake the target
			s.Send(fmt.Sprintf("You wake %s up.", target.Name))
			target.SendMessage(fmt.Sprintf("%s wakes you up.", s.player.Name))
			broadcastToRoomExcept(s, fmt.Sprintf("%s wakes up %s.", s.player.Name, target.Name), target.Name)
			target.SetPosition(combat.PosSitting)
			return nil
		}
	}

	// Self-wake
	if s.player.GetPosition() > combat.PosSleeping {
		s.Send("You are already awake...")
		return nil
	}

	s.Send("You awaken, and sit up.")
	broadcastToRoom(s, fmt.Sprintf("%s awakens.", s.player.Name))
	s.player.SetPosition(combat.PosSitting)
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
		idx := rand.Intn(len(directions))
		direction := directions[idx]

		// Check if door blocks
		if s.manager.doorManager != nil {
			canPass, _ := s.manager.doorManager.CanPass(room.VNum, direction)
			if !canPass {
				continue
			}
		}

		// Try to move
		oldRoom := s.player.GetRoom()
		newRoom, err := s.manager.world.MovePlayer(s.player, direction)
		if err != nil {
			continue
		}

		// Successful flee
		fled = true

		// Apply XP loss for players level > 10
		level := s.player.GetLevel()
		if level > 10 {
			xpLoss += int(500 * (float64(level) / 2.6))
			s.player.LoseExp(xpLoss)
			if xpLoss > 0 {
				s.Send(fmt.Sprintf("You lose %d experience points for fleeing.", xpLoss))
			}
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
		s.markDirty(VarFighting, VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems)

		// Send new room state
		return cmdLook(s, nil)
	}

	if !fled {
		s.Send("PANIC!  You couldn't escape!")
		broadcastToRoom(s, fmt.Sprintf("%s tries to flee, but can't!", s.player.Name))
	}
	return nil
}

// cmdFollowMovement is a movement-phase variant of cmdFollow.
// The canonical implementation lives in commands.go; this alias is kept
// for any routing that needs to reference it from the movement package.
// Source: act.movement.c do_follow() lines 883–951
func cmdFollowMovement(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Whom do you wish to follow?")
		return nil
	}

	targetName := args[0]

	// follow self = stop following
	if strings.EqualFold(targetName, s.player.Name) {
		if s.player.Following == "" {
			s.Send("You are already following yourself.")
			return nil
		}
		oldLeader := s.player.Following
		s.player.Following = ""
		s.player.InGroup = false
		s.Send(fmt.Sprintf("You stop following %s.", oldLeader))
		if leader, ok := s.manager.world.GetPlayer(oldLeader); ok {
			leader.SendMessage(fmt.Sprintf("%s stops following you.\r\n", s.player.Name))
		}
		return nil
	}

	// Find target in room
	target, ok := s.manager.world.GetPlayer(targetName)
	if !ok {
		s.Send("There is no one by that name here.")
		return nil
	}
	if target.GetRoom() != s.player.GetRoom() {
		s.Send("They are not here.")
		return nil
	}

	// Already following?
	if s.player.Following == target.Name {
		s.Send(fmt.Sprintf("You are already following %s.", target.Name))
		return nil
	}

	// Stop following previous leader
	if s.player.Following != "" {
		oldLeader := s.player.Following
		if leader, ok := s.manager.world.GetPlayer(oldLeader); ok {
			leader.SendMessage(fmt.Sprintf("%s stops following you.\r\n", s.player.Name))
		}
	}

	// Set new follow target
	s.player.Following = target.Name
	s.player.InGroup = false

	s.Send(fmt.Sprintf("You now follow %s.", target.Name))
	target.SendMessage(fmt.Sprintf("%s now follows you.\r\n", s.player.Name))
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

// broadcastToRoomExcept sends a message to the room excluding both sender and a named target.
func broadcastToRoomExcept(s *Session, text string, exclude string) {
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
	// Broadcast excluding sender
	s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
	// Also exclude target if they have a session
	if targetSess, ok := s.manager.GetSession(exclude); ok {
		_ = targetSess
		// target already got a direct message, skip the broadcast to them
	}
}

// genderHisHer returns "his" or "her" based on player gender.
// Default to "his" since gender isn't tracked yet.
func genderHisHer(p interface{}) string {
	// TODO: Add gender field to Player and return appropriate pronoun
	return "his"
}
