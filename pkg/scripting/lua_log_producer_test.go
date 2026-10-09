package scripting

import "testing"

// scripts.c:780-793 through real Lua calls: the success arm logs the text at
// BRF / LVL_IMMORT / file TRUE and returns 0, while the bad-argument arm logs
// its own line at BRF / LVL_IMMORT / file FALSE and returns 1. The recording
// bridge carries the game adapter's file-TRUE sink, so the flag is asserted,
// not inferred.
func TestLuaLogProducerFileFlag(t *testing.T) {
	t.Run("string argument writes the file", func(t *testing.T) {
		engine := newScriptEngine(t, "log.lua",
			"function oncmd()\n  log('hello from lua')\n  return TRUE\nend\n")
		bridge := newRecordingBridge()
		if _, err := engine.RunScript(luaMudlogContext(bridge), "log.lua", "oncmd"); err != nil {
			t.Fatal(err)
		}
		assertProducers(t, fileLogs(bridge), []recordedMudlog{
			{msg: "hello from lua", typ: 1, level: 31, toFile: true},
		})
	})

	t.Run("bad argument stays file FALSE", func(t *testing.T) {
		// lua_isstring accepts numbers, so a table is C's bad-argument case.
		engine := newScriptEngine(t, "log.lua",
			"function oncmd()\n  log({})\n  return TRUE\nend\n")
		bridge := newRecordingBridge()
		if _, err := engine.RunScript(luaMudlogContext(bridge), "log.lua", "oncmd"); err != nil {
			t.Fatal(err)
		}
		if got := fileLogs(bridge); len(got) != 0 {
			t.Fatalf("the bad-argument arm wrote the file: %+v", got)
		}
		const payload = "[Lua] Invalid argument passed to lua_log."
		seen := 0
		for _, line := range bridge.logs {
			if line == payload {
				seen++
			}
		}
		if seen != 1 || len(bridge.logs) != 1 {
			t.Fatalf("bad-argument producer calls=%d of %d; logs=%q", seen, len(bridge.logs), bridge.logs)
		}
	})

	t.Run("number argument takes the success arm", func(t *testing.T) {
		// C's lua_isstring coerces a number, so log(42) logs "42" at file TRUE
		// rather than taking the bad-argument arm.
		engine := newScriptEngine(t, "log.lua",
			"function oncmd()\n  log(42)\n  return TRUE\nend\n")
		bridge := newRecordingBridge()
		if _, err := engine.RunScript(luaMudlogContext(bridge), "log.lua", "oncmd"); err != nil {
			t.Fatal(err)
		}
		assertProducers(t, fileLogs(bridge), []recordedMudlog{
			{msg: "42", typ: 1, level: 31, toFile: true},
		})
	})
}
