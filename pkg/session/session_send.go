// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/game"
)

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

	// Welcome text — matches C WELC_MESSG (config.c:256).
	welcomeMsg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "text",
			Text: "\r\nWelcome to Dark Pawns! May your visit here be... Interesting.\r\n\r\n",
		},
	})
	if err == nil {
		s.send <- welcomeMsg
	}

	// The canonical room result now supplies both C-faithful text and the
	// unchanged structured "you're in the world" state signal.
	s.sendRoomObservation(roomVNum, false, token)
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
	case s.menuActive:
		s.resendCurrentMenuPrompt()
	case s.charCreating:
		s.resendCurrentCharPrompt()
	case !s.authenticated:
		// Not in char creation, not authenticated — prompt for login
		s.sendCharCreatePrompt("login", "Send a login message to begin.",
			charOpts("login", "{type:'login', data:{player_name, password, new_char}}"))
	case s.authenticated && s.player != nil:
		// Already in the world — re-send room state
		s.sendCurrentRoomState()
	}
}

// resendCurrentCharPrompt dispatches the correct char creation prompt based on s.charStage.
// Pure lookup — no DB access, no side effects.
func (s *Session) resendCurrentCharPrompt() {
	switch s.charStage {
	case "get_name":
		s.sendCharCreatePrompt("get_name", "Name: ", nil)
	case "confirm_name":
		s.sendCharCreatePrompt("confirm_name", fmt.Sprintf("Did I get that right, %s (Y/N)? ", s.charName), nil)
	case "create_password":
		s.sendCharCreatePromptWithSecret("create_password", "Password: ", nil, true)
	case "confirm_password":
		s.sendCharCreatePromptWithSecret("confirm_password", "\r\nPlease retype password: ", nil, true)
	case "color":
		s.sendCharCreatePrompt("color", "Do you want ANSI color (Y/N)? ", nil)
	case "sex":
		s.sendCharCreatePrompt("sex", "What is your sex (M/F)? ", nil)
	case "race":
		s.sendCharCreatePrompt("race", RaceMenuText+"\r\nRace: ", s.getRaceOptions())
	case "class":
		menu := ClassMenuText
		if s.charRace == game.RaceHuman {
			menu = HumanClassMenuText
		}
		s.sendCharCreatePrompt("class", menu+"\r\nClass: ", s.getClassOptions(s.charRace))
	case "hometown":
		s.sendCharCreatePrompt("hometown", HometownMenuText+"\r\nSelect: ",
			charOpts("K", "Kir Drax'in", "O", "Kir-Oshi", "A", "Alaozar"))
	case "stats_roll":
		s.sendStatsRollPrompt()
	default:
		// Unknown stage — fall back to login prompt
		s.sendCharCreatePrompt("login", "Send a login message to begin.",
			charOpts("login", "{type:'login', data:{player_name, password, new_char}}"))
	}
}

// sendCurrentRoomState re-sends the player's current room state.
// Used by SEEP when an authenticated player sends an unrecognizable message.
func (s *Session) sendCurrentRoomState() {
	if s.player == nil {
		return
	}
	roomVNum := s.player.GetRoom()
	if room, ok := s.manager.world.GetRoom(roomVNum); !ok || room == nil {
		return
	}
	s.sendRoomObservation(roomVNum, false, "")
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
	// RLock lets concurrent sends proceed but blocks the exclusive close, so we
	// never send on a closed channel (which panics even inside a select). If the
	// channel is already closed, drop silently — the client is gone.
	s.sendMu.RLock()
	defer s.sendMu.RUnlock()
	if s.sendClosed {
		return nil
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

// sendRawEvent sends a transport event without the telnet adapter appending a
// line ending. C's infobar writes VT100 control bytes directly to the
// descriptor and the following command text begins immediately afterward.
func (s *Session) sendRawEvent(message string) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{Type: "raw", Text: message},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.sendMu.RLock()
	defer s.sendMu.RUnlock()
	if s.sendClosed {
		return
	}
	select {
	case s.send <- msg:
	default:
		slog.Warn("session sendRawEvent channel full — dropping message", "player", s.playerName)
	}
}

// SendPrompt enqueues a prompt marker on the session's outgoing channel so the
// transport writes the prompt only after all earlier output (FIFO ordering).
// Telnet renders it as the "> " command prompt; WebSocket clients may ignore
// it. Sharing the channel with command output guarantees C's game-loop order
// (comm.c:643-648): output is flushed first, the prompt is written after.
// Safe to call when the channel is closed — the send is dropped like SendMessage.
func (s *Session) SendPrompt() {
	cmdInfoBarUpdate(s)
	msg, err := json.Marshal(ServerMessage{
		Type: MsgPrompt,
		Data: map[string]interface{}{"text": s.promptText()},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.sendMu.RLock()
	defer s.sendMu.RUnlock()
	if s.sendClosed {
		return
	}
	select {
	case s.send <- msg:
	default:
		slog.Warn("session send channel full — dropping prompt", "player", s.playerName)
	}
}

// promptText returns the state prefixes that C's make_prompt emits after the
// regular display fields. When both flags are set, PRF_INACTIVE wins because
// the later sprintf in comm.c overwrites the earlier PRF_AFK prompt.
func (s *Session) promptText() string {
	if s.player == nil {
		return "> "
	}
	flags := s.player.GetFlags()
	// C make_prompt() emits a bare "] " while d->str is active. The board,
	// note, and mail editors all set PLR_WRITING, so preserve that framing
	// before the normal AFK/inactive prompt prefixes.
	if flags&(1<<uint(game.PlrWriting)) != 0 {
		return "\r\n] "
	}
	if flags&(1<<uint(game.PrfInactive)) != 0 {
		return "INACTIVE > "
	}
	if flags&(1<<uint(game.PrfAFK)) != 0 {
		return "AFK > "
	}
	return "> "
}

// MarkDirty marks a variable as dirty for agent subscriptions.
// Deprecated: prefer markDirty (unexported) which uses the agent mutex.
