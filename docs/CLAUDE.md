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
- `class.c` — class definitions, THAC0 tables, advance_level, backstab_mult
- `constants.c` — str_app, dex_app, con_app, wis_app, and other stat tables
- `structs.h` / `utils.h` — constants, macros, alignment thresholds
- `mobact.c` — mob AI: aggression, memory, scavenger, helper, wimpy
- `act.item.c` — item commands: get, drop, wear, wield, remove
- `interpreter.c` — character creation flow (nanny), class/race validation
- `config.c` — world configuration (start rooms, etc.)
- `lib/world/` — world files (.wld, .mob, .obj, .zon)

---

## Stack

- **Language:** Go 1.24.2
- **Transport:** WebSocket (gorilla/websocket)
- **Database:** PostgreSQL (wired — save/load on connect/disconnect)
- **Scripting:** gopher-lua (Phase 3 — not yet started)
- **World files:** Original Dark Pawns `.wld`, `.mob`, `.obj`, `.zon` format, parsed in `pkg/parser/`

---

## Project Status

See [ROADMAP.md](ROADMAP.md) for complete phase history and current progress.

**Current focus:** Phase 5c (Engine Completeness) — event queue, affect system, doors, shop commands, skills.

**Latest milestone:** Phase 5b complete — all 115 original Lua scripts ported and working.

---

## Known TODOs (Deferred — Do Not Fix Now)

- **Hitroll/damroll from equipment** — returns 0 until Phase 3 equipment affect system
- **Attack-type corpse descriptions** — attack type not tracked until Phase 3 spell system
- **Practices** — wis_app bonus calculated in advance_level() but field not added to Player yet
- **Move points** — calculated in advance_level() but not tracked on Player yet
- **Player resurrection** — currently instant respawn; original required other players. Phase 3+
- **Weight limits** — CAN_CARRY_W not enforced (requires str_app carry_w lookup). Phase 3
- **Skills persistence** — Skills map not saved to DB yet. Phase 3
- **Alignment persistence** — Alignment not saved to DB yet. Phase 3

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
./server \
  -world /home/zach/.openclaw/workspace/darkpawns/lib/world \
  -port 4350 \
  -db "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable" \
  -scripts ./test_scripts
```

**Binary:** `/tmp/dp-server6` — rebuild from source after any merge.

**World files:** `/home/zach/.openclaw/workspace/darkpawns/lib/world`
> `lib/world` is gitignored. If missing: `git checkout origin/master -- lib/world/`

**DB:** `postgres://postgres:postgres@localhost/darkpawns?sslmode=disable`

**BRENDA69 agent key:** stored in Vaultwarden as **"Dark Pawns Agent Key — brenda69"**

Connect via WebSocket at `ws://localhost:4350/ws`:
```json
{"type":"login","data":{"player_name":"YourName","class":3,"race":0,"new_char":true}}
{"type":"login","data":{"player_name":"brenda69","api_key":"<key>","mode":"agent"}}
{"type":"command","data":{"command":"look"}}
{"type":"command","data":{"command":"hit","args":["goblin"]}}
{"type":"command","data":{"command":"wield","args":["sword"]}}
```

---

## Key Files Added (Phase 4–5)

| File | Purpose |
|------|---------|
| `docs/agent-protocol.md` | Full agent protocol spec |
| `RESEARCH-LOG.md` | Living research log — update when making significant design decisions |
| `SWARM-PLAN.md` | World restoration execution plan for K2.6 agent swarms |
| `scripts/dp_brenda.py` | BRENDA69 agent (SOUL.md + mem0 + minimax-m2.7) |
| `scripts/dp_bot.py` | Deterministic FSM agent (638 lines) |
| `pkg/game/party.go` | Party/group system |
| `pkg/session/commands.go` | Social commands (score/who/tell/emote/shout/where) |
| `pkg/session/agent_vars.go` | Agent variable subscription + dirty tracking |
| `pkg/session/agent.go` | Agent WebSocket handler |
| `cmd/agentkeygen/main.go` | Agent key generation CLI |

**Research log:** When making significant design decisions (new systems, architectural choices, protocol changes), add an entry to `RESEARCH-LOG.md`. This feeds the AIIDE 2027 paper.

---

## What Not To Do

- Do not invent combat formulas — they exist in `fight.c`
- Do not invent stat tables — they exist in `constants.c` and `class.c`
- Do not add "modern improvements" to game mechanics without flagging them as deviations
- Do not start the next phase while current phase items are open
- Do not commit without building (`go build ./...` must succeed)
- Do not write `if isAgent { ... }` in game logic — agents play by the same rules
