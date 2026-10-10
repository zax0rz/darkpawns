package session

// crash_load_mudlog_test.go — R5h proofs for Crash_load's four live entry
// producers (src/objsave.c:474-541, DP-1404).
//
// The stored rent code is the port's object_saves.kind. Its two pre-existing
// values already coincide with C's RENT_CRASH (1) and RENT_RENTED (2), and the
// load-time header rewrite to RENT_CRASH is db.ObjectSaveLoaded; DP-1404 only
// added RENT_CRYO (3) for the PLR_NODELETE quit and the producer. Each case
// below drives the real menu entry (MsgCharInput "1" -> handleMenuChoice ->
// enterReturningPlayer) or the real creation flow, and asserts the exact bytes
// at an independent immortal, the level/type/writing filters, and the file
// flag.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// crashLoadEntryEnv builds the stack a menu entry runs on: a real SQLite store
// holding the returning character, a world with the mortal start room and the
// observer's room, the wired object-save seam, and a fresh menu-stage session
// whose player is the stored record.
func crashLoadEntryEnv(t *testing.T, name string) (*Manager, *db.DB, *Session) {
	t.Helper()
	t.Setenv("JWT_SECRET", "crash-load-entry-secret-at-least-32-bytes")
	store := entryDatabase(t)
	record := entrySeed(t, store, name)
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Watch Room", Zone: 1},
		{VNum: game.MortalStartRoom, Name: "The Temple", Zone: 80},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	m := newTestManager(t, w, store)
	m.WirePlayerSaver(w)
	p, err := db.RecordToPlayer(record, w)
	if err != nil {
		t.Fatalf("RecordToPlayer: %v", err)
	}
	s := makeCharSession(t, m)
	s.player = p
	s.playerName = name
	s.authenticated = true
	s.menuActive = true
	s.menuStage = "menu"
	return m, store, s
}

// The four live arms. C selects them from rent.rentcode, where a missing crash
// file is the no-file arm (src/objsave.c:474-489, :516-541). All are NRM /
// MAX(LVL_IMMORT, invis) / file TRUE, and the entry rewrites the header to
// RENT_CRASH (:659-663) without creating a file that was not there.
func TestCrashLoadEntryProducers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    int
		present bool
		line    string
	}{
		{"no equipment", 0, false, "Enterer entering game with no equipment."},
		{"un-renting", rentRented, true, "Enterer un-renting and entering game."},
		{"crash-saved", rentCrash, true, "Enterer retrieving crash-saved items and entering game."},
		{"cryo", rentCryo, true, "Enterer un-cryo'ing and entering game."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, store, s := crashLoadEntryEnv(t, "Enterer")
			observer := mudlogObserver(t, m, "Entrywatch", game.LVL_GOD, game.PrfLog2)
			file := captureMudlogFile(t)
			if tc.present {
				if err := db.SaveObjectSnapshot(store, "Enterer", tc.kind, []byte(`[]`), []byte(`[]`)); err != nil {
					t.Fatal(err)
				}
			}
			if err := entryInput(s, "1"); err != nil {
				t.Fatalf("menu entry: %v", err)
			}
			want := "[ " + tc.line + " ]\r\n"
			if got := strings.Join(drainSessionText(t, observer), ""); got != want {
				t.Fatalf("observer saw %q, want %q", got, want)
			}
			if !strings.Contains(file.String(), tc.line) {
				t.Fatalf("file flag: %q lacks %q", file.String(), tc.line)
			}
			snapshot, err := store.GetObjectSave("Enterer")
			if err != nil {
				t.Fatal(err)
			}
			if !tc.present {
				if snapshot != nil {
					t.Fatalf("entry alone created an object save: %+v", snapshot)
				}
				return
			}
			if snapshot == nil || snapshot.Kind != rentCrash {
				t.Fatalf("entry header rewrite: %+v", snapshot)
			}
		})
	}
}

// The audience is NRM / MAX(LVL_IMMORT, invis): a below-immortal, a
// brief-syslog (PRF_LOG1 only) and a writing immortal all see nothing, while
// the file write still happens (utils.c:258-270).
func TestCrashLoadEntryAudienceFilters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(*game.Player)
	}{
		{"below immortal", func(p *game.Player) { p.SetLevel(game.LVL_IMMORT - 1) }},
		{"brief syslog", func(p *game.Player) { p.SetPlrFlag(game.PrfLog2, false) }},
		{"writing", func(p *game.Player) { p.SetPlrFlag(game.PlrWriting, true) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, store, s := crashLoadEntryEnv(t, "Enterer")
			observer := mudlogObserver(t, m, "Entrywatch", game.LVL_GOD, game.PrfLog1, game.PrfLog2)
			file := captureMudlogFile(t)
			tc.prepare(observer.player)
			if err := db.SaveObjectSnapshot(store, "Enterer", rentRented, []byte(`[]`), []byte(`[]`)); err != nil {
				t.Fatal(err)
			}
			if err := entryInput(s, "1"); err != nil {
				t.Fatalf("menu entry: %v", err)
			}
			if got := strings.Join(drainSessionText(t, observer), ""); got != "" {
				t.Fatalf("%s observer saw %q", tc.name, got)
			}
			if !strings.Contains(file.String(), "Enterer un-renting and entering game.") {
				t.Fatalf("%s: the producer did not fire (file %q)", tc.name, file.String())
			}
		})
	}
}

// C's no-file arm fires on every new character's first entry: the crash file
// does not exist yet (src/objsave.c:474-489). Driven through the real creation
// flow, and pinned against do_start's own producer so the order is C's:
// Crash_load first, then advance_level (src/interpreter.c:2184-2214).
func TestCrashLoadFirstEntryProducer(t *testing.T) {
	database := entryDatabase(t)
	t.Setenv("DP_FRESH_MUD", "")
	entrySeed(t, database, "Founder")
	s := entrySession(t, database)
	driveEntryToStats(t, s, "Newhero")
	if err := entryInput(s, "Y"); err != nil {
		t.Fatal(err)
	}
	// The pre-entry stages emit their own producers; watch only the entry.
	observer := mudlogObserver(t, s.manager, "Entrywatch", game.LVL_GOD, game.PrfLog2)
	file := captureMudlogFile(t)
	for _, line := range []string{"", "1"} {
		if err := entryInput(s, line); err != nil {
			t.Fatal(err)
		}
	}
	want := "[ Newhero entering game with no equipment. ]\r\n" +
		"[ Newhero advanced to level 1 ]\r\n"
	if got := strings.Join(drainSessionText(t, observer), ""); got != want {
		t.Fatalf("observer saw %q, want %q", got, want)
	}
	if !strings.Contains(file.String(), "Newhero entering game with no equipment.") {
		t.Fatalf("file flag: %q", file.String())
	}
	if snapshot, err := database.GetObjectSave("Newhero"); err != nil || snapshot != nil {
		t.Fatalf("first entry created an object save: %+v %v", snapshot, err)
	}
}

// C picks the stored code in do_quit: PLR_NODELETE takes Crash_cryosave
// (RENT_CRYO), everyone else Crash_rentsave (RENT_RENTED)
// (src/act.other.c:164-165). This is what makes the un-cryo arm reachable.
func TestQuitStoresLastExitRentCode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		nodelete bool
		kind     int
	}{
		{"rent", false, rentRented},
		{"cryo", true, rentCryo},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := entryDatabase(t)
			m := makeQuitTestManager(t, store)
			m.WirePlayerSaver(m.world)
			s := makeQuitSession(t, m, 1, "Quitter", 1, 8004)
			if tc.nodelete {
				s.player.SetPlrFlag(game.PlrNODELETE, true)
			}
			if err := ExecuteCommand(s, "quit", nil); err != nil {
				t.Fatalf("quit: %v", err)
			}
			snapshot, err := store.GetObjectSave("Quitter")
			if err != nil {
				t.Fatal(err)
			}
			if snapshot == nil || snapshot.Kind != tc.kind {
				t.Fatalf("stored last-exit code: %+v, want kind %d", snapshot, tc.kind)
			}
		})
	}
}
