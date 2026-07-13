# BRIEF — Stream 5: Lua sandbox hardening (F16)

**Linear:** DP-957 (F16 — Lua sandbox has no memory ceiling; collectgarbage still exposed)
**Effort:** M
**Agent:** Reek (DeepSeek)
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — F16

## Goal

1. Block `collectgarbage` in the Lua sandbox.
2. Add a memory ceiling using gopher-lua's `SetAllowance()`.
3. Document the trust model in engine.go.
4. Add sandbox regression tests.

## Problem

The Lua sandbox (`pkg/scripting/engine.go`) has a 5-second wall-clock CPU timeout but **no memory ceiling**. `collectgarbage` is still exposed, allowing forced GC cycles as a DoS vector. The trust model is implicit — no documentation exists explaining who authors scripts and what the containment goals are.

## Current State

### Blocked functions (denylist in `newSafeLState()`, lines 54-117):
- `dofile`, `loadfile`, `load`, `loadstring` — code loading
- `os.clock`, `os.execute`, `os.exit`, `os.getenv`, `os.remove`, `os.rename`, `os.setenv`, `os.setlocale`, `os.tmpname` — OS access
- `string.dump` — bytecode export
- `math.randomseed` — RNG seed control
- `package` global + `_LOADED` + `_PRELOAD` — package system
- `debug` global — debug introspection
- `io` global — filesystem I/O

### NOT blocked (gaps):
- **`collectgarbage`** — Lua built-in GC control. Can force full GC cycles.
- **`os.time`, `os.date`, `os.difftime`** — system clock observation. Low risk but unexpected.
- **No memory ceiling** — gopher-lua v1.1.2 supports `L.SetAllowance(n)` but it's unused.
- **No instruction limit** — gopher-lua supports `L.SetInstructionLimit(n)` but it's unused. The misleading comment about "instruction limit" at line 250 should be fixed or removed.

### Trust model
Scripts are **operator-authored** — loaded from `scriptsDir` on the server filesystem. Players cannot inject or upload scripts. The threat model is buggy/malicious operator scripts, not adversarial player input.

## Fix

### 1. Block collectgarbage

In `newSafeLState()`, after the existing denylist blocks:

```go
// Block collectgarbage to prevent GC abuse
L.SetGlobal("collectgarbage", L.NewFunction(func(L *lua.LState) int {
    L.Push(lua.LString("collectgarbage is disabled in this sandbox"))
    return 1
}))
```

### 2. Add memory ceiling

In `newSafeLState()`, after `L.OpenLibs()`:

```go
// Limit Lua-side memory allocation to 4MB per script execution.
// gopher-lua tracks allocation internally; exceeding this triggers a runtime error.
L.SetAllowance(4 * 1024 * 1024)
```

4MB is generous for game scripts but prevents unbounded table allocation. Adjust if real scripts hit this limit.

### 3. Fix misleading instruction limit comment

At engine.go line 250, if the comment mentions "instruction limit" as a panic reason but no instruction limit is set, either:
- Remove the mention, or
- Actually add `L.SetInstructionLimit(1_000_000)` (1M instructions is generous)

Recommendation: add the instruction limit — it's a better CPU governor than the wall-clock timeout.

### 4. Document trust model

Add a comment block at the top of `engine.go` (or near `newSafeLState()`):

```go
// Lua Script Security Model
//
// Scripts are operator-authored (immortal/builder), loaded from the server
// filesystem. Players cannot inject or upload scripts. The sandbox prevents
// buggy or malicious operator scripts from:
//
//   - Accessing the filesystem (io, os.execute, etc.)
//   - Loading arbitrary code (dofile, load, require)
//   - Exporting bytecode (string.dump)
//   - Escaping the Lua VM (package, debug)
//   - Forcing garbage collection cycles (collectgarbage)
//   - Consuming excessive memory (4MB allowance)
//   - Running indefinitely (5s wall-clock + 1M instruction limit)
//
// Scripts CAN access: math, string, table, coroutine, os.time/os.date, and
// ~60 registered game API functions (act, say, spell, oload, steal, etc.).
```

### 5. Add sandbox regression tests

In `pkg/scripting/engine_runscript_test.go` (existing test file):

- `TestLuaSandbox_CollectgarbageBlocked` — attempt `collectgarbage("collect")`, verify it returns an error or disabled message
- `TestLuaSandbox_MemoryCeiling` — allocate large table in a loop, verify it errors before consuming unlimited memory
- `TestLuaSandbox_InstructionLimit` — tight infinite loop, verify it stops before 5s wall clock

Note: the memory ceiling and instruction limit tests may be tricky to write deterministically. If they're too flaky, skip them and just test `collectgarbage` blocking.

## Files

| File | Change |
|---|---|
| `pkg/scripting/engine.go` | Block collectgarbage, add SetAllowance, add trust model docs, optionally add SetInstructionLimit |
| `pkg/scripting/engine_runscript_test.go` | Add collectgarbage blocked test, optional memory/instruction tests |

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. Do NOT change the denylist approach to an allowlist — too disruptive.
2. Do NOT block `os.time`/`os.date` — scripts legitimately need timestamps for game mechanics.
3. Do NOT block `coroutine` — game scripts may use them.
4. Do NOT change the 5s wall-clock timeout — keep it as a safety net alongside the instruction limit.
5. The memory ceiling (4MB) is a suggestion. If real scripts need more, adjust up. If they need less, adjust down. Document the chosen value in the trust model comment.
6. Single PR.

## C Fidelity

The C codebase has no Lua scripting — this is entirely a Go-era feature. No C behavior to match.
