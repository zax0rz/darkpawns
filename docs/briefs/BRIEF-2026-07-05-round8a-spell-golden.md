# Brief: Round 8 — Spell Golden Tests (C→Go Fidelity)

**Scope:** Transcribe C spell tables into Go golden tests to verify `pkg/spells/` fidelity against `src/magic.c`, `src/spell_parser.c`, and `src/class.c`
**Date:** 2026-07-05
**Effort:** L (3 test files, ~4000+ assertions)
**Impact:** Pushes `pkg/spells/` from 29% → ~60%+ coverage

---

## Background

The Go spell system in `pkg/spells/` has zero golden tests. The existing `saving_throws_golden_test.go` (2,460 assertions) covers one table from `src/magic.c`, but the remaining spell tables are unverified. The fidelity audit (`docs/reports/reek/2026-05-17-fidelity.md`) found 3 CRITICAL spell bugs (flamestrike is DOT in C but instant in Go, protect evil/good missing lethal consequence, hellfire level skip inverted). Golden tests would catch these regressions.

### What Already Has Golden Tests (DO NOT re-test)
- ✅ `saving_throws[][]` — `pkg/spells/saving_throws_golden_test.go` (2,460 assertions)
- ✅ `thaco[][]` — `pkg/combat/thaco_golden_test.go` (480 assertions)
- ✅ `str_app[]` — `pkg/combat/str_app_golden_test.go` (62 assertions)
- ✅ `dex_app[]` — `pkg/combat/dex_app_golden_test.go` (78 assertions)
- ✅ `find_exp()` — `pkg/game/find_exp_golden_test.go` (361 assertions)
- ✅ Regen formulas — `pkg/game/regen_golden_test.go` (~30 assertions)

---

## Test 1: Spell Info Metadata — `pkg/spells/spell_info_golden_test.go`

**C source:** `src/spell_parser.c:1225-1626` — the ~90 `spello()` calls

Each `spello()` call defines:
```c
spello(spellnum, mana_max, mana_min, mana_change, min_position, targets, violent_flag, spell_routines);
```

**Go target:** `pkg/spells/spell_info.go:83-94` — `SpellInfo` struct + `spellInfoTable` map, registered via `setupSpellInfo()` at line 146. Access via `GetSpellInfo(spellNum)` at line 97.

**What to test:** For each spell that has a `spello()` call in C, verify the Go `SpellInfo` matches:
- `ManaMin`, `ManaMax`, `ManaChange`
- `MinPosition`
- `MinLevel[class]` for each of the 12 classes
- `Routines` (spell function assignment)
- `Violent` flag
- `Targets` flags

**Cite:** Transcribe all `spello()` calls from `src/spell_parser.c:1225-1626`. The call format is:
```c
spello(  1, 10,  1,  1, POS_STANDING, TAR_CHAR_DEFENSIVE, FALSE, MAG_MANA),
```

**How:** 
1. Read the C file and transcribe all `spello()` calls into a Go `[]spellInfoGolden` struct
2. Call `GetSpellInfo(spellnum)` for each
3. Assert all fields match

**Estimated assertions:** ~90 spells × 7 fields = ~630

---

## Test 2: Spell Level Assignments — `pkg/spells/spell_levels_golden_test.go`

**C source:** `src/class.c:768-1076` — `init_spell_levels()` with ~200 `spell_level()` calls

Each call: `spell_level(gsn, class, level)` assigns the minimum level a class learns a spell/skill.

**Go target:** The `MinLevel[class]` field in `SpellInfo` (populated from the same `spello()` data above). This is redundant with Test 1 if `spello()` already embeds per-class levels. **Verify which source is authoritative** — if `spello()` in C calls `spell_level()` internally, then Test 1 already covers this. If they're separate, this is a standalone test.

**Cite:** `src/class.c:768-1076` — each line like `spell_level(gsn_bash, CLASS_WARRIOR, 1);`

**How:**
1. Transcribe all `spell_level()` calls as `{gsn, class_index, level}` tuples
2. For each tuple, call `GetSpellInfo(gsn)` and assert `MinLevel[class_index] == level`

**Estimated assertions:** ~200

---

## Test 3: Spell Damage Formulas — `pkg/spells/spell_damage_golden_test.go`

**C source:** `src/magic.c:601-849` — `mag_damage()` switch on spellnum

Each case has a dice formula. These are NOT in a table — they're a giant switch statement. Examples:
```c
case SPELL_MAGIC_MISSILE:  dam = dice(4, 3) + reag + level;       break;
case SPELL_FIREBALL:        dam = dice(12, 8) + 20 + level + level + reag; break;
case SPELL_DISINTEGRATE:   dam = dice(18, 8) + 3*level + reag;   break;
case SPELL_EARTHQUAKE:      dam = dice(7, 7) + level;            break;
case SPELL_HARM:            dam = dice(12, 8) + level*2;          break;
case SPELL_PSIBLAST:        dam = dice(15, 13) + 3*level;        break;
```

**Go target:** `pkg/spells/damage_spells.go:12` — `func MagDamage(level int, ch, victim interface{}, spellNum, savetype int, world interface{})`. The switch at line 21 should match the C formulas.

**What to test:** For each damage spell in the C switch, verify the Go implementation produces the same damage range when using a seeded RNG. The test should:
1. Transcribe each spell's formula from C as `{spellnum, numDice, sizeDice, flatBonus, levelCoeff, reagentCoeff}`
2. Use a seeded RNG (like the combat transcript test uses `NewSeededRoller`) 
3. Call `MagDamage()` with known inputs and assert damage falls in expected range

**IMPORTANT:** Since damage is probabilistic, use a seeded RNG and run each spell 1000 times to build a distribution, then assert the min/max/mean match what the C formula would produce. OR: if `MagDamage` is hard to mock, test the formula functions directly (extract the per-spell formulas into testable pure functions if they aren't already).

**Cite:** `src/magic.c:601-849` — full switch statement

**How:**
1. Transcribe ~20-25 damage spell formulas from C
2. For each, create a subtest that verifies the Go formula matches
3. Use table-driven test pattern

**Estimated assertions:** ~25 spells × 3-5 checks each = ~100

---

## Test 4: Spell Healing Formulas — `pkg/spells/spell_healing_golden_test.go`

**C source:** `src/magic.c:1753-1829` — `mag_points()` switch on spellnum

Examples:
```c
case SPELL_CURE_LIGHT:  hit = dice(2, 8) + 1 + (level >> 2);           break;
case SPELL_CURE_CRITIC: hit = dice(5, 8) + 3 + (level >> 2);           break;
case SPELL_HEAL:         hit = 100 + dice(3, 8);                         break;  // non-psi
case SPELL_MASS_HEAL:    hit = 200;                                       break;
case SPELL_LAY_HANDS:    hit = dice(3, GET_LEVEL(ch));                    break;
```

**Go target:** `pkg/spells/` — find the healing spell implementation (likely in a healing_spells.go or affect_spells.go file).

**What to test:** Same pattern as damage — verify each healing formula matches C for a range of caster levels.

**Estimated assertions:** ~8 spells × 3 level checks = ~24

---

## Test 5: Spell Affect Parameters — `pkg/spells/spell_affects_golden_test.go`

**C source:** `src/magic.c:862-1380` — `mag_affects()` switch on spellnum

Each spell applies affects like armor, bless, shield, detect invis, etc. with specific:
- `location` (APPLY_AC, APPLY_STR, APPLY_HITROLL, etc.)
- `modifier` (e.g., -15 for armor)
- `duration` (in game ticks)
- `bitvector` (AFF_BLIND, AFF_INVISIBLE, etc.)

**Go target:** `pkg/spells/` — find the affect spell implementation.

**What to test:** For each affect spell, verify the Go code applies the correct location, modifier, and duration.

**Cite:** `src/magic.c:862-1380`

**Estimated assertions:** ~30 affects × 3 fields = ~90

---

## Build Gate

```bash
go build ./... && go vet ./... && go test ./pkg/spells/...
```

All must pass. The golden test files should be in `pkg/spells/` alongside the existing `saving_throws_golden_test.go`.

## Key Files to Reference

| What | File | Lines |
|------|------|-------|
| C spell damage formulas | `src/magic.c` | 601-849 |
| C spell affects | `src/magic.c` | 862-1380 |
| C spell healing | `src/magic.c` | 1753-1829 |
| C spell info (spello) | `src/spell_parser.c` | 1225-1626 |
| C spell levels | `src/class.c` | 768-1076 |
| Go SpellInfo struct | `pkg/spells/spell_info.go` | 83-94 |
| Go spell damage | `pkg/spells/damage_spells.go` | 12+ |
| Go spell constants | `pkg/spells/spells.go` | 46-197 |
| Existing golden example | `pkg/spells/saving_throws_golden_test.go` | (reference pattern) |
| Existing golden example | `pkg/combat/combat_transcript_golden_test.go` | (seeded RNG pattern) |
| Fidelity audit report | `docs/reports/reek/2026-05-17-fidelity.md` | (known gaps) |
