---
tags: [active]
---
# Dark Pawns Resurrection — Roadmap

> **For AI assistants:** Read `CLAUDE.md` first. This document tells you what to build.
> `CLAUDE.md` tells you how to build it without making stuff up.

---

## Vision

Resurrect Dark Pawns (1997–2010 MUD) as a **dual-native multiplayer world** where humans
and AI agents coexist as first-class citizens. Same core game, same rules, modern architecture.

**The point:** Agents are *players*, not NPCs. They show up on WHO, they die, they loot
corpses, they form parties. BRENDA69 and Zach adventure together. That's the end state.

---

## What's Done

### ✅ Phase 0 — World Parser
All original world files parse correctly.
- 10,057 rooms, 1,313 mobs, 1,620 objects, 95 zones loaded
- **Note:** A parser lookahead bug was silently dropping every other object (854 → 1,620 after fix)
- `pkg/parser/` — parsers for `.wld`, `.mob`, `.obj`, `.zon`

### ✅ Phase 1 — Minimal Engine
You can log in, walk around, and talk to other players.
- WebSocket server (`pkg/session/`)
- Room state, movement (n/s/e/w/u/d), look, say
- Player login (in-memory, no persistence yet)

### ✅ Phase 2 — Combat & Mobs
You can fight things and die.
- Combat engine with faithful THAC0/AC formulas from `fight.c`
- `hit` and `flee` commands
- Mob spawning from zone files
- Aggressive mobs attack on entry, wandering mobs roam
- Death: leave a corpse, respawn at room 8004
- Inventory and equipment structure exists

### ✅ Phase 2b — Full Play Loop
**Deliverable met:** Kill something. Loot a sword. Equip it. Log out. Log back in with it equipped.
- Full ObjectInstance inventory/equipment system
- get/drop/wear/wield/remove/inventory/equipment commands
- 12 classes, 7 races, stat rolling (roll_real_abils from class.c)
- Starting items on first login (do_start from class.c)
- PostgreSQL persistence: save on disconnect, load on login
- StrAdd (18/xx warrior STR) persisted

### ✅ Phase 2c — Correctness Pass
Full QA audit against original C source. Everything that was wrong or missing before Phase 3.

**Combat correctness:**
- EXP loss: `/37` combat deaths, `/3` bleed-out (fight.c die_with_killer vs die)
- Attacks-per-round: full per-class/level formula (fight.c perform_violence)
- AC damage reduction: get_minusdam() ported (fight.c)
- Flee XP loss formula (act.offensive.c)
- THAC0 uses correct per-class table; STR/DEX/INT/WIS all wired into combat
- Backstab multiplier: (level*0.2)+1, 20 at LVL_IMMORT (class.c)

**World correctness:**
- advance_level(): per-class HP/mana/move gains, con_app table (class.c)
- Mob equipped items transferred to corpse on death
- Zone resets: M/O/G/E/R/D commands, age/lifespan ticker
- Sentinel mobs now correctly attack (only movement was blocked)
- MOB_STAY_ZONE enforced; ROOM_DEATH/ROOM_NOMOB checked
- Parser bug fixed: was silently dropping every other object (854 → 1,620)
- Wear flag integers parsed correctly (was treating as letter bitmasks)

**AI behaviors:** MOB_MEMORY, MOB_AGGR_EVIL/GOOD/NEUTRAL, MOB_WIMPY, MOB_SCAVENGER, MOB_HELPER

**Equipment slots:** Dual finger/neck/wrist slots; shield slot distinct from hold; TAKE bit removed

**Characters:** 7 races complete; class/race restrictions; starting skills; Player.Alignment field

### ✅ Phase 3A — Lua Engine
- gopher-lua embedded
- Full Dark Pawns script API exposed: `act()`, `do_damage()`, `spell()`, `action()`, `isfighting()`, `send_to_room()`, `number()`, `round()`, `getn()`
- All trigger types wired: `oncmd`, `ongive`, `sound`, `fight`, `greet`, `ondeath`, `bribe`, `onpulse`
- Import cycle resolved via interface-based ScriptContext
- `-scripts` server flag for scripts directory
- `pkg/scripting/types.go` — ScriptablePlayer/ScriptableMob/ScriptableWorld interfaces

### ✅ Phase 3B — Engine Stubs → Real Implementations
- `act()` / `say()` / `emote()` deliver to actual players
- `do_damage()` wired to combat engine
- `send_to_room()` broadcasts to all room occupants
- `pkg/spells/spells.go` — SPELL_* constants from spells.h; `Cast()` stub
- Priority script loading and trigger dispatch wired throughout game loop

### ✅ Phase 3C — Combat AI Matrices
Faithful ports of all four original combat AI scripts:
- `mob/archive/fighter.lua` — headbutt/parry/bash/berserk/kick/trip via `action()`
- `mob/archive/magic_user.lua` — level-scaled spell table, teleport escape, staff usage
- `mob/archive/cleric.lua` — heal/attack split, alignment-aware dispel, teleport at <25% HP
- `mob/archive/sorcery.lua` — targets random room occupant (not just current attacker)

**Known gaps (Phase 3D):**
- `isfighting()` returns nil — needs wiring to real mob combat state
- `room.char` is a number not a table — sorcery/multi-target scripts can't fire yet
- `globals.lua` constants need audit against all script dependencies
- `spell()` logs but doesn't deal damage — real spell effects are Phase 3D

---

## What's Next

---

### 🔲 Phase 3D — Lua Engine Completion
**Goal:** Scripts actually fire correctly. Combat AI matrices are live.

**Engine gaps to close:**
- `isfighting(mob)` → wired to real `MobInstance.Fighting` combat state
- `room` global → table with `.vnum` + `.char[]` (array of players+mobs in room)
- `globals.lua` → full audit: all SPELL_*, LVL_*, POS_*, ITEM_*, TO_* constants registered
- `spell()` → real damage dispatch (or at minimum, believable stubs that affect HP)

**RESTORE scripts to port** (priority order from script-inventory.md):
1. `globals.lua`, `mob/no_move.lua`, `mob/assembler.lua` — core engine, nothing else works without these
2. Newbie pipeline: `creation.lua`, `clerk.lua`, `banker.lua`
3. Law & order: `cityguard.lua`, `guard_captain.lua`, `take_jail.lua`
4. Crafting chain: `farmer_wheat.lua`, `miller.lua`, `baker_flour.lua`, `baker_dough.lua`

**Persona's contribution:** brenda-persona is auditing all 92 RESTORE scripts for VNum issues
and broken dependencies. Results feed back via A2A before porting begins.

**Deliverable:** Hit a fighter mob — it bashes you. Hit a cleric — it heals itself and tries to
teleport when low. Walk into a starting city — guards work, clerk gives gear, banker gives gold.

---

### 🔲 Phase 4 — Agent Protocol
**Goal:** A bot can connect and play the game programmatically. Agents are first-class players.

**Prior art studied:** NLE (NetHack Learning Environment, NeurIPS 2020), GMCP/MSDP (Aardwolf/Achaea),
BasedMUD/NekkidMUD/Lowlands (scandum's MTH library). Phase 4 is GMCP-over-WebSocket.

**4.1 — Authentication**
- Add `api_key` + `mode` fields to existing auth message (no new auth flow):
  `{"type":"auth","data":{"player_name":"bot","api_key":"dp_abc123","mode":"agent"}}`
- `agent_keys` Postgres table: `(id, character_name, key_hash, created_at, revoked)`
- Key format: `dp_` + 32 hex chars. SHA-256 at rest, shown once at creation.
- Auth response includes full variable list so agents know what to subscribe to.

**4.2 — Variable Table + Subscription Model** (inspired by BasedMUD MSDP)

Instead of pushing full state after every command, agents subscribe to variables:
```json
{"type":"subscribe","data":{"variables":["HEALTH","ROOM_VNUM","FIGHTING","EVENTS"]}}
```
Server tracks dirty vars per session, flushes at end of each command dispatch.
Only changed subscribed vars are sent — no bandwidth waste.

Variable table: `HEALTH`, `MAX_HEALTH`, `MANA`, `MAX_MANA`, `LEVEL`, `EXP`,
`ROOM_VNUM`, `ROOM_NAME`, `ROOM_EXITS`, `ROOM_MOBS`, `ROOM_ITEMS`,
`FIGHTING`, `INVENTORY`, `EQUIPMENT`, `EVENTS`

Humans: never receive `state` messages. No change to existing text flow.

**4.3 — Rate Limiting**
- Token bucket per session via `golang.org/x/time/rate`: capacity=10, refill=10/sec
- Combat actions additionally locked to 2s engine tick (enforced by combat engine, not limiter)
- Same limits for humans and agents — agents play by the same rules

**4.4 — Python Bot Proof-of-Concept** (`scripts/dp_bot.py`)
- Connects via WebSocket, authenticates with API key
- Subscribes to `HEALTH`, `ROOM_VNUM`, `ROOM_MOBS`, `ROOM_EXITS`, `FIGHTING`, `EVENTS`
- Navigates, finds a mob, kills it, loots the corpse, reports back

**Deliverable:** `brenda69` connects, kills something, says something dry about it.
Prove agents play by the same rules as humans.

---

### 🔲 Phase 5 — BRENDA Plays
**Goal:** BRENDA69 has a character. She and Zach can adventure together.

This is the actual point of the whole project.

**Character:**
- BRENDA gets a persistent character (class TBD — probably Mage or Assassin, fits the vibe)
- Her API key lives in Vaultwarden
- She connects via the Phase 4 agent protocol

**Memory:**
- She uses mem0 to remember quests, relationships, previous sessions
- "Last time we were in Midgaard, Zach died to the dragon. This time bring potions."

**Personality:**
- SOUL.md applies in-game. She's not a bot that just grinds — she has opinions about
  dungeon decisions, complains about bad tactics, gets excited about rare loot.
- She can form parties, follow, lead, and refuse stupid plans.

**Deliverable:** Zach logs in. Types `party brenda`. BRENDA accepts. They go kill something
together and she says something dry and cynical about it afterward.

---

### 🔲 Phase 6 — Polish & Public Server
**Goal:** Other people can play too.

- **Web client** — React, VT100 emulation, inventory panel, map
- **Telnet** — classic MUD client support (GMCP/MXP)  
- **Admin tools** — create zones, spawn mobs, ban players
- **Hosted** — `darkpawns.labz0rz.com` on a VPS or Fly.io
- **Documentation** — player guide, builder guide, agent SDK

---

## Architecture at a Glance

```
Humans (WebSocket/Telnet)          Agents (WebSocket/JSON)
         │                                  │
         └──────────────┬───────────────────┘
                        │
              ┌─────────▼──────────┐
              │   Go Game Server   │
              │  pkg/session/      │  ← WebSocket, commands
              │  pkg/game/         │  ← world state, combat, AI
              │  pkg/combat/       │  ← formulas from fight.c
              │  pkg/parser/       │  ← world file loading
              └─────────┬──────────┘
                        │
          ┌─────────────┼─────────────┐
          │             │             │
     PostgreSQL       Lua VM      (Redis — future)
   (characters,    (gopher-lua,
    inventory)      world scripts)
```

---

## Key Rules (Repeat From CLAUDE.md)

1. **Read the original source before writing game logic.** It's at `/home/zach/.openclaw/workspace/darkpawns/src/`
2. **Port faithfully, deviate intentionally.** If you add something the original didn't have, say so with a comment.
3. **Don't start the next phase until the current one is done.**
4. **`go build ./cmd/server` must pass before committing.**
5. **Agents play by the same rules as humans. No exceptions.**

---

## Resources

| Thing | Where |
|-------|-------|
| Original Dark Pawns source | `/home/zach/.openclaw/workspace/darkpawns/src/` |
| Original world files | `/home/zach/.openclaw/workspace/darkpawns/lib/` |
| This repo | `/home/zach/.openclaw/workspace/darkpawns-phase1/` |
| GitHub | https://github.com/zax0rz/darkpawns |
| Research docs | `/home/zach/.openclaw/workspace/research/darkpawns_*.md` |
| BRENDA's soul | `/home/zach/.openclaw/workspace/SOUL.md` |
