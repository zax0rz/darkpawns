// Package storage provides persistence backends for game data.
package storage

import (
	"github.com/zax0rz/darkpawns/pkg/game"
)

// PlayerStore defines the interface for player persistence.
type PlayerStore interface {
	// Save persists a player's state.
	Save(player *game.Player) error

	// Load retrieves a player by name.
	Load(name string) (*game.Player, error)

	// Delete removes a player's saved state.
	Delete(name string) error

	// Exists checks whether a player has saved data.
	Exists(name string) (bool, error)

	// List returns all saved player names.
	List() ([]string, error)
}

// WorldStore defines the interface for world persistence.
type WorldStore interface {
	// SaveWorld persists the full world state (rooms, mobs, shops, etc.).
	SaveWorld(w *game.World) error

	// LoadWorld restores dynamic world state into an existing World.
	// Must be called after NewWorld() and zone resets so mobs are spawned.
	LoadWorld(w *game.World) error
}

// FullBackend combines both stores into a single persistence backend.
type FullBackend interface {
	PlayerStore
	WorldStore
	// Close releases any backend resources.
	Close() error
}

// Compile-time assertions that the concrete backend satisfies every storage
// interface. Adding a method to any of these interfaces without updating
// SQLiteBackend will then fail at compile time instead of panicking at runtime
// when a *SQLiteBackend is assigned to the interface (DP-814).
var (
	_ PlayerStore = (*SQLiteBackend)(nil)
	_ WorldStore  = (*SQLiteBackend)(nil)
	_ FullBackend = (*SQLiteBackend)(nil)
)
