# Pass 1: Architecture & Idiomatic Go

**Reviewer:** Claude Opus 4 (code review subagent)  
**Date:** 2026-04-26  
**Codebase:** Dark Pawns MUD server (~80K lines Go)  
**Scope:** Package structure, interface design, dependency injection, thread safety, extensibility

---

## Executive Summary

Dark Pawns is a CircleMUD-to-Go port that's impressively functional for its stage. The package layout is generally logical and the event bus / snapshot manager show good Go sensibility. However, the codebase has several critical architecture issues that will cause real bugs under load — most notably a dual send-channel design where messages silently vanish, pervasive use of package-level mutable globals for dependency wiring, and a `game` package that's grown into a 38K-line god package. These are all tractable problems, but they need attention before any kind of production deployment.

---

## Package Structure Assessment

| Package | Lines | Responsibility | Assessment |
|---------|-------|---------------|------------|
| `game` | 38,298 | World state, player, mob, items, movement, combat hooks, skills, clans, houses, boards, bans, AI, socials, spec procs, death, dreams | **God package.** Should be split. |
| `session` | 12,641 | WebSocket/telnet sessions, command dispatch, all command handlers, wizard commands | Overloaded — command handlers should live in `command` |
| `spells` | 3,689 | Spell effects | Reasonable, but coupled via globals |
| `engine` | 3,020 | Affect manager, comm infra, compat wrappers | Mixed bag — contains both real logic and C-compat stubs |
| `command` | 2,791 | Registry + admin/shop/skill command handlers | Good registry design, but most commands live in `session` |
| `scripting` | 2,515 | Lua VM, sandbox, API bindings | Clean separation |
| `combat` | 2,196 | Combat engine, formulas, combatant interface | Good isolation, but uses 70+ package-level `var` hooks |
| `optimization` | 2,325 | ? | Should be reviewed for whether it belongs |
| `parser` | 1,466 | World file parser | Clean |
| `common` | 152 | Shared interfaces | Appropriate |
| `events` | 488 | Event bus + event queue | Well-designed |

---

## Findings

### CRITICAL-1: Dual Send Channel — Messages Silently Lost

**Files:** `pkg/game/player.go:148,795-800`, `pkg/session/manager.go:210,391-420,764-767`  
**Severity:** CRITICAL

`Player` has a `Send chan []byte` (buffered 256). `Session` has a separate `send chan []byte` (also buffered 256). The WebSocket `writePump()` reads from `Session.send`. The telnet `writeLoop()` reads from `Session.SendChannel()` which returns `Session.send`.

But `Session.Send(message)` calls `Player.SendMessage()`, which writes to `Player.Send` — a channel that **nothing ever reads** in the WebSocket or telnet path. Meanwhile, `Session.sendText()` and `Session.sendError()` correctly write to `Session.send`.

This means:
- Any code calling `Session.Send()` (used by `commandSessionWrapper`, the `Send()` method itself) writes to a black hole
- Any code calling `Player.SendMessage()` from the game layer (combat messages, room broadcasts via `world.SendToAll()`, mob attack messages in `mob.go:175,181`, spawn notifications in `world.go:438`) — all lost
- The buffer fills to 256 messages then silently drops everything

**Impact:** Players likely don't receive a significant portion of game messages. Combat output, mob actions, and room broadcasts probably all fail silently.

**Fix:** Remove `Player.Send` entirely. Give `Player` a reference to its `Session.send` channel (or a `MessageSink` interface), so there's one path to the client. Or start a bridging goroutine that drains `Player.Send` into `Session.send`. The single-channel approach is cleaner:

```go
// In Player:
type MessageSink interface {
    SendMessage(msg string)
}

// Player delegates to its sink (the session)
// Session.SendMessage formats as JSON and writes to Session.send
```

---

### CRITICAL-2: 70+ Package-Level Mutable `var` Function Hooks in `combat`

**File:** `pkg/combat/fight_core.go:15-82`  
**Severity:** CRITICAL

The `combat` package declares ~70 package-level `var` function pointers (`BroadcastMessage`, `SkillMessageFunc`, `GainExp`, `ExtractChar`, `MakeCorpseFunc`, `GetRace`, `HasAffect`, `RemoveAffect`, `GetSkill`, `DoFlee`, etc.) as its mechanism for calling back into the game layer. These are set at runtime by the session/game packages.

Problems:
1. **Race condition:** These are written by one goroutine and read by the combat ticker goroutine with no synchronization. Go's race detector will flag every one.
2. **Nil panic risk:** Any function called before being set will panic. No nil checks on most call sites (e.g., `ChangeAlignment` calls `GetAlignment` and `SetAlignment` without nil guards at line ~236).
3. **Untestable:** Can't run two combat engines with different configurations. Global state means tests pollute each other.
4. **Untrackable:** With 70+ hooks, it's extremely difficult to know which are set and which aren't at any given point in the boot sequence.

**Fix:** Replace with a `GameCallbacks` interface (or struct of function fields) injected into `CombatEngine` at construction time:

```go
type GameCallbacks struct {
    BroadcastMessage func(roomVNum int, msg string, exclude string)
    GainExp          func(name string, amount int)
    ExtractChar      func(name string)
    // ... group logically
}

func NewCombatEngine(callbacks GameCallbacks) *CombatEngine { ... }
```

This makes dependencies explicit, prevents nil panics via construction-time validation, and allows testing with mock callbacks.

---

### CRITICAL-3: `game` Package is a 38K-Line God Package

**File:** `pkg/game/` (entire directory)  
**Severity:** CRITICAL (for maintainability and extensibility)

The `game` package contains: World, Player, MobInstance, ObjectInstance, Inventory, Equipment, Skills, Spells integration, Movement, Communication (say/tell/shout), Combat hooks, AI, Zone resets, Spawner, Shops, Houses, Clans, Boards, Bans, Death handling, Dreams, Socials, Spec procs (3 files totaling 3.6K lines), Object save/load, Constants, and more.

At 38K lines, this is the single largest package in the codebase and it's only going to grow. Every new game mechanic (new spell types, new command categories, new item types) goes here, creating an ever-expanding blast radius.

**Fix:** Split incrementally along domain boundaries:
- `game/world` — World struct, room/mob/object management, zone resets
- `game/player` — Player struct, inventory, equipment, stats
- `game/mob` — MobInstance, AI, spec procs
- `game/social` — Socials system
- `game/clan` — Clan management (already 1.7K lines)
- `game/housing` — Houses (already 1.2K lines)
- `game/board` — Board system
- `game/ban` — Ban management

Each sub-package should interact through interfaces defined in `game` or `common`.

---

### HIGH-1: Package-Level `var` for Cross-Package Wiring Throughout

**Files:** `pkg/game/scripts.go:14` (`var ScriptEngine`), `pkg/game/ai.go:28` (`var aiCombatEngine`), `pkg/game/merge_bridge.go:28` (`var HasActiveCharacter`), `cmd/server/main.go:64` (`game.ScriptEngine = scriptEngine`), `cmd/server/main.go:78` (`game.SetAICombatEngine(...)`)  
**Severity:** HIGH

The codebase uses package-level mutable variables extensively for dependency injection. This is the Go anti-pattern that leads to test pollution, initialization order bugs, and race conditions.

Key instances:
- `game.ScriptEngine` — set in `main.go`, read in `scripts.go` and `world.go`
- `game.HasActiveCharacter` — callback set by session manager, read by game's `ValidName()`
- `aiCombatEngine` — package-level var in `game/ai.go`
- All 70+ function hooks in `combat/fight_core.go` (see CRITICAL-2)

**Fix:** Pass dependencies through constructors. `World` should accept a `ScriptEngine` interface in `NewWorld()` or `PostInit()`. The combat engine should accept callbacks at construction. `HasActiveCharacter` should be a method on an interface passed to whatever needs it.

---

### HIGH-2: Session/Command Split is Confused

**Files:** `pkg/session/commands.go` (1,664 lines), `pkg/session/wizard_cmds.go` (1,792 lines), `pkg/command/registry.go`, `pkg/command/skill_commands.go` (1,591 lines)  
**Severity:** HIGH

Command handlers are split between `pkg/session/` and `pkg/command/` with no clear principle:
- `session/commands.go` contains command registration (`init()` with 80+ commands), `ExecuteCommand()`, and most command handler implementations
- `session/wizard_cmds.go` has wizard/admin commands
- `command/registry.go` has the `Registry` type with `Register`, `Lookup`, `Execute`
- `command/skill_commands.go` and `command/admin_commands.go` have additional handlers

The `command` package defines `SessionInterface` that imports `*game.Player` directly (not an interface), defeating the purpose of the interface pattern. Meanwhile `common.CommandSession` is a separate interface with `GetPlayer() interface{}`. And `session/commands.go` has a `commandSession` wrapper to adapt between them.

**Fix:** All command handlers should live in `pkg/command/`. The `session` package should only handle WebSocket lifecycle and dispatch to the registry. `command.SessionInterface` should not import `game` types — use interfaces from `common` instead.

---

### HIGH-3: `World` Struct Does Too Much

**File:** `pkg/game/world.go:21-73`  
**Severity:** HIGH

The `World` struct owns: rooms, mobs, objects, zones, players, activeMobs, roomItems, objectInstances, AI ticker, spawner, shopManager, EventQueue, event bus, zone dispatcher, lastTellers, HouseControl, Clans, Boards, Bans, WhodDisplay, and a snapshot manager.

That's 20+ distinct concerns in a single struct with a single `sync.RWMutex`. This means:
- Any write to any part of the world blocks reads on all other parts
- Methods like `CharTransfer()` (line 595) hold the write lock for extended operations including iterating all players and all mobs
- The struct is impossible to test in isolation

**Fix:** Factor out subsystems with their own locks:
- `PlayerRegistry` — owns `players` map with its own `sync.RWMutex`
- `MobManager` — owns `activeMobs` with its own lock
- `ItemManager` — owns `roomItems` and `objectInstances`
- Leave `World` as a thin coordinator that composes these

---

### HIGH-4: `interface{}` Used to Break Import Cycles

**Files:** `pkg/game/world.go:247-270` (`ForEachPlayerInZoneInterface`, `ForEachPlayerInRoomInterface`, `ForEachMobInRoomInterface`), `pkg/game/world.go:451` (`SpawnMobWithLevelI`), `pkg/game/world.go:462` (`LookAtRoomSimple`), `pkg/game/world.go:668` (`GetAllCharsInRoom`), `pkg/game/world.go:699` (`AddFollowerQuiet`), `pkg/game/world.go:774` (`GetItemsInRoomI`)  
**Severity:** HIGH

Multiple methods accept or return `interface{}` to work around import cycles between packages. This erases all type safety and requires runtime type assertions that can panic.

Examples:
- `ForEachPlayerInRoomInterface(roomVNum int, fn func(p interface{}))` — the "Interface" suffix is a code smell indicating a design problem
- `LookAtRoomSimple(roomVNum int, sender interface{})` — does `sender.(interface{ SendMessage(string) })` assertion at runtime
- `AddFollowerQuiet(ch, leader interface{})` — type-switches on `*Player` and `*MobInstance`
- `GetAllCharsInRoom` returns `[]interface{}`

**Fix:** Define narrow interfaces in `common` or a shared types package:

```go
type MessageSender interface {
    SendMessage(string)
}

type Character interface {
    GetName() string
    GetRoom() int
    IsNPC() bool
}
```

Then methods accept these interfaces instead of `interface{}`.

---

### HIGH-5: Mutex Discipline Inconsistencies

**Files:** Various in `pkg/game/world.go`  
**Severity:** HIGH

Several patterns risk deadlocks or data races:

1. **Lock ordering violation in `SendToZone`** (line 308): `SendToZone` does NOT acquire `w.mu` but iterates `w.players`. Yet it's called from both locked and unlocked contexts.

2. **`GetMobsInRoom` called within locked context** (line 980): `OnPlayerEnterRoom` calls `GetMobsInRoom` which acquires `RLock`. If the caller already holds a write lock, this deadlocks on `sync.RWMutex`.

3. **`removeItemFromRoomLocked` pattern** (line 810): The code duplicates `RemoveItemFromRoom` as `removeItemFromRoomLocked` for when the caller already holds the lock. This works but is fragile — there's no compile-time enforcement of the "caller must hold lock" contract.

4. **Player fields accessed without lock**: `Player.RoomVNum` is read directly in `GetPlayersInRoom` (line 350) while holding `World.mu.RLock`, but `Player.RoomVNum` is written in `MovePlayer` (line 360) under `World.mu.Lock`. This is safe IF world.mu is always held — but `Player.SetRoom()` exists and can be called from other contexts.

**Fix:** 
- Adopt a consistent locking strategy: either World.mu protects all mutable state (including player positions), or each entity gets its own lock
- Add `// requires: w.mu held` comments and consider a locking analysis tool
- Use `go test -race` extensively

---

### HIGH-6: `common` Package Interfaces Are Too Wide

**File:** `pkg/common/common.go:7-73`  
**Severity:** HIGH

The `Affectable` interface has 27 methods. The `ShopManager` interface returns `interface{}` everywhere. The `CommandManager` interface exposes `Lock()/Unlock()/RLock()/RUnlock()/Mu()` — leaking synchronization details through the interface.

Go idiom: interfaces should be small and focused. The consumer defines the interface it needs, not the provider.

**Fix:**
- Split `Affectable` into `StatBlock`, `CombatStats`, `StatusEffectable` etc.
- `CommandManager` should not expose lock methods — the implementation handles synchronization internally
- `ShopManager` should return typed values, not `interface{}`

---

### MEDIUM-1: Error Handling is Inconsistent

**Files:** Throughout  
**Severity:** MEDIUM

No custom error types. All errors are `fmt.Errorf()` strings. This makes it impossible to programmatically handle different error conditions.

Examples:
- `session/manager.go:871`: Package-level sentinel errors (`ErrPlayerAlreadyOnline`, `ErrNotAuthenticated`) — good pattern, but only used in `session`
- `game/world.go`: All errors are `fmt.Errorf()` with no wrapping
- `combat/engine.go`: Returns `fmt.Errorf()` strings
- Error handling in `handleLogin` (session/manager.go:439-581): 140 lines of error handling with inconsistent patterns — some close the connection, some return errors, some return nil after sending error messages

**Fix:** Define domain-specific error types:
```go
type ErrRoomNotFound struct{ VNum int }
type ErrMobNotFound struct{ VNum int }
type ErrPlayerNotFound struct{ Name string }
```

Use `errors.Is()` / `errors.As()` patterns consistently. Wrap errors with `%w`.

---

### MEDIUM-2: `engine/comm_infra.go` Contains Dead Code

**File:** `pkg/engine/comm_infra.go`  
**Severity:** MEDIUM

This file contains C-to-Go compat wrappers that are largely no-ops or unused:
- `Nonblock()` — explicit no-op, Go handles this
- `SetupLog()` — no-op, slog handles logging
- `OpenLogfile()` — no-op, always returns true
- `TxtQ` — queue type with Put/Get/Flush but never used (session uses channels)
- `PerformAlias()` / `PerformSubst()` — port of alias system but no callers in the codebase

The `MakePrompt()` function IS used and belongs here, but the rest is dead weight.

**Fix:** Remove the no-op functions and unused types. Keep `MakePrompt()` and `PerformSubst()` if they have callers or planned use.

---

### MEDIUM-3: Snapshot Manager Only Covers Rooms

**File:** `pkg/game/snapshot*.go`  
**Severity:** MEDIUM

The `SnapshotManager` and `WorldSnapshot` pattern (atomic pointer swap for lock-free reads) is a great idea — but it only snapshots rooms. Players, mobs, items, and all other mutable state still require lock acquisition.

The `GetRoom()` method uses the snapshot path, but `GetPlayersInRoom()`, `GetMobsInRoom()`, `GetItemsInRoom()` all go through the mutex path. This creates an inconsistent API where some reads are lock-free and others aren't, with no obvious indicator which is which.

**Fix:** Either expand snapshots to cover the read-heavy paths (mob/player room lookups), or document clearly which methods are snapshot-based vs lock-based. Consider whether the snapshot complexity is worth it for rooms-only.

---

### MEDIUM-4: `init()` Used for Command Registration

**File:** `pkg/session/commands.go:30-100`  
**Severity:** MEDIUM

The `init()` function registers 80+ commands into a package-level `cmdRegistry`. This:
- Makes the registration order invisible to `main()`
- Prevents dynamic command registration (e.g., plugins)
- Runs automatically on import, which can cause test issues
- Creates an implicit dependency on `command.NewRegistry()` being safe to call at init time

**Fix:** Replace with explicit registration called from `main()` or `NewManager()`:
```go
func RegisterBuiltinCommands(r *command.Registry) {
    r.Register("north", wrapMove("north"), "Move north.", 0, 0, "n")
    // ...
}
```

---

### MEDIUM-5: Combatant Interface is Consumed But Characters Don't Implement It Cleanly

**File:** `pkg/combat/combatant.go:12-57`  
**Severity:** MEDIUM

The `Combatant` interface has 21 methods. `Player` and `MobInstance` both implement it, but some methods exist solely to satisfy the interface (like `GetStrAdd()` for exceptional strength on mobs that don't have it).

More significantly, the combat engine operates on string names (`GetFighting() string`, `SetFighting(target string)`) rather than entity references. This means every combat operation requires a name-based lookup through the world's maps, which is O(n) for mobs.

**Fix:** Consider reducing the interface or splitting into `CombatIdentity` + `CombatStats` + `CombatActions`. For the name-based lookup issue, consider giving combat pairs entity references (via `Combatant` interface pointers) rather than string names.

---

### MEDIUM-6: Two Command Session Interfaces

**Files:** `pkg/common/command_interfaces.go:4` (`CommandSession`), `pkg/command/interface.go:8` (`SessionInterface`)  
**Severity:** MEDIUM

There are two competing "session for command handlers" interfaces:
- `common.CommandSession` — returns `GetPlayer() interface{}`
- `command.SessionInterface` — returns `GetPlayer() *game.Player`

Plus `common.Session` (a third interface in `common/common.go:38`) with `SendText(string)`.

And `session.commandSession` wraps `*Session` to satisfy `common.CommandSession`, while command handlers in `session/commands.go` use `*Session` directly.

**Fix:** Consolidate to one interface. Since command handlers need typed player access, use generics or accept that `command` imports `game` (which it already does).

---

### MEDIUM-7: `main.go` Has Manual Wiring With No Lifecycle Management

**File:** `cmd/server/main.go`  
**Severity:** MEDIUM

The main function manually wires everything together with method calls like:
```go
manager.SetCombatBroadcastFunc()
manager.SetDeathFunc()
manager.RegisterMemoryHooks()
manager.SetDamageFunc()
manager.SetScriptFightFunc()
game.SetAICombatEngine(manager.GetCombatEngine())
```

There's no lifecycle management — if one of these fails or is omitted, the system runs with nil function pointers. The zone reset goroutine is started with `go func()` and errors are logged but not propagated.

**Fix:** Create an `App` struct that owns all subsystems, validates wiring at construction, and manages start/stop lifecycle:
```go
type App struct {
    world    *game.World
    sessions *session.Manager
    combat   *combat.CombatEngine
    scripts  *scripting.Engine
}

func NewApp(cfg Config) (*App, error) { /* validate all deps */ }
func (a *App) Start() error { /* start in correct order */ }
func (a *App) Stop() { /* graceful shutdown */ }
```

---

## Key Questions Answered

1. **Package separation logical?** Mostly, but `game` is a god package (38K lines) that needs splitting. `session` also does too much (command handlers + websocket lifecycle). The leaf packages (`events`, `parser`, `combat`, `scripting`) are well-scoped.

2. **Interfaces minimal and focused?** No. `Affectable` has 27 methods. `Combatant` has 21. `ScriptableWorld` has 17. `CommandManager` leaks mutex methods. Multiple competing session interfaces exist.

3. **Proper dependency injection?** No. Heavy use of package-level mutable `var` for wiring (`ScriptEngine`, `aiCombatEngine`, `HasActiveCharacter`, 70+ combat hooks). This prevents testing, creates race conditions, and makes boot order fragile.

4. **World struct too much?** Yes. It owns 20+ concerns with a single RWMutex. Should be decomposed into focused subsystems with independent locks.

5. **Session management thread-safe?** `Manager` uses `sync.RWMutex` correctly for the sessions map. But `Player` fields are accessed through mixed locking strategies, and the dual send-channel issue means messages are silently lost.

6. **Extensibility bottlenecks?** Adding new spell types requires modifying `spells/affect_spells.go` (single 2K-line file). Adding commands requires modifying `session/commands.go`'s `init()`. Adding game mechanics requires modifying the `game` god package. The command registry pattern in `command/registry.go` is actually good — it's just not used consistently.

7. **Error handling consistent?** No. Mix of sentinel errors (session), fmt.Errorf (game/combat), silent nil returns, and ignored errors.

8. **Separation of game logic, networking, persistence?** Partially. Networking is in `session`/`telnet`. Persistence is in `db`. But game logic bleeds into `session/commands.go` (1,664 lines of game command implementations), and `main.go` manually bridges everything with function pointer assignments.

---

## Priority Order

1. **CRITICAL-1** (dual send channel) — This is likely causing missing messages *right now*
2. **CRITICAL-2** (combat globals) — Race conditions under concurrent combat
3. **HIGH-1 + HIGH-5** (global vars + mutex discipline) — Systemic race condition risk
4. **HIGH-3** (World decomposition) — Prerequisite for fixing locking issues
5. **HIGH-2** (session/command split) — Makes all other refactoring harder
6. **CRITICAL-3** (game god package) — Long-term, split incrementally alongside other work
7. Everything else follows naturally from the above
