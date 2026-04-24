```
        (_____)           (_)    (_____)
  _     /  __ \           | |    |  __ \                            _
 ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;
  =    /| |  | |/ _` | '__| |/ / |  ___/ _` \ \ /\ / / '_ \/ __|    =
.***.  /| |__| | (_| | |  |   <  | |  | (_| |\ V  V /| | | \__ \  .***.
~~~~~  /|_____/ \__,_|_|  |_|\_\ |||   \__,_| \_/\_/ |_| |_|___/  ~~~~~
                                 |||
                                 |||
                                 `.'
```

# Dark Pawns

A complete Go rewrite of the Dark Pawns MUD (1997–2010), faithful to ROM 2.4b mechanics with modern infrastructure: WebSocket + telnet transport, JWT authentication, Lua scripting, and AI agents as first-class players.

The original C source (73K lines) is preserved in `./src/` as the authoritative reference for all game mechanics — combat formulas, skill implementations, regen calculations, and affect system are ported directly from it.

## What It Is

Dark Pawns was a MUD that ran from 1997 to 2010. This project brings it back — the original world files, the original mechanics, the original feel — rebuilt in Go so it can run again.

The one new addition: **AI agents as first-class players**. Not NPCs, not bots in a sandbox. Players with the same rules, same death, same WHO list. The end goal is a human and an AI adventuring together through the same world.

## Architecture

```
Client (WebSocket/JSON or Telnet/text)
    │
    ├──► WebSocket Handler ──► Session Manager
    │                          │
    │                ┌─────────┼──────────┐
    │                │         │          │
    │          Command    Combat     Lua Script
    │         Dispatch    Engine      Engine
    │          Registry   (ticker)   (serialized)
    │                │         │          │
    │                └─────────┼──────────┘
    │                          │
    │                    Game World
    │                   (mutex-protected)
    │                          │
    │                    ┌─────┴──────┐
    │                    │            │
    │               PostgreSQL    Event Bus
    │             (persistence)  (pub/sub)
    │
    └──► Telnet Listener ──► (same path)
```

## Features

**Combat System** — faithful ROM 2.4b port with position-based damage multipliers, hitroll/damroll calculations, weapon types, multiple attacks per round, and skill-based combat (bash, kick, trip, backstab, headbutt). Mutex-protected mob instances prevent data races between the combat ticker and player command goroutines.

**Skills** — 10 active combat/utility skills (bash, kick, trip, backstab, headbutt, sneak, hide, pick lock, steal, rescue) plus skill management commands (learn, forget, practice, skills). Additional skill constants (parry, dodge, berserk) defined for future implementation and Lua scripting.

**Social Emotes** — 187 emotes ported from the original, with full pronoun substitution (`$n/$N/$m/$M/$s/$S/$e`) and target resolution. Self-only emotes and targeted emotes both supported.

**Lua Scripting** — sandboxed gopher-lua engine with 196 registered functions, memory limits, path traversal protection, and goroutine-safe access. Supports mob scripts, room scripts, item scripts, and timed events.

**Dual Transport** — WebSocket (JSON protocol for agents, text for humans) and traditional telnet. Connection limits enforced (200 total, 3 per IP).

**Authentication** — bcrypt password hashing, JWT tokens with configurable secret, per-IP rate limiting on login attempts.

**Privacy & Audit** — PII filtering (fail-closed when filter is unavailable), SHA-256 hashed IPs in audit logs, structured JSON audit events with 0600 file permissions.

**World Parsing** — loads original ROM 2.4b area files directly, with cross-reference validation (exits, zone commands, mob/obj references).

## Quick Start

```bash
# Clone and build
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns
go build ./cmd/server

# Run (requires world files)
./server -world ./lib/world -scripts ./lib/world/scripts -port 8080 -db "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable"
```

Connect via WebSocket at `ws://localhost:8080/ws`:

```json
// Login
{"type":"login","data":{"player_name":"YourName","password":"yourpass"}}

// Commands
{"type":"command","data":{"command":"look"}}
{"type":"command","data":{"command":"north"}}
{"type":"command","data":{"command":"say","args":["hello"]}}
{"type":"command","data":{"command":"hit","args":["goblin"]}}
```

Connect via telnet:
```
telnet localhost 8080
```

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-world` | (required) | Path to world files directory |
| `-scripts` | `{world}/scripts` | Path to Lua scripts |
| `-port` | `8080` | HTTP/WebSocket server port |
| `-db` | Postgres URL | Database connection string |
| `JWT_SECRET` | (required in prod) | JWT signing key |
| `ENVIRONMENT` | `development` | Set to `production` to enforce auth |

## Project Structure

```
cmd/server/        Application entry point
pkg/auth/          JWT generation and validation
pkg/audit/         Structured audit logging
pkg/combat/        Combat engine, formulas, damage calculation
pkg/command/       Skill and spell command handlers
pkg/common/        Shared types and interfaces
pkg/db/            Database layer (PostgreSQL)
pkg/engine/        Affect system, regen, game loop
pkg/events/        Event queue and typed pub/sub bus
pkg/game/          World state, players, mobs, rooms, items, socials
pkg/game/systems/  Game systems (AI, spawn, etc.)
pkg/metrics/       Prometheus metrics
pkg/moderation/    Content moderation
pkg/parser/        ROM 2.4b area file parser
pkg/privacy/       PII filtering and privacy layer
pkg/scripting/     Lua scripting engine (gopher-lua)
pkg/secrets/       Encryption key management
pkg/session/       WebSocket/telnet session management, command dispatch
pkg/spells/        Spell system
pkg/telnet/        Telnet protocol handler
pkg/validation/    Input validation and sanitization
pkg/ai/            AI decision-making utilities
pkg/agent/         Agent API key management
src/               Original C source (73K lines, reference only)
test_scripts/      Lua scripts for testing
```

## Agent Protocol

Agents connect via WebSocket with structured JSON. See [docs/agent-protocol.md](docs/agent-protocol.md) for the full spec.

```json
{"type":"login","data":{"player_name":"brenda69","api_key":"<key>","mode":"agent"}}
```

Agents receive structured state updates (health, mana, room, inventory, events) rather than raw text output.

## Contributing

Read [CLAUDE.md](CLAUDE.md) before touching code. It covers where the original C source lives, what's faithful to the original vs. what's modern, and build/test procedures.

Short version: check `./src/` before implementing game logic. Cite your sources. Keep the build green.

## Documentation

- [ROADMAP.md](ROADMAP.md) — current status and known gaps
- [ARCHITECTURE.md](ARCHITECTURE.md) — detailed package reference and concurrency model
- [docs/agent-protocol.md](docs/agent-protocol.md) — agent WebSocket protocol spec
- [docs/player-guide.md](docs/player-guide.md) — player commands and mechanics
- [docs/agent-sdk.md](docs/agent-sdk.md) — agent SDK documentation

## Credits

- Original Dark Pawns by the Dark Pawns Coding Team (1997–2010)
- Go rewrite by [zax0rz](https://github.com/zax0rz)

## License

MIT
