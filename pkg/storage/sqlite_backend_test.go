package storage

import (
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newTestBackend opens a fresh SQLiteBackend in a temp directory and returns
// it along with a cleanup function. Each test gets its own isolated database.
func newTestBackend(t *testing.T) (*SQLiteBackend, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	backend, err := NewSQLiteBackend(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}
	return backend, func() { _ = backend.Close() }
}

// newTestPlayer builds a fully-initialized player via NewPlayer (which sets up
// Inventory, Equipment, SkillManager, etc.) then overrides the fields we want
// to verify in round-trip tests.
func newTestPlayer(name string) *game.Player {
	p := game.NewPlayer(1, name, 3001)
	p.Level = 5
	p.Gold = 750
	p.Exp = 12000
	return p
}

// ----- PlayerStore -----

func TestSQLiteBackend_PlayerRoundTrip(t *testing.T) {
	backend, cleanup := newTestBackend(t)
	defer cleanup()

	player := newTestPlayer("TestHero")
	if err := backend.Save(player); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := backend.Load("TestHero")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Name != "TestHero" {
		t.Errorf("Name = %q, want TestHero", loaded.Name)
	}
	if loaded.Level != 5 {
		t.Errorf("Level = %d, want 5", loaded.Level)
	}
	if loaded.Gold != 750 {
		t.Errorf("Gold = %d, want 750", loaded.Gold)
	}
	if loaded.Exp != 12000 {
		t.Errorf("Exp = %d, want 12000", loaded.Exp)
	}
}

func TestSQLiteBackend_PlayerSaveUpsert(t *testing.T) {
	backend, cleanup := newTestBackend(t)
	defer cleanup()

	// Save once.
	p1 := game.NewPlayer(1, "Dup", 3001)
	p1.Level = 1
	p1.Gold = 10
	if err := backend.Save(p1); err != nil {
		t.Fatalf("Save #1: %v", err)
	}

	// Save again with updated state — must replace, not duplicate.
	p2 := game.NewPlayer(1, "Dup", 3001)
	p2.Level = 10
	p2.Gold = 999
	if err := backend.Save(p2); err != nil {
		t.Fatalf("Save #2: %v", err)
	}

	loaded, err := backend.Load("Dup")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Level != 10 {
		t.Errorf("Level = %d, want 10 (upsert should replace)", loaded.Level)
	}
	if loaded.Gold != 999 {
		t.Errorf("Gold = %d, want 999", loaded.Gold)
	}

	// List should have exactly one entry.
	names, err := backend.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 1 {
		t.Errorf("List() = %v, want exactly 1 entry after upsert", names)
	}
}

func TestSQLiteBackend_LoadNotFound(t *testing.T) {
	backend, cleanup := newTestBackend(t)
	defer cleanup()

	_, err := backend.Load("Nonexistent")
	if err == nil {
		t.Fatal("Load of nonexistent player should return an error")
	}
}

func TestSQLiteBackend_Exists(t *testing.T) {
	backend, cleanup := newTestBackend(t)
	defer cleanup()

	exists, err := backend.Exists("Ghost")
	if err != nil {
		t.Fatalf("Exists on missing player: %v", err)
	}
	if exists {
		t.Error("Exists should be false for a player that was never saved")
	}

	if err := backend.Save(game.NewPlayer(1, "Ghost", 3001)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	exists, err = backend.Exists("Ghost")
	if err != nil {
		t.Fatalf("Exists after save: %v", err)
	}
	if !exists {
		t.Error("Exists should be true after saving the player")
	}
}

func TestSQLiteBackend_Delete(t *testing.T) {
	backend, cleanup := newTestBackend(t)
	defer cleanup()

	if err := backend.Save(game.NewPlayer(1, "ToDelete", 3001)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := backend.Delete("ToDelete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	exists, _ := backend.Exists("ToDelete")
	if exists {
		t.Error("player should not exist after Delete")
	}

	// Deleting a nonexistent player should not error (idempotent).
	if err := backend.Delete("ToDelete"); err != nil {
		t.Errorf("Delete of nonexistent player should be idempotent, got: %v", err)
	}
}

func TestSQLiteBackend_List(t *testing.T) {
	backend, cleanup := newTestBackend(t)
	defer cleanup()

	// Empty list initially.
	names, err := backend.List()
	if err != nil {
		t.Fatalf("List (empty): %v", err)
	}
	if len(names) != 0 {
		t.Errorf("List() = %v, want empty slice", names)
	}

	for i, name := range []string{"Charlie", "Alice", "Bob"} {
		if err := backend.Save(game.NewPlayer(i+1, name, 3001)); err != nil {
			t.Fatalf("Save %s: %v", name, err)
		}
	}

	names, err = backend.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("List() = %v, want 3 entries", names)
	}
	// List is ordered by name.
	want := []string{"Alice", "Bob", "Charlie"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("names[%d] = %q, want %q", i, names[i], w)
		}
	}
}

// ----- WorldStore -----

func TestSQLiteBackend_WorldRoundTrip(t *testing.T) {
	backend, cleanup := newTestBackend(t)
	defer cleanup()

	world, err := game.NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	if err := backend.SaveWorld(world); err != nil {
		t.Fatalf("SaveWorld: %v", err)
	}

	// LoadWorld into a fresh world must not error.
	world2, err := game.NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld #2: %v", err)
	}
	if err := backend.LoadWorld(world2); err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
}

func TestSQLiteBackend_LoadWorldEmpty(t *testing.T) {
	backend, cleanup := newTestBackend(t)
	defer cleanup()

	// LoadWorld on a fresh backend (no prior SaveWorld) returns nil —
	// first boot with no saved world state.
	world, err := game.NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if err := backend.LoadWorld(world); err != nil {
		t.Errorf("LoadWorld on empty backend should return nil, got: %v", err)
	}
}
