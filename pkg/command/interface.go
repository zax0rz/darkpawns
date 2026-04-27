// ARCHITECTURAL NOTE [M-06]: This is the canonical session interface.
//
// command.SessionInterface is the preferred contract for command handlers.
// See pkg/common/command_interfaces.go for the competing common.CommandSession
// interface and the full migration plan. Once migration completes, that file
// will be removed and this becomes the sole session abstraction.
//
// Methods still needed from common.CommandSession:
//   IsAuthenticated() bool, HasPlayer() bool, Close()
// These should be added here before deprecating the common package interface.

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

	// GetWorld returns the game world
	GetWorld() *game.World

	// GetCombatEngine returns the combat engine
	GetCombatEngine() interface{}

	// Temporary data storage methods
	SetTempData(key string, value interface{})
	GetTempData(key string) interface{}
	ClearTempData(key string)

	// Random number generation
	RandomInt(maxValue int) int
}
