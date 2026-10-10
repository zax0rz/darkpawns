package session

// crash_file_delete_test.go — the extraction-time crash-file delete
// (src/handler.c:1163; Crash_delete_crashfile, src/objsave.c:177-201), ported
// with DP-1404 because the entry producers are wrong without it.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// C deletes the crash file at PC extraction only when its header still holds
// RENT_CRASH, so a rent or cryo file survives for the next Crash_load to log
// while a LOSTEQ quit — which skipped Crash_rentsave — leaves nothing behind.
// The entry's own header rewrite to RENT_CRASH stands in for C's `save`, so both
// cases start from exactly the state C would have on disk.
func TestExtractionDeletesOnlyCrashFiles(t *testing.T) {
	for _, tc := range []struct {
		name    string
		room    int
		command string
		want    int // 0 = no row survives
	}{
		{"losteq quit deletes the crash file", 1001, "reallyquit", 0},
		{"legal quit keeps the rent file", game.MortalStartRoom, "quit", rentRented},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, store, s := crashLoadEntryEnv(t, "Extracted")
			if err := db.SaveObjectSnapshot(store, "Extracted", rentRented, []byte(`[]`), []byte(`[]`)); err != nil {
				t.Fatal(err)
			}
			if err := entryInput(s, "1"); err != nil {
				t.Fatalf("menu entry: %v", err)
			}
			snapshot, err := store.GetObjectSave("Extracted")
			if err != nil || snapshot == nil || snapshot.Kind != rentCrash {
				t.Fatalf("entry header rewrite: %+v %v", snapshot, err)
			}
			s.player.SetRoom(tc.room)
			if err := ExecuteCommand(s, tc.command, nil); err != nil {
				t.Fatalf("%s: %v", tc.command, err)
			}
			m.ExtractPendingChars()
			snapshot, err = store.GetObjectSave("Extracted")
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == 0 {
				if snapshot != nil {
					t.Fatalf("extraction kept a crash file C deletes: %+v", snapshot)
				}
				return
			}
			if snapshot == nil || snapshot.Kind != tc.want {
				t.Fatalf("extraction result: %+v, want kind %d", snapshot, tc.want)
			}
		})
	}
}
