# Brief 9: Standalone Fidelity

**Issues:** DP-453, DP-443
**Priority:** HIGH + MEDIUM
**Files:** `pkg/game/zone_dispatcher.go`, `cmd/server/main.go`, `src/config.c`
**C Sources:** `src/config.c`, `src/db.c`

---

## Problem

Two standalone fidelity gaps that don't fit into other clusters: a dead zone dispatcher that was never wired in, and three sets of special rooms (donation, immortal start, frozen start) that were never ported from C config constants.

---

## Issues in This Brief

### DP-453 — ZoneDispatcher implemented but never started (HIGH)

**Go:** `pkg/game/zone_dispatcher.go` — full concurrent zone reset engine, fully implemented.
**Go:** `cmd/server/main.go:289` — only starts the serial fallback:
```go
gameWorld.StartPeriodicResets(60 * time.Second)
```
`StartZoneDispatcher()` is never called.

**Impact:** The codebase implies a concurrent zone model exists, but the server runs on the serial fallback. This is a maintenance hazard — correctness assumptions around the dispatcher may not match runtime behavior. It's also dead code that could confuse future contributors.

**Two options:**

**Option A — Wire it in (recommended if ready):**
In `cmd/server/main.go`, after the `StartPeriodicResets` line, add:
```go
if err := gameWorld.StartZoneDispatcher(); err != nil {
    log.Printf("ZoneDispatcher failed to start: %v, falling back to periodic resets", err)
    gameWorld.StartPeriodicResets(60 * time.Second)
}
```

But FIRST: verify `StartZoneDispatcher` is actually correct. Read `zone_dispatcher.go` thoroughly. Check:
- Does it handle zone age tracking correctly?
- Does it respect zone reset modes (0=never, 1=empty, 2=no players, 3=always)?
- Does it handle the case where a zone reset fails?
- Does it properly integrate with the global tick?

If any of these are broken, fix them before wiring in.

**Option B — Mark as disabled (safe, quick):**
Add a clear comment in `main.go`:
```go
// ZoneDispatcher is implemented but not yet wired in. See pkg/game/zone_dispatcher.go.
// Using serial periodic resets for now. Wire in StartZoneDispatcher() when ready.
gameWorld.StartPeriodicResets(60 * time.Second)
```

And add a comment at the top of `zone_dispatcher.go`:
```go
// ZoneDispatcher: NOT YET ACTIVE. See cmd/server/main.go:289.
// This code is complete but untested in production. Do not remove.
```

**Recommendation:** Option B unless The Architect says otherwise. The dispatcher code is clean but untested at scale. Don't wire it in during a fidelity sweep — that's a separate QA task.

### DP-443 — Donation/immortal/frozen start rooms unported (MEDIUM)

**C:** `src/config.c:142-161`
```c
int mortal_start_room = 8004;
int kiroshi_start_room = 18201;
int alaozar_start_room = 21258;
int immort_start_room = 1204;
int frozen_start_room = 1202;
int donation_room_1 = 8053;
int donation_room_2 = 18204;
int donation_room_3 = NOWHERE;
```

**Go:** None of these constants are referenced. The Go codebase has no concept of donation rooms, immortal start room, or frozen start room.

**Impact:**
- **Donation rooms:** Items dropped in rooms flagged as "donation" stay on the ground instead of routing to community chests. The donation system doesn't work.
- **Immortal start room:** Immortals spawn in the mortal start room (8004) instead of their dedicated room (1204).
- **Frozen start room:** Frozen (punished) players aren't contained in a penalty box — they spawn normally.

**Fix:** This requires:

1. **Add config constants** — Find where Go stores server config (likely `pkg/game/config.go` or loaded from a config file). Add:
   ```go
   const (
       ImmortStartRoom  = 1204
       FrozenStartRoom  = 1202
       DonationRoom1    = 8053
       DonationRoom2    = 18204
   )
   ```

2. **Wire donation rooms** — Find the code that handles `do_drop` or item dropping. When a player drops an item in a room whose VNUM matches a donation room, route the item to the designated donation room instead of the current room. Check `src/db.c` for how C handles donation room routing — it's likely in the `do_drop` function or a helper.

3. **Wire immortal start** — Find the login/respawn code. When a player logs in or respawns, check their level. If immortal (level ≥ LVL_IMMORT), place them in `ImmortStartRoom` instead of `mortal_start_room`.

4. **Wire frozen start** — Find the freeze/punishment code. When a frozen player logs in or respawns, place them in `FrozenStartRoom`.

**Note:** These are config values that affect game behavior. They should be constants, not hardcoded VNUMs scattered through the code. Centralize them.

---

## Execution Order

1. **DP-443 first** — donation/immortal/frozen rooms are independent config additions
2. **DP-453 second** — ZoneDispatcher decision (wire in or mark disabled)

## Verification

After all fixes:
```bash
cd darkpawns_repo
go build ./...
go vet ./...
go test ./...
```

For DP-443: verify the constants exist and are referenced in the appropriate login/drop code paths.
For DP-453: verify `go vet` passes and no dead code warnings if option B is chosen.
