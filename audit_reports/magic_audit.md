# Audit Report: magic.c & spells.c vs pkg/spells/

**C files:** `src/magic.c` (2,000 lines), `src/spells.c` (1,219 lines)
**Go file(s):** `pkg/spells/spells.go` (231 lines), `pkg/spells/spell_info.go` (203 lines), `pkg/spells/call_magic.go` (206 lines), `pkg/spells/saving_throws.go` (229 lines), `pkg/spells/damage_spells.go` (369 lines), `pkg/spells/affect_spells.go` (2,743 lines)
**Mapping type:** N:N
**Functions audited:** ~60

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Sleep Spell Severe Logic Mismatch / Simplification
- **Location:** `pkg/spells/affect_spells.go:74` in `case SpellSleep`.
- **C behavior:** Casting Sleep is a multi-gated, highly restrictive combat and gameplay balance check:
  - **Reagent Verification:** Gated by `has_reagents(ch, SPELL_SLEEP)` ("bit of sand") which awards a duration bonus (`+ reag`).
  - **Outlaw Protection:** Players cannot sleep other players unless the caster is marked as an Outlaw (`!IS_NPC(victim) && !IS_OUTLAW(ch)`).
  - **Level Restrictions:** PvP sleep fails if the target's level is more than 3 levels above or below the caster's level.
  - **Immunity Flags:** Failing the save does not affect targets flagged with `MOB_NOSLEEP`.
  - **Position Mutation:** A successful sleep sets the victim's position directly to `POS_SLEEPING`.
  - **Aggro/Retaliation:** A successful saving throw forces NPC targets to attack the caster.
- **Go behavior:** The Go implementation of `SpellSleep` only does a saving throw roll and then applies the `AFFSleep` affect. None of the reagent checks, outlaw protection, level difference gates, sleep immunity checks, target position mutations, or NPC retaliations are implemented.
- **Discrepancy:** Players can freeze any player (regardless of level gap or outlaw status) indefinitely without sand reagents. Additionally, the victim's position is never set to sleeping, meaning they remain standing physically while carrying the sleep flag. This is a severe disruption of combat and player interaction balance.
- **Severity:** HIGH
- **Type:** STUB / DRIFT

### [FINDING-002]: Poison Spell Missing Strength & Hitroll Penalties
- **Location:** `pkg/spells/affect_spells.go:104` in `case SpellPoison`.
- **C behavior:** A successful Poison spell applies *two* distinct affects to the victim:
  - **Affect 0:** location `APPLY_STR` with modifier `-2` (decreases Strength).
  - **Affect 1:** location `APPLY_HITROLL` with modifier `-2` (decreases Hitroll).
  - Both affects carry the `AFF_POISON` flag and run for `(level/2)-2` duration.
- **Go behavior:** Go's `SpellPoison` applies a single affect (`engine.NewAffectDirect`) with location `engine.ApplyNone` and modifier `-2`.
- **Discrepancy:** Poisoned targets in Go do not suffer any Strength or Hitroll penalties. The spell only carries the cosmetic/timer flag without the designed mechanical combat debuffs.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-003]: Curse Spell Damroll Penalty constructed but Discarded Bug
- **Location:** `pkg/spells/affect_spells.go:60-69` in `case SpellCurse`.
- **C behavior:** In `magic.c:967`, the Curse spell applies *two* active modifiers:
  - **Affect 0:** location `APPLY_HITROLL` with modifier `-(3 + reag)`.
  - **Affect 1:** location `APPLY_DAMROLL` with modifier `-(3 + reag)`.
- **Go behavior:** Go constructs and applies the first affect as `ApplyNone` (incorrect location, should be `ApplyHitroll`) on line 67. It then constructs the second affect:
  `aff = engine.NewAffect(SpellCurse, engine.ApplyDamroll, curseDur, -3, "curse")`
  However, it **never** invokes `applyAffect(victim, aff)` on it! The case block ends immediately and falls through to `case SpellInvisible:` on line 70.
- **Discrepancy:** The Damroll penalty of the Curse spell is entirely discarded and never applied. The Hitroll penalty is incorrectly mapped as `ApplyNone`, completely stripping the Curse spell of its mechanical impact. Reagents are also ignored.
- **Severity:** HIGH
- **Type:** STUB / DRIFT

### [FINDING-004]: Hellfire Area Spell is a Dead Dummy
- **Location:** `pkg/spells/affect_spells.go:1583` under `mag_areas()`.
- **C behavior:** `spell_hellfire` (defined in `spells.c:701`) is a high-level area damage spell that opens the "bowels of hell" beneath victims' feet, carrying a chance to knock targets to their knees (`POS_SITTING`), and dealing an instant kill (`GET_MAX_HIT * 12`) to targets under level 5.
- **Go behavior:** Go's `mag_areas()` case simply returns:
  `case SpellHellfire: return;`
- **Discrepancy:** The Hellfire area spell is a completely non-functional dead dummy stub in Go, doing absolutely nothing when invoked.
- **Severity:** HIGH
- **Type:** STUB

---

## Type & Boundary Vulnerabilities

### [FINDING-005]: Uncapped Disintegrate / Disrupt Self-Harm Backfire
- **Location:** `pkg/spells/damage_spells.go:74` and `85`.
- **C behavior:** In `magic.c:703` and `715`, `!number(0,50)` (1-in-51 chance) triggers a spell backfire, routing the damage to the caster (`victim = ch`).
- **Go behavior:** Go implements this as `!randBool(51)` (1-in-51 chance).
- **Risk:** Type boundary nil pointer panic. If the caster is a NPC or doesn't have a backing session, and `victim = ch` triggers, the subsequent `inflictDamage` call might crash or panic if intermediate fields are nil or if `ch` does not support specific player-only assertions.
- **Severity:** LOW

---

## Concurrency & Mutex Safety

### [FINDING-006]: Unprotected Player Object Mutations in Manual Spells
- **Location:** `pkg/spells/affect_spells.go` in `castSobriety()`, `castZen()`, etc.
- **C behavior:** Synchronous main MUD loop; completely thread-safe.
- **Go behavior:** Go's manual casts directly modify player fields (e.g. `GET_COND(victim, DRUNK) = 0`, `SetHP()`) without acquiring `ch.mu` or `victim.mu`.
- **Impact:** Potential data race or memory corruption when player state is modified by the spell engine concurrently with player connection or input handling goroutines.
- **Severity:** HIGH

---

## Unported Functions

The following legacy C functions from `magic.c` and `spells.c` have no Go counterparts:

| C Function | Line | Description | Ported? |
|------------|------|-------------|---------|
| `spell_control_weather` | 997 (spells.c) | Modifies MUD weather pressure change trends. (Go has a basic dummy registration but lacks the real dice formulas). | PARTIAL |
| `spell_identify` | 476 (spells.c) | Tells the player full stats of the item/victim. (Go has a stub but does not fetch and print the exact stats and apply types). | PARTIAL |

---

## Summary

- **Total findings:** 6
- **Critical:** 0
- **High:** 4
- **Medium:** 1
- **Low:** 1
- **Unported functions:** 2
