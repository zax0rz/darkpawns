package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func writeZonFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return p
}

// basic single zone with M, O, E, G, P, D commands
func TestParseZonFile_BasicZone(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#100
Test Zone~
300 15 1
M 0 1 3 100
O 0 500 2 100
E 0 501 1 17
G 0 502 1
P 0 503 1 504
D 0 100 1 1
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}

	if zone.Number != 100 {
		t.Errorf("number: expected 100, got %d", zone.Number)
	}
	if zone.Name != "Test Zone" {
		t.Errorf("name: expected 'Test Zone', got %q", zone.Name)
	}
	if zone.TopRoom != 300 {
		t.Errorf("toproom: expected 300, got %d", zone.TopRoom)
	}
	if zone.Lifespan != 15 {
		t.Errorf("lifespan: expected 15, got %d", zone.Lifespan)
	}
	if zone.ResetMode != 1 {
		t.Errorf("resetmode: expected 1, got %d", zone.ResetMode)
	}
	if len(zone.Commands) != 6 {
		t.Fatalf("commands: expected 6 (S terminates parsing), got %d", len(zone.Commands))
	}

	// Check M command
	m := zone.Commands[0]
	if m.Command != "M" || m.IfFlag != 0 || m.Arg1 != 1 || m.Arg2 != 3 || m.Arg3 != 100 {
		t.Errorf("M command: got %+v", m)
	}

	// Check O command
	o := zone.Commands[1]
	if o.Command != "O" || o.IfFlag != 0 || o.Arg1 != 500 || o.Arg2 != 2 || o.Arg3 != 100 {
		t.Errorf("O command: got %+v", o)
	}

	// Check E command
	e := zone.Commands[2]
	if e.Command != "E" || e.IfFlag != 0 || e.Arg1 != 501 || e.Arg2 != 1 || e.Arg3 != 17 {
		t.Errorf("E command: got %+v", e)
	}

	// Check G command
	g := zone.Commands[3]
	if g.Command != "G" || g.IfFlag != 0 || g.Arg1 != 502 || g.Arg2 != 1 || g.Arg3 != 0 {
		t.Errorf("G command: got %+v", g)
	}

	// Check P command
	p := zone.Commands[4]
	if p.Command != "P" || p.IfFlag != 0 || p.Arg1 != 503 || p.Arg2 != 1 || p.Arg3 != 504 {
		t.Errorf("P command: got %+v", p)
	}

	// Check D command
	d := zone.Commands[5]
	if d.Command != "D" || d.IfFlag != 0 || d.Arg1 != 100 || d.Arg2 != 1 || d.Arg3 != 1 {
		t.Errorf("D command: got %+v", d)
	}
}

// zone with all command types: M, O, G, E, P, D, L, R
func TestParseZonFile_AllCommandTypes(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#200
All Commands Zone~
400 10 2
M 1 10 5 300
O 0 20 2 400
G 1 30 1
E 0 40 1 3
P 1 50 1 60
D 0 300 2 1
L 1 300 2 7
R 0 400 10 1
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}

	if len(zone.Commands) != 8 {
		t.Fatalf("expected 8 commands (S is not included), got %d", len(zone.Commands))
	}

	// M
	if zone.Commands[0].Command != "M" || zone.Commands[0].IfFlag != 1 || zone.Commands[0].Arg1 != 10 {
		t.Errorf("M command: got %+v", zone.Commands[0])
	}
	// O
	if zone.Commands[1].Command != "O" || zone.Commands[1].IfFlag != 0 || zone.Commands[1].Arg1 != 20 {
		t.Errorf("O command: got %+v", zone.Commands[1])
	}
	// G — only has IfFlag, Arg1, Arg2 (no Arg3)
	if zone.Commands[2].Command != "G" || zone.Commands[2].IfFlag != 1 || zone.Commands[2].Arg1 != 30 || zone.Commands[2].Arg2 != 1 {
		t.Errorf("G command: got %+v", zone.Commands[2])
	}
	// E
	if zone.Commands[3].Command != "E" || zone.Commands[3].IfFlag != 0 || zone.Commands[3].Arg1 != 40 || zone.Commands[3].Arg3 != 3 {
		t.Errorf("E command: got %+v", zone.Commands[3])
	}
	// P
	if zone.Commands[4].Command != "P" || zone.Commands[4].IfFlag != 1 || zone.Commands[4].Arg1 != 50 || zone.Commands[4].Arg3 != 60 {
		t.Errorf("P command: got %+v", zone.Commands[4])
	}
	// D
	if zone.Commands[5].Command != "D" || zone.Commands[5].IfFlag != 0 || zone.Commands[5].Arg1 != 300 || zone.Commands[5].Arg2 != 2 || zone.Commands[5].Arg3 != 1 {
		t.Errorf("D command: got %+v", zone.Commands[5])
	}
	// L
	if zone.Commands[6].Command != "L" || zone.Commands[6].IfFlag != 1 || zone.Commands[6].Arg1 != 300 || zone.Commands[6].Arg2 != 2 || zone.Commands[6].Arg3 != 7 {
		t.Errorf("L command: got %+v", zone.Commands[6])
	}
	// R
	if zone.Commands[7].Command != "R" || zone.Commands[7].IfFlag != 0 || zone.Commands[7].Arg1 != 400 || zone.Commands[7].Arg2 != 10 || zone.Commands[7].Arg3 != 1 {
		t.Errorf("R command: got %+v", zone.Commands[7])
	}
}

// zone with comments and blank lines
func TestParseZonFile_CommentsAndBlankLines(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#300
Zone With Comments~
100 30 0
* This is a comment
M 0 1 3 100

* Another comment
O 0 500 2 100

S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}

	if zone.Number != 300 {
		t.Errorf("number: expected 300, got %d", zone.Number)
	}
	if len(zone.Commands) != 2 {
		t.Fatalf("commands: expected 2, got %d", len(zone.Commands))
	}
	if zone.Commands[0].Command != "M" {
		t.Errorf("first command: expected M, got %s", zone.Commands[0].Command)
	}
	if zone.Commands[1].Command != "O" {
		t.Errorf("second command: expected O, got %s", zone.Commands[1].Command)
	}
}

// zone without any commands (immediate S)
func TestParseZonFile_EmptyCommands(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#400
Empty Zone~
0 0 0
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}

	if zone.Number != 400 {
		t.Errorf("number: expected 400, got %d", zone.Number)
	}
	if len(zone.Commands) != 0 {
		t.Errorf("commands: expected 0, got %d", len(zone.Commands))
	}
}

// reset message handling — the name field ends with a ~ which is stripped
func TestParseZonFile_NameTildeStripped(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#500
Room of Doom~
50 5 1
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}

	if zone.Name != "Room of Doom" {
		t.Errorf("name: expected 'Room of Doom', got %q", zone.Name)
	}
	// trailing whitespace in name should not introduce extra characters
	if zone.Name != "Room of Doom" {
		t.Errorf("name has extra trailing content: %q", zone.Name)
	}
}

// zone number from line starting with #
func TestParseZonFile_ZoneNumberParsing(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#999
High Number Zone~
0 0 0
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}
	if zone.Number != 999 {
		t.Errorf("number: expected 999, got %d", zone.Number)
	}
}

// zone number 0 edge case
func TestParseZonFile_ZoneNumberZero(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#0
Zero Zone~
0 0 0
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}
	if zone.Number != 0 {
		t.Errorf("number: expected 0, got %d", zone.Number)
	}
}

// path traversal rejection
func TestParseZonFile_PathTraversal(t *testing.T) {
	_, err := ParseZonFile("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

// missing zone number line
func TestParseZonFile_MissingZoneNumber(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `Name~
0 0 0
S
$`)

	_, err := ParseZonFile(f)
	if err == nil {
		t.Fatal("expected error for missing zone number, got nil")
	}
}

// missing zone name line
func TestParseZonFile_MissingZoneName(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#100
`)

	_, err := ParseZonFile(f)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

// truncated file after zone number
func TestParseZonFile_Truncated(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#100
`)
	_, err := ParseZonFile(f)
	if err == nil {
		t.Fatal("expected error for truncated file, got nil")
	}
}

// empty file
func TestParseZonFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", "")

	_, err := ParseZonFile(f)
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

// non-existent file
func TestParseZonFile_FileNotFound(t *testing.T) {
	_, err := ParseZonFile("/nonexistent/path.zon")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

// short command lines with fewer fields than expected (should use defaults/zeros)
func TestParseZonFile_ShortCommand(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#600
Short Commands~
0 0 0
M
O
G
E
P
D
L
R
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}

	// All single-letter commands should have zero-valued args
	// S terminates parsing so it's not included in Commands
	for i, cmd := range zone.Commands {
		if cmd.IfFlag != 0 || cmd.Arg1 != 0 || cmd.Arg2 != 0 || cmd.Arg3 != 0 {
			t.Errorf("command %d (%s): expected all zeros, got %+v", i, cmd.Command, cmd)
		}
	}
}

// multiple zone files parsed together
func TestParseAllZonFiles(t *testing.T) {
	tmpDir := t.TempDir()

	c1 := `#100
First Zone~
10 5 1
M 0 1 3 100
S
$`
	c2 := `#200
Second Zone~
20 10 2
O 0 500 2 200
S
$`

	_ = writeZonFile(t, tmpDir, "a.zon", c1)
	_ = writeZonFile(t, tmpDir, "b.zon", c2)

	zones, err := ParseAllZonFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all zone files: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}
	if zones[0].Number != 100 {
		t.Errorf("first zone number: expected 100, got %d", zones[0].Number)
	}
	if zones[1].Number != 200 {
		t.Errorf("second zone number: expected 200, got %d", zones[1].Number)
	}
}

// ParseAllZonFiles ignores non-.zon files and directories
func TestParseAllZonFiles_IgnoresNonZon(t *testing.T) {
	tmpDir := t.TempDir()

	_ = writeZonFile(t, tmpDir, "test.zon", `#100
Ignore Zone~
0 0 0
S
$`)
	_ = writeZonFile(t, tmpDir, "readme.txt", "not a zone file")
	_ = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	zones, err := ParseAllZonFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all zone files: %v", err)
	}
	if len(zones) != 1 {
		t.Errorf("expected 1 zone, got %d", len(zones))
	}
}

// command 'L' (lock) parsing
func TestParseZonFile_LockCommand(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#700
Lock Test~
50 5 1
L 0 300 2 7
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}
	if len(zone.Commands) != 1 {
		t.Fatalf("expected 1 command (S terminates), got %d", len(zone.Commands))
	}
	l := zone.Commands[0]
	if l.Command != "L" || l.IfFlag != 0 || l.Arg1 != 300 || l.Arg2 != 2 || l.Arg3 != 7 {
		t.Errorf("L command: got %+v", l)
	}
}

// command 'R' (remove) parsing
func TestParseZonFile_RemoveCommand(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeZonFile(t, tmpDir, "test.zon", `#800
Remove Test~
60 5 1
R 0 400 10 1
S
$`)

	zone, err := ParseZonFile(f)
	if err != nil {
		t.Fatalf("parse zon file: %v", err)
	}
	if len(zone.Commands) != 1 {
		t.Fatalf("expected 1 command (S terminates), got %d", len(zone.Commands))
	}
	r := zone.Commands[0]
	if r.Command != "R" || r.IfFlag != 0 || r.Arg1 != 400 || r.Arg2 != 10 || r.Arg3 != 1 {
		t.Errorf("R command: got %+v", r)
	}
}
