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

---

## What's Next

---

### 🔲 Phase 3 — Lua Scripting
**Goal:** The world feels alive. Traps spring, mobs talk, quests trigger.

The original has 179 Lua scripts. We don't port all of them — we build the engine
and port the most important ones. The rest follow naturally.

**Lua engine:**
- Embed `gopher-lua` (`github.com/yuin/gopher-lua`)
- Expose the original API surface to Lua (from `scripting.c`):
  - `act(msg, ch)` — send message
  - `do_damage(ch, victim, dam)` — deal damage
  - `ch.hp`, `ch.level`, `ch.name` — character state reads
  - `number(low, high)` — RNG (maps to `rand.Intn`)
  - `send_to_room(msg, room)` — broadcast to room
- Trigger types from original: `onpulse()`, `oncmd()`, `onenter()`, `ondeath()`

**Script priority order** (start here, not with all 179):
1. Room traps — demonstrate `onenter()` working
2. Shopkeeper mobs — demonstrate `oncmd()` working  
3. Quest-giving mobs — demonstrate state tracking
4. Special items (magic, cursed) — demonstrate `onuse()`

**Source reference:** `scripting.c`, `lib/scripts/*.lua`

**Deliverable:** Walk into a trapped room, spring the trap, take damage.
Buy something from a shopkeeper. Accept a quest.

---

### 🔲 Phase 4 — Agent Protocol
**Goal:** A bot can connect and play the game programmatically.

The WebSocket/JSON transport already exists. What's missing is the agent-specific layer.

**API key auth:**
- `{"type":"auth_apikey","data":{"key":"dp_abc123"}}` instead of player_name/password
- Keys stored in PostgreSQL, associated with a character
- Source reference: LoFP bot API design (see `research/darkpawns_resurrection_research.md`)

**Structured state for agents:**
- After every action, agents receive full structured state (not just text):
```json
{
  "room": {"vnum": 3001, "name": "...", "exits": ["north","east"], "mobs": [...], "items": [...]},
  "self": {"hp": 45, "max_hp": 60, "level": 3, "fighting": "goblin"},
  "events": [{"type":"combat","attacker":"goblin","damage":8}]
}
```
- Human players continue to receive text. Agents opt into structured mode at auth.

**Rate limiting:**
- Same rules as humans: can't spam 1000 commands/second
- 1 action per 100ms, combat locked to engine tick rate (2s rounds)

**Deliverable:** A simple Python bot connects, walks around, finds a mob, kills it,
loots the corpse, and reports back. Prove agents play by the same rules.

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
