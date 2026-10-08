package scripting

import (
	"testing"
)

// recordedMudlog is one producer call with its full C contract.
type recordedMudlog struct {
	msg    string
	typ    int
	level  int
	toFile bool
}

// recordingBridge adds the file-TRUE sink (pkg/game's WorldScriptableAdapter
// MudLog) to the package's fake bridge, so a test can assert the producer's
// message, type, level and file flag. The embedded fake supplies every Bridge
// method, including Log (BRF/LVL_IMMORT/FALSE) for the file-FALSE sites.
type recordingBridge struct {
	*fakeBridge
	mudlogs []recordedMudlog
}

func (r *recordingBridge) MudLog(msg string, typ, level int, toFile bool) {
	r.mudlogs = append(r.mudlogs, recordedMudlog{msg: msg, typ: typ, level: level, toFile: toFile})
}

// The engine reaches the file-TRUE sink by inline assertion; this keeps the
// test bridge honest if that contract changes.
var _ interface{ MudLog(string, int, int, bool) } = (*recordingBridge)(nil)

func newRecordingBridge() *recordingBridge {
	return &recordingBridge{fakeBridge: newFakeBridge()}
}

// luaMudlogContext carries the owner identity RunScript needs to select the
// bridge; without it the engine runs the legacy path and no script producer can
// fire.
func luaMudlogContext(b *recordingBridge) *ScriptContext {
	me := player
	return &ScriptContext{World: b, MeRef: &me, OwnerType: "mob"}
}

// newScriptEngine writes one script file into a temp dir and returns an engine
// bound to that directory.
func newScriptEngine(t *testing.T, name, src string) *Engine {
	t.Helper()
	dir := t.TempDir()
	writeScript(t, dir, name, src)
	engine := NewEngine(dir, nil)
	t.Cleanup(func() { engine.l.Close() })
	return engine
}

// fileLogs returns only the producers that carry file TRUE.
func fileLogs(r *recordingBridge) []recordedMudlog {
	var out []recordedMudlog
	for _, entry := range r.mudlogs {
		if entry.toFile {
			out = append(out, entry)
		}
	}
	return out
}

func assertProducers(t *testing.T, got, want []recordedMudlog) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("producers=%+v want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("producer[%d]=%+v want %+v", i, got[i], want[i])
		}
	}
}

// TestLuaSkipSpacesInvalidArgumentProducer drives scripts.c:1393-1394 through a
// real Lua call with a bad argument: C logs "[Lua] Invalid argument passed to
// lua_skip_spaces." at BRF/LVL_IMMORT/file FALSE and still returns 1. A string
// argument must stay silent.
func TestLuaSkipSpacesInvalidArgumentProducer(t *testing.T) {
	const payload = "[Lua] Invalid argument passed to lua_skip_spaces."
	for _, tc := range []struct {
		name    string
		call    string
		wantLog bool
	}{
		{"number argument", "skip_spaces(42)", true},
		{"table argument", "skip_spaces({})", true},
		{"string argument", "skip_spaces('  padded')", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := newScriptEngine(t, "sp.lua",
				"function oncmd()\n  "+tc.call+"\n  return TRUE\nend\n")
			bridge := newRecordingBridge()
			if _, err := engine.RunScript(luaMudlogContext(bridge), "sp.lua", "oncmd"); err != nil {
				t.Fatalf("run: %v", err)
			}
			got := 0
			for _, line := range bridge.logs {
				if line == payload {
					got++
				}
			}
			if tc.wantLog && got != 1 {
				t.Fatalf("producer calls=%d want 1; logs=%q", got, bridge.logs)
			}
			if !tc.wantLog && got != 0 {
				t.Fatalf("a valid argument logged the diagnostic: %q", bridge.logs)
			}
			if len(bridge.mudlogs) != 0 {
				t.Fatalf("BRF/file-FALSE producer used the file-TRUE sink: %+v", bridge.mudlogs)
			}
		})
	}
}

// TestLuaRunScriptLoadFailureProducers drives scripts.c:1674-1694 and
// 1777-1779 through real RunScript loads: the classified open_lua_file line at
// CMP/LVL_IMMORT/file TRUE, then run_script's own line at BRF/LVL_IMMORT/file
// TRUE, in that order.
func TestLuaRunScriptLoadFailureProducers(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		kind string
	}{
		{"syntax error", "function oncmd(\n", "Syntax error."},
		{"execution failure", "error('boom')\nfunction oncmd() return TRUE end\n", "Execution failed."},
		{"missing script", "", "No such file."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "bad.lua"
			var engine *Engine
			if tc.name == "missing script" {
				// The directory exists and holds one other script, so the
				// requested name resolves to nothing.
				dir := t.TempDir()
				writeScript(t, dir, "present.lua", "function oncmd() return TRUE end\n")
				engine = NewEngine(dir, nil)
				t.Cleanup(func() { engine.l.Close() })
				name = "missing.lua"
			} else {
				engine = newScriptEngine(t, name, tc.src)
			}
			bridge := newRecordingBridge()
			if _, err := engine.RunScript(luaMudlogContext(bridge), name, "oncmd"); err == nil {
				t.Fatal("a failing load must return an error")
			}
			assertProducers(t, fileLogs(bridge), []recordedMudlog{
				{msg: "[Lua] Could not call script " + name + ": " + tc.kind, typ: 3, level: 31, toFile: true},
				{msg: "SYSERR: Error opening lua script " + name + ".", typ: 1, level: 31, toFile: true},
			})
		})
	}
}

// TestLuaRunScriptCallFailureProducer drives scripts.c:1798-1800 two ways: a
// trigger the script never defines, and a trigger that raises. Both are C's
// "being called with an error" case and both log BRF/LVL_IMMORT/file TRUE,
// before any bridge write-back.
func TestLuaRunScriptCallFailureProducer(t *testing.T) {
	for _, tc := range []struct {
		name    string
		src     string
		wantErr bool
	}{
		{"missing trigger", "function onother() return TRUE end\n", false},
		{"trigger raises", "function oncmd() error('boom') end\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := newScriptEngine(t, "call.lua", tc.src)
			bridge := newRecordingBridge()
			_, err := engine.RunScript(luaMudlogContext(bridge), "call.lua", "oncmd")
			if tc.wantErr != (err != nil) {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			assertProducers(t, fileLogs(bridge), []recordedMudlog{{
				msg:    "[Lua] Script call.lua being called with an error, function 'oncmd'.",
				typ:    1,
				level:  31,
				toFile: true,
			}})
		})
	}
}

// TestLuaRunScriptCachedLoadFailureStillLogs: C's run_script retries
// lua_dofile and logs both load-failure producers on every call
// (src/scripts.c:1674-1694, 1777-1779). The port's negative cache (DP-903)
// skips the retry, so it must still emit the same two lines on each call.
func TestLuaRunScriptCachedLoadFailureStillLogs(t *testing.T) {
	engine := newScriptEngine(t, "bad.lua", "function oncmd(\n")
	want := []recordedMudlog{
		{msg: "[Lua] Could not call script bad.lua: Syntax error.", typ: 3, level: 31, toFile: true},
		{msg: "SYSERR: Error opening lua script bad.lua.", typ: 1, level: 31, toFile: true},
	}
	for call := 1; call <= 3; call++ {
		bridge := newRecordingBridge()
		if _, err := engine.RunScript(luaMudlogContext(bridge), "bad.lua", "oncmd"); err == nil {
			t.Fatalf("call %d: a failing load must return an error", call)
		}
		if got := fileLogs(bridge); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("call %d: producers=%+v want %+v (the cached failure must log like C's retry)", call, got, want)
		}
	}
}
