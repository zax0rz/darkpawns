# BRIEF — QA Residuals: fastpath regression + race + nil panic + cache staleness

**Effort:** S/M (all four together)
**Agent:** Kimi
**Source of truth:** QA review of Streams 1-5 (2026-07-06)

## Goal

Fix four residual issues found during post-Stream-5 QA review. These are all small, interrelated fixes in the spec-proc / mobact / world code.

## Issues

### #1 — F17 fastpath regression: player-carried spec objects can't fire

**Severity: Regression (was working before Stream 4d)**

The `HasSpecInRoom()` fastpath (world.go:825) gates ALL spec scans in commands.go:530 — including equipment/inventory scans (commands.go:554-579). But `rebuildSpecRoomsLocked()` (world.go:843) only flags rooms containing spec mobs, spec room-items, and room specs. It never scans player equipment or inventory.

**Result:** If a player carries a spec object (e.g., horn, VNum 14415) into a room with no other spec entities, the equipment/inventory spec scans never run. The object's spec proc is unreachable.

**Before the fastpath**, these scans were unconditional — they ran every command. The fastpath (Stream 4d) accidentally made them conditional on the room cache.

**Fix:** Pull the equipment/inventory scans (commands.go:552-579) OUT of the `HasSpecInRoom()` gate. They should run unconditionally — the scan is cheap (iterates the player's own bags, typically < 20 items). Only the mob-scan and room-scan should be gated.

```go
// Current (broken):
if s.manager.world.HasSpecInRoom(roomVNum) {
    // mob scans...
    // room scan...
    // equipment scans...  ← wrongly gated
    // inventory scans... ← wrongly gated
}

// Fixed:
if s.manager.world.HasSpecInRoom(roomVNum) {
    // mob scans...
    // room scan...
}
// Equipment and inventory spec scans — always run, cheap.
if s.player.Equipment != nil {
    // ... equipped item scans
}
if s.player.Inventory != nil {
    // ... inventory item scans
}
```

### #2 — Data race on Player.Exp / Player.Gold in specDump

**Severity: Race condition (real since F5 made specDump reachable)**

`spec_procs.go:201,203` writes `ch.Exp` and `ch.Gold` without holding `ch.mu`:
```go
ch.Exp += value   // line 201 — no lock
ch.Gold += value  // line 203 — no lock
```

Other spec procs in the same file correctly lock (e.g., lines 290-295, 807-809, 841-844). This is just an oversight.

**Fix:** Wrap lines 201-203 in `ch.mu.Lock()` / `ch.mu.Unlock()`.

### #3 — specHorn panics on nil `me`

**Severity: Nil pointer dereference (latent — blocked by #1)**

Object specs are dispatched with `me = nil` (commands.go:559, 573). `specHorn` (spec_procs4.go:521) calls `me.GetName()` at line 527 and `me.GetRoom()` at line 533 — both panic on nil.

Currently hidden because #1 means specHorn can never fire. Once #1 is fixed, using the horn will crash the goroutine (recovered by the session pump, but the command is broken).

Also: lines 534-535 pass literal `$n` and `$P` tokens through `roomMessage()`, which does no act-token interpolation. The player sees raw `$n blows into $P.` instead of the mob's name.

**Fix:** Object specs receive the object info via `arg`. The current specHorn checks `strings.Contains(arg, me.GetName())` to match "use horn" — but since `me` is nil for object specs, the arg matching must change.

Looking at the C source for reference:
```c
// Object specs in C receive (ch, obj, cmd, arg) where obj is the Object, not nil.
// The Go port passes nil for me on object specs — this is the root mismatch.
```

The correct fix: specHorn should match on `arg` containing "horn" directly (the argument to "use"), not on `me.GetName()`. For the room messages, use `ch.GetName()` for `$n` and "a horn" for `$P` (or get the actual object name from the player's inventory/equipment).

```go
func specHorn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
    if cmd != "use" {
        return false
    }
    arg = strings.TrimSpace(arg)
    if !strings.Contains(strings.ToLower(arg), "horn") {
        return false
    }
    sendToChar(ch, "You inhale deeply then blow hard!\r\n")
    sendToChar(ch, "A blaring note resounds through the air.\r\n")
    w.roomMessage(ch.GetRoomVNum(), ch.GetName()+" blows into a horn.")
    w.roomMessage(ch.GetRoomVNum(), "A horn lets out a blaring note...")
    return true
}
```

### #4 — Spec-room cache staleness for wandering/hunting mobs

**Severity: Stale-negative (spec mobs that wander lose their room flag)**

The `specRooms` cache rebuilds at boot and on the periodic-reset tick. If a spec mob wanders (ai.go:150 `mob.SetRoom`) or hunts (graph.go:336 `m.SetRoom`) into a new room, that room's flag stays false until the next rebuild. The cache's own doc comment says "stale-negative is not" acceptable.

Tick-driven mobact behavior (cmd == "") is unaffected because mobact's path is ungated. But command interception (HasSpecInRoom gate) can miss a spec mob that just wandered in.

**Fix:** After every mob room change, check if the mob has a spec and flag the new room. Add a helper:

```go
// flagSpecRoomForMob checks if the mob has a spec proc and flags its current room.
// Call after any mob room change (wander, hunt, teleport, etc.).
func (w *World) flagSpecRoomForMob(mob *MobInstance) {
    if GetMobSpec(mob.VNum) != nil {
        w.specRoomsMu.Lock()
        w.specRooms[mob.GetRoom()] = true
        w.specRoomsMu.Unlock()
    }
}
```

Call it from both movement sites:
- `ai.go:150` — after `mob.SetRoom(targetRoom.VNum)` in `wanderMob`
- `graph.go:336` — after `m.SetRoom(toRoomVNum)` in `mobPerformMove`

Also call it from `SpawnMob` (world.go:1014) when the mob is first placed.

## Files

| File | Change | Issue |
|---|---|---|
| `pkg/session/commands.go` | Pull equipment/inventory scans outside HasSpecInRoom gate | #1 |
| `pkg/game/spec_procs.go` | Wrap Exp/Gold writes in ch.mu.Lock() | #2 |
| `pkg/game/spec_procs4.go` | Fix specHorn nil-deref and $n/$P tokens | #3 |
| `pkg/game/world.go` | Add `flagSpecRoomForMob` helper | #4 |
| `pkg/game/ai.go` | Call `flagSpecRoomForMob` after wander | #4 |
| `pkg/game/graph.go` | Call `flagSpecRoomForMob` after hunt move | #4 |
| `pkg/game/world.go` (SpawnMob) | Call `flagSpecRoomForMob` after spawn | #4 |

## Tests

- `TestSpecHorn_UseHornDoesNotPanic` — use horn command, verify no panic and correct messages
- `TestSpecDump_ExpGoldLocked` — verify specDump awards without race (may need -race to be meaningful)
- `TestHasSpecInRoom_WanderingSpecMob` — spawn spec mob, move it, verify destination room is flagged

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. **#1 fix must NOT re-gate equipment/inventory scans behind HasSpecInRoom.** The whole point is to make them unconditional. The mob and room scans stay gated.
2. **#3 specHorn must work when me is nil** (object spec dispatch). Do NOT change the dispatch signature — other object specs depend on me being nil.
3. **#4 must not hold w.mu while writing specRoomsMu.** The helper only needs specRoomsMu (a fast write lock on a bool map entry). Do NOT acquire w.mu inside flagSpecRoomForMob — the caller context varies.
4. Single PR.
