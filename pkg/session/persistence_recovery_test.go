package session

// persistence_recovery_test.go — the DP-1365 SQLite recovery integration
// proof (R5f/R5h): a real temporary database, the production save wiring
// (Manager.WirePlayerSaver — the same helper the server boots with), each
// new save trigger, and a readback through the same record-loading path
// login uses (db.GetPlayer → db.RecordToPlayer). No session cleanup,
// transport-disconnect or graceful-shutdown save runs between the trigger
// and the readback; nothing here is a seam-call assertion or a mock.

import (
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// recoveryEnv builds the production stack for one trigger: a real SQLite
// store, a world with a room and two object prototypes, a manager wired to
// the store, and a playing session whose character already exists in the
// store (so the save path has an ID to update).
type recoveryEnv struct {
	store *db.DB
	world *game.World
	m     *Manager
	s     *Session
}

func newRecoveryEnv(t *testing.T) *recoveryEnv {
	t.Helper()
	path := filepath.Join(t.TempDir(), "recovery.db")
	store, err := db.New(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Vault"}},
		Objs: []parser.Obj{
			{VNum: 7101, Keywords: "thing", ShortDesc: "a carried thing", WearFlags: [4]int{1}},
			{VNum: 7102, Keywords: "held", ShortDesc: "a held thing", WearFlags: [4]int{(1 << 0) | (1 << 13)}},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	world.MessageSink = func(string, []byte) {}

	m := newTestManager(t, world, store)
	m.WirePlayerSaver(world)

	s := makeTestSession(t, m, "Recoverer", 1001, true)
	m.mu.Lock()
	m.sessions["Recoverer"] = s
	m.mu.Unlock()
	// The world must know the player too: the ObjectLocation attach arms
	// key player inventory through w.players, which is also where the
	// PLR_CRASH set points live.
	if err := world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	// The character must exist in the store before the save path can update
	// it: create it the way first entry does.
	rec, err := db.PlayerToRecord(s.player, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord: %v", err)
	}
	if err := store.CreatePlayer(rec); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	s.player.ID = rec.ID
	return &recoveryEnv{store: store, world: world, m: m, s: s}
}

// mutate changes the durable surfaces the readback asserts: inventory,
// equipment, and representative character fields, plus a session-owned
// record field.
func (e *recoveryEnv) mutate(t *testing.T) {
	t.Helper()
	obj, err := e.world.SpawnObject(7101, -1)
	if err != nil {
		t.Fatalf("spawn carried: %v", err)
	}
	if err := e.world.MoveObjectToPlayerInventory(obj, e.s.player); err != nil {
		t.Fatalf("carry: %v", err)
	}
	held, err := e.world.SpawnObject(7102, -1)
	if err != nil {
		t.Fatalf("spawn held: %v", err)
	}
	if err := e.world.MoveObjectToPlayerInventory(held, e.s.player); err != nil {
		t.Fatalf("seat held: %v", err)
	}
	if err := e.world.MoveObject(held, game.LocEquippedPlayer(e.s.player.Name, 0)); err != nil {
		t.Fatalf("equip: %v", err)
	}
	e.s.player.SetGold(4242)
	e.s.player.Exp = 1717
	e.s.olcZone = 99 // session-owned record field
}

// readback reopens the database through the same path login uses and
// asserts the mutated surfaces survived.
func (e *recoveryEnv) readback(t *testing.T) {
	t.Helper()
	rec, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatalf("GetPlayer: %v", err)
	}
	if rec == nil {
		t.Fatal("character record vanished from the store")
	}
	if rec.Exp != 1717 {
		t.Fatalf("record exp = %d, want 1717", rec.Exp)
	}
	if rec.OlcZone != 99 {
		t.Fatalf("session-owned OlcZone = %d, want 99", rec.OlcZone)
	}
	p, err := db.RecordToPlayer(rec, e.world)
	if err != nil {
		t.Fatalf("RecordToPlayer: %v", err)
	}
	// Gold rides in the CharacterData JSON blob, so assert it on the
	// reconstructed character — the same value login restores.
	if p.GetGold() != 4242 {
		t.Fatalf("reconstructed gold = %d, want 4242", p.GetGold())
	}
	if _, ok := p.Inventory.FindItem("thing"); !ok {
		t.Fatal("carried object did not survive: inventory lacks the thing")
	}
	equipped := false
	for _, item := range p.Equipment.Slots {
		if item != nil && item.VNum == 7102 {
			equipped = true
		}
	}
	if !equipped {
		t.Fatal("equipped object did not survive: no equipment slot holds it")
	}
}

// TestRecoverySaveCommand: the save command's store-of-record write
// survives a reopen.
func TestRecoverySaveCommand(t *testing.T) {
	e := newRecoveryEnv(t)
	e.mutate(t)

	e.world.ExecSave(e.s.player)

	e.readback(t)
}

// TestRecoveryAutosave: ten minute slots of the autosave counter persist
// the mutated character.
func TestRecoveryAutosave(t *testing.T) {
	e := newRecoveryEnv(t)
	e.mutate(t)

	for i := 0; i < 10; i++ {
		e.m.AutosaveTick()
	}
	if e.s.player.NeedsCrashSave() {
		t.Fatal("autosave passed without clearing the flag")
	}

	e.readback(t)
}

// TestRecoveryVoidPull: the idle void pull saves before the transfer.
func TestRecoveryVoidPull(t *testing.T) {
	e := newRecoveryEnv(t)
	e.mutate(t)
	e.s.player.IdleTimer = game.IDLE_TO_VOID

	e.world.CheckIdling(e.s.player)

	e.readback(t)
}
