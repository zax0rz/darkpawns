package scripting

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// TestBootRunsGlobalsDefault boots the engine on the shipped script tree,
// which is C's lib/scripts byte for byte (DP-1333). C's boot_lua loads
// globals.lua and calls default() (scripts.c:1711-1714), which defines the
// script constants and dofile()s scripts/mob/no_move.lua.
func TestBootRunsGlobalsDefault(t *testing.T) {
	e := NewEngine("../../lib/world/scripts", nil)
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	L := e.l
	if got := L.GetGlobal("LVL_IMPL"); got != lua.LNumber(40) {
		t.Fatalf("LVL_IMPL = %v, want 40 from globals.lua default()", got)
	}
	// The engine no longer loads no_move.lua itself, so this function exists
	// only if default() ran and its dofile resolved C's "scripts/" path.
	if _, ok := L.GetGlobal("no_move").(*lua.LFunction); !ok {
		t.Fatal("no_move is not defined: default()'s dofile(\"scripts/mob/no_move.lua\") did not run")
	}
}
