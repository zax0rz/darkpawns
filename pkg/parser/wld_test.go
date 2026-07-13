package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
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

	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
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

func TestParseWldFilePreservesDescriptionIndentation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "indent.wld")
	content := "#100\nIndented Room~\n   First line.\n Second line.\n~\n1 0 0 0 0 0\nS\n$\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	rooms, err := ParseWldFile(path)
	if err != nil {
		t.Fatalf("ParseWldFile: %v", err)
	}
	if len(rooms) != 1 || rooms[0].Description != "   First line.\n Second line.\n" {
		t.Fatalf("description indentation was not preserved: %#v", rooms)
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

	_ = os.WriteFile(filepath.Join(tmpDir, "1.wld"), []byte(content1), 0o644)
	_ = os.WriteFile(filepath.Join(tmpDir, "2.wld"), []byte(content2), 0o644)

	rooms, err := ParseAllWldFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all wld files: %v", err)
	}

	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms total, got %d", len(rooms))
	}
}

func TestParseWldFile_InvalidSectorRejected(t *testing.T) {
	tmpDir := t.TempDir()

	valid := `#100
Valid Room~
Desc.
~
1 0 0 0 0 15
S
$
`
	invalidHigh := `#101
Bad Room~
Desc.
~
1 0 0 0 0 16
S
$
`
	invalidNegative := `#102
Bad Room~
Desc.
~
1 0 0 0 0 -1
S
$
`
	invalidNaN := `#103
Bad Room~
Desc.
~
1 0 0 0 0 abc
S
$
`

	for _, tc := range []struct {
		name    string
		content string
	}{
		{"sector too high", invalidHigh},
		{"sector negative", invalidNegative},
		{"sector not a number", invalidNaN},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tc.name+".wld")
			if err := os.WriteFile(testFile, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("write test file: %v", err)
			}
			if _, err := ParseWldFile(testFile); err == nil {
				t.Error("expected parse error for invalid sector, got nil")
			}
		})
	}

	// Valid maximum sector should parse successfully.
	validFile := filepath.Join(tmpDir, "valid.wld")
	if err := os.WriteFile(validFile, []byte(valid), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	rooms, err := ParseWldFile(validFile)
	if err != nil {
		t.Fatalf("expected valid sector to parse, got error: %v", err)
	}
	if len(rooms) != 1 || rooms[0].Sector != 15 {
		t.Errorf("expected sector 15, got %v", rooms)
	}
}

func TestRoom_HasFlagOutOfBounds(t *testing.T) {
	room := &Room{
		Flags: []string{"0", "0", "0", "0"},
	}

	// Should return false and not panic
	if room.HasFlag(64) {
		t.Error("Expected HasFlag(64) to be false")
	}
	if room.HasFlag(100) {
		t.Error("Expected HasFlag(100) to be false")
	}
	if room.HasFlag(-1) {
		t.Error("Expected HasFlag(-1) to be false")
	}
}

func TestReadTildeString_TrailingWhitespace(t *testing.T) {
	input := "hello ~  "
	scanner := bufio.NewScanner(strings.NewReader(input))

	got, err := readTildeString(scanner)
	if err != nil {
		t.Fatalf("readTildeString returned error: %v", err)
	}
	if got != "hello " {
		t.Errorf("expected %q, got %q", "hello ", got)
	}
}

func TestExitInfoDoorCapabilityAndRuntimeStateAreIndependent(t *testing.T) {
	tests := []struct {
		doorState int
		wantInfo  int
	}{
		{doorState: 0, wantInfo: 0},
		{doorState: 1, wantInfo: ExitIsDoor},
		{doorState: 2, wantInfo: ExitIsDoor | ExitPickproof},
	}
	for _, tt := range tests {
		if got := ExitInfoFromDoorState(tt.doorState); got != tt.wantInfo {
			t.Errorf("ExitInfoFromDoorState(%d) = %d, want %d", tt.doorState, got, tt.wantInfo)
		}
	}

	capabilities := ExitIsDoor | ExitPickproof
	if got := ApplyDoorReset(capabilities, 1); got != capabilities|ExitClosed {
		t.Fatalf("closed reset = %d, want %d", got, capabilities|ExitClosed)
	}
	locked := ApplyDoorReset(capabilities, 2)
	if locked != capabilities|ExitClosed|ExitLocked {
		t.Fatalf("locked reset = %d, want %d", locked, capabilities|ExitClosed|ExitLocked)
	}
	if got := ApplyDoorReset(locked, 0); got != capabilities {
		t.Fatalf("open reset = %d, want preserved capabilities %d", got, capabilities)
	}
	if got := LegacyDoorState(locked); got != 2 {
		t.Fatalf("LegacyDoorState(locked) = %d, want 2", got)
	}
}
