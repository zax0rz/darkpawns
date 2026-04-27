# Source Accuracy Audit — 2026-04-23

**Auditor:** BRENDA69 (automated source comparison)  
**Scope:** 8 critical game systems, C source (`src/`) vs Go port (`pkg/`)  
**C codebase:** CircleMUD-derived Dark Pawns (1996-2008)  
**Go codebase:** Current working tree

---

## System 1: Combat Formulas

**Status:** ⚠️ MINOR DIVERGENCE (3 issues)

**C source:** `src/fight.c` lines 1722–1975 (`get_minusdam`, `hit`, `perform_violence`)  
**Go source:** `pkg/combat/formulas.go` lines 1–395

### Findings:

#### 1a. Position damage multiplier — formula mismatch
- **C source:** `fight.c` line ~1854: `dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3;`
  - Computes multiplier first via integer division, then multiplies: `dam * (1 + delta/3)`
  - Sitting (delta=1): multiplier = `1 + 0 = 1.0` → no change
  - Resting (delta=2): multiplier = `1 + 0 = 1.0` → no change
  - Sleeping (delta=3): multiplier = `1 + 1 = 2.0` → double damage
- **Go source:** `formulas.go` line ~298: `dam = dam * (1 + (PosFighting - defPos)) / 3`
  - This is `(dam * (1 + delta)) / 3` — completely different formula!
  - Sitting (delta=1): `(dam * 2) / 3 = 0.66×` — **REDUCES** damage instead of no change
  - Sleeping (delta=3): `(dam * 4) / 3 = 1.33×` — only +33% instead of +100%
- **Impact:** **SIGNIFICANT.** Sleeping/sitting/resting victims take wrong damage. This changes PvP balance noticeably. The C comment even documents the intended multipliers (sitting x1.33, resting x1.66, sleeping x2.0, etc.) — the Go code doesn't match either its own comments or the C original.
- **Fix:** Change to `dam = dam * (1 + (PosFighting - defPos) / 3)` (parentheses around the division).

#### 1b. getMinusDam — missing high-AC entries
- **C source:** `fight.c` lines 1722–1760: extends from ac > 90 down to ac <= -290, with entries at -95/-110/-130/-150/-170/-190/-210/-230/-250/-270/-290, maxing at 0.32×pcmod
- **Go source:** `formulas.go` lines ~197–258: stops at `ac <= -150` with `0.24*pcmod`. Missing entries for ac -170 through -290 (0.25 through 0.32).
- **Impact:** Characters with very low AC (< -150) get less damage reduction in Go than C. Uncommon but affects endgame tanks.

#### 1c. Missing combat modifiers in Go
- **Blessed weapon THAC0 bonus:** C `fight.c` line ~1796: `if (IS_OBJ_STAT(wielded, ITEM_BLESS)) calc_thaco -= 1;` — Go `CalculateHitChance` does not check this.
- **Drunk THAC0 penalty:** C `fight.c` line ~1806: `if (GET_COND(ch, DRUNK) > 1) calc_thaco += 2;` — Go does not check drunkenness.
- **Race hate damage bonus:** C `fight.c` line ~1680: loops `GET_RACE_HATE(ch, i)` and adds `GET_LEVEL(ch)` per match — Go does not have this.
- **Sanctuary halving:** C `fight.c` line ~1683: `if (IS_AFFECTED(victim, AFF_SANCTUARY)) dam /= 2;` — Go `CalculateDamage` does not apply sanctuary.
- **Protection from evil/good:** C `fight.c` lines ~1685-1688: reduces damage by `GET_LEVEL(victim)/4` — Go does not apply this.
- **Note:** Some of these may be intentionally deferred to Phase 3, but they're not marked as TODOs in the combat formulas code.

#### 1d. THAC0 tables — ✅ EXACT MATCH
All 12 class THAC0 tables (41 entries each) match the C source exactly.

#### 1e. str_app/dex_app tables — ✅ EXACT MATCH
str_app (31 entries, tohit/todam) and dex_app (26 entries, reaction/miss_att/defensive) are verbatim matches.

#### 1f. strIndex (STRENGTH_APPLY_INDEX) — ✅ EXACT MATCH
Exceptional strength handling (18/01-50 → index 26, through 18/100 → index 30) matches.

#### 1g. Hit chance formula — ✅ EXACT MATCH
calc_thaco computation, diceroll, victim_ac calculation, miss/hit logic all match.

#### 1h. Damage calculation base — ✅ EXACT MATCH  
str_app todam, damroll, weapon dice, bare hands (number(0, level/3)), and minimum 1 damage all match.

#### 1i. Attacks per round — ✅ EXACT MATCH
Both mob and player attack count logic matches, including all class-specific thresholds and random chances. Minimum attacks clamped to 0 in C, 1 in Go (Go is more generous — C allows 0 attacks with slow).

#### 1j. Backstab multiplier — ✅ EXACT MATCH
`(level * 0.2) + 1` formula matches. C returns int (truncates), Go returns float64 — this is correct since C does integer truncation via implicit cast, and Go uses the float for multiplication.

---

## System 2: Class & Character Stats

**Status:** ❌ SIGNIFICANT DIVERGENCE

**C source:** `src/class.c` lines 297–740, `src/constants.c` lines 1124–1213  
**Go source:** `pkg/game/level.go` lines 1–230

### Findings:

#### 2a. wis_app table — ❌ COMPLETELY WRONG
- **C source:** `constants.c` lines 1187–1213: `{0,0,0,0,0,0,0,0,0,0,0,0,2,2,3,3,3,4,5,6,6,6,6,6,7,7,7}` (26 entries, indices 0–25)
- **Go source:** `level.go` lines ~52–78: `{-50,-45,-40,-35,-30,-25,-20,-15,-10,-5,0,0,0,0,0,1,1,2,2,2,3,3,3,3,3,3}` (26 entries)
- **Impact:** **CRITICAL.** This table controls practice point gains per level. In C, a WIS 18 character gets +5 practices per level; in Go they get +2. WIS 0 in C gives 0 bonus; in Go it gives -50 (which would be clamped to 2 by MAX(2, ...) but still wrong for MIN(2, MAX(1, ...)) paths). Every single value from index 0–11 and 17–25 is wrong. This changes the entire practice economy.

#### 2b. con_app table — ✅ EXACT MATCH
All 26 entries (hitp, shock) match.

#### 2c. AdvanceLevel class gains — ✅ EXACT MATCH
All 12 class HP/mana/move ranges, mana caps, and practice formulas match the C source. The `number(a, b)` → `rand.Intn(b-a+1)+a` conversions are correct.

#### 2d. AdvanceLevel mana gain — ✅ CORRECT
Go correctly computes `number(level, 3*level)` as `rand.Intn(2*level+1)+level`. The MIN caps (10 or 5) are applied correctly per class.

#### 2e. Missing: Mana gain only at level > 1
- **C source:** `class.c` line ~714: `if (GET_LEVEL(ch) > 1) ch->points.max_mana += add_mana;`
- **Go source:** `level.go`: Unconditionally adds mana at all levels including level 1.
- **Impact:** Level 1 characters get mana they shouldn't in Go.

#### 2f. Missing: Immortal condition reset
- **C source:** `class.c` lines ~716-718: At LVL_IMMORT, sets all 3 conditions to -1 (no hunger/thirst/drunk) and sets HOLYLIGHT.
- **Go source:** Has TODO comment but doesn't implement.
- **Impact:** Immortals would starve/die of thirst in Go.

---

## System 3: Spell System

**Status:** ❌ NOT IMPLEMENTED (stub only)

**C source:** `src/magic.c` (1700+ lines), `src/spells.c`  
**Go source:** `pkg/spells/spells.go` (60 lines, constants only)

### Findings:

#### 3a. Spell casting — ❌ NOT IMPLEMENTED
`Cast()` is a stub function with no logic.

#### 3b. Saving throws — ❌ NOT IMPLEMENTED
C has full `saving_throws[NUM_CLASSES][5][41]` table (lines 83-406 of magic.c) and `mag_savingthrow()` function. Go has none of this.

#### 3c. Spell damage formulas — ❌ NOT IMPLEMENTED
C has `mag_damage()` with spell-specific damage calculations, saving throw halving, and special effects. None ported.

#### 3d. Spell constants — ✅ CORRECT
The numeric spell/skill constants in Go match the C `#define` values from `spells.h`.

---

## System 4: Regen & Limits

**Status:** ⚠️ MINOR DIVERGENCE

**C source:** `src/limits.c` lines 59–580  
**Go source:** `pkg/game/limits.go` lines 1–340

### Findings:

#### 4a. Mana gain class modifiers — ✅ CORRECT
Base gain 14, position modifiers (sleeping x2, resting +50%, sitting +25%), and class bonuses match. The Go switch correctly mirrors the C if/else-if chain.

#### 4b. Hit gain — ⚠️ MINOR ISSUE
- C: Only MAGIC_USER and CLERIC get half HP regen (`gain >>= 1`)
- Go: Same (`gain >>= 1`)
- Missing in Go: `is_veteran()` +12 bonus, KK_JIN skill bonus, equipment regen bonuses. These are marked as Phase 3 TODOs — acceptable.

#### 4c. Move gain — ✅ CORRECT  
Base 20, position modifiers match. Missing veteran/KK_ZHEN/equipment bonuses are marked as Phase 3 TODOs.

#### 4d. GainCondition — ⚠️ MINOR ISSUE
- C: Clamps condition to `[0, 48]` with `MAX(0, MIN(48, value))`
- Go: Also clamps to `[0, 48]` — ✅ MATCH
- Messages: C sends messages when condition <= 1 AND not writing. Go checks `> 1` to return early, then checks `> 0` for partial vs `== 0` for empty. This matches C's behavior.
- Missing: `PLR_WRITING` flag check (Go has TODO comment)

#### 4e. PointUpdate — ⚠️ MINOR DIVERGENCE
- C: Iterates `character_list` (both PCs and NPCs). Go only iterates players.
- C: Mob regen uses different formulas (`mana_gain` for NPCs = `GET_LEVEL(ch)`, `hit_gain` for NPCs = `2.5*level` or `4*level`, `move_gain` for NPCs = `GET_LEVEL(ch)`).
- Go: No mob regen at all.
- C: Applies poison damage (10), cutthroat damage (13) via `damage()` calls.
- Go: Has TODO comments for Phase 3.
- C: Checks `PRF_INACTIVE` before regen — Go has TODO.
- C: Calls `dream()` for sleeping characters — Go doesn't have dream system.
- C: Increments jail timer, tattoo timer — Go doesn't have these.
- C: `update_char_objects()` for object timers — Go doesn't have this.

#### 4f. Hunger/thirst damage — ⚠️ DIVERGENCE
- C: Hunger/thirst damage at 0 is handled inside the position check section — characters at POS_STUNNED+ take 1 damage from poison/cutthroat, and separately at POS_INCAP take 1, at POS_MORTALLY take 2. Hunger/thirst at 0 doesn't directly deal damage in `point_update()` in C — it only reduces regen gains.
- Go: Explicitly deals 1 damage when hunger OR thirst <= 0 AND hp > 0, with "STARVING"/"DYING OF THIRST" messages.
- **This may be a Go addition, not from C.** The C code doesn't appear to directly damage from hunger/thirst in `point_update()`. It only reduces regen by `gain >>= 2`.

---

## System 5: Affect System

**Status:** ❌ SIGNIFICANT DIVERGENCE (architectural difference)

**C source:** `src/handler.c` lines 280–500 (`affect_modify`, `affect_total`, `affect_to_char`, `affect_remove`)  
**Go source:** `pkg/engine/affect.go`, `pkg/engine/affect_manager.go`

### Findings:

#### 5a. Architecture difference — ❌ FUNDAMENTAL
- **C:** Uses `affect_total()` which strips ALL affects, resets character to `real_abils` (base stats), then re-applies everything. This is a full recalculation every time any affect changes. Guarantees correctness.
- **Go:** Uses incremental apply/remove. `applyAffectImmediate()` adds magnitude to stat, `removeAffectImmediate()` subtracts it. No full recalculation.
- **Impact:** If affects are applied/removed in the wrong order, or if stats are modified between affect application and removal, the Go system can drift from correct values. The C approach is self-correcting; the Go approach is not.

#### 5b. Affect locations — ❌ MISMATCH
- **C:** Uses numeric `APPLY_*` constants (APPLY_STR=1, APPLY_DEX=2, ..., APPLY_HITROLL=18, APPLY_DAMROLL=19, etc.) in `aff_apply_modify()`. Equipment and spells both use these same location codes.
- **Go:** Uses a custom `AffectType` enum (AffectStrength=0, AffectDexterity=1, ..., AffectHitRoll=6, AffectDamageRoll=7, etc.). These are NOT the same numbering as C's APPLY_ constants.
- **Impact:** Any code that converts between C affect locations and Go affect types must map correctly. The Go system cannot directly use C affect data without translation.

#### 5c. Stat clamping — ❌ MISSING
- **C:** `affect_total()` clamps all stats to `[0, 18]` for PCs (or `[0, 25]` for NPCs), and handles exceptional strength overflow (str > 18 converts excess to str_add).
- **Go:** No stat clamping in `applyAffectImmediate()`. Stats can go negative or exceed 18/25.
- **Impact:** A debuff could reduce STR below 0, or a buff could push DEX above 18, causing out-of-bounds array access when indexing into stat tables like str_app/dex_app.

#### 5d. Bitvector handling — ⚠️ DIFFERENT
- **C:** Affects carry a `bitvector` (long) and an array version (`bitv[]`). `affect_modify()` sets/clears bitvector flags on `AFF_FLAGS(ch)`. Multiple affects can share the same bitvector flag.
- **Go:** Status flags use individual bits (`1 << n`) per affect type. Removing a status affect clears its bit, even if another affect also set that bit.
- **Impact:** If two affects both grant Sanctuary, removing one would remove Sanctuary in Go but not in C (C uses `affect_total()` which re-applies the remaining affect).

#### 5e. Duration/tick model — ⚠️ DIFFERENT
- **C:** Affect durations are decremented in `affect_update()` (separate from `point_update()`). When duration hits 0, `affect_remove()` is called.
- **Go:** Affects track `ExpiresAt` as wall-clock time AND `Duration` as tick count. The `Tick()` method decrements duration. Two time-tracking systems may drift.

---

## System 6: Object/Equipment System

**Status:** ⚠️ MINOR DIVERGENCE

**C source:** `src/handler.c` lines 185–188, `src/fight.c` (GET_HITROLL/GET_DAMROLL macros)  
**Go source:** `pkg/game/player.go` lines 470–510 (`GetHitroll`, `GetDamroll`)

### Findings:

#### 6a. Hitroll/Damroll calculation — ❌ SIGNIFICANT
- **C:** `GET_HITROLL(ch)` reads `ch->points.hitroll` which is maintained by `affect_total()`. This includes base value + equipment affects (APPLY_HITROLL) + spell affects (APPLY_HITROLL from buff spells).
- **Go:** `GetHitroll()` only sums APPLY_HITROLL (location 18) from equipped items. It does NOT include spell-based hitroll modifiers or any base hitroll value.
- **Impact:** Any spell or affect that modifies hitroll (e.g., a bless spell giving +1 hitroll) has NO effect in Go. This is a gameplay-affecting bug.

#### 6b. Equipment slots — ✅ FUNCTIONALLY EQUIVALENT
Both systems use a slot-based map. Equipment bonuses iterate over slots and sum affect modifiers. The slot enumeration and iteration match.

---

## System 7: Event Queue

**Status:** ✅ ACCURATE (functionally equivalent)

**C source:** `src/events.c`, `src/queue.c`  
**Go source:** `pkg/events/queue.go`

### Findings:

#### 7a. Core semantics — ✅ MATCH
- Create event with delay: `event_create(func, obj, when)` → Go `Create(delay, ...)`
- Cancel event: `event_cancel(event)` → Go `Cancel(id)`
- Process events: `event_process()` fires all events with `when <= pulse` → Go `Process()` does the same
- Re-enqueue on positive return: C returns `long` from event func, if > 0, re-enqueue at `pulse + return_value` → Go matches this exactly (lines ~206-210)

#### 7b. Data structure — ⚠️ DIFFERENT (acceptable)
- **C:** Uses bucketed linked-list queue (`struct q_element` with doubly-linked list)
- **Go:** Uses `container/heap` priority queue
- **Impact:** Different performance characteristics but identical semantics. Go's heap is actually better for large event counts.

#### 7c. Cancel during process — ✅ HANDLED
Both systems handle cancellation gracefully. Go marks events as `Cancelled` and skips them during Process(). C calls `queue_deq()` to remove. Functionally equivalent.

#### 7d. Event fields — ⚠️ ENHANCED
- **C:** Events carry `(func, event_obj, when)` only
- **Go:** Events carry `(Source, Target, Obj, Argument, Trigger, EventType)` — richer metadata for Lua scripting integration
- **Impact:** This is a deliberate enhancement, not a bug.

---

## System 8: Death & Experience

**Status:** ⚠️ MINOR DIVERGENCE

**C source:** `src/fight.c` lines 535–690 (`die`, `die_with_killer`, `raw_kill`, `make_corpse`, `group_gain`, `calc_level_diff`)  
**Go source:** `pkg/game/death.go`, `pkg/combat/engine.go`

### Findings:

#### 8a. XP loss formulas — ✅ EXACT MATCH
- Combat death: `GET_EXP(ch)/37` — Go matches (death.go line comment "fight.c line 590")
- Non-combat death: `GET_EXP(ch)/3` — Go matches
- Note: C uses integer division, Go also uses integer division. Match.

#### 8b. Corpse creation — ✅ MATCHES CORE BEHAVIOR
- Transfer inventory + equipment + gold into corpse container — Go does this correctly
- Corpse descriptions based on attack type — Go has partial coverage (fire, cold, blast, energy drain, lightning, psiblast, slash, disembowel, drowning, petrify, crush, bruised, pierce, neckbreak)
- Missing: Some C corpse descriptions for specific attack types (whip, bludgeon, etc.) use a broader mapping that Go simplifies

#### 8c. Disintegrate (make_dust) — ✅ MATCH
Scatters inventory + equipment to room floor, creates ash object. Matches C behavior.

#### 8d. CON loss on death — ❌ MISSING
- **C:** `fight.c` line ~604: If level > 5 and `!number(0,3)`, CON decreases by 1. If level > 20 AND `!number(0,5)`, CON decreases by another 1.
- **Go:** Not implemented.
- **Impact:** Death has no permanent stat penalty in Go. This makes death less punishing.

#### 8e. Kill counter rewards — ❌ MISSING
- **C:** `counter_procs()` at milestone kills (1000, 2000, 5000, etc.) gives max HP/mana/move bonuses, full heals, and global blessing.
- **Go:** Not implemented.
- **Impact:** No veteran rewards system.

#### 8f. Group XP — ⚠️ PARTIAL
- **C:** `group_gain()` splits XP among group members, applies `calc_level_diff()` penalty for level gaps, handles gold auto-loot/split.
- **Go:** `AwardMobKillXP()` exists but the full group gain logic isn't visible in death.go.
- **Impact:** Group play XP distribution may not match.

#### 8g. Auto-gold looting — ❌ MISSING
- **C:** `PRF_AUTOGOLD` flag causes automatic gold looting from corpse, with `PRF_AUTOSPLIT` for group distribution.
- **Go:** Not implemented.

#### 8h. Alignment change — ❌ MISSING
- **C:** `change_alignment()` adjusts killer alignment toward victim's opposite.
- **Go:** Not implemented.

#### 8i. PK tracking — ❌ MISSING
- **C:** Tracks PKs, deaths, sets OUTLAW flag for non-outlaw PKs.
- **Go:** Not implemented.

#### 8j. Mob death spec procs/scripts — ❌ MISSING
- **C:** `die_with_killer()` calls mob spec proc and runs "death" Lua script if flagged.
- **Go:** Not implemented.

---

## Summary Table

| System | Status | Critical Issues | Minor Issues |
|--------|--------|----------------|--------------|
| 1. Combat Formulas | ⚠️ MINOR | Position damage multiplier wrong | Missing bless/drunk/race_hate/sanctuary/protect |
| 2. Class & Stats | ❌ SIGNIFICANT | **wis_app table completely wrong** | Mana gain at level 1, immortal conditions |
| 3. Spell System | ❌ NOT IMPLEMENTED | No spells, no saving throws | — |
| 4. Regen & Limits | ⚠️ MINOR | Hunger/thirst direct damage (Go addition) | Missing mob regen, veteran bonuses |
| 5. Affect System | ❌ SIGNIFICANT | No affect_total recalc, no stat clamping | Bitvector handling, duration model |
| 6. Equipment | ⚠️ MINOR | Hitroll/damroll missing spell affects | — |
| 7. Event Queue | ✅ ACCURATE | None | Enhanced fields (deliberate) |
| 8. Death & XP | ⚠️ MINOR | Missing CON loss, kill rewards | Missing auto-gold, alignment, PK tracking |

## Priority Fixes (would produce different in-game behavior)

1. **wis_app table** — Replace with correct C values. This affects every class's practice gains.
2. **Position damage multiplier** — Fix parentheses in `CalculateDamage`. Currently gives wrong damage for sleeping/sitting/resting/stunned targets.
3. **getMinusDam extended entries** — Add missing high-AC entries (-170 through -290).
4. **GetHitroll/GetDamroll** — Include spell-based modifiers, not just equipment.
5. **Affect system stat clamping** — Add min/max bounds after applying affects.
6. **Mana gain at level 1** — Add `if level > 1` guard.

## Phase 3 Items (acceptable TODOs)

These are clearly marked as future work in the Go code and not audit failures:
- Spell system implementation
- Equipment regen bonuses (APPLY_MANA_REGEN, APPLY_HIT_REGEN, APPLY_MOVE_REGEN)
- Veteran bonuses (is_veteran)
- KK_JIN / KK_ZHEN skill bonuses
- Regen room bonuses
- Poison/cutthroat tick damage
- Full group gain / auto-gold / auto-split
- Kill counter rewards
- CON loss on death
- PK system
