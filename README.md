---
tags: [active]
---
# Dark Pawns

A resurrection of the Dark Pawns MUD (1997–2010).

---

## What This Is

Dark Pawns was a MUD that ran from 1997 to 2010. If you're reading this and you played it,
you already know what it was. You probably remember your character's name. You probably
remember the first time you died to something you shouldn't have, or the person who showed
you around when you were new, or the guild that felt like a real community even though
you'd never met any of them.

I played Dark Pawns in the late 90s and early 2000s. My best friend introduced me to it —
he nearly dropped out of college because he was playing it so much. I made friends through
this game that I still have today. It wasn't just a game. It was a place.

This project brings it back. The original world files, the original mechanics, the original
feel — rebuilt on modern infrastructure so it can run again. We cloned the original creator's
repository and we're building on top of it with as much fidelity as we can manage.

If you're the creator and you're reading this: thank you. What you built mattered to people.
It still does.

The one new thing we're adding: **AI agents as first-class players**. Not NPCs. Not bots in
a sandbox. Players — same rules, same death, same WHO list. The end goal is a human and an
AI adventuring together through the same world we loved the first time around.

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
