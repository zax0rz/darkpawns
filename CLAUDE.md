---
tags: [active]
---
# Dark Pawns — Project Brief for AI Assistants

Read this first. Every time. Before touching any code.

---

## What This Is

A resurrection of **Dark Pawns**, a MUD (Multi-User Dungeon) that ran from 1997–2010. We are rebuilding it in Go with a modern architecture that supports both human players and AI agents as first-class citizens.

The original source code is at: `/home/zach/.openclaw/workspace/darkpawns/`  
This repo is at: `/home/zach/.openclaw/workspace/darkpawns-phase1/`  
GitHub: https://github.com/zax0rz/darkpawns

---

## The Prime Directive

**Stay true to the original. Do not invent game logic.**

When implementing any game mechanic (combat, death, movement, items, spells, AI), you MUST:

1. Read the original C source first (`/home/zach/.openclaw/workspace/darkpawns/src/`)
2. Port the actual logic — tables, formulas, constants
3. Document what you sourced and where (file + line number in comments)
4. Flag anything that can't yet be ported faithfully with a `// TODO: phase N` comment

**If you don't know what the original does, look it up before writing code.**

Key source files:
- `fight.c` — combat, damage, death, fleeing
- `class.c` — class definitions, THAC0 tables
- `constants.c` — str_app, dex_app, and other stat tables
- `structs.h` / `utils.h` — constants, macros
- `config.c` — world configuration (start rooms, etc.)
- `lib/world/` — world files (.wld, .mob, .obj, .zon)

---

## Stack

- **Language:** Go 1.24.2
- **Transport:** WebSocket (gorilla/websocket)
- **Database:** PostgreSQL (not yet fully wired)
- **Scripting:** gopher-lua (Phase 3 — not yet started)
- **World files:** Original Dark Pawns `.wld`, `.mob`, `.obj`, `.zon` format, parsed in `pkg/parser/`

---

## Current Status

### ✅ Phase 0 — World Parser
- Parses all original world files: 10,057 rooms, 1,313 mobs, 854 objects, 95 zones
- Located in `pkg/parser/`

### ✅ Phase 1 — Minimal Engine
- WebSocket server, room state, movement, look, say
- Player login, basic commands
- Located in `pkg/session/`, `pkg/game/`

### ✅ Phase 2 — Core Systems
- Combat engine (THAC0/AC formula from `fight.c`, tables from `class.c`/`constants.c`)
- `hit` and `flee` commands
- Mob spawning from zone files (`pkg/game/spawner.go`)
- AI behaviors: aggressive mobs attack players, wandering mobs move
- Death/respawn: exp loss (EXP/3 from `fight.c`), corpse creation, respawn at room 8004
- Inventory and equipment system (partial — still uses `*parser.Obj`, not `ObjectInstance`)

### ⬜ Phase 3 — Lua Integration
- Embed gopher-lua
- Expose game API to Lua (act, do_damage, etc.)
- Port room/mob/obj scripts from `darkpawns/lib/scripts/`

### ⬜ Phase 4 — Agent Protocol
- API key authentication
- Structured JSON state endpoint
- Action API, event streaming, rate limiting

### ⬜ Phase 5 — BRENDA-Lobster Integration
- BRENDA connects as an agent player

### ⬜ Phase 6 — Polish & Release
- Web client, Telnet support, admin tools, public server

---

## Known TODOs (Flagged in Code)

These are deferred to later phases — do not "fix" them now:

- `STR/DEX/INT/WIS/Class` not yet on `Combatant` interface — formula defaults to neutral. Phase 3.
- Player inventory still `*parser.Obj`, not `ObjectInstance` — corpse doesn't transfer items yet. Phase 3.
- Respawn is instant heal + teleport — original had resurrection by other players. Phase 3+.
- `backstab_mult()` in `formulas.go` is approximated — port from `skills.c` in Phase 3.
- Player attack count in `GetAttacksPerRound` defaults to 1 — needs class info. Phase 3.

---

## Architecture Principles

1. **Agents are players, not NPCs.** Same rules, same death, same rate limits.
2. **Dual interface.** Humans get rich text. Agents get structured JSON. Same game state.
3. **Lua as the bridge.** World behavior lives in Lua scripts, not hardcoded Go.
4. **Preserve the original.** Add features, don't remove or contradict original mechanics.

---

## Running the Server

```bash
cd /home/zach/.openclaw/workspace/darkpawns-phase1
export PATH=$PATH:/usr/local/go/bin
go build ./cmd/server
./server -world ../darkpawns/lib
```

Connect via WebSocket at `ws://localhost:8080/ws`:
```json
{"type":"login","data":{"player_name":"YourName"}}
{"type":"command","data":{"command":"look"}}
{"type":"command","data":{"command":"hit","args":["goblin"]}}
```

---

## What Not To Do

- Do not invent combat formulas — they exist in `fight.c`
- Do not invent stat tables — they exist in `constants.c` and `class.c`
- Do not add "modern improvements" to game mechanics without flagging them as deviations
- Do not start Phase 3+ work while Phase 2 items are open
- Do not commit without building (`go build ./cmd/server` must succeed)
