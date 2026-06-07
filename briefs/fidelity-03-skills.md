# Fidelity Brief 03: Skills System

**Date:** 2026-05-27
**Priority:** HIGH — defines thief/assassin playstyle, affects all melee
**C source:** `src/act.other.c` (sneak/hide/steal lines 214-543), `src/act.offensive.c` (backstab line 165), `src/class.c` (skill tables)
**Go source:** `pkg/combat/skill_messages.go`, `pkg/combat/fight_core.go`, `pkg/session/eat_cmds.go`

---

## Scope

Skills are non-magical abilities — backstab, sneak, hide, steal, pick lock, peek, toggle, dodge, parry, etc. Each has:
- A skill check formula (level + stat + d20)
- Success/failure messages
- Improvement on use (`improve_skill()`)
- Prerequisites (position, equipment, group status)

This brief covers:

1. **Skill check formula** — `compute_skill()` or equivalent
2. **Thief skills** — backstab, sneak, hide, steal, pick lock, peek
3. **Combat skills** — dodge, parry, block, second attack, dual wield
4. **Skill messages** — attacker/victim/room for each skill
5. **Skill improvement** — `improve_skill()` on successful use
6. **Skill tables** — which classes get which skills at what levels

---

## What to Verify

### 1. Skill Check Formula

**C source** (class.c, handler.c):
The skill check is typically:
```
skill_chance = base + level_bonus + stat_bonus + d20
success if skill_chance > difficulty
```

Different skills have different difficulty values. Backstab is easier at low levels, harder at high levels. Sneak check runs every movement tick.

**Check:** Does the Go skill check match? Look for:
- Base chance per skill (from skill table)
- Level bonus formula
- Stat bonus (DEX for thief skills, STR for combat skills)
- Difficulty class (varies per skill)

### 2. Backstab

**C source** (act.offensive.c:165):
```c
ACMD(do_backstab)
{
  // Must be standing
  // Must be behind victim (opposite direction)
  // Must be hidden (AFF_HIDE)
  // Must be wielding a weapon
  // Can't backstab group members
  // Damage: weapon_damage * backstab_multiplier(level)
  // backstab_multiplier: level/10 + 1 (min 1, max ~4)
  // After backstab, position changes
}
```

**Check:**
- Position requirement (standing)
- Direction requirement (must be behind)
- Hidden requirement (AFF_HIDE)
- Weapon requirement (must be wielding)
- Damage multiplier formula
- Message variants ("You backstab $N!", "$n backstabs you!", "$n backstabs $N!")
- Skill improvement call

### 3. Sneak

**C source** (act.other.c:214):
```c
ACMD(do_sneak)
{
  // Position must be standing
  // Roll skill check
  // On success: SET_BIT(AFF_FLAGS(ch), AFF_SNEAK)
  // On failure: "You stumble.\r\n" + SET_BIT(AFF_SNEAK) anyway? No, fail = no sneak
  // Duration: until you stop sneaking or get hit
}
```

**Check:**
- Success/failure messages
- AFF_SNEAK bitvector
- Does sneak break on attack? (should it?)
- Does sneak break on movement failure?

### 4. Hide

**C source** (act.other.c:247):
```c
ACMD(do_hide)
{
  // Position must be standing
  // Roll skill check
  // On success: SET_BIT(AFF_FLAGS(ch), AFF_HIDE)
  // Message: "You attempt to hide.\r\n" (always shown)
  // Hidden until: attack, move, or get attacked
}
```

**Check:**
- AFF_HIDE bitvector
- Does hide break on movement?
- Does hide break on attack?
- Can you hide in combat? (C source: must be standing, not fighting)

### 5. Steal

**C source** (act.other.c:309):
```c
ACMD(do_steal)
{
  // Must be standing
  // Can't steal from group members
  // Can't steal from NPCs with SCRIPT (some protection)
  // Skill check against victim's level
  // On success: transfer object or gold
  // On failure: 10% chance of "You get caught!" + reputation hit
  // Gold steal: percentage of victim's gold
  // Item steal: random item from victim's inventory
}
```

**Check:**
- Gold steal percentage formula
- Item steal random selection
- Caught chance (10%?)
- Caught consequence (agro? reputation? jail?)
- Messages ("You steal $p from $N." / "$n steals $p from you!" / "$n steals something from $N!")

### 6. Pick Lock

**C source** (act.movement.c:536, `ok_pick()`):
```c
int ok_pick(struct char_data *ch, int keynum, int pickproof, int scmd)
{
  // Must be standing
  // Need lock picks or open lock skill
  // Skill check against door difficulty
  // On success: unlock/open the door
  // On failure: "It resists your attempt.\r\n"
  // Pickproof doors: can't be picked at all
}
```

**Check:**
- Lock pick item requirement
- Skill check formula
- Pickproof check
- Messages for success/failure

### 7. Skill Tables

**C source** (class.c, constants.c):
Each class gets skills at specific levels. The skill table looks like:
```
Skill           Thief  Cleric  Warrior  Magic-user  ...
Backstab        1      -       -        -           ...
Hide            1      -       -        -           ...
Sneak           1      -       -        -           ...
Pick Lock       1      -       -        -           ...
Peek            1      -       -        -           ...
...
```

**Check:** Does the Go skill table match? Are the level requirements correct?

---

## Implementation Notes

- Skill data may be loaded from `lib/etc/skills` or `lib/etc/spell_index` (combined with spells)
- `improve_skill()` is called on successful skill use — skill improves faster at lower levels
- The skill percentage is stored on the player struct
- Combat skills (dodge, parry, block) are checked passively during `hit()`

---

## Verification

1. Create a thief, skill up sneak/hide/backstab
2. Test backstab: hidden + behind target + wielding weapon
3. Test sneak: move through rooms, verify mobs don't detect
4. Test hide: hide in room, verify mobs don't attack
5. Test steal: steal gold and items from shopkeeper
6. Test pick lock: pick a locked door
7. Verify skill improvement after successful use
8. Run `go test ./pkg/combat/... ./pkg/game/...`
