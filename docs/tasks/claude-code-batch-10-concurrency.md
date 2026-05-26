# Claude Code Batch — Run 10: Concurrency Fixes

## Overview
3 data race fixes + 1 edge case. These are correctness bugs where player state is read from goroutines without acquiring locks.

## Issues
- DP-383: Data race in look/examine (HIGH)
- DP-387: Data race in performGive (HIGH)
- DP-425: RemoveMsg lock race during board save (MEDIUM)
- DP-424: Board binary serialization drift (MEDIUM)

## DEFERRED
- DP-377: Look sends JSON to telnet (CRITICAL) — flagged for The Architect. This is an architecture decision: the JSON path is intentional for the browser client. Telnet support needs a dual-protocol dispatch, not a simple fix.

---

## Task 1: Fix data race in look/examine (DP-383)

**Files:** `pkg/session/cmd_look.go`, `pkg/session/examine.go`

**Problem:** `cmdLookAt` and `cmdExamine` call `world.GetPlayersInRoom()` and read player attributes (Name, Level, Health, affects) from the session goroutine without acquiring `p.mu` on the target player. Multiple concurrent lookers cause data races.

**Fix pattern:** Before reading any player fields, acquire `p.mu.RLock()`, do the read, then `p.mu.RUnlock()`.

However — the proper fix depends on how the Go concurrency model works here. The `Player.mu` is meant to protect player state during mutations. For read-only operations like look, we need to either:

**Option A — Lock around the read:**
```go
for _, p := range room.Players {
    p.mu.RLock()
    name := p.Name
    level := p.Level
    // ... read other fields
    p.mu.RUnlock()
    // ... format and add to output
}
```

**Option B — Use getter methods (preferred):**
If the Player has getter methods that internally lock (like `p.GetHP()`, `p.GetLevel()`), use those instead of direct field access. Check what getters exist:
- `p.GetHP()` vs `p.Health`
- `p.GetLevel()` vs `p.Level`
- `p.GetName()` vs `p.Name`

**Verify:** Check `pkg/game/player.go` for which fields have thread-safe getters. The getters likely acquire `p.mu.RLock()` internally.

**Action:** For each field read in `cmdLookAt` and `cmdExamine`, replace direct field access with getter methods where available. If getters don't exist for some fields, add them.

Also check `listCharToChar` in `pkg/game/look.go` — same issue if it reads player fields directly.

---

## Task 2: Fix data race in performGive (DP-387)

**File:** `pkg/game/item_transfer.go:303` — `performGive()`

**Problem:** `performGive` reads `vict.Inventory.Items`, `vict.Inventory.Capacity`, and `vict.Inventory.GetWeight()` from the giver's goroutine without locking the recipient's inventory.

**Fix:** Acquire the recipient's inventory lock before reading:
```go
vict.Inventory.mu.RLock()
canCarry := vict.Inventory.GetWeight()+item.GetWeight() <= vict.Inventory.GetCapacity()
vict.Inventory.mu.RUnlock()
```

**Check:** Does `Inventory` have a `mu sync.RWMutex`? Look at `pkg/game/inventory.go` for the struct definition. If it does, lock around reads. If it doesn't, this needs to be added.

**Also check:** The `equip()` and `unequip()` paths on Equipment — same class of bug if they read equipment from other goroutines without locking. But focus on `performGive` for this task.

---

## Task 3: Fix RemoveMsg lock race (DP-425)

**File:** Likely `pkg/game/boards.go` or wherever `RemoveMsg` is defined.

**Problem:** The board system has a race condition during save — `RemoveMsg` may be called while `SaveBoard` is writing, or vice versa. The lock is released and re-acquired, creating a window for corruption.

**Investigation needed:** Grep for `RemoveMsg` and `SaveBoard` to find the exact files and understand the locking pattern. The fix is likely:
- Hold the lock for the entire operation (remove + save), not release between them
- Or use a separate save lock

**C source:** `src/boards.c` — `do_board_remove_msg()`. The C version is single-threaded so no lock issue.

---

## Task 4: Board binary serialization drift (DP-424)

**File:** Likely `pkg/game/boards.go`

**Problem:** The board save/load format may not match the C binary format, or may have drifted. This is a data persistence issue — boards may not survive server restarts correctly.

**Investigation needed:** Read the board save/load code and compare with `src/boards.c` serialization format. Check:
- How messages are serialized (binary vs JSON)
- How the board header is written
- Whether the format matches the C version's expectations

This may be a larger task. If the serialization is working correctly (just different format from C), it can be deprioritized.

---

## Execution Order
1. Task 1 (look/examine race) — most impactful, affects every player
2. Task 2 (performGive race) — smaller, same pattern
3. Task 3 (RemoveMsg race) — needs investigation first
4. Task 4 (board serialization) — may be deferred if format is working

## Verification
1. `go build ./...` — must pass
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
4. Run with `-race` flag if available: `go test -race ./...` to verify no remaining races
