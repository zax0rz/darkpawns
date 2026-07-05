package scripting

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

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

// TestResolveScriptPathSubdirectory verifies that script resolution searches
// scriptsDir first, then falls back to mob/, room/, and obj/ subdirectories,
// matching C's SCRIPT_DIR/type/script_name pattern (DP-926).
func TestResolveScriptPathSubdirectory(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "globals.lua"), []byte(""), 0o644); err != nil {
		t.Fatalf("write globals.lua: %v", err)
	}

	mobDir := filepath.Join(dir, "mob")
	if err := os.MkdirAll(mobDir, 0o755); err != nil {
		t.Fatalf("mkdir mob: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mobDir, "troll.lua"), []byte(""), 0o644); err != nil {
		t.Fatalf("write troll.lua: %v", err)
	}

	engine := NewEngine(dir, nil)

	if got := engine.resolveScriptPath("globals.lua"); got != filepath.Join(dir, "globals.lua") {
		t.Errorf("globals.lua: expected %q, got %q", filepath.Join(dir, "globals.lua"), got)
	}
	if got := engine.resolveScriptPath("troll.lua"); got != filepath.Join(mobDir, "troll.lua") {
		t.Errorf("troll.lua: expected %q, got %q", filepath.Join(mobDir, "troll.lua"), got)
	}
	if got := engine.resolveScriptPath("nonexistent.lua"); got != "" {
		t.Errorf("nonexistent.lua: expected empty path, got %q", got)
	}
}

// TestRunScriptSubdirectory verifies that RunScript can execute a script located
// in a subdirectory under scriptsDir (DP-926).
func TestRunScriptSubdirectory(t *testing.T) {
	dir := t.TempDir()

	mobDir := filepath.Join(dir, "mob")
	if err := os.MkdirAll(mobDir, 0o755); err != nil {
		t.Fatalf("mkdir mob: %v", err)
	}
	src := `function onpulse()
	return TRUE
end
`
	if err := os.WriteFile(filepath.Join(mobDir, "test.lua"), []byte(src), 0o644); err != nil {
		t.Fatalf("write test.lua: %v", err)
	}

	engine := NewEngine(dir, nil)
	handled, err := engine.RunScript(&ScriptContext{}, "test.lua", "onpulse")
	if err != nil {
		t.Fatalf("RunScript error: %v", err)
	}
	if !handled {
		t.Error("expected handled=true")
	}
}

// TestScriptLoadFailureCaching verifies that a missing/broken script is
// attempted once, recorded in the negative cache, and skipped on subsequent
// calls — the second call must return a cheap cached error without hitting
// disk again. This is the core of the DP-903 fix: without caching, every
// pulse retried the load and logged ERROR+WARN pairs per mob.
func TestScriptLoadFailureCaching(t *testing.T) {
	dir := t.TempDir()
	engine := NewEngine(dir, nil)

	const missing = "nonexistent.lua"

	// First call: file does not exist → DoFile errors, cached, slog.Error once.
	_, err := engine.RunScript(&ScriptContext{}, missing, "test")
	if err == nil {
		t.Fatal("first call: expected error for missing script, got nil")
	}

	// Confirm the script landed in the negative cache.
	if _, ok := engine.failedScripts[filepath.Clean(missing)]; !ok {
		t.Fatalf("first call: expected %q in failedScripts cache after load error", missing)
	}

	// Second call: must return the cached error and NOT re-attempt the load.
	// We detect a re-attempt by checking the error message — the cached path
	// returns "previously failed to load", whereas a fresh DoFile failure
	// would surface a filesystem error from gopher-lua.
	_, err2 := engine.RunScript(&ScriptContext{}, missing, "test")
	if err2 == nil {
		t.Fatal("second call: expected cached error, got nil")
	}
	if !strings.Contains(err2.Error(), "previously failed to load") {
		t.Fatalf("second call: expected cached-error message, got: %v", err2)
	}

	// Sanity: a different missing script is NOT cached by the first failure.
	if _, ok := engine.failedScripts["other_missing.lua"]; ok {
		t.Error("failedScripts should only contain the script that actually failed")
	}
}

// TestScriptLoadFailureDoesNotFloodLog verifies that calling RunScript many
// times for the same missing script produces exactly ONE slog.Error line,
// not one per call. Before the DP-903 fix, 100 pulses × 2 log lines = 200
// log lines per affected script; on the live server this was ~86K lines in
// 25 minutes across 5 missing scripts.
func TestScriptLoadFailureDoesNotFloodLog(t *testing.T) {
	dir := t.TempDir()
	engine := NewEngine(dir, nil)

	// Counting slog handler — captures only Error-level records.
	handler := &countingHandler{level: slog.LevelError}
	// Replace the default logger for the duration of the test.
	restore := replaceSlogDefault(handler)
	defer restore()

	const missing = "troll.lua" // matches one of the 5 missing scripts in the bug report
	const pulses = 100
	for i := 0; i < pulses; i++ {
		if _, err := engine.RunScript(&ScriptContext{}, missing, "onpulse"); err == nil {
			t.Fatalf("pulse %d: expected error, got nil", i)
		}
	}

	if got := handler.count(); got != 1 {
		t.Errorf("expected exactly 1 Error log line for %d pulses of a missing script, got %d", pulses, got)
	}
}

// TestPathTraversalNotCached verifies that path-traversal rejections are NOT
// negative-cached. The DP-903 brief is explicit: only file-not-found / load
// errors are cached, since traversal attempts are a security signal that
// should keep firing every time.
func TestPathTraversalNotCached(t *testing.T) {
	dir := t.TempDir()
	engine := NewEngine(dir, nil)

	const traversal = "../../etc/passwd"

	// First call: rejected with an error.
	_, err1 := engine.RunScript(&ScriptContext{}, traversal, "test")
	if err1 == nil {
		t.Fatal("first traversal call: expected error, got nil")
	}

	// Must NOT be in the cache.
	if _, ok := engine.failedScripts[filepath.Clean(traversal)]; ok {
		t.Error("path-traversal rejection must not be cached — it should keep logging")
	}

	// Second call: also rejected (and re-logged, since not cached).
	_, err2 := engine.RunScript(&ScriptContext{}, traversal, "test")
	if err2 == nil {
		t.Fatal("second traversal call: expected error, got nil")
	}
}

// --- slog counting harness for the no-flood test ---

// countingHandler is a minimal slog.Handler that counts records at or above
// the configured level. Used to assert that a missing script logs exactly once.
type countingHandler struct {
	mu    sync.Mutex
	n     int
	level slog.Level
}

func (h *countingHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *countingHandler) Handle(_ context.Context, _ slog.Record) error {
	h.mu.Lock()
	h.n++
	h.mu.Unlock()
	return nil
}

func (h *countingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *countingHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *countingHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.n
}

// replaceSlogDefault swaps slog's default logger for one using h and returns
// a restore function. Tests use this to capture slog.Error output from
// RunScript's error paths.
func replaceSlogDefault(h slog.Handler) func() {
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	return func() { slog.SetDefault(prev) }
}

func TestEngine_CloseStopsCleanupGoroutine(t *testing.T) {
	runtime.GC()
	before := runtime.NumGoroutine()

	engine := NewEngine(t.TempDir(), nil)
	// Give the cleanup goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	engine.Close()
	engine.Close() // idempotency check

	// Give the goroutine time to exit.
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()

	if after > before {
		t.Errorf("expected goroutine count to return to baseline, before=%d after=%d", before, after)
	}
}

func TestRunScript_RecoveredPanicReturnsError(t *testing.T) {
	dir := t.TempDir()

	panicScript := filepath.Join(dir, "panic.lua")
	panicSrc := `function oncmd()
	trigger_panic()
	return TRUE
end
`
	if err := os.WriteFile(panicScript, []byte(panicSrc), 0o644); err != nil {
		t.Fatalf("write panic.lua: %v", err)
	}

	goodScript := filepath.Join(dir, "good.lua")
	goodSrc := `function oncmd()
	return TRUE
end
`
	if err := os.WriteFile(goodScript, []byte(goodSrc), 0o644); err != nil {
		t.Fatalf("write good.lua: %v", err)
	}

	engine := NewEngine(dir, nil)
	defer engine.Close()

	// Register a Go function that panics when called from Lua.
	engine.LState().SetGlobal("trigger_panic", engine.LState().NewFunction(func(L *lua.LState) int {
		panic("intentional test panic")
	}))

	handled, err := engine.RunScript(&ScriptContext{}, "panic.lua", "oncmd")
	if err == nil {
		t.Fatal("RunScript expected error after recovered panic, got nil")
	}
	if handled {
		t.Error("expected handled=false after panic")
	}

	// Verify the engine recovered and can still run scripts.
	handled, err = engine.RunScript(&ScriptContext{}, "good.lua", "oncmd")
	if err != nil {
		t.Fatalf("RunScript failed after panic recovery: %v", err)
	}
	if !handled {
		t.Error("expected handled=true for good script")
	}
}

func TestLuaSandbox_RequireBlocked(t *testing.T) {
	dir := t.TempDir()

	requireScript := filepath.Join(dir, "require.lua")
	if err := os.WriteFile(requireScript, []byte(`function oncmd() require("os") return true end`), 0o644); err != nil {
		t.Fatalf("write require.lua: %v", err)
	}

	engine := NewEngine(dir, nil)
	defer engine.Close()

	handled, err := engine.RunScript(&ScriptContext{}, "require.lua", "oncmd")
	if err == nil {
		t.Fatal("expected error when sandboxed script calls require, got nil")
	}
	if handled {
		t.Error("expected handled=false when require fails")
	}
}

func TestRunScript_BooleanReturn(t *testing.T) {
	dir := t.TempDir()

	boolTrue := filepath.Join(dir, "bool_true.lua")
	if err := os.WriteFile(boolTrue, []byte(`function oncmd() return true end`), 0o644); err != nil {
		t.Fatalf("write bool_true.lua: %v", err)
	}

	boolFalse := filepath.Join(dir, "bool_false.lua")
	if err := os.WriteFile(boolFalse, []byte(`function oncmd() return false end`), 0o644); err != nil {
		t.Fatalf("write bool_false.lua: %v", err)
	}

	engine := NewEngine(dir, nil)
	defer engine.Close()

	handled, err := engine.RunScript(&ScriptContext{}, "bool_true.lua", "oncmd")
	if err != nil {
		t.Fatalf("RunScript failed: %v", err)
	}
	if !handled {
		t.Error("expected handled=true for boolean true return")
	}

	handled, err = engine.RunScript(&ScriptContext{}, "bool_false.lua", "oncmd")
	if err != nil {
		t.Fatalf("RunScript failed: %v", err)
	}
	if handled {
		t.Error("expected handled=false for boolean false return")
	}
}
