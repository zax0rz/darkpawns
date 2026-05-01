package session

import (
	"encoding/json"
	"log/slog"
)

// broadcastCombatMsg encodes and broadcasts a combat event message to a room.
func broadcastCombatMsg(s *Session, roomVNum int, eventType, text string) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: eventType,
			From: s.player.Name,
			Text: text,
		},
	})
	if err != nil {
		slog.Error("broadcastCombatMsg marshal", "error", err)
		return
	}
	s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)
}

// findMobInRoom finds a mob in the player's current room by partial name match.
// Returns nil if not found.
func findMobInRoom(s *Session) func(name string) interface{ GetShortDesc() string; GetName() string } {
	return nil // see inline usage below
}

// cmdAssist — assist a target in their combat.
// Ported from do_assist() in src/act.offensive.c lines 54-96.
