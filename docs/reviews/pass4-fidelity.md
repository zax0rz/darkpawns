# Pass 4: C-to-Go Fidelity Review

**Reviewer:** Opus (pass 4/5)  
**Date:** 2026-04-26  
**Scope:** Core game logic fidelity — Go port vs original C source (`~/darkpawns/src/`)

## Executive Summary

The Go port faithfully reproduces the core combat hit/miss formula (THAC0, AC, d20 roll) and damage calculation pipeline. However, the **saving throw system has been completely rewritten** from a percentile-based (d100) system with 41-level tables to a d20 system with 21-level tables — this is the single largest mechanical divergence and affects every spell in the game. The combat command handlers (backstab, kick, bash, etc.) are **stub implementations** that don't calculate damage, check skill percentages, or apply the C's complex success/failure formulas. The affect/tick system uses real-time seconds instead of CircleMUD's game-hour ticks, which fundamentally changes buff/debuff duration balance. The `die_with_killer` constitution loss is deterministic in Go but probabilistic in C, making death significantly more punishing in the Go port.

---

## CRITICAL — Game-Breaking Logic Errors

### C1. Saving Throw System Completely Rewritten (d100 → d20, Wrong Tables)

**Go:** `pkg/spells/saving_throws.go` — `CheckSavingThrow()` rolls `rand.Intn(20)+1` (d20, range 1-20). Tables have values 5-17.  
**C:** `src/magic.c:408-424` — `mag_savingthrow()` rolls `number(0, 99)` (d100, range 0-99). Tables have values 0-90 (e.g., Mage level 1 PARA = 70, SPELL = 60).

**What C does:**
```c
save = saving_throws[class][type][level];
save += GET_SAVE(ch, type);  // apply_saving_throw modifier
if (MAX(1, save) < number(0, 99))
    return TRUE;  // saved
```

**What Go does:**
```go
target := GetSavingThrow(class, level, saveType)
roll := rand.Intn(20) + 1
return roll >= target
```

**Impact:** Every spell that checks a saving throw produces wildly different save rates. Example: C Mage level 1 vs SPELL has target 60, meaning ~60% chance to save. Go has ~75% chance (target ~5 on d20). This affects blindness, curse, sleep, poison, chill touch, and every damage spell (saves halve damage). The entire spell balance of the game is broken.

**Additional issue:** Go tables only go to level 21; C tables go to level 40. Go classes 5-11 are copy-pasted from base classes (Psionic=Mage, Paladin=Warrior) rather than using the actual C tables which have distinct values per class.

**Fix:** Port the actual `saving_throws[NUM_CLASSES][5][41]` table from `src/magic.c:83-406` verbatim. Change the roll to `rand.Intn(100)` and the comparison to `MAX(1, save) < roll`. Add `apply_saving_throw` modifier support.

---

### C2. Combat Commands Are Stubs — No Damage Calculation

**Go:** `pkg/session/act_offensive.go` — All combat commands (backstab, kick, bash, dragon_kick, tiger_punch, disembowel, neckbreak, etc.) are **display-only stubs** that print a flavor message and call `StartCombat()` but never calculate or apply damage.

**C:** `src/act.offensive.c` — Each command has specific formulas:
- **Backstab** (C:218-229): `percent = number(1,101)` vs `GET_SKILL(ch, SKILL_BACKSTAB)`, then `hit(ch, vict, SKILL_BACKSTAB)` which in `fight.c:1856` applies `backstab_mult(level)` multiplier to damage.
- **Kick** (C:620-631): `damage(ch, vict, GET_LEVEL(ch) >> 1, SKILL_KICK)` — damage = level/2.
- **Bash** (C:485-495): `damage(ch, vict, (GET_LEVEL(ch)/2)+1, SKILL_BASH)` — knocks victim to POS_SITTING on success.
- **Dragon Kick** (C:679-687): `damage(ch, vict, GET_LEVEL(ch)*1.5, SKILL_DRAGON_KICK)`.
- **Tiger Punch** (C:736-740): `damage(ch, vict, GET_LEVEL(ch)*2.5, SKILL_TIGER_PUNCH)`.
- **Disembowel** (C:271-288): Special formula in `fight.c:1867`: `dam = (GET_LEVEL(ch)*2) + GET_DAMROLL(ch)`.

**Impact:** Players can execute combat skills but they deal zero damage. Skills never improve. Wait states aren't applied. Bash doesn't knock down. Backstab doesn't multiply damage. The entire skill-based combat system is non-functional.

**Fix:** Port each command's damage formula, skill check, WAIT_STATE, success/failure logic, and `improve_skill()` call from C.

---

### C3. Parry/Dodge System Not Implemented

**Go:** `pkg/session/fight.go` — `cmdParry()` is a 3-line stub that prints "Parry toggled." The parry skill is registered in `skill_manager.go` but never used in combat resolution.

**C:** `src/fight.c:1958-1975` — Parry is a significant combat mechanic in `perform_violence()`:
```c
if (!IS_NPC(ch) && number(0,10000) <= GET_SKILL(ch, SKILL_PARRY) &&
    FIGHTING(ch) && FIGHTING(FIGHTING(ch)) == ch) {
    IS_PARRIED(FIGHTING(ch)) = TRUE;
}
// Later: if IS_PARRIED, attacks reduced by dex_app[dex].defensive or -1
```

NPC dodge is also checked (C:1968-1975): `IS_AFFECTED(ch, AFF_DODGE) && number(0,100)<GET_LEVEL(ch)`.

**Impact:** A core defensive mechanic is completely missing. High-skill players can't parry. Mobs with AFF_DODGE don't dodge. This makes combat significantly more lethal than intended.

**Fix:** Implement parry check in the combat round resolution. Track IS_PARRIED state. Apply attack reduction when parried.

---

## HIGH — Significant Behavior Differences

### H1. Position Damage Multiplier: Float vs Integer Division (fight_core.go vs formulas.go)

**Go (fight_core.go:806):**
```go
dam = int(float64(dam) * (1.0 + float64(PosFighting-defPos)/3.0))
```

**Go (formulas.go:445):**
```go
dam *= 1 + delta/3  // integer division — correct
```

**C (fight.c:1852):**
```c
dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3;  // integer division
```

The `MakeHit()` function in fight_core.go uses float division, producing different multipliers than the C code and the library function in formulas.go:

| Position | C (int div) | fight_core.go (float) | formulas.go (int) |
|----------|-------------|----------------------|-------------------|
| Sitting (6) | ×1 (1+0) | ×1.33 | ×1 (correct) |
| Resting (5) | ×1 (1+0) | ×1.67 | ×1 (correct) |
| Sleeping (4) | ×2 (1+1) | ×2.0 | ×2 (correct) |
| Stunned (3) | ×2 (1+1) | ×2.33 | ×2 (correct) |

**Impact:** `MakeHit()` (the actual combat path) deals ~33-67% more damage to sitting/resting targets than C does. This makes it easier to kill resting players and mobs that are knocked down.

**Fix:** Use integer division in `MakeHit()` to match C: `dam *= 1 + (PosFighting-defPos)/3`

---

### H2. Die-With-Killer Constitution Loss Is Deterministic (C Is Probabilistic)

**Go (fight_core.go DieWithKiller):**
```go
conVal := GetConstitution(chName) - 1  // Always lose 1 CON
```

**C (fight.c:601-611):**
```c
if (GET_LEVEL(ch) > 5 && !number(0,3)) {      // 25% chance if level > 5
    ch->real_abils.con--;
    if (GET_LEVEL(ch) > 20 && !number(0,5))    // additional 16.7% chance if level > 20
        ch->real_abils.con--;
    affect_total(ch);
}
```

**Impact:** In C, low-level players (≤5) never lose CON on death. Higher-level players have only a 25% chance, with a further ~17% chance of losing a second point if over level 20. In Go, **every death always costs 1 CON regardless of level.** This makes death much more punishing, especially for low-level characters who should be protected. Over many deaths, Go characters will have significantly lower CON.

**Fix:** Add level checks and random rolls matching C's formula. Also note C calls `affect_total(ch)` to recalculate derived stats — Go doesn't.

---

### H3. Mob THAC0 Comment Says "Set in HitRoll" But Go Uses Flat 20

**Go (formulas.go:247):**
```go
func getTHAC0(c Combatant) int {
    if c.IsNPC() {
        return 20  // "from fight.c line 1786"
    }
```

**C (fight.c:1785-1786):**
```c
else     /* THAC0 for monsters is set in the HitRoll */
    calc_thaco = 20;
```

This is actually faithful to the C code — the base mob THAC0 is 20, and their hitroll modifier brings it down. However, the C code then applies `calc_thaco -= GET_HITROLL(ch)` which for mobs is their mob hitroll stat. The Go code does `calcThaco -= ch.GetHitroll()` which should be equivalent **if** the mob's hitroll is properly loaded from the zone files. This is correct but worth verifying that mob hitroll data flows correctly through the parser.

**Impact:** Potentially none if data loading is correct. Flagged for verification.

---

### H4. Affect Duration Uses Real-Time Seconds Instead of Game Ticks

**Go (engine/affect.go:82-85):**
```go
ExpiresAt: now.Add(time.Duration(duration) * time.Second)
```

**C (magic.c affect_update, called from comm.c):**
`affect_update()` is called once per "game hour" — 75 seconds in CircleMUD (SECS_PER_MUD_HOUR = 75). Each affect's `duration` field represents game hours.

**Go implementation:** 1 tick = 1 second. A spell with `duration = 24` lasts 24 seconds in Go vs 24 × 75 = 1800 seconds (30 minutes) in C.

**Impact:** All buff/debuff durations are approximately **75× shorter** than in the C codebase. Armor spell lasts 24 seconds instead of 30 minutes. Sanctuary lasts 4 seconds instead of 5 minutes. This fundamentally breaks spell balance — buffs are useless if they wear off in seconds.

**Fix:** Either multiply durations by 75, or set tick interval to 75 seconds, or convert the affect system to use game-hour durations.

---

### H5. `number(1,100)` vs `rand.Intn(100)` Off-By-One in Attack Count

**C (fight.c:1928):** `number(1,100) < 60+GET_LEVEL(ch)` — range 1-100  
**Go (formulas.go:527):** `rand.Intn(100) < (60+level)` — range 0-99

C's `number(1,100)` generates 1-100 inclusive. Go's `rand.Intn(100)` generates 0-99. The comparison `< 60+level` means:
- C: values 1 to 59+level succeed → (59+level)/100 probability
- Go: values 0 to 59+level succeed → (60+level)/100 probability

This is a ~1% probability shift per check, applied across 4-5 attack count rolls. **Also** affects `rand.Intn(501)` in Go vs `!number(0, 500)` in C for the level 30+ bonus attack.

**Impact:** Players get very slightly more attacks per round (~1% more likely per check). Small but systematic — compounds across multi-attack builds.

**Fix:** Use `rand.Intn(100) + 1` to match C's `number(1,100)` range.

---

### H6. Counter_Procs Switch Fall-Through Not Faithfully Reproduced

**C (fight.c:1283-1296):**
```c
switch(number(1,3)) {
case 1: GET_MAX_HIT(ch)++;   // falls through
case 2: GET_MAX_MANA(ch)++;  // falls through
case 3: GET_MAX_MOVE(ch)++;  // falls through
default: GET_MAX_HIT(ch)++; break;
}
```

The C switch has **no break statements**, so:
- Roll 1: HP+2, MANA+1, MOVE+1 (falls through all + default)
- Roll 2: HP+1, MANA+1, MOVE+1 (falls through 3 + default)
- Roll 3: HP+1, MOVE+1 (falls through default)

**Go (fight_core.go:984-988):**
```go
IncreaseMaxStat(ch.GetName(), "hp")
IncreaseMaxStat(ch.GetName(), "mana")
IncreaseMaxStat(ch.GetName(), "move")
```

Go always gives +1 to all three, which matches roll 2 but not rolls 1 or 3. The C code has a classic fall-through bug that gives extra HP. The Go comment acknowledges this but doesn't reproduce it.

**Impact:** Minor — only triggers at kill milestone thresholds (1000, 2000, 10000, etc.). Average reward is slightly different. The C bug actually gives an expected +1.67 HP, +1 MANA, +0.67 MOVE per milestone.

**Fix:** Intentional deviation — document as such. If strict fidelity desired, implement the fall-through logic.

---

## MEDIUM — Minor Deviations

### M1. Spell Affect Durations Don't Match C Values

**Go (spells/affect_spells.go):**
- `SpellInvisible`: duration = `12 + getLevel(ch)/4`
- `SpellSanctuary`: duration = `4`
- `SpellCurse` hitroll: duration = `getLevel(ch)/2`

**C (magic.c):**
- `SPELL_INVISIBLE`: duration = `12 + (GET_LEVEL(ch) >> 2)` — same (>>2 = /4)
- `SPELL_SANCTUARY`: duration = `4` — matches
- `SPELL_CURSE` hitroll: duration = `1 + (GET_LEVEL(ch) >> 1)` — C adds 1, Go doesn't

**Impact:** Curse duration is 1 game-hour shorter in Go. Combined with H4 (tick timing), this compounds.

**Fix:** Add the `+1` to curse duration.

---

### M2. Bless Spell: Go Applies AC Instead of Saving Throw

**Go (affect_spells.go:33-34):**
```go
aff = engine.NewAffect(engine.AffectHitRoll, 6, 2, "bless")
aff = engine.NewAffect(engine.AffectArmorClass, 6, -2, "bless")  // Wrong!
```

**C (magic.c:934-942):**
```c
af[0].location = APPLY_HITROLL; af[0].modifier = 2;
af[1].location = APPLY_SAVING_SPELL; af[1].modifier = -2;  // Saving throw bonus
```

**Impact:** Bless should improve saving throws vs spells by -2, but Go applies -2 AC instead. Players lose a defensive benefit against magic and gain an unintended AC bonus.

**Fix:** Change second affect to saving throw type.

---

### M3. Blindness Spell Missing Reagent Bonus and NPC Retaliation

**Go (affect_spells.go:37-41):** No reagent check, no mob retaliation on resist.

**C (magic.c:945-972):** Mages can use a "small lens" reagent for bonus to hitroll penalty and duration. If save succeeds, NPCs hit the caster: `hit(victim, ch, TYPE_UNDEFINED)`.

**Impact:** Mage reagent system for blindness doesn't work. Mobs don't retaliate when they resist blindness.

---

### M4. Movement System Missing Sector-Based Move Costs

**Go (session/act_movement.go):** `cmdSimpleMove()` calls `MovePlayer()` with no movement point check. No sector-type based movement cost.

**C (act.movement.c:151-152):**
```c
need_movement = (movement_loss[SECT(ch->in_room)] +
    movement_loss[SECT(EXIT(ch, dir)->to_room)]) >> 1;
```

Movement costs range from 2 (city/inside) to 8 (desert), averaged between source and destination sectors.

**Impact:** Players can move infinitely without expending movement points. Terrain has no mechanical effect.

**Fix:** Implement sector-based movement costs using the `movement_loss` table from `constants.c`.

---

### M5. Mob AI (mobact) Not Fully Ported

**Go:** No dedicated `mobact.go` or `mobile_activity()` equivalent found. The Go codebase has AI behavior hooks in `pkg/ai/behaviors.go` and `pkg/combat/fight_core.go` (MOB_MEMORY, MOB_HUNTER via PerformCommand hooks), but the full mobile_activity loop is absent.

**C (mobact.c):** Complex mob behavior loop including:
- Double-speed hunting (`hunt_victim` called twice per tick)
- Scavenger picking up highest-value objects
- Random wandering (1-in-3 chance, respects SENTINEL/STAY_ZONE/water/flying)
- Aggressive mob attacks (with alignment-based targeting)
- Race-hate mob attacks
- Memory-based revenge attacks
- Helper mob assist
- AGGR24 (attack players level 24+)
- Sound scripts and onpulse triggers

**Impact:** Mobs are likely static — they don't wander, scavenge, hunt, or aggro. This significantly reduces the game's dynamism.

---

### M6. Spell MagPoints (Heal) Not Verified

**Go (spells/affect_spells.go:98):** References `MagPoints()` but the implementation was truncated in review. The C `mag_points()` (magic.c:1754) handles HEAL, CURE_LIGHT, CURE_CRITIC, etc. with specific dice formulas.

**Impact:** Needs verification that healing amounts match C formulas.

---

### M7. Experience Gain Cap Logic: Go Missing `max_exp-1` Single-Level Cap

**Go (fight_core.go CalcLevelDiff):** Applies level-diff penalties and max_exp_gain cap.

**C (limits.c:319-321):**
```c
gain = MIN(max_exp_gain, gain);
gain = MIN(max_exp-1, gain);  // can only level one time!
```

The C code has a second cap: `max_exp = find_exp(class, level+1) - GET_EXP(ch)`, ensuring you can gain at most `max_exp-1` XP per kill (can't double-level). Go doesn't implement this second cap.

**Impact:** Go players could potentially skip levels with large XP gains from high-level mobs.

---

## LOW — Cosmetic/Style

### L1. Damage Message Thresholds Differ Slightly

**Go (fight_core.go:860-873):** Uses `{0, 1, 3, 5, 7, 11, 18, 26, 36, 48, 60, 80, 101, 10000}` thresholds.

**C (fight.c:889-1020):** Uses the same general structure. Would need exact line-by-line verification of message text but the thresholds appear to match standard CircleMUD.

### L2. Token Replacement in Messages Missing Gender Awareness

**Go (fight_core.go replaceMessageTokens):** `$e` always resolves to "he", `$E` to "him", `$s` to "his". No gender check.

**C:** Uses `HSSH()`, `HMHR()`, etc. macros that check character sex (male/female/neutral).

**Impact:** All combat messages use male pronouns regardless of character gender.

### L3. Missing `flesh_altered_type()` for Unarmed NPC Attack Types

**Go:** When NPC has no weapon and no `attack_type`, defaults to `TYPE_HIT`.  
**C:** Checks `AFF_FLESH_ALTER` and calls `flesh_altered_type()` to determine unarmed attack type.

**Impact:** Flesh-altered mobs won't have correct attack text.

### L4. `randomString()` in affect.go Uses Broken PRNG

**Go (engine/affect.go:163):**
```go
b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
```

All characters in the string are generated from the same nanosecond, so the "random" string is likely all the same character.

**Impact:** Affect IDs may collide. Not a fidelity issue but a real bug.

---

## Prioritized Summary

| # | Severity | Title | Files | Effort |
|---|----------|-------|-------|--------|
| C1 | CRITICAL | Saving throw system completely wrong (d100→d20, wrong tables) | spells/saving_throws.go vs magic.c | High |
| C2 | CRITICAL | Combat commands are stubs — no damage/skill checks | session/act_offensive.go vs act.offensive.c | High |
| C3 | CRITICAL | Parry/dodge system not implemented | session/fight.go vs fight.c:1958-1975 | Medium |
| H4 | HIGH | Affect durations 75× too short (seconds vs game-hours) | engine/affect.go vs magic.c | Medium |
| H1 | HIGH | Position damage multiplier float vs int division | combat/fight_core.go:806 vs fight.c:1852 | Low |
| H2 | HIGH | Constitution loss deterministic vs probabilistic | combat/fight_core.go DieWithKiller vs fight.c:601-611 | Low |
| H5 | HIGH | number(1,100) vs rand.Intn(100) off-by-one | combat/formulas.go:527 vs fight.c:1928 | Low |
| H6 | HIGH | Counter_procs fall-through not reproduced | combat/fight_core.go:984 vs fight.c:1283 | Low |
| M4 | MEDIUM | Movement missing sector-based move costs | session/act_movement.go vs act.movement.c | Medium |
| M5 | MEDIUM | Mob AI (mobile_activity) not fully ported | — vs mobact.c | High |
| M2 | MEDIUM | Bless spell applies AC instead of saving throw | spells/affect_spells.go:34 vs magic.c:934 | Low |
| M7 | MEDIUM | Missing single-level XP cap | — vs limits.c:320 | Low |
| M1 | MEDIUM | Curse duration off by 1 | spells/affect_spells.go vs magic.c | Low |
| M3 | MEDIUM | Blindness missing reagent/retaliation | spells/affect_spells.go vs magic.c:945 | Low |
| L4 | LOW | randomString() PRNG broken | engine/affect.go:163 | Low |
| L2 | LOW | Gender-unaware message tokens | combat/fight_core.go | Low |
| L3 | LOW | Missing flesh_altered_type() | combat/fight_core.go | Low |
| L1 | LOW | Damage message text verification needed | combat/fight_core.go | Low |

**Recommended priority order:** C1 → H4 → C2 → C3 → H1 → H2 → M4 → M5 → M2. The saving throw rewrite (C1) and affect timing (H4) are the most impactful because they affect every spell interaction in the game.
