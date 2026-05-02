# Dark Pawns C→Go Port Fidelity Audit

**Date:** 2026-04-25  
**Auditor:** Claude Opus 4 (6 parallel subagents)  
**Scope:** 66 C source files (~69K lines) vs Go port (189 files, ~69K lines)

---

## Executive Summary

The Go port is **functionally complete at a high level** — all major game systems exist and the core gameplay loop works. However, the audit identified significant gaps in edge cases, secondary systems, and game text infrastructure. The most critical finding is that the **`act()` message substitution engine is entirely missing**, which affects combat messages, socials, mobprogs, and spec procs throughout the game.

### By Category

| Category | Status | Key Issue |
|----------|--------|-----------|
| Spec Procs | ✅ Good | 116/126 SPECIALs ported, 4 are dead code, 3 are stubs |
| Player Commands | ⚠️ Mixed | Core commands work, 12+ informational commands missing, some simplified |
| Combat System | ⚠️ Mixed | Formulas match, but missing edge cases (parry, autogold, death mechanics) |
| World Loading | ⚠️ Mixed | Core parsing works, mob stats not scaled, object flags truncated, zone reset simplified |
| Scripts/Lua | ⚠️ Mixed | Engine works, 20 of 53 C lua_ functions are stubs |
| Mobact | ❌ Weak | 3 of 4 functions missing/incomplete |
| Handler | ❌ Weak | Mount system, follower management, equipment edge cases missing |
| Networking | ⚠️ Mixed | Core works, `act()` substitution missing, MCCP missing |

---

## Critical Gaps (Fix First)

### 1. `act()` Message Substitution Engine — ❌ MISSING
- **C:** `src/comm.c` — `act()` and `perform_act()` with `$n`, `$N`, `$e`, `$E`, `$m`, `$M`, `$s`, `$S` substitution
- **Go:** Not found anywhere in the codebase
- **Impact:** HIGH — used by every social, combat message, mobprog trigger, and many spec procs
- **Files affected:** Nearly every file in `pkg/game/` and `pkg/session/`

### 2. Mount/Riding System — ❌ MOSTLY MISSING
- **C:** `src/utils.c` — `get_mount()`, `get_rider()`, `unmount()`, `die_follower()`
- **C:** `src/handler.c` — mount checks in `do_stand()`, `do_sit()`, `do_rest()`, `do_sleep()`
- **C:** `src/act.movement.c` — dismount on move, ride command
- **Go:** Mount/ride command exists but underlying system is skeletal
- **Impact:** HIGH — riding mechanics are a core player feature

### 3. Follower Management — ❌ MOSTLY MISSING
- **C:** `src/utils.c` — `add_follower()`, `stop_follower()`, `die_follower()`, `circle_follow()`, `add_follower_quiet()`
- **Go:** Many `// TODO` comments in `spec_procs2.go` referencing these
- **Impact:** HIGH — affects group behavior, mob AI, and many spec procs

### 4. Mob Parser — Stat Scaling Missing
- **C:** `src/db.c` `parse_simple_mob()` — level-based stat boosts for mobs 15+, THAC0→hitroll conversion, AC scaling
- **Go:** `pkg/parser/mob.go` — reads base stats but doesn't apply level scaling
- **Impact:** CRITICAL — every mob in the game has incorrect combat stats

### 5. Object Parser — WearFlags Truncated
- **C:** `src/db.c` `parse_object()` — WearFlags is [4] array
- **Go:** `pkg/parser/obj.go` — WearFlags is [3] array (missing one slot)
- **Impact:** HIGH — objects in the 4th wear position won't equip correctly

### 6. Room Parser — Missing Fields
- **C:** `src/db.c` `parse_room()` — 6 numeric fields including room_flags[4], extra descriptions, room scripts
- **Go:** `pkg/parser/wld.go` — only reads 3 fields, no extra descriptions, no room scripts
- **Impact:** HIGH — room flags and descriptions are incomplete

### 7. Zone Reset — `percent_load()` Missing
- **C:** `src/db.c` — object load probability (0-100%)
- **Go:** All objects load at 100%
- **Impact:** MEDIUM — rare items spawn every reset instead of probabilistically

### 8. Zone Reset — Conditional/Loop Commands Missing
- **C:** `src/db.c` `reset_zone()` — `L` loop command, `if_flag` conditional chaining, random zone placement
- **Go:** `pkg/game/spawner.go` — basic M/O/G/E/P/D/R only
- **Impact:** MEDIUM — several zones rely on these for correct behavior

---

## Combat System Gaps

### `fight.c`
- **`die_with_killer()`**: Go always loses 1 con; C has level checks and random chance
- **`damage()`**: Race hate applies once instead of up to 5x; missing jail guard logic, charm retarget, NPC target switching, autoloot, PK outlaw flagging
- **`group_gain()`**: Autogold/autosplit (~100 lines) entirely missing
- **`perform_violence()`**: Parry system, NPC dodge, NPC mob_wait missing
- **`dam_message()`**: Damage tier thresholds and messages differ from C
- **`counter_procs()`**: Kill milestone rewards are a stub (checks but does nothing)
- **`load_messages()`**: Combat skill messages not loaded from file
- **`make_dust()`**: Doesn't distinguish dust vnum 18 vs vampire_dust vnum 1230
- **Bug:** Damage message uses damage amount for position lookup instead of victim HP

### `spells.c` / `magic.c` / `spell_parser.c`
- Core spell system works; individual spell edge cases need verification
- `call_magic()` ported; tier dispatch works

---

## Player Command Gaps

### `act.informative.c`
- **`do_gen_ps()`**: 12+ commands missing — credits, news, motd, imotd, version, wizlist, handbook, policies, whoami, clear
- **`do_score()`**: Heavily simplified — missing alignment strings, play time, age, weight, detailed AC
- **`do_who()`**: Complex flag filtering simplified away (-o, -k, -l, class filter, level range)
- **`show_mult_obj_to_char()`**, **`list_obj_in_heap()`**, **`find_exdesc()`**: Missing display functions
- **`do_abils()`**, **`do_levels()`**, **`do_coins()`**: Exist but NOT registered in command registry

### `act.other.c`
- **`do_steal()`**: Missing ROOM_PEACEFUL check, mounted check, kender_steal, robbery affect

### `act.item.c` / `act.comm.c` / `act.display.c`
- Core commands (get/put/drop/give/wear/remove/drink/eat/say/tell/shout) all ported
- Minor commands (do_auction, do_music) may be missing

---

## Handler / Equipment Gaps

### `handler.c`
- **`equip_char()`**: Missing anti-alignment zap (ITEM_ANTI_EVIL/GOOD/NEUTRAL), invalid_class check, light tracking, check_for_bad_stats()
- **`unequip_char()`**: Missing ITEM_TAKE_NAME restoration, AC restoration, light tracking
- **`apply_ac()`**: Missing body-part multiplier (body×3, head/legs×2, others×1)
- **`affect_total()`**: Missing tattoo_af() calls, equipment iteration, stat clamping, alignment clamping
- **`check_for_bad_stats()`**: Entirely missing — zero-stat penalties
- **`aff_apply_modify()`**: Missing APPLY_AGE, APPLY_CHAR_WEIGHT/HEIGHT, APPLY_GOLD, regen rates, APPLY_RACE_HATE, APPLY_SPELL

---

## Movement Gaps

### `act.movement.c`
- **Mount integration**: Missing from do_stand, do_sit, do_rest, do_sleep, do_move
- **`do_follow()`**: Missing circle_follow() loop detection, AFF_CHARM restriction, shadow quiet-follow, add_follower_quiet()
- **`ok_pick()`**: Missing lockpick breakage mechanic (break → vnum 8028), level-based break chance
- **`do_enter()`**: Missing indoors fallback when no argument
- **`do_leave()`**: Different logic — Go searches "exit" keyword, C checks ROOM_INDOORS
- **`do_wake()`**: Missing AFF_SLEEP check, position threshold check

---

## World Loading Gaps

### `db.c`
- **`parse_room()`**: Only 3 of 6 numeric fields; missing room_flags[4], extra descriptions, room scripts
- **`parse_mob()`**: Missing level-based stat scaling, THAC0→hitroll, AC scaling, save throws, race hate arrays, auto-lowercase of articles
- **`parse_object()`**: WearFlags [3] instead of [4]; missing object script parsing, container weight validation
- **`percent_load()`**: Missing — all objects load at 100%
- **`reset_zone()`**: Missing L loop, if_flag conditionals, random zone placement, door-state logic
- **`reset_time()`**: MUD time/date system not implemented
- **`load_help()`**: Help system not ported
- **`init_review_strings()`**: Review skill string table setup missing

### `class.c`
- Stat rolling, starting gear, practice costs, exp tables — mostly ported
- Some edge cases in stat assignment order may differ

### `shop.c`
- Core buy/sell works
- **repair, identify, value commands not registered**
- **Shop gold balance doesn't exist** (C has it, Go treats as unlimited)

### `objsave.c`
- Save/load ported from binary struct to JSON — functionally equivalent but format changed

---

## Spec Procs — Good Shape

- **116 of 126 SPECIALs fully ported**
- 10 "missing" specs are dead code (declared but never ASSIGNMOB'd/ASSIGNROOM'd)
- 3 stubs in spec_procs3.c (elements_galeru_column, elements_minion, elements_galeru_alive)
- **spec_procs.c**: guild, magic_user, mayor partially simplified
- **spec_procs2.c**: tattoo_af partially simplified
- **spec_assign.c**: All 6 functions ported

---

## Mobact — Weak

- **`mobile_activity()`**: Mostly missing — Go has basic structure but missing:
  - Scavenger behavior (pick up items)
  - Memory behavior (remember attackers, hunt)
  - Helper behavior (assist friendly mobs)
  - AGGRESSIVE behavior with door opening
  - AGGR24 behavior
  - HUNTER behavior
  - STAY_ZONE behavior
  - Race hate checks
- **`find_mob_in_room()`**: Present but simplified
- **`mob_ai_tick()`**: Exists but shallow

---

## Scripts/Lua — Mixed

- **31 of 53 C lua_ functions fully ported**
- **20 functions are STUBS** — declared but return nil/empty
- **4 functions entirely MISSING**
- **6 NEW Go-only functions** (no C equivalent)
- Key stubs include: several Lua library functions, script state management

---

## Networking — Mixed

- Core game loop, session management, telnet, WebSocket — all working
- **`act()` substitution engine entirely missing**
- **`sprintbit()`/`sprinttype()`** — bitvector/type converters missing
- **`color_expansion()`** — color code expansion missing
- MCCP compression not implemented
- Signal handlers replaced by Go context cancellation (by design)
- DNS cache not needed in Go

---

## Utility Functions Missing

- `sprintbit()`, `sprinttype()` — bitvector/type display
- `parse_race()` — race name parsing
- `age()` — character age calculation
- `playing_time()` — play-time calculation
- `circle_follow()` — follow loop detection
- Mount system: `get_mount()`, `get_rider()`, `unmount()`, `die_follower()`, `add_follower()`, `stop_follower()`
- `log_death_trap()` — simplified to slog
- `mud_time_passed()` — simplified

---

## Priority Fix List

### P0 — Breaks Core Gameplay
1. Port `act()` substitution engine
2. Fix mob parser stat scaling (level-based boosts, THAC0, AC)
3. Fix object parser WearFlags [3]→[4]
4. Fix room parser missing fields (flags, exdesc, scripts)

### P1 — Significantly Impacts Gameplay
5. Implement follower management (add/stop/die_follower, circle_follow)
6. Implement mount/riding system (get_mount, dismount, mount checks in position commands)
7. Fix `die_with_killer()` death mechanics (level check, random chance)
8. Fix `damage()` edge cases (race hate 5x, jail guard, charm retarget, autoloot)
9. Implement group_gain autogold/autosplit
10. Implement parry system and NPC dodge in perform_violence()

### P2 — Visible to Players
11. Port `do_gen_ps()` (credits, news, motd, version, etc.)
12. Fix `do_score()` display
13. Fix `do_who()` flag filtering
14. Implement help system (`load_help()`)
15. Implement MUD time/date system
16. Fix dam_message() tiers to match C

### P3 — Polish / Edge Cases
17. Fix `equip_char()` alignment zap, invalid_class check
18. Fix `apply_ac()` body-part multipliers
19. Implement lockpick breakage
20. Fix zone reset (percent_load, loop commands, conditionals)
21. Port remaining Lua library functions (20 stubs)
22. Implement mobact behaviors (scavenger, memory, helper, aggressive, hunter)
23. Fix shop commands (repair, identify, value, gold balance)
24. Register unregistered commands (abils, levels, coins)

---

*Generated by Claude Opus 4 — 6 parallel subagents, ~69K lines C vs ~69K lines Go*
