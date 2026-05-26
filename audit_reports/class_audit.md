# Port Fidelity Audit: Module 18 (`class.c`)

This audit examines the port fidelity between the legacy C source file `src/class.c` and its Go implementations in `pkg/game/` and `pkg/combat/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/class.c` (1,191 lines)
- **Functions**: `parse_class`, `find_class_bitvector`, `roll_real_abils`, `do_start`, `advance_level`, `backstab_mult`, `invalid_class`, `init_spell_levels`, `find_exp`, `exp_needed_for_level`.

### Go Port Files
To optimize package organization, the original `class.c` logic has been distributed across multiple domain-focused packages and files:
- `pkg/game/class_tables.go` (Defines `PCClassTypes`, `PracParams`, `GuildInfo`, `ParseClass`, `FindClassBitvector`, `InvalidClass`)
- `pkg/game/character.go` (Defines Class/Race constants, `ClassAbbrevs`, `ClassNames`, `RaceNames`, `CharStats`, `RollRealAbils`, `ValidUserClassChoice`, `DoStart`, `GiveStartingSkills`)
- `pkg/game/level.go` (Implements `AdvanceLevel`, `conApp`, `wisApp` tables)
- `pkg/game/limits_exp.go` (Implements `FindExp`, `ExpNeededForLevel`, `GainExp`, `GainExpRegardless`)
- `pkg/combat/fight_core.go` (Implements a package-local copy of `backstabMult` and references local `LVL_IMMORT` constants)
- `pkg/game/skill_combat.go` (Implements a duplicate copy of `backstabMult`)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Split-Brain Level Scale Discrepancy (Severe Security/Gameplay Exploit)
- **Source Context**: `pkg/game/limits.go#L20-L24`, `pkg/session/wizard_cmds.go#L10-L15`, `pkg/combat/fight_core.go#L115`
- **Fidelity Bug**: The codebase has a major architectural mismatch regarding wizard levels.
  - In `pkg/game/` and `pkg/combat/` (core engine/mechanics):
    `LVL_IMMORT` is defined as `31`.
    `LVL_IMPL` is defined as `40`.
  - In `pkg/session/` (command interpreter/privilege gates):
    `LVL_IMMORT` is defined as `50`.
    `LVL_IMPL` is defined as `61`.
- **Impact**:
  - **Mortal Level Cap**: In `GainExp` in `limits_exp.go`, characters level up only if `p.Level < LVL_IMPL-1`. Since `LVL_IMPL` is `40` in the `game` package, players are hard-capped at level `38` under normal gameplay XP progression.
  - **Split-Brain Behavior (Levels 31-49)**: If a character's level falls between `31` and `49`:
    1. The command interpreter in `pkg/session/` treats them as a **mortal** player (since `Level < 50`), completely restricting access to wizard commands.
    2. However, the game engine and combat formula in `pkg/game/` and `pkg/combat/` treat them as a **fully privileged immortal** (since `Level >= 31`). As a result, they suffer no hunger or thirst (conditions set to `-1` in `level.go`), get automatic holy light (`PRF_HOLYLIGHT`), bypass aggressive mobs and mob hunting memory targets (`ch.GetLevel() < LVL_IMMORT` checks in `fight_core.go`), and their backstab multiplier is capped at the immortal max of `20.0x`.
  This mismatch allows a player manually set to level `31` to gain complete immortal immunities while staying categorized as a mortal by the command router.

### 2. Slashing/Shield Equipment Constraints are Always Disabled
- **Source Context**: `pkg/game/equipment.go#L193` (`EquipForPlayer`), `pkg/game/class_tables.go#L178` (`InvalidClass`)
- **Fidelity Bug**: In `equipment.go`, the item-wear validation routing passes hardcoded `false` values for weapon category and shield flags when calling class validation:
  ```go
  if InvalidClass(class, uint32(xf), false, false) {
  ```
- **Impact**:
  - **Slashing Clerics Allowed**: Clerics are strictly prohibited from wielding slashing weapons in CircleMUD. Because `isWieldedSlashWeapon` is hardcoded to `false`, clerics can freely equip and use slashing weapons with no penalties.
  - **Thief/Assassin/Ninja Shields Allowed**: Thieves and stealth classes are prohibited from equipping shields. Hardcoding `isShield` to `false` permits all thief sub-classes to equip shields, completely breaking gameplay balancing.

### 3. Duplicate and Inconsistent `backstabMult` Implementations
- **Source Context**: `pkg/combat/fight_core.go#L1130` and `pkg/game/skill_combat.go#L72`
- **Fidelity Bug**: The backstab damage multiplier formula is copy-pasted in two different packages:
  - `pkg/combat/fight_core.go`: Uses package-local `LVL_IMMORT` (which is `31`).
  - `pkg/game/skill_combat.go`: Uses a hardcoded literal `31` (instead of the `Limits` constant).
- **Impact**: If the immortal scale is ever consolidated or refactored, having duplicate copies of core combat math increases technical debt and introduces high risks of math desynchronization between basic combat skills and secondary attacks.

---

## 3. Go Improvements Over C

### 1. Robust Ability Generation
- **Fidelity Improvement**: Go's `RollRealAbils` utilizes bubble-sorting during descending insertion, which is far cleaner and more performant than the legacy C nested bit-flip XOR operations (`temp ^= table[k]`, `table[k] ^= temp`).

### 2. Centralized Race and Class Selectors
- **Fidelity Improvement**: Character creation menu routing has been organized into clear, type-safe Go constant maps (`ClassNames`, `RaceNames`) and validation guards (`ValidUserClassChoice`), replacing the fragile Diku-style null-terminated arrays and character-comparison structures.

---

## 4. Concurrency & Thread Safety

- **Save-During-Level-Up Lock Release**:
  - In `AdvanceLevel` (`pkg/game/level.go`), the player level-up procedure releases the player state lock `p.mu.Unlock()` before calling `SavePlayer(p)` to avoid deadlocking:
    ```go
    // Release lock before I/O — SavePlayer acquires RLock via playerToSaveData.
    level := p.Level
    name := p.Name
    p.mu.Unlock()

    if err := SavePlayer(p); err != nil {
        ...
    }
    ```
  - This is an excellent, thread-safe practice. However, since the lock is released prior to persisting to disk, there is a tiny race-window where a concurrent command could check the player's level (getting the incremented value) before the new stats are fully written.

---

## 5. Summary of Recommended Fixes

1. **Consolidate Split-Brain Level Scale Constants**:
   - Establish a single, global source of truth for MUD levels (e.g. `LVL_IMMORT`, `LVL_GOD`, `LVL_IMPL`) inside a shared package.
   - Refactor `pkg/game/`, `pkg/combat/`, and `pkg/session/` to import this shared constant so that mortal caps and immortal immunities align perfectly across the entire codebase.
2. **Properly Wire Slashing and Shield Checks in `equipment.go`**:
   - In `EquipForPlayer`, extract the weapon category and shield state from the `ObjectInstance` instead of hardcoding `false, false`:
     ```go
     isShield := item.IsArmor() && wearFlagsContains(item, FlagWearShield)
     isSlash := item.IsWeapon() && item.Prototype.Values[3] == (TypeSlash - TypeHit)
     if InvalidClass(class, uint32(xf), isSlash, isShield) {
     ```
3. **De-duplicate `backstabMult`**:
   - Export `BackstabMult` from a shared package (e.g. `pkg/combat/` or `pkg/game/`) and import it inside combat skill handlers rather than maintaining twin copies of the formula.
