# BRIEF — COV-6: Lua adversarial test suite (DP-967)

**Linear:** DP-967 (COV-6: Lua adversarial test suite — sandbox escape attempts)
**Effort:** S
**Agent:** Reek (DeepSeek)
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — §3C item 6

## Goal

Write a suite of Lua scripts that attempt sandbox escape, and Go tests that execute them and assert the sandbox holds. Documents and locks in the F16 trust boundary.

## Background

F16 (DP-956) added the Lua sandbox: `collectgarbage` blocked, `os.execute`/`os.remove`/`os.getenv`/etc. nilled, `io` library removed, `package`/`require` disabled, `load`/`loadstring`/`dofile` nilled, `string.dump` nilled, `debug` library removed, `math.randomseed` nilled.

The sandbox is implemented in `pkg/scripting/engine.go:73-133` (`newSafeLState()`). One test already exists for collectgarbage (`TestLuaSandbox_CollectgarbageBlocked` at engine_runscript_test.go:441).

This brief adds a comprehensive adversarial suite covering ALL sandbox boundaries, not just collectgarbage.

## Current Sandbox (engine.go:73-133)

Blocked:
- `dofile`, `loadfile`, `load`, `loadstring` → nil
- `os.clock`, `os.execute`, `os.exit`, `os.getenv`, `os.remove`, `os.rename`, `os.setenv`, `os.setlocale`, `os.tmpname` → nil
- `string.dump` → nil
- `math.randomseed` → nil
- `package` → nil (whole table)
- `_LOADED`, `_PRELOAD` registry entries → nil
- `debug` → nil (whole library)
- `io` → nil (whole library)
- `collectgarbage` → stub returning error message

Still available (intentionally):
- `os.time`, `os.date` — time queries, no side effects
- `math`, `string`, `table`, `coroutine` — safe standard libs
- ~60 registered game API functions

## Fix

### Test Structure

Create `pkg/scripting/sandbox_adversarial_test.go` with a table-driven test pattern:

```go
func TestSandbox_Adversarial(t *testing.T) {
    tests := []struct {
        name    string
        script  string
        wantErr bool     // true = script should error
        wantMsg string   // substring expected in error message
    }{
        // Filesystem access
        {"os_execute", `os.execute("rm -rf /")`, true, ""},
        {"os_remove", `os.remove("/etc/passwd")`, true, ""},
        {"io_open", `io.open("/etc/passwd", "r")`, true, ""},

        // Code loading
        {"require", `require("os")`, true, ""},
        {"load", `load("print(1)")()`, true, ""},
        {"loadstring", `loadstring("print(1)")()`, true, ""},
        {"dofile", `dofile("/etc/passwd")`, true, ""},

        // Bytecode
        {"string_dump", `string.dump(print)`, true, ""},

        // Process control
        {"os_exit", `os.exit(1)`, true, ""},

        // Environment
        {"os_getenv", `os.getenv("HOME")`, true, ""},
        {"os_setenv", `os.setenv("PATH", "/tmp")`, true, ""},

        // Debug
        {"debug_getinfo", `debug.getinfo(1)`, true, ""},

        // GC abuse
        {"collectgarbage", `collectgarbage("collect")`, false, "disabled"}, // returns string, not error

        // Random seed manipulation
        {"math_randomseed", `math.randomseed(12345)`, true, ""},

        // Temp file
        {"os_tmpname", `os.tmpname()`, true, ""},

        // Package loaders via registry
        {"package_loaded", `package.loaded["os"]`, true, ""},
        {"package_preload", `package.preload["os"]`, true, ""},
    }
    // ... execute each and assert
}
```

### Implementation Approach

Each test script needs a `ScriptContext` and a file on disk (since `RunScript` takes a filename). Use `t.TempDir()` for script files.

Check existing test patterns in `pkg/scripting/engine_runscript_test.go` for how to:
1. Create an `Engine` with a test `ScriptContext`
2. Write script files to temp dir
3. Call `RunScript` or `newSafeLState` + `DoString`
4. Check results

For scripts that should error: the error might come from Lua runtime (nil function call) or from the engine. Either way, assert that the script does NOT successfully execute the dangerous operation.

For `collectgarbage`: this is special — it returns a string ("collectgarbage is disabled") instead of erroring. Assert the return value contains "disabled".

### Additional Edge Cases

Add a few more creative attacks:

1. **Reintroduction via global assignment:** `os = {execute = function() end}` — this should NOT grant real os access since the original `os` table is still sandboxed. But verify the script doesn't crash.

2. **Coroutine escape:** `co = coroutine.create(function() return os.execute("ls") end); coroutine.resume(co)` — verify os.execute is still nil inside coroutines.

3. **Rawget bypass:** `rawget(os, "execute")` — verify this returns nil.

4. **Metatable abuse:** `setmetatable(os, {__index = function(t, k) if k == "execute" then return function() end end end}); os.execute("ls")` — creative but should still fail since `os.execute` is nil on the original table.

## Files

| File | Change |
|---|---|
| `pkg/scripting/sandbox_adversarial_test.go` | New file — adversarial test suite |

## Existing Tests

- `pkg/scripting/engine_runscript_test.go:441` — `TestLuaSandbox_CollectgarbageBlocked` (existing)
- `pkg/scripting/engine_runscript_test.go` — 582 lines of existing Lua test infrastructure

Follow the patterns from the existing test file for Engine creation and script execution.

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. **Do NOT modify the sandbox itself.** These tests document the existing boundaries, not add new ones.
2. **Do NOT run tests that actually perform dangerous operations.** The scripts attempt to call nil'd functions — the Lua runtime will error before any real I/O happens.
3. **Follow existing test patterns.** Use `t.TempDir()` for script files, match the Engine/ScriptContext setup from existing tests.
4. **Table-driven tests preferred.** One `TestSandbox_Adversarial` function with sub-tests is cleaner than 15 separate test functions.
5. Single PR.

## Documentation Value

The test file itself serves as documentation of the trust boundary. Add a file-level doc comment listing what's blocked and what's allowed, so future developers can see the boundary without reading `newSafeLState()`.

## C Fidelity

C (Merc 2.2) had no scripting engine — this is a Go-era feature. The sandbox is a security boundary unique to the Go port. Tests here protect against regressions in that boundary.
