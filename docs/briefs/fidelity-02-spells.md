# Fidelity Brief 02: Spells System

**Date:** 2026-05-27
**Priority:** HIGH — 40+ spells, core mechanic for 3 classes
**C source:** `src/spells.c` (1218 lines), `src/spell_parser.c` (1626 lines)
**Go source:** `pkg/spells/` (4500+ lines)

---

## Scope

The spell system defines every spell in the game — damage, healing, buffs, debuffs, summons, teleports. Each spell has specific mana costs, casting times, durations, saving throws, and messages. This brief covers:

1. **Spell info table** — `spell_info[]` array, mana costs, min levels per class
2. **Spell damage** — `mag_damage()` and per-spell damage formulas
3. **Spell effects** — `mag_unaffects()`, `mag_alter_escapes()`, `mag_groups()`, `mag_masses()`, `mag_areas()`
4. **Saving throws** — `mag_savingthrow()` and per-spell save types
5. **Spell messages** — casting messages, damage messages, effect messages
6. **Spell components** — reagent system (if implemented)
7. **Spell naming** — `do_cast()` parsing, abbreviated spell names

---

## What to Verify

### 1. Spell Info Table

**C source** (spell_parser.c:36):
```c
struct spell_info_type spell_info[TOP_SPELL_DEFINE + 1];
```

**C source** (spells.h:306):
```c
struct spell_info_type {
   byte min_position;   /* Position for caster   */
   int mana_min;    /* Min amount of mana used by a spell (highest lev) */
   int mana_max;    /* Max amount of mana used by a spell (lowest lev) */
   int mana_change; /* Change in mana used by spell from lev to lev */
   int min_level[NUM_CLASSES];
   int routines;
   byte violent;
   int targets;         /* See below for use of TAR_XXX  */
};
```

**Check:**
- Is the `spell_info` array populated from a file or hardcoded?
- Does each spell have correct mana_min, mana_max, mana_change?
- Are min_level values correct per class?
- Are the target flags (TAR_CHAR_ROOM, TAR_OBJ_INV, etc.) correct?

### 2. Spell Routines

**C source** (spells.h):
```
#define SPELL_TYPE_SPELL   0
#define SPELL_TYPE_POTION  1
#define SPELL_TYPE_WAND    2
#define SPELL_TYPE_STAFF   3
#define SPELL_TYPE_SCROLL  4
```

Each spell has `routines` — bitfield of effect functions:
- `DAM_MESSAGES` — sends damage messages
- `DAM_NONE` — no damage
- `AFFECTS` — applies affect
- `UNAFFECTS` — removes affect
- `ALTEREscapes` — dispel/summon
- `GROUPS` — group spell
- `MASSES` — mass spell
- `AREAS` — area spell
- `SUMMONS` — summon creature
- `CREATIONS` — create object
- `MANUAL` — custom handling

**Check:** Does each spell's routine bitfield match the C source? This determines which effect function runs.

### 3. Damage Spells

**C source** (spells.c `mag_damage()`):
Each damage spell has:
- Base damage dice (e.g., magic missile: `dice(1, 6)`)
- Level scaling (damage per caster level)
- Saving throw for half damage (if applicable)
- Message to caster, victim, room

**Check:** For each damage spell:
- Do the damage dice match?
- Is the level scaling correct?
- Is the save type correct (SPELL_SAVE_NEGATION vs SPELL_SAVE_HALF)?
- Are the messages identical?

**Spells to verify (highest use):**
| Spell | C source line | Damage | Save |
|-------|---------------|--------|------|
| MAGIC MISSILE | spells.c | dice(1,6) per level | none |
| FIREBALL | spells.c | dice(1,8) per level | half |
| LIGHTNING BOLT | spells.c | dice(1,8) per level | half |
| CHILL TOUCH | spells.c | dice(1,4) per level | none |
| SHOCKING GRASP | spells.c | dice(1,8) per level | none |
| COLOR SPRAY | spells.c | dice(1,4) per level | none |
| ENERGY DRAIN | spells.c | dice(1,8) | none |

### 4. Healing Spells

**C source** (spells.c `mag_heal()`):
- HEAL: `dice(3, 8) + level` (max 30)
- CURE CRITICAL: `dice(3, 8) + level` (max 22)
- CURE SERIOUS: `dice(2, 8) + level` (max 16)
- CURE LIGHT: `dice(1, 8) + level` (max 10)

**Check:** Do these match? Is the cap correct?

### 5. Affect Spells

**C source** (spells.c `mag_unaffects()`, `mag_alter_escapes()`):
Each affect spell has:
- Duration (in ticks or turns)
- Modifier (stat bonus/penalty, AC change, etc.)
- Bitvector (AFF_* flag)
- Saving throw (if any)

**Check:** For each affect spell:
- Is the duration correct?
- Is the modifier value correct?
- Is the AFF_* bitvector correct?
- Is the save type correct?

**Key spells to verify:**
| Spell | Duration | Modifier | Save |
|-------|----------|----------|------|
| HASTE | 12 rounds | +1 attack, +2 dex | negation |
| SLOW | 12 rounds | -1 attack, -2 dex | negation |
| SHIELD | level/2 + 4 | -4 AC | none |
| STONE SKIN | level | -10 AC | none |
| INVISIBILITY | level/2 | AFF_INVISIBLE | negation |
| SANCTUARY | level/4 | white aura | negation |

### 6. Saving Throws

**C source** (spells.c `mag_savingthrow()`):
```c
int mag_savingthrow(struct char_data * ch, int type)
{
  int save = GET_LEVEL(ch);

  // Class bonuses
  if (GET_CLASS(ch) == CLASS_MAGIC_USER) save += 2;
  if (GET_CLASS(ch) == CLASS_CLERIC) save += 1;
  if (GET_CLASS(ch) == CLASS_PALADIN) save += 1;
  if (GET_CLASS(ch) == CLASS_RANGER) save += 1;
  if (GET_CLASS(ch) == CLASS_PSIONIC) save += 3;

  // DEX bonus
  save += (GET_DEX(ch) - 11) / 2;

  // Target bonus (different from caster)
  // ...

  return MIN(save, 95);  // 95% max save chance
}
```

**Check:** Does the Go saving throw formula match exactly? This affects every spell in the game.

### 7. Spell Messages

**C source** (spells.c):
Each spell sends messages to:
- `to_char` — caster sees "$n casts $N at $M!" or "You cast $N at $N!"
- `to_vict` — target sees "$n casts $N at you!"
- `to_room` — room sees "$n casts $N at $N!"

**Check:** Are all three message variants present for each spell? The `$N` in these messages is the spell name (from `skill_name()`).

---

## Implementation Notes

- The spell info table is likely loaded from `lib/etc/spell_index` or `lib/etc/spells` — verify the Go server reads the same file
- `spell_parser.c` handles the `cast` command parsing (abbreviated names, target selection)
- Reagent system (if implemented) would be in `lib/world/` or `lib/etc/reagents`
- Spell messages use `act()` — same `$` codes as combat messages

---

## Verification

1. Cast each damage spell on a mob — verify damage dice and messages
2. Cast healing spell on self — verify healing amount and cap
3. Cast buff spell — verify duration and modifier
4. Test saving throw — cast on mob with high save, verify reduced effect
5. Test abbreviated spell names (e.g., "cas fir" for "cast fireball")
6. Test reagent consumption (if applicable)
7. Run `go test ./pkg/spells/...`
