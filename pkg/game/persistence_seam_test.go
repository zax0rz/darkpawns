package game

// persistence_seam_test.go — R5h unit proofs for the DP-1365 store-of-record
// seam: the save command's kind and load room, the PLR_CRASH set/clear
// policy at the inventory move points, and the void pull's save position.
// Each test fails when its production line is reverted.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// seamRecorder captures the seam's calls and returns a scripted result.
type seamRecorder struct {
	calls    []string
	result   SaveResult
	onCall   func(p *Player)
	lastRoom int
}

func (r *seamRecorder) saver() PlayerSaver {
	return func(p *Player, why string, loadRoom int) SaveResult {
		r.calls = append(r.calls, why)
		r.lastRoom = loadRoom
		if r.onCall != nil {
			r.onCall(p)
		}
		return r.result
	}
}

func newSeamWorld(t *testing.T, rec *seamRecorder) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Vault"}, {VNum: 1, Name: "The Void"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	if rec != nil {
		w.PlayerSaver = rec.saver()
	}
	p := NewPlayer(1, "Seamtest", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, p
}

// TestDoSaveWritesStoreOfRecordWithNowhere: the save command is C's do_save
// — save_char(ch, NOWHERE) + Crash_crashsave (act.other.c:186-203) — so the
// seam must fire once with NOWHERE and the crash kind. Link-dead characters
// and mobiles get nothing (act.other.c:188).
func TestDoSaveWritesStoreOfRecordWithNowhere(t *testing.T) {
	t.Run("save-command", func(t *testing.T) {
		rec := &seamRecorder{result: SaveSucceeded}
		w, p := newSeamWorld(t, rec)

		w.ExecSave(p)

		if len(rec.calls) != 1 || rec.calls[0] != "save" {
			t.Fatalf("seam calls = %v, want one save", rec.calls)
		}
		if rec.lastRoom != LoadRoomNowhere {
			t.Fatalf("loadRoom = %d, want NOWHERE (%d)", rec.lastRoom, LoadRoomNowhere)
		}
	})
	t.Run("linkdead-silent", func(t *testing.T) {
		rec := &seamRecorder{result: SaveSucceeded}
		w, p := newSeamWorld(t, rec)
		p.SetLinkless(true)

		w.ExecSave(p)

		if len(rec.calls) != 0 {
			t.Fatalf("link-dead save produced seam calls %v", rec.calls)
		}
	})
	t.Run("npc-silent", func(t *testing.T) {
		rec := &seamRecorder{result: SaveSucceeded}
		w, p := newSeamWorld(t, rec)
		mob := &MobInstance{VNum: 2001}

		w.doSave(p, mob, "save", "")

		if len(rec.calls) != 0 {
			t.Fatalf("NPC-routed save produced seam calls %v", rec.calls)
		}
	})
}

// TestInventoryMovesSetCrashFlag: PLR_CRASH is set wherever an object enters
// or leaves a player's inventory or equipment (handler.c:569-571,
// 596-598), stays set through a character-only save, clears after a
// successful crash save, and survives failed and skipped crash saves.
func TestInventoryMovesSetCrashFlag(t *testing.T) {
	mkObj := func(w *World, vnum int) *ObjectInstance {
		obj := newTransferItem(vnum, "a thing", "thing", 1)
		obj.SetWeight(0)
		registerTransferObject(w, obj)
		return obj
	}

	t.Run("enter-and-leave-inventory", func(t *testing.T) {
		w, p := newSeamWorld(t, nil)
		obj := mkObj(w, 7101)
		if p.NeedsCrashSave() {
			t.Fatal("fresh player already flagged")
		}
		if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
			t.Fatalf("carry: %v", err)
		}
		if !p.NeedsCrashSave() {
			t.Fatal("obj_to_char did not set PLR_CRASH")
		}
		p.SetPlrFlag(PlrCrash, false)
		if err := w.MoveObjectToRoom(obj, 1001); err != nil {
			t.Fatalf("drop: %v", err)
		}
		if !p.NeedsCrashSave() {
			t.Fatal("obj_from_char did not set PLR_CRASH on inventory exit")
		}
	})
	t.Run("equip-from-inventory", func(t *testing.T) {
		w, p := newSeamWorld(t, nil)
		obj := newTransferItem(7102, "a thing", "thing", (1<<0)|(1<<13))
		obj.SetWeight(0)
		registerTransferObject(w, obj)
		if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
			t.Fatalf("carry: %v", err)
		}
		p.SetPlrFlag(PlrCrash, false)
		if err := w.MoveObject(obj, LocEquippedPlayer(p.Name, 0)); err != nil {
			t.Fatalf("equip: %v", err)
		}
		if !p.NeedsCrashSave() {
			t.Fatal("equipping (obj_from_char) did not set PLR_CRASH")
		}
	})
	t.Run("char-only-save-keeps-flag", func(t *testing.T) {
		rec := &seamRecorder{result: SaveSucceeded}
		w, p := newSeamWorld(t, rec)
		p.MarkCrashNeeded()

		if res := w.SavePlayerRecord(p, "char only", LoadRoomNowhere, SaveCharOnly); res != SaveSucceeded {
			t.Fatalf("result = %v", res)
		}
		if !p.NeedsCrashSave() {
			t.Fatal("a character-only save cleared PLR_CRASH; only Crash_crashsave clears it")
		}
	})
	t.Run("crash-save-clears-on-success", func(t *testing.T) {
		rec := &seamRecorder{result: SaveSucceeded}
		w, p := newSeamWorld(t, rec)
		p.MarkCrashNeeded()

		if res := w.SavePlayerRecord(p, "crash", LoadRoomNowhere, SaveCrash); res != SaveSucceeded {
			t.Fatalf("result = %v", res)
		}
		if p.NeedsCrashSave() {
			t.Fatal("successful crash save did not clear PLR_CRASH")
		}
	})
	t.Run("failed-and-skipped-retain-flag", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			result SaveResult
		}{{"failed", SaveFailed}, {"skipped", SaveSkipped}} {
			t.Run(tc.name, func(t *testing.T) {
				rec := &seamRecorder{result: tc.result}
				w, p := newSeamWorld(t, rec)
				p.MarkCrashNeeded()

				if res := w.SavePlayerRecord(p, "crash", LoadRoomNowhere, SaveCrash); res != tc.result {
					t.Fatalf("result = %v", res)
				}
				if !p.NeedsCrashSave() {
					t.Fatalf("a %s write cleared PLR_CRASH; the flag must stay for retry", tc.name)
				}
			})
		}
		w2, p2 := newSeamWorld(t, nil) // nil saver = skipped
		p2.MarkCrashNeeded()
		if res := w2.SavePlayerRecord(p2, "crash", LoadRoomNowhere, SaveCrash); res != SaveSkipped {
			t.Fatalf("nil saver result = %v, want SaveSkipped", res)
		}
		if !p2.NeedsCrashSave() {
			t.Fatal("a skipped write cleared PLR_CRASH")
		}
	})
	t.Run("newer-dirtiness-survives-the-save", func(t *testing.T) {
		// The saver mutates the inventory mid-write: the change races the
		// saved snapshot, so the flag must survive even a successful save.
		w, p := newSeamWorld(t, nil)
		racer := mkObj(w, 7103)
		rec := &seamRecorder{result: SaveSucceeded}
		rec.onCall = func(*Player) {
			if err := w.MoveObjectToPlayerInventory(racer, p); err != nil {
				t.Errorf("race carry: %v", err)
			}
		}
		w.PlayerSaver = rec.saver()
		p.MarkCrashNeeded()

		if res := w.SavePlayerRecord(p, "crash", LoadRoomNowhere, SaveCrash); res != SaveSucceeded {
			t.Fatalf("result = %v", res)
		}
		if !p.NeedsCrashSave() {
			t.Fatal("a change made after the saved snapshot lost its PLR_CRASH")
		}
	})
}

// TestCheckIdlingSavesBeforeVoidTransfer: C's check_idling saves while the
// character still stands in the room, before the transfer to room 1
// (limits.c:434-435, DP-1353).
func TestCheckIdlingSavesBeforeVoidTransfer(t *testing.T) {
	rec := &seamRecorder{result: SaveSucceeded}
	roomAtSave := -1
	rec.onCall = func(p *Player) { roomAtSave = p.GetRoomVNum() }
	w, p := newSeamWorld(t, rec)
	p.IdleTimer = IDLE_TO_VOID

	w.CheckIdling(p)

	if len(rec.calls) != 1 || rec.calls[0] != "void pull" {
		t.Fatalf("seam calls = %v, want one void-pull save", rec.calls)
	}
	if roomAtSave != 1001 {
		t.Fatalf("room at save time = %d, want 1001 (save must precede the transfer)", roomAtSave)
	}
	if p.GetRoomVNum() != 1 {
		t.Fatalf("player room after CheckIdling = %d, want the void (1)", p.GetRoomVNum())
	}
}
