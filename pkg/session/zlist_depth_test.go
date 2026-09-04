package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZlistRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["zlist"]
	if !ok {
		t.Fatal("zlist command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("zlist gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("zlist")
	if !ok {
		t.Fatal("zlist command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("zlist registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestCmdZlistMatchesCFixedReadBound(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1204, Name: "Zlist test room", Zone: 12}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Zlistadmin", LVL_IMMORT, 1204)

	worldPath := filepath.Join(t.TempDir(), "world")
	zoneDir := filepath.Join(worldPath, "zon")
	if err := os.MkdirAll(zoneDir, 0o750); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	w.WorldPath = worldPath
	data := strings.Repeat("0123456789abcdefghijklmnopqrstuvwxyz\n", 300)
	if len(data) <= zlistReadLimit {
		t.Fatalf("test data length = %d, want more than C read limit %d", len(data), zlistReadLimit)
	}
	if err := os.WriteFile(filepath.Join(zoneDir, "12.zon"), []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := cmdZlist(s, []string{"12", "ignored", "words"}); err != nil {
		t.Fatalf("cmdZlist: %v", err)
	}
	defer drainSendChannel(t, s)
	if !s.IsPaging() {
		t.Fatal("zlist did not enter the pager for oversized file")
	}
	var paged strings.Builder
	for _, page := range s.pagerPages {
		paged.Write(page)
	}
	if got, want := paged.String(), data[:zlistReadLimit]; got != want {
		t.Fatalf("paged file length/content = %d/%q, want %d-byte C prefix", len(got), got[len(got)-40:], len(want))
	}
}
