# Audit Report: fight.c vs pkg/combat/ & Go Combat Helpers

**C file:** `src/fight.c` (2,034 lines)
**Go file(s):** `pkg/combat/formulas.go` (710 lines), `pkg/combat/engine.go` (466 lines), `pkg/game/death.go` (791 lines), `pkg/game/deferred_fight_fns.go` (374 lines), `pkg/game/party.go` (217 lines)
**Mapping type:** 1:N
**Functions audited:** 26

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Complete Omission of Experience Level-Difference Penalty
- **Location:** `pkg/game/party.go` in `AwardMobKillXP()` and `pkg/game/death.go` in `HandleDeath()`.
- **C behavior:** In legacy C, experience gained from killing a mob is scaled by calling `calc_level_diff(ch, victim, base)` (defined in `fight.c:660`). This penalizes experience awards:
  - By 30% if the player is >5 levels higher than the victim.
  - By 50% if the player is >10 levels higher than the victim.
  - By 70% if the player is >15 levels higher than the victim.
  - By an additional 20% if the player's level is above 20.
- **Go behavior:** The Go codebase has absolutely no implementation of `calc_level_diff()` or its logic. Experience is awarded at 100% value to both solo players and group members in `AwardMobKillXP()`.
- **Discrepancy:** Players can power-level instantly and exploit the MUD economy by continuously farming extremely low-level monsters, as no over-level or level-difference penalties are applied to their experience gain. This breaks the core progression balancing of the MUD.
- **Severity:** CRITICAL
- **Type:** DRIFT

### [FINDING-002]: Tattoos Apply Zero Stat Bonuses (`TattooAf` is a Stub)
- **Location:** `pkg/game/deferred_fight_fns.go:354` in `TattooAf()`.
- **C behavior:** In `tattoo.c:104`, `tattoo_af(ch, add)` maps the player's active tattoo (e.g. Dragon, Tribal, Spider, Skull) to specific stat modifications (e.g. +2 Strength/Damroll for Dragon, +3 Dex for Spider) and applies/removes them from the character struct by adding/joining them as spells/affects.
- **Go behavior:** Go's `TattooAf()` is a complete stub. It retrieves the bonuses but assigns them to blank identifiers (`_ = bonuses`, `_ = add`) and returns without applying any modifiers or affects to the player.
- **Discrepancy:** Player tattoos are purely cosmetic in Go; they fail to apply any of the mechanical stat, hitroll, damroll, or mana bonuses designed in the legacy game.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-003]: Parry and Dodge Mechanics Mismatch
- **Location:** `pkg/combat/engine.go:275` in `processCombatPair()`.
- **C behavior:** In `fight.c:1949` and `1965`, successfully parrying/dodging sets the *opponent's* `IS_PARRIED` flag to `TRUE`. When the opponent's turn to attack arrives during the violence cycle, their total attack rolls (number of attacks) for that round is decremented by 1 (or scaled by Dexterity if negative).
- **Go behavior:** In Go, parry/dodge are evaluated inside the individual attack loops. If a check succeeds, it simply discards that single hit's damage via `continue`.
- **Discrepancy:** In Go, parry/dodge only cancels one individual hit. In C, a single successful defensive parry or dodge dampens the opponent's entire martial presence for that round, decreasing their overall capacity to execute multiple attacks.
- **Severity:** HIGH
- **Type:** DRIFT

### [FINDING-004]: `counter_procs` Missing (Kills Milestone Blessings Disabled)
- **Location:** `pkg/game/death.go` (entire file).
- **C behavior:** In C `fight.c:1251`, `counter_procs(ch)` is invoked on every kill. If a player reaches a milestone number of kills (1000, 2000, 5000, 10000, etc.), the gods reward them with permanent stat increases (Max HP, Max Mana, Max Move) and trigger a global message healing all active players in the MUD to full.
- **Go behavior:** Go's `handleMobDeath` and `handlePlayerDeath` do not invoke any equivalent to `counter_procs()`. Kills are incremented, but milestone rewards, permanent stat growth, and global blessings never trigger.
- **Discrepancy:** Players are deprived of permanent end-game stat milestones and the MUD loses its classic cooperative "kill milestone blessings" event.
- **Severity:** MEDIUM
- **Type:** STUB

### [FINDING-005]: Shopkeeper Protection Missing in Combat Engine Ticks
- **Location:** `pkg/combat/engine.go` in `processCombatPair()`.
- **C behavior:** In `fight.c:1360`, any attempt to damage a shopkeeper immediately halts combat on both sides (`stop_fighting(ch)`, `stop_fighting(victim)`) and returns `FALSE`.
- **Go behavior:** Go's `processCombatPair()` lacks any shopkeeper protection check. If a fight is initiated with a shopkeeper, combat will proceed until the shopkeeper or player dies, unless blocked at the command parser layer.
- **Discrepancy:** Potential for shopkeepers to be killed or lock players in unintended death loops.
- **Severity:** MEDIUM
- **Type:** DRIFT

---

## Type & Boundary Vulnerabilities

### [FINDING-006]: Gender Pronoun Mapping Inversion
- **Location:** `pkg/game/death.go:663` in `genderPronoun()`.
- **C behavior:** The `HSHR(ch)` macro in `src/utils.h:505` defines:
  - `GET_SEX(ch) == SEX_MALE` (1) → `"his"`
  - `GET_SEX(ch) == SEX_FEMALE` (2) → `"her"`
  - `GET_SEX(ch) == SEX_NEUTRAL` (0) → `"its"`
- **Go behavior:** Go's `genderPronoun()` switch maps:
  - `1` → `"her"`
  - `2` → `"its"`
  - Default (0) → `"his"`
- **Risk:** Type/Constant mapping inversion. Male characters/mobs are referred to as `"her"`, females as `"its"`, and neutral entities as `"his"`. This leads to extremely broken corpse descriptions (e.g., "the corpse of Zach is lying here, her neck snapped in two").
- **Severity:** HIGH

### [FINDING-007]: Position Damage Multipliers Truncation
- **Location:** `pkg/combat/formulas.go:486` in `CalculateDamage()`.
- **C behavior:** In C `fight.c:1859`, victim position damage multiplier is calculated as `dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3`. Because this is integer math:
  - `POS_SITTING` (6): `1 + (7-6)/3 = 1` (no change).
  - `POS_RESTING` (5): `1 + (7-5)/3 = 1` (no change).
  - `POS_SLEEPING` (4): `1 + (7-4)/3 = 2` (x2 damage).
  - `POS_STUNNED` (3): `1 + (7-3)/3 = 2` (x2 damage).
  - `POS_DEAD` (0): `1 + (7-0)/3 = 3` (x3 damage).
- **Go behavior:** Go uses the exact same integer division formula: `dam *= 1 + delta/3`.
- **Risk:** No logical risk (it matches C exactly), but the developer comments in both C and Go claim that sitting gives `x1.33` and resting gives `x1.66` damage. In reality, Go's integer division truncates `1/3` and `2/3` to `0`, meaning sitting/resting players receive *zero* extra damage, while sleeping/stunned receive *exactly* `x2`. If the intent is to match the *documented* behavior rather than the buggy legacy C implementation, float math should be used.
- **Severity:** LOW

---

## Control Flow & Mathematical Fidelity

### [FINDING-008]: Inconsistent Haste/Slow Attack Round Boundaries
- **Location:** `pkg/combat/formulas.go:587` in `GetAttacksPerRound()`.
- **C behavior:** In C `fight.c:1920-1922` and `1943-1946`, `attacks++` for haste and `attacks--` for slow are applied. If the final attack count drops below zero, it is capped: `if (attacks < 0) attacks = 0;`. This means slow can reduce a character's attacks to 0 for a round.
- **Go behavior:** Go caps the minimum attacks per round at 1: `if attacks < 1 { attacks = 1 }`.
- **Impact:** Slowed players in Go will always get at least 1 attack per round, whereas in legacy C they could have their round completely skipped (0 attacks), making slow significantly weaker in the Go port.
- **Severity:** MEDIUM

---

## Concurrency & Mutex Safety

### [FINDING-009]: Potential Data Race on Player Experience and Gold Mutators
- **Location:** `pkg/game/party.go` in `AwardMobKillXP()`.
- **C behavior:** Strictly single-threaded synchronous loop; no concurrency safety required.
- **Go behavior:** `AwardMobKillXP()` modifies player stats via `m.AddExp(base)` and `m.SetGold(...)` in concurrent session/combat tickers without acquiring the player's individual mutex lock (`m.mu`).
- **Impact:** Calling mutators across different goroutines concurrently (e.g. session reader inputting `sell` or `practice` while combat ticker calls `AwardMobKillXP`) can lead to partial writes, memory corruption, or race conditions on the `Player` structure.
- **Severity:** HIGH

---

## Unported Functions

The following legacy C functions from `fight.c` have no equivalent behavioral Go implementation in `pkg/combat/` or `pkg/game/`:

| C Function | Line | Description | Ported? |
|------------|------|-------------|---------|
| `appear` | 107 | Removes invisibility and hide flags, sending room message. (In Go, this is done in-line in `TakeDamage` but lacks the standard room message lookups). | PARTIAL |
| `load_messages` | 125 | Reads the legacy MUD `MESS_FILE` (combat messages) from disk into a linked list. (Go uses a static structure/JSON file). | NO |

---

## Summary

- **Total findings:** 9
- **Critical:** 1
- **High:** 4
- **Medium:** 3
- **Low:** 1
- **Unported functions:** 2
