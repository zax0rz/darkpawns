# Code Hygiene Audit — 2026-04-23

## Priority: HIGH (blocks contributor onboarding)

### 1. `pkg/telnet/listener.go` — Compilation Failure (Broken Package)
- **File:** `pkg/telnet/listener.go`
- **Lines:** 73, 132, 311, 323
- **Issue:** The telnet package references methods that don't exist on `session.Manager` and `session.Session`:
  - `manager.NewSession()` — no such method exists
  - `s.SendChannel()` — no such method; `send` field is unexported
  - `s.HandleMessage()` — method exists but is unexported (`handleMessage`)
- **Impact:** `go test ./...` fails with compile errors. This package is completely non-functional.
- **Fix:** Either implement the missing exported methods on `session.Manager`/`Session`, or remove the telnet package until it's ready.

### 2. `tests/unit/combat_test.go` — Compilation Failure (Stale Tests)
- **File:** `tests/unit/combat_test.go`
- **Lines:** 50, 92, 134, 144
- **Issue:** Tests call `combat.CalculateDamage` and `combat.CalculateHitChance` with wrong signatures (passing `int` instead of `combat.Combatant` and `combat.DiceRoll`). `CalculateCriticalChance` is undefined. `combat.Combatant` is used as a concrete type in a composite literal.
- **Impact:** Breaks `go test ./...`. These tests are from an earlier API and haven't been updated.
- **Fix:** Update test signatures to match current `combat` package API, or delete if no longer relevant.

### 3. `pkg/session/commands.go` — Giant Switch Statement for Command Dispatch
- **File:** `pkg/session/commands.go`
- **Lines:** 22–130
- **Issue:** `ExecuteCommand` uses a ~100-line `switch` for all game commands. New commands require editing this central function. The `pkg/command` package has a `RegisterCommand` pattern (`AdminCommands.RegisterCommands`), but `pkg/session/commands.go` doesn't use it — it hardcodes everything.
- **Impact:** Adding a command means modifying a file that imports half the codebase. This is exactly the 500-line `interpreter.c` problem the port was supposed to solve.
- **Fix:** Migrate all commands to the `pkg/command` registry pattern. `ExecuteCommand` should look up commands in a `map[string]CommandHandler` registered at init time.

### 4. `pkg/game/scripts.go` — Global Mutable `ScriptEngine`
- **File:** `pkg/game/scripts.go`
- **Line:** 14
- **Issue:** `var ScriptEngine interface { ... }` is a package-level mutable variable. Set once at startup, but it's global state that any package can mutate. No synchronization.
- **Impact:** Race-prone if anything ever reassigns it. Hard to test in isolation. Violates dependency injection.
- **Fix:** Make it a field on `World` or pass it as a parameter to `RunScript`/`CreateScriptContext`.

### 5. `pkg/game/ai.go` — Global Mutable `aiCombatEngine`
- **File:** `pkg/game/ai.go`
- **Lines:** 27–33
- **Issue:** `var aiCombatEngine CombatEngine` is a package-level mutable variable set via `SetAICombatEngine()`. `AITick()` reads it without synchronization.
- **Impact:** Race condition if `SetAICombatEngine` is called concurrently with `AITick`. Also makes unit testing `runMobAI` impossible without the global.
- **Fix:** Pass `CombatEngine` as a field on `World` or as a parameter to `runMobAI`.

### 6. `pkg/session/commands.go` — `json.Marshal` Errors Silently Ignored
- **File:** `pkg/session/commands.go`
- **Lines:** 175, 199, 210, 241, 251 (and 22 more across `pkg/session/`)
- **Issue:** Pattern: `msg, _ := json.Marshal(...)` — errors from `json.Marshal` are discarded with `_`.
- **Impact:** If marshaling fails (e.g., cyclic references, nil pointer in struct), the code silently sends an empty or partial message. This will be impossible to debug in production.
- **Fix:** At minimum, log the error. Better: return it so the caller can handle it.

### 7. `pkg/session/manager.go` — `Manager` Exposes Mutex Methods Publicly
- **File:** `pkg/session/manager.go`
- **Lines:** 700–720
- **Issue:** `Manager` has exported methods `Lock()`, `Unlock()`, `RLock()`, `RUnlock()`, and `Mu()` that expose the internal `sync.RWMutex`.
- **Impact:** Any external package can lock the manager's mutex, creating deadlock risks and violating encapsulation. The `commandSessionWrapper` doesn't even use these.
- **Fix:** Remove these methods. If `commandSessionWrapper` needs synchronized access, provide higher-level methods instead.

---

## Priority: MEDIUM (should fix, not blocking)

### 8. Legacy `log` Package Instead of `log/slog`
- **Files:** Across `pkg/` — 151 uses of `log.Printf`, 9 of `log.Println`
- **Issue:** The codebase uses `log` (pre-Go 1.21) instead of `log/slog` for structured logging. `pkg/scripting/engine.go` alone has ~30 `log.Printf` calls for debug tracing.
- **Impact:** No log levels (debug/info/warn/error). No structured fields. Hard to filter logs in production. Inconsistent formatting.
- **Fix:** Migrate to `log/slog` with a configured handler. Use `slog.Debug()` for script engine tracing, `slog.Info()` for normal ops, `slog.Error()` for failures.

### 9. `pkg/session/commands.go` — Direct Player Struct Field Access
- **File:** `pkg/session/commands.go`
- **Issue:** Command handlers directly mutate `s.player.Following`, `s.player.InGroup`, `s.player.RoomVNum`, etc., without going through methods.
- **Impact:** Bypasses any validation, dirty-tracking, or side-effects that setter methods might need. `Player` has mutexes that aren't always held during these mutations.
- **Fix:** Use `Player` setter methods where they exist. Add setters for fields that don't have them.

### 10. `pkg/game/world.go` — `OnPlayerEnterRoom` Spawns Goroutine Without Context
- **File:** `pkg/game/world.go`
- **Line:** 332
- **Issue:** `go func(m *MobInstance) { ... }(mob)` starts a goroutine for aggressive mob combat initiation with no `context.Context`, no timeout, no waitgroup.
- **Impact:** Goroutine leaks if `StartCombat` blocks. No way to cancel or wait for shutdown.
- **Fix:** Pass a `context.Context` from the caller, or use a worker pool instead of ad-hoc goroutines.

### 11. `pkg/combat/engine.go` — `CombatEngine` Uses Function Pointers Instead of Interfaces
- **File:** `pkg/combat/engine.go`
- **Lines:** 36–50
- **Issue:** `BroadcastFunc`, `DeathFunc`, `ScriptFightFunc`, `DamageFunc` are function fields on the struct. This works but is less testable than interfaces.
- **Impact:** Mocking requires setting function fields. No compile-time check that all required callbacks are provided.
- **Fix:** Define small interfaces (e.g., `MessageBroadcaster`, `DeathHandler`) and accept them in `NewCombatEngine` or via constructor options.

### 12. `pkg/command/admin_commands.go` — `RegisterCommand` Is a Stub
- **File:** `pkg/command/admin_commands.go`
- **Lines:** 30–40
- **Issue:** `AdminCommands.RegisterCommands()` calls `ac.manager.RegisterCommand(...)` for each admin command, but `session.Manager.RegisterCommand` is a stub that just logs "stub implementation".
- **Impact:** The admin command registry pattern exists but doesn't actually work. Commands are still hardcoded in the switch.
- **Fix:** Implement the registry in `session.Manager` and migrate commands out of the switch.

### 13. `pkg/game/world.go` — `World` Holds `*events.EventQueue` Exported
- **File:** `pkg/game/world.go`
- **Line:** 49
- **Issue:** `EventQueue` is exported (`*events.EventQueue`) but should probably be internal to `World`.
- **Impact:** External packages can directly manipulate the event queue, bypassing `World` methods.
- **Fix:** Unexport it (`eventQueue`) and expose methods like `ScheduleEvent` if needed.

### 14. `pkg/session/manager.go` — `upgrader` Is a Package-Level `var`
- **File:** `pkg/session/manager.go`
- **Line:** 26
- **Issue:** `var upgrader = websocket.Upgrader{...}` is a package-level variable with a closure over `os.Getenv`.
- **Impact:** Hard to test with different origins. Initialized at import time, not configurable.
- **Fix:** Make it a field on `Manager` or pass it as a parameter to `NewManager`.

### 15. `pkg/game/character.go` — `ClassNames` and `RaceNames` Are Package-Level Maps
- **File:** `pkg/game/character.go`
- **Lines:** 36, 52
- **Issue:** `var ClassNames = map[int]string{...}` and `var RaceNames` are mutable package-level maps.
- **Impact:** Any code can mutate them at runtime. No synchronization.
- **Fix:** Make them unexported and provide read-only accessor functions, or use `sync.Once` to initialize immutable maps.

---

## Priority: LOW (nice to have)

### 16. `pkg/game/` — Large Package with Mixed Responsibilities
- **Issue:** `pkg/game` contains: `World`, `Player`, `MobInstance`, `ObjectInstance`, `Equipment`, `Inventory`, `Spawner`, `AI`, `Death`, `Level`, `Scripts`, `Serialize`, `Party`, `Shop` (via `systems/` subpackage). That's 15+ concepts in one package.
- **Impact:** The package is large and does many things. Contributors won't know where to look.
- **Fix:** Consider splitting into `pkg/world`, `pkg/entity`, `pkg/item`, `pkg/mob`. But this is a big refactor — only do if the package keeps growing.

### 17. `pkg/common/common.go` — `Affectable` Interface Is Too Large
- **File:** `pkg/common/common.go`
- **Issue:** `Affectable` has 25+ methods. This is the "fat interface" anti-pattern.
- **Impact:** Any type implementing `Affectable` must implement everything, even if it only needs a subset.
- **Fix:** Split into smaller interfaces: `StatModifier`, `HealthTracker`, `StatusFlagHolder`, etc. Compose them where needed.

### 18. `pkg/session/manager.go` — `commandSessionWrapper` Is Redundant
- **File:** `pkg/session/manager.go`
- **Lines:** 705–740
- **Issue:** `commandSessionWrapper` wraps `*Session` to implement `common.CommandSession`, but `*Session` already has most of these methods. The wrapper adds indirection for no clear benefit.
- **Fix:** Have `*Session` implement `common.CommandSession` directly, or eliminate the wrapper.

### 19. `pkg/scripting/engine.go` — Excessive Debug Logging
- **File:** `pkg/scripting/engine.go`
- **Issue:** ~30 `log.Printf` calls for script engine internals (stack tops, Lua types, field values). These are clearly debug traces.
- **Impact:** Clutters production logs. No way to disable without code changes.
- **Fix:** Replace with `slog.Debug()` and use a debug-level logger in development.

### 20. `pkg/metrics/metrics.go` and `pkg/optimization/pool.go` — Empty `init()` Functions
- **Files:** `pkg/metrics/metrics.go:175`, `pkg/optimization/pool.go:250`
- **Issue:** `init()` functions that do nothing useful (`// Metrics are registered automatically via promauto`, `// Initialize errors package`).
- **Impact:** Slight startup cost. Confusing to readers.
- **Fix:** Remove them.

### 21. Leftover Mega-Session Files
- **Files:** `BUILD_REPORT.md`, `CI_CD_FIX_REPORT.md`, `CODE_STYLE_MODERNIZATION_REPORT.md`, `COMPILATION_SUMMARY.md`, `DOCUMENTATION_CONSOLIDATION_REPORT.md`, `EMOTIONAL_VALENCE_IMPLEMENTATION_SUMMARY.md`, `FINAL_COMPILATION_REPORT.md`, `GAME-SYSTEMS-FIXES.md`, `MODERNIZATION_CHECKLIST.md`, `MODERNIZATION_PLAN_2026.md`, `MODERNIZATION_SUMMARY.md`, `MODERNIZATION_TOOLING_RECOMMENDATIONS.md`, `ONBOARDING_SUMMARY.md`, `OPTIMIZATION_FIXES.md`, `OPTIMIZATION_SUMMARY.md`, `PERFORMANCE_ANALYSIS_REPORT.md`, `PERFORMANCE_TEST_SUMMARY.md`, `PERFORMANCE_TUNING_GUIDE.md`, `SECURITY_AUDIT_REPORT.md`, `SECURITY_FIXES_SUMMARY.md`, `SECURITY_HARDENING_GUIDE.md`, `SECURITY_HARDENING_PHASE2_REPORT.md`, `SECURITY_HARDENING_SUMMARY.md`
- **Issue:** ~20 auto-generated report files from mega-sessions clutter the repo root.
- **Impact:** New contributors will be confused about which docs matter. `CONTRIBUTING.md` and `README.md` get lost in the noise.
- **Fix:** Move all auto-generated reports to `docs/archive/` or delete them. Keep only `README.md`, `CONTRIBUTING.md`, `CLAUDE.md`, `ROADMAP.md`, `RESEARCH-LOG.md`.

### 22. `pkg/game/memory_hooks.go` — `var _ = time.Now` to Suppress Unused Import
- **File:** `pkg/game/memory_hooks.go`
- **Line:** 177
- **Issue:** `var _ = time.Now` is a hack to keep the `time` import when nothing uses it.
- **Impact:** Code smell. Indicates the file is half-implemented or the import is stale.
- **Fix:** Remove the import if unused, or use `time` properly.

### 23. `pkg/game/player.go` and `pkg/combat/formulas.go` — Duplicate `thaco` Table
- **Files:** `pkg/game/player.go:165`, `pkg/combat/formulas.go:48`
- **Issue:** Both files declare `var thaco = [12][41]int{...}` with identical data.
- **Impact:** Maintenance burden. Update one, forget the other.
- **Fix:** Define once in `pkg/combat` or `pkg/game` and reference from the other.

---

## Test Coverage Summary

| Package | Tests | Coverage | Status |
|---------|-------|----------|--------|
| `pkg/events` | Yes | N/A (covdata tool missing) | ✅ PASS |
| `pkg/engine` | Yes | N/A | ✅ PASS |
| `pkg/scripting` | Yes (integration) | N/A | ✅ PASS |
| `pkg/parser` | Yes | N/A | ✅ PASS |
| `pkg/game/systems` | Yes | N/A | ✅ PASS |
| `pkg/metrics` | Yes | N/A | ✅ PASS |
| `pkg/moderation` | Yes | N/A | ✅ PASS |
| `pkg/privacy` | Yes | N/A | ✅ PASS |
| `pkg/game` | No | N/A | ⚠️ NO TESTS |
| `pkg/session` | No | N/A | ⚠️ NO TESTS |
| `pkg/combat` | No | N/A | ⚠️ NO TESTS |
| `pkg/command` | No | N/A | ⚠️ NO TESTS |
| `pkg/db` | No | N/A | ⚠️ NO TESTS |
| `pkg/ai` | No | N/A | ⚠️ NO TESTS |
| `pkg/auth` | No | N/A | ⚠️ NO TESTS |
| `pkg/common` | No | N/A | ⚠️ NO TESTS |
| `pkg/audit` | No | N/A | ⚠️ NO TESTS |
| `pkg/validation` | No | N/A | ⚠️ NO TESTS |
| `pkg/telnet` | No | N/A | ❌ BROKEN (compile) |
| `tests/unit` | Yes | N/A | ❌ BROKEN (compile) |

**Notes:**
- `go test -cover` fails with `go: no such tool "covdata"` — the Go toolchain on this machine may be incomplete or `GOCOVERDIR` is misconfigured.
- 12 test files exist across 96 Go source files (12.5% test file ratio).
- Core game logic (`pkg/game`, `pkg/session`, `pkg/combat`, `pkg/command`) has **zero** unit tests. All existing tests are in peripheral packages (`systems`, `parser`, `events`, `engine`).
- `pkg/scripting/integration_test.go` and `pkg/events/lua_integration_test.go` appear to be integration tests, not unit tests.

---

## Package Structure Notes

### What's Good
- **No circular dependencies** between `pkg/game` ↔ `pkg/session` — `pkg/game` doesn't import `pkg/session`. The `command` package uses `SessionInterface` to break the cycle.
- **`pkg/combat/combatant.go`** defines a clean, small `Combatant` interface (16 methods, all combat-relevant). `CombatEngine` depends only on this interface, not on `game.Player` directly.
- **`pkg/common/common.go`** exists as a shared interface package, which is the standard Go pattern for breaking cycles.
- **`pkg/game/systems/`** subpackage for shop/door logic is a reasonable split.

### What's Concerning
- **`pkg/telnet`** is broken and untested. It should either be fixed or removed.
- **`pkg/optimization/`** contains `errors.go`, `pool.go`, `websocket.go`, `database.go`, `python_ai.go` — these seem like unrelated utilities mashed together. The `python_ai.go` name is suspicious for a Go codebase.
- **`pkg/agent/`** and **`pkg/ai/`** are separate packages with unclear boundaries. `pkg/agent/memory_hooks.go` and `pkg/ai/brain.go`/`behaviors.go` — are these both for NPC AI? Merge or clarify.
- **No `internal/` usage** — everything in `pkg/` is publicly importable. Packages like `pkg/audit`, `pkg/metrics`, `pkg/validation` could live in `internal/` since they're implementation details.
- **`pkg/spells/`** exists but appears to be a thin wrapper. `pkg/scripting/engine.go` has spell handling inline (lines 1129–1269). Decide if spells belong in `pkg/spells` or `pkg/scripting`.

---

## Summary

The codebase has solid bones — `Combatant` is a good interface, `World` uses mutexes properly, and the dependency graph is clean. But three things would stop a new contributor cold:

1. **It doesn't compile** (`pkg/telnet`, `tests/unit`).
2. **The command dispatch is a giant switch** — the exact anti-pattern this port was supposed to escape.
3. **There are no tests for the core game loop** (`game`, `session`, `combat`).

Fix the compile errors and the command registry, and this becomes a reasonable codebase to contribute to. Everything else is polish.
