# Go Code Quality Review

**Date:** 2026-05-03
**Files Reviewed:** ~45 of 270 (prioritized core + sample)
**Reviewer:** BRENDA69 (subagent)

## Summary

This is a **remarkably faithful C-to-Go port** of a 1990s MUD codebase. The translation effort is extensive and technically competent — the original C semantics are preserved, concurrency safety has been retrofitted, and the code runs. However, it reads like C written with Go syntax more than idiomatic Go. The project is in an intentional transitional state: fidelity to original behavior is prioritized over Go idiom, which is defensible for a preservation port but creates technical debt that will compound.

**Key themes:**
- Heavy use of `interface{}` to break import cycles instead of proper interface design
- C patterns preserved literally (name-based lookups, bitmasks, numeric constants)
- Mutex discipline is mostly correct but inconsistent
- Error handling is present but shallow — many errors are logged and dropped
- Zero panics found, which is excellent
- Tests exist but are thin — heavy on "does it run" and light on "does it behave correctly"

---

## Critical Go Anti-Patterns

### 1. `interface{}` as a Crutch for Circular Dependencies
**Severity: HIGH**

There are **299 uses of `interface{}`** across the codebase. Many exist solely to break import cycles:

- `pkg/spells/say_spell.go:10`: "Parameters use interface{} since this package cannot import game/combat directly"
- `pkg/game/world.go:353`: `ForEachPlayerInZoneInterface` — accepts `func(p interface{})` so spells package can iterate without importing `game`
- `pkg/common/interfaces.go` doesn't exist — the common package has types scattered across files instead of a coherent interface layer

**Why this is wrong:** Go's type system is being bypassed instead of leveraged. The correct fix is to define narrow interfaces in a shared package (or `common/`) that both sides can import. `Combatant` in `pkg/combat` is a good example of the right pattern — but it's 25 methods wide and itself noted as needing decomposition (see `combatant.go` ARCHITECTURAL NOTE [M-05]).

### 2. Name-Based Entity Resolution
**Severity: HIGH**

```go
// combatant.go
SetFighting(target string)
GetFighting() string
```

Combat targets, followers, and message routing all use **player names as unique identifiers**. This is a direct C port (`ch->master` was a pointer; in Go it became a string). Problems:
- Renaming a player breaks all in-flight references
- No way to distinguish same-named mobs
- Race conditions between name lookup and entity removal
- The combat engine has to do O(n) scans to find targets by name

The `combatant.go` file itself documents this as deferred technical debt. It should use stable instance IDs.

### 3. Global Mutable State in `weather.go`
**Severity: MEDIUM**

```go
var (
    timeInfo   = TimeInfoData{...}
    weatherInfo = WeatherData{...}
    weatherMu   sync.RWMutex
    weatherWorld *World
)
```

Package-level mutable state with a mutex is a C pattern. In Go, this should be a `WeatherSystem` struct owned by `World`. The `SetWeatherWorld()` function is a workaround for not having the state in the right place.

### 4. `sync.Mutex` Inside `sync.RWMutex` World Lock
**Severity: MEDIUM**

`Player` has its own `mu sync.RWMutex`. `World` has `mu sync.RWMutex`. Operations like `handlePlayerDeath` in `death.go` acquire `player.mu.Lock()` while `World` methods may hold `w.mu.RLock()`. The documented lock ordering says:

> "Equipment.mu is safe to acquire here; no ordering hierarchy places it above player.mu"

But there's no formal hierarchy document. With 55 mutexes across the codebase, this is a deadlock waiting to happen. The `ExtractObject` comment in tests documents that carrier removal doesn't work for `*Inventory` carriers — this kind of partial correctness is dangerous.

### 5. Swallowed Errors in Hot Paths
**Severity: MEDIUM**

```go
// session/manager.go
default:
    // Channel full, drop
```

Messages are silently dropped when the WebSocket send channel is full. This is a correctness issue for a MUD — players missing combat messages or death notifications because of backpressure is unacceptable. At minimum, this should be logged at `Warn` level.

```go
// death.go
if err := w.MoveObjectToRoom(corpse, roomVNum); err != nil {
    slog.Warn("...", "error", err)
}
```

Corpse placement failure is logged but the game continues. A corpse with a player's entire inventory just vanished.

---

## Common Patterns to Fix

### 1. Magic Numbers Everywhere
The codebase is saturated with magic numbers from the C port:

```go
// death.go
if attackType == 93 { // SPELL_DISINTEGRATE
// act_movement.go
if ch.GetLevel() >= lvlImmort { return true }
// mobprogs.go
case vnum == 13108:
```

These should be `const` blocks with names. The `death.go` file actually does this well for corpse attack types — but then falls back to raw integers in the switch.

### 2. C-Style Bitmask Operations
```go
if playerKiller.Flags&(1<<PrfAutoLoot) != 0 {
```

Go has `bits` package and typed constants. A `PlayerFlags` type with methods like `HasFlag(PrfAutoLoot)` would be idiomatic and readable.

### 3. `nolint:unused` Annotations
Dozens of functions are marked `//nolint:unused` because they're ported C functions not yet wired to the command registry. This is technical debt snowballing. Either wire them or delete them. The `act_movement.go` file starts with `//nolint:unused // Complete C port — command handlers not yet wired to registry.`

### 4. String Concatenation in Message Building
```go
msg := fmt.Sprintf("%s %s", mob.GetName(), args)
```

For one-off messages this is fine, but the MUD generates hundreds of messages per second. `strings.Builder` would be more appropriate for multi-line room descriptions and bulk message assembly.

### 5. `make([]T, 0)` Instead of Pre-allocation
```go
var result []parser.Room
for _, r := range w.rooms {
    result = append(result, *r)
}
```

`world.go:Rooms()` and similar patterns repeatedly under-allocate slices. With thousands of rooms, this causes unnecessary GC pressure.

---

## Package-by-Package Notes

### `pkg/game` — The Core
- **world.go (1042+ lines):** Too large. Does world state, mob commands, item management, messaging, and zone dispatch. Should be split: `world_state.go`, `world_mobs.go`, `world_items.go`, `world_messages.go`.
- **player.go:** 90+ fields. The `Player` struct is a god object carrying every possible character state. The C original had this problem too. At minimum, group related fields into embedded structs.
- **death.go:** Well-commented C port. Lock ordering in `handlePlayerDeath` is explicit and appreciated. The `makeCorpse`/`makeDust` split is clean.
- **save.go:** Good use of a separate `savePlayerData` struct for serialization. Clean separation of concerns.
- **mobprogs.go:** Hardcoded vnum checks (`vnum == 8014`, `vnum == 13108`) are unmaintainable. These should be data-driven (JSON config or Lua triggers).
- **act_movement.go:** 769+ lines of mostly unwired commands. The `nolint:unused` blanket at the top is a red flag.

### `pkg/session` — Connection Management
- **manager.go:** Clean separation between WebSocket concerns and game logic. The `MessageSink`/`CloseConn` callback pattern is a good decoupling mechanism.
- **commands.go:** Global `cmdRegistry` initialized in `init()`. This makes testing hard — you can't create an isolated command registry per test. The `init()` function registers ~80 commands unconditionally.
- Per-IP connection tracking is correct. Login attempt throttling is correct. Wizlock state uses its own mutex (good — doesn't block session map).

### `pkg/combat` — Combat Engine
- **engine.go:** Combat ticker at 2s intervals is simple and correct. `CombatPairKey` using names instead of IDs is the same problem as `Combatant`.
- **combatant.go:** The 25-method `Combatant` interface is explicitly flagged by its own authors as too large. The decomposition suggestion in the comment (`Damageable`, `Positionable`, `Skilled`, `Identifiable`) should be implemented.
- No tests for `CombatEngine` itself — the `unit/combat_test.go` only tests `CalculateHitChance` and `CalculateDamage`.

### `pkg/scripting` — Lua Engine
- **engine.go:** Excellent sandboxing. The `newSafeLState` function properly strips dangerous Lua functions. The panic recovery + state recreation pattern is robust.
- **Single shared `LState`:** All scripts run in one Lua VM with a global mutex. This serializes all script execution. For a MUD with hundreds of mobs running scripts, this will bottleneck. Consider a pool of `LState`s.
- 2647+ lines in one file. Should be split: `engine_core.go`, `engine_lua_api.go`, `engine_transit.go`.

### `pkg/moderation` — Moderation System
- **manager.go:** Good separation of DB vs in-memory fallback. Cleanup goroutine is correct.
- `hasPenalty` is private and documented as untestable. This is a smell — either export it for tests or test through the public interface.
- Word filter regex compilation is done on every check (see `matches()`). Should compile regexes once at load time.

### `pkg/command` — Command System
- **admin_commands.go:** In-memory `reports` slice with `reportsMu` is correct for a zero-DB fallback. But `reportSeq` is an unprotected `int` — needs atomic or mutex.
- The `cmdReport` function creates an `AbuseReport` struct and... discards it:
  ```go
  _ = moderation.AbuseReport{...}
  ```
  This is clearly a bug — the report is never submitted to the DB moderation manager.

### `pkg/parser` — World Data Parsing
- No reviewed issues. The parser appears clean.

### `pkg/events` — Event Queue
- Uses `time.Ticker` correctly. No reviewed issues.

### `pkg/common` — Shared Types
- **Missing `interfaces.go`:** The common package should define the narrow interfaces that break the `interface{}` cycle (`ScriptablePlayer`, `ScriptableMob`, etc. are defined in `scripting` but should be here).

---

## Testing Assessment

### Coverage: Thin
17 test files for 270 source files (~6%). Key gaps:

| Area | Tests | Quality |
|------|-------|---------|
| Object movement | `object_movement_test.go` | Good — documents surprising behavior, regression-focused |
| Combat formulas | `unit/combat_test.go` | Weak — only tests that functions return non-zero |
| Lua scripting | `integration_test.go` | Weak — mostly "does it parse", not "does it behave" |
| Moderation | `manager_test.go` | Good — table-driven, covers censor/block/spam |
| Door system | `door_test.go` | Good — table-driven, covers states |
| World serialization | `save_world_test.go` | Good — validates JSON round-trip |
| Event queue | `queue_test.go` | Not reviewed |
| Auth/ratelimit | `ratelimit_test.go` | Not reviewed |

### What's Missing
- **Combat engine integration tests:** No tests for `StartCombat` → `PerformRound` → `StopCombat` lifecycle
- **Session manager tests:** No tests for WebSocket upgrade, login flow, or session takeover
- **World integration tests:** No tests for multi-player interaction (two players in same room, combat between players)
- **Death/respawn tests:** `death.go` has zero tests
- **Mob AI tests:** No tests for mob behavior, spawner, or zone reset
- **Error path tests:** Most tests only test happy paths

### Test Pattern Quality
- Table-driven tests are used where appropriate (door_test, moderation_test)
- Mock `mockCombatant` in `unit/combat_test.go` is a proper struct with method implementations
- `newTestWorld(t)` helper in `object_movement_test.go` is well-designed with `t.Cleanup`

---

## Top 10 Recommendations

### 1. Define Narrow Interfaces in `pkg/common` (HIGH)
Break the `interface{}` cycle. Move `ScriptablePlayer`, `ScriptableMob`, `ScriptableObject` interfaces from `scripting` to `common`. Define `PlayerLister`, `MobLister`, `RoomAccessor` interfaces there. Both `game` and `spells` can import `common` without cycles.

### 2. Replace Name-Based Lookups with Instance IDs (HIGH)
Change `SetFighting(target string)` to `SetFighting(targetID int)`. Add `GetID() int` to `Combatant`. This fixes correctness, enables O(1) lookups, and eliminates an entire class of race conditions.

### 3. Split `pkg/game/world.go` (HIGH)
1000+ line files are unmaintainable. Split into:
- `world_state.go` — maps, IDs, getters/setters
- `world_mobs.go` — mob lifecycle, spawning
- `world_items.go` — object movement, containers
- `world_messages.go` — SendToAll, SendToZone, broadcasts

### 4. Fix the Admin Report Bug (HIGH)
In `command/admin_commands.go:cmdReport`, the `AbuseReport` struct is created and discarded. It should be:
```go
if ac.mod != nil {
    report := moderation.AbuseReport{...}
    if err := ac.mod.SubmitReport(report); err != nil {
        slog.Warn("failed to submit report", "error", err)
    }
}
```

### 5. Add Combat Engine Tests (MEDIUM)
Test the full combat lifecycle: start combat, perform rounds, verify damage application, stop combat, verify cleanup. Test edge cases: both players die simultaneously, one flees, one disconnects mid-combat.

### 6. Compile Regexes Once in Moderation (MEDIUM)
In `moderation/manager.go`, compile regex patterns at load time and store `*regexp.Regexp`. Currently `matches()` calls `regexp.MatchString()` which compiles on every check.

### 7. Replace Global Weather State (MEDIUM)
Move `timeInfo`, `weatherInfo`, `weatherMu` into a `WeatherSystem` struct owned by `World`. Pass `*WeatherSystem` to functions that need it.

### 8. Add Error Logging for Dropped Messages (MEDIUM)
In `session/manager.go` MessageSink, the `default:` case dropping messages should log at Warn level with player name and message size. Silent message loss is a gameplay bug.

### 9. Pre-allocate Slices in Hot Paths (LOW)
Audit `Rooms()`, `GetPlayersInRoom()`, `GetMobsInRoom()` for slice pre-allocation. Most iterate maps and append — use `make([]T, 0, len(sourceMap))`.

### 10. Remove or Wire `nolint:unused` Functions (LOW)
Either connect the unwired command handlers in `act_movement.go` and `mobprogs.go` to the registry, or delete them. Dead code with `nolint` annotations is worse than no code — it looks maintained but isn't.

---

## Appendices

### A. Notable Positives
- **Zero panics:** No `panic()` calls found in `pkg/`. This is excellent discipline.
- **Error wrapping:** 123 uses of `fmt.Errorf("...: %w", err)` — good error wrapping practice.
- **Context usage:** `context.Background()` used correctly in event publishing.
- **Structured logging:** Consistent use of `log/slog` with key-value pairs.
- **Security awareness:** `#nosec G404` annotations on game RNG, Lua sandboxing is thorough.
- **C source attribution:** Comments like `Source: fight.c:283-370` are invaluable for maintenance.

### B. Metrics
| Metric | Value |
|--------|-------|
| Source files | 270 |
| Test files | 17 |
| Mutexes/RWMutexes | 55 |
| Goroutine spawns (`go func`) | 21 |
| `interface{}` uses | 299 |
| `fmt.Errorf(%w)` uses | 123 |
| `panic()` calls | 0 |
