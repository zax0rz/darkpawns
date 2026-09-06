// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

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
	s.forwardSnoopOutput(message)
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
	s.notePlayerOutput()
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

// forwardSnoopOutput mirrors comm.c:1646-1651. C forwards the target's
// flushed descriptor output to its snooper with a percent delimiter; the
// session transport has no shared descriptor buffer, so each player-facing
// message is the smallest faithful flush unit available here.
func (s *Session) forwardSnoopOutput(message string) {
	if s == nil || s.manager == nil || message == "" {
		return
	}
	s.manager.snoopMu.RLock()
	snooper := s.snoopBy
	s.manager.snoopMu.RUnlock()
	if snooper != nil && snooper != s {
		snooper.Send("% " + message + "%%")
	}
}

// forwardSnoopInput mirrors comm.c:1992-1998 for descriptor-backed player
// input. It is called before command/editor routing so snooping observes the
// original line rather than a rewritten command.
func (s *Session) forwardSnoopInput(command, rawArgs string, args []string) {
	if s == nil || s.manager == nil || s.player == nil || command == "" {
		return
	}
	s.manager.snoopMu.RLock()
	snooper := s.snoopBy
	s.manager.snoopMu.RUnlock()
	if snooper == nil || snooper == s {
		return
	}
	if rawArgs == "" && len(args) > 0 {
		rawArgs = strings.Join(args, " ")
	}
	line := command
	if rawArgs != "" {
		line += " " + rawArgs
	}
	snooper.Send("% " + line + "\r\n")
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

// notePlayerOutput records player-bound output since the last prompt. C's
// process_output appends "\r\n" plus the prompt to every output flush
// (comm.c:1624-1640), so the session needs to know whether a flush happened
// to reproduce that trailing framing.
func (s *Session) notePlayerOutput() {
	s.outputSincePrompt.Add(1)
}

// SendPrompt enqueues a prompt marker on the session's outgoing channel so the
// transport writes the prompt only after all earlier output (FIFO ordering).
// Telnet renders it as the "> " command prompt; WebSocket clients may ignore
// it. Sharing the channel with command output guarantees C's game-loop order
// (comm.c:643-648): output is flushed first, the prompt is written after.
// When player output was flushed since the previous prompt, C's flush frame
// is "\r\n" + prompt (non-compact process_output); without pending output the
// bare game-loop prompt pass writes the prompt alone.
// Safe to call when the channel is closed — the send is dropped like SendMessage.
func (s *Session) SendPrompt() {
	cmdInfoBarUpdate(s)
	text := s.promptText()
	if s.outputSincePrompt.Swap(0) > 0 {
		text = "\r\n" + text
	}
	msg, err := json.Marshal(ServerMessage{
		Type: MsgPrompt,
		Data: map[string]interface{}{"text": text},
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
	// C's make_prompt returns a bare "] " immediately while d->str is active.
	// The board, note, and mail editors therefore suppress invisibility and
	// vitals fields as well as the normal AFK/inactive prefixes.
	if flags&(1<<uint(game.PlrWriting)) != 0 {
		return "] "
	}
	// C's status branches rebuild prompt in-place with sprintf(prompt, ...)
	// after the vitals/invisibility fields. On the oracle libc this overwrites
	// those earlier fields, leaving only the status marker; INACTIVE is the
	// later branch and therefore wins if both flags are present.
	if flags&(1<<uint(game.PrfInactive)) != 0 {
		return "INACTIVE > "
	}
	if flags&(1<<uint(game.PrfAFK)) != 0 {
		return "AFK > "
	}
	prefix := ""
	if level := s.player.GetInvisLevel(); level > 0 {
		// C's make_prompt adds the wizinvis marker itself; process_output owns
		// the preceding CRLF when an output buffer is being flushed
		// (comm.c:1062-1065, 1624-1640).
		prefix = fmt.Sprintf("i%d ", level)
	}
	// C's make_prompt playing branch renders the vitals fields (HP/mana/move)
	// only when the infobar is off (comm.c:1064-1105); the VT100 infobar owns
	// that data otherwise. Colors are transport presentation stripped by the
	// differential normalizer, so only the numeric fields are emitted.
	if s.infobarMode != InfobarOn {
		if flags&(1<<uint(game.PrfDisphp)) != 0 {
			prefix += fmt.Sprintf("%dH ", s.player.Health)
		}
		if flags&(1<<uint(game.PrfDispmmana)) != 0 {
			prefix += fmt.Sprintf("%dM ", s.player.Mana)
		}
		if flags&(1<<uint(game.PrfDispmove)) != 0 {
			prefix += fmt.Sprintf("%dV ", s.player.Move)
		}
	}
	return prefix + "> "
}

// MarkDirty marks a variable as dirty for agent subscriptions.
// Deprecated: prefer markDirty (unexported) which uses the agent mutex.
