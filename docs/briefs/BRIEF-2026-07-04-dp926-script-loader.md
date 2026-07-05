# Brief: Script Loader Subdirectory Resolution — DP-926

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Problem

The script loader resolves `Script: troll.lua` as `scripts/troll.lua` (flat lookup). But the files live at `scripts/mob/troll.lua`. Every mob pulse throws an ERROR. 44,673 errors in ~24h. Affected mobs: janitor (14311, 8061), bearcub (9111), mount (2739), troll (100, 121, 140, 141, 143, 158, 199).

## C source (ground truth)

`src/scripts.c:1775`:
```c
sprintf(buf, "%s/%s/%s", SCRIPT_DIR, type, script_name);
```

C builds the path as `scripts/<type>/<script_name>` where `type` is `mob`, `room`, or `obj`. The type is passed from the caller based on what entity triggered the script.

## Go behavior (current — BROKEN)

`pkg/scripting/engine.go:301`:
```go
scriptPath := filepath.Join(e.scriptsDir, cleanName)
```

Go resolves `troll.lua` as `scripts/troll.lua`. Flat lookup. No subdirectory search.

## Fix

In `RunScript` (engine.go), after the direct lookup fails, search subdirectories. The C code uses `type` (mob/room/obj), but mob files reference scripts by filename only (`Script: troll.lua 320`). Solution: search all three known subdirectories when the root lookup fails.

Replace the single `filepath.Join` + `DoFile` with a lookup function:

```go
// resolveScriptPath looks for a script file in scriptsDir, then in known
// subdirectories (mob/, room/, obj/). Matches C's SCRIPT_DIR/type/script_name
// pattern from scripts.c:1775.
func (e *Engine) resolveScriptPath(cleanName string) string {
    // 1. Direct lookup (flat — globals.lua lives here)
    direct := filepath.Join(e.scriptsDir, cleanName)
    if _, err := os.Stat(direct); err == nil {
        return direct
    }
    // 2. Search known subdirectories (matches C's type parameter)
    for _, sub := range []string{"mob", "room", "obj"} {
        candidate := filepath.Join(e.scriptsDir, sub, cleanName)
        if _, err := os.Stat(candidate); err == nil {
            return candidate
        }
    }
    return "" // not found
}
```

Then in `RunScript`, replace:
```go
scriptPath := filepath.Join(e.scriptsDir, cleanName)
```
with:
```go
scriptPath := e.resolveScriptPath(cleanName)
if scriptPath == "" {
    e.failedScripts[cleanName] = struct{}{}
    slog.Error("error loading script", "file", fname, "error", "script not found")
    return false, fmt.Errorf("script not found: %s", fname)
}
```

Keep the existing path traversal checks before calling `resolveScriptPath`.

## Test

```go
func TestResolveScriptPathSubdirectory(t *testing.T) {
    // Create temp scriptsDir with:
    //   scripts/globals.lua (empty)
    //   scripts/mob/troll.lua (empty)
    // Assert resolveScriptPath("globals.lua") returns scripts/globals.lua
    // Assert resolveScriptPath("troll.lua") returns scripts/mob/troll.lua
    // Assert resolveScriptPath("nonexistent.lua") returns ""
}

func TestRunScriptSubdirectory(t *testing.T) {
    // Create engine with scriptsDir containing scripts/mob/test.lua
    // RunScript("test.lua", "onpulse")
    // Assert no error, script executed
}
```

## Verification on prod

After deploy, check server log for reduction in "error loading script" lines. Affected mobs (14311, 8061, 9111, 2739) should stop throwing errors. Janitor should start cleaning, bear cub should follow mother, mount should return to stables.

## Linear update (after merge)

DP-926: "Fixed — script loader searches mob/room/obj subdirectories per C scripts.c:1775, commit <hash>"
