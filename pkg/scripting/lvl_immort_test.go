package scripting

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestLVLImmortLuaExportUsesCanonicalCValue(t *testing.T) {
	const cLVLImmort = 31 // src/structs.h:620; independent C expectation

	engine := NewEngine(t.TempDir(), nil)
	defer engine.Close()

	value := engine.LState().GetGlobal("LVL_IMMORT")
	got, ok := value.(lua.LNumber)
	if !ok {
		t.Fatalf("LVL_IMMORT global has type %T, want lua.LNumber", value)
	}
	if got != lua.LNumber(cLVLImmort) {
		t.Fatalf("Lua LVL_IMMORT = %v, want independent C value %d", got, cLVLImmort)
	}
}
