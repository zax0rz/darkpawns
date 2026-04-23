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

### ✅ Phase 3D — Lua Engine Completion (2026-04-21)
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

---

## What's Next

---

### ✅ Phase 4 — Agent Protocol (2026-04-21)
**Goal:** A bot can connect and play the game programmatically. Agents are first-class players.

**Deliverable met:** `brenda69` connects via API key, receives full var dump, navigates, attacks.
Smoke test confirmed: agent auth, variable subscription, ROOM_MOBS targeting all working.

**What shipped:**
- `pkg/session/agent_vars.go` — full variable set, dirty tracking, flush hook, ROOM_MOBS disambiguation
- `pkg/session/agent.go` — sendFullVarDump(), subscribe handler
- `pkg/db/player.go` — agent_keys table, CreateAgentKey(), ValidateAgentKey()
- `cmd/agentkeygen/main.go` — key generation CLI
- `scripts/dp_bot.py` — 638-line deterministic state machine bot
- `scripts/dp_playtester.py` — LLM-driven playtester (requires working LiteLLM)
- **Rate limiting:** token bucket 10/sec via golang.org/x/time/rate
- **Bugs fixed:** spawner mutex deadlock, SpawnMob world.mu self-deadlock, zone resets on startup

---

### ✅ Phase 5 — BRENDA Plays (2026-04-21)
**Goal:** BRENDA69 has a character. She and Zach can adventure together.

**Deliverable met:** First live party session. BRENDA69 connected, partied with Zach, fought a knight templar 3 times (~100 misses, 2 hits), fled, invoked the ZFS pool. Transcript: `docs/brenda-first-fight-2026-04-21.txt`

**Shipped:**
- party/follow/group/gtell commands (`pkg/game/party.go`, XP sharing from fight.c:1638)
- score/who/tell/emote/shout/where commands (`pkg/session/commands.go`)
- Lua script fixes: fight trigger arg, nil ch in onpulse_pc, bane state machine, breed_killer obj
- Engine stubs: isnpc, cansee, plr_flagged, tell, has_item, obj_in_room, objfrom, objto
- `scripts/dp_brenda.py` — BRENDA69 agent with SOUL.md personality, mem0 (frankendell .15), minimax-m2.7 LLM
- `docs/agent-protocol.md` — Agent protocol specification
- `RESEARCH-LOG.md` — living research document for AIIDE 2027 paper
- `agent_narrative_memory` Postgres schema + kill/death hooks + memory bootstrap in auth response
- `scripts/dp_session_consolidate.py` — nightly LLM session summary (cron 2:00 AM)
- `scripts/dp_salience_decay.py` — nightly salience decay + pruning (cron 2:30 AM)
- mem0 pointed at frankendell (192.168.1.15) Qdrant + Ollama
- BRENDA69 API key in Vaultwarden as "Dark Pawns Agent Key — brenda69"

**Discovery:** Emergent private cognition — BRENDA generated `Terminal:` internal monologue that stayed in the bot process, never hit the game. Public speech vs private thought. The seed of the paper.

---

### ✅ Phase 5b — World Restoration (2026-04-21)
**Goal:** All 115 archive Lua scripts ported and working.

**Deliverable met:** 115/115 scripts ported in one session via K2.6/Claude Code parallel swarms.

**All tiers complete:**
- Tier 1: Engine stubs (isnpc/cansee/plr_flagged/tell/has_item/obj_in_room/objfrom/objto)
- Tier 2: Combat AI (10/10) — dragon_breath, anhkheg, drake, bradle, caerroil, ettin, snake, troll, mindflayer, paladin
- Tier 3: Economy (10/10) — shopkeeper, shop_give, identifier, stable, merchant_inn, merchant_walk, teacher, recruiter, pet_store, remove_curse
- Tier 4: Environmental (10/10) — donation, eq_thief, aurumvorax, brain_eater, beholder, memory_moss, medusa, sandstorm, phoenix, souleater
- Tier 5: Crafting chains — farmer_wheat, miller, baker_flour, baker_dough, crystal_forger, dragon_forger, enchanter, golem trio, tattoo, town_teleport
- Tier 6: Ambient/flavor — beggar, citizen, carpenter, towncrier, minstrel, mime, singingdrunk, bearcub, and 30+ more
- Special mechanics — never_die, sungod, teleporter, teleport_vict, take_jail, triflower, quanlo

**Engine gaps still open (deferred to Phase 5c):**
- `create_event` — timer/event queue; ~20 scripts have TODO stubs waiting on this
- Affect/buff system (`handler.c` affect_to_char) — spells don’t persist without it
- Doors (open/close/lock/unlock)
- Shop buy/sell commands (scripts exist, engine commands missing)
- Skills (backstab, bash, kick, etc.)
- Regen / move points / hunger (`limits.c`)

---

### 🔲 Phase 5c — Engine Completeness
**Goal:** The engine gaps that scripts are waiting on.

- **`create_event` / event queue** — timer-based callbacks; unblocks ~20 scripts with TODO stubs
- **Affect/buff system** — port `handler.c` affect_to_char/affect_remove/affect_total; persistent spell effects
- **Doors** — open/close/lock/unlock/pick from `act.movement.c`
- **Shop buy/sell commands** — `shop.c` (1445 lines); scripts exist, commands missing
- **Skills** — backstab, bash, kick, trip, headbutt from `act.offensive.c`
- **Regen / move points / hunger** — `limits.c` gain_condition(), mana_gain(), hit_gain()
- **Social emotes** — ~100 emotes from `act.social.c` (laugh, bow, wave, etc.)
- **Hitroll/damroll from equipment** — currently returns 0 (TODO in formulas.go)

---

### 🔲 Phase 5d — Memory Dreaming Layer
**Goal:** BRENDA builds durable memory that actually changes her behavior over sessions.

Inspired by OpenClaw's dreaming system (Light → REM → Deep phases). The insight: don't promote every memory — rank candidates by how *useful* they actually were, then promote only the ones that passed threshold.

**Three-phase model:**
- **Light phase** (`dp_session_consolidate.py`, already live) — ingest raw session events, generate compact summary. No long-term write yet.
- **REM phase** (`dp_rem_synthesis.py`, to build) — weekly pass across 7 days of session summaries. Finds recurring patterns. "BRENDA dies to templar-class mobs consistently" is a REM insight, not a session note. Writes high-salience PATTERN memories.
- **Deep phase** (to build) — ranking + threshold gate before any memory promotes to permanent. Six signals: frequency, relevance (was this memory retrieved and *used*?), query diversity, recency, multi-session recurrence, conceptual richness.

**Retrieval tracking** — when bootstrap delivers a memory and BRENDA acts on it, mark it used. Unused memories stop promoting. This is the relevance signal.

**Private/public split** — `Terminal:` internal monologue writes to mem0 separately from `say` output. BRENDA builds a private world model that diverges from her public persona over time. First observed 2026-04-21 (see RESEARCH-LOG.md).

**Scripts to build:**
- `scripts/dp_rem_synthesis.py` — weekly REM cron, pattern extraction across session summaries
- `scripts/dp_memory_promote.py` — deep phase ranking + threshold gate
- Retrieval tracking hooks in `dp_brenda.py` — log which bootstrap memories influenced decisions
- Private thought writer — route `Terminal:` output to mem0 separately from game speech

**Cron schedule (target):**
- 2:00 AM daily — Light (consolidate) ✅ live
- 2:30 AM daily — Salience decay ✅ live
- 3:00 AM Sunday — REM synthesis (weekly)
- 3:30 AM Sunday — Deep promotion (weekly, after REM)

**Research value:** Once retrieval tracking is live, we can measure whether memories actually influenced behavior — that's the "narrative coherence" evaluation metric the AIIDE 2027 paper needs.

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
