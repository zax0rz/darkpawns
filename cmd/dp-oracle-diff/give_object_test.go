package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

// zone121 is the shape Sol's #2174 note named: the requested mob's M line is
// NOT the zone's last M, so a G appended before the terminal S would attach
// to whatever mob loads last (here mob 8) instead of the requested keeper.
const giveObjectZone = `#121
Nameless~
Version 2
Top 12199
~
M 0 12100 1 12133
G 1 12105 10 -1
M 0 12116 1 12107
M 0 8 1 12100
S
$
`

func writeGiveObjectZone(t *testing.T) string {
	t.Helper()
	world := filepath.Join(t.TempDir(), "world", "zon")
	if err := os.MkdirAll(world, 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(world, "121.zon")
	if err := os.WriteFile(path, []byte(giveObjectZone), 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(world))
}

// The G must land directly after the requested mob's own M line, never
// before the terminal S where it would attach to the zone's last M.
func TestGiveObjectAttachesToRequestedMob(t *testing.T) {
	root := writeGiveObjectZone(t)
	fixtures := []oraclediff.MobObjectFixture{{MobVNum: 12100, ObjectVNum: 8010, MaxExisting: 1, ZoneNumber: 121}}
	if err := applyMobObjectFixtures(filepath.Join(root, "world"), fixtures); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "world", "zon", "121.zon"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	gAt, mobAt, lastMAt := -1, -1, -1
	for i, line := range lines {
		switch {
		case line == "M 0 12100 1 12133":
			mobAt = i
		case line == "G 1 8010 1 -1":
			gAt = i
		case strings.HasPrefix(line, "M 0 "):
			lastMAt = i
		}
	}
	if mobAt < 0 || gAt < 0 {
		t.Fatalf("fixture lost a line; mobAt=%d gAt=%d", mobAt, gAt)
	}
	// The requirement is attachment: the G must fall after the requested
	// mob's M line and before the zone's next M (so it executes while
	// 12100 is the last loaded mob), never before the terminal S where it
	// would attach to the zone's last M (mob 8, line 9 here).
	nextM := lastMAt
	for i, line := range lines {
		if i > mobAt && strings.HasPrefix(line, "M 0 ") {
			nextM = i
			break
		}
	}
	if mobAt >= gAt || gAt >= nextM {
		t.Fatalf("G for mob 12100 at line %d must sit between the mob's M (line %d) and the next M (line %d)", gAt, mobAt, nextM)
	}
}

// Multiple objects for the same mob stack after the M line in fixture order,
// and the authored G that followed the M keeps its own attachment.
func TestGiveObjectStacksAfterMobLine(t *testing.T) {
	root := writeGiveObjectZone(t)
	fixtures := []oraclediff.MobObjectFixture{
		{MobVNum: 12100, ObjectVNum: 8010, MaxExisting: 1, ZoneNumber: 121},
		{MobVNum: 12100, ObjectVNum: 8011, MaxExisting: 2, ZoneNumber: 121},
	}
	if err := applyMobObjectFixtures(filepath.Join(root, "world"), fixtures); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "world", "zon", "121.zon"))
	if err != nil {
		t.Fatal(err)
	}
	first := bytes.Index(data, []byte("M 0 12100 1 12133\n"))
	window := string(data[first : first+90])
	// Authored cluster first, then fixture objects in fixture order, then
	// the next mob's M: everything after the 12100 M and before the 12116 M
	// attaches to 12100.
	want := "M 0 12100 1 12133\nG 1 12105 10 -1\nG 1 8010 1 -1\nG 1 8011 2 -1\nM 0 12116 1 12107\n"
	if !strings.HasPrefix(window, want) {
		t.Fatalf("zone tail = %q, want prefix %q", window, want)
	}
}

// The assertion arm: a mob with no M line in the named zone is refused
// rather than silently misattached.
func TestGiveObjectRejectsMissingMob(t *testing.T) {
	root := writeGiveObjectZone(t)
	fixtures := []oraclediff.MobObjectFixture{{MobVNum: 99999, ObjectVNum: 8010, MaxExisting: 1, ZoneNumber: 121}}
	err := applyMobObjectFixtures(filepath.Join(root, "world"), fixtures)
	if err == nil || !strings.Contains(err.Error(), "no preceding mob fixture") {
		t.Fatalf("expected the missing-mob refusal, got %v", err)
	}
}
