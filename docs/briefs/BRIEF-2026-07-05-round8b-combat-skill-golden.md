# Brief: Round 8b — Combat & Skill Golden Tests (C→Go Fidelity)

**Scope:** Transcribe C combat/skill tables into Go golden tests to verify `pkg/combat/`, `pkg/command/`, and `pkg/game/` fidelity against `src/fight.c`, `src/class.c`, and `src/constants.c`
**Date:** 2026-07-05
**Effort:** L (3-4 test files, ~2000+ assertions)
**Impact:** Pushes `pkg/combat/` from 59% → ~80%+ and `pkg/command/` from 8.8% → ~20%+

---

## Background

Combat and skill formulas are split across `pkg/combat/` (formulas), `pkg/game/skill_combat.go` (Do* functions), and `pkg/command/skill_commands.go` (Cmd* dispatchers). Each skill has its own ad-hoc success formula — there is no common `SkillCheck()` function. The C source has tables and formulas for backstab multiplier, AC reduction, damage messages, attack rounds, and position multipliers that are entirely untested in Go.

### What Already Has Golden Tests (DO NOT re-test)
- ✅ `thaco[][]` — `pkg/combat/thaco_golden_test.go`
- ✅ `str_app[]` — `pkg/combat/str_app_golden_test.go`
- ✅ `dex_app[]` — `pkg/combat/dex_app_golden_test.go`
- ✅ `find_exp()` — `pkg/game/find_exp_golden_test.go`
- ✅ Regen formulas — `pkg/game/regen_golden_test.go`
- ✅ Combat transcript — `pkg/combat/combat_transcript_golden_test.go`

---

## Test 1: Backstab Multiplier — `pkg/combat/backstab_golden_test.go`

**C source:** `src/class.c:720-728`
```c
int backstab_mult(int level) {
    if (level <= 0)  return 1;
    if (level >= 100) return 20;
    return (int)(level * 0.2) + 1;
}
```

**Go target:** `pkg/combat/fight_core.go:997`
```go
func BackstabMult(level int) float64
```
Formula: `float64(level)*0.2 + 1.0`, capped at 20.0 for immortals.

**What to test:**
- Level 0 → 1 (no backstab for level 0)
- Level 1 → 1.2
- Level 5 → 2.0
- Level 10 → 3.0
- Level 15 → 4.0
- Level 20 → 5.0
- Level 25 → 6.0
- Level 30 → 7.0
- Level 99 → 20.0 (immortal cap)
- Level 101 → 20.0 (immortal cap)

**How:** Table-driven test with ~15 entries.

**Estimated assertions:** ~15

---

## Test 2: AC Damage Reduction — `pkg/combat/ac_reduction_golden_test.go`

**C source:** `src/fight.c:1721-1759` — `get_minusdam()`

The AC reduction is NOT a table — it's a chain of if-statements mapping AC ranges to damage reduction percentages. Transcribable as ~23 threshold entries:

```c
if (AC >= 100)        reduction = 0;
else if (AC >= 90)    reduction = 1;   // 2%
else if (AC >= 80)    reduction = 2;   // 4%
else if (AC >= 70)    reduction = 3;   // 6%
else if (AC >= 60)    reduction = 5;   // 10%
else if (AC >= 50)    reduction = 7;   // 14%
else if (AC >= 40)    reduction = 9;   // 18%
else if (AC >= 30)    reduction = 11;  // 22%
else if (AC >= 20)    reduction = 13;  // 26%
else if (AC >= 10)    reduction = 15;  // 30%
else if (AC >= 0)     reduction = 17;  // 34%
else if (AC >= -10)   reduction = 18;  // 36%
else if (AC >= -20)   reduction = 19;  // 38%
else if (AC >= -30)   reduction = 20;  // 40%
else if (AC >= -40)   reduction = 21;  // 42%
else if (AC >= -60)   reduction = 23;  // 46%
else if (AC >= -80)   reduction = 25;  // 50%
else if (AC >= -100)  reduction = 27;  // 54%
else if (AC >= -150)  reduction = 29;  // 58%
else if (AC >= -200)  reduction = 31;  // 62%
else if (AC >= -250)  reduction = 32;  // 64%
else                   reduction = 32;  // 64% (floor)
```

Actual formula: `dam -= (dam * reduction * 2) / 100`

**Go target:** Find the AC reduction function in `pkg/combat/`. Search for `get_minusdam`, `AC`, `armor class` in combat files.

**What to test:** For each AC threshold, given a fixed damage (e.g., 100), assert the reduced damage matches C.

**Cite:** `src/fight.c:1721-1759`

**How:** Table-driven test with 23 entries. For each: `assertReducedDamage(inputAC: 100, inputDam: 100, expected: calculated)`.

**Estimated assertions:** ~23 × 2 = ~46

---

## Test 3: Damage Message Thresholds — `pkg/combat/damage_messages_golden_test.go`

**C source:** `src/fight.c:895-976` — `dam_message()`

Maps damage ranges to singular/plural message strings:
| Index | Damage Range | Messages |
|-------|-------------|----------|
| 0 | 0 (miss) | `$n's attack misses $N.`, etc. |
| 1 | 1-2 | `$n scratches $N.` |
| 2 | 3-4 | `$n barely grazes $N.` |
| 3 | 5-6 | `$n barely grazes $N.` (wait — check exact C text) |
| 4 | 7-10 | `$N is hit hard.` |
| 5 | 11-14 | `$N is hit very hard.` |
| 6 | 15-19 | `$N is hit extremely hard.` |
| 7 | 20-23 | `$N massacres $N.` |
| 8 | 24-33 | `$N OBLITERATES $N.` |
| 9 | 34-43 | `$N EVISCERATES $N.` |
| 10 | 44-53 | `$N DESTROY(S) $N.` |
| 11 | 54+ | `$N ROCKS THE HELL OUT OF $N!!!` |

**Go target:** Find the damage message function in `pkg/combat/`. Likely in `fight_core.go` or `messages.go`.

**What to test:** For each threshold boundary, assert the correct message string is selected.

**Cite:** `src/fight.c:895-976` — transcribe exact strings and thresholds.

**Estimated assertions:** ~12 thresholds × 3 messages (room, char, victim) = ~36

---

## Test 4: Skill Success Formulas — `pkg/game/skill_formulas_golden_test.go`

**C source:** Various locations in `src/fight.c` — each skill's `do_bash`, `do_kick`, `do_trip`, etc.

The Go implementations are in `pkg/game/skill_combat.go`. Each skill has a UNIQUE formula:

| Skill | Go Location | C Source | Success Formula |
|-------|------------|----------|-----------------|
| Bash | `skill_combat.go:118` | `src/fight.c` | `percent = ((5-(AC/10))*2) + rand(1,101); success if percent <= skill` |
| Kick | `skill_combat.go:204` | `src/fight.c` | `percent = ((7-(AC/10))*2) + rand(1,101); success if percent <= skill` |
| Trip | `skill_combat.go:255` | `src/fight.c` | `percent = rand(1,121) + max(vict_level-ch_level, 0); success if percent <= skill` |
| Headbutt | `skill_combat.go:339` | `src/fight.c` | `percent = rand(1,121); success if percent <= skill` |
| Backstab | `skill_combat.go:24` | `src/fight.c` | `percent = rand(1,101)+1; success if percent <= skill` |
| Circle | `skill_combat.go:633` | `src/fight.c` | `percent = rand(1,101)+1; success if percent <= skill` |
| Rescue | `skill_combat.go:428` | `src/fight.c` | `percent = rand(1,101)+1; success if percent <= skill` |
| Charge | `skill_combat.go:735` | `src/fight.c` | `percent = ((5-(AC/10))*2) + rand(1,101); success if percent <= skill` |
| Disembowel | `skill_combat.go` | `src/fight.c` | (find the formula) |
| DragonKick | `skill_combat.go` | `src/fight.c` | (find the formula) |

**Go target:** Each `DoXxx()` function in `pkg/game/skill_combat.go` returns a `SkillResult` with a `Success` bool.

**What to test:** For each skill, verify:
1. The success probability formula matches C (test with seeded RNG at different skill levels)
2. The damage formula on success matches C
3. Edge cases: AC of victim affects bash/kick/charge but not trip/headbutt

**IMPORTANT:** These use probabilistic RNG. Use a seeded RNG and test at boundary skill values (0%, 50%, 100%) to verify the formula structure.

**Cite:** Find each `do_xxx` function in `src/fight.c` and match to the Go equivalent.

**How:**
1. For each skill, transcribe the C formula
2. Create a test that sets up a mock character with known skill level, AC, etc.
3. Run 10000 iterations with a counter to verify the success rate matches the expected probability
4. Assert damage-on-success values

**Estimated assertions:** ~8-10 skills × 3-5 checks = ~40

---

## Test 5: Position Damage Multipliers — `pkg/combat/position_damage_golden_test.go`

**C source:** `src/fight.c:1854-1861`
```c
dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3
```

Effective multipliers based on position enum values:
| Victim Position | Multiplier |
|-----------------|------------|
| Standing (POS_FIGHTING=4) | 1.0 (no bonus) |
| Resting (POS_RESTING=2) | ~1.67 |
| Sitting (POS_SITTING=1) | ~1.33 |
| Sleeping (POS_SLEEPING=0) | ~2.0 |
| Stunned | ~2.33 |
| Incapacitated | ~2.67 |
| Mortally wounded | ~3.0 |
| Dead | ~3.33 |

**Go target:** Find the position damage multiplier in `pkg/combat/`. Search for position-based damage multiplier in `formulas.go` or `fight_core.go`.

**What to test:** For each position, given a base damage of 100, assert the multiplied damage matches C.

**Cite:** `src/fight.c:1854-1861`

**How:** Table-driven with ~8 positions.

**Estimated assertions:** ~8

---

## Test 6: NPC/PC Attack Rounds — `pkg/combat/attack_rounds_golden_test.go`

**C source:** `src/fight.c:1910-1947`

NPC attack schedule (deterministic by level):
| NPC Level | Base Attacks |
|-----------|--------------|
| ≤10 | 1 |
| 11-20 | 2 |
| 21-27 | 3 |
| 28-30 | 4 |
| ≥31 | 5 |
| +1 if `number(0,900) < level` | bonus attack |
| Haste +1, Slow -1 | modifier |

PC attack schedule (probabilistic, per-class):
| Class | Extra attack threshold | Formula |
|-------|----------------------|---------|
| Warrior/Paladin/Ranger | level>10 | `number(1,100) < 60+level` |
| Avatar/Ninja | level>12 | `number(1,100) < 60+level` |
| Thief/Assassin | level>15 | `number(1,100) < 30+level` |
| All classes | level>25 | `number(1,100) < 75` |
| All classes | level>30 or `!number(0,500)` | always / 50% |
| All classes | level>39 | +2 (cumulative) |

**Go target:** Find the attack round calculation in `pkg/combat/`. Likely in `fight_core.go` where NPC attack counts are determined.

**What to test:** 
1. NPC attack counts at each level threshold (deterministic part)
2. PC attack probability distributions per class (probabilistic — use seeded RNG over 10000 iterations)

**Cite:** `src/fight.c:1910-1947`

**How:** Table-driven for NPC levels, statistical for PC levels.

**Estimated assertions:** ~10 deterministic + ~30 statistical = ~40

---

## Test 7: Practice Parameters — `pkg/game/prac_params_golden_test.go`

**C source:** `src/class.c:261-267`
```c
int prac_params[4][NUM_CLASSES] = {
    /* learned_level: */ {95, 95, 85, 80, 95, 95, 85, 80, 85, 95, 80, 95},
    /* max_per_prac:  */ {100, 100, 25, 25, 100, 100, 25, 25, 25, 100, 25, 100},
    /* min_per_prac:  */ {25, 25, 0, 0, 25, 25, 0, 0, 0, 25, 0, 25},
    /* prac_type:    */ {SPELL, SPELL, SKILL, SKILL, SPELL, BOTH, SKILL, BOTH, BOTH, BOTH, SKILL, BOTH}
};
```

Class order: Mage(0), Cleric(1), Thief(2), Warrior(3), Magus(4), Avatar(5), Assassin(6), Paladin(7), Ninja(8), Psionic(9), Ranger(10), Mystic(11)

**Go target:** Find practice parameter definitions in `pkg/game/` or `pkg/session/`. Search for `learned_level`, `max_per_prac`, `practice`, `prac_type`.

**What to test:** For each class, assert the learned_level, max_per_prac, min_per_prac, and practice type match C.

**Cite:** `src/class.c:261-267`

**Estimated assertions:** 4 rows × 12 classes = 48

---

## Test 8: con_app, int_app, wis_app — `pkg/game/attribute_app_golden_test.go`

**C source:** `src/constants.c:1124-1214`

Three more attribute application tables not yet golden-tested:

**con_app[]** (con→{hitp, shock}) — 26 entries (con 0-25)
**int_app[]** (int→{learn}) — 26 entries (int 0-25)  
**wis_app[]** (wis→{bonus}) — 26 entries (wis 0-25)

**Go target:** Find the Go equivalents in `pkg/game/` or `pkg/combat/`. Search for `ConApp`, `IntApp`, `WisApp`, `con_app`, `int_app`, `wis_app`.

**What to test:** Same pattern as existing `str_app_golden_test.go` and `dex_app_golden_test.go`.

**Cite:** `src/constants.c:1124-1214`

**Estimated assertions:** 3 tables × 26 entries × 2-3 fields = ~200

---

## Build Gate

```bash
go build ./... && go vet ./... && go test ./pkg/combat/... ./pkg/game/... ./pkg/command/...
```

All must pass.

## Key Files to Reference

| What | File | Lines |
|------|------|-------|
| C backstab_mult | `src/class.c` | 720-728 |
| C AC reduction | `src/fight.c` | 1721-1759 |
| C damage messages | `src/fight.c` | 895-976 |
| C NPC/PC attacks | `src/fight.c` | 1910-1947 |
| C position multipliers | `src/fight.c` | 1854-1861 |
| C practice params | `src/class.c` | 261-267 |
| C con/int/wis_app | `src/constants.c` | 1124-1214 |
| C skill formulas | `src/fight.c` (search `do_bash`, `do_kick`, etc.) | various |
| Go BackstabMult | `pkg/combat/fight_core.go` | 997 |
| Go skill combat | `pkg/game/skill_combat.go` | all |
| Go combat formulas | `pkg/combat/formulas.go` | all |
| Existing golden pattern | `pkg/combat/str_app_golden_test.go` | (reference) |
| Combat fidelity audit | `docs/reports/reek/2026-05-10-fidelity.md` | (known gaps) |
