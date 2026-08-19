// Package storage provides persistence backends for game data.
package storage

import (
	"context"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// PlayerStore defines the interface for player persistence.
//
// Every method takes a context.Context (DP-759): the SQLite backend holds a
// single connection (MaxOpenConns=1), so one hung call — WAL checkpoint
// stall, filesystem hang, busy-timeout exhaustion — would otherwise block
// every subsequent caller with no cancellation path. Callers should pass a
// context with a deadline appropriate to the operation (e.g. 10s for a
// player save); context.Background() is acceptable only where no natural
// deadline exists (e.g. offline tooling).
type PlayerStore interface {
	// Save persists a player's state.
	Save(ctx context.Context, player *game.Player) error

	// Load retrieves a player by name.
	Load(ctx context.Context, name string) (*game.Player, error)

	// Delete removes a player's saved state.
	Delete(ctx context.Context, name string) error

	// Exists checks whether a player has saved data.
	Exists(ctx context.Context, name string) (bool, error)

	// List returns all saved player names.
	List(ctx context.Context) ([]string, error)
}

// WorldStore defines the interface for world persistence.
type WorldStore interface {
	// SaveWorld persists the full world state (rooms, mobs, shops, etc.).
	SaveWorld(ctx context.Context, w *game.World) error

	// LoadWorld restores dynamic world state into an existing World.
	// Must be called after NewWorld() and zone resets so mobs are spawned.
	LoadWorld(ctx context.Context, w *game.World) error
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
