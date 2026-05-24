---
title: "Architecture Reference"
description: "Technical architecture of the Dark Pawns Go server"
date: 2026-04-22
draft: false
section: "docs"
---

# Architecture Reference

**Last updated:** 2026-05-16

This page documents the internal architecture of the Dark Pawns Go server, covering the data flow from client connection to command dispatch, the concurrency model, and the package breakdown. The full source is in `docs/architecture/ARCHITECTURE.md` in the repository.

---

## Overview

Dark Pawns is a Go MUD server faithful to ROM 2.4b / Dark Pawns C source. It reads unmodified `.wld`, `.mob`, `.obj`, and `.zon` area files via a custom parser, then serves the game over WebSocket (primary) and telnet (legacy). Concurrency is goroutine-per-connection with a shared `sync.RWMutex` on the world state, a 2-second combat ticker, and serialized Lua scripting via a single `gopher-lua` VM. The design prioritizes behavioral fidelity to the original C codebase — same command set, same combat formulas, same Lua script API — while replacing the single-threaded C event loop with Go's concurrency primitives.

---

## Data Flow

```
  Client (Browser/Telnet/Agent)
       │           │            │
   WebSocket    TCP/Telnet    WebSocket
       │           │            │  (is_agent: true, api_key auth)
       ▼           ▼            ▼
  ┌────────┐  ┌──────────────────────────────────┐
  │/ws     │  │ pkg/telnet                       │
  │handler │  │ Listen() → handleConn()          │
  └───┬────┘  │    → manager.NewSession()        │
      │       │    → JSON shim → HandleMessage() │
      │       └──────────────┬───────────────────┘
      │                      │
      ▼                      ▼
  ┌──────────────────────────────────────┐
  │ pkg/session — Manager                │
  │  HandleWebSocket() / NewSession()    │
  │  ┌──────────────┐                    │
  │  │ Session      │                    │
  │  │ readPump()   │──┐                 │
  │  │ writePump()  │◀─┘  (goroutines)   │
  │  └──────┬───────┘                    │
  └─────────┼────────────────────────────┘
            │ handleMessage()
            ▼
  ┌──────────────────────────────────────┐
  │ Command Dispatch (pkg/session)       │
  │  ExecuteCommand()                    │
  │   1. Check mob oncmd scripts         │
  │   2. cmdRegistry.Lookup(cmd)         │
  │   3. handler(session, args)          │
  └──────────────┬───────────────────────┘
                 │
       ┌─────────┼─────────┐
       ▼         ▼         ▼
  ┌────────┐ ┌────────┐ ┌──────────────────┐
  │World   │ │Combat  │ │Scripting (Lua)   │
  │(game)  │ │Engine  │ │  RunScript()     │
  │        │ │2s tick │ │  Serialized VM   │
  └───┬────┘ └───┬────┘ └────────┬─────────┘
      │          │               │
      ▼          ▼               ▼
  ┌──────────────────────────────────────┐
  │ Response Path                        │
  │  session.send ← []byte (JSON)        │
  │  writePump() → WebSocket/Telnet      │
  └──────────────────────────────────────┘
```

---

## Package Reference

### `cmd/server`

Entry point. Parses flags (`-world`, `-port`, `-db`, `-scripts`, `-web`, `-hugo`), calls `parser.ParseWorld()`, constructs `game.World`, initializes `scripting.Engine`, connects to Postgres via `pkg/db`, creates `session.Manager`, wires callbacks (combat broadcast, death handler, memory hooks, fight scripts, damage tracking), registers HTTP routes (`/ws`, `/health`, `/metrics`, `/api/`, `/admin/`), starts zone resets, and serves HTTP (with optional TLS).

**Initialization order is strict:**
1. Parse world files
2. Create game world
3. Init scripting engine (depends on world)
4. Connect to database (optional, graceful fallback)
5. Create session manager (depends on world + db)
6. Register manager hooks
7. Setup HTTP routes
8. Start zone reset goroutine
9. Start HTTP server
10. Block on signal for shutdown

### `pkg/session`

WebSocket connection lifecycle and command dispatch. `Manager` holds all active sessions in a map keyed by player name, plus references to `game.World`, `combat.CombatEngine`, and `db.DB`. Each WebSocket upgrade spawns two goroutines (`readPump`, `writePump`) with a buffered send channel (cap 256).

Handles login (new player creation, bcrypt password verification, DB load/save), agent auth (API key validation), character creation state machine, and command routing. Agent sessions get variable subscription/dirty-tracking for push-based state sync.

**Key types:** `Manager`, `Session`

### `pkg/game`

The game world: rooms, mobs (prototypes + instances), objects, zones, players, items on the ground, AI ticker, spawner/zone resets, point update ticker (regen/hunger). Single `sync.RWMutex` (`World.mu`) protects top-level world state. `SnapshotManager` provides lock-free room snapshots via atomic pointer swaps.

`MobInstance` uses mutex-protected getters/setters. Door and shop types live in `pkg/game/systems/`.

**Key types:** `World`, `Player`, `MobInstance`, `ObjectInstance`, `Spawner`, `SnapshotManager`, `ZoneDispatcher`

### `pkg/combat`

Tick-based combat engine. Runs a 2-second ticker goroutine that snapshots all `CombatPair`s under write lock, then processes each pair (hit chance → damage → death check → fight scripts) outside the lock. Death handling delegates to game layer via `DeathFunc` callback. Damage formulas are faithful to Dark Pawns C `fight.c`.

**Key types:** `CombatEngine`, `CombatPair`, `Combatant` (interface)

### `pkg/scripting`

Lua scripting engine using `gopher-lua`. Single VM (`lua.LState`) protected by `sync.Mutex` — all script execution is serialized. Registers ~60 Lua API functions. Sandboxed: no `io`, `os.execute`, `dofile` (path-restricted), `debug`, `package`. 10MB memory limit.

Scripts live at `world/lib/scripts/` matching original C paths (e.g., `mob/144/hisc.lua`).

**Trigger types:** `greet` (player enters room), `oncmd` (before command processing — can intercept), `fight` (after each combat round), custom (via `create_event()`)

### `pkg/parser`

Reads original ROM/Dark Pawns area files from disk. `ParseWorld(libDir)` scans `wld/`, `mob/`, `obj/`, `zon/` subdirectories and returns a parsed `World` struct. Parsed data is immutable after boot.

### `pkg/telnet`

Raw TCP telnet listener. Accepts connections, negotiates IAC/WILL/DO for ECHO and SGA, then translates line-based input into JSON `ClientMessage` format and feeds it to `session.Manager`.

### `pkg/auth`

JWT generation/validation using HMAC-SHA256 with `JWT_SECRET` env var (24-hour expiry). `IPRateLimiter` for login rate limiting.

### `pkg/command`

Command registry and skill handlers. `Registry` maps command names to `Handler` functions with aliases, minimum level, and position requirements. Middleware decorator chain: `LoggingMiddleware`, `RateLimitMiddleware`.

### `pkg/db`

Postgres persistence. Player records serialized to/from JSON columns. Agent API key validation. Graceful: server runs without DB (no persistence).

### `pkg/events`

Two subsystems:
1. **EventQueue** — timer-based priority queue for scheduled Lua `create_event()` callbacks (100ms per pulse).
2. **InProcessBus** — typed in-process pub/sub bus (`MobKilledEvent`, `PlayerLeveledEvent`, `RoomEnteredEvent`, etc.).

### `pkg/storage`

Optional SQLite persistence backend (WAL mode). Players, world state, narrative memory. **Status:** Implemented but not wired into `cmd/server/main.go` yet — currently in-memory only.

---

## Concurrency Model

### Goroutine Layout

| Goroutine | Count | Purpose |
|---|---|---|
| Per-WebSocket session | 2 | `readPump` + `writePump` |
| Per-telnet session | 2 | input reader + output writer |
| Combat engine | 1 | 2-second tick |
| AI ticker | 1 | NPC behavior (game.World) |
| Point update ticker | 1 | Regen every 30s |
| Zone dispatcher | 1 per zone | Resets, AI processing |
| Lua transit cleanup | 1 | 10s ticker, orphaned item cleanup |
| Zone resets | 1 | 60s periodic check |

### Lock Hierarchy

See the [Development Guide](/docs/development/) for the complete 14-level lock acquisition hierarchy. The short version: acquire from outermost (`World.mu`) to innermost (`logWriterMu`), never in reverse.

---

## Authentication Flow

```
WebSocket connect → /ws
  │
  ▼
handleLogin()
  ├─ Agent path: is_agent=true, password=api_key → db.ValidateAgentKey()
  │
  ├─ Returning player: db.GetPlayer() → bcrypt.CompareHashAndPassword()
  │
  ├─ New player: bcrypt.GenerateFromPassword() → db.CreatePlayer()
  │     GiveStartingItems() + GiveStartingSkills()
  │
  └─ Success:
       manager.Register(name, session)
       world.AddPlayer(player)
       auth.GenerateJWT(name, isAgent, agentKeyID)
       sendWelcome(jwt_token)   ← "state" message type
       BroadcastToRoom("X has arrived.")
```

JWT tokens are 24-hour HMAC-SHA256, issued on login and sent in the initial `"state"` message. Agent sessions additionally receive a full variable dump and memory bootstrap immediately after login.

---

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
  │     If script returns TRUE → stop (handled by script)
  │
  ├─ 2. cmdRegistry.Lookup("kill")
  │     Returns "hit" entry (registered with alias "kill")
  │
  ├─ 3. entry.Handler(&commandSession{s}, ["goblin"])
  │     → cmdHit() → find target → combat engine
  │
  └─ 4. If agent session → flush dirty vars
```

---

## Snapshot System (Lock-Free Reads)

`WorldSnapshot` + `SnapshotManager` provide an atomic pointer-swapped read-only view of world rooms:

- `World` holds a `SnapshotManager` with an `atomic.Pointer[WorldSnapshot]`
- World writers mutate under `World.mu` write lock, then call `PublishSnapshot()`
- `PublishSnapshot` allocates a new snapshot, copies the rooms map, then atomically stores the pointer (no lock held on swap)
- Readers call `World.Snapshot()` which does a load-free pointer read

**Status:** Initializing and publishing on boot. Readers (GetRoom, look, movement) still use the mutex path. Transition readers to `Snapshot()` for zero-lock lookups in performance-critical paths.

---

## Middleware Pipeline

```go
type Middleware func(Handler) Handler

cmdRegistry.Use(LoggingMiddleware())
cmdRegistry.Use(RateLimitMiddleware(250 * time.Millisecond))
```

**Built-in middleware** (`pkg/command/middleware.go`):
- `LoggingMiddleware` — logs command name, duration, and error status
- `RateLimitMiddleware` — enforces minimum interval between commands per session

**Status:** Implemented; not currently wired on the registry (add noise during development).

---

## Lua Scripting

### Execution Model

1. Acquires `Engine.mu` (serialized — one script at a time)
2. Marshals context to Lua globals: `ch` (player), `me` (mob), `obj`, `room`, `argument`
3. `DoFile(scriptPath)` to load the script
4. `PCall(triggerName)` to invoke the trigger function
5. Reads back mutations from `ch` and `me` tables
6. Returns whether the script "handled" the event

### Object Transfer Protocol

`objfrom(item, "room"|"char")` removes an item to a transit map. `objto(item, "room"|"char", target)` places it. Orphaned items (not placed within 30s) are logged and discarded by the Lua transit cleanup goroutine.
