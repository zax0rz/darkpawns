// ARCHITECTURAL NOTE [M-06]: Competing command session interfaces
//
// This file defines common.CommandSession and common.CommandManager.
// There is a parallel interface at pkg/command/interface.go
// (command.SessionInterface) that serves a similar purpose.
//
// common.CommandSession:
//   - Defined here to break circular dependencies between pkg/session and consumers.
//   - Returns GetPlayer() as interface{} — loses type safety.
//   - Used by: pkg/command command registrations (legacy path).
//
// command.SessionInterface (pkg/command/interface.go):
//   - Returns GetPlayer() as *game.Player — full type safety.
//   - Exposes GetWorld(), GetCombatEngine(), temp data, and RNG.
//   - Used by: newer command handlers that need rich session context.
//
// Canonical: command.SessionInterface should be the single session contract.
//   It provides type-safe player access and the richer context that
//   command handlers actually need.
//
// Migration path:
//   1. Add missing methods from CommandSession to SessionInterface
//      (IsAuthenticated, HasPlayer, Close).
//   2. Update command.CommandManager to use SessionInterface instead.
//   3. Deprecate common.CommandSession and common.CommandManager.
//   4. Remove this file once all callers migrate.
//
// Deferred to future refactor. See RESEARCH-LOG.md [DESIGN].

// Package common provides shared interfaces and types to break circular dependencies.
package common

// CommandSession defines the interface for a session that can execute commands.
// DEPRECATED: prefer command.SessionInterface. See architectural note above.
type CommandSession interface {
	// Send sends a message to the session
	Send(string)

	// Close closes the session
	Close()

	// GetPlayer returns the player associated with this session
	GetPlayer() interface{}

	// GetPlayerName returns the name of the player associated with this session
	GetPlayerName() string

	// GetPlayerRoomVNum returns the room VNum where the player is located
	GetPlayerRoomVNum() int

	// IsAuthenticated returns whether the session is authenticated
	IsAuthenticated() bool

	// HasPlayer returns true if the session has a player associated with it
	HasPlayer() bool
}

// CommandManager defines the interface for managing commands.
type CommandManager interface {
	// RegisterCommand registers a command handler
	RegisterCommand(name string, handler func(CommandSession, []string) error)

	// Sessions returns all active sessions
	Sessions() []CommandSession

	// Lock/Unlock for thread-safe access
	Lock()
	Unlock()
	RLock()
	RUnlock()

	// Mu returns the mutex for synchronization
	Mu() interface{} // Returns sync.Locker
}
