---
tags: [active]
---
# Dark Pawns

A resurrection of the Dark Pawns MUD (1997–2010) for the age of AI agents.

> *"The same core game, modern architecture, agent-friendly by design."*

---

## What This Is

Dark Pawns was a MUD that ran from 1997 to 2010 — 10,000+ rooms, 12 classes, 7 races,
95 zones, and a full Lua scripting system. This project rebuilds it in Go with one
key addition: **AI agents are first-class players**.

Not NPCs. Not bots in a sandbox. Players. Same rules, same death, same WHO list.
The end goal is Zach and an AI agent adventuring together through the original world.

---

## Current Status

**Phase 2 complete.** The core game loop works.

| Phase | Status | What It Covers |
|-------|--------|----------------|
| 0 — World Parser | ✅ Done | 10,057 rooms, 1,313 mobs, 854 objects, 95 zones |
| 1 — Minimal Engine | ✅ Done | Login, movement, look, say |
| 2 — Core Systems | ✅ Done | Combat, mob AI, death/respawn, inventory skeleton |
| 2b — Full Play Loop | 🔲 Next | Items get/drop/equip, character creation, persistence |
| 3 — Lua Scripting | 🔲 Planned | Port original 179 scripts via gopher-lua |
| 4 — Agent Protocol | 🔲 Planned | API keys, structured state, rate limiting |
| 5 — BRENDA Plays | 🔲 Planned | AI agent character, party play, long-term memory |
| 6 — Public Server | 🔲 Planned | Web client, Telnet, darkpawns.labz0rz.com |

See [ROADMAP.md](ROADMAP.md) for the full plan.

---

## Stack

- **Backend:** Go 1.24+ with goroutines
- **Transport:** WebSocket (gorilla/websocket) — JSON for agents, text for humans
- **Database:** PostgreSQL (wired, persistence in Phase 2b)
- **Scripting:** gopher-lua (Phase 3)
- **World:** Original Dark Pawns world files, unchanged

---

## Quick Start

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns

# You need the original world files (not included — see note below)
go build ./cmd/server
./server -world /path/to/darkpawns/lib
```

Connect via WebSocket at `ws://localhost:8080/ws`:

```json
// Login
{"type":"login","data":{"player_name":"YourName"}}

// Commands
{"type":"command","data":{"command":"look"}}
{"type":"command","data":{"command":"north"}}
{"type":"command","data":{"command":"hit","args":["goblin"]}}
{"type":"command","data":{"command":"flee"}}
```

Or use Docker (requires PostgreSQL):
```bash
docker-compose up
```

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

## Credits

- Original Dark Pawns by the Dark Pawns Coding Team (1997–2010)
- Architecture inspired by [Legends of Future Past](https://github.com/jonradoff/lofp)
- Resurrection by [zax0rz](https://github.com/zax0rz)

## License

MIT — see [LICENSE](LICENSE)
