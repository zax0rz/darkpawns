# Port Fidelity Audit: Module 32 (`magic.c`)

This audit examines the port fidelity between the legacy C source file `src/magic.c` and its Go counterparts in `pkg/spells/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/magic.c` (2,000 lines)
- **Functions**: 
  - `mag_savingthrow`: Evaluates d100 rolls against character stats and gear bonuses to check for magic saves.
  - `mag_materials`: Checks and extracts spell reagents (obsidian shards, ashes, beholder eyes).
  - `mag_damage`: Inflicts spell damage, handles backfire chances, and scales average damage based on levels.
  - `mag_affects` & `perform_mag_groups` & `mag_groups`: Apply spell affects (durations, modifiers) to single targets or group followers.
  - `mag_summons`: Summons pets and animated zombies from room corpses.
  - `mag_points`: Directly restores character pool points (HP, Mana, Move).
  - `mag_unaffects`: Dispels or clears specific adverse affects (blindness, poison, curse).
  - `mag_alter_objs` & `mag_creations`: Spawn or modify items (such as creating food/water or enchanting weapons).

### Go Port Files
- **Unified Spells System**:
  - [spells.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/spells.go): Defines spell constants and central `Cast()` entry point.
  - [call_magic.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/call_magic.go): central spell dispatcher (`CallMagic`), position validations, and room flags checks.
  - [saving_throws.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/saving_throws.go): Verbatim saving throws table mapping class/level/type, plus `CheckSavingThrow` d100 rolls.
  - [damage_spells.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/damage_spells.go): Implements `MagDamage` scaling formulas, backfire logic, and zone warning broadcasts.
  - [affect_spells.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/affect_spells.go): Massive template file implementing `MagAffects`, `MagPoints`, `MagUnaffects`, `MagGroups`, `MagSummons`, and `ExecuteManualSpell` dispatches.
  - [say_spell.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/say_spell.go): Implements `ObfuscateSpellName` syllables translation for verbal spell chants.

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. `SpellAnimateDead` Zombie Spawn Out of Thin Air (Critical Bug)
- **Source Context**: `src/magic.c#L1693` (C corpse validation), `pkg/spells/affect_spells.go#L486` (`MagSummons`).
- **Fidelity Bug**: In legacy C, `mag_summons` checks that the spell target (`obj`) is a valid container with a corpse flag (`GET_OBJ_TYPE(obj) == ITEM_CONTAINER && GET_OBJ_VAL(obj, 3)`). If not, the spell fails immediately. If successful, it animates a zombie follower, moves all equipment/inventory inside the corpse to the zombie, and extracts (deletes) the corpse.
- **Impact**: In Go's `MagSummons`, the corpse checking and extraction logic is completely bypassed. Casting `SpellAnimateDead` immediately spawns a zombie follower in the room out of thin air, with no corpse targeted, no items transferred, and no corpse consumed. This permits infinite zombie army creation without any materials.

### 2. `SpellCreateFood` Item Placement Discrepancy (Behavior Gap)
- **Source Context**: `src/magic.c#L1996` (C target container), `pkg/spells/affect_spells.go#L533` (`MagCreations`).
- **Fidelity Bug**: In legacy C, `mag_creations` reads the magic mushroom prototype and places the created food item directly into the caster's inventory bags (`obj_to_char(tobj, ch)`).
- **Impact**: In Go, `MagCreations` spawns the spawned magic mushrooms directly on the room floor (`w.SpawnObject(8062, roomVNum)`), requiring the player to perform a separate `"get mushrooms"` command.

### 3. Reachable Semicolon / Fallthrough Fixes in Go
- **Source Context**: `src/magic.c#L1445` (C missing break), `pkg/spells/affect_spells.go#L350` (Go switch block).
- **Fidelity Improvement**: In C's `perform_mag_groups`, a missing `break;` statement in `SPELL_MASS_DOMINATE` caused group dispatches to fall through directly into `SPELL_GROUP_INVIS`. This meant dominated NPCs were accidentally hit with invisibility. Go cleans this up with isolated non-fallthrough switch cases.

### 4. Dead C Code for `SPELL_CLONE`
- **Source Context**: `src/magic.c#L1705` (C default switch), `src/magic.c#L1730` (C dead clone logic).
- **Fidelity Gap**: In C, the switch block in `mag_summons` only checks `SPELL_ANIMATE_DEAD` and defaults to `return;`. This rendered the post-switch clone validation checks (`if (spellnum == SPELL_CLONE)`) entirely unreachable dead code. Go replaces this by delegating clone/summon mirrors safely to isolated manual dispatches (`castMirrorImage` / `castConjureElemental`).

---

## 3. Go Improvements Over C

### 1. Robust Syllables Obfuscation
- **Go Enhancement**: Go’s `ObfuscateSpellName` utilizes `strings.Builder` and isolated syllable matching arrays to cleanly translate spell names to magical chants, completely avoiding C's unsafe character indexing and potential out-of-bounds pointer increments during parsing.

### 2. Centralized Interface Decoupling
- **Go Enhancement**: Go’s `CallMagic` dispatcher accepts `interface{}` parameters and validates methods dynamically via simple trait-assertion interfaces (e.g. `type rg interface{ GetRoomVNum() int }`, `type sender interface{ SendMessage(string) }`). This eliminates tight package dependencies and prevents compilation circular loops cleanly.

### 3. Concurrency Protection
- **Go Enhancement**: Individual spell updates read caster and target traits thread-safely by fetching stats under localized locks, preventing concurrent read-write races on character attributes in multi-client combat.

---

## 4. Concurrency & Thread Safety

- **Read-Only Saving Throw Tables**:
  - `savingThrowTable` is a statically initialized multi-dimensional array. It is entirely read-only post-init, ensuring concurrent Lookups (`GetSavingThrow`) are inherently thread-safe without requiring lock synchronization overhead.
- **RNG Seed Safety**:
  - Saving throws and damage checks use standard `math/rand` rolls. In Go, the package-level random functions are thread-safe and protected by internal locks.

---

## 5. Summary of Recommended Fixes

1. **Wire Target Corpse Validation for `Animate Dead`**:
   Update `MagSummons` in `pkg/spells/affect_spells.go` to require an targeted object argument (`tobj`) representing a corpse container. Add checks to ensure the corpse is present, extract its contents to transfer them to the zombie, and safely delete the corpse upon animation success.
2. **Move Created Food to Inventory**:
   Update `MagCreations` in `pkg/spells/affect_spells.go#L533` to transfer the spawned food object directly to the player's inventory list instead of dropping it onto the ground.
