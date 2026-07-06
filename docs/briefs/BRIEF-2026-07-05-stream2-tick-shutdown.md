# Brief: Stream 2 — Tick & Shutdown Hygiene — 2026-07-05

**Workspace:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.
**Agent:** Kimi k2.6/k2.7 (primary) or Gemini 3.5-flash (fallback)

---

## Fix 1: DP-946 — Shutdown leaves AI/point/reset tickers running during SaveWorld (Urgent)

**File:** `cmd/server/main.go` — shutdown sequence (lines 410-442)
**Also:** `pkg/game/world.go` (StopAITicker at line 929), `pkg/game/ai.go` (point ticker at line 206), `pkg/game/spawner.go` (StopPeriodicResets at line 686)

**Problem:**
The shutdown sequence in main.go stops the game loop, telnet, HTTP, sessions, and waits for the zone reset WaitGroup — but **three ticker families keep running during `game.SaveWorld`** at line 438:

1. **AI ticker** (10s interval, `pkg/game/ai.go:181-201`) — calls `AITick()` which runs mob AI, potentially mutating rooms/mobs
2. **Point update ticker** (30s interval, `pkg/game/ai.go:206-219`) — calls `PointUpdate()` which modifies player HP/hunger/thirst (regen)
3. **Periodic zone resets** (60s interval, `pkg/game/spawner.go:670-684`) — calls `resetEmptyZones()` which respawns mobs and resets zone state

The stop functions exist but are **never called in production**:
- `StopAITicker()` (world.go:929-934) — closes `w.done` channel, which stops **both** the AI ticker and the point update ticker (they both select on `<-w.done`)
- `StopPeriodicResets()` (spawner.go:686-691) — closes `s.done` channel, stops the periodic reset goroutine

Current shutdown order:
```
Line 413: gameLoop.Stop()         // stops engine heartbeat, NOT the standalone tickers
Line 416: telnet.Stop()
Line 421: srv.Shutdown(ctx)
Line 426: manager.ShutdownGracefully()
Line 431: decisionLogWriter.Stop()
Line 435: wg.Wait()               // waits for initial zone reset goroutine only
Line 438: game.SaveWorld()         // ← tickers still mutating world here!
```

This is the same class as the clawpatch fix (commit 02b452e) which stopped the engine game loop from mutating world during shutdown. That fix only covered the engine loop — these three standalone ticker goroutines were missed.

**Fix:**
Add two calls between `gameLoop.Stop()` and `telnet.Stop()`, before any session draining:

```go
// 1. Stop heartbeat callbacks before draining sessions or saving world state.
gameLoop.Stop()

// 1a. Stop AI ticker and point update ticker (both share w.done channel)
gameWorld.StopAITicker()

// 1b. Stop periodic zone reset ticker
gameWorld.StopPeriodicResets()

// 2. Stop telnet listener...
```

Implementation details:
1. `gameWorld.StopAITicker()` already exists and stops both AI + point tickers (they share `w.done`). No new code needed for this path.
2. `StopPeriodicResets()` — this exists on `Spawner` but **not as a method on `World`**. Add a `World.StopPeriodicResets()` wrapper method:
   ```go
   // In pkg/game/world_zone.go (next to StartPeriodicResets at line 157)
   func (w *World) StopPeriodicResets() {
       if w.spawner != nil {
           w.spawner.StopPeriodicResets()
       }
   }
   ```
3. Insert both calls in main.go after line 413 (`gameLoop.Stop()`).
4. Optionally add a brief sleep (e.g., `time.Sleep(100 * time.Millisecond)`) after stopping tickers to let in-flight ticks complete before proceeding. This is defensive — the tickers select on `<-done` which exits immediately, but an in-flight `AITick()` or `PointUpdate()` may still be mid-execution on its goroutine.

**Cite:** No direct C equivalent — C's single-threaded shutdown simply stops calling the main game loop. Go's goroutine model requires explicit stop signals. The shutdown hygiene pattern follows the clawpatch fix principle: **no goroutine may mutate world state while SaveWorld is serializing it**.

**Deviation:** Go-only shutdown hygiene fix. No C fidelity concern.

**Regression Test:** `pkg/game/world_test.go`
- `TestStopAITickerStopsBothTickers`: start a World, verify AI tick and point update tick fire, call `StopAITicker()`, sleep briefly, assert neither fires again (use a sync/atomic counter).
- `TestStopPeriodicResetsStopsTicker`: start periodic resets, verify they fire, call `StopPeriodicResets()`, assert no more fires.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-947 — PointUpdate double-driven — regen/hunger ticks at ~3.5× C rate (Medium)

**File:** `pkg/game/world.go` (line 193 — NewWorld starts 30s ticker), `cmd/server/main.go` (lines 220-223 — gameLoop callback)
**Also:** `pkg/game/ai.go` (line 204-205 — "faster tick" comment)

**Problem:**
`PointUpdate()` (which handles player regen, hunger, thirst) is called from **two independent tickers**:

1. **Standalone ticker** (world.go:193 → `StartPointUpdateTicker(30 * time.Second)` in ai.go:206): fires every 30 seconds
2. **GameLoop callback** (main.go:220-223 → `OnPointUpdate`): fires every 75 real seconds (`SECS_PER_MUD_HOUR * PASSES_PER_SEC` = 75 * 10 = 750 pulses)

The game loop heartbeat fires at 100ms intervals. `PASSES_PER_SEC = 10` (engine constants). `SECS_PER_MUD_HOUR = 75`. So the callback fires every 750 pulses = 75 seconds.

Both tickers call `gameWorld.PointUpdate()` independently. Net effect: ~3.5× the C regen rate (C fires point_update every 75 pulses = 75 seconds, once).

The `ai.go:204-205` comment says:
```
// Source: limits.c point_update() — called every ~75 pulses in stock CircleMUD.
// Dark Pawns uses a faster tick (30 seconds).
```

**This comment indicates the 30s tick was a deliberate design choice.** But having both tickers active means the deliberate "faster tick" is actually faster than intended — it's the 30s tick PLUS the 75s tick, not the 30s tick instead of the 75s tick.

**Fix:**
Pick one driver. Given the comment says "Dark Pawns uses a faster tick (30 seconds)", remove the gameLoop callback:

1. In `cmd/server/main.go:220-223`, **remove** the `OnPointUpdate` callback:
   ```go
   // Before:
   gameLoop := engine.NewGameLoop(engine.GameLoopCallbacks{
       OnPointUpdate: func() {
           gameWorld.PointUpdate()
       },
       ...
   })

   // After:
   gameLoop := engine.NewGameLoop(engine.GameLoopCallbacks{
       // OnPointUpdate removed — handled by World.StartPointUpdateTicker (30s)
       ...
   })
   ```
2. Verify `engine.NewGameLoop` allows `OnPointUpdate` to be nil/omitted without panicking. Check `pkg/engine/gameloop.go:257-267` — the callback is guarded by `if cb.OnPointUpdate != nil`.
3. Update the comment in `ai.go:204-205` to clarify this is the sole driver.

**Alternative:** If C fidelity is preferred (75s tick matching CircleMUD), remove the NewWorld ticker instead and keep the gameLoop callback. **Recommendation: keep the 30s tick** — it's a deliberate design deviation for faster-paced gameplay, and removing the duplicate is what matters.

**Cite:** C source — `limits.c:point_update()` is called once per 75 pulses (~75 real seconds). This is the only driver in C. Having two drivers is a Go-specific bug.

**Deviation:** If keeping the 30s tick, this is a documented design deviation from C. The deviation is the 30s cadence (faster regen). The *bug* is having two drivers, not the cadence itself.

**Regression Test:** `pkg/game/ai_test.go`
- `TestPointUpdateSingleDriver`: create a World (without starting gameLoop), verify `PointUpdate` fires exactly once per 30s interval. Use an atomic counter inside a test override.
- Alternatively: if this is hard to test deterministically, just verify the build passes and both `go vet` and `go test` are clean. The fix is a removal, not new logic.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 3: DP-948 — ZoneDispatcher implemented but unwired — decide wire or delete (Low)

**File:** `pkg/game/zone_dispatcher.go` (179 lines, fully implemented), `cmd/server/main.go:339-341` (commented out)

**Problem:**
`ZoneDispatcher` is a per-zone goroutine dispatcher that was implemented as a replacement for the serial `StartPeriodicResets`. It exists at `zone_dispatcher.go` with `NewZoneDispatcher`, `Start`, `Stop`, per-zone goroutines, diagnostic methods. The file header says:

```
// ZoneDispatcher: NOT YET ACTIVE. See cmd/server/main.go for the StartZoneDispatcher() call site.
// This code is complete but untested at scale. Do not remove.
// Wire in StartZoneDispatcher() when ready to replace StartPeriodicResets().
```

`NewZoneDispatcher` IS called in `NewWorld` (world.go:184-186), but `.Start()` is never called anywhere. Production uses `StartPeriodicResets` (serial, single-goroutine) at 60s intervals.

The DP-904 ratchet (dead code cleanup) argues for a decision: wire it in or delete it. Limbo is worse than either choice.

**Fix (recommended): DELETE the ZoneDispatcher.**

Rationale:
1. The serial `StartPeriodicResets` works fine at current scale
2. `ZoneDispatcher` is "untested at scale" — wiring it in would require load testing
3. Per-zone goroutines add complexity (N goroutines for N zones) with unclear benefit
4. The U1000 ratchet says: decide, don't defer

Steps:
1. Delete `pkg/game/zone_dispatcher.go`
2. Remove `NewZoneDispatcher` call from `pkg/game/world.go:184-186` and the `zoneDispatcher` field from the `World` struct
3. Remove any `ZoneDispatcher` references from `world.go`, `world_zone.go`, and test files
4. Clean up the comment at `main.go:339-341` (remove the ZoneDispatcher mention)

**Alternative (not recommended):** Wire it in by calling `gameWorld.zoneDispatcher.Start()` and replacing `StartPeriodicResets`. This is more work and risk for unclear benefit. Only do this if Zach explicitly wants to keep it.

**Cite:** No C equivalent — CircleMUD uses a single-threaded zone reset loop. Both the serial `StartPeriodicResets` and `ZoneDispatcher` are Go-original implementations.

**Deviation:** N/A — this is dead code cleanup.

**Regression Test:**
- Remove any tests for `ZoneDispatcher` (if they exist in `zone_dispatcher_test.go`, delete that file too)
- Verify `go test ./...` still passes after deletion
- Verify `go build ./...` has no compile errors (no dangling references)

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. **Fix 1 (DP-946)** — Shutdown ticker stops. Highest impact — prevents world save corruption. Add the two stop calls + World wrapper method.
2. **Fix 2 (DP-947)** — Remove duplicate PointUpdate driver. Remove the gameLoop callback. Depends on Fix 1 being in place (so the standalone ticker is the sole driver and gets stopped during shutdown).
3. **Fix 3 (DP-948)** — Delete ZoneDispatcher dead code. Independent cleanup. Can be done last.

**Suggested batch order:** 1 → 2 → 3

---

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace/darkpawns_repo
git checkout -b fix/stream2-tick-shutdown
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: tick & shutdown hygiene — stop tickers before SaveWorld, remove duplicate PointUpdate, delete ZoneDispatcher (DP-946, DP-947, DP-948)"
git push -u origin fix/stream2-tick-shutdown
```

Wait for review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-946: Add comment "Fixed — StopAITicker + StopPeriodicResets called before SaveWorld in shutdown sequence", commit <hash>, move to Done
- DP-947: Add comment "Fixed — removed duplicate OnPointUpdate callback from gameLoop; standalone 30s ticker is sole driver", commit <hash>, move to Done
- DP-948: Add comment "Fixed — deleted ZoneDispatcher dead code (untested at scale, serial resets work fine)", commit <hash>, move to Done
