// Package session manages WebSocket connections and player sessions.
package session

import (
	"fmt"
	"log/slog"
	"encoding/json"
)
import "github.com/zax0rz/darkpawns/pkg/game"

func (s *Session) sendWelcome(token string) {
	roomVNum := s.player.GetRoom()
	room, ok := s.manager.world.GetRoom(roomVNum)
	if !ok || room == nil {
		roomVNum = game.MortalStartRoom
		room, ok = s.manager.world.GetRoom(roomVNum)
		if !ok || room == nil {
			slog.Error("sendWelcome: mortal start room not found", "vnum", roomVNum)
			return
		}
	}

	state := StateData{
		Player: PlayerState{
			Name:      s.player.Name,
			Health:    s.player.Health,
			MaxHealth: s.player.MaxHealth,
			Level:     s.player.Level,
			Class:     game.ClassNames[s.player.Class],
			Race:      game.RaceNames[s.player.Race],
			Str:       s.player.Stats.Str,
			Int:       s.player.Stats.Int,
			Wis:       s.player.Stats.Wis,
			Dex:       s.player.Stats.Dex,
			Con:       s.player.Stats.Con,
			Cha:       s.player.Stats.Cha,
		},
		Room: RoomState{
			VNum:        room.VNum,
			Name:        room.Name,
			Description: room.Description,
			Exits:       getExitNames(room.Exits),
			Doors:       getDoorInfo(s.manager.doorManager, room.VNum, room.Exits),
		},
		Token: token,
	}

	// Send MOTD before the room state
	motd := game.ShowMOTD(s.manager.world.WorldPath)
	if motd != "" {
		motdMsg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "motd",
				Text: motd,
			},
		})
		if err == nil {
			s.send <- motdMsg
		}
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgState,
		Data: state,
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.send <- msg
}

// sendError sends an error message to the player.
// Safe to call after session takeover — uses recover to handle closed channel.
func (s *Session) sendError(text string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("sendError: channel closed (session takeover)", "player", s.playerName)
		}
	}()
	msg, err := json.Marshal(ServerMessage{
		Type: MsgError,
		Data: ErrorData{Message: text},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.send <- msg
}

func (s *Session) SendMessage(message string) error {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "text",
			Text: message,
		},
	})
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	select {
	case s.send <- msg:
	default:
		slog.Warn("session send channel full — dropping message", "player", s.playerName)
	}
	return nil
}

// Send sends a text message to the client (alternative method name).
// Routes through Session.send directly — not through Player.Send.
func (s *Session) Send(message string) {
	_ = s.SendMessage(message)
}

// MarkDirty marks a variable as dirty for agent subscriptions.
// Deprecated: prefer markDirty (unexported) which uses the agent mutex.
