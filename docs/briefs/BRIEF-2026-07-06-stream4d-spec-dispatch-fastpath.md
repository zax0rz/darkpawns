# BRIEF — Stream 4d: Spec Dispatch Fast Path (F17)

**Linear:** DP-955 (F17 — per-command spec dispatch O(n) scan)
**Effort:** S (single PR)
**Agent:** Kimi
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — F17

## Goal

Add a fast path to the spec procedure dispatch pipeline at `pkg/session/commands.go:526-590`. Precompute per-room spec presence so the 4 linear scans (mobs + equipment + inventory + room items) can be skipped entirely when no specs are present. Most rooms have no specs at all.

## Problem

Every command from every player triggers four linear scans in `pkg/session/commands.go:526-590`:

1. **Mob spec procedures** (line 532): `GetMobsInRoom()` snapshots ALL active mobs, then filters by room. For each mob, does `GetMobSpec(mob.VNum)` — a map lookup. Out of ~228 mobs with specs, most rooms have zero.

2. **Room spec procedure** (line 544): `GetRoomSpec(roomVNum)` — single map lookup. Cheap, but still called every command.

3. **Equipped items** (line 552): Iterates all equipped items, `GetObjSpec()` each. Players rarely have spec objects equipped.

4. **Inventory items** (line 566): `FindItems("")` returns ALL inventory, then `GetObjSpec()` each. Same — almost never has specs.

5. **Room items** (line 580): `GetItemsInRoom()` returns all room objects, `GetObjSpec()` each. Only rooms with board/shop/moon-gate objects have specs.

The total cost per command: 1 full `activeMobs` snapshot + 4 linear scans + N map lookups. Fine at current player count, but will not scale.

## Current Spec Distribution

| Type | VNums with specs | Total VNums in game | Hit rate |
|---|---|---|---|
| Mob (MobSpecAssign) | 228 | ~thousands | ~5-10% |
| Object (ObjSpecAssign) | 27 | ~thousands | <1% |
| Room (RoomSpecAssign) | 25 | ~thousands | <1% |

**Most rooms have ZERO specs.** The fast path should skip all scanning for those rooms.

## Fix

### Phase 1: Per-room spec presence bitmap on World

Add a set (map[int]struct{} or similar) to World that tracks which room VNums contain at least one entity (mob, object, or room itself) with a spec procedure. Recompute when entities enter/leave rooms.

### Implementation approach

**Step 1: Add `specRooms` set to World**

In `pkg/game/world.go`, add:
```go
type World struct {
    // ... existing fields ...

    // specRooms tracks room VNums that contain at least one entity with a spec.
    // Used by the session dispatch fast path to skip spec scanning.
    specRooms map[int]bool
}
```

Initialize in `NewWorld()`.

**Step 2: Add update methods**

```go
// hasSpecInRoom checks if any entity in the room has a spec procedure.
func (w *World) hasSpecInRoom(roomVNum int) bool {
    if w.specRooms == nil {
        return true // safe fallback — scan always
    }
    return w.specRooms[roomVNum]
}
```

**Step 3: Populate at init time**

After world load (all mobs, items, rooms loaded), compute the initial set by iterating all active mobs and room items:
- For each mob in activeMobs, if `GetMobSpec(mob.VNum) != nil`, mark `mob.GetRoom()` in specRooms
- For each room, if `GetRoomSpec(roomVNum) != nil`, mark roomVNum in specRooms
- For each room item, if `GetObjSpec(item.VNum) != nil`, mark the room containing it

This runs once at boot. It's O(total_entities) but only once.

**Step 4: Update on entity moves**

When a mob enters/leaves a room, or an object moves between rooms, check if the entity has a spec and update specRooms accordingly. The relevant functions:

- `MobInstance.SetRoom()` — if mob has spec, add/remove room from specRooms
- `ObjectInstance` room changes — if object has spec, add/remove room from specRooms
- This is the trickiest part. For now, a simpler approach: **recompute per-tick** or **recompute lazily**.

### Simpler alternative (recommended for S-effort)

Instead of maintaining `specRooms` incrementally, recompute it lazily with a generation counter:

```go
type World struct {
    specRoomCache    map[int]bool
    specRoomGen      int
    specRoomDirtyGen int  // incremented whenever mobs/items move between rooms
}
```

When `specRoomDirtyGen != specRoomGen`, recompute the cache on next access. Increment `specRoomDirtyGen` in mob movement and object transfer paths. This amortizes the cost and avoids per-move bookkeeping.

**Even simpler:** Since mob/item movement is already relatively rare compared to command dispatch, just recompute `specRooms` periodically (e.g., every 60s tick alongside zone resets). A room might have a stale "has spec" bit for up to 60s after a spec entity leaves — the worst case is one unnecessary scan, which is the current behavior anyway. This is the safest approach.

### Step 5: Fast path in session dispatch

In `pkg/session/commands.go`, before the four scan blocks:

```go
// Spec procedure command interception — fast path
if s.player != nil && s.player.GetRoomVNum() > 0 {
    roomVNum := s.player.GetRoomVNum()
    if s.manager.world.HasSpecInRoom(roomVNum) {
        argStr := strings.Join(args, " ")
        // ... existing four scan blocks unchanged ...
    }
}
```

Add `HasSpecInRoom` as a public method on World:
```go
func (w *World) HasSpecInRoom(roomVNum int) bool {
    return w.specRooms[roomVNum]
}
```

When the room has no specs, this is a single map lookup and we skip the entire scan. When it does have specs, we scan as before (same behavior, just an extra map lookup overhead — negligible).

## Files

| File | Change |
|---|---|
| `pkg/game/world.go` | Add `specRooms map[int]bool` to World, `HasSpecInRoom()`, init population, periodic refresh |
| `pkg/session/commands.go` | Wrap spec scan block (lines 526-590) with `HasSpecInRoom` fast path |
| `pkg/game/world_test.go` | Add tests: `TestHasSpecInRoom_MobWithSpec`, `TestHasSpecInRoom_NoSpecs`, `TestHasSpecInRoom_ObjectMoved` |

## C Source Reference

The C source (`src/comm.c`) has the same linear scan pattern in `command_interpreter()`. This is a Go-only optimization — C never had this fast path either. No C fidelity concern since we're not changing behavior, only skipping work.

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

All five must pass.

## Regression Tests

- `TestHasSpecInRoom_MobWithSpec` — room with a mob that has a spec → returns true
- `TestHasSpecInRoom_NoSpecs` — empty room → returns false
- `TestHasSpecInRoom_BoardObject` — room with a board VNum object → returns true (ObjSpecAssign has board VNMs)
- `TestHasSpecInRoom_RoomSpec` — room with a room spec → returns true
- `TestHasSpecInRoom_AfterRefresh` — spec entity removed, refresh, returns false

All tests must use the public `HasSpecInRoom` API — do not test internal map directly.

## Constraints

1. **No behavioral changes.** When `HasSpecInRoom` returns true, the four scans must produce identical results to current behavior. The fast path only skips when NO specs exist.
2. **Stale-positive is safe.** If `specRooms` says a room has specs but the spec entity just left, the worst case is one unnecessary full scan. Never stale-negative (claiming no specs when specs exist — that would break spec proc behavior).
3. **Thread-safe.** `specRooms` reads happen from session goroutines. Writes happen from init/tick goroutines. Use `sync.RWMutex` or atomics.
4. **Keep GetMobsInRoom snapshot pattern.** Do not modify `GetMobsInRoom` — it has the nested-lock avoidance documented at line 1101-1106. The fast path just skips calling it.
5. **Single PR.** This is S-effort — one commit, one PR.

## Suggested PR Title

`perf: add spec dispatch fast path — skip room scans when no specs present (DP-955 / F17)`
