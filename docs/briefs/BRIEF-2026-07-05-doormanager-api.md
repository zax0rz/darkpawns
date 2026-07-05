# BRIEF: DoorManager API — Return Door values, not pointers

**Date:** 2026-07-05
**Issues:** DP-698
**Priority:** Medium
**Files:** `pkg/game/systems/door_manager.go`, `pkg/game/systems/door.go` (Door struct)
**Cite:** No C equivalent — Go-only concurrency wrapper. In the C source (`src/`), doors are accessed via direct array index (`room_data[room].exit[dir]`), which is a flat struct, not a pointer. The Go DoorManager added synchronization that the C code didn't need (single-threaded DikuMUD).

---

## Fix: DP-698 — DoorManager exposes mutable *Door pointers outside lock (MEDIUM)

**File:** `pkg/game/systems/door_manager.go`

**Problem:**
`GetDoor()`, `GetDoorBetween()`, `GetDoorsInRoom()`, and `GetVisibleDoorsInRoom()` all return mutable `*Door` pointers (or `[]*Door` slices) after releasing `RLock`. Callers can mutate `Closed`, `Locked`, `Hp`, etc. without synchronization, creating data races with other goroutines that may be modifying the same doors concurrently.

```go
func (dm *DoorManager) GetDoor(fromRoom int, direction string) (*Door, bool) {
    dm.mu.RLock()
    defer dm.mu.RUnlock()

    key := dm.key(fromRoom, direction)
    door, ok := dm.doors[key]
    return door, ok  // *Door pointer escapes the lock!
}
```

**Fix:**
First, check the `Door` struct definition in `pkg/game/systems/door.go`. If it's a small struct (likely: closed bool, locked bool, hp int, maybe a few string fields), return by value instead of pointer.

Change all getter methods to return Door values:

```go
func (dm *DoorManager) GetDoor(fromRoom int, direction string) (Door, bool) {
    dm.mu.RLock()
    defer dm.mu.RUnlock()

    key := dm.key(fromRoom, direction)
    door, ok := dm.doors[key]
    if !ok {
        return Door{}, false
    }
    return *door, true  // return copy, not pointer
}
```

Similarly for:
- `GetDoorBetween(fromRoom, toRoom int)` — return `(Door, bool)` instead of `(*Door, bool)`
- `GetDoorsInRoom(room int)` — return `[]Door` instead of `[]*Door`
- `GetVisibleDoorsInRoom(room int, playerLevel int)` — return `[]Door` instead of `[]*Door`
- `GetDoorStatus(fromRoom, direction string)` — check what it returns and update similarly

**IMPORTANT — Caller Audit:**
After changing the return types, find ALL callers of these methods and update them. The callers will need to take the address of the returned value if they need a pointer, or better yet, work with the value directly. Key call sites to check:
- `pkg/game/combat.go` (door bashing)
- `pkg/game/skills.go` (pick lock, knock)
- `pkg/session/commands.go` or `pkg/command/` (open, close, lock, unlock commands)
- Any code that checks `door.Closed`, `door.Locked`, `door.Hp`

The mutating operations (`OpenDoor`, `CloseDoor`, `LockDoor`, `UnlockDoor`, `PickDoor`, `BashDoor`) already hold `Lock()` and mutate through the map directly, so they are unaffected — they don't call the getter methods.

**Cite:** C source — `src/db.c` `get_exit()` returns `EXIT_DATA *` but in C this is single-threaded so no data race. The Go port must account for goroutine concurrency.

**Regression Test:** `pkg/game/systems/door_manager_test.go`
- Add `TestDoorManager_GetDoorReturnsCopy` — get a door, modify the returned value, verify the original in the DoorManager is unchanged.
- Add `TestDoorManager_ConcurrentAccess` — concurrently call GetDoor and OpenDoor with `-race` flag. Verify no data race.

**Verification:** `go build ./... && go vet ./... && go test -race ./...`

---

## Execution Order

1. Read `pkg/game/systems/door.go` to confirm Door struct size and fields
2. Update all getter return types in `door_manager.go`
3. Find and update all callers (use `grep -rn 'GetDoor\b\|GetDoorBetween\|GetDoorsInRoom\|GetVisibleDoorsInRoom' pkg/`)
4. Add tests

## After All Fixes

- Run `go build ./... && go vet ./... && go test -race ./...`
- Create feature branch: `fix/dp-698-doormanager-api`
- Commit: `fix: DoorManager getters return Door value instead of *Door pointer (DP-698)`
- Open PR against `main`
- Mark DP-698 as Done in Linear
