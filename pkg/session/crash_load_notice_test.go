package session

// crash_load_notice_test.go — R5h proof for Crash_load's non-ENOENT arm
// (src/objsave.c:477-491): a saved-object load that fails for any reason
// other than "no crash file" shows the player C's NOTICE block verbatim,
// mudlogs the ordinary "entering game with no equipment." line, and still
// enters the game. The store error is injected by wrapping the real SQLite
// store's GetObjectSave.

import (
	"errors"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// failingObjectSaveStore delegates everything to the real store but fails
// GetObjectSave, standing in for C's fopen succeeding and the read failing.
type failingObjectSaveStore struct {
	db.GameStore
	inner *db.DB
}

func (f *failingObjectSaveStore) GetObjectSave(string) (*db.ObjectSave, error) {
	return nil, errors.New("injected object-save read failure")
}

// The failing reader must NOT implement db.ObjectSaveLoaded's writer half
// differently: it embeds the real store, so the header rewrite keeps its
// missing-row no-op semantics.
func TestCrashLoadNoticeOnStoreFailure(t *testing.T) {
	m, store, s := crashLoadEntryEnv(t, "Enterer")
	observer := mudlogObserver(t, m, "Entrywatch", game.LVL_GOD, game.PrfLog2)
	file := captureMudlogFile(t)
	if err := db.SaveObjectSnapshot(store, "Enterer", rentRented, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	m.db = &failingObjectSaveStore{GameStore: store, inner: store}

	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry aborted on object-save failure: %v (C enters with no equipment, objsave.c:488-490)", err)
	}

	notice := "\r\n********************* NOTICE *********************\r\n" +
		"There was a problem loading your objects from disk.\r\n" +
		"Contact a God for assistance.\r\n"
	if got := strings.Join(drainSessionText(t, s), ""); !strings.Contains(got, notice) {
		t.Fatalf("player bytes lack C's NOTICE block:\n%q\nwant substring %q", got, notice)
	}
	wantLog := "[ Enterer entering game with no equipment. ]\r\n"
	if got := strings.Join(drainSessionText(t, observer), ""); got != wantLog {
		t.Fatalf("observer saw %q, want %q (the no-equipment mudlog fires on the failure arm too, objsave.c:488-489)", got, wantLog)
	}
	if !strings.Contains(file.String(), "Enterer entering game with no equipment.") {
		t.Fatalf("file flag: %q lacks the no-equipment line", file.String())
	}
}
