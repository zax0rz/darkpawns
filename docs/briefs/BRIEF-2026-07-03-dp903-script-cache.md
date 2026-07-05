# Brief: DP-903 — Cache Script Load Failures — 2026-07-03

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.
**Milestone:** Fable Review (2026-07-03)

---

## Fix: DP-903 — Missing Lua scripts re-logged every pulse (HIGH)

**File:** `pkg/scripting/engine.go` — `RunScript()` (line ~195)

**Problem:**
5 scripts referenced by mob prototypes don't exist on disk: `troll.lua`, `bearcub.lua`, `janitor.lua`, `mount.lua`, `golem_to_crate.lua`. Every pulse tick, every mob with one of these scripts triggers `RunScript` → `L.DoFile(scriptPath)` → error → `slog.Error("error loading script", ...)`. With hundreds of mobs, this produces ~86K log lines in 25 minutes on an idle server.

**Root cause:** `RunScript()` attempts to load the file every single call. There's no negative cache. The error path at line ~323 logs every time.

**Fix:**
Add a `failedScripts` map to the Engine struct. Before calling `L.DoFile`, check if the script is in the failure cache. If so, return immediately with a cheap error (no log). On first failure, log once and add to cache.

Specifically:

1. Add a field to `Engine`:
```go
type Engine struct {
    // ... existing fields ...
    failedScripts map[string]struct{} // negative cache for missing/broken scripts
}
```

2. Initialize it in `NewEngine()`:
```go
failedScripts: make(map[string]struct{}),
```

3. In `RunScript()`, after path validation and before `L.DoFile(scriptPath)` (around line 315), add:
```go
if _, failed := e.failedScripts[cleanName]; failed {
    return false, fmt.Errorf("script %s previously failed to load", cleanName)
}
```

4. In the error path after `L.DoFile` fails (line ~323), add:
```go
e.failedScripts[cleanName] = struct{}{}
```

5. In the timeout error path (line ~320), also cache:
```go
e.failedScripts[cleanName] = struct{}{}
```

**Important:** Do NOT cache path-traversal errors (lines 289-295) — those should keep logging. Only cache file-not-found / load errors.

**Regression Test:**
`pkg/scripting/engine_test.go`:
- Add `TestScriptLoadFailureCaching`: create engine with empty scripts dir, RunScript with "nonexistent.lua" → error. RunScript again → error but no log flood (count calls or verify map entry exists).
- Add `TestScriptLoadFailureDoesNotFloodLog`: run RunScript 100 times with missing script, verify the slog.Error fires at most once.

**Cite:** C source — `scripts.c` `run_script()` lines 1732-1795. C had the same issue but disk was fast enough not to matter. Go's structured logging makes it visible.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## After Fix

```bash
git add pkg/scripting/engine.go pkg/scripting/engine_test.go
git commit -m "fix: cache Lua script load failures — stop 86K log lines per 25min (DP-903)"
git push -u origin fix/dp-903-script-cache
gh pr create --title "fix: cache Lua script load failures (DP-903)" --body "Fixes DP-903. See docs/briefs/BRIEF-2026-07-03-dp903-script-cache.md"
```

## Linear Update (after merge)
- DP-903: Add comment "Fixed — added failedScripts negative cache to Engine", commit hash, move to Done.
