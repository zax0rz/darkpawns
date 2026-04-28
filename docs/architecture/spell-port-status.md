# Spell Port Scope — Manual Spells from spells.c

**Generated:** 2026-04-26  
**C source:** `src/spells.c` lines 87–1219  
**Go framework:** `pkg/spells/` (8 files, ~1,645 lines)

---

## Existing Go Spell Infrastructure

| Component | File | Status |
|-----------|------|--------|
| `CallMagic` — main dispatch | `call_magic.go` | ✅ Routes: damage, affects, points, unaff, groups, masses, areas, creations, summons, manual |
| `MagAffects` — 20+ affect spells | `affect_spells.go` | ✅ bless, curse, sanctuary, blind, sleep, poison, haste, slow, fly, detect magic/inviz/infravision, water breathe |
| `MagDamage` — damage formulas | `damage_spells.go` | ✅ With saves, attack modifiers |
| `MagPoints` — healing | `affect_spells.go` | ✅ cure light/critic/heal/vitality |
| `MagUnaffects` — removal | `affect_spells.go` | ✅ remove curse/poison/blind |
| `MagGroups` / `MagMasses` / `MagAreas` | `affect_spells.go` | ✅ Framework present |
| `MagCreations` / `MagSummons` | `affect_spells.go` | ✅ Framework present |
| `CheckSavingThrow` / `GetSavingThrow` | `saving_throws.go` | ✅ Full 6×21×5 table |
| `SaySpell` — syllable substitution | `say_spell.go` | ✅ |
| `GetSpellInfo` / `SetSpellInfo` | `spell_info.go` | ✅ Spell template system |
| `Cast()` | `spells.go` | ⚠️ Partial — dispatches but manual spells fall through to stub |

---

## Blocking Go Dependencies (shared across multiple spells)

| Dependency | Needed By | Status |
|------------|-----------|--------|
| Object type/val/extra_flags/affect slots read/write | create_water, enchant_weapon, enchant_armor, identify, silken_missile, coc | ⚠️ Partial — ObjectInstance has Prototype but no direct val mutation |
| `char_from_room` / `char_to_room` (room transfer) | recall, teleport, summon, mindsight | ✅ Player has RoomVNum; needs proper world room transfer |
| `look_at_room` / room display | recall, teleport, summon, mindsight | ⚠️ Session-level, not available from spell layer |
| `world[].people` iteration (same-room characters) | hellfire, meteor_swarm, mindsight | ❌ No global character list iterator |
| `object_list` iteration (global object search) | locate_object | ❌ No global object list |
| `add_follower` / `stop_follower` | charm, conjure_elemental, divine_int | ❌ Follower system not ported |
| `damage()` function call | summon, charm, identify, hellfire, meteor_swarm | ✅ `combat.TakeDamage` via hooks |
| `create_mobile()` — mob creation from VNUM | conjure_elemental, mirror_image, divine_int | ❌ No runtime mob creation from proto |
| `extract_obj()` — destroy object instance | silken_missile, conjure_elemental | ⚠️ Exists in inventory system? |
| `read_object()` — create object from VNUM | silken_missile, coc | ❌ No runtime object creation from proto |
| `send_to_zone()` — message to same zone | hellfire, meteor_swarm | ❌ No zone messaging |
| `are_grouped()` — group check | hellfire, meteor_swarm | ⚠️ Player has InGroup but no cross-check |
| `circle_check()` — glyph check | summon | ❌ Not ported |
| Mount system (`unmount`, `get_mount`, `get_rider`, `IS_MOUNTED`) | recall, summon | ❌ Not ported |
| `HUNTING()` / `set_hunting()` — mob hunting AI | mental_lapse, mirror_image, mindsight | ❌ Not ported |
| `MOB_MEMORY` / `remember` / `forget` | mirror_image | ❌ Not ported |
| `weather_info` / weather system | control_weather | ⚠️ `pkg/game/weather.go` exists (Wave 13) |
| PLR flags (PLR_WEREWOLF, PLR_VAMPIRE) | lycanthropy, vampirism | ❌ No PLR_FLAGS on Player |
| MOB flags (MOB_NOSUMMON, MOB_NOCHARM, MOB_AGGRESSIVE, etc.) | summon, charm | ❌ No MOB_FLAGS on NPCs |
| `act()` — formatted action messages | Almost all | ⚠️ Partial — some act() equivalents exist |
| PRF flags (PRF_SUMMONABLE) | summon | ❌ Not ported |
| `circle_follow()` — follow loop check | charm | ❌ Not ported |
| OUTSIDE() check — room outdoors flag | meteor_swarm | ⚠️ Room flags exist in parser |
| Room flags (ROOM_PEACEFUL, ROOM_NOMAGIC, ROOM_BFR, ROOM_PRIVATE) | recall, teleport, summon, hellfire, meteor_swarm | ⚠️ Partial — flags defined but not checked by spell layer |

---

## Spell-by-Spell Breakdown

### spell_create_water (lines 87–120)
**Origin:** CircleMUD standard  
**What:** Fills a drink container with water. If container has non-water liquid, poisons it.  
**Dependencies:** Object type check (ITEM_DRINKCON), obj val read/write, `name_from_drinkcon`/`name_to_drinkcon`, `weight_change_object`  
**Complexity:** LOW  
**Blocking:** Object val mutation, drink container name helpers  
**Framework fit:** Custom (manual spell — object manipulation)

### spell_recall (lines 124–165)
**Origin:** CircleMUD standard + DP hometowns (Kiroshi, Alaozar)  
**What:** Teleports victim to their hometown. Can't use while fighting or in BFR rooms. Unmounts.  
**Dependencies:** Room transfer, hometown lookup, mount system, `look_at_room`, ROOM_BFR flag  
**Complexity:** MEDIUM  
**Blocking:** Room transfer, mount system, hometown extern vars  
**Framework fit:** Custom (movement spell)

### spell_teleport (lines 168–217)
**Origin:** CircleMUD standard  
**What:** Random room teleport. Self-only for PCs. NPCs get saving throw. Can't use in peaceful rooms.  
**Dependencies:** Room transfer, `top_of_world`, ROOM_PRIVATE flag check, saving throw, `look_at_room`  
**Complexity:** MEDIUM  
**Blocking:** Room transfer, world room count, room flag checks  
**Framework fit:** Custom (movement spell)

### spell_summon (lines 220–355)
**Origin:** CircleMUD standard + heavy DP modifications  
**What:** Summons victim to caster's room. Complex: PvP restrictions, outlaw mechanics, backfire chance, mount handling, room exit checks, dump room check.  
**Dependencies:** Circle glyph check, PRF_SUMMONABLE, MOB_NOSUMMON, MOB_NOCHARM, room flags, mount system, saving throw, `character_list` proximity, damage(), room exit iteration  
**Complexity:** HIGH  
**Blocking:** Follower system, PRF flags, MOB flags, mount system, circle_check, ARE_GROUPED  
**Framework fit:** Custom (summons framework exists but this is entirely custom logic)

### spell_locate_object (lines 355–407)
**Origin:** CircleMUD standard  
**What:** Searches global object_list for named object. Reports location (carried by, in room, in container, worn by). Max level/2 results. ITEM_NOLOCATE blocks.  
**Dependencies:** Global object_list iteration, object location tracking (carried_by, in_room, in_obj, worn_by)  
**Complexity:** MEDIUM  
**Blocking:** Global object list — requires world-state architecture decision  
**Framework fit:** Custom (information spell)

### spell_charm (lines 407–476)
**Origin:** CircleMUD standard + DP outlaw/PvP rules  
**What:** Charms a mob/PC to follow caster. Checks: sanctuary, nocharm, self-charm, already charmed, level check, shopkeeper immunity, follow circles, saving throw, follower count (CHA/2), outlaw requirement for PvP.  
**Dependencies:** add_follower, affect_to_char (AFF_CHARM), MOB flags, PLR_OUTLAW flag, circle_follow, num_followers, saving throw, damage()  
**Complexity:** HIGH  
**Blocking:** Follower system, MOB flags, PLR flags, circle_follow  
**Framework fit:** Uses AffectToChar (AFF_CHARM) — partially framework-compatible

### spell_identify (lines 476–621)
**Origin:** CircleMUD standard  
**What:** Reveals item stats (type, affects, extra flags, weight, value, spell contents, weapon damage, armor AC) or PC stats (name, age, height, weight, level, hp, mana, AC, hitroll, damroll, all stats).  
**Dependencies:** Object type/val/extra_flags/affects read, sprinttype/sprintbitarray formatting, item_types/extra_bits/apply_types/affected_bits string arrays, age() function, PC stat reads  
**Complexity:** MEDIUM  
**Blocking:** Display formatting helpers, age function  
**Framework fit:** Custom (information spell — pure display)

### spell_enchant_weapon (lines 621–662)
**Origin:** CircleMUD standard  
**What:** Adds +hitroll/+damroll to a non-magic weapon. Sets ITEM_MAGIC flag. Alignment-colored glow. Level 18+ gets +2 hitroll, 20+ gets +2 damroll.  
**Dependencies:** Object type/extra_flags/affect slots read/write  
**Complexity:** MEDIUM  
**Blocking:** Object affect slot mutation, ITEM_MAGIC/ITEM_ANTI_EVIL/ITEM_ANTI_GOOD flag setting  
**Framework fit:** Custom (object manipulation)

### spell_lycanthropy (lines 662–700)
**Origin:** DP original  
**What:** Sets PLR_WEREWOLF flag on target. Checks for existing werewolf/vampire flags.  
**Dependencies:** PLR_WEREWOLF, PLR_VAMPIRE flags  
**Complexity:** LOW  
**Blocking:** PLR flags on Player  
**Framework fit:** Custom (flag toggle)

### spell_sobriety (lines 700–715)
**Origin:** DP original  
**What:** Sets drunk condition to 0, updates position.  
**Dependencies:** GET_COND, update_pos  
**Complexity:** LOW  
**Blocking:** None — conditions exist on Player  
**Framework fit:** Custom (condition modifier) — **can implement immediately**

### spell_hellfire (lines 701–766)
**Origin:** DP original  
**What:** Room-wide AoE damage. Hits all non-grouped characters in room. High damage (12d5+2*level-10). Low-level targets (<=4) get instakill (12x max HP). DEX check to knock prone (POS_SITTING). Peaceful room blocks. Zone-wide message.  
**Dependencies:** world[].people iteration, are_grouped, ROOM_PEACEFUL, send_to_zone, damage(), GET_DEX, position setting, `act()`  
**Complexity:** HIGH  
**Blocking:** Room character iteration, zone messaging, are_grouped  
**Framework fit:** Custom AoE (MagAreas framework exists but this has unique mechanics)

### spell_vampirism (lines 766–794)
**Origin:** DP original  
**What:** Sets PLR_VAMPIRE flag. PCs only. Checks existing werewolf/vampire.  
**Dependencies:** PLR_VAMPIRE, PLR_WEREWOLF flags  
**Complexity:** LOW  
**Blocking:** PLR flags on Player  
**Framework fit:** Custom (flag toggle)

### spell_detect_poison (lines 794–836)
**Origin:** CircleMUD standard  
**What:** Checks victim for AFF_POISON or object for poison flag (ITEM_FOOD/DRINKCON/FOUNTAIN val[3]).  
**Dependencies:** AFF_POISON check, object type/val check, `act()`  
**Complexity:** LOW  
**Blocking:** None — affect flags and object vals accessible  
**Framework fit:** Custom (information spell) — **can implement immediately**

### spell_enchant_armor (lines 836–871)
**Origin:** CircleMUD standard  
**What:** Adds AC improvement to non-magic armor/worn items. Level-based bonus: -(level-20)/2. Alignment-colored glow.  
**Dependencies:** Object type/extra_flags/affect slots read/write  
**Complexity:** MEDIUM  
**Blocking:** Object affect slot mutation, flag setting  
**Framework fit:** Custom (object manipulation)

### spell_zen (lines 871–883)
**Origin:** DP original  
**What:** Heals 2*level HP, sets position to stunned (meditating).  
**Dependencies:** GET_POS, GET_HIT, GET_MAX_HIT, GET_LEVEL  
**Complexity:** LOW  
**Blocking:** None  
**Note:** C bug — uses `victim` instead of `ch` for healing, but sets `ch` position. Likely intended to heal caster.  
**Framework fit:** Custom — **can implement immediately**

### spell_silken_missile (lines 883–912)
**Origin:** DP original  
**What:** Converts armor/clothing into a missile arrow object (VNUM 3). Destroys source item.  
**Dependencies:** read_object (VNUM→instance), extract_obj, obj_to_char, ITEM_WORN/ITEM_ARMOR type check  
**Complexity:** LOW  
**Blocking:** Runtime object creation from VNUM, object extraction  
**Framework fit:** Custom (object creation)

### spell_mindsight (lines 912–960)
**Origin:** DP original  
**What:** Psionic remote viewing. Temporarily moves caster to victim's room, runs `do_look`, shows "being watched" message to psionic/mystic PCs there, then returns. Level+4 resistance check. Blocked by AFF_NOTRACK.  
**Dependencies:** Room transfer (temp), do_look command, world[].people iteration, IS_PSIONIC/IS_MYSTIC checks, AFF_NOTRACK  
**Complexity:** HIGH  
**Blocking:** Room transfer, command execution from spell layer, class checks  
**Framework fit:** Custom (psionic ability)

### spell_mental_lapse (lines 960–983)
**Origin:** DP original  
**What:** Clears mob's HUNTING target. Only works if mob is hunting the caster. Level 30+ can redirect mobs hunting others.  
**Dependencies:** HUNTING(), set_hunting()  
**Complexity:** LOW  
**Blocking:** Mob hunting system  
**Framework fit:** Custom (psionic ability)

### spell_calliope (lines 983–997)
**Origin:** DP original  
**What:** Fires MAX(4, rand(level/6, level*2)) magic missiles at single target via repeated `call_magic()`.  
**Dependencies:** call_magic (SPELL_MAGIC_MISSILE)  
**Complexity:** LOW  
**Blocking:** None — call_magic exists  
**Framework fit:** **Framework-compatible** — just calls CallMagic in a loop — **can implement immediately**

### spell_control_weather (lines 997–1012)
**Origin:** CircleMUD standard  
**What:** Adjusts weather change variable: "better" adds dice(level/3, 4), "worse" subtracts.  
**Dependencies:** weather_info.change, dice()  
**Complexity:** LOW  
**Blocking:** Weather system mutable state  
**Framework fit:** Custom (weather manipulation)

### spell_coc (lines 1012–1039)
**Origin:** DP original  
**What:** Creates a "circle of summoning" object (COC_VNUM) in room with timer = level/2.  
**Dependencies:** read_object (VNUM→instance), obj_to_room, GET_OBJ_TIMER  
**Complexity:** LOW  
**Blocking:** Runtime object creation from VNUM, object timer  
**Framework fit:** Custom (object creation)

### spell_conjure_elemental (lines 1039–1088)
**Origin:** CircleMUD standard + DP elementals  
**What:** Scans room for elemental component objects (earth/water/wind/fire VNUMs), consumes one, creates a charmed elemental mob via `create_mobile()`.  
**Dependencies:** Room contents iteration, read_object, create_mobile (level-override), affect_to_char (AFF_CHARM), add_follower_quiet, extract_obj  
**Complexity:** HIGH  
**Blocking:** Room contents iteration, runtime mob creation, follower system  
**Note:** C bug — `comp` used uninitialized before first assignment on line 1055 (null deref if no components in room).  

### spell_meteor_swarm (lines 1088–1132)
**Origin:** DP original  
**What:** Outdoor-only room AoE. Damage = level*6 + rand(-10, level*3). Hits all non-grouped, non-immortal chars in room.  
**Dependencies:** OUTSIDE() check, world[].people iteration, are_grouped, ROOM_PEACEFUL, damage()  
**Complexity:** MEDIUM  
**Blocking:** Room character iteration, are_grouped, OUTSIDE check  
**Framework fit:** Custom AoE

### spell_mirror_image (lines 1132–1170)
**Origin:** DP original  
**What:** Creates a clone mob (VNUM 69) with caster's name, title, description, sex. Redirects mob memory and hunting from caster to clone.  
**Dependencies:** create_mobile (VNUM→mob), SET_NAME, str_dup, MOB_MEMORY, remember/forget, HUNTING, set_hunting  
**Complexity:** HIGH  
**Blocking:** Runtime mob creation, mob memory system, hunting system  
**Framework fit:** Custom (illusion)

### spell_divine_int (lines 1170–1219)
**Origin:** DP original  
**What:** Summons 1-2 angel mobs (good=VNUM 85, evil=VNUM 86) based on alignment. Neutral blocked. Extreme alignment (-1000/+1000) gets 2 angels. Saving throw check.  
**Dependencies:** create_mobile (level-override), affect_to_char (AFF_CHARM), add_follower_quiet, alignment checks  
**Complexity:** HIGH  
**Blocking:** Runtime mob creation, follower system  
**Note:** C bug — stray semicolons after #define GOOD_ANGEL and EVIL_ANGEL make them expand to empty in some contexts.

---

## Summary

| Complexity | Count | Spells |
|-----------|-------|--------|
| LOW | 9 | sobriety, zen, lycanthropy, vampirism, detect_poison, mental_lapse, calliope, coc, control_weather |
| MEDIUM | 8 | create_water, recall, teleport, enchant_weapon, enchant_armor, silken_missile, identify, meteor_swarm |
| HIGH | 7 | charm, summon, locate_object, hellfire, mindsight, conjure_elemental, mirror_image, divine_int |

**Total C lines to port:** ~1,133 (lines 87–1219)

---

## Recommended Porting Waves

### Wave A — Immediate (no blocking deps)
Can implement right now, existing infrastructure sufficient.
- `spell_sobriety` — condition reset
- `spell_zen` — heal + position (fix C bug: use `ch` not `victim`)
- `spell_detect_poison` — affect/object flag check
- `spell_calliope` — loop CallMagic
- `spell_lycanthropy` — PLR_WEREWOLF flag (add PLR flags first)
- `spell_vampirism` — PLR_VAMPIRE flag

### Wave B — Object Manipulation (needs object val/flag mutation)
- `spell_enchant_weapon` — add affects to weapon
- `spell_enchant_armor` — add affects to armor
- `spell_create_water` — fill drink container
- `spell_silken_missile` — item conversion (needs read_object)
- `spell_identify` — pure display (needs format helpers)

### Wave C — Movement/Teleport (needs room transfer)
- `spell_recall` — hometown teleport (needs mount system)
- `spell_teleport` — random room teleport

### Wave D — Follower/Charm (needs follower system)
- `spell_charm` — charm + follow
- `spell_conjure_elemental` — elemental creation + charm (fix C bug)
- `spell_divine_int` — angel summoning + charm (fix C bug)

### Wave E — AoE Damage (needs room character iteration)
- `spell_hellfire` — room-wide fire (knockdown mechanic)
- `spell_meteor_swarm` — outdoor AoE

### Wave F — Information/Scrying (needs global state)
- `spell_locate_object` — global object search (architectural decision needed)
- `spell_mindsight` — remote viewing (needs command execution from spell layer)

### Wave G — Complex Summon (needs multiple systems)
- `spell_summon` — most complex spell in the game (PvP rules, mount, backfire, outlaw, circle check)

### Wave H — Illusion/Hunting (needs mob memory + hunting)
- `spell_mirror_image` — clone + memory redirect
- `spell_mental_lapse` — clear hunting target
- `spell_control_weather` — weather mutation

---

## C Bugs to Fix During Port

1. **spell_zen (line 882):** Uses `victim` for healing but `ch` for position. Should heal `ch`.
2. **spell_conjure_elemental (line 1055):** `comp` used before initialization — null deref if room has no component objects.
3. **spell_divine_int (lines 1133-1134):** Stray semicolons after `#define GOOD_ANGEL 85;` and `#define EVIL_ANGEL 86;` — macro expansion bug.
4. **spell_sobriety (line 709):** `assert(victim)` — crashes on null instead of graceful failure.

---

## Key Constants

| Constant | Value | Source |
|----------|-------|--------|
| COC_VNUM | (from defines) | spells.c |
| MOB_CLONE | 69 | spells.c:1133 |
| GOOD_ANGEL | 85 | spells.c:1171 |
| EVIL_ANGEL | 86 | spells.c:1172 |
| MISSILE (silken_missile output) | 3 | spells.c:883 |
| EARTH_ELEMENTAL | 81 | spells.c:1041 |
| WATER_ELEMENTAL | 82 | spells.c:1042 |
| WIND_ELEMENTAL | 83 | spells.c:1043 |
| FIRE_ELEMENTAL | 84 | spells.c:1044 |
| LEVEL_IMMORT | (from structs.h) | — |
| LVL_IMPL | (from structs.h) | — |
| NUM_OF_DIRS | 6 | — |
| ROOM_BFR | (room flag) | — |
| ROOM_PEACEFUL | (room flag) | — |
| ROOM_NOMAGIC | (room flag) | — |
| ROOM_PRIVATE | (room flag) | — |
| PLR_WEREWOLF | (player flag) | — |
| PLR_VAMPIRE | (player flag) | — |
| MOB_NOSUMMON | (mob flag) | — |
| MOB_NOCHARM | (mob flag) | — |
| MOB_AGGRESSIVE | (mob flag) | — |
| MOB_SPEC | (mob flag) | — |
| MOB_MEMORY | (mob flag) | — |
| PRF_SUMMONABLE | (pref flag) | — |
| AFF_CHARM | (affect flag) | — |
| AFF_POISON | (affect flag) | — |
| AFF_NOTRACK | (affect flag) | — |
| AFF_SANCTUARY | (affect flag) | — |
| IS_PSIONIC / IS_MYSTIC | (class checks) | — |
| LIQ_WATER / LIQ_SLIME | (liquid types) | — |
