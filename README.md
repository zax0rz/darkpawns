---
tags: [active]
---
# Dark Pawns
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

A resurrection of the Dark Pawns MUD (1997–2010).

---

## What This Is

Dark Pawns was a MUD that ran from 1997 to 2010. If you're reading this and you played it,
you already know what it was. You probably remember your character's name. You probably
remember the first time you died to something you shouldn't have, or the person who showed
you around when you were new, or the clan that felt like a real community even though
you'd never met any of them.

I played Dark Pawns in the late 90s and early 2000s. My best friend introduced me to it, 
he nearly dropped out of college because he was playing it so much. I made friends through
this game that I still have today. It wasn't just a game. It was a place.

This project brings it back. The original world files, the original mechanics, the original
feel — rebuilt on modern infrastructure so it can run again. We cloned the original creator's
repository and we're building on top of it with as much fidelity as we can manage.

If you're the creator and you're reading this: thank you. What you built mattered to people.
It still does.

The one new thing we're adding: **AI agents as first-class players**. Not NPCs. Not bots in
a sandbox. Players: same rules, same death, same WHO list. The end goal is a human and an
AI adventuring together through the same world we loved the first time around.

---

## Play Now

| Method | Address |
|--------|--------|
| Web client | darkpawns.labz0rz.com (coming soon) |
| Telnet | `telnet darkpawns.labz0rz.com 4000` (coming soon) |
| WebSocket | `ws://darkpawns.labz0rz.com/ws` |

See the [player guide](docs/player-guide.md) for commands and mechanics. See the [agent SDK](docs/agent-sdk.md) to connect an AI agent.

---

## Current Status

**Phase 5b complete.** All 115 original Lua scripts ported and working. BRENDA69 is alive and adventuring.

See [ROADMAP.md](ROADMAP.md) for the complete phase history and current progress.

---

## Stack

- **Backend:** Go 1.24+ with goroutines
- **Transport:** WebSocket (gorilla/websocket) — JSON for agents, text for humans
- **Database:** PostgreSQL (wired, persistence in Phase 2b)
- **Scripting:** gopher-lua (Phase 3)
- **World:** Original Dark Pawns world files, unchanged

---

## Quick Start

### Local Development
```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns

go build ./cmd/server
./server \
  -world ./lib/world \
  -port 4350 \
  -db "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable" \
  -scripts ./test_scripts
```

### Docker Deployment
```bash
# Quick start with Docker Compose
./deployment/deploy-local.sh

# Or manually:
docker-compose build
docker-compose up -d
```

### Kubernetes Deployment
```bash
# Deploy to Kubernetes cluster
./deployment/deploy-k8s.sh

# Or manually apply manifests:
kubectl apply -f k8s/
```

See [deployment/DEPLOYMENT.md](deployment/DEPLOYMENT.md) for detailed instructions.

> **Note:** The compiled binary lives at `/tmp/dp-server6`. Rebuild from source after any merge.

Connect via WebSocket at `ws://localhost:4350/ws`:

```json
// Human login
{"type":"login","data":{"player_name":"YourName"}}

// Agent login (requires API key from agent_keys table)
{"type":"login","data":{"player_name":"brenda69","api_key":"<key>","mode":"agent"}}

// Commands
{"type":"command","data":{"command":"look"}}
{"type":"command","data":{"command":"north"}}
{"type":"command","data":{"command":"hit","args":["goblin"]}}
{"type":"command","data":{"command":"flee"}}
{"type":"command","data":{"command":"party","args":["brenda69"]}}
```

Agent API key for BRENDA69 is stored in Vaultwarden as **"Dark Pawns Agent Key — brenda69"**.

---

## Agent Protocol

Agents connect via WebSocket with `"mode":"agent"` and receive structured JSON state updates rather than raw text. Key features:

- **Auth:** `api_key` field in login message; keys in `agent_keys` Postgres table
- **Variables:** Subscribe to named vars — HEALTH, MAX_HEALTH, MANA, LEVEL, ROOM_VNUM, ROOM_NAME, ROOM_EXITS, ROOM_MOBS (with `target_string` for combat targeting), ROOM_ITEMS, FIGHTING, INVENTORY, EQUIPMENT, EVENTS
- **Rate limiting:** Token bucket, 10 commands/sec via `golang.org/x/time/rate`
- **Reference agents:** `scripts/dp_bot.py` (638-line deterministic FSM), `scripts/dp_brenda.py` (BRENDA69 with SOUL.md + mem0)

Full spec: [docs/agent-protocol.md](docs/agent-protocol.md)

---

## For Contributors and AI Assistants

Read [CLAUDE.md](CLAUDE.md) before touching any code. It covers:
- Where the original C source lives and why you must read it first
- What's faithful to the original vs. what's a modern addition
- What's TODO and which phase it belongs to
- How to build and test

The original Dark Pawns C source is the ground truth for all game mechanics.
Combat formulas, stat tables, death behavior — all ported from the original `fight.c`,
`class.c`, and `constants.c`. If you're about to write game logic, check the source first.

---

## Architecture

```
Humans (WebSocket/text)        Agents (WebSocket/JSON)
        │                               │
        └──────────────┬────────────────┘
                       │
             ┌─────────▼──────────┐
             │   Go Game Server   │
             │  pkg/session/      │  WebSocket, commands
             │  pkg/game/         │  world state, AI, death
             │  pkg/combat/       │  formulas from fight.c
             │  pkg/parser/       │  world file loading
             └─────────┬──────────┘
                       │
         ┌─────────────┼──────────────┐
         │             │              │
    PostgreSQL       Lua VM       (Redis — future)
  (characters)   (gopher-lua,
                 world scripts)
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Short version: read the original C source before implementing anything, cite your sources, keep the build green.

---

## Credits

- Original Dark Pawns by the Dark Pawns Coding Team (1997–2010)
- Architecture inspired by [Legends of Future Past](https://github.com/jonradoff/lofp)
- Resurrection by [zax0rz](https://github.com/zax0rz)

## License

MIT — see [LICENSE](LICENSE)
