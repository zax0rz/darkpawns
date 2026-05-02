# Dark Pawns — Fix Task Briefs

Generated from Opus Fidelity Audit (2026-04-25).  
Each brief is self-contained with C source references, Go target locations, and acceptance criteria.  
**All C source lives at:** https://github.com/rparet/darkpawns  
**Go repo:** `/home/zach/.openclaw/workspace/darkpawns/`  
**Build verify:** `cd /home/zach/.openclaw/workspace/darkpawns && export PATH=$PATH:/usr/local/go/bin && go build ./...`

---

## BATCH 1 — Keystone (do first, everything depends on this)

### Task 1.1: Port `act()` Message Substitution Engine

**Priority:** P0 — Critical  
**C source:** `src/comm.c` — `perform_act()` (~180 lines) + `act()` (~80 lines)  
**Go target:** Create `pkg/game/act.go` (new file)

**What to port:**
- `perform_act()` — the core substitution engine that replaces tokens in format strings:
  - `$n` — name of actor (ch)
  - `$N` — name of victim/target (vict)
  - `$e` — subjective pronoun of actor (he/she/it)
  - `$E` — subjective pronoun of target
  - `$m` — objective pronoun of actor (him/her/it)
  - `$M` — objective pronoun of target
  - `$s` — possessive pronoun of actor (his/her/its)
  - `$S` — possessive pronoun of target
  - `$p` — name of object (obj)
  - `$P` — name of object's owner (obj → obj->carried_by or obj->worn_by)
  - `$a` — "a" or "an" for object
  - `$A` — "a" or "an" for object's owner
  - `$i` — "it" or "them" for object (singular/plural)
  - `$I` — "it" or "them" for object's owner
  - `$o` — object short desc
  - `$O` — object owner short desc
  - `$t` — arg string
  - `$T` — arg2 string
  - `$r` — arg string capitalized
  - `$R` — arg2 string capitalized
  - `$q` — arg string lowercased
  - `$Q` — arg2 string lowercased
  - `$$` — literal `$`
- `act()` — wrapper that calls perform_act() with proper visibility checks:
  - TO_CHAR, TO_VICT, TO_NOTVICT, TO_ROOM, TO_GHOST
  - SKIP_SLEEPING check (don't send to sleeping chars unless TO_GHOST)
  - Handle invisible targets (hide $N/$p when observer can't see target)
  - Handle null actors/victims

**Acceptance criteria:**
- `act.go` exports `Act(minPos, type int, format string, ch, vict, obj, targetObj *Character, arg, arg2 string)`
- All 22+ substitution tokens work correctly
- Visibility checks match C (sleeping, invisible, ghost)
- `go build ./...` passes
- At least 3 social commands use `Act()` for output (e.g., smile, wave, bow)

**Estimated complexity:** High — this is the most-used function in the entire codebase.

---

## BATCH 2 — Parsers (parallel, after Batch 1)

### Task 2.1: Fix Mob Parser Stat Scaling

**Priority:** P0 — Critical  
**C source:** `src/db.c` — `parse_simple_mob()` lines with level-based stat scaling  
**Go target:** `pkg/parser/mob.go` — `parseMob()`

**What to fix:**
- Level-based stat boosts for mobs level 15+ (C adds random stat bonuses based on level)
- THAC0 → hitroll conversion: C stores THAC0, Go should convert: `hitroll = 20 - thaco`
- AC scaling: C applies `ac = 10 * raw_ac_value`
- Default weight/height if not specified in area file
- Default race/noise if not specified
- Save throws (if present in enhanced mob format)
- Race hate arrays parsing from E-specs
- Auto-lowercase of articles in short_desc ("A", "An", "The" → "a", "an", "the")
- Auto-set `IS_NPC` flag on all parsed mobs

**Also in `interpret_espec()` / enhanced mob parsing:**
- Add missing E-spec keys: `BareHandAttack`, `Str`, `StrAdd`, `Int`, `Wis`, `Dex`, `Con`, `Cha`

**Acceptance criteria:**
- Level 20 mob has higher stats than level 5 mob (verify with test zone)
- THAC0 values from area files convert to hitroll correctly
- AC values scale properly
- `go build ./...` passes

### Task 2.2: Fix Object Parser

**Priority:** P0 — High  
**C source:** `src/db.c` — `parse_object()`  
**Go target:** `pkg/parser/obj.go` — `parseObj()`

**What to fix:**
- `WearFlags` array: change from [3] to [4] to match C's 4-element wear position array
- Add 'S' (Script) line parsing for objects — Go currently skips `Script:` lines
- Add auto-cap of description first letter
- Add `MAX_OBJ_AFFECT` check (C limits affects on objects, Go just appends)
- Add container weight validation
- Fix `asciiflag_conv()` — ensure letter-encoded flags work in all parsers (currently only in obj.go; wld.go uses atoi only, mob.go doesn't parse flag bitmasks)

**Acceptance criteria:**
- Objects with 4 wear positions parse correctly
- Object scripts load and execute
- `go build ./...` passes

### Task 2.3: Fix Room Parser

**Priority:** P0 — High  
**C source:** `src/db.c` — `parse_room()`  
**Go target:** `pkg/parser/wld.go` — `parseRoom()`

**What to fix:**
- Read all 6 numeric fields (currently only reads 3): zone, flags[4], sector
- Parse room_flags as 4-element array matching C format
- Parse 'E' (extra description) blocks: keyword line + multiline description
- Parse 'R' (room script) lines
- Fix exit parsing to handle multi-line descriptions (C reads fread_string for desc, Go reads single lines)

**Acceptance criteria:**
- Room flags load correctly from area files
- Extra descriptions visible via `look` and `examine`
- Room scripts execute
- `go build ./...` passes

### Task 2.4: Fix Zone Reset System

**Priority:** P1  
**C source:** `src/db.c` — `reset_zone()`, `percent_load()`  
**Go target:** `pkg/game/spawner.go`

**What to fix:**
- Implement `percent_load()` — object load probability (0-100%). Currently all objects load at 100%.
- Add 'L' (loop) command parsing and execution
- Add `if_flag` conditional chaining (Go currently ignores IfFlag)
- Fix 'D' command door-state logic
- Fix 'R' command semantics: C uses arg3 as obj_rnum for removal, Go uses arg3 as is_obj flag
- Add `MOB_RANDZON` random zone placement
- Add zone79 random mob placement logic

**Acceptance criteria:**
- Objects with 50% load probability spawn ~half the time
- Zone resets with loop commands work correctly
- `go build ./...` passes

---

## BATCH 3 — Systems (parallel, after Batch 1)

### Task 3.1: Port Follower Management

**Priority:** P1  
**C source:** `src/utils.c` — `add_follower()`, `stop_follower()`, `die_follower()`, `circle_follow()`, `add_follower_quiet()`  
**Go target:** Create `pkg/game/follow.go` (new file) or add to existing file

**What to port:**
- `add_follower(ch, leader)` — add ch as follower of leader
- `add_follower_quiet(ch, leader)` — same but no "N follows you" message
- `stop_follower(ch)` — remove ch from leader's follower list, send message
- `die_follower(ch)` — called when ch dies, clean up follower relations
- `circle_follow(ch, victim)` — detect follow loops before they happen
- `get_mount(ch)` — return ch's mount (if riding)
- `get_rider(ch)` — return character riding ch
- `get_rider_in_room(ch, room)` — find a rider for ch in given room

**Acceptance criteria:**
- `follow <name>` works and creates leader/follower relationship
- `circle_follow()` prevents infinite loops
- Dismount on death works
- `go build ./...` passes

### Task 3.2: Port Mount/Riding System

**Priority:** P1  
**C source:** `src/handler.c` (mount checks), `src/act.movement.c` (ride command, dismount on move)  
**Go target:** Add mount checks to existing handler functions + movement commands

**What to fix:**
- Add mount awareness to: `do_stand()`, `do_sit()`, `do_rest()`, `do_sleep()` (can't change position while mounted)
- Add dismount-on-move: if moving while mounted, move mount + rider together
- Implement `ride` command properly
- Implement `dismount` command
- Mount checks in combat (mounted players get combat bonuses per C source)

**Depends on:** Task 3.1 (follower system)

**Acceptance criteria:**
- Can mount a mob, move while mounted, dismount
- Can't sit/rest/sleep while mounted
- `go build ./...` passes

### Task 3.3: Fix Equipment Edge Cases

**Priority:** P1  
**C source:** `src/handler.c` — `equip_char()`, `unequip_char()`, `apply_ac()`, `affect_total()`, `check_for_bad_stats()`, `aff_apply_modify()`  
**Go target:** `pkg/game/handler.go` or equivalent

**What to fix in `equip_char()`:**
- Add anti-alignment zap: if item has ANTI_EVIL/GOOD/NEUTRAL and char doesn't match, zap the char and refuse equip
- Add invalid class check
- Add light tracking (LIGHT source items affect room visibility)
- Call `check_for_bad_stats()` after equip

**What to fix in `unequip_char()`:**
- Restore ITEM_TAKE_NAME (if item was renamed by take flag)
- Restore AC contribution from unequipped item
- Update light tracking

**What to fix in `apply_ac()`:**
- Add body-part AC multipliers: body ×3, head/legs ×2, all others ×1

**What to fix in `affect_total()`:**
- Add tattoo_af() calls
- Add equipment iteration
- Add stat clamping (stats can't go below 3 or above 25 in C)
- Add alignment clamping

**Port `check_for_bad_stats()`:**
- Entirely missing — implements zero-stat penalties (STR 0 = can't move, DEX 0 = can't dodge, etc.)

**Fix `aff_apply_modify()`:**
- Add missing APPLY types: APPLY_AGE, APPLY_CHAR_WEIGHT, APPLY_CHAR_HEIGHT, APPLY_GOLD, regen rates, APPLY_RACE_HATE, APPLY_SPELL

**Acceptance criteria:**
- Anti-evil/anti-good items zap players of wrong alignment
- AC calculated correctly with body-part multipliers
- Stats clamp to 3–25 range
- Zero-stat penalties apply
- `go build ./...` passes

### Task 3.4: Port Utility Functions

**Priority:** P2  
**C source:** `src/utils.c`  
**Go target:** Various files

**What to port:**
- `sprintbit(bitvector, names[])` → bitvector to readable flag string
- `sprinttype(type, names[])` → type index to name string
- `parse_race(name)` → race name to enum
- `age(ch)` → character age calculation based on play time
- `playing_time(ch)` → formatted play-time string
- `log_death_trap(ch)` → proper death trap logging

**Acceptance criteria:**
- `sprintbit()` produces correct flag strings for room/mob/obj flags
- `parse_race()` handles all race names
- `go build ./...` passes

---

## BATCH 4 — Combat (after Batch 1)

### Task 4.1: Fix Death Mechanics

**Priority:** P1  
**C source:** `src/fight.c` — `die_with_killer()`  
**Go target:** `pkg/game/fight.go` or equivalent

**What to fix:**
- Go always loses 1 constitution on death. C has: level 1–4 = lose 1 con; level 5–9 = lose 1 con (50% chance); level 10+ = lose 1 con (33% chance)
- Add proper death effects from C: exp loss formula, gold drop, corpse creation, equipment drop to corpse
- Add killer tracking for bounty/guard systems

**Acceptance criteria:**
- Low-level deaths always cost 1 con
- High-level deaths sometimes don't cost con
- Corpses contain player's equipment
- `go build ./...` passes

### Task 4.2: Fix Damage() Edge Cases

**Priority:** P1  
**C source:** `src/fight.c` — `damage()`  
**Go target:** `pkg/game/fight.go`

**What to fix:**
- Race hate: C applies race hate up to 5 times per damage call (loops through mob race hate list). Go only applies once.
- Add jail guard logic (guards respond to PK in cities)
- Add charm retarget (charmed mobs retarget when ordered to attack friendly)
- Add NPC target switching (NPCs switch to attackers with higher damage)
- Add autoloot on kill (if char has autoloot flag)
- Add PK outlaw flagging

**Acceptance criteria:**
- Race hate applies correctly (multiple race hate targets)
- NPCs switch targets intelligently
- `go build ./...` passes

### Task 4.3: Implement Group Autogold/Autosplit

**Priority:** P1  
**C source:** `src/fight.c` — `group_gain()` (~100 lines for autogold/autosplit)  
**Go target:** Same function

**What to port:**
- After group XP distribution, if victim dropped gold:
  - If killer has AUTOAUTOGOLD flag: automatically loot gold to inventory
  - If killer has AUTOSPLIT flag: split gold among group members equally
  - Send "You get X gold coins" / "X gold split among Y members" messages

**Acceptance criteria:**
- Gold from kills goes to group members who have split enabled
- Messages match C format
- `go build ./...` passes

### Task 4.4: Fix Combat Messages

**Priority:** P2  
**C source:** `src/fight.c` — `dam_message()`, `load_messages()`  
**Go target:** `pkg/game/fight.go`

**What to fix:**
- Fix `dam_message()` damage tiers to match C thresholds
- Fix bug: damage message uses damage amount for position lookup instead of victim HP
- Port `load_messages()` — combat skill messages (skill_name messages loaded from file, not hardcoded)
- Fix `make_dust()` to distinguish dust vnum 18 vs vampire_dust vnum 1230

**Acceptance criteria:**
- Damage messages match C thresholds (miss, scratch, light, etc.)
- Combat skill messages are accurate
- `go build ./...` passes

### Task 4.5: Implement Parry/Dodge/Mob Wait

**Priority:** P1  
**C source:** `src/fight.c` — `perform_violence()`  
**Go target:** Same function

**What to port:**
- Player parry check (if player has parry skill, chance to negate incoming attack)
- NPC dodge behavior (NPCs can dodge attacks)
- mob_wait state (NPCs pause between actions)

**Acceptance criteria:**
- Players with parry skill occasionally negate attacks
- Combat feels closer to C timing
- `go build ./...` passes

---

## BATCH 5 — Player Commands (parallel, after Batch 1)

### Task 5.1: Port `do_gen_ps()` Informational Commands

**Priority:** P2  
**C source:** `src/act.informative.c` — `do_gen_ps()`  
**Go target:** `pkg/game/commands/` or equivalent

**What to port (12+ commands):**
- `credits` — game credits
- `news` — MOTD news file
- `motd` — message of the day
- `imotd` — immortal MOTD
- `version` — MUD version string
- `wizlist` — list of immortal characters
- `handbook` — immortal handbook
- `policies` — game policies
- `whoami` — player's current character info
- `clear` — clear screen
- `motd` variants

**Acceptance criteria:**
- Each command registered and callable
- `credits` displays game credits
- `version` shows MUD version
- `go build ./...` passes

### Task 5.2: Fix `do_score()` and `do_who()`

**Priority:** P2  
**C source:** `src/act.informative.c`  
**Go target:** Same

**`do_score()` fixes:**
- Add alignment string ("pure evil" through "pure good")
- Add play time display
- Add character age
- Add character weight
- Add detailed AC breakdown

**`do_who()` fixes:**
- Add `-o` flag (show only online imms)
- Add `-k` flag (show only killers)
- Add `-l` flag (show only level range)
- Add class filter argument
- Add level range filter argument

**Register missing commands:**
- `abils` — show character abilities
- `levels` — show level progression
- `coins` — show character coins

**Acceptance criteria:**
- `score` shows alignment, play time, age, weight
- `who -o` shows only imms
- `abils`, `levels`, `coins` registered and working
- `go build ./...` passes

### Task 5.3: Fix `do_steal()` Edge Cases

**Priority:** P2  
**C source:** `src/act.other.c`  
**Go target:** Same

**What to fix:**
- Add ROOM_PEACEFUL check (can't steal in peaceful rooms)
- Add mounted check (harder to steal while mounted)
- Add kender_steal (kender race has enhanced steal)
- Add robbery affect tracking

**Acceptance criteria:**
- Can't steal in peaceful rooms
- `go build ./...` passes

---

## BATCH 6 — Movement (parallel)

### Task 6.1: Fix Follow/Movement Edge Cases

**Priority:** P1  
**C source:** `src/act.movement.c`  
**Go target:** Same

**`do_follow()` fixes:**
- Add `circle_follow()` loop detection before accepting follow
- Add AFF_CHARM restriction (can't follow while charmed by someone else)
- Add shadow quiet-follow mode
- Use `add_follower_quiet()` where appropriate

**`do_wake()` fixes:**
- Add AFF_SLEEP check (can't wake if under sleep spell)
- Add position threshold check

**`do_leave()` fixes:**
- Match C logic: check ROOM_INDOORS, not exit keyword search

**`ok_pick()` fixes:**
- Add lockpick breakage: if pick fails, chance to break → create obj vnum 8028
- Level-based break chance

**Acceptance criteria:**
- Can't follow in circles
- Lockpicks can break
- `go build ./...` passes

---

## BATCH 7 — World Systems

### Task 7.1: Port Help System

**Priority:** P2  
**C source:** `src/db.c` — `load_help()`, help data structures  
**Go target:** Create `pkg/game/help.go` + `data/help/` directory

**What to port:**
- `load_help()` — load help entries from help file at boot
- `do_help()` — search help by keyword, display to player
- Help file format: `#<keyword>` header, body text, `#0` terminator

**Acceptance criteria:**
- `help <topic>` returns help text
- Help loads from file on boot
- `go build ./...` passes

### Task 7.2: Port MUD Time System

**Priority:** P2  
**C source:** `src/db.c` — `reset_time()`, mud time globals  
**Go target:** Add to engine or game package

**What to port:**
- MUD time/day/month/year tracking
- `reset_time()` initialization
- Time-based events (shops closing at night, etc.)
- `write_mud_date_to_file()` for persistence

**Acceptance criteria:**
- In-game time advances correctly
- Time accessible for time-based game logic
- `go build ./...` passes

### Task 7.3: Fix Shop System

**Priority:** P2  
**C source:** `src/shop.c`  
**Go target:** `pkg/game/shop.go` or equivalent

**What to fix:**
- Register repair, identify, and value commands
- Implement shop gold balance (shops have finite gold, deplete when buying from players)
- Shop profit margin logic

**Acceptance criteria:**
- `value <item>` works in shops
- Shop gold depletes when buying items from players
- `go build ./...` passes

---

## BATCH 8 — Networking / Comm

### Task 8.1: Port `color_expansion()`

**Priority:** P2  
**C source:** `src/comm.c` — `color_expansion()`  
**Go target:** `pkg/engine/comm_infra.go` or session package

**What to port:**
- ANSI color code expansion: `\x1B[%dm` → readable color tokens
- Support all C color codes: RED, GREEN, YELLOW, BLUE, MAGENTA, CYAN, WHITE, BLACK, BOLD, BLINK, UNDERLINE, REVERSE

**Acceptance criteria:**
- Color codes render correctly in telnet client
- `go build ./...` passes

### Task 8.2: Port Prompt Helpers

**Priority:** P2  
**C source:** `src/comm.c` — `get_status()`, `target_status()`, `tank_status()`  
**Go target:** `pkg/engine/comm_infra.go`

**What to port:**
- `get_status(ch)` — HP percentage string for prompt
- `target_status(ch)` — target's HP percentage for prompt
- `tank_status(ch)` — group tank's HP percentage for prompt

**Acceptance criteria:**
- Prompt shows correct HP percentages
- `go build ./...` passes

---

## BATCH 9 — Mobact

### Task 9.1: Implement Mobile AI Behaviors

**Priority:** P2  
**C source:** `src/mobact.c` — `mobile_activity()`  
**Go target:** `pkg/game/mobact.go`

**What to port:**
- **Scavenger**: mobs pick up items on the ground (MOB_SCAVENGER flag)
- **Memory**: mobs remember attackers and hunt them (MOB_MEMORY flag)
- **Helper**: mobs assist friendly mobs in same room (MOB_HELPER flag)
- **Aggressive**: mobs attack players on sight, including opening doors (MOB_AGGRESSIVE flag)
- **AGGR24**: aggressive to level 1–24 players only
- **Hunter**: mobs track and hunt remembered players (MOB_HUNTER flag)
- **Stay Zone**: mobs don't leave their home zone (MOB_STAY_ZONE flag)
- **Race hate**: mobs attack specific races on sight

**Acceptance criteria:**
- Aggressive mobs attack players entering room
- Scavenger mobs pick up dropped items
- Helper mobs assist allies
- Memory mobs remember attackers after combat
- `go build ./...` passes

---

## BATCH 10 — Scripts / Lua

### Task 10.1: Flesh Out Lua Library Stubs

**Priority:** P3  
**C source:** `src/scripts.c`  
**Go target:** `pkg/scripts/` or equivalent

**What to fix:**
- Review 20 stubbed lua_ functions
- Implement or explicitly remove (with comment) each stub
- Priority stubs: any used by spec_procs or mobprogs

**Acceptance criteria:**
- No lua_ function is an empty stub if it's referenced by game scripts
- `go build ./...` passes

---

## Execution Order

```
BATCH 1 (keystone): act() engine
    │
    ├── BATCH 2 (parsers): 2.1, 2.2, 2.3, 2.4 — all parallel
    ├── BATCH 3 (systems): 3.1, 3.3, 3.4 — parallel
    ├── BATCH 4 (combat): 4.1, 4.2, 4.3, 4.4, 4.5 — parallel
    ├── BATCH 5 (commands): 5.1, 5.2, 5.3 — parallel
    ├── BATCH 6 (movement): 6.1
    ├── BATCH 7 (world): 7.1, 7.2, 7.3 — parallel
    └── BATCH 8 (comm): 8.1, 8.2 — parallel
    │
BATCH 3.2 (mount) — depends on 3.1
BATCH 9 (mobact) — depends on 3.1 (followers)
BATCH 10 (lua) — last, after all game logic settled
```

**Total: 24 tasks across 10 batches**
**P0: 5 tasks | P1: 10 tasks | P2: 8 tasks | P3: 1 task**
