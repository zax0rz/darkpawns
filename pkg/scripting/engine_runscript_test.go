package scripting

import (
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// TestRunScriptClearsPerRunGlobals verifies that RunScript does not leak
// context globals or trigger functions between executions.
func TestRunScriptClearsPerRunGlobals(t *testing.T) {
	dir := t.TempDir()

	first := filepath.Join(dir, "first.lua")
	firstSrc := `function ongive()
	if argument ~= "from-first" then return FALSE end
	if room == nil or room.vnum ~= 123 then return FALSE end
	return TRUE
end
`
	if err := os.WriteFile(first, []byte(firstSrc), 0644); err != nil {
		t.Fatalf("write first.lua: %v", err)
	}

	second := filepath.Join(dir, "second.lua")
	secondSrc := `leak = "no"
if argument ~= nil then leak = "yes" end
if room ~= nil then leak = "yes" end
`
	if err := os.WriteFile(second, []byte(secondSrc), 0644); err != nil {
		t.Fatalf("write second.lua: %v", err)
	}

	engine := NewEngine(dir, nil)
	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	ctx1 := &ScriptContext{Argument: "from-first", RoomVNum: 123}
	handled1, err := engine.RunScript(ctx1, "first.lua", "ongive")
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}
	if !handled1 {
		t.Fatalf("first run expected handled=true, got false")
	}

	ctx2 := &ScriptContext{}
	handled2, err := engine.RunScript(ctx2, "second.lua", "ongive")
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}
	if handled2 {
		t.Fatalf("second run expected handled=false because trigger is missing")
	}

	leak := engine.l.GetGlobal("leak")
	if leak.Type() != lua.LTString || leak.String() != "no" {
		t.Fatalf("stale context leaked into second script: leak=%v", leak)
	}
}
