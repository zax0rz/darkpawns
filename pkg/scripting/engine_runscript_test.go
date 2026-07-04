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
	if err := os.WriteFile(first, []byte(firstSrc), 0o644); err != nil {
		t.Fatalf("write first.lua: %v", err)
	}

	second := filepath.Join(dir, "second.lua")
	secondSrc := `leak = "no"
if argument ~= nil then leak = "yes" end
if room ~= nil then leak = "yes" end
`
	if err := os.WriteFile(second, []byte(secondSrc), 0o644); err != nil {
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

	// After cleanup, script-defined globals (including "leak") should be nil.
	// This verifies that the stale "argument" and "room" from the first run
	// did NOT leak into the second script's execution, AND that the second
	// script's own globals were cleaned up after completion.
	leak := engine.l.GetGlobal("leak")
	if leak.Type() != lua.LTNil {
		t.Fatalf("script-defined global 'leak' was not cleaned up after second run: leak=%v (type=%s)", leak, leak.Type())
	}

	// Also verify no stale globals remain from either script
	for _, name := range []string{"argument", "ongive", "room"} {
		val := engine.l.GetGlobal(name)
		if val.Type() != lua.LTNil {
			t.Errorf("stale global '%s' leaked between runs: %v (type=%s)", name, val, val.Type())
		}
	}
}

// TestRunScriptPathTraversal verifies that RunScript rejects path traversal
// attempts and returns an error rather than loading an arbitrary file.
func TestRunScriptPathTraversal(t *testing.T) {
	dir := t.TempDir()

	// Create a legitimate script in the test directory
	legit := filepath.Join(dir, "legit.lua")
	if err := os.WriteFile(legit, []byte(`function test() return TRUE end`), 0o644); err != nil {
		t.Fatalf("write legit.lua: %v", err)
	}

	engine := NewEngine(dir, nil)
	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	tests := []struct {
		name  string
		fname string
	}{
		{"parent_dir_traversal", "../../etc/passwd"},
		{"deep_traversal", "mob/../../../etc/passwd"},
		{"absolute_path", "/etc/passwd"},
		{"encoded_traversal", "..%2f..%2fetc%2fpasswd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.RunScript(&ScriptContext{}, tt.fname, "test")
			if err == nil {
				t.Fatal("expected error for path traversal, got nil")
			}
			t.Logf("correctly rejected: %s -> %v", tt.fname, err)
		})
	}

	// Verify legitimate paths still work
	_, err := engine.RunScript(&ScriptContext{}, "legit.lua", "test")
	if err != nil {
		t.Fatalf("legitimate path should not error: %v", err)
	}
}
