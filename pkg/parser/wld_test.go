package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWldFile(t *testing.T) {
	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.wld")

	content := `#10011
In the Stands~
You are standing high in the stands of a large stadium. Down in the center
of the stadium, you can see a game in progress.
~
100 32768 0 0 0 0
D1
~
~
0 0 10012
D2
~
~
0 0 10017
S
#10012
Another Room~
Another room description here.
~
100 0 0 0 0 0
D3
~
~
0 0 10011
S
$
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	rooms, err := ParseWldFile(testFile)
	if err != nil {
		t.Fatalf("parse wld file: %v", err)
	}

	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}

	// Check first room
	room := rooms[0]
	if room.VNum != 10011 {
		t.Errorf("expected vnum 10011, got %d", room.VNum)
	}
	if room.Name != "In the Stands" {
		t.Errorf("expected name 'In the Stands', got %q", room.Name)
	}
	if room.Zone != 100 {
		t.Errorf("expected zone 100, got %d", room.Zone)
	}
	if len(room.Exits) != 2 {
		t.Errorf("expected 2 exits, got %d", len(room.Exits))
	}

	// Check east exit
	east, ok := room.Exits["east"]
	if !ok {
		t.Error("expected east exit")
	} else {
		if east.ToRoom != 10012 {
			t.Errorf("expected east to_room 10012, got %d", east.ToRoom)
		}
	}
}

func TestParseAllWldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	content1 := `#100
Room One~
Description one.
~
1 0 0 0 0 0
S
$
`
	content2 := `#200
Room Two~
Description two.
~
2 0 0 0 0 0
S
$
`

	os.WriteFile(filepath.Join(tmpDir, "1.wld"), []byte(content1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "2.wld"), []byte(content2), 0644)

	rooms, err := ParseAllWldFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all wld files: %v", err)
	}

	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms total, got %d", len(rooms))
	}
}
