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

## Current Status

### ✅ Phase 0 — World Parser
- Parses all original world files: 10,057 rooms, 1,313 mobs, 1,620 objects, 95 zones
- **Note:** Object count was 854 before Phase 2c — a parser lookahead bug was silently dropping every other object. Fixed.
- Located in `pkg/parser/`

### ✅ Phase 1 — Minimal Engine
- WebSocket server, room state, movement, look, say
- Player login, basic commands
- Located in `pkg/session/`, `pkg/game/`

### ✅ Phase 2b — Full Play Loop
- Character creation: 12 classes, 7 races, stat rolling (roll_real_abils from class.c)
- Starting items given on first login (do_start from class.c)
- Full inventory and equipment system (ObjectInstance, not parser.Obj)
- get/drop/wear/wield/remove/inventory/equipment commands
- PostgreSQL persistence: save on disconnect, load on login
- StrAdd (18/xx warrior STR) persisted

### ✅ Phase 2c — Correctness Pass (QA audit against original C source)

**Combat:**
- EXP loss on death: `/37` for combat deaths (die_with_killer), `/3` for bleed-out (die) — fight.c
- Attacks-per-round: full per-class/level formula from fight.c perform_violence()
- AC damage reduction: get_minusdam() ported from fight.c
- Flee XP loss: formula from act.offensive.c do_flee()
- THAC0 now uses correct per-class table (was always warrior)
- STR/DEX stat indices wired into hit/damage (str_app, dex_app from constants.c)
- Hitroll/damroll wired (return 0 until equipment affects in Phase 3)
- INT/WIS THAC0 reduction: (stat-13)/1.5 from fight.c
- Backstab multiplier: (level*0.2)+1, 20 at LVL_IMMORT — class.c

**World:**
- advance_level() implemented — per-class HP/mana/move gains, con_app table — class.c
- Mob equipped items transferred to corpse on death (was discarded)
- Zone resets implemented — M/O/G/E/R/D commands, zone age/lifespan ticker
- Sentinel mobs now correctly attack (was blocking aggression — mobact.c fix)
- MOB_STAY_ZONE enforced — mobs don't wander across zones
- ROOM_DEATH and ROOM_NOMOB checked before mob movement
- Room flags parsed from .wld bitmask (structs.h)

**Equipment:**
- ITEM_WEAR_TAKE (bit 0) no longer maps to equip slot
- ITEM_WEAR_SHIELD maps to distinct SlotShield
- Dual slots: SlotFingerR/L, SlotNeck1/2, SlotWristR/L

**AI behaviors:**
- MOB_MEMORY: mob hunts players who attacked it (mobact.c)
- MOB_AGGR_EVIL/GOOD/NEUTRAL: alignment-based aggression (mobact.c)
- MOB_WIMPY: skips awake players (mobact.c)
- MOB_SCAVENGER: picks up highest-value room item (mobact.c)
- MOB_HELPER: joins combat to assist other fighting mobs (mobact.c)

**Characters:**
- 7 races fully implemented (Rakshasa, Ssaur added)
- Class/race restrictions: Ninja is Human-only, remort classes blocked at creation
- Starting class skills assigned (Thief/Assassin, Kender/Minotaur racial)
- Player.Alignment field added (-1000 to +1000)
- Attack-type corpse descriptions scaffolded (fire/cold/slash — Phase 3 for spell types)

### ✅ Phase 3A — Lua Engine
- gopher-lua embedded; full Dark Pawns script API exposed
- All trigger types wired: oncmd/ongive/sound/fight/greet/ondeath/bribe/onpulse
- `pkg/scripting/` — engine.go, types.go, ScriptContext
- `pkg/spells/spells.go` — SPELL_* constants from spells.h

### ✅ Phase 3B — Engine Stubs → Real Implementations
- act/say/emote/do_damage/send_to_room deliver to actual players
- Trigger dispatch wired throughout game loop

### ✅ Phase 3C — Combat AI Matrices
- fighter/magic_user/cleric/sorcery — faithful ports of originals
- `test_scripts/mob/archive/` — all four live

### ✅ Phase 3D — Engine Completion (2026-04-21)
- `isfighting()` → wired to real `MobInstance.Fighting` state
- `room` global → proper table with `.vnum` + `.char[]` (all players+mobs in room)
- `spell()` → real damage dispatch with formulas from `magic.c mag_damage()` and `mag_points()`
- All 13 wrong formulas corrected, 9 non-damage spells reclassified, 5 missing spells added
- Newbie pipeline ported: creation.lua, clerk.lua, banker.lua, cityguard.lua
- Fight trigger callback wired into combat engine (ScriptFightFunc)
- Spell-specific corpse descriptions from fight.c (fire/cold/lightning/disintegrate)
- DISINTEGRATE scatters gear to room floor, drops ash object
- Source line comments on every formula for traceability
- `test_scripts/mob/newbie/` — all newbie scripts live
- Integration test: `pkg/scripting/integration_test.go`

**Deliverable met:** Fighter bashes you. Cleric heals and teleports. Guards work. Clerk gives gear.

### ✅ Phase 4 — Agent Protocol (2026-04-21)
**Prior art:** NLE, GMCP/MSDP (BasedMUD/MTH), Aardwolf. This is GMCP-over-WebSocket.

- **Auth:** `api_key` + `mode:"agent"` in existing auth message; `agent_keys` Postgres table
- **State:** Subscription model (MSDP-inspired) — agents subscribe to named variables,
  server flushes dirty vars at end of each command dispatch. Not full state every tick.
- **Variables:** HEALTH, MAX_HEALTH, MANA, LEVEL, ROOM_VNUM, ROOM_NAME, ROOM_EXITS,
  ROOM_MOBS, ROOM_ITEMS, FIGHTING, INVENTORY, EQUIPMENT, EVENTS
- **Rate limiting:** Token bucket (golang.org/x/time/rate), capacity=10 refill=10/sec;
  combat locked to 2s engine tick
- **Deliverable met:** `scripts/dp_bot.py` (638 lines) — connects, navigates, finds mobs,
  attacks using `target_string`, circuit breaker, death handling, reconnect logic
- **Key files:** `pkg/session/agent_vars.go`, `pkg/session/agent.go`,
  `pkg/db/player.go` (agent_keys), `cmd/agentkeygen/main.go`
- **Bugs fixed:** Spawner mutex deadlock (zone resets), SpawnMob self-deadlock on world.mu

### 🔄 Phase 5 — BRENDA Plays (CURRENT, ~65% done)

**Shipped:**
- party/follow/group/gtell commands (`pkg/game/party.go`, XP sharing from fight.c:1638)
- score/who/tell/emote/shout/where commands (`pkg/session/commands.go`)
- Lua script fixes: fight trigger arg, nil ch in onpulse_pc, bane state machine, breed_killer obj
- Engine stubs implemented: isnpc, cansee, plr_flagged, tell, has_item, obj_in_room, objfrom, objto
- `scripts/dp_brenda.py` — BRENDA69 agent with SOUL.md personality, mem0 cross-session memory, minimax-m2.7 LLM, server text feedback in LLM context
- `PHASE4-AGENT-PROTOCOL.md` — 64KB production spec with narrative memory architecture
- `RESEARCH-LOG.md` — living research document for AIIDE 2027 paper
- `SWARM-PLAN.md` — world restoration execution plan (K2.6 agent swarms, 92 RESTORE scripts)

**Pending:**
- Party smoke test: Zach + BRENDA adventure together
- `agent_narrative_memory` Postgres schema (narrative memory layer)
- Server-side memory writing via callback hooks
- Memory bootstrap in auth response
- Session consolidation script + salience decay cron

### 🔄 Phase 5b — World Restoration (SWARM-PLAN.md) — IN PROGRESS
**Tier 2 Combat AI (10/10)** ✅ — fighter/magic_user/cleric/sorcery + 6 specialized scripts
**Tier 3 Economy (10/10)** ✅ — shopkeeper, merchant_inn, merchant_walk, teacher, identifier, stable, pet_store, remove_curse, recruiter, shop_give
**Tier 4 Environmental (0/10)** — next after OpenClaw 4.20 upgrade
- K2.6 agent swarms restore all 92 RESTORE scripts
- Engine stubs: 17 added for Tier 3 (item_check, load_room, follow, mount, etc.)
- Critical engine gaps: gold API, practice sessions, charm system, event system, constants
- See `SWARM-PLAN.md` for execution plan

### ⬜ Phase 6 — Polish & Public Server
- Web client (React, VT100, inventory panel)
- Telnet support (GMCP/MXP)
- Admin tools, public hosting at darkpawns.labz0rz.com

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
