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

A dark fantasy MUD, resurrected.

Dark Pawns ran from 1997 to 2010 — CircleMUD-derived, ROM 2.4b mechanics, a world that stretched across two continents and didn't particularly care if you survived them. 10,057 rooms. 1,319 mobs. 1,661 objects. 95 zones. Thirteen years of development by a team that typed room descriptions in dorm rooms at 3 AM and never stopped.

This is that game, rebuilt in Go. Same world files, same combat formulas, same everything. 73,000 lines of C became 329 Go files. The world didn't change. The walls did.

[Play now](https://darkpawns.labz0rz.com/play) · [Report a bug](https://github.com/zax0rz/darkpawns/issues)

---

## Try It

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns
go build -o server ./cmd/server
./server -world ./lib/world -port 4350
```

Connect:

```
telnet localhost 4350
```

The world files in [`lib/world/`](lib/world/) are the original ROM 2.4b area files, loaded directly. No conversion step. Every room, every mob, every poorly-spelled shopkeeper description is intact.

### Requirements

- **Go 1.26+**
- **PostgreSQL** (optional — the server runs without it, player data won't persist)

Without a database, you can connect and walk around. With one, characters save.

### Docker

```bash
docker build -t darkpawns .
docker run -p 4350:4350 -v ./lib/world:/app/lib/world darkpawns
```

See [`Dockerfile`](Dockerfile) for the multi-stage build. Additional Docker configurations:

| File | Purpose |
|------|---------|
| `Dockerfile` | Full server with Lua scripting |
| `Dockerfile.ai-agent` | AI agent sidecar (Python) |
| `Dockerfile.local` | Local development build |
| `Dockerfile.prebuilt` | Pre-built binary copy |
| `Dockerfile.privacy-filter` | PII filtering sidecar |

### Kubernetes

Full manifests in [`k8s/`](k8s/):

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/server.yaml
kubectl apply -f k8s/ai-agent.yaml
```

---

## AI Agents as Players

> **Status:** Protocol implemented. Agent SDK and public documentation in progress.

This is the part that's different.

Dark Pawns treats AI agents as first-class players. Not NPCs with scripted dialogue. Not bots in a sandbox with a separate API. Players. They log in, they move through the same rooms, they hit the same mobs, they die the same permanent death. Type WHO and your agent shows up on the list right next to every human player on the server.

An agent doesn't get a special channel or a simplified world state. It connects over WebSocket, receives structured state updates — health, mana, room contents, nearby mobs, inventory events — and issues the same commands a human would type at a telnet prompt. The game doesn't know the difference. Neither do the other players.

This opens the door to things MUDs have never had: human and AI characters adventuring together in the same party, emergent behavior from agents that have to navigate the same brutal combat system you do, and a live testbed for game AI research where "the environment" is a 25-year-old dark fantasy world with working vampirism and werewolf mechanics.

### How it works

```bash
# Generate an API key
go run ./cmd/agentkeygen -name "MyAgent" -db "$DATABASE_URL"

# Configure and connect
dp-agent config --key dp_your_key_here
dp-agent play
```

Or use the Python SDK for custom agents:

```python
from example_agent import DarkPawnsAgent

agent = DarkPawnsAgent(api_key="YOUR_KEY", player_name="brenda69")
agent.connect()
agent.explore()  # autonomous exploration with combat loop
agent.get_health()    # structured, no text parsing
agent.get_room_mobs() # [{"name": "a goblin", "target_string": "goblin"}, ...]
```

**[`dp-agent`](docs/agents/dp-agent.md)** — Go CLI with FSM combat survival, LLM decisions, session logging, and dreaming.
**[`memory-system.md`](docs/agents/memory-system.md)** — Server-hosted emotional memory, valence computation, narrative summaries.
**Protocol spec:** [`docs/architecture/agent-protocol.md`](docs/architecture/agent-protocol.md)
**SDK reference:** [`docs/architecture/agent-sdk.md`](docs/architecture/agent-sdk.md)

---

## Features

**Combat** — ROM 2.4b damage formulas, position-based multipliers, weapon types, multiple attacks per round. Bash, kick, trip, backstab, headbutt. The combat ticker and your command queue share the same mob instances without data races.

**Skills** — Active combat and utility skills (bash, kick, trip, backstab, headbutt, sneak, hide, pick lock, steal, rescue) plus skill management (learn, forget, practice). More skills ported from the original C source and wired into the combat engine.

**Social Emotes** — 187 emotes ported from the original, with full pronoun substitution. If `$n` burps at `$N`, everyone in the room knows about it.

**Lua Scripting** — A sandboxed gopher-lua engine with 199 registered Lua API functions, 112 game constants, memory limits, and path traversal protection. Mob scripts, room scripts, item scripts, and timed events. Goroutine-safe.

**Dual Transport** — WebSocket for agents (JSON protocol) and telnet for humans (text). Both hit the same command dispatch path. Connection limits enforced: 200 total, 3 per IP.

**Authentication** — bcrypt password hashing, JWT tokens with a configurable secret, per-IP rate limiting on login attempts.

**Privacy & Audit** — PII filtering is fail-closed (if the filter is down, nothing goes out). IP addresses are SHA-256 hashed in audit logs. Audit events are structured JSON with 0600 file permissions.

**World Parsing** — Original ROM 2.4b area files loaded directly, with cross-reference validation for exits, zone commands, mob and object references.

**Vampirism & Lycanthropy** — Available to players. Transform at night, bite victims, or get staked. Good luck with the moon.

**Multi-Character** — Play up to three characters simultaneously, party-based in the spirit of Ultima and Final Fantasy. Quit in your Temple to save your equipment. 100% rent-free.

---

## The World

> "Like a great game of chess, the world has become a board filled with bishops and kings, stately queens, white knights and dark pawns striving to rise through the ranks into godhood."
>
> — Friar Drake, letter to the cities of Kir Drax'in, Kir-oshi, and Alaozar

Two continents, realistic terrain, oceans between them. The civilized cities sleep behind thick walls. Outside lies the Wyldlands — untamed country where magick runs strong and wild, twisting the earth itself and giving the land its name.

**10,057 rooms** across 95 zones. Deserts that swallow caravans. Undercities lit by phosphorescent fungi. A bottomless chasm where the room description itself falls off the page, one word at a time, into the void. Someone typed that in 1996. It's still there.

### Races

| Race         | Traits                                                                                                         |
|--------------|----------------------------------------------------------------------------------------------------------------|
| **Human**    | Adaptable, widespread. The only race the eastern mercenary guilds will train as ninjas.                        |
| **Elf**      | Pale, frail, generations-long lives. Faster than they look.                                                    |
| **Dwarf**    | Stubborn as the stone they work, stocky, and built for wars that last centuries.                               |
| **Kender**   | Small, fearless, insatiably curious. Will pick up anything that isn't nailed down.                             |
| **Minotaur** | Seven feet of muscle and labyrinth instinct. Rarely lost, rarely gentle.                                       |
| **Rakshasa** | Malevolent tiger spirits in humanoid flesh. Recently some have decided to try adventuring instead of tyranny.  |
| **Ssaur**    | Evolved lizardmen, too smart for their own tribes, cast out and wandering.                                     |

### Classes

**Base classes:** Thief, Cleric, Warrior, Ninja, Psionic, Mage

**Remort-only classes** (unlocked after your first life): Assassin, Avatar, Magus, Paladin, Ranger, Mystic

Full help files, race descriptions, class details, and skill documentation live in [`lib/text/help/`](lib/text/help/) — written by the original Dark Pawns coding team across thirteen years of development.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  Client (Browser / Telnet / Agent)                                  │
│       │           │            │                                    │
│   WebSocket    TCP/Telnet    WebSocket (mode="agent")               │
│       │           │            │                                    │
│       ▼           ▼            ▼                                    │
│  ┌────────┐  ┌───────────────────────────────────┐                  │
│  │ /ws    │  │ pkg/telnet                        │                  │
│  │ handler│  │ Listen() → handleConn()           │                  │
│  └───┬────┘  │    → manager.NewSession()         │                  │
│      │       │    → JSON shim → HandleMessage()  │                  │
│      │       └──────────────┬────────────────────┘                  │
│      ▼                      ▼                                       │
│  ┌──────────────────────────────────────┐                           │
│  │ pkg/session — Manager                │                           │
│  │  HandleWebSocket() / NewSession()    │                           │
│  │  ┌──────────────┐                    │                           │
│  │  │ Session      │                    │                           │
│  │  │ readPump()   │──┐                 │                           │
│  │  │ writePump()  │◀─┘  (goroutines)   │                           │
│  │  └──────┬───────┘                    │                           │
│  └─────────┼────────────────────────────┘                           │
│            │ handleMessage()                                        │
│            ▼                                                        │
│  ┌──────────────────────────────────────┐                           │
│  │ Command Dispatch (pkg/session)       │                           │
│  │  ExecuteCommand()                    │                           │
│  │   1. Check mob oncmd scripts         │                           │
│  │   2. cmdRegistry.Lookup(cmd)         │                           │
│  │   3. handler(session, args)          │                           │
│  └──────────────┬───────────────────────┘                           │
│                 │                                                   │
│       ┌─────────┼─────────┐                                         │
│       ▼         ▼         ▼                                         │
│  ┌────────┐ ┌────────┐ ┌──────────────────┐                         │
│  │World   │ │Combat  │ │Scripting (Lua)   │                         │
│  │(game)  │ │Engine  │ │  RunScript()     │                         │
│  │        │ │2s tick │ │  Serialized VM   │                         │
│  └───┬────┘ └───┬────┘ └────────┬─────────┘                         │
│      │          │               │                                   │
│      └──────────┼───────────────┘                                   │
│                 │                                                   │
│           ┌─────┴──────┐                                            │
│      PostgreSQL    Event Bus                                        │
│    (persistence)  (pub/sub)                                         │
└─────────────────────────────────────────────────────────────────────┘
```

Go, all the way down. WebSocket and telnet listeners feed the same command dispatch. Combat runs on a ticker goroutine. Game world is mutex-protected with established lock ordering: `Manager.mu → World.mu → MobInstance.mu → CombatEngine.mu`. Lua scripts run in a sandboxed VM with goroutine-safe access. An in-process event bus (`pkg/events/`) handles decoupled subsystem communication. Persistence via PostgreSQL (optional, graceful fallback).

**Concurrency model:** Goroutine-per-connection for clients. Dedicated ticker goroutines for combat (2s) and AI ticks. Mob-level locking in AI processing. Atomic alive checks for fast pre-filtering. The system processes thousands of mobs per tick without data races.

Full architecture docs: [`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md)

---

## Repository Structure

| Repository | Contents |
|------------|----------|
| [`zax0rz/darkpawns`](https://github.com/zax0rz/darkpawns) | Game server, agent CLI, dreaming pipeline, world files, docs |
| [`zax0rz/dp-client`](https://github.com/zax0rz/dp-client) | Human terminal client — WebSocket, bubbletea TUI, JSONL logging |
| [`zax0rz/darkpawns-site`](https://github.com/zax0rz/darkpawns-site) | Website — Hugo, landing page, help files, play client |

The server and agent CLI live together (shared `pkg/` imports). The client talks WebSocket — no Go imports from the server. The website is static content with zero coupling to Go code.

## Project Status

| Component | Status |
|-----------|--------|
| **Core game loop** — movement, combat, skills | ✅ Working |
| **World loading** — all 1,190 original area files | ✅ Working |
| **WebSocket + telnet transport** | ✅ Working |
| **187 social emotes** | ✅ Working |
| **Lua scripting engine** — 68 API functions, sandboxed | ✅ Working |
| **AI agent protocol** — WebSocket, API key auth, variable subscriptions | ✅ Working |
| **dp-agent CLI** — FSM combat, LLM decisions, session logging, dreaming | ✅ Working |
| **Memory system** — server-hosted valence, narrative summaries, graph consolidation | ✅ Working |
| **Privacy & audit layer** — fail-closed PII filtering | ✅ Working |
| **Concurrency suite** — mob-level locking, atomic alive checks, O(N) AI tick | ✅ Working |
| **Combat engine** — ROM 2.4b formulas, position multipliers, multi-attack | ✅ Working |
| **Mob AI** — wander, aggro, memory, scavenging, spec procs | ✅ Working |
| **BFS pathfinding** — track, hunt, intelligent navigation | ✅ Working |
| **Help system** — 433 entries, Go-native | ✅ Working |
| **Spell system** — 103 spells, 113 constants, full affect/damage/call magic dispatch | ✅ Working |
| **Clans** — full system: create, destroy, enroll, expel, promote, demote, bank, private rooms | ✅ Working |
| **Houses** — ownership, save/load, guest management, transfers, boot initialization | ✅ Working |
| **Quests** — not in original C source; Lua scripting stubs for future implementation | ⬜ Planned |
| **AI agents as players** — protocol, CLI, memory system, docs | ✅ Working |
| **dp-client** — [human terminal client](https://github.com/zax0rz/dp-client), WebSocket, JSONL logging | ✅ Working |
| **Public server** | 🟡 Running in development |

**Port complete.** All 103 spells, combat formulas, and game systems fully ported from C to Go. World files load unmodified. Remaining work is polish: mapcode rendering, text editor pagination, and a handful of low-priority helpers with zero call sites in Go.

---

## Infrastructure

| Component | Details |
|-----------|---------|
| **Server** | `cmd/server/main.go` — single binary, flag-driven |
| **CI/CD** | GitHub Actions — test → build → Docker → deploy |
| **Container registry** | `ghcr.io/zax0rz/darkpawns` |
| **Kubernetes** | Full manifests in `k8s/` |
| **Monitoring** | Prometheus metrics in `pkg/metrics/` |
| **Privacy filter** | Separate sidecar (`Dockerfile.privacy-filter`) |

### Server Flags

```
-world    <path>    Path to world files (lib directory)  [required]
-scripts  <path>    Path to Lua scripts                  [defaults to world/lib/scripts]
-port     <port>    Server port                          [default: 4350]
-db       <url>     PostgreSQL connection string          [optional]
-web      <path>    Path to web client files              [optional]
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [`docs/agents/README.md`](docs/agents/README.md) | Agent documentation hub |
| [`docs/agents/dp-agent.md`](docs/agents/dp-agent.md) | Go agent CLI — play, session, dream, exec |
| [`docs/agents/memory-system.md`](docs/agents/memory-system.md) | Server-hosted memory: valence, narrative summaries, dreaming |
| [`docs/clients/dp-client.md`](docs/clients/dp-client.md) | Human MUD client — terminal UI, split panes, session logging |
| [`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md) | Detailed package reference and concurrency model |
| [`docs/architecture/agent-protocol.md`](docs/architecture/agent-protocol.md) | Agent WebSocket protocol specification |
| [`docs/architecture/agent-sdk.md`](docs/architecture/agent-sdk.md) | Agent SDK reference |
| [`docs/player-guide/player-guide.md`](docs/player-guide/player-guide.md) | Player commands, classes, races, mechanics |
| [`docs/brand-voice.md`](docs/brand-voice.md) | Brand voice guide — three-layer voice framework |
| [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) | How to contribute |
| [`docs/architecture/PORT_SCOPE.md`](docs/architecture/PORT_SCOPE.md) | Port completion status, function-by-function audit |
| [`lib/text/help/`](lib/text/help/) | 433 in-game help entries (the original voice) |

---

## Contributing

PRs welcome. Read [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) before submitting.

The short version: check [`src/`](src/) (the original C source, 73K lines) before implementing game logic — it's the authoritative reference for all mechanics. Cite your sources. Keep the build green.

```bash
go build ./...
go vet ./...
go test ./...
```

All three must pass before any commit.

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
