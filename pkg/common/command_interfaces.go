// Package common provides shared interfaces and types to break circular dependencies.
package common

// CommandSession defines the interface for a session that can execute commands.
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