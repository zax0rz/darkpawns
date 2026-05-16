# Marathon Audit: Combat, Engine, Spells — 2026-05-15

**Auditor:** Daeron (automated code review)  
**Scope:** `pkg/combat/`, `pkg/engine/`, `pkg/spells/`, `pkg/game/gates.go`  
**Files read:** engine.go, fight_core.go, formulas.go, skill_messages.go, affect_manager.go, affect_tick.go, affect_helpers.go, skill.go, affect.go, affect_spells.go, damage_spells.go, spells.go, call_magic.go, saving_throws.go, spell_info.go, affect_effects.go, gates.go

---

## CRITICAL

### C-010: Spell constant collision — `SpellEnergyDrain` and `SpellDetectPoison` both = 21
**File:** `pkg/spells/spells.go:80,108`  
**What:** `SpellEnergyDrain = 21` and `SpellDetectPoison = 21` share the same constant value.  
**Why:** When `CallMagic` dispatches spell 21, it will match BOTH the `RoutineDamage` branch (energy drain → `MagDamage`) AND the `RoutineManual` branch (detect poison → `ExecuteManualSpell`). The damage routine fires first, dealing energy drain damage to the target. Then the manual routine fires, running `castDetectPoison` on the same target. A caster casting "detect poison" would also deal damage. A caster casting "energy drain" would also trigger detect poison logic.  
**Fix:** Change `SpellDetectPoison` to its correct C constant. In CircleMUD, `SPELL_DETECT_POISON = 21` is correct per `spells.h`, but `SPELL_ENERGY_DRAIN` should be a different value. Verify against `globals.lua` or C `spells.h` — `SPELL_ENERGY_DRAIN` in standard CircleMUD is spell 21, and `SPELL_DETECT_POISON` is also 21 in some forks. The real fix is checking whether Dark Pawns uses both: if detect poison was moved to a different number in the DP customize, update the Go constant. If not, one of these spells shouldn't exist or should share the same routine.

### C-011: Spell constant collision — `SpellDivineInt` and `SpellIntellect` both = 81
**File:** `pkg/spells/spells.go:75-76`  
**What:** `SpellDivineInt = 81` and `SpellIntellect = 81`.  
**Why:** `SpellDivineInt` is routed through `ExecuteManualSpell` → `castDivineInt`. `SpellIntellect` is registered as `RoutineAffects` in the init() table and handled by `MagAffects` → `case SpellIntellect`. Since both map to 81, casting spell 81 would execute BOTH the manual `castDivineInt` AND the `MagAffects` intel buff. The `MagAffects` path would fire because `setupSpellInfo(SpellIntellect, ...)` registers it with `RoutineAffects`. Then `ExecuteManualSpell` would also fire because the `RoutineManual` path in `CallMagic` also matches.  
**Fix:** Verify the correct C constant for each. In DP's `globals.lua`, `SPELL_DIVINE_INT` and `SPELL_INTELLECT` may be the same spell (divine intelligence IS the intellect spell for clerics). If so, remove the duplicate constant and register it under one routine only. If they're different spells, assign different numbers.

---

## HIGH

### H-010: Dual affect systems don't communicate
**File:** `pkg/engine/affect_manager.go` + `pkg/engine/affect_helpers.go`  
**What:** The codebase has TWO independent affect systems that don't know about each other.  
**Why:**  
- **System A (`AffectManager`):** Uses `engine.Affect` structs with `AffectType` iota enum. Used by `MagAffects()` (spell buffs/debuffs) and `ApplySpellAffects()`. Managed by `AffectManager.Tick()` for duration expiry.  
- **System B (`MasterAffect`):** Uses `MasterAffect` structs with integer spell types and `APPLY_*` locations. Used by item consumables, equipment affects, and `AffectTotal()` recalculation. Managed by explicit `AffectRemove()` calls.  

The `Player` struct stores `MasterAffects []*engine.MasterAffect` for System B, and implements `AddAffect(*engine.Affect)` for System A. Neither system cleans up or knows about the other's affects. `AffectTotal()` (System B) doesn't account for System A stat changes. `AffectManager.Tick()` doesn't decrement System B durations.

**Impact:** Equipment affects and spell affects operate on separate state. A bless spell (System A) and a blessed weapon (System B) both modify hitroll, but removing one doesn't affect the other's contribution. If the C code expected a unified affect list (which it does — `master_affected_type` is the single list), this is a port fidelity gap.  
**Fix:** Unify into a single affect system. The `MasterAffect` system in `affect_helpers.go` is closer to C's `master_affected_type`. The `AffectManager` was built as a newer abstraction. Pick one and bridge the other.

### H-011: `AffectManager.GetAffects()` returns internal slice without copy
**File:** `pkg/engine/affect_manager.go:228`  
**What:** `GetAffects()` returns `am.affects[entityID]` directly — the internal slice, not a copy.  
**Why:** Any caller that modifies the returned slice (append, delete, reassign elements) corrupts the internal state without holding the lock. This is a data race if the caller doesn't hold its own synchronization. The method holds `am.mu.RLock` during the return, but the caller can mutate the slice after the lock is released.  
**Fix:** Return a copy: `result := make([]*Affect, len(affects)); copy(result, affects); return result`

### H-012: Spell damage bypasses combat engine damage modifiers
**File:** `pkg/spells/damage_spells.go:inflictDamage()`  
**What:** `inflictDamage()` applies spell damage via direct `SetHP()` call, bypassing `combat.TakeDamage()`.  
**Why:** The combat engine's `TakeDamage()` applies critical modifiers: sanctuary halving, protection evil/good reduction, race hate bonuses, immortality invulnerability, damage caps (3000 max), peaceful room checks, and wimpy flee triggers. None of these apply to spell damage. A spell hitting a sanctuary'd target does full damage. A spell hitting an immortal does full damage (no `dam = 0` override). There's no damage cap on spell damage.  
**Fix:** Route spell damage through the combat engine's `TakeDamage()`, or replicate the modifier chain in `inflictDamage()`. At minimum, add sanctuary check and immortality check.

### H-013: `AffectManager.Tick()` holds write lock while calling entity callbacks
**File:** `pkg/engine/affect_manager.go:236-260`  
**What:** `Tick()` acquires `am.mu.Lock()` and holds it while calling `removeAffectImmediate()` → `entity.ClearStatusFlag()` and `sendAffectMessage()` → `entity.SendMessage()`.  
**Why:** If `SendMessage()` triggers game logic that tries to apply or query affects (e.g., a death message triggers a spell that checks `HasAffect`), the goroutine will deadlock trying to re-acquire `am.mu`. This is a classic lock-ordering issue.  
**Fix:** Collect expired affects under the lock, release the lock, then process removals and messages outside the lock.

### H-014: Saving throw table anomaly — Cleric level 2 PARA worse than level 1
**File:** `pkg/spells/saving_throws.go:CLASS_CLERIC PARA row`  
**What:** Cleric paralysis save values: level 1 = 50, level 2 = 59. Higher value = harder to save, so level 2 clerics save WORSE against paralysis than level 1.  
**Why:** This is likely a data entry error in the C source that was ported verbatim. Every other class shows monotonically improving (decreasing) save values with level. The PARA row for Cleric goes 50→59→48→46... The jump from 50 to 59 at level 2 is anomalous.  
**Fix:** Verify against C source `magic.c` saving throws table. If the C source also has this anomaly, it's a known C bug. If not, fix the Go table to match. The correct value at level 2 is likely 49 (following the decreasing trend).

### H-015: Status flag reference counting missing — two affects setting same bit interfere on removal
**File:** `pkg/engine/affect_manager.go:applyAffectImmediate/removeAffectImmediate`  
**What:** Both `applyAffectImmediate` (SetStatusFlag) and `removeAffectImmediate` (ClearStatusFlag) operate on raw bitmask bits. No reference counting.  
**Why:** If Affect A sets bit 4 (Sanctuary) and Affect B also sets bit 4, removing Affect A clears bit 4 even though Affect B still needs it. In practice, duplicate status affects are rare, but the `AffectManager` allows multiple affects with the same `AffectType` to coexist (stacking is optional and defaults to MaxStacks=1 with empty StackID).  
**Fix:** Either enforce single-instance per AffectType (reject duplicates), or use reference counting on status flag bits.

---

## MEDIUM

### M-010: `inflictDamage` death handler doesn't use combat engine's death flow
**File:** `pkg/spells/damage_spells.go:inflictDamage()`  
**What:** When `inflictDamage` reduces HP to 0, it calls `world.HandleSpellDeath(victim)` — a separate path from the combat engine's `handleDeath()` → `DeathFunc()`.  
**Why:** The combat engine's death flow handles: death messages, corpse creation, experience distribution, kill tracking, PK logging, outlaw flagging, counter procs, attitude loot, autoloot/autogold. The spell death handler likely doesn't replicate all of this. A spell kill may not create a corpse, grant XP, or log the kill.  
**Fix:** Route spell-triggered deaths through the same death pipeline as combat deaths, or verify `HandleSpellDeath` replicates all necessary behavior.

### M-011: `CalculateDamage` double-applies mob damage roll
**File:** `pkg/combat/formulas.go:CalculateDamage()`  
**What:** The function receives `weaponDamage` as a parameter, but for NPCs ignores it and calls `attacker.GetDamageRoll()` instead. The caller in `engine.go:processCombatPair` passes `attacker.GetDamageRoll()` as `weaponDamage`.  
**Why:** For mobs, the damage is: `strApp.ToDam + damroll + RollDice(damRoll.Num, damRoll.Sides) + damRoll.Plus`. The `damRoll` comes from `attacker.GetDamageRoll()` which is the mob's base damage dice. This is correct — the `weaponDamage` parameter is only used for players wielding weapons. But the naming is misleading and the caller wastes a call.  
**Fix:** Clarify the parameter contract with a comment. No behavioral bug, just confusing API.

### M-012: `applyLocFromAffectType` maps HP and MaxHP to same APPLY location
**File:** `pkg/engine/affect_helpers.go:applyLocFromAffectType()`  
**What:** Both `AffectHP` and `AffectMaxHP` map to `ApplyHit` (11). Both `AffectMana` and `AffectMaxMana` map to `ApplyMana` (10).  
**Why:** When `AffectTotal` recalculates via the MasterAffect system, it can't distinguish between a +5 HP buff and a +5 MaxHP buff. Both become `ApplyHit` with modifier +5. The `DefaultApplyModify` for `ApplyHit` calls `addMaxStat(ch, "HP", mod)` — so both affect max HP. But the AffectManager's `applyAffectImmediate` correctly handles them separately (AffectHP → `SetHP`, AffectMaxHP → `SetMaxHP`). This inconsistency means the two systems apply the same spell differently.  
**Fix:** Add `ApplyMaxHit` and `ApplyMaxMana` locations, or document that the MasterAffect system intentionally only handles max-stat affects.

### M-013: `NewAffect` default StackID="" means most affects stack infinitely
**File:** `pkg/engine/affect.go:NewAffect()`  
**What:** Default `StackID` is empty and `MaxStacks` is 1. The stacking check in `AffectManager.ApplyAffect` only triggers when `affect.StackID != ""`.  
**Why:** If you cast Bless twice, two separate Affect objects with empty StackID are added. Neither is removed because the stacking logic only activates for non-empty StackIDs. This means duplicate affects accumulate — the player gets +2 hitroll from two separate bless affects instead of the second one replacing the first.  
**Fix:** Either set a default StackID based on AffectType (e.g., `StackID = "bless"` for AffectHitRoll from bless), or change the default behavior to replace existing affects of the same type.

### M-014: Port fidelity — `castIdentify` explicitly simplified
**File:** `pkg/spells/affect_spells.go:1519`  
**What:** `castIdentifyObject` is marked "Simplified Go implementation" of C `spells.c:476-621`.  
**Why:** The C identify spell handles 145+ lines of type-specific logic including weapon hit/dam apply values, armor AC values, container lock picking, portal destinations, and detailed affect descriptions. The Go version handles basic type info, weight, cost, and a few type-specific details. Missing: detailed weapon stat breakdown, full affect display, container/key/portal specifics.  
**Fix:** Not urgent — identify is informational only. But players will notice missing details compared to C. Track as tech debt.

### M-015: `PositionRecovery` goroutine has no lock coordination with combat state
**File:** `pkg/combat/engine.go:StartMobPositionRecovery()`  
**What:** The position recovery goroutine reads `mob.GetFighting()` and `mob.GetStatus()` without holding any combat engine lock.  
**Why:** A mob could be in the middle of combat (SetFighting called by combat engine) when the recovery goroutine reads an empty fighting string and stands the mob up. The 3-second interval makes this a narrow window, but under high load it's possible. The `SetStatus("standing")` could also interfere with combat position checks.  
**Fix:** Either integrate position recovery into the combat engine's tick, or add a brief mutex check.

### M-016: `CounterProcs` reproduces C fall-through bug intentionally
**File:** `pkg/combat/fight_core.go:CounterProcs()`  
**What:** The major milestone cases (1000, 2000, etc.) intentionally reproduce C's missing `break` statements, granting +1 HP, +1 MANA, AND +1 MOVE for every major milestone.  
**Why:** The comment says "Reproducing the bug for fidelity." In C, the switch falls through all three stat increases. This is intentional but means every major milestone gives all three stats instead of one random stat. Players at kill milestones get triple the stat benefit the developer likely intended.  
**Fix:** Confirm this is desired behavior with The Architect. If the C bug was unintentional, fix it to pick one random stat.

### M-017: `AffectManager.GetEntityID` uses name + ID concatenation
**File:** `pkg/engine/affect_manager.go:getEntityID()`  
**What:** Entity IDs are `"name_id"` strings. If two entities have the same name and ID (e.g., after a mob respawn with the same VNUM), they share affect state.  
**Why:** Mobs are identified by name + VNUM. If a mob dies and respawns with the same VNUM and name, the new instance inherits the old instance's affects from the AffectManager.  
**Fix:** Use a truly unique identifier (pointer address, UUID, or generation counter).

---

## LOW

### L-010: `backstabMult` uses floating-point multiplication for damage
**File:** `pkg/combat/fight_core.go:backstabMult()`  
**What:** `dam = int(float64(dam) * backstabMult(ch.GetLevel()))` — float64 multiplication truncated to int.  
**Why:** C uses integer math. `int(float64(10) * 5.0)` = 50, which matches C. But for large damage values, float64 precision loss could cause off-by-one differences vs C's integer multiplication.  
**Fix:** Use integer arithmetic: `dam = dam * (level*2 + 10) / 10` for equivalent result without float.

### L-011: `IsInGroup` always returns false for non-empty names
**File:** `pkg/combat/fight_core.go:IsInGroup()`  
**What:** The first branch checks `ch.GetName() == ""` which is never true for real characters. This means the `GetFollowersInRoom` path is dead code.  
**Why:** `NewNamedCombatant` always sets a non-empty name. The only way `GetName()` returns "" is if the caller creates a combatant with an empty name, which shouldn't happen.  
**Fix:** Remove the dead branch or clarify the intent.

### L-012: `ChangeAlignment` uses right-shift on potentially negative values
**File:** `pkg/combat/fight_core.go:ChangeAlignment()`  
**What:** `newAlign := killerAlign + (-victimAlign-killerAlign)>>4`  
**Why:** Go's `>>` on signed integers is arithmetic (sign-extending), matching C. For negative values, `>>4` rounds toward negative infinity, not toward zero. C89 left this implementation-defined, but modern C and Go agree. This is correct but worth noting for portability.  
**Fix:** No action needed.

### L-013: `MagAffects` creates Affects but some cases call `applyAffect` early
**File:** `pkg/spells/affect_spells.go:MagAffects()`  
**What:** Most spell cases set `aff = engine.NewAffect(...)` and fall through to the final `applyAffect(victim, aff)`. But `SpellBless`, `SpellProtFromEvil`, `SpellProtFromGood`, `SpellInvulnerability`, `SpellGreatPercept`, `SpellLessPercept` call `applyAffect` mid-case AND set `aff` for the final call.  
**Why:** `SpellBless` applies two separate affects (hitroll + saving throw) — the first via mid-case `applyAffect`, the second via the fall-through. `SpellProtFromEvil` applies the affect mid-case AND returns, skipping the fall-through. This is correct but the control flow is confusing. `SpellInvulnerability` applies the AC affect mid-case, then sets `aff` for the saving spell affect — both apply.  
**Fix:** Add comments explaining the dual-apply pattern. No behavioral bug.

### L-014: `gates.go` `RawKill` uses string attack type instead of int
**File:** `pkg/game/gates.go:RawKill()`  
**What:** `w.RawKill(ch, "suffering")` passes a string, but `w.rawKill(ch, at)` expects an int.  
**Why:** The `RawKill` method converts the string to an int via a switch. `TYPE_SUFFERING = 399`. This works but is inconsistent with the combat engine's integer-based attack types.  
**Fix:** Pass the int constant directly: `w.rawKill(ch, 399)`.

### L-015: `skill_messages.go` `basicTokenReplace` replaces `$M` with attacker's objective pronoun
**File:** `pkg/combat/skill_messages.go:basicTokenReplace()`  
**What:** `$M` is replaced with `chPronouns.objective` (attacker's him/her/it). In CircleMUD, `$M` is the victim's objective pronoun (him/her/it), and `$m` is the attacker's.  
**Why:** The replacement maps `$m` → attacker objective, `$M` → attacker objective. But in C `act()`, `$m` is victim objective and `$M` is attacker objective (or vice versa depending on the act() implementation). The skill messages in this file use `$M` to refer to the victim (e.g., "$n slashes $M"), but the replacement treats `$M` as attacker pronoun. This means "$n slashes $M" would produce "$n slashes him" when the ATTACKER is male, not when the VICTIM is male.  
**Fix:** Verify the C `act()` token mapping. In standard CircleMUD: `$n`=attacker name, `$N`=victim name, `$e`=attacker subjective, `$E`=victim subjective, `$m`=attacker objective, `$M`=victim objective, `$s`=attacker possessive, `$S`=victim possessive. The current code maps `$M` to attacker objective, which is wrong if the messages expect victim objective.

---

## Summary

| Severity | Count | Key Themes |
|----------|-------|------------|
| CRITICAL | 2 | Spell constant collisions (shared IDs cause dual dispatch) |
| HIGH | 6 | Dual affect systems, bypassed combat modifiers, lock ordering, data races |
| MEDIUM | 8 | Death flow gaps, stacking defaults, port fidelity simplifications |
| LOW | 6 | Float math, dead code, token replacement order |

**Top 3 priorities:**
1. **C-010/C-011** — Fix spell constant collisions immediately. These cause incorrect spell dispatch behavior in production.
2. **H-013** — Fix `AffectManager.Tick()` lock ordering before it causes a server deadlock.
3. **H-010** — Plan the unified affect system migration. The dual-system architecture is the root cause of many medium-severity inconsistencies.
