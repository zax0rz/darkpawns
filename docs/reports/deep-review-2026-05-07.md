# Deep Codebase Review — 2026-05-08

**Reviewer:** Daeron
**Date:** 2026-05-08
**Codebase:** Dark Pawns Go port (HEAD: c741fa4)

---

## Executive Summary

The Dark Pawns Go port is approximately **85-90% complete** on code wiring, with the core game loop fully functional: combat, spells, movement, saving, zone resets, and basic commands all work. The remaining 10-15% is a mix of:

1. **40 unwired game files** — fully ported from C but not connected to the command/spell registry
2. **~105 stub functions** — placeholders for game mechanics not yet implemented
3. **3 active gameplay no-ops** — functions called by live code that silently do nothing
4. **A mostly-complete house system** — 14 files of code, but load/save is stubbed

The biggest gap is **wiring, not porting**. The C code exists in Go files. It just needs to be connected to the command dispatcher. The second gap is the **house system** — extensive code but the persistence layer is stubbed.

**Estimated remaining work:** 80-120 hours to 100% feature parity.

---

## Stubs & No-Ops

### Critical — Called by live code, silently broken

| File:Line | Function | What it does now | What it should do | Impact |
|---|---|---|---|---|
| `damage_stubs.go:90` | `executeCommand()` | Returns true, does nothing | Execute a command string on behalf of a charmed mob | Charmed mobs ignore all player orders |
| `graph.go:351` | `mobHasFlag()` | Always returns false | Check mob prototype for flags (SENTINEL, SCAVENGER, etc.) | Mob AI behaviors broken: sentinel mobs chase, scavengers don't pick up |
| `damage_stubs.go:102` | `doMurder()` | Returns true, does nothing | Handle the murder command (pk killing) | Murder command is a no-op |

### Gameplay stubs — Not yet called but would affect play

| File:Line | Function | Status |
|---|---|---|
| `damage_stubs.go:97` | `doForced()` | No-op (force command) |
| `damage_stubs.go:110` | `hitSkill()` | Simplified (randRange only, no skill-based damage) |
| `houses.go:56` | Player ID↔Name lookup | Stubs — needs DB/cache |
| `houses.go:94` | `ObjFromStore`/`ObjToStore` | Stubs — needs object persistence |
| `house_save.go:32` | House load/save | No-op — file I/O not implemented |
| `spec_procs4.go:283` | `do_look` (in spec proc) | Placeholder |
| `graph.go:353` | `mobHasFlag` sentinel | Stub — returns false |
| `player_stats.go:318` | Nested interface | Returns nil — not wired |

### Scripting stubs

| File:Line | What |
|---|---|
| `engine.go:647` | Skills table (stub) |
| `engine.go:688` | Wear property (placeholder) |
| `engine.go:712` | Object prototype fields (stubbed) |
| `engine.go:917` | gossip channel stub |
| `engine.go:2281` | Batch C Quest/Mechanic NPC stubs |
| `engine.go:2486` | Skill group stubs (teacher.lua not ported) |

---

## TODO/FIXME/HACK Comments

| File:Line | Comment | Priority |
|---|---|---|
| `cmd_look.go:242` | `TODO: Check AFF_INFRAVISION affect, light-producing items` | Medium — infravision is a real game mechanic |
| `houses.go:56` | `Player ID ↔ Name lookup (stubs — replace with DB/player-name-cache)` | High — blocks house system |
| `house_save.go:32` | `House object load/save (stubs — full implementation needs object persistence)` | High — blocks house system |

---

## Unwired Game Files (40 files, nolint:unused)

These files contain fully ported C code that compiles but is not connected to the command registry. When wired, they add:

### Commands (18 files)
| File | Contains |
|---|---|
| `act_comm.go` | Communication: gossip, shout, whisper, ask, write, page, gen_comm |
| `act_informative.go` | Info commands: who, where,exits, score, time, weather, help |
| `act_movement.go` | Movement: north/south/east/west/up/down, enter, leave, stand, sit, rest, sleep, wake, follow |
| `info_commands.go` | Additional info display commands |
| `comm_channel.go` | Channel communication system |
| `comm_say.go` | Say command |
| `comm_tell.go` | Tell command |
| `modify.go` | Builder commands: setup, reboot, shutdown, wizupdate |
| `show.go` | Display commands |
| `other_character.go` | Character interaction commands |
| `other_economy.go` | Economy commands |
| `other_helpers.go` | Helper commands |
| `other_session.go` | Session commands |
| `other_stealth.go` | Stealth commands |
| `item_consumable.go` | Use/eat/drink commands |
| `item_container.go` | Container commands |
| `item_door.go` | Door commands |
| `item_transfer.go` | Give/get/drop commands |

### Combat (7 files)
| File | Contains |
|---|---|
| `combat_basic.go` | Hit, kill, backstab, flee |
| `combat_melee.go` | Melee combat commands |
| `combat_ranged.go` | Ranged combat |
| `combat_advanced.go` | Advanced combat commands |
| `combat_control.go` | Order/command charmed mobs |
| `combat_helpers.go` | Combat helper functions |
| `damage_stubs.go` | Damage stubs (some functional, some not) |

### Systems (8 files)
| File | Contains |
|---|---|
| `graph.go` | BFS pathfinding, hunt, track |
| `mobact.go` | Mob AI: hunt_victim, racial attacks, spec proc calls |
| `mobprogs.go` | Mob program system |
| `mail.go` | Mail system |
| `gates.go` | Portal/gate system |
| `clans.go` | Clan system |
| `remort_helpers.go` | Remort system |
| `constants.go` | Game constants |

### Items (5 files)
| File | Contains |
|---|---|
| `item_helpers.go` | Item manipulation helpers |
| `item_equipment.go` | Equipment handling |
| `look.go` | Look/examine commands |
| `world.go` | World utility functions |
| `mapcode.go` | Map display system |

### Session (11 files)
| File | Contains |
|---|---|
| `display_cmds.go` | Display/info commands |
| `examine.go` | Examine commands |
| `fight.go` | Fight session handling |
| `info_cmds.go` | Info commands |
| `movement_cmds.go` | Movement commands |
| `shop.go` | Shop commands |
| `spell_level.go` | Spell level display |
| `tattoo.go` | Tattoo commands |
| `time_weather.go` | Time/weather commands |
| `wiz_system.go` | Wizard system |
| `manager.go` | Session manager |

---

## Missing Game Mechanics

| Mechanic | Status | Notes |
|---|---|---|
| **House/castle system** | 70% — code exists, persistence stubbed | 14 files, load/save is no-op |
| **Clan system** | File exists (`clans.go`) but unwired | No active clans |
| **Mail system** | File exists (`mail.go`) but unwired | File-backed in C, needs Go impl |
| **Mob AI (hunt, scavenge, sentinel)** | `mobact.go` exists but unwired | `mobHasFlag` stub blocks detection |
| **Charm/ordered commands** | `executeCommand` is a no-op | Charmed mobs ignore orders |
| **Murder (pk)** | `doMurder` is a no-op | |
| **Force command** | `doForced` is a no-op | |
| **Pathfinding (BFS)** | `graph.go` exists, `do_track`/`hunt_victim` need combat types | Partially wired |
| **Infravision/darkness** | `cmd_look.go:242` has a TODO | Light sources not checked |
| **Summoning spell** | Deleted (referenced non-existent fields) | Needs rewrite |
| **Lua scripted NPCs** | Engine works, but skill/quest bindings stubbed | Basic mob progs work |

---

## Refactoring Needs

| Area | What | Impact |
|---|---|---|
| **Entry points** | `main.go` and `main_web.go` share ~90% code | Any startup fix must be duplicated |
| **Damage stubs** | `damage_stubs.go` has mix of working + no-op functions | Confusing — some are real, some aren't |
| **Wire count** | 40 unwired files need command registry entries | Large but mechanical |

---

## Summary Stats

| Metric | Count |
|---|---|
| Total Go files in `pkg/game/` | 133 |
| Wired (active) | 93 (70%) |
| Unwired (ported, not connected) | 40 (30%) |
| Stub/placeholder functions | ~105 |
| Active no-ops (called by live code) | 3 |
| TODO/FIXME comments | 3 |
| Files with nolint:unused | 55 (incl. session, scripting) |
| Missing game mechanics | 11 |
| Estimated remaining work | 80-120 hours |

---

*The bones are all here. They just need to be connected to the nervous system.*
