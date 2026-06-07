# Fidelity Brief 01: Combat System

**Date:** 2026-05-27
**Priority:** CRITICAL — core game loop
**C source:** `src/fight.c` (2033 lines), `src/act.offensive.c` (1510 lines)
**Go source:** `pkg/combat/` (4600+ lines), `pkg/game/death.go` (905 lines), `pkg/game/combat_helpers.go`

---

## Scope

The combat system is the game. Every swing, every spell, every death runs through this code. This brief covers word-for-word fidelity for:

1. **Attack messages** — `attack_hit_text[]` table, `msg_type` die/hit/miss/god messages
2. **`hit()` function** — the main attack loop (fight.c:533)
3. **`damage()` function** — damage application, position updates (fight.c:107)
4. **`raw_kill()` / `die()` / `death_cry()`** — death sequence (fight.c:506-629)
5. **`update_pos()`** — position state machine (fight.c:186)
6. **`stop_fighting()`** — combat cleanup (fight.c:230)
7. **Offensive commands** — `do_backstab`, `do_kick`, `do_bash`, `do_flee`, `do_retreat`, `do_disarm`, `do_berserk`, `do_charge`
8. **Combat messages** — attacker/victim/room message variants

---

## What to Verify

### 1. Attack Hit Text Table

**C source** (fight.c:84):
```c
struct attack_hit_type attack_hit_text[] =
{
   {"hit", "hits"},       /* 0  */
   {"sting", "stings"},   /* 1  */
   {"whip", "whips"},     /* 2  */
   {"slash", "slashes"},  /* 3  */
   {"bite", "bites"},     /* 4  */
   {"bludgeon", "bludgeons"}, /* 5 */
   {"crush", "crushes"},  /* 6  */
   {"pound", "pounds"},   /* 7  */
   {"claw", "claws"},     /* 8  */
   {"maul", "mauls"},     /* 9  */
   {"thrash", "thrashes"}, /* 10 */
   {"pierce", "pierces"}, /* 11 */
   {"blast", "blasts"},   /* 12 */
   {"punch", "punches"},  /* 13 */
   {"stab", "stabs"}      /* 14 */
};
```

**Check:** Does Go have this exact table? Are all 15 entries present with correct singular/plural forms?

### 2. Damage Calculation

**C source** (fight.c:107, `damage()`):
- Base damage = `MAX(1, dice(num, size))` + str_dam
- Special damage types (backstab multiplier, spec proc damage)
- Armor reduction
- Critical hit check

**Check:** Does the Go damage formula match the C formula exactly? Look for:
- Dice rolling order (num × size + modifiers)
- Strength bonus application
- Armor class reduction formula
- Minimum damage floor (always ≥ 1)

### 3. Hit Function Flow

**C source** (fight.c:533, `hit()`):
1. Check if attacker can attack (position, not fighting, not dead)
2. Check if target is valid (not dead, not same group)
3. Determine weapon type
4. Calculate attack roll (die roll + str + skill)
5. Calculate defense roll (dex + dodge skill + parry)
6. Compare rolls → hit or miss
7. Apply damage or show miss message
8. Update positions

**Check:** Does the Go `hit()` follow this exact flow? Are there any skipped steps or reordered logic?

### 4. Death Sequence

**C source** (fight.c:534-629):
```
raw_kill(ch, attacktype):
  1. Stop combat for all fighting ch
  2. Remove AFF_GROUP if set
  3. Call death_cry(ch)
  4. Check for corpse creation
  5. Extract character (if NPC) or move to death room (if PC)
  6. Check for death triggers (scripts)

die(ch):
  1. Call raw_kill(ch, TYPE_UNDEFINED)

die_with_killer(ch, killer, attacktype):
  1. Check for death scripts (MS_DEATH flag)
  2. If script exists, run it
  3. Otherwise, raw_kill(ch, attacktype)
```

**Check:** Does the Go death sequence match? Key things:
- `death_cry()` messages (fight.c:506-532) — 5 message variants with room/zone broadcasting
- Corpse creation with correct contents
- Player vs NPC death handling
- Death room assignment for PCs

### 5. Position Updates

**C source** (fight.c:186-228):
```c
void update_pos(struct char_data * victim)
{
  if ((GET_HIT(victim) >= 1) && (GET_POS(victim) > POS_STUNNED))
    return;
  else if ((GET_HIT(victim) >= 1) && (GET_POS(victim) == POS_STUNNED))
    GET_POS(victim) = POS_STANDING;
  else if (GET_HIT(victim) == -1)
    GET_POS(victim) = POS_STUNNED;
  else if (GET_HIT(victim) < -1)
    GET_POS(victim) = POS_DEAD;
  else if (GET_HIT(victim) <= -6)
    GET_POS(victim) = POS_DEAD;
  ...
}
```

**Check:** Does Go match these exact thresholds? The positions are:
- POS_DEAD (≤ -6 or < -1 depending on context)
- POS_MORTALLYW (-1)
- POS_STUNNED (0)
- POS_SLEEPING
- POS_RESTING
- POS_SITTING
- POS_FIGHTING
- POS_STANDING

### 6. Message Variants

**C source** (fight.c:1050-1060):
```c
act(msg->die_msg.attacker_msg, ...)
act(msg->die_msg.victim_msg, ...)
act(msg->die_msg.room_msg, ...)
```

Same pattern for hit_msg, miss_msg, god_msg. Each message type has attacker/victim/room variants.

**Check:** Does the Go code send all three variants for each message type? Are the messages loaded from `lib/world/text/fight.messages` or hardcoded?

### 7. Offensive Commands

**C source** (act.offensive.c):

| Command | Line | Key behavior |
|---------|------|--------------|
| `do_backstab` | 165 | Must be hidden, behind target, wielding weapon. 2x damage. |
| `do_kick` | 587 | Level-based damage, dex bonus |
| `do_bash` | 419 | Strength check, knockdown chance |
| `do_flee` | 360 | Random exit, -1 exp per flee |
| `do_retreat` | 1001 | Directional flee |
| `do_disarm` | (spec_procs) | Weapon check, skill check |
| `do_berserk` | (spec_procs) | HP sacrifice, damage boost |
| `do_charge` | (spec_procs) | Mounted combat |

**Check:** For each command:
- Does the Go version match the C damage formula?
- Are the success/failure messages identical?
- Are the position requirements correct?
- Is the skill improvement call (`improve_skill()`) present?

---

## Implementation Notes

- Fight messages may be loaded from `lib/world/text/fight.messages` — check if the Go server reads this file or has messages hardcoded
- The `act()` function is the message system — it parses `$n`, `$N`, `$s`, `$S`, `$q`, `$Q` etc. Check that Go's equivalent handles all `$` codes
- Damage types (TYPE_HIT through TYPE_SUFFERING) must map correctly
- Weapon damage dice are stored on the weapon object — verify the Go code reads them correctly

---

## Verification

1. Create a character, find a mob, fight it
2. Verify attack messages match C format ("$n hits $N!", etc.)
3. Die intentionally — verify death sequence (death_cry, corpse, respawn)
4. Test backstab as thief (must be hidden, behind target)
5. Test flee — verify -1 exp penalty
6. Test bash — verify knockdown chance
7. Run `go test ./pkg/combat/... ./pkg/game/...`
