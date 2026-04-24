---
tags: [active]
---
# Dark Pawns Resurrection — Roadmap

> **For AI assistants:** Read `CLAUDE.md` first. This document tells you what's built.
> `CLAUDE.md` tells you how to build it without making stuff up.
> 
> **Active execution plan → `PORT-PLAN.md`** — Wave details, completion status, model routing, and immediate next steps.
> `ROADMAP.md` = high-level vision and feature inventory.

---

## Vision

Resurrect Dark Pawns (1997–2010 MUD) as a **dual-native multiplayer world** where humans
and AI agents coexist as first-class citizens. Same core game, same rules, modern architecture.

**The point:** Agents are *players*, not NPCs. They show up on WHO, they die, they loot
corpses, they form parties. BRENDA69 and Zach adventure together. That's the end state.

---

## Completed

### World Parser
- 10,057 rooms, 1,313 mobs, 1,620 objects, 95 zones loaded
- Fixed lookahead bug that silently dropped every other object (854 → 1,620)
- Wear flag integers parsed correctly (was treating as letter bitmasks)
- Cross-reference validation (`World.ValidateCrossReferences()`)
- `pkg/parser/` — parsers for `.wld`, `.mob`, `.obj`, `.zon`

### Minimal Engine (Login, Movement, Chat)
- WebSocket server (`pkg/session/`)
- Room state, movement (n/s/e/w/u/d), look, say
- Player login with bcrypt password hashing, JWT tokens (24h expiry, HS256)
- IP-based login rate limiting (token bucket 10/sec, `golang.org/x/time/rate`)
- Per-IP connection limits on telnet (3/IP, 200 total)
- `pkg/auth/` — JWT generation/validation, `JWT_SECRET` required (no fallback)

### Combat & Mobs
- Combat engine with faithful THAC0/AC formulas from `fight.c`
- Mutex-protected `CombatEngine` with `sync.RWMutex`; spawner deadlock fixed
- Attacks-per-round: full per-class/level formula from `fight.c perform_violence`
- AC damage reduction: `get_minusdam()` ported
- EXP loss: `/37` combat deaths, `/3` bleed-out
- Flee XP loss formula from `act.offensive.c`
- Backstab multiplier: `(level*0.2)+1`, capped at 20 for LVL_IMMORT
- `hit`, `flee` commands
- Mob spawning from zone files; aggressive mobs attack on entry; wandering mobs roam
- MOB_MEMORY, MOB_AGGR_EVIL/GOOD/NEUTRAL, MOB_WIMPY, MOB_SCAVENGER, MOB_HELPER
- MOB_STAY_ZONE enforced; ROOM_DEATH/ROOM_NOMOB checked
- Death: leave corpse, respawn at room 8004
- ScriptFightFunc wired — fight trigger fires on mobs each combat round

### Full Play Loop
- Full `ObjectInstance` inventory/equipment system
- get/drop/wear/wield/remove/inventory/equipment commands
- 12 classes, 7 races, stat rolling (`roll_real_abils` from `class.c`)
- Starting items on first login (`do_start` from `class.c`)
- PostgreSQL persistence: save on disconnect, load on login
- StrAdd (18/xx warrior STR) persisted
- advance_level(): per-class HP/mana/move gains, con_app table
- Mob equipped items transferred to corpse on death
- Zone resets: M/O/G/E/R/D commands, age/lifespan ticker

### Lua Engine
- gopher-lua embedded with 10MB memory limit, panic recovery on instruction limit
- Full Dark Pawns script API: `act()`, `do_damage()`, `spell()`, `action()`, `isfighting()`, `send_to_room()`, `number()`, `round()`, `getn()`, `create_event()`
- All trigger types: `oncmd`, `ongive`, `sound`, `fight`, `greet`, `ondeath`, `bribe`, `onpulse`
- Interface-based ScriptContext (`pkg/scripting/types.go`) — broke import cycle
- `room` global with `.vnum` + `.char[]` (all players+mobs)
- `spell()` function binding exists (preliminary stub, full damage dispatch from `magic.c` port is a next step)
- Spell-specific corpse descriptions (fire/cold/lightning/disintegrate)
- DISINTEGRATE scatters gear to room floor, drops ash object
- Priority script loading and trigger dispatch throughout game loop
- `-scripts` server flag for scripts directory

### 115 Archive Scripts Ported
- Combat AI (10): dragon_breath, anhkheg, drake, bradle, caerroil, ettin, snake, troll, mindflayer, paladin
- Economy (10): shopkeeper, shop_give, identifier, stable, merchant_inn, merchant_walk, teacher, recruiter, pet_store, remove_curse
- Environmental (10): donation, eq_thief, aurumvorax, brain_eater, beholder, memory_moss, medusa, sandstorm, phoenix, souleater
- Crafting chains: farmer→miller→baker, crystal_forger, dragon_forger, enchanter, golem trio, tattoo, town_teleport
- Ambient/flavor: beggar, citizen, carpenter, towncrier, minstrel, mime, singingdrunk, bearcub, and 30+ more
- Newbie pipeline: creation.lua, clerk.lua, banker.lua, cityguard.lua
- Special: never_die, sungod, teleporter, teleport_vict, take_jail, triflower, quanlo

### Skills (10 implemented)
All from `act.offensive.c` / `act.other.c` with faithful formulas:
- **backstab** — weapon dam × multiplier, requires piercing weapon, target not fighting
- **bash** — STR check, target sits + stunned, miss = self falls, costs 10 MV
- **kick** — AC-based check, damage = level/2
- **trip** — DEX check, target falls, miss = self falls
- **headbutt** — high damage + stun, 25% self-damage on miss, costs 15 MV
- **rescue** — interpose between attacker and target (stub — combat engine swap not wired)
- **sneak** — toggle, skill check on activation
- **hide** — toggle, skill check on activation
- **steal** — gold or item, weight/level penalty, alert messages on failure
- **pick_lock** — skill check (placeholder, actual door logic pending)

Class/level restrictions and position requirements ported from `class.c` and `interpreter.c`.

Commands: `skills`, `practice`, `learn`, `forget`, `skillinfo`, `listskills`, plus individual skill commands (`backstab`, `bash`, `kick`, `trip`, `headbutt`, `rescue`, `sneak`, `hide`, `steal`, `pick`).

SkillManager with progression, slots, skill points — `pkg/engine/skill.go`, `pkg/engine/skill_manager.go`.

### Snapshot System (lock-free reads)
- `SnapshotManager` with `atomic.Pointer[WorldSnapshot]` — atomic pointer swap
- World writers mutate under write lock, then `PublishSnapshot()` copies the rooms map
- Readers get zero-lock reads of a consistent world view
- Wired into World initialization; generation tracking via `atomic.Uint64`

### Middleware Pipeline
- `command.Registry` supports decorator-style middleware chain
- `LoggingMiddleware()` — logs command name, duration, error status
- `RateLimitMiddleware()` — enforces minimum interval between commands
- `Registry.Use()` API ready; currently no middleware wired on (logging adds noise during dev)

### SQLite Persistence Backend
- `pkg/storage/sqlite.go` — mattn/go-sqlite3 with WAL mode, write-optimized connection pool
- Auto-migration on init: players table with serialized JSON state + timestamps
- Zone reset tracking, mob respawn timers (prepared)
- Not yet wired into `cmd/server/main.go` (currently in-memory only)

### Event Bus (typed pub/sub)
- `events.InProcessBus` — typed events with subscriber pattern
- Event types: `combat.*`, `player.*`, `economy.*`, `world.*`, `game.*`, `admin.*`
- Handlers run sequentially per publish call; subscribe returns unsubscribe function
- Wired into death handling: publishes `MobKilledEvent` and `PlayerKilledEvent`

### Zone Dispatcher (per-zone goroutines)
- `ZoneDispatcher` spawns one goroutine per zone for isolated reset processing
- Initialized with 100ms tick in World setup
- 95 zones each get their own goroutine for resets, AI processing, and state
- `context.Context` per worker for graceful shutdown; separate from serial zone reset loop

### Socials
- ~100 emotes from `lib/misc/socials` in `pkg/game/socials.go` (1,136 lines)
- Wired to command dispatch in `pkg/session/commands.go` via `cmdSocial()`
- `$n`/`$N`/`$e`/`$m`/`$s` pronoun substitution

### Agent Protocol
- Agent auth via API keys (`pkg/db/player.go` — `agent_keys` table)
- Full variable dump with dirty tracking (`pkg/session/agent_vars.go`)
- `ROOM_MOBS` targeting with disambiguation
- Token bucket rate limiting: 10/sec
- `cmd/agentkeygen/main.go` — key generation CLI
- `scripts/dp_bot.py` — 638-line deterministic state machine bot
- `scripts/dp_playtester.py` — LLM-driven playtester

### BRENDA Plays (First Session)
- party/follow/group/gtell commands, XP sharing from `fight.c:1638`
- score/who/tell/emote/shout/where commands
- `scripts/dp_brenda.py` — BRENDA69 agent with SOUL.md personality, mem0, minimax-m2.7
- Narrative memory: Postgres schema, kill/death hooks, memory bootstrap in auth response
- Nightly consolidation (`dp_session_consolidate.py`) and salience decay (`dp_salience_decay.py`)
- Emergent private cognition (`Terminal:` internal monologue)

### Event Queue
- `pkg/events/` — priority-queue event system (container/heap) based on `src/events.c`
- `create_event()` wired from Lua to `World.CreateEvent()`
- Event bus with typed events

### Affect/Buff System
- `pkg/engine/affect.go` — 30+ affect types (stats, combat mods, status effects)
- `pkg/engine/affect_manager.go` — `ApplyAffect()`, tick processing, removal
- `pkg/engine/affect_tick.go` — tick-based affect processing with duration tracking
- Tests: `pkg/engine/affect_test.go`

### Regen / Limits
- `pkg/game/limits.go` — full port of `limits.c`
- `HitGain()`, `ManaGain()`, `MoveGain()` with position bonuses, class modifiers
- Equipment regen bonuses (APPLY_HIT_REGEN, APPLY_MANA_REGEN, APPLY_MOVE_REGEN)
- `GainCondition()` — hunger/thirst/drunk from `limits.c gain_condition()`
- PointUpdate loop applies HMV regen and condition decay

### Spells
- `pkg/spells/spells.go` — 40+ spell constants from `spells.h`
- `Cast()` dispatches damage via `magic.c` formulas
- Non-damage spells classified (teleport, heal, cure, blindness, etc.)

### Privacy & Audit
- `pkg/audit/logger.go` — structured JSON audit log, SHA-256 IP hashing, file-based (0600)
- `pkg/privacy/client.go` — PII filter config (OpenAI-compatible categories)
- `LogSecurityEvent()` for rate limit, auth failures

---

> **Active execution plan → `PORT-PLAN.md`** — See "Immediate Next Steps" section for canonical status.

## In Progress

See `PORT-PLAN.md` → **Immediate Next Steps** for the full breakdown. Summary:

1. **Door commands** — port `act.movement.c` do_gen_door() (open/close/lock/unlock/pick)
2. **Shop buy/sell/list** — port `shop.c` (~1,445 lines). Scripts exist; engine commands missing
3. **Rescue combat wiring** — `DoRescue()` needs `StopCombat()`/`StartCombat()` swap
4. **Hitroll/damroll from equipment** — `formulas.go` returns 0 for gear bonuses
5. **Spell effects beyond damage** — affects exist; need wiring from spell dispatch
6. **Memory dreaming layer** — REM synthesis (`dp_rem_synthesis.py`) and deep promotion (`dp_memory_promote.py`)

---

## Known Issues

- **No Lua instruction limit** — memory limit (10MB) enforced, but no opcode/instruction cap. A runaway script burns CPU until memory limit hits.
- **No shop commands wired** — scripts fire but players can't buy/sell.
- **No door commands** — `door_commands.go` deleted, not yet replaced.
- **Hitroll/damroll from equipment = 0** — combat accuracy/damage ignores gear bonuses.
- **Non-damage spells are fire-and-forget** — `spell()` deals damage but doesn't apply affects (blindness, curse, etc.).
- **Sneak/hide not checked in combat/movement** — state tracked but no movement bonus or combat avoidance.
- **Pick lock is a placeholder** — `DoPickLock()` returns a message but doesn't interact with doors (because doors don't exist yet).
- **No mob position recovery** — bash/trip set position to sitting, but mobs don't stand back up on their own.
- **`dp_playtester.py` requires working LiteLLM** — not standalone.

---

## Out of Scope

- **Web client** (React/VT100) — Phase 6, post-engine-completeness
- **GMCP/MXP** — telnet protocol extensions, Phase 6
- **Admin tools** (zone editor, spawn commands) — Phase 6
- **Public hosting** (VPS/Fly.io) — Phase 6
- **Redis** — future caching layer, not needed yet
- **Original C codebase** — reference only, not running
- **Mobile companion app** — never planned

---

## Architecture

```
Humans (WebSocket/Telnet)          Agents (WebSocket/JSON)
         │                                  │
         └──────────────┬───────────────────┘
                        │
              ┌─────────▼──────────┐
              │   Go Game Server   │
              │  pkg/session/      │  ← WebSocket, commands, auth
              │  pkg/game/         │  ← world state, combat, AI, limits
              │  pkg/combat/       │  ← formulas from fight.c
              │  pkg/parser/       │  ← world file loading
              │  pkg/engine/       │  ← affects, skill manager
              │  pkg/events/       │  ← timer event queue
              │  pkg/scripting/    │  ← gopher-lua VM
              │  pkg/spells/       │  ← spell constants, Cast()
              │  pkg/command/      │  ← command handlers, skill commands
              └─────────┬──────────┘
                        │
          ┌─────────────┼─────────────┐
          │             │             │
     PostgreSQL       Lua VM      Events
   (characters,    (gopher-lua,  (timer queue,
    inventory)      115 scripts)   create_event)
```

---

## Key Rules (From CLAUDE.md)

1. **Read the original source before writing game logic.** It's at `src/`
2. **Port faithfully, deviate intentionally.** Comment when you add something new.
3. **Don't start the next phase until the current one is done.**
4. **`go build ./cmd/server` must pass before committing.**
5. **Agents play by the same rules as humans. No exceptions.**

---

## Resources

| Thing | Where |
|-------|-------|
| Original Dark Pawns source | `src/` |
| Original world files | `lib/` |
| GitHub | https://github.com/zax0rz/darkpawns |
| Active execution plan | `PORT-PLAN.md` |
| Architecture research | `docs/research.md` |
| Session journal | `RESEARCH-LOG.md` |
| Swarm learnings | `docs/SWARM-LEARNINGS.md` |
| BRENDA's soul | `/home/zach/.openclaw/workspace/SOUL.md` |
| Agent protocol spec | `docs/agent-protocol.md` |
| First fight transcript | `docs/brenda-first-fight-2026-04-21.txt` |
