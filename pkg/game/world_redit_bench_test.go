package game

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// The routine-vs-structural split is measured on the real shipped world
// (lib/world, ~10k rooms across 93 zone files) so the per-door-command cost
// reflects production sizes, not fixture sizes.
var realWorldBench struct {
	once   sync.Once
	parsed *parser.World
	err    error
}

func realWorldForBenchmark(b *testing.B) *parser.World {
	b.Helper()
	realWorldBench.once.Do(func() {
		// The parser's validateWorldPath rejects ".." components, so resolve
		// the repository's lib/world from this file's location instead;
		// filepath.Join cleans the result back to absolute form.
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			realWorldBench.err = fmt.Errorf("runtime.Caller could not locate the benchmark source file")
			return
		}
		realWorldBench.parsed, realWorldBench.err = parser.ParseWorld(filepath.Join(filepath.Dir(thisFile), "..", "..", "lib", "world"))
	})
	if realWorldBench.err != nil {
		b.Fatalf("parse real world: %v", realWorldBench.err)
	}
	return realWorldBench.parsed
}

func benchmarkWorldWithOneExit(b *testing.B) (*World, int, string) {
	b.Helper()
	w, err := NewWorld(realWorldForBenchmark(b))
	if err != nil {
		b.Fatalf("NewWorld: %v", err)
	}
	b.Cleanup(w.StopAITicker)
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, vnum := range w.sortedRoomVNums {
		room := w.rooms[vnum]
		if room == nil {
			continue
		}
		for direction := range room.Exits {
			return w, vnum, direction
		}
	}
	b.Fatal("real world has no exits to benchmark")
	return nil, 0, ""
}

// BenchmarkDoorOperationRoutinePath is the per-call cost every door command
// (open/close/lock), zone-reset D command, and scripted door pays.
func BenchmarkDoorOperationRoutinePath(b *testing.B) {
	w, vnum, direction := benchmarkWorldWithOneExit(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info := parser.ExitIsDoor
		if i%2 == 1 {
			info |= parser.ExitPickproof
		}
		if !w.SetExitInfo(vnum, direction, info) {
			b.Fatal("SetExitInfo failed")
		}
	}
}

// BenchmarkEditedRoomCommitStructuralPath is the cost of a REDIT save/new-room
// publication: the parsed-definition copy, rooms-map rebuild, and sorted-vnum
// maintenance that routine mutations must not pay.
func BenchmarkEditedRoomCommitStructuralPath(b *testing.B) {
	w, vnum, _ := benchmarkWorldWithOneExit(b)
	room, ok := w.SnapshotRoom(vnum)
	if !ok {
		b.Fatalf("snapshot room %d missing", vnum)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		room.Name = "Bench Commit"
		if !w.CommitEditedRoom(room) {
			b.Fatal("CommitEditedRoom failed")
		}
	}
}
