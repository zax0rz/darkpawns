// Package session manages WebSocket connections and player sessions.
package session

import (
	"context"
	"time"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"golang.org/x/time/rate"
)

func (s *Session) GetPlayer() *game.Player {
	return s.player
}

// GetPlayerInterface returns the player as interface{} for common.CommandSession
func (s *Session) GetPlayerInterface() interface{} {
	return s.player
}

// SendMessage sends a message to the client.
// Routes through Session.send (which writePump reads) — not through Player.Send.
func (s *Session) MarkDirty(vars ...string) {
	s.markDirty(vars...)
}

// GetManager returns the session manager (needed for some admin commands)
func (s *Session) GetManager() interface{} {
	return s.manager
}

// GetWorld returns the game world.
func (s *Session) GetWorld() *game.World {
	return s.manager.world
}

// GetCombatEngine returns the combat engine.
func (s *Session) GetCombatEngine() interface{} {
	return s.manager.combatEngine
}

// GetPlayerName returns the name of the player associated with this session
func (s *Session) GetPlayerName() string {
	if s.player != nil {
		return s.player.Name
	}
	return s.playerName
}

// IsAuthenticated returns whether the session is authenticated
func (s *Session) IsAuthenticated() bool {
	return s.authenticated
}

// GetPlayerRoomVNum returns the room VNum where the player is located
func (s *Session) GetPlayerRoomVNum() int {
	if s.player != nil {
		return s.player.GetRoomVNum()
	}
	return 0
}

// HasPlayer returns true if the session has a player associated with it
func (s *Session) HasPlayer() bool {
	return s.player != nil
}

// GetPlayerLevel returns the player's level (0 if no player).
func (s *Session) GetPlayerLevel() int {
	if s.player != nil {
		return s.player.Level
	}
	return 0
}

// NewSession creates a bare session not associated with any WebSocket (for telnet/embed use).
func (m *Manager) NewSession() *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		manager:        m,
		send:           make(chan []byte, 256),
		limiter:        rate.NewLimiter(rate.Limit(10), 10),
		subscribedVars: make(map[string]bool),
		dirtyVars:      make(map[string]bool),
		connectedAt:    time.Now(),
		sessionCtx:     ctx,
		cancelFunc:     cancel,
		transportDone:  make(chan struct{}),
	}
}

// Manager returns the session manager that owns this session.
func (s *Session) Manager() *Manager {
	return s.manager
}

// PlayerName returns the player name associated with this session.
func (s *Session) PlayerName() string {
	return s.playerName
}

// CloseSend closes the session's outgoing message channel.
func (s *Session) CloseSend() {
	if s.send == nil {
		return
	}
	// Hold the exclusive lock so no SendMessage is mid-send when we close, and
	// flip sendClosed so subsequent sends become no-ops instead of panicking.
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	s.sendOnce.Do(func() {
		s.sendClosed = true
		close(s.send)
	})
}

// SendClosed reports whether the session's outgoing message channel has been
// closed. Transport layers (telnet, WebSocket) check this after login to detect
// that handleLogin rejected the client, so they can flush pending error
// messages before closing the raw connection (DP-591 race).
func (s *Session) SendClosed() bool {
	s.sendMu.RLock()
	defer s.sendMu.RUnlock()
	return s.sendClosed
}

// SendChannel returns the session's outgoing message channel (for telnet/embed).
func (s *Session) SendChannel() <-chan []byte {
	return s.send
}

// HandleMessage is the exported version of handleMessage (for telnet/embed).
func (s *Session) HandleMessage(data []byte) error {
	return s.handleMessage(data)
}

// Close closes the session
func (s *Session) Close() {
	// Close the connection only; channel close is handled by Unregister()
	if s.conn != nil {
		_ = s.conn.Close()
	}
	if s.closeFunc != nil {
		s.closeFunc()
	}
}

// SetCloseFunc sets an optional transport-specific close callback. Telnet uses
// this so that the linkdead reaper can force-close the raw TCP connection
// (DP-928).
func (s *Session) SetCloseFunc(fn func()) {
	s.closeFunc = fn
}

// DetachTransport stops the transport writer while leaving the authenticated
// session and player registered for C-style linkdead handling. Telnet EOF is
// not an orderly quit: close_socket() keeps the playing character in the
// world with a NULL descriptor so tell can report the linkless state.
func (s *Session) DetachTransport() {
	s.transportOnce.Do(func() {
		close(s.transportDone)
	})
}

// TransportDone is closed when a transport disappears but the session remains
// registered as linkdead. It is consumed by transport-specific writer loops.
func (s *Session) TransportDone() <-chan struct{} {
	return s.transportDone
}

// IsCharCreating returns whether the session is currently in character creation.
func (s *Session) IsCharCreating() bool {
	return s.charCreating
}

// IsMenuActive reports whether the session is waiting at the post-MOTD menu.
func (s *Session) IsMenuActive() bool {
	return s.menuActive
}

// HasDatabase returns whether the session manager has a database connection.
func (m *Manager) HasDatabase() bool {
	return m.hasDB
}

// GetDatabase returns the database instance.
func (m *Manager) GetDatabase() db.Database {
	return m.db
}

// RemoteIP extracts the client IP address from request (WS) or directly (Telnet).
func (s *Session) RemoteIP() string {
	if s.request != nil {
		return auth.GetIPFromRequest(s.request)
	}
	if s.remoteIP != "" {
		return s.remoteIP
	}
	return "127.0.0.1" // fallback
}

// SetRemoteIP explicitly sets the IP address for non-HTTP sessions (e.g. Telnet).
func (s *Session) SetRemoteIP(ip string) {
	s.remoteIP = ip
}

// SetBanLevel stores the site ban level for this session (game.BanNew or game.BanSelect).
// Called after connection is accepted to enforce restrictions at login/char-creation time.
func (s *Session) SetBanLevel(level int) {
	s.banLevel = level
}
