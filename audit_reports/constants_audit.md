# Port Fidelity Audit: Module 21 (`constants.c`)

This audit examines the port fidelity between the legacy C source file `src/constants.c` (global game constants, array descriptors, and strength/dexterity application tables) and its Go implementations in `pkg/game/` and `pkg/session/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/constants.c` (1,451 lines)
- **Constant Arrays**:
  - `phases`, `hometowns`, `abil_names`, `crowd_size`, `dirs`, `races`, `mob_races`, `tattoos`
  - `race_menu`, `race_help`, `help_human`, `help_dwarf`, `help_elf`, `help_kender`, `help_minotaur`, `help_rakshasa`, `help_ssaur`
  - `intelligent_races`, `room_bits`, `exit_bits`, `sector_types`, `genders`, `position_types`
  - `player_bits`, `action_bits`, `preference_bits`, `mscript_bits`, `rscript_bits`, `oscript_bits`
  - `affected_bits`, `affected_names`, `connected_types`, `where`, `equipment_types`, `item_types`
  - `wear_bits`, `extra_bits`, `apply_types`, `container_bits`, `drinks`, `drinknames`, `drink_aff`, `color_liquid`, `fullness`
  - `str_app`, `dex_app_skill`, `dex_app`, `con_app`, `int_app`, `wis_app`
  - `spell_wear_off_msg`, `npc_class_types`, `rev_dir`, `movement_loss`, `weekdays`, `month_name`
  - `sharp`, `tat` (tattoo definitions), `field_objs`

### Go Port Files
- `pkg/game/constants.go` (Central definitions of game strings, sector/affected/position tables, moon phases, month names, and reverse-directions)
- `pkg/game/liquids.go` (Defines structured `Liquid` objects, color tables, and thirst/hunger/drunk properties)
- `pkg/game/player.go` (Hardcodes local approximations of the `str_app` carrying and wielding capacity tables)
- `pkg/session/tattoo.go` (Hardcodes local switch statements for tattoo descriptors and stat modifiers)
- `pkg/game/spec_procs3.go` (Implements `field_objs` properties inside the `specFieldObject` procedurals)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Out-of-Bounds Undefined Apply Types (Regeneration Stats UI Bug)
- **Source Context**: `pkg/game/constants.go#L500-L526` (`ApplyTypeNames`), `pkg/game/limits.go#L14-L17` (`ApplyHitRegen`)
- **Fidelity Bug**: The Go linter/compiler tracks equipment regeneration modifiers via index constants:
  ```go
  ApplyHitRegen  = 26
  ApplyManaRegen = 27
  ApplyMoveRegen = 28
  ```
  However, the global description array `ApplyTypeNames` only has **25 elements** (indexes 0 to 24, ending at `SAVING_SPELL`).
- **Impact**: When players use the `stat` or `examine` commands on high-end items that bestow `HIT_REGEN` or `MANA_REGEN`, the system queries `Sprinttype(apply, ApplyTypeNames)`. Because index 26/27 is out of bounds, the call safely but confusingly returns `"UNDEFINED"`. Players cannot see what regeneration statistics their gear provides.

### 2. Locked Out Exceptional Strength (Warrior Wield Weight Nerfed)
- **Source Context**: `pkg/game/player.go#L390-L408` (`MaxWieldWeight`)
- **Fidelity Bug**: Legacy C accommodates warrior exceptional strength (e.g. `18/100`) by reading `str_app[18]..str_app[22]` (and `18/01-50` mapping to virtual indexes 26 to 30).
  The Go port has a hardcoded array matching this Virtual Index map:
  ```go
  strWield := [...]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 18, 20, ... 22, 24, 26, 28, 30}
  ```
  However, the function evaluates:
  ```go
  str := p.Strength
  if str >= len(strWield) { ... }
  return strWield[str]
  ```
  Since `p.Strength` is an integer clamped to `18` (except for spells/enchantments), and the code **fails to check `p.Stats.StrAdd` (exceptional strength)**, a warrior with `18/100` strength will always return `strWield[18] = 20` lbs limit. The virtual indexes `26..30` are dead code.
- **Impact**: Warriors with `18/100` strength are locked out of their higher wielding capacity, limiting them to 20 lbs instead of the designed 30 lbs limit.

### 3. Truncated Carrying Capacity (High Strength Nerfed 7x)
- **Source Context**: `pkg/game/player.go#L373-L388` (`MaxCarryWeight`)
- **Fidelity Bug**: In `src/constants.c`, strength carrying limits are defined up to STR 25 (e.g. STR 25 carries 1,750 lbs). 
  The Go port's `MaxCarryWeight` array has only **19 elements** (indexes 0 to 18):
  ```go
  strCarry := [...]int{0, 3, 3, 10, 25, 55, 80, 90, 100, 100, 115, 115, 140, 140, 170, 170, 195, 220, 255}
  ```
  If a character has their strength elevated to 25 (common for Minotaurs, Giants, or magically buffed characters), the function clamps them to index 18 (`255` lbs).
- **Impact**: Strong or buffed characters are capped at carrying 255 lbs—a massive **7x nerf** compared to the designed 1,750 lbs capacity.

### 4. Hydration & Nutrition Mappings Broken (Chronic In-Game Starvation)
- **Source Context**: `pkg/game/liquids.go#L38-L55` (`Liquids`)
- **Fidelity Bug**: In `src/constants.c`, `drink_aff[][3]` maps liquid nutrition values. For water: `drunkenness = 0`, `hunger = 1`, `thirst = 10`.
  The Go port's liquid mapping table severely drifts:
  ```go
  LiqWater: {Name: "water", DrunkAffect: 0, FullAffect: 0, ThirstAffect: 1}
  LiqBeer:  {Name: "beer",  DrunkAffect: 3, FullAffect: 2, ThirstAffect: 2}  // C is thirst=5
  ```
- **Impact**: Water is **10 times less effective** at quenching thirst, and beer is **2.5 times less effective**. This severe configuration drift explains why players suffer from constant, aggressive hunger and thirst messages during gameplay.

### 5. Massive Volume of Dead Code and Mismatched Arrays
- **Hometowns Mismatch**: `constants.go` defines a list of **Dragonlance** hometowns (`"Kalaman"`, `"Solace"`, `"Port Storm"`), which is completely dead code. `char_creation.go` hardcodes the actual cities (`"Kir Drax'in"`, `"Kir-Oshi"`, `"Alaozar"`).
- **Races Mismatch**: `MobRaceNames` has **50 entries** including multiple duplicates (e.g. `Dwarf` and `Elf` appear twice at different offsets), whereas `constants.c` defines a clean 31-entry race list.
- **Tattoos Bypassed**: The tattoo descriptors and stats table `tat[]` is completely omitted in Go. Instead, `tattoo.go` hardcodes these properties inside large switch statements.

---

## 3. Go Improvements Over C

### 1. Unified Object Modeling for Liquids
- **Fidelity Improvement**: In legacy C, liquid properties were scattered across four separate, index-aligned arrays (`drinks[]`, `drinknames[]`, `drink_aff[][]`, `color_liquid[]`), which made adding new drinks extremely prone to indexing errors. Go encapsulates this cleanly into a single, cohesive `Liquid` struct array (`Liquids`), improving maintainability.

### 2. Streamlined Connected States
- **Fidelity Improvement**: Go refactors the convoluted, 32-state C nanny state router (`connected_types[]` in C) down to a streamlined 15-state layout (`ConnectedTypeNames` in Go), improving session handshakes and client state tracking.

---

## 4. Summary of Configuration Mismatches

The following table summarizes the constant string discrepancies and limit arrays between C and Go:

| Constant Array | C Behavior | Go Behavior | Status / Impact |
|---|---|---|---|
| `phases` | Descriptive: `"half full(waxing)"` | Modern: `"First Quarter"` | Moon phase strings differ. |
| `hometowns` | Kir Drax'in, Kir-Oshi, Alaozar | Dragonlance (Kalaman, Solace...) | **Dead Code**: Mismatched array is unused. |
| `abil_names` | Quality descriptors: `"average"` | Stat names: `"Strength"`, `"Dexterity"` | Table represents completely different properties. |
| `mob_races` | Clean 31-entry list | 50-entry list with duplicate names | **Dead Code**: Mismatched array is unused. |
| `drink_aff[Water]` | Thirst recovery = `10` | Thirst recovery = `1` | **Gameplay Balance**: Quenching thirst is 10x harder in Go. |
| `str_app[STR 25]` | Carry weight = `1750` | Carry weight = `255` (clamped) | **Gameplay Balance**: Strong characters have 7x less capacity. |
| `apply_types` | 30 types (includes `HIT_REGEN`) | 25 types (omits `HIT_REGEN` etc.) | **Severe Bug**: regeneration stats print as `"UNDEFINED"`. |
| `tat[]` | Defined struct array | Hardcoded switches in `tattoo.go` | Functional equivalent, but structural drift. |

---

## 5. Summary of Recommended Fixes

1. **Fix `ApplyTypeNames` Out-of-Bounds Bug**:
   Append the missing regeneration flags to `ApplyTypeNames` in `pkg/game/constants.go`:
   ```go
   var ApplyTypeNames = []string{
       ...
       "SAVING_BREATH",
       "SAVING_SPELL",
       "RACE_HATE",
       "HIT_REGEN",
       "MANA_REGEN",
       "MOVE_REGEN",
       "PERM_SPELL",
   }
   ```
2. **Correct exceptional Strength in `MaxWieldWeight`**:
   Refactor `MaxWieldWeight` in `pkg/game/player.go` to compute warrior virtual index:
   ```go
   str := p.Strength
   if str == 18 && p.Stats.StrAdd > 0 {
       strAdd := p.Stats.StrAdd
       if strAdd <= 50 { str = 26 }
       else if strAdd <= 75 { str = 27 }
       else if strAdd <= 90 { str = 28 }
       else if strAdd <= 99 { str = 29 }
       else { str = 30 }
   }
   ```
3. **Restore Strength-based Carrying Capacity Limits**:
   Expand `strCarry` in `pkg/game/player.go` to match the full C `str_app` carrying capacity limit values up to STR 25:
   ```go
   strCarry := [...]int{
       0, 3, 3, 10, 25, 55, 80, 90, 100, 100,
       115, 115, 140, 140, 170, 170, 195, 220, 255,
       640, 700, 810, 970, 1130, 1440, 1750,
   }
   ```
4. **Fix Liquid Nutrition & Thirst Factors**:
   Correct the thirst and hunger values in `pkg/game/liquids.go` to mirror `constants.c` values (e.g., Water: Thirst recovery `10`, Hunger recovery `1`).
5. **Clean up unused Global Arrays**:
   Safely remove the unused `Hometowns`, `MobRaceNames`, and `IntelligentRaces` arrays in `pkg/game/constants.go` to reduce compiled dead-code footprints.
