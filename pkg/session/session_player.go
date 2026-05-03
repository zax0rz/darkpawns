// Package session manages WebSocket connections and player sessions.
package session

import (
	"time"

	"golang.org/x/time/rate"
	"github.com/zax0rz/darkpawns/pkg/game"
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
	return &Session{
		manager:        m,
		send:           make(chan []byte, 256),
		limiter:        rate.NewLimiter(rate.Limit(10), 10),
		subscribedVars: make(map[string]bool),
		dirtyVars:      make(map[string]bool),
		connectedAt:    time.Now(),
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
	if s.send != nil {
		s.sendOnce.Do(func() { close(s.send) })
	}
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
}

// SetTempData stores temporary data in the session
