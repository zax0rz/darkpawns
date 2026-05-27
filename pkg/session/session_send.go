// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
			Mana:      s.player.Mana,
			MaxMana:   s.player.MaxMana,
			Move:      s.player.Move,
			MaxMove:   s.player.MaxMove,
			Gold:      s.player.Gold,
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

	// Send MOTD first (splash screen before room — matches original CircleMUD order).
	// Agents still receive the state signal immediately after.
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

	// Send state — this is the "you're in the world" signal for both agents and humans.
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

// sendErrorWithState sends an error message, then re-sends the current expected prompt.
// This is the SEEP (State-Echo Error Protocol) implementation.
// Agents receive deterministic state recovery after every error.
// Humans see the same error + prompt they already see — no behavioral change.
func (s *Session) sendErrorWithState(err error) {
	s.sendError(err.Error())

	// Re-broadcast the current expected input, derived from server state.
	switch {
	case s.charCreating:
		s.resendCurrentCharPrompt()
	case !s.authenticated:
		// Not in char creation, not authenticated — prompt for login
		s.sendCharCreatePrompt("login", "Send a login message to begin.",
			map[string]string{"login": "{type:'login', data:{player_name, password, new_char}}"})
	case s.authenticated && s.player != nil:
		// Already in the world — re-send room state
		s.sendCurrentRoomState()
	}
}

// resendCurrentCharPrompt dispatches the correct char creation prompt based on s.charStage.
// Pure lookup — no DB access, no side effects.
func (s *Session) resendCurrentCharPrompt() {
	switch s.charStage {
	case "color":
		s.sendCharCreatePrompt("color", "Do you want ANSI color? (Y/N):",
			map[string]string{"Y": "Yes", "N": "No"})
	case "sex":
		s.sendCharCreatePrompt("sex", "Select your sex (M/F):",
			map[string]string{"M": "Male", "F": "Female"})
	case "race":
		s.sendCharCreatePrompt("race", "Select your race:", s.getRaceOptions())
	case "class":
		s.sendCharCreatePrompt("class", "Select your class:", s.getClassOptions(s.charRace))
	case "hometown":
		s.sendCharCreatePrompt("hometown", "Choose your hometown:", map[string]string{
			"K": "Kir Drax'in", "O": "Kir-Oshi", "A": "Alaozar",
		})
	case "stats_roll":
		s.sendStatsRollPrompt()
	default:
		// Unknown stage — fall back to login prompt
		s.sendCharCreatePrompt("login", "Send a login message to begin.",
			map[string]string{"login": "{type:'login', data:{player_name, password, new_char}}"})
	}
}

// sendCurrentRoomState re-sends the player's current room state.
// Used by SEEP when an authenticated player sends an unrecognizable message.
func (s *Session) sendCurrentRoomState() {
	if s.player == nil {
		return
	}
	roomVNum := s.player.GetRoom()
	room, ok := s.manager.world.GetRoom(roomVNum)
	if !ok || room == nil {
		return
	}

	state := StateData{
		Player: PlayerState{
			Name:      s.player.Name,
			Health:    s.player.Health,
			MaxHealth: s.player.MaxHealth,
			Mana:      s.player.Mana,
			MaxMana:   s.player.MaxMana,
			Move:      s.player.Move,
			MaxMove:   s.player.MaxMove,
			Gold:      s.player.Gold,
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
