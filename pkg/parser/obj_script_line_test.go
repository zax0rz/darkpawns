package parser

import (
	"testing"
)

// TestParseObjectScriptLine proves the S script-attach line parses from one
// line — "S <name> <flags>" — exactly as C's parse_object reads it with
// sscanf(line + 1, " %s %d", ...) from the same get_line (db.c:1463-1468).
// The port's earlier two-line read (a bare "S" then the next line) never
// matched the world's object files, so portals and other scripted objects
// booted with no script at all.
func TestParseObjectScriptLine(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "script.obj", `#19593
portal golden~
a shimmering golden portal~
A shimmering portal of golden light hovers in the middle of the room.~
~
1 0 0 0 0 0 0 0 0
0 0 2 0
0 8008 0.00
S portal.lua 2
#19594
plain thing~
a plain thing~
A plain thing.~
~
1 0 0 0 0 0 0 0 0
0 0 0 0
0 0 0.00
$
`)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objs))
	}
	if objs[0].ScriptName != "portal.lua" || objs[0].LuaFunctions != 2 {
		t.Fatalf("script attach = (%q, %d), want (portal.lua, 2)", objs[0].ScriptName, objs[0].LuaFunctions)
	}
	if objs[1].ScriptName != "" || objs[1].LuaFunctions != 0 {
		t.Fatalf("unscripted object parsed a script: (%q, %d)", objs[1].ScriptName, objs[1].LuaFunctions)
	}
}
