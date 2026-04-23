package command

import (
	"github.com/zax0rz/darkpawns/pkg/game"
)

// SessionInterface defines the methods that command handlers need from a session.
// This breaks the circular dependency between pkg/session and pkg/command.
type SessionInterface interface {
	// GetPlayer returns the player associated with this session
	GetPlayer() *game.Player

	// SendMessage sends a message to the client
	SendMessage(message string) error

	// Send sends a message to the client (alternative method name)
	Send(message string)

	// MarkDirty marks a variable as dirty for agent subscriptions
	MarkDirty(vars ...string)

	// GetManager returns the session manager (needed for some admin commands)
	GetManager() interface{}

	// Temporary data storage methods
	SetTempData(key string, value interface{})
	GetTempData(key string) interface{}
	ClearTempData(key string)

	// Random number generation
	RandomInt(max int) int
}
