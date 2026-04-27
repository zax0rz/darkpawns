# Pass 2: Concurrency & Data Integrity Review

**Reviewer:** Claude Opus 4 (automated)
**Date:** 2026-04-26
**Scope:** All concurrency primitives, goroutine lifecycle, data race vectors, lock discipline

## Executive Summary

The codebase has a **systemic concurrency problem**: Player.Send is a dead channel (never consumed), Session fields are mutated from multiple goroutines without synchronization, save.go snapshots Player state without any locking, and multiple global mutable maps (playerSneakState, playerHideState) have zero synchronization. The zone dispatcher and game loop are architecturally sound but several supporting goroutines (Spawner.StartPeriodicResets, StartPointUpdateTicker) have no shutdown mechanism. The combat engine has correct internal locking but its callback hooks (DamageFunc, ScriptFightFunc) execute on the combat goroutine while mutating Session state owned by the readPump goroutine, creating data races on every combat round.

---

## CRITICAL

### C1. Player.Send Channel Is Dead — Messages Silently Dropped

**Files:** `pkg/game/player.go:797`, `pkg/game/world.go:438`, `pkg/game/mob.go:175,181`, `pkg/game/other_session.go:59`

`Player.Send` is a buffered channel (256) created in `NewPlayer()`. Messages are written to it from:
- `Player.SendMessage()` (player.go:797) — select with default (drops on full)
- `World.SpawnMob()` (world.go:438) — direct `player.Send <-` (blocks if full)
- `MobInstance.AttackPlayer()` (mob.go:175,181) — direct `player.Send <-`
- `other_session.go:59` — `ch.Send <- []byte("CLOSE_CONNECTION")`

**Nobody reads from `Player.Send`.** The `writePump` goroutine reads from `Session.send` (a completely separate channel). There is no bridge between the two. Every message written to `Player.Send` either:
1. Silently drops (when using select/default)
2. Blocks forever, leaking the goroutine (when using direct `<-` send, e.g., `SpawnMob` at world.go:438)

**Impact:** Players never see mob spawn messages, mob attack messages, or the `CLOSE_CONNECTION` signal. The blocking sends in `SpawnMob` and `AttackPlayer` can permanently hang goroutines when the buffer fills.

**Fix:** Either:
- **Option A (simple):** Remove `Player.Send` entirely. Route all messages through `Session.send` via the manager's `BroadcastToRoom` or a `SendToPlayer(name, msg)` method.
- **Option B (bridge):** Add a goroutine per player that drains `Player.Send` into `Session.send`. But this adds complexity for no real benefit — Option A is cleaner.

---

### C2. save.go Reads Player Fields Without Any Locking

**File:** `pkg/game/save.go:130-200` (`playerToSaveData`)

`playerToSaveData(p *Player)` reads every field on the Player struct directly — `p.ID`, `p.Name`, `p.Health`, `p.Mana`, `p.Gold`, `p.Exp`, `p.RoomVNum`, `p.Stats`, `p.SpellMap`, `p.Inventory.Items`, `p.Equipment.Slots`, `p.ActiveAffects` — without acquiring `p.mu`.

`SavePlayer()` and `SavePlayerWithRent()` call `playerToSaveData()` without locking either.

This is called from:
- `Manager.Unregister()` (session/manager.go:261) — on disconnect, concurrent with game loop
- `CrashLoad()` (save.go:375) — during login, calls `SavePlayer()` and `SavePlayerWithRent()` while the player may already be in the world
- Auto-save callbacks (from game loop timer)

**Impact:** Torn reads on every Player field during save. The `SpellMap` (a Go map) and `Inventory.Items` (a slice) are particularly dangerous — concurrent read during a map write is a fatal panic in Go.

**Fix:**
```go
func playerToSaveData(p *Player) savePlayerData {
    p.mu.RLock()
    defer p.mu.RUnlock()
    // ... all field reads inside lock
}
```

Also need to protect `p.Inventory.Items` iteration — either copy under the player lock or add a lock to Inventory.

---

### C3. Session.dirtyVars / subscribedVars / pendingEvents Mutated From Multiple Goroutines

**Files:** `pkg/session/manager.go:165,773`, `pkg/session/agent_vars.go:70,77-83,91-98,168-169`

These Session fields are Go maps/slices with no synchronization:
- `s.dirtyVars` — written by `markDirty()` from `DamageFunc` (combat goroutine) and read/cleared by `flushDirtyVars()` from `handleCommand` (readPump goroutine)
- `s.subscribedVars` — written by `handleSubscribe()` (readPump goroutine), read by `markDirty()` (combat goroutine)
- `s.pendingEvents` — appended by `handleCommand()` (readPump), read/cleared by `flushDirtyVars()` (readPump) — these are actually same-goroutine, so safe. But `markDirty` from DamageFunc is the cross-goroutine vector.

The combat engine's `DamageFunc` callback (set at `manager.go:165`) runs on the combat ticker goroutine:
```go
m.combatEngine.DamageFunc = func(victimName string) {
    if s, ok := m.GetSession(victimName); ok {
        s.markDirty(VarHealth, VarMaxHealth)  // ← runs on combat goroutine
    }
}
```

`markDirty` reads `s.subscribedVars` and writes `s.dirtyVars` — both unprotected Go maps.

**Impact:** Concurrent map read/write = fatal runtime panic. This will crash the server during any combat round involving an agent session.

**Fix:** Add a `sync.Mutex` to Session for agent state, or use a channel to serialize mutations:
```go
type Session struct {
    agentMu        sync.Mutex
    subscribedVars map[string]bool
    dirtyVars      map[string]bool
    pendingEvents  []interface{}
}
```

---

### C4. Global Mutable Maps Without Synchronization: playerSneakState, playerHideState

**File:** `pkg/game/skills.go:627-650`

```go
var playerSneakState = make(map[string]bool)
var playerHideState = make(map[string]bool)

func IsSneaking(name string) bool { return playerSneakState[name] }
func SetSneaking(name string, val bool) { playerSneakState[name] = val }
func IsHidden(name string) bool { return playerHideState[name] }
func SetHidden(name string, val bool) { playerHideState[name] = val }
```

These are package-level Go maps accessed from:
- Command handlers (readPump goroutines — one per player)
- MobileActivity (AI ticker goroutine)
- Zone dispatcher goroutines (mob AI checks visibility)

Concurrent map access in Go is a fatal runtime panic.

**Fix:** Either:
- Add a `sync.RWMutex` guarding both maps
- Move sneak/hide state onto the Player struct (which already has `p.mu`)
- Use `sync.Map`

The Player struct already has an `Affects` bitmask with `AFF_SNEAK` and `AFF_HIDE` bits. These global maps are redundant — remove them and use the existing affect flags.

---

## HIGH

### H1. World.SendToZone and World.SendToAll Iterate players Without Lock

**File:** `pkg/game/world.go:333-347`

```go
func (w *World) SendToZone(roomVNum int, msg string) {
    room := w.GetRoomInWorld(roomVNum)  // holds+releases w.mu.RLock
    // ... no lock held ...
    for _, p := range w.players {       // ← w.players accessed without lock
        pr := w.GetRoomInWorld(p.RoomVNum) // ← p.RoomVNum accessed without p.mu
        // ...
    }
}

func (w *World) SendToAll(msg string) {
    for _, p := range w.players {  // ← no lock
        p.SendMessage(msg)
    }
}
```

`w.players` is a Go map mutated by `AddPlayer`/`RemovePlayer` under `w.mu.Lock()`. Iterating it without `w.mu.RLock()` races with any player login/logout. `p.RoomVNum` is accessed directly instead of via `p.GetRoom()`.

**Fix:** Hold `w.mu.RLock()` for the iteration, use `p.GetRoom()` for field access.

---

### H2. mobact.go Accesses MobInstance.RoomVNum Directly, Bypassing Mutex

**File:** `pkg/game/mobact.go:116,129,139,148,161,206,212,227,233,241,252,258,274`

`MobileActivity()` correctly snapshots the mob list under `w.mu.RLock()` but then accesses `ch.RoomVNum` directly (15 occurrences) instead of using `ch.GetRoom()`. Since `GetRoom()` acquires `m.mu.RLock()`, the direct field access races with `SetRoom()` calls from other goroutines (zone dispatcher, combat movement, player commands).

**Fix:** Replace all `ch.RoomVNum` with `ch.GetRoom()` in mobact.go.

---

### H3. PointUpdate Reads Player.Flags Without Lock

**File:** `pkg/game/limits.go:455`

```go
if p.Flags&(1<<PrfInactive) == 0 {
```

Reads `p.Flags` without holding `p.mu`. This runs on the PointUpdate timer goroutine while commands on the readPump goroutine may be writing to `p.Flags` via `SetPlrFlag()`.

**Fix:** Use `p.GetFlags()` (which acquires `p.mu.RLock()`).

---

### H4. Multiple s.send Close Paths — Double-Close Panic Risk

**File:** `pkg/session/manager.go:267,843,1054,1209`

`s.send` is closed in four places:
1. `Unregister()` line 267
2. `CloseSend()` line 843
3. `UnregisterAndClose()` line 1054
4. `CheckIdlePasswords()` line 1209

The `readPump()` defer calls `Unregister()`. If something also calls `UnregisterAndClose()` or `CloseSend()`, the channel is closed twice → runtime panic.

There is also no protection against writing to a closed channel. After `Unregister()` closes `s.send`, any lingering `BroadcastToRoom` call that finds the session (before it's removed from the map) will panic on `s.send <- message`.

**Fix:**
- Use `sync.Once` for channel close:
  ```go
  type Session struct {
      sendOnce sync.Once
      send     chan []byte
  }
  func (s *Session) closeSend() {
      s.sendOnce.Do(func() { close(s.send) })
  }
  ```
- All close paths call `s.closeSend()` instead of direct `close(s.send)`.
- `BroadcastToRoom` should use `select` with `default` (which it does in some paths but not all — `sendWelcome` at line 692 uses direct `s.send <- msg`).

---

### H5. Spawner.StartPeriodicResets — Goroutine Leak (No Shutdown)

**File:** `pkg/game/spawner.go:580-588`

```go
func (s *Spawner) StartPeriodicResets(interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            s.resetEmptyZones()
        }
    }()
}
```

This goroutine has no done channel, no context, and no way to stop. The ticker is never stopped. On server shutdown, this goroutine leaks.

Same issue in `StartPointUpdateTicker` (ai.go:204-214) — uses `w.done` but `StartAITicker` also uses `w.done` with a single `close()`. If `StopAITicker` runs, `w.done` closes, which stops the PointUpdate goroutine too. But if `StartPeriodicResets` is called, it has no such channel.

**Fix:** Add a `stopCh` to `Spawner` and select on it in the goroutine, or use `context.Context`.

---

### H6. World.SpawnMob Sends on Player.Send While Holding w.mu.Lock

**File:** `pkg/game/world.go:434-440`

```go
func (w *World) SpawnMob(vnum int, roomVNum int) (*MobInstance, error) {
    w.mu.Lock()
    defer w.mu.Unlock()
    // ...
    for _, player := range w.players {
        if player.GetRoom() == roomVNum {
            player.Send <- []byte(...)  // BLOCKING send while holding world lock
        }
    }
}
```

This is a **blocking channel send** while holding `w.mu.Lock()`. If `Player.Send` is full (256 buffer) and nobody reads it (see C1 — nobody does), this will deadlock the entire world. Any other goroutine trying to acquire `w.mu` will block forever.

**Fix:** Remove the send (it goes to the dead channel anyway), or send via `player.SendMessage()` (which uses select/default), or move the notification outside the lock.

---

## MEDIUM

### M1. Combat Engine PerformRound Snapshots Under Write Lock

**File:** `pkg/combat/engine.go:193-206`

```go
func (ce *CombatEngine) PerformRound() {
    ce.mu.Lock()
    pairs := make([]*CombatPair, 0, len(ce.combatPairs))
    for _, pair := range ce.combatPairs {
        pairs = append(pairs, pair)
    }
    ce.mu.Unlock()
    // processes pairs without lock
}
```

Uses a write lock (`Lock()`) for what is a read-only operation. Should use `RLock()`. The comment says "to prevent TOCTOU races" but the snapshot itself doesn't mutate anything.

**Fix:** Use `ce.mu.RLock()` / `ce.mu.RUnlock()`.

---

### M2. handleDeath Acquires ce.mu.RLock After Already Being Called From PerformRound

**File:** `pkg/combat/engine.go:283-295`

```go
func (ce *CombatEngine) handleDeath(victim, killer Combatant) {
    // ...
    ce.mu.RLock()           // ← tries to read-lock
    for key, pair := range ce.combatPairs {
        // ...
    }
    ce.mu.RUnlock()
    ce.DeathFunc(...)       // calls game layer
}
```

`handleDeath` is called from `processCombatPair`, which is called from `PerformRound`. `PerformRound` releases the lock before calling `processCombatPair`, so this specific path works. However, `processCombatPair` also calls `ce.StopCombat()` which acquires `ce.mu.Lock()`. After `StopCombat` returns, `handleDeath` tries `ce.mu.RLock()` — this succeeds because StopCombat released its lock. No deadlock here, but the interleaving is fragile and should be documented.

---

### M3. TickManager.SetTickInterval Race With tickLoop

**File:** `pkg/engine/affect_tick.go:63-73`

```go
func (tm *TickManager) SetTickInterval(interval time.Duration) {
    tm.mu.Lock()
    defer tm.mu.Unlock()
    tm.tickInterval = interval
    if tm.running && tm.ticker != nil {
        tm.ticker.Stop()
        tm.ticker = time.NewTicker(tm.tickInterval)  // new ticker
    }
}
```

The `tickLoop` goroutine reads `tm.ticker.C`. When `SetTickInterval` replaces `tm.ticker`, the old channel is abandoned but `tickLoop` may still be blocked on the old `<-tm.ticker.C`. The new ticker's channel is never consumed by the goroutine until the old select unblocks (which it won't, since the old ticker is stopped).

**Fix:** Signal `tickLoop` to restart its select, or use a channel to send the new interval.

---

### M4. Event Bus Unsubscribe Compares Function Pointers Incorrectly

**File:** `pkg/events/bus.go:58-67`

```go
func (b *InProcessBus) Subscribe(eventType string, handler Handler) func() {
    // ...
    return func() {
        for i, h := range handlers {
            if &h == &handler {  // ← comparing addresses of loop variable copies
```

This comparison `&h == &handler` compares the address of the loop iteration variable `h` with the address of the closure's captured `handler`. These will never be equal — `h` is a stack-local copy. The unsubscribe function is broken; handlers can never be removed.

**Fix:** Use an ID-based subscription system or compare via a wrapper struct with an ID field.

---

### M5. immortalSessionProvider Global Assigned Without Synchronization

**File:** `pkg/game/logging.go:204,210-212`

```go
var immortalSessionProvider ImmortalSessionProvider

func SetImmortalSessionProvider(provider ImmortalSessionProvider) {
    immortalSessionProvider = provider  // no lock
}

func getImmortalSessionProvider() ImmortalSessionProvider {
    return immortalSessionProvider  // no lock
}
```

Written at startup, read from MudLog on any goroutine. While startup-only writes are often safe in practice, Go's memory model doesn't guarantee visibility without synchronization.

**Fix:** Use `atomic.Pointer[ImmortalSessionProvider]` or a `sync.Once`.

---

### M6. Weather/Time Global State Lock Discipline Is Good But Called From Unlocked World Methods

**File:** `pkg/game/weather.go:76-95`

The `weatherMu` protects `timeInfo` and `weatherInfo` correctly. However, `GetTimeInfo()` and `GetWeatherInfo()` (if they exist) need to also hold the lock. Check callers.

*Verified: weather.go uses `weatherMu.Lock()` in `WeatherAndTime()` and `weatherMu.RLock()` in accessor functions. This is correct.*

---

## LOW

### L1. GoldMu on Player Is Redundant

**File:** `pkg/game/player.go:30`

```go
GoldMu sync.Mutex // Mutex for Gold field (C6)
```

`Player` already has `mu sync.RWMutex` that protects all fields. `GoldMu` is a second lock for the same data. If code acquires `p.mu.Lock()` and then `p.GoldMu.Lock()`, or vice versa, this is a deadlock hazard. If code only uses `GoldMu` for Gold, then Gold is unprotected from the broader `p.mu` readers.

**Fix:** Remove `GoldMu`. Use `p.mu` consistently for all Player fields.

---

### L2. SpecRegistry Map Written During init(), Read During Runtime

**File:** `pkg/game/spec_assign.go:391-395`

`SpecRegistry` is populated by `init()` calls and `RegisterSpec()`. If registration only happens during startup (before goroutines start), this is safe. But if any registration can happen after the server is running, it's a race.

**Fix:** Either document that registration must complete before `Start()`, or use `sync.Map`.

---

### L3. math/rand Usage Without Seed in Tests

**File:** `pkg/session/manager.go:868`

Uses `rand.Intn()` — in Go 1.20+ this auto-seeds. Not a concurrency issue, but noted.

---

### L4. Zone Dispatcher Worker Tick Counter Not Atomic

**File:** `pkg/game/zone_dispatcher.go:35,101`

```go
type zoneWorker struct {
    ticks  uint64  // written in zoneLoop goroutine
}

func (zd *ZoneDispatcher) ZoneTicks(zoneNum int) uint64 {
    zd.mu.RLock()
    defer zd.mu.RUnlock()
    if w, ok := zd.zones[zoneNum]; ok {
        return w.ticks  // read from caller goroutine
    }
}
```

`w.ticks` is written by the zone goroutine (line 101: `worker.ticks++`) and read by `ZoneTicks()` from another goroutine. The `zd.mu.RLock` protects the map access but doesn't synchronize the `ticks` field.

**Fix:** Use `atomic.Uint64` for `ticks`.

---

## Prioritized Summary Table

| ID | Severity | Title | Risk | Effort |
|----|----------|-------|------|--------|
| C1 | CRITICAL | Player.Send dead channel — messages lost, blocking sends deadlock | Server hang | Medium |
| C2 | CRITICAL | save.go reads Player without lock — torn reads, map panic | Data corruption, crash | Low |
| C3 | CRITICAL | Session agent fields mutated cross-goroutine — map panic | Server crash | Low |
| C4 | CRITICAL | playerSneakState/playerHideState unsync global maps | Server crash | Low |
| H1 | HIGH | SendToZone/SendToAll iterate w.players without lock | Crash | Low |
| H2 | HIGH | mobact.go direct field access bypasses mutex | Data race | Low |
| H3 | HIGH | PointUpdate reads p.Flags without lock | Data race | Low |
| H4 | HIGH | Multiple s.send close paths → double-close panic | Server crash | Low |
| H5 | HIGH | Spawner goroutine leak — no shutdown mechanism | Resource leak | Low |
| H6 | HIGH | SpawnMob blocking send while holding world lock | Deadlock | Low |
| M1 | MEDIUM | PerformRound uses write lock for read-only snapshot | Contention | Trivial |
| M2 | MEDIUM | handleDeath lock ordering fragile | Maintenance | Low |
| M3 | MEDIUM | SetTickInterval replaces ticker without notifying goroutine | Stale ticker | Medium |
| M4 | MEDIUM | Event bus unsubscribe broken (pointer comparison) | Memory leak | Low |
| M5 | MEDIUM | immortalSessionProvider no sync | Visibility | Trivial |
| L1 | LOW | Redundant GoldMu on Player | Deadlock risk | Trivial |
| L2 | LOW | SpecRegistry init-time safety | Documentation | Trivial |
| L4 | LOW | Zone worker ticks not atomic | Data race | Trivial |

**Recommended fix order:** C1 → C2 → C4 → C3 → H4 → H6 → H1 → H2 → H3 → H5 → rest.

C1 is the root cause of the "dual send channel" issue identified in pass 1. Fixing it (removing Player.Send, routing everything through Session.send via the manager) will also fix H6 and simplify the entire message-passing architecture.
