//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package session

import (
	"encoding/json"
	"log/slog"
	"strings"
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
