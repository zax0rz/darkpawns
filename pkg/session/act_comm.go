package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// ---------------------------------------------------------------------------
// Communication commands (ported from act.comm.c)
// ---------------------------------------------------------------------------

// cmdRaceSay speaks in the player's racial tongue.
// Format: "You say in $race '$msg'"
func cmdRaceSay(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("What do you want to say?")
		return nil
	}

	msg := strings.Join(args, " ")
	raceName := game.RaceNames[s.player.Race]
	if raceName == "" {
		raceName = "tongue"
	}

	s.Send(fmt.Sprintf("You say in %s '%s'", raceName, msg))

	// Broadcast to others in room
	roomVNum := s.player.GetRoomVNum()
	broadcastToRoomText(s, roomVNum,
		fmt.Sprintf("%s says in %s '%s'", s.player.Name, raceName, msg))

	return nil
}

// cmdQcomm handles question communication (question asked to all).
func cmdQcomm(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("What is your question?")
		return nil
	}

	msg := strings.Join(args, " ")
	formatted := fmt.Sprintf("%s asks '%s'", s.player.Name, msg)

	s.Send(fmt.Sprintf("You ask '%s'", msg))

	// Broadcast to all online players
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil || sess == s {
			continue
		}
		sess.Send(formatted)
	}
	s.manager.mu.RUnlock()

	return nil
}

// ---------------------------------------------------------------------------
// Whisper
// ---------------------------------------------------------------------------

// cmdWhisper whispers a private message to a player in the same room.
func cmdWhisper(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Whisper whom what?")
		return nil
	}

	targetName := args[0]
	message := strings.Join(args[1:], " ")

	// Word filter + spam check
	filtered, block := filterCommMessage(s, message)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	message = filtered

	roomVNum := s.player.GetRoomVNum()

	// Find target in the same room
	var targetSess *Session
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil {
			continue
		}
		if strings.EqualFold(sess.player.Name, targetName) && sess.player.GetRoomVNum() == roomVNum {
			targetSess = sess
			break
		}
	}
	s.manager.mu.RUnlock()

	if targetSess == nil {
		s.Send("No one by that name is here.")
		return nil
	}

	// Send to victim
	targetSess.Send(fmt.Sprintf("\x1B[1;33m%s whispers, '%s'\033[0m\r\n", s.player.Name, message))

	// Confirm to sender
	s.Send(fmt.Sprintf("You whisper to %s, '%s'", targetSess.player.Name, message))

	// Broadcast to rest of room that whisper occurred
	roomText := fmt.Sprintf("%s whispers something to %s.\r\n", s.player.Name, targetSess.player.Name)
	broadcastToRoomText(s, roomVNum, roomText)

	return nil
}

// ---------------------------------------------------------------------------
// Ask
// ---------------------------------------------------------------------------

// cmdAsk asks a question to a player in the same room.
func cmdAsk(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Ask whom what?")
		return nil
	}

	targetName := args[0]
	message := strings.Join(args[1:], " ")
	roomVNum := s.player.GetRoomVNum()

	var targetSess *Session
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil {
			continue
		}
		if strings.EqualFold(sess.player.Name, targetName) && sess.player.GetRoomVNum() == roomVNum {
			targetSess = sess
			break
		}
	}
	s.manager.mu.RUnlock()

	if targetSess == nil {
		s.Send("No one by that name is here.")
		return nil
	}

	targetSess.Send(fmt.Sprintf("\x1B[1;36m%s asks, '%s'\033[0m\r\n", s.player.Name, message))
	s.Send(fmt.Sprintf("You ask %s, '%s'", targetSess.player.Name, message))

	roomText := fmt.Sprintf("%s asks %s something.\r\n", s.player.Name, targetSess.player.Name)
	broadcastToRoomText(s, roomVNum, roomText)

	return nil
}
