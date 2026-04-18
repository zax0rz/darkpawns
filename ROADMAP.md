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
- 10,057 rooms, 1,313 mobs, 854 objects, 95 zones loaded
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
- Death: lose EXP/3, leave a corpse, respawn at room 8004
- Inventory and equipment structure exists but uses prototypes (`*parser.Obj`), not runtime instances

---

## What's Next

### 🔲 Phase 2b — "I Can Actually Play This"
**Goal:** A complete play loop. Log in, kill something, loot it, equip the weapon,
log out, log back in and still have your stuff.

This phase finishes what Phase 2 started before moving to Lua scripting.
Lua scripts manipulate items and check character stats constantly — none of that
works until this foundation is solid.

**Items (get/drop/equip/use):**
- Migrate player inventory from `*parser.Obj` to `*ObjectInstance` (runtime state)
- `get <item>` — pick up from room
- `drop <item>` — drop to room
- `equip <item>` / `wear <item>` — equip from inventory
- `remove <item>` — unequip
- `inventory` / `equipment` commands
- Corpse now actually transfers items (blocked on ObjectInstance migration)
- Source reference: `act.obj.c`, `handler.c`

**Character creation:**
- Pick class (12 classes from original) and race (7 races)
- Stats rolled at creation — port `do_start()` from `db.c`
- Classes: Mage, Cleric, Thief, Warrior, Magus, Avatar, Assassin, Paladin, Ninja, Psionic, Ranger, Mystic
- Races: Human, Elf, Gnome, Dwarf, Half-Elf, Halfling, Half-Orc
- Stats now flow into combat formulas (STR tohit/todam, DEX defensive, etc.)
- Source reference: `class.c`, `db.c`

**Player persistence (PostgreSQL):**
- Save and load character on login/logout
- Schema: name, class, race, level, exp, hp, stats, room_vnum, inventory
- The DB connection is already wired in `pkg/db/` — just needs the actual save/load logic
- Source reference: `db.c` (save_char, load_char)

**Deliverable:** Kill a rat. Loot a sword from its corpse. Equip it. Log out. Log back in
with the sword still equipped. THAC0 reflects your class and STR.

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
