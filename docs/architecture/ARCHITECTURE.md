# Dark Pawns — Architecture

**Last updated:** 2026-05-16

## Overview

Dark Pawns is a Go MUD server faithful to ROM 2.4b / Dark Pawns C source. It reads unmodified `.wld`, `.mob`, `.obj`, and `.zon` area files via a custom parser, then serves the game over WebSocket (primary) and telnet (legacy). Concurrency is goroutine-per-connection with a shared `sync.RWMutex` on the world state, a 2-second combat ticker, and serialized Lua scripting via a single `gopher-lua` VM. The design prioritizes behavioral fidelity to the original C codebase — same command set, same combat formulas, same Lua script API — while replacing the single-threaded C event loop with Go's concurrency primitives.

## Data Flow

```
  ┌─────────────────────────────────────────────────────────────────────┐
  │                                                                     │
  │  Client (Browser/Telnet/Agent)                                      │
  │       │           │            │                                    │
  │   WebSocket    TCP/Telnet    WebSocket                               │
  │       │           │            │  (mode="agent", api_key auth)      │
  │       ▼           ▼            ▼                                    │
  │  ┌────────┐  ┌──────────────────────────────────┐                   │
  │  │/ws     │  │ pkg/telnet                        │                   │
  │  │handler │  │ Listen() → handleConn()            │                   │
  │  └───┬────┘  │    → manager.NewSession()          │                   │
  │      │       │    → JSON shim → HandleMessage()   │                   │
  │      │       └──────────────┬─────────────────────┘                   │
  │      │                      │                                        │
  │      ▼                      ▼                                        │
  │  ┌──────────────────────────────────────┐                            │
  │  │ pkg/session — Manager                │                            │
  │  │  HandleWebSocket() / NewSession()    │                            │
  │  │  ┌──────────────┐                    │                            │
  │  │  │ Session      │                    │                            │
  │  │  │ readPump()   │──┐                 │                            │
  │  │  │ writePump()  │◀─┘  (goroutines)   │                            │
  │  │  └──────┬───────┘                    │                            │
  │  └─────────┼────────────────────────────┘                            │
  │            │ handleMessage()                                         │
  │            ▼                                                         │
  │  ┌──────────────────────────────────────┐                            │
  │  │ Command Dispatch (pkg/session)       │                            │
  │  │  ExecuteCommand()                    │                            │
  │  │   1. Check mob oncmd scripts         │                            │
  │  │   2. cmdRegistry.Lookup(cmd)         │                            │
  │  │   3. handler(session, args)          │                            │
  │  └──────────────┬───────────────────────┘                            │
  │                 │                                                     │
  │       ┌─────────┼─────────┐                                          │
  │       ▼         ▼         ▼                                          │
  │  ┌────────┐ ┌────────┐ ┌──────────────────┐                         │
  │  │World   │ │Combat  │ │Scripting (Lua)   │                         │
  │  │(game)  │ │Engine  │ │  RunScript()     │                         │
  │  │        │ │2s tick │ │  Serialized VM   │                         │
  │  └───┬────┘ └───┬────┘ └────────┬─────────┘                         │
  │      │          │               │                                    │
  │      ▼          ▼               ▼                                    │
  │  ┌──────────────────────────────────────┐                            │
  │  │ Response Path                         │                            │
  │  │  session.send ← []byte (JSON)        │                            │
  │  │  writePump() → WebSocket/Telnet      │                            │
  │  └──────────────────────────────────────┘                            │
  │                                                                      │
  └──────────────────────────────────────────────────────────────────────┘
```

## Package Reference

### `cmd/server`
Entry point. Parses flags (`-world`, `-port`, `-db`), calls `parser.ParseWorld()`, constructs `game.World`, initializes `scripting.Engine`, connects to Postgres via `pkg/db`, creates `session.Manager`, wires callbacks (combat broadcast, death handler, memory hooks, fight scripts, damage tracking), registers HTTP routes (`/ws`, `/health`, `/metrics`), starts zone resets, and serves HTTP (with optional TLS). **Key types:** none exported. **Depends on:** `game`, `parser`, `scripting`, `session`, `db`, `metrics`, `web`.

### `pkg/session`
WebSocket connection lifecycle and command dispatch. `Manager` holds all active sessions in a map keyed by player name, plus references to `game.World`, `combat.CombatEngine`, and `db.DB`. `Manager.mu` (`sync.RWMutex`) protects the sessions map — separate from world lock; sessions register/unregister independently of world state. Each WebSocket upgrade spawns two goroutines (`readPump`, `writePump`) with a buffered send channel. Handles login (new player creation, bcrypt password verify, DB load/save), agent auth (API key validation), character creation state machine, and command routing. Agent sessions get variable subscription/dirty-tracking for push-based state sync. **Key types:** `Manager`, `Session`. **Depends on:** `game`, `combat`, `db`, `auth`, `command`, `common`, `events`, `parser`, `validation`, `audit`.

### `pkg/game`
The game world: rooms, mobs (prototypes + instances), objects, zones, players, items on the ground, AI ticker, spawner/zone resets, point update ticker (regen/hunger). Single `sync.RWMutex` (`World.mu`) protects top-level world state. `SnapshotManager` provides lock-free room snapshots via atomic pointer swaps. `ZoneDispatcher` runs per-zone goroutines for reset processing. `MobInstance` uses mutex-protected getters/setters (18 methods: Get/SetTarget, Get/SetAffectFlags, Get/SetHuntingID, etc.) — direct field access only permitted under existing locks (save.go, deferred_fight_fns.go). Door and shop types (`Door`, `Shop`, `DoorManager`, `ShopManager`) live in `pkg/game/systems/`. **Key types:** `World`, `Player`, `MobInstance`, `ObjectInstance`, `Spawner`, `SnapshotManager`, `ZoneDispatcher`, `WorldScriptableAdapter`. **Depends on:** `parser`, `combat`, `events`, `scripting`, `common`.

### `pkg/game/systems/`
Door and shop subsystems. Contains `Door`, `DoorManager`, `Shop`, and `ShopManager`. Separated from core world state to keep pkg/game focused on entity management. **Depends on:** `game` (interfaces), `common`.

### `pkg/combat`
Tick-based combat engine. Runs a 2-second ticker goroutine that snapshots all `CombatPair`s under write lock (`CombatEngine.mu`), then processes each pair (hit chance → damage → death check → fight scripts) outside the lock. Death handling delegates to game layer via `DeathFunc` callback. `Combatant` interface abstracts players and mobs. Damage formulas faithful to Dark Pawns C (`fight.c`). `GetAttacksPerRound()`, `CalculateHitChance()`, `CalculateDamage()`, position constants (`PosStanding`, `PosFighting`, etc.) used by both the engine and `pkg/game/skills.go`. **Key types:** `CombatEngine`, `CombatPair`, `Combatant` (interface). **Depends on:** none (core package).

### `pkg/game/skills.go`
Skill implementations: backstab, bash, kick, trip, headbutt, rescue, sneak, hide, steal, pick lock. Each `Do*()` function returns a `SkillResult` with damage, messages (to char/victim/room), and side effects (stun, knockdown, self-stumble). Class/level/position requirements in `SkillClassReq` and `SkillPosReq` maps. Pronoun substitution (`$n`, `$N`, `$e`, etc.) faithful to ROM `act()`. **Key types:** `SkillResult`, `Pronouns`. **Depends on:** `combat`.

### `pkg/scripting`
Lua scripting engine using `gopher-lua`. Single VM (`lua.LState`) protected by `sync.Mutex` — all script execution is serialized. Registers ~60 Lua API functions matching the original C `cmdlib` array (`act`, `do_damage`, `say`, `spell`, `oload`, `mload`, `objfrom`, `objto`, `create_event`, etc.). Sandboxed: no `io`, `os.execute`, `dofile` (overridden to script-safe version), `debug`, `package`. 10MB memory limit. Scripts loaded from `world/lib/scripts/` matching original C paths (e.g., `mob/144/hisc.lua`). `RunScript()` marshals Go structs to Lua tables, executes the trigger function, reads back mutations (hp, gold), and returns whether the script "handled" the event. **Key types:** `Engine`, `ScriptContext`, `ScriptableWorld`, `ScriptablePlayer`, `ScriptableMob`, `ScriptableObject` (all interfaces). **Depends on:** `combat`.

### `pkg/events`
Two subsystems: (1) `EventQueue` — a timer-based priority queue (min-heap) for scheduled game events (Lua `create_event`, mob timers). Pulse-based timing (100ms per pulse, matching original's `OPT_USEC`). Events can re-schedule themselves by returning a positive delay. (2) `InProcessBus` — a typed in-process pub/sub event bus for decoupled subsystem communication (`MobKilledEvent`, `PlayerLeveledEvent`, `RoomEnteredEvent`, etc.). Not Redis-backed; all events are delivered synchronously within the process. **Key types:** `EventQueue`, `Bus`, `InProcessBus`, `BusEvent`. **Depends on:** none.

### `pkg/parser`
Reads original ROM/Dark Pawns area files from disk. `ParseWorld(libDir)` scans `wld/`, `mob/`, `obj/`, `zon/` subdirectories and returns a `World` struct containing all parsed rooms, mobs, objects, and zones. Parsed data is immutable after boot — runtime state (mob instances, item instances) lives in `pkg/game`. **Key types:** `World`, `Room`, `Mob`, `Obj`, `Zone`, `Exit`. **Depends on:** none.

### `pkg/telnet`
Raw TCP telnet listener. Accepts connections (with per-IP and total connection limits), negotiates IAC/WILL/DO for ECHO and SGA, then translates line-based input into JSON `ClientMessage` format and feeds it to `session.Manager.NewSession()` + `HandleMessage()`. Output writer goroutine formats `ServerMessage` structs as plain text. **Key types:** none exported. **Depends on:** `session`, `validation`.

### `pkg/auth`
Authentication utilities. `GenerateJWT()` / `ValidateJWT()` using HMAC-SHA256 with `JWT_SECRET` env var (24h expiry). `IPRateLimiter` for login rate limiting. `GetIPFromRequest()` for IP extraction. **Key types:** `Claims`, `IPRateLimiter`. **Depends on:** none.

### `pkg/command`
Command registry and skill command handlers. `Registry` maps command names to `Handler` functions with aliases, minimum level, and position requirements. Skill commands (`CmdBackstab`, `CmdBash`, etc.) use `SessionInterface` to interact with session state. **Key types:** `Registry`, `Handler`, `SessionInterface`. **Depends on:** `common`, `combat`.

### `pkg/common`
Shared interfaces to break circular dependencies. `CommandSession` abstracts session for command handlers. `CommandManager` abstracts the session manager. `ShopManager` interface for shop operations. **Key types:** `CommandSession`, `CommandManager`, `ShopManager`. **Depends on:** none.

### `pkg/db`
Postgres persistence. Player records (stats, inventory, equipment) serialized to/from JSON columns. Agent API key validation. `New()` connects, `SavePlayer`/`GetPlayer`/`CreatePlayer` for CRUD. Graceful: server runs without DB (no persistence). **Key types:** `DB`, `PlayerRecord`. **Depends on:** `game`.

### `pkg/storage`
Optional SQLite persistence backend via mattn/go-sqlite3 with WAL mode. Players table (JSON player state with timestamps), world state (zone reset tracking, mob respawn timers), narrative memory (agent long-term memory, future use). **Status:** Backend implemented and tested. Not wired into `cmd/server/main.go` yet — currently in-memory only. Wire-in: `NewSQLiteBackend(path)` → pass through World constructor. **Depends on:** none.

### Other packages
- **`pkg/ai`** — AI agent combat integration, AI ticker for NPC behavior
- **`pkg/agent`** — Agent session variable protocol (subscribe/dirty/flush)
- **`pkg/audit`** — Security event logging
- **`pkg/metrics`** — Prometheus metrics endpoint
- **`pkg/moderation`** — Content moderation hooks
- **`pkg/optimization`** — Performance optimization utilities
- **`pkg/privacy`** — Privacy/data handling utilities
- **`pkg/secrets`** — Secret management
- **`pkg/spells`** — Spell system (in progress)
- **`pkg/validation`** — Input validation (player names, etc.)
- **`web/`** — Security headers middleware

## Concurrency Model

### Goroutine layout
- **Main goroutine:** HTTP server, signal handling
- **Per-WebSocket session:** 2 goroutines (`readPump`, `writePump`)
- **Per-telnet session:** 2 goroutines (input reader, output writer)
- **Combat engine:** 1 goroutine (2-second ticker)
- **AI ticker:** 1 goroutine (game.World)
- **Point update ticker:** 1 goroutine (regen every 30s)
- **Zone dispatcher:** 1 goroutine per zone (resets, AI processing)
- **Lua transit cleanup:** 1 goroutine (10s ticker, orphans items not placed by `objto` within 30s) — added 2026-05-08
- **Zone resets:** 1 goroutine (60s periodic check)

### Locking

**`pkg/game` lock hierarchy** (from `pkg/game/locks.go`, audited 2026-05-07):

```
  1. World.mu           — top-level game state (rooms, mobs, objs)
  2. World.gossipMu     — gossip channel history
  3. World.weatherMu    — weather state
  4. World.mailWriteMu  — mail persistence
  5. Clan.mu            — clan membership, ranks
  6. Player.mu          — player stats, gold, exp, position
  7. Equipment.mu       — equipped item slots
  8. Inventory.mu       — carried item list
  9. MobInstance.mu     — mob state, HP, position  [added 2026-05-08]
 10. Spawner.mu         — zone reset scheduling
 11. BoardState.mu      — bulletin board messages
 12. Shop.mu            — shop inventory, pricing
 13. ZoneDispatcher.mu  — zone command routing
 14. logWriterMu        — log file writes (independent)
```

Acquire locks from top to bottom only. Never hold a lower-numbered lock while acquiring a higher-numbered one.

**Same-level locks** (e.g., multiple `Player.mu` on different players): always acquire in consistent order (by Name/ID) to prevent ABBA deadlocks.

**Outside pkg/game hierarchy:**
- **`Manager.mu` (`sync.RWMutex`, pkg/session):** Protects sessions map. Separate from world lock — sessions register/unregister independently of world state. Follows outermost-first principle.
- **`CombatEngine.mu` (`sync.RWMutex`, pkg/combat):** Protects combatPairs map. Combat round snapshots pairs under write lock, processes outside lock. Follows outermost-first principle.
- **`scripting.Engine.mu` (`sync.Mutex`):** Serializes all Lua VM access. Single VM, single thread of execution for scripts. No concurrent script runs. Lua scripts hold no Go locks (VM mutex released between script invocations).

**Verified safe nested patterns** (from locks.go audit):
- `World.mu → MobInstance.mu` (save.go — deserialization)
- `Player.mu → Equipment.mu` (death.go — death cleanup)
- `Clan.mu → Player.mu` (item_transfer.go — gold transfer)
- `World.mu.RLock → Player/Mob.mu` (party.go — group handling)

**`Session.send` (buffered channel, cap 256):** Lock-free message passing between session goroutines and game code.

Never upgrade `RLock → Lock` without releasing first. `World.mu` is always outermost — never call World methods that acquire `World.mu` while holding `Player.mu`, `MobInstance.mu`, or `Clan.mu`.

## Command Dispatch

```
Input: "kill goblin"
  │
  ▼
Session.handleCommand()
  │  Rate limit check (10 cmd/s token bucket)
  │
  ▼
ExecuteCommand(session, "kill", ["goblin"])
  │
  ├─ 1. Check mobs in room for oncmd scripts
  │     mob.HasScript("oncmd") → mob.RunScript("oncmd", ctx)
  │     If script returns TRUE → stop (command handled by script)
  │
  ├─ 2. cmdRegistry.Lookup("kill")
  │     Returns registered entry (command "hit" registered with alias "kill")
  │
  ├─ 3. entry.Handler(&commandSession{s}, ["goblin"])
  │     → cmdHit() → find target → combat engine
  │
  └─ 4. If agent session → flush dirty vars
```

Commands are registered in `init()` via `cmdRegistry.Register(name, handler, help, minLevel, minPos, aliases...)`. The registry supports aliases (e.g., "hit" also matches "attack", "kill"; "look" matches "l"). Social emotes are checked if no command matches. Wizard commands require `LVL_IMMORT` (31) or higher.

## Authentication Flow

```
WebSocket connect → /ws
  │
  ▼
handleLogin()
  ├─ Agent path: mode="agent", api_key → db.ValidateAgentKey()
  │     Sets isAgent=true, agentKeyID
  │
  ├─ Returning player: db.GetPlayer() → bcrypt.CompareHashAndPassword()
  │     On failure → close connection, audit log
  │
  ├─ New player: bcrypt.GenerateFromPassword() → db.CreatePlayer()
  │     GiveStartingItems() + GiveStartingSkills()
  │
  └─ Success:
       manager.Register(name, session)
       world.AddPlayer(player)
       auth.GenerateJWT(name, isAgent, agentKeyID)
       sendWelcome(jwt_token)
       BroadcastToRoom("X has arrived.")
```

JWT tokens are 24-hour HMAC-SHA256, issued on login and sent in the welcome message. Agent sessions additionally get a full variable dump and memory bootstrap immediately after login.

## Event System

### EventQueue (timer-based)
Based on original C `events.c`. Used for Lua `create_event()` — scheduled callbacks that fire after N PULSE_VIOLENCE units (1 unit = 2 seconds = 20 pulses). When an event fires, it dispatches the Lua trigger on the source mob. Events can re-schedule themselves by returning a positive delay from the callback. Events are cancelled when mobs die (`CancelBySource`).

### InProcessBus (pub/sub)
Typed in-process event bus for decoupled subsystem communication. Not Redis-backed. Events are processed sequentially per publish call. Event types:
- `combat.*` — mob killed, player killed, damage dealt
- `player.*` — connected, disconnected, leveled
- `economy.*` — item bought/sold, gold earned
- `world.*` — room entered, mob spawned
- `game.*` — command executed
- `admin.*` — wizard commands

Handlers subscribe by event type string.

## Snapshot System (lock-free reads)

`WorldSnapshot` + `SnapshotManager` provide an atomic pointer-swapped read-only view of world rooms.

**How it works:**
- `World` holds a `SnapshotManager` with an `atomic.Pointer[WorldSnapshot]`
- World writers mutate the live `rooms` map under `World.mu` write lock, then call `PublishSnapshot()`
- `PublishSnapshot` allocates a new `WorldSnapshot`, copies the rooms map, then atomically stores the pointer (no lock held on swap)
- Readers call `World.Snapshot()` which does a load-free pointer read — no locks held

**Status:** Initialized and publishing on world boot. Readers (GetRoom, look, movement) still use the mutex path via `World.rooms`. Transition readers to `Snapshot()` for zero-lock lookups in performance-critical paths.

### Generation tracking
`SnapshotManager.generation` (`atomic.Uint64`) increments on each publish, enabling stale-snapshot detection for readers that hold references across yields.

## Middleware Pipeline

`command.Registry` supports a decorator-style middleware chain: each registered handler is wrapped through all registered middleware at lookup time, outermost first.

```go
type Middleware func(Handler) Handler

cmdRegistry.Use(LoggingMiddleware())
cmdRegistry.Use(RateLimitMiddleware(250 * time.Millisecond))
```

**Built-in middleware (pkg/command/middleware.go):**
- **LoggingMiddleware** — logs command name, duration, and error status at slog.Debug level
- **RateLimitMiddleware** — enforces minimum interval between commands per session

**Status:** Functions exist, `Registry.Use()` is implemented. No middleware is currently wired onto the registry. Intended for production — logging middleware adds noise during active development.

## Persistence (SQLite)

`pkg/storage/` provides an optional SQLite persistence backend via mattn/go-sqlite3 with WAL mode:
- **Players table:** serialized JSON player state with timestamps
- **World state:** zone reset tracking, mob respawn timers
- **Narrative memory:** agent long-term memory (future use)

**Status:** Backend implemented and tested. Server (`cmd/server/main.go`) does not wire it in yet — currently in-memory only. Wire-in: `NewSQLiteBackend(path)` → pass through World constructor.

## Lua Scripting

### Loading
On boot, `scripting.NewEngine(scriptsDir, worldAdapter)` creates the VM, sandboxes it, registers ~60 API functions, and loads `globals.lua` (constants: directions, spells, positions, flags). Mob scripts live at paths like `mob/144/hisc.lua`, matching the original C directory layout.

### Execution model
All Lua execution goes through `Engine.RunScript(ctx, filename, triggerName)`, which:
1. Acquires `Engine.mu` (serialized — one script at a time)
2. Marshals context to Lua globals: `ch` (player table), `me` (mob table), `obj`, `room`, `argument`
3. `DoFile(scriptPath)` to load the script
4. `PCall(triggerName)` to invoke the trigger function
5. Reads back mutations from `ch` and `me` tables (hp, gold, etc.)
6. Returns whether the script "handled" the event (returned 1/TRUE)

### Sandbox
- Removed: `io`, `os.execute/exit/remove/rename`, `package`, `debug`, `loadfile`, `load`, `loadstring`
- `dofile` overridden to restrict to scripts directory (path traversal blocked)
- 10MB memory limit per VM
- Panic recovery on each script invocation

### Script triggers
- **`greet`** — fired when a player enters a mob's room
- **`oncmd`** — fired before command processing; returns TRUE to swallow the command
- **`fight`** — fired after each combat round on mob attackers
- **Custom triggers** — scheduled via `create_event()` (e.g., "port", "jail", "bane_one")

### Object transfer protocol
`objfrom(item, "room"|"char")` removes an item and holds it in a transit map. `objto(item, "room"|"char", target)` places it. Orphaned items (not placed within 30s) are logged and discarded by the Lua transit cleanup goroutine (10s ticker).
