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

A dark fantasy MUD server, rebuilt in Go from the original C codebase (CircleMUD 3.0 → ROM 2.4b derivative). 114,000 lines of Go across 406 source files. The original 73,000 lines of C remain in `src/` as the authoritative reference for game mechanics — the Go port is complete.

Dark Pawns ran from 1997 to 2010. 10,057 rooms. 1,319 mobs. 1,661 objects. 95 zones. This is that world, running again, same area files loaded directly, no conversion step.

[Play now](https://darkpawns.labz0rz.com/play) · [Website](https://darkpawns.labz0rz.com) · [Report a bug](https://github.com/zax0rz/darkpawns/issues)

---

## Quick Start

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns
go build -o server ./cmd/server
./server -world ./lib/world -port 4350 -telnet-port 7777 -db "$DATABASE_URL"
```

Connect with any telnet client:

```
telnet localhost 7777
```

Or point a browser at `http://localhost:4350` for the WebSocket client.

### Requirements

- **Go 1.26+**
- **PostgreSQL** — character persistence, audit logs, agent memory, moderation. The server requires a database connection.

### Docker

```bash
docker build -t darkpawns .
docker run -p 4350:4350 -p 7777:7777 \
  -v "$PWD/lib:/app/lib:ro" \
  -e DATABASE_URL="postgres://..." \
  darkpawns ./server -world /app/lib
```

Additional Docker configurations in the repository root:

| File | Purpose |
|------|---------|
| `Dockerfile` | Server with Lua scripting |
| `Dockerfile.ai-agent` | AI agent sidecar (Python) |
| `Dockerfile.local` | Local development build |
| `Dockerfile.prebuilt` | Pre-built binary copy |
| `Dockerfile.privacy-filter` | PII filtering sidecar |

---

## Features

**Combat** — ROM 2.4b damage formulas, position-based multipliers, weapon types, multiple attacks per round. Bash, kick, trip, backstab, headbutt. Combat runs on a dedicated 2-second ticker goroutine, sharing mob instances with the command dispatch path under established lock ordering.

**Skills & Spells** — 103 spells with full affect/damage/call magic dispatch. Active combat skills (bash, kick, trip, backstab, headbutt, sneak, hide, pick lock, steal, rescue) with skill management (learn, forget, practice).

**Lua Scripting** — Sandboxed gopher-lua engine. Mob scripts, room scripts, item scripts, and timed events. Standard library restrictions, dangerous functions removed, path traversal protection, per-script execution timeouts. Goroutine-safe.

**Dual Transport** — WebSocket (port 4350, JSON protocol for agents and browser clients) and telnet (port 7777, for humans who prefer a real client). Both hit the same command dispatch path. Connection limits enforced.

**AI Agents as Players** — Agents connect over WebSocket, receive structured state updates (health, mana, room contents, nearby mobs, inventory events), and issue the same commands a human would type. Type WHO and an agent shows up on the list right next to every human player. The game doesn't know the difference.

**Memory & Dreaming** — Server-hosted emotional memory with valence computation, narrative summaries, and graph consolidation. Agents dream: session data is extracted, consolidated into a knowledge graph, pruned, and summarized. A memory pipeline runs offline.

**Grapevine** — Inter-MUD communication via the Grapevine WebSocket protocol. Channels, tells, and presence shared across Grapevine-connected MUDs. Configured through environment variables, runs offline if credentials aren't set.

**Bulletin Boards** — Persistent in-game bulletin boards (ported from `boards.c`). Read, write, remove messages. 12 boards, 60 messages per board, persisted to disk.

**Moderation** — Mute, ban, word filter, spam detection. Database-backed with persisted moderation state.

**Privacy & Audit** — PII filtering is fail-closed (if the filter is down, nothing goes out). IP addresses are SHA-256 hashed in audit logs. Structured JSON audit events.

**Admin Panel** — React admin UI served at `/admin/`. JWT-protected, role-gated. Live log buffer, server health, and operational controls.

**Clans & Houses** — Full clan system (create, destroy, enroll, expel, promote, demote, bank, private rooms). Player-owned houses with save/load, guest management, and transfers.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Client (Browser / Telnet / Agent)                          │
│       │           │            │                             │
│   WebSocket    TCP/Telnet    WebSocket (mode="agent")       │
│       │           │            │                             │
│       ▼           ▼            ▼                             │
│  ┌────────┐  ┌──────────────────────────────────┐            │
│  │ /ws    │  │ pkg/telnet                        │            │
│  │ handler│  │ Listen() → handleConn()           │            │
│  └───┬────┘  │    → manager.NewSession()         │            │
│      │       │    → JSON shim → HandleMessage()  │            │
│      ▼       └──────────────┬────────────────────┘            │
│      ▼                      ▼                                │
│  ┌──────────────────────────────────────┐                     │
│  │ pkg/session — Manager                │                     │
│  │  HandleWebSocket() / NewSession()    │                     │
│  │  ┌──────────────┐                   │                     │
│  │  │ Session      │                   │                     │
│  │  │ readPump()   │──┐                │                     │
│  │  │ writePump()  │◀─┘  (goroutines) │                     │
│  │  └──────┬───────┘                   │                     │
│  └─────────┼───────────────────────────┘                     │
│            │ handleMessage()                                   │
│            ▼                                                   │
│  ┌──────────────────────────────────────┐                     │
│  │ Command Dispatch (pkg/command)       │                     │
│  │  cmdRegistry.Lookup(cmd) → handler() │                     │
│  │  Middleware: auth, rate-limit, mod    │                     │
│  └──────────────┬───────────────────────┘                     │
│       ┌─────────┼─────────┐                                   │
│       ▼         ▼         ▼                                   │
│  ┌────────┐ ┌────────┐ ┌──────────────────┐                  │
│  │World   │ │Combat  │ │Scripting (Lua)   │                  │
│  │(game)  │ │Engine  │ │  RunScript()     │                  │
│  │        │ │2s tick │ │  Serialized VM   │                  │
│  └───┬────┘ └───┬────┘ └────────┬─────────┘                  │
│      │          │               │                             │
│      └──────────┼───────────────┘                             │
│                 │                                             │
│           ┌─────┴──────┐                                      │
│      PostgreSQL    Event Bus                                  │
│    (persistence)  (pub/sub)                                   │
└─────────────────────────────────────────────────────────────┘
```

Go, all the way down. Goroutine-per-connection for clients. Dedicated ticker goroutines for combat (2s), AI ticks, weather, zone resets, and affect updates. Lock ordering: `Manager.mu → World.mu → MobInstance.mu → CombatEngine.mu`. Lua scripts run serialized through the engine mutex. An in-process event bus (`pkg/events/`) handles decoupled subsystem communication.

**Concurrency model:** ~114K lines of Go, mutex-protected world state, atomic alive checks for fast pre-filtering, O(N) AI tick processing. The combat engine processes thousands of mob instances per tick without data races.

Full architecture docs: [`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md)

---

## Server Flags

```
-world       <path>    Path to world files (lib directory)    [required]
-scripts     <path>    Path to Lua scripts                    [defaults to world/lib/scripts]
-port        <port>    HTTP/WebSocket port                    [default: 4350]
-telnet-port <port>    Telnet port (0 to disable)             [default: 7777]
-db          <url>     PostgreSQL connection string            [required]
-web         <path>    Path to web client files                [optional]
-hugo        <path>    Path to Hugo static site                [optional]
```

TLS is optional. Set `TLS_CERT_FILE`, `TLS_KEY_FILE`, and `USE_TLS=true` for production.

---

## Repository Layout

```
cmd/
  server/       Game server entrypoint
  dp-agent/     AI agent CLI — play, session, dream, exec
  agentkeygen/  API key generation for agents
  dp-goatd/     Daemon process
pkg/
  game/         Core world state, mobs, objects, rooms, combat hooks
  session/      Player sessions, command handling, wizard commands
  command/      Command registry and middleware
  combat/       Combat formulas and damage calculation
  spells/       Spell system (saving throws, damage, affect spells)
  parser/       ROM 2.4b area file parsing
  scripting/   Lua scripting engine (gopher-lua, sandboxed)
  telnet/       Telnet protocol handling
  db/           PostgreSQL persistence, player save/load
  agent/        AI agent memory hooks
  agentcli/     Agent CLI library (FSM combat, LLM decisions, session logging)
  dreaming/     Memory consolidation — graph-based, offline dreaming pipeline
  grapevine/    Inter-MUD communication (Grapevine WebSocket protocol)
  boards/       Bulletin board system
  events/       In-process pub/sub event bus
  optimization/ Object pools, caches, database connection pooling, WebSocket pools
  privacy/      Fail-closed PII filtering
  moderation/   Mute, ban, word filter, spam detection
  auth/         bcrypt passwords, JWT tokens, rate limiting
  audit/        Structured audit logging
  admin/        Admin panel API routes
  metrics/      Prometheus metrics
  secrets/       Secret management
  validation/   Input validation
  common/       Shared utilities
  storage/      SQLite storage interface
  testutil/     Test helpers
web/            HTTP middleware, security headers, API handlers
website/        Hugo static site (landing page, help, play client)
admin-ui/       React admin panel
k8s/            Kubernetes manifests
lib/            Original world files, area data, help text
src/            Original C source (reference only — do not re-port)
```

---

## Repository Family

| Repository | Contents |
|------------|----------|
| [`zax0rz/darkpawns`](https://github.com/zax0rz/darkpawns) | Game server, agent CLI, dreaming pipeline, world files, docs |
| [`zax0rz/dp-client`](https://github.com/zax0rz/dp-client) | Human terminal client — WebSocket, bubbletea TUI, JSONL logging |
| [`zax0rz/darkpawns-site`](https://github.com/zax0rz/darkpawns-site) | Website — Hugo, landing page, help files, play client |

---

## Development

```bash
# All three must pass before committing
go build ./...
go vet ./...
go test ./...

# Format
gofumpt -w .

# Lint (includes format check + golangci-lint)
make lint

# Full test suite
make test-all
```

The short version: check `src/` (the original C source) before implementing game logic — it's the authoritative reference for all mechanics. Cite your sources. Keep the build green.

See [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) for the full contribution guide.

---

## Infrastructure

| Component | Details |
|-----------|---------|
| **CI/CD** | GitHub Actions — test, lint, race detector, e2e smoke tests → Docker → K8s deploy |
| **Container registry** | `ghcr.io/zax0rz/darkpawns` |
| **Kubernetes** | Full manifests in `k8s/` — namespace, configmap, secrets, Postgres, Redis, server, AI agent |
| **Monitoring** | Prometheus metrics at `/metrics` |
| **Privacy filter** | Separate sidecar (`Dockerfile.privacy-filter`) |
| **Website** | Hugo static site, deployed via `make deploy-site` |

---

## Documentation

| Document | Description |
|----------|-------------|
| [`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md) | Package reference and concurrency model |
| [`docs/architecture/agent-protocol.md`](docs/architecture/agent-protocol.md) | Agent WebSocket protocol specification |
| [`docs/architecture/agent-sdk.md`](docs/architecture/agent-sdk.md) | Agent SDK reference |
| [`docs/agents/dp-agent.md`](docs/agents/dp-agent.md) | Go agent CLI — play, session, dream, exec |
| [`docs/agents/memory-system.md`](docs/agents/memory-system.md) | Server-hosted memory: valence, narrative summaries, dreaming |
| [`docs/brand-voice.md`](docs/brand-voice.md) | Brand voice guide — three-layer voice framework |
| [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) | How to contribute |
| [`DEPLOYMENT.md`](DEPLOYMENT.md) | Server deployment guide |
| [`lib/text/help/`](lib/text/help/) | 433 in-game help entries (the original voice) |

---

## Credits

- **Derek Karnes (Serapis)** — conceived and masterminded Dark Pawns (1997)
- **R.E. Paret (Frontline)** — post-2.0 development, open-sourced the codebase, wrote the world
- **S. Thompson (Orodreth)** — admin support and infrastructure
- **Tarrant Martin (Aralius)** — world design and implementation
- **Jeremy Elson** — CircleMUD 3.0, the foundation everything was built on
- **The Dark Pawns community** — players, builders, testers across thirteen years
- **Go rewrite** by [zax0rz](https://github.com/zax0rz)

Original C source: [rparet/darkpawns](https://github.com/rparet/darkpawns)

---

## License

MIT
