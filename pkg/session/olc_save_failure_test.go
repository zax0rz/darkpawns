package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

func TestOLCSaveAllBoundedFailure(t *testing.T) {
	for _, tc := range []struct {
		kind            olc.Kind
		ext, diagnostic string
	}{
		{olc.KindRoom, "wld", "SYSERR: OLC: Cannot open room file!"},
		{olc.KindObject, "obj", "SYSERR: OLC: Cannot open objects file!"},
		{olc.KindZone, "zon", "SYSERR: OLC: zedit_save_to_disk:  Can't write zone 30."},
		{olc.KindMob, "mob", "SYSERR: OLC: Cannot open mob file!"},
		{olc.KindShop, "shp", "SYSERR: OLC: Cannot open shop file!"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			w, m, s := olcSaveAllFixture(t)
			watch := makeCommandTestSession(t, m, "Failwatch", 31, 3001)
			watch.player.SetPlrFlag(game.PrfLog1, true)
			below := makeCommandTestSession(t, m, "Failbelow", 30, 3001)
			below.player.SetPlrFlag(game.PrfLog1, true)
			below.player.SetPlrFlag(game.PrfLog2, true)
			for _, observer := range []*Session{watch, below} {
				registerTestSession(t, m, observer, observer.player.Name)
			}
			earlier := olc.KindRoom
			if earlier == tc.kind {
				earlier = olc.KindShop
			}
			for _, kind := range []olc.Kind{olc.KindRoom, olc.KindObject, olc.KindZone, olc.KindMob, olc.KindShop} {
				if kind != earlier && kind != tc.kind {
					olcSaveList.Mark(kind, 30)
				}
			}
			olcSaveList.Mark(tc.kind, 30)
			olcSaveList.Mark(earlier, 30)
			before := olcSaveList.Ordered()
			parent := filepath.Join(w.WorldPath, tc.ext)
			if err := os.Remove(parent); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(parent, []byte("parent obstruction"), 0o600); err != nil {
				t.Fatal(err)
			}
			file := captureMudlogFile(t)
			calls := 0
			want := olcSaveLabel(earlier) + " saved for zone 30.\r\n" + olcSaveLabel(tc.kind) + " saved for zone 30.\r\n"
			var output string
			game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: tc.diagnostic, onError: func() {
				calls++
				// Let a compiling retry mutant terminate after proving its second
				// attempt. The control fails assertions, never a hung C-shaped loop.
				if calls > 1 {
					t.Error("failing head retried instead of returning")
					if err := os.Remove(parent); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(parent, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				output += strings.Join(drainSessionText(t, s), "")
				if output != want {
					t.Errorf("failure must follow head acknowledgement: %q", output)
				}
				if !olcSaveList.Dirty(tc.kind, 30) {
					t.Error("failure producer cleared failing marker")
				}
				lock := zoneSaveLock(30)
				if !lock.TryLock() {
					t.Error("error producer holds zone save lock")
				} else {
					lock.Unlock()
				}
			}})
			if err := ExecuteCommand(s, "olc", []string{"save"}); err != nil {
				t.Fatal(err)
			}
			output += strings.Join(drainSessionText(t, s), "")
			if output != want || calls != 1 {
				t.Fatalf("failure attempts/output: calls=%d output=%q", calls, output)
			}
			if strings.Contains(file.String(), "saves all") || !strings.Contains(file.String(), tc.diagnostic+"\n") {
				t.Fatalf("wrong failure log=%q", file.String())
			}
			if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+tc.diagnostic+" ]\r\n" {
				t.Fatalf("error observer=%q", got)
			}
			if len(below.send) != 0 {
				t.Fatal("error threshold leak")
			}
			remaining := olcSaveList.Ordered()
			if len(remaining) != len(before)-1 {
				t.Fatalf("failed head/later markers lost: %v", remaining)
			}
			for i, entry := range remaining {
				if entry != before[i+1] {
					t.Fatalf("failure changed list order: %v", remaining)
				}
			}
			ext := map[olc.Kind]string{olc.KindRoom: "wld", olc.KindObject: "obj", olc.KindZone: "zon", olc.KindMob: "mob", olc.KindShop: "shp"}
			if _, err := os.Stat(filepath.Join(w.WorldPath, ext[earlier], "30."+ext[earlier])); err != nil {
				t.Fatalf("earlier successful file missing: %v", err)
			}
			for _, entry := range before[2:] {
				if _, err := os.Stat(filepath.Join(w.WorldPath, ext[entry.Kind], "30."+ext[entry.Kind])); !os.IsNotExist(err) {
					t.Fatalf("later entry written after failure: %v: %v", entry, err)
				}
			}
			// Repairing the parent and retrying resumes the retained list normally.
			if err := os.Remove(parent); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(parent, 0o755); err != nil {
				t.Fatal(err)
			}
			file.Reset()
			if err := ExecuteCommand(s, "olc", []string{"save"}); err != nil {
				t.Fatal(err)
			}
			if len(olcSaveList.Ordered()) != 0 || !strings.Contains(file.String(), "OLC: Saveactor saves all\n") {
				t.Fatal("repaired retry did not finish retained work")
			}
		})
	}
}

// The retry divergence covers a failed writer, but does not pretend that a
// later atomic rename error is C's fopen error. Existing diagnostic boundaries
// remain narrow (R5f), and later dirty entries remain untouched in either case.
func TestOLCSaveAllAtomicFailure(t *testing.T) {
	w, m, s := olcSaveAllFixture(t)
	watch := makeCommandTestSession(t, m, "Atomicwatch", 40, 3001)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	watch.player.SetPlrFlag(game.PrfLog2, true)
	registerTestSession(t, m, watch, watch.player.Name)
	olcSaveList.Mark(olc.KindObject, 30)
	olcSaveList.Mark(olc.KindRoom, 30)
	if err := os.Mkdir(filepath.Join(w.WorldPath, "wld", "30.wld"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := captureMudlogFile(t)
	if err := ExecuteCommand(s, "olc", []string{"save"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(drainSessionText(t, s), ""); got != "Rooms saved for zone 30.\r\n" {
		t.Fatalf("atomic failure output = %q", got)
	}
	if file.Len() != 0 || len(watch.send) != 0 {
		t.Fatal("atomic rename failure invented C open diagnostic or success")
	}
	if len(olcSaveList.Ordered()) != 2 || !olcSaveList.Dirty(olc.KindRoom, 30) {
		t.Fatal("atomic failure lost markers")
	}
	if _, err := os.Stat(filepath.Join(w.WorldPath, "obj", "30.obj")); !os.IsNotExist(err) {
		t.Fatal("atomic failure saved a later entry")
	}
}
