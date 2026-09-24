package session

import (
	"log/slog"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// performDupeCheck ports perform_dupe_check (interpreter.c:1528-1659), which
// CON_PASSWORD runs as soon as a returning character's password is accepted
// (interpreter.c:1914-1916). When another session already holds the
// character in the game, this session takes over that live body instead of
// the one just read from the store, and play resumes where it was: fighting,
// affects and position included. It returns true when the login became a
// reconnect, so the caller must not continue to the MOTD and menu.
//
// C's three modes, checked in its order:
//   - UNSWITCH: the old session is an immortal switched into another body
//     (k->original). It is disconnected, and the new session takes the
//     immortal's own body with "Reconnecting to unswitched char.".
//   - USURP: the old session still has a connection. It is told the body was
//     usurped and disconnected, and the room sees the new spirit take over.
//   - RECON: the old session is linkdead (DP-1323). "Reconnecting." to the
//     player and "$n has reconnected." to the room.
//
// A session that is not playing (at the menu, say) is simply disconnected
// with "Multiple login detected", as C does for any other descriptor on the
// same character, and the login continues normally.
func (s *Session) performDupeCheck() bool {
	if s == nil || s.player == nil || s.manager == nil {
		return false
	}
	m := s.manager
	name := s.player.Name

	m.mu.Lock()
	old, ok := m.sessions[name]
	if !ok || old == s || old.player == nil {
		m.mu.Unlock()
		return false
	}
	if old.menuActive || old.charCreating {
		m.mu.Unlock()
		// Not CON_PLAYING: disconnected, and no target (interpreter.c:1561-1571).
		old.Send("\r\nMultiple login detected -- disconnecting.\r\n")
		m.Unregister(name)
		old.CloseSend()
		old.Close()
		return false
	}
	unswitch := old.isSwitched
	usurp := !unswitch && old.hasTransport() && !old.SendClosed()
	// The old session's transport teardown must not treat the body as its own
	// any more: it neither goes linkdead nor unregisters the new session.
	old.superseded.Store(true)
	m.sessions[name] = s
	m.mu.Unlock()

	switch {
	case unswitch:
		old.Send("\r\nMultiple login detected -- disconnecting.\r\n")
		old.CloseSend()
		old.Close()
	case usurp:
		old.Send("\r\nThis body has been usurped!\r\n")
		old.Send("\r\nMultiple login detected -- disconnecting.\r\n")
		old.CloseSend()
		old.Close()
	default:
		// A linkdead session's pumps have already exited; release what is left.
		old.CloseSend()
		if old.cancelFunc != nil {
			old.cancelFunc()
		}
	}

	// Connect this descriptor to the live body (interpreter.c:1618-1628).
	p := old.player
	s.player = p
	s.playerName = name
	s.authenticated = true
	s.olcZone = old.olcZone
	s.charCreating = false
	s.charStage = ""
	s.charPassword = ""
	s.clearMenuState()
	s.lastActive.Store(time.Now().UnixNano())
	p.SetLinkless(false)
	p.SetIdleTimer(0)
	p.SetPlrFlag(game.PlrMailing, false)
	p.SetPlrFlag(game.PlrWriting, false)

	// CON_PASSWORD's echo_on() comes before everything else: IAC WONT ECHO and
	// then CR LF (comm.c:954-967). The empty non-secret entry frame is the
	// WONT ECHO, and takes a browser out of password mode; the CR LF follows.
	// Without it a telnet client would stay silent after reconnecting.
	s.sendCharCreatePrompt("reconnect", "", nil)
	s.Send("\r\n")

	w := m.world
	switch {
	case unswitch:
		// No line ending in C (interpreter.c:1650); the prompt follows.
		s.sendRawEvent("Reconnecting to unswitched char.")
		slog.Info("reconnected to unswitched char", "player", name, "ip", s.RemoteIP())
	case usurp:
		s.Send("You take over your own body, already in use!\r\n")
		game.Act(w, true, p, nil, nil, nil,
			"$n suddenly keels over in pain, surrounded by a white aura...\r\n"+
				"$n's body has been taken over by a new spirit!", "", game.ToRoom)
		slog.Info("re-logged in; disconnecting old socket", "player", name)
	default:
		s.Send("Reconnecting.\r\n")
		if game.HasMail(int(p.GetID())) {
			s.Send("You have mail waiting.\r\n")
		}
		game.Act(w, true, p, nil, nil, nil, "$n has reconnected.", "", game.ToRoom)
		slog.Info("reconnected", "player", name, "ip", s.RemoteIP())
	}

	// A GMCP client learns who it is again; the text bytes above are C's.
	s.gmcpSync()
	if s.wantsStructuredData {
		s.sendFullVarDump()
	}
	// C's next game-loop pass prints the prompt (comm.c:643-648).
	s.SendPrompt()
	return true
}
