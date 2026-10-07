package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

func olcSaveAllFixture(t *testing.T) (*game.World, *Manager, *Session) {
	t.Helper()
	isolateOLCSaveList(t)
	w := makeZeditTestWorld(t)
	for _, extension := range []string{"wld", "obj", "mob", "shp"} {
		if err := os.Mkdir(filepath.Join(w.WorldPath, extension), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Saveactor", 38, 3000)
	s.player.SetInvisLevel(40)
	s.olcZone = 99
	registerTestSession(t, m, s, s.player.Name)
	return w, m, s
}

func TestOLCSaveAllSuccess(t *testing.T) {
	w, m, s := olcSaveAllFixture(t)
	watch := makeCommandTestSession(t, m, "Savewatch", 31, 3001)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	watch.player.SetPlrFlag(game.PrfLog2, true)
	below := makeCommandTestSession(t, m, "Savebelow", 30, 3001)
	below.player.SetPlrFlag(game.PrfLog1, true)
	below.player.SetPlrFlag(game.PrfLog2, true)
	normal := makeCommandTestSession(t, m, "Savenormal", 40, 3001)
	normal.player.SetPlrFlag(game.PrfLog2, true)
	for _, observer := range []*Session{watch, below, normal} {
		registerTestSession(t, m, observer, observer.player.Name)
	}
	// A pending editor is not an olc-save refusal. Its draft is not committed.
	holder := makeCommandTestSession(t, m, "Saveholder", 40, 3000)
	if _, ok := m.claimObjEdit(3003, holder); !ok {
		t.Fatal("cannot acquire test claim")
	}
	t.Cleanup(func() { m.releaseObjEdit(3003, holder) })
	if err := m.SaveOLCZone(30); err == nil {
		t.Fatal("admin lost its active-claim gate")
	}
	for _, kind := range []olc.Kind{olc.KindRoom, olc.KindZone, olc.KindObject, olc.KindMob, olc.KindShop} {
		olcSaveList.Mark(kind, 30)
	}
	olcSaveList.Mark(olc.KindRoom, 30)
	want := "Shops saved for zone 30.\r\nMobiles saved for zone 30.\r\nObjects saved for zone 30.\r\nZone info saved for zone 30.\r\nRooms saved for zone 30.\r\n"
	file := captureMudlogFile(t)
	payload := "OLC: Saveactor saves all"
	calls := 0
	var output string
	game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: payload, onError: func() {
		calls++
		output += strings.Join(drainSessionText(t, s), "")
		if output != want {
			t.Errorf("acknowledgements at final log = %q", output)
		}
		if len(olcSaveList.Ordered()) != 0 {
			t.Error("final producer before markers drained")
		}
		lock := zoneSaveLock(30)
		if !lock.TryLock() {
			t.Error("final producer holds zone lock")
		} else {
			lock.Unlock()
		}
		for _, ext := range []string{"wld", "zon", "obj", "mob", "shp"} {
			data, err := os.ReadFile(filepath.Join(w.WorldPath, ext, "30."+ext))
			if err != nil || len(data) == 0 {
				t.Errorf("file not saved before final log: %s: %v", ext, err)
			}
		}
	}})
	// C compares the first four characters, after lowercasing and skipping fills.
	if err := ExecuteCommand(s, "olc", []string{"with", "SAVEjunk", "ignored"}); err != nil {
		t.Fatal(err)
	}
	output += strings.Join(drainSessionText(t, s), "")
	if output != want || calls != 1 || !strings.Contains(file.String(), payload+"\n") {
		t.Fatalf("save-all output=%q calls=%d log=%q", output, calls, file.String())
	}
	if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+payload+" ]\r\n" {
		t.Fatalf("CMP/31 observer = %q", got)
	}
	if len(below.send) != 0 || len(normal.send) != 0 {
		t.Fatal("final producer threshold/type leak")
	}
	file.Reset()
	if err := ExecuteCommand(s, "olc", []string{"save"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(drainSessionText(t, s), ""); got != "The database is up to date.\r\n" {
		t.Fatalf("empty save = %q", got)
	}
	if calls != 1 || file.Len() != 0 || len(watch.send) != 0 {
		t.Fatal("empty save emitted final producer")
	}
}

func TestOLCSaveAllOnlyDirtyKinds(t *testing.T) {
	w, _, s := olcSaveAllFixture(t)
	file := captureMudlogFile(t)
	olcSaveList.Mark(olc.KindRoom, 30)
	if err := ExecuteCommand(s, "olc", []string{"save"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(drainSessionText(t, s), ""); got != "Rooms saved for zone 30.\r\n" {
		t.Fatalf("single-kind ack = %q", got)
	}
	for _, ext := range []string{"zon", "obj", "mob", "shp"} {
		if _, err := os.Stat(filepath.Join(w.WorldPath, ext, "30."+ext)); !os.IsNotExist(err) {
			t.Fatalf("unmarked %s was saved: %v", ext, err)
		}
	}
	if !strings.Contains(file.String(), "OLC: Saveactor saves all\n") {
		t.Fatal("single-kind save omitted final producer")
	}
	// Player-switch metadata must not replace the acting PC's identity.
	s.playerName = "Original"
	olcSaveList.Mark(olc.KindRoom, 30)
	file.Reset()
	if err := cmdOlc(s, []string{"save"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(file.String(), "OLC: Saveactor saves all\n") || strings.Contains(file.String(), "OLC: Original") {
		t.Fatal("save-all used original descriptor name")
	}
}

func TestOLCSaveAllConcurrentCommit(t *testing.T) {
	w, m, s := olcSaveAllFixture(t)
	// Exercise the actual command and the web bridge's mark operation, each
	// with the shared zone lock. Latest committed bytes must be saved or dirty.
	olcSaveList.Mark(olc.KindObject, 30)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 10 {
			if err := cmdOlc(s, []string{"save"}); err != nil {
				t.Error(err)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := range 10 {
			lock := zoneSaveLock(30)
			lock.Lock()
			obj, ok := w.SnapshotObj(3003)
			if !ok {
				lock.Unlock()
				t.Error("object missing")
				return
			}
			obj.Keywords = fmt.Sprintf("last-commit-%d", i)
			w.CommitEditedObj(obj)
			m.MarkOLCDirty(olc.KindObject, 30)
			lock.Unlock()
		}
	}()
	wg.Wait()
	data, err := os.ReadFile(filepath.Join(w.WorldPath, "obj", "30.obj"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "last-commit-9~") && !olcSaveList.Dirty(olc.KindObject, 30) {
		t.Fatal("latest commit neither saved nor dirty")
	}
}
