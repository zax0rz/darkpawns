package session

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShowRentKindsAndSavedObjects(t *testing.T) {
	e := newRecoveryEnv(t)
	actor := makeCommandTestSession(t, e.m, "Showgod", 40, 1001)
	actor.wantsStructuredData = true
	proto, _ := e.world.GetObjPrototype(7102)
	proto.LoadPercent = 38.345
	for kind, label := range map[int]string{0: "Undef", 1: "Crash", 2: "Rent", 3: "Cryo", 4: "TimedOut", 5: "TimedOut", 99: "Undef"} {
		if err := db.SaveObjectSnapshot(e.store, "Recoverer", kind, []byte(`[{"vnum":7101,"count":1},{"vnum":99999,"count":1},{"vnum":7101,"count":1}]`), []byte(`[{"vnum":7102,"count":1,"locate":17}]`)); err != nil {
			t.Fatal(err)
		}
		if err := cmdShow(actor, []string{"rent", "RECOVERER"}); err != nil {
			t.Fatal(err)
		}
		want := "plrobjs/P-T/recoverer.objs\r\n" + label + "\r\n" + fmt.Sprintf(" [%5d] (38.35au) <%2d> %-20s\r\n", 7102, 17, "a held thing") + strings.Repeat(fmt.Sprintf(" [%5d] (0.00au) <%2d> %-20s\r\n", 7101, 0, "a carried thing"), 2)
		if got := renderedOutput(actor); got != want {
			t.Fatalf("kind %d: %q want %q", kind, got, want)
		}
	}
	if len(e.world.GetAllObjects()) != 0 {
		t.Fatal("report allocated objects")
	}
}

func TestShowRentAbsentAndEmpty(t *testing.T) {
	e := newRecoveryEnv(t)
	actor := makeCommandTestSession(t, e.m, "Showgod", 40, 1001)
	if err := cmdShow(actor, []string{"rent", "Recoverer"}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(actor); got != "recoverer has no rent file.\r\n" {
		t.Fatalf("absent=%q", got)
	}
	if err := db.SaveObjectSnapshot(e.store, "Recoverer", 1, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := cmdShow(actor, []string{"rent", "Recoverer"}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(actor); got != "plrobjs/P-T/recoverer.objs\r\nCrash\r\n" {
		t.Fatalf("empty=%q", got)
	}
}

func TestObjectSaveBoundariesPreserveSnapshot(t *testing.T) {
	e := newRecoveryEnv(t)
	obj, err := e.world.SpawnObject(7101, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.world.MoveObject(obj, game.LocInventoryPlayer("Recoverer")); err != nil {
		t.Fatal(err)
	}
	if got := e.world.SavePlayerRecord(e.s.player, "test", -1, game.SaveCrash); got != game.SaveSucceeded {
		t.Fatalf("crash save=%v", got)
	}
	snap, err := e.store.GetObjectSave("Recoverer")
	if err != nil || snap == nil || snap.Kind != 1 || len(snap.Objects) != 1 {
		t.Fatalf("crash snapshot: %+v %v", snap, err)
	}
	e.world.ExtractObject(obj, 1001)
	e.m.SaveCharSite(e.s.player, "set")
	snap, err = e.store.GetObjectSave("Recoverer")
	if err != nil || len(snap.Objects) != 1 {
		t.Fatal("character-only save rewrote object snapshot")
	}
	e.s.leaveGameToMenu(true)
	snap, err = e.store.GetObjectSave("Recoverer")
	if err != nil || snap.Kind != 2 || len(snap.Objects) != 0 {
		t.Fatalf("rent snapshot: %+v %v", snap, err)
	}
}

func TestShowRentNativePaging(t *testing.T) {
	e := newRecoveryEnv(t)
	actor := makeCommandTestSession(t, e.m, "Showgod", 40, 1001)
	if err := db.SaveObjectSnapshot(e.store, "Recoverer", 2, []byte(`[{"vnum":7101,"count":30}]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := cmdShow(actor, []string{"rent", "Recoverer"}); err != nil {
		t.Fatal(err)
	}
	first := renderedOutput(actor)
	if !strings.Contains(first, "plrobjs/P-T/recoverer.objs\r\nRent\r\n") || actor.pagerCount < 2 {
		t.Fatalf("first page=%q pages=%d", first, actor.pagerCount)
	}
	actor.navigatePager("")
	if last := renderedOutput(actor); !strings.Contains(last, "a carried thing") || actor.pagerCount != 0 {
		t.Fatalf("last page=%q pages=%d", last, actor.pagerCount)
	}
}

func TestIdleRentObjectSnapshot(t *testing.T) {
	e := newRecoveryEnv(t)
	obj, err := e.world.SpawnObject(7101, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.world.MoveObjectToPlayerInventory(obj, e.s.player); err != nil {
		t.Fatal(err)
	}
	e.s.player.Level = 1
	e.s.player.WasInRoom = 1001
	e.s.player.IdleTimer = game.IDLE_DISCONNECT
	e.world.CheckIdling(e.s.player)
	snap, err := e.store.GetObjectSave("Recoverer")
	if err != nil || snap == nil || snap.Kind != 2 || len(snap.Objects) != 1 {
		t.Fatalf("idle rent snapshot: %+v %v", snap, err)
	}
}

type objectSnapshotFailure struct{ db.GameStore }

func (d objectSnapshotFailure) Exec(query string, args ...interface{}) (sql.Result, error) {
	if strings.HasPrefix(query, "INSERT INTO object_saves") {
		return nil, errors.New("forced snapshot failure")
	}
	return d.GameStore.Exec(query, args...)
}

func TestObjectSnapshotFailureRetainsCrash(t *testing.T) {
	e := newRecoveryEnv(t)
	e.m.db = objectSnapshotFailure{e.store}
	e.s.player.MarkCrashNeeded()
	if got := e.world.SavePlayerRecord(e.s.player, "test", -1, game.SaveCrash); got != game.SaveFailed {
		t.Fatalf("failure=%v", got)
	}
	if !e.s.player.NeedsCrashSave() {
		t.Fatal("failed snapshot cleared crash retry")
	}
}

func TestMenuDeleteRemovesObjectSaveIdentity(t *testing.T) {
	t.Chdir(t.TempDir())
	e := newRecoveryEnv(t)
	if err := db.SaveObjectSnapshot(e.store, "Recoverer", 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := e.s.confirmDelete("yes"); err != nil {
		t.Fatal(err)
	}
	saved, err := e.store.GetObjectSave("Recoverer")
	if err != nil || saved != nil {
		t.Fatalf("menu delete left identity: %+v %v", saved, err)
	}
}
