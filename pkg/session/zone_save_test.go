package session

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestAtomicWriteFileDeliversIdenticalBytes pins atomicWriteFile's contract:
// the bytes on disk are exactly what was passed in, the requested permission
// bits are applied, no staging files are left behind, and overwrites replace
// the previous content wholesale.
func TestAtomicWriteFileDeliversIdenticalBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "30.obj")

	want := "#3001\nsword rusty~\na rusty sword~\n$~\n"
	if err := atomicWriteFile(path, []byte(want), 0o666); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o666 {
		t.Fatalf("perm = %o, want 666", fi.Mode().Perm())
	}

	// Overwrite: the old content must vanish completely, never partially.
	want2 := "#3002\nsack leather~\n$~\n"
	if err := atomicWriteFile(path, []byte(want2), 0o666); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want2 {
		t.Fatalf("overwritten content = %q, want %q", got, want2)
	}

	// No staging files may be left behind in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Fatalf("staging file left behind: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Fatalf("dir holds %d files, want exactly the target", len(entries))
	}

	// A missing directory surfaces an error and creates nothing.
	if err := atomicWriteFile(filepath.Join(dir, "nope", "x.obj"), []byte("x"), 0o666); err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
}

// TestZoneSaveLockIsPerZone pins the registry: one shared mutex per zone
// number, distinct zones get distinct mutexes, and the mutex actually
// excludes concurrent holders.
func TestZoneSaveLockIsPerZone(t *testing.T) {
	lockA := zoneSaveLock(30)
	lockB := zoneSaveLock(30)
	if lockA != lockB {
		t.Fatal("same zone returned different locks")
	}
	if zoneSaveLock(30) == zoneSaveLock(31) {
		t.Fatal("different zones returned the same lock")
	}

	const workers = 64
	const increments = 1000
	var total int64
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu := zoneSaveLock(7)
			for j := 0; j < increments; j++ {
				mu.Lock()
				total++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if want := int64(workers * increments); total != want {
		t.Fatalf("total = %d, want %d (lock did not exclude)", total, want)
	}
}

// TestZoneSaveConcurrentSavesStayValid hammers saveOeditZone from multiple
// goroutines: every save must land atomically, so the final file is
// byte-identical to a sequential save and reparses cleanly.
func TestZoneSaveConcurrentSavesStayValid(t *testing.T) {
	w := makeOeditTestWorld(t)
	dir := w.GetParsedWorld().SourceDir
	zone := &parser.Zone{Number: 30, TopRoom: 3099}

	// Hermetic w.r.t. the package-global save list: drop any marker another
	// test left for this zone before starting.
	olcSaveList.Remove(olcKindObject, zone.Number)

	if err := saveOeditZone(w, zone); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "obj", "30.obj")
	baseline, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	const savers = 8
	var wg sync.WaitGroup
	for i := 0; i < savers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := saveOeditZone(w, zone); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, baseline) {
		t.Fatal("concurrent saves produced a file differing from the sequential baseline")
	}
	if _, err := parser.ParseObjFile(path); err != nil {
		t.Fatalf("saved .obj does not reparse: %v", err)
	}

	// The save list must be drained: the zone was persisted.
	if olcSaveList.Dirty(olcKindObject, zone.Number) {
		t.Fatal("object save marker remains after zone save")
	}
}

// TestZoneSaveSaveBlockedByHeldLock pins that saveOeditZone actually takes
// the zone save lock: with the lock held by the test, a concurrent save must
// not reach its snapshot, let alone the disk write.
func TestZoneSaveSaveBlockedByHeldLock(t *testing.T) {
	w := makeOeditTestWorld(t)
	dir := w.GetParsedWorld().SourceDir
	zone := &parser.Zone{Number: 30, TopRoom: 3099}
	path := filepath.Join(dir, "obj", "30.obj")

	mu := zoneSaveLock(zone.Number)
	mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := saveOeditZone(w, zone); err != nil {
			t.Error(err)
		}
	}()
	proceeded := false
	select {
	case <-done:
		proceeded = true
	case <-time.After(200 * time.Millisecond):
	}
	mu.Unlock()
	if proceeded {
		t.Fatal("saveOeditZone proceeded while the zone save lock was held")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("saveOeditZone did not complete after the lock was released")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("zone file missing after save: %v", err)
	}
}

// TestZoneSaveCommitBlockedByHeldLock pins that the working-copy commit path
// takes the same zone save lock: with the lock held by the test, a commit
// must not reach the world write or the dirty-marker set. Together with
// TestZoneSaveSaveBlockedByHeldLock this proves the lost-marker interleaving
// (commit between a save's snapshot and its marker cleanup) is impossible:
// the two critical sections are mutually exclusive.
func TestZoneSaveCommitBlockedByHeldLock(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Lockobj", 35)
	openOedit(t, s, "3010")

	mu := zoneSaveLock(30)
	mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.saveOeditInternallyLocked()
	}()
	proceeded := false
	select {
	case <-done:
		proceeded = true
	case <-time.After(200 * time.Millisecond):
	}
	committed := false
	if _, ok := w.SnapshotObj(3010); ok {
		committed = true
	}
	marked := olcSaveList.Dirty(olcKindObject, 30)
	mu.Unlock()
	if proceeded {
		t.Fatal("commit proceeded while the zone save lock was held")
	}
	if committed {
		t.Fatal("commit wrote the world while the lock was held")
	}
	if marked {
		t.Fatal("commit set the dirty marker while the lock was held")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("commit did not complete after the lock was released")
	}
	if _, ok := w.SnapshotObj(3010); !ok {
		t.Fatal("commit did not land in the world after lock release")
	}
	marked = olcSaveList.Dirty(olcKindObject, 30)
	olcSaveList.Remove(olcKindObject, 30)
	if !marked {
		t.Fatal("commit did not set the dirty marker after lock release")
	}
}

// TestZoneSaveKeepsMarkerForRacingCommit is the end-to-end form of the lost
// save marker regression test: commits racing a zone save must either land in
// the snapshot (and be written) or keep their dirty marker — never be cleared
// unpersisted. It mirrors exactly what saveOeditInternallyLocked does at the
// world level: world commit + marker set held under the zone save lock.
func TestZoneSaveKeepsMarkerForRacingCommit(t *testing.T) {
	w := makeOeditTestWorld(t)
	dir := w.GetParsedWorld().SourceDir
	zone := &parser.Zone{Number: 30, TopRoom: 3099}
	path := filepath.Join(dir, "obj", "30.obj")

	const commits = 32
	var seq atomic.Int64
	lastKW := make(map[int64]string)
	var kwMu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < commits; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu := zoneSaveLock(zone.Number)
			mu.Lock()
			obj, ok := w.SnapshotObj(3001)
			if !ok {
				mu.Unlock()
				t.Error("3001 missing from world")
				return
			}
			my := seq.Add(1)
			obj.Keywords = fmt.Sprintf("race-sword-%d-x", my)
			w.CommitEditedObj(obj)
			olcSaveList.Mark(olcKindObject, zone.Number)
			kwMu.Lock()
			lastKW[my] = obj.Keywords
			kwMu.Unlock()
			mu.Unlock()
		}()
	}
	// Saves racing the commits.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := saveOeditZone(w, zone); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	last := seq.Load()
	kwMu.Lock()
	wantKW := lastKW[last]
	kwMu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	marked := olcSaveList.Dirty(olcKindObject, zone.Number)

	// The last commit (in zone-lock order) is either in the final file or
	// still marked dirty. Anything else is the lost-marker bug.
	if !strings.Contains(string(data), "#3001\n"+wantKW+"~\n") && !marked {
		t.Fatalf("last commit %q neither on disk nor marked dirty: marker lost", wantKW)
	}

	// Leave the global save list as we found it.
	if err := saveOeditZone(w, zone); err != nil {
		t.Fatal(err)
	}
	olcSaveList.Remove(olcKindObject, zone.Number)
}

// TestSaveMeditZoneWritesTheLoadedDirectory pins the disk-path resolution:
// the .mob file must land beside the wld directory (SourceDir/mob), where
// ParseAllMobFiles reads. The pre-fix "../mob" form wrote to <lib>/mob, a
// directory no boot ever reads — silent data loss for builder saves.
func TestSaveMeditZoneWritesTheLoadedDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "mob"), 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := game.NewWorld(&parser.World{
		SourceDir: dir,
		Rooms:     []parser.Room{{VNum: 1001, Name: "R", Zone: 1, Flags: []string{"0", "0", "0", "0"}}},
		Mobs:      []parser.Mob{{VNum: 1101, Keywords: "k", ShortDesc: "s", LongDesc: "l\r\n"}},
		Zones:     []parser.Zone{{Number: 1, TopRoom: 1999}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	w.WorldPath = dir

	if err := saveMeditZone(w, &parser.Zone{Number: 1, TopRoom: 1999}); err != nil {
		t.Fatalf("saveMeditZone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mob", "1.mob")); err != nil {
		t.Errorf("zone file missing at the loaded location (SourceDir/mob/1.mob): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "mob", "1.mob")); err == nil {
		t.Errorf("zone file escaped the world directory (../mob) - the loader never reads there")
	}
}
