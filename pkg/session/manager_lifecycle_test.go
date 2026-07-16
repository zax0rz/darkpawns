package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// newTestManager ensures NewManager's combat ticker cannot outlive its test.
// Use this instead of NewManager directly in package session tests.
func newTestManager(t *testing.T, world *game.World, database db.Database) *Manager {
	t.Helper()
	m := NewManager(world, database)
	t.Cleanup(m.Stop)
	return m
}
