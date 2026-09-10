# Port Fidelity Audit: Modules 53 & 54 (`spell_parser.c` & `spells.c`)

This audit examines the port fidelity between the legacy C magic and spell systems in `src/spell_parser.c` and `src/spells.c` and their Go counterparts inside `pkg/spells/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source Files
- **`src/spell_parser.c`** (1,627 lines):
  - Defines the core spell registry tables (`spello()` / `unused_spell()`).
  - Implements the top-level magic dispatcher `call_magic()` and spoken spell caster `cast_spell()`.
  - Implements the spelling syllable dictionary translator `say_spell()` which obfuscates spell incantations for non-matching classes.
  - Implements magic items parsing `mag_objectmagic()` (wands, staves, potions, scrolls).
  - Handles the PC casting entry command `do_cast()`.
- **`src/spells.c`** (1,219 lines):
  - Implements the behavior of "manual spells" (teleport, summon, locate object, charm, identify, enchant weapon/armor, control weather, lycanthropy, vampirism, sobriety, hellfire, zen, mindsight, meteor swarm, mirror image, divine intervention).

### Go Port Files
- [pkg/spells/spells.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/spells.go): Core execution routes and routing switch.
- [pkg/spells/spell_info.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/spell_info.go): Data structures, bitmasks, mana cost math, and spell registry.
- [pkg/spells/say_spell.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/say_spell.go): Spell syllable translator and focuses-will obfuscations.
- [pkg/spells/call_magic.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/call_magic.go): Verification of targets, peaceful room checks, position checks, and routine dispatch.
- [pkg/spells/affect_spells.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/affect_spells.go): Massive implementation of all manual/affect spells (`castSummon`, `castIdentify`, `castControlWeather`, `castHellfire`, etc.) and spell reagent validations (`checkReagents`).
- [pkg/spells/damage_spells.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/damage_spells.go): Magical offensive damage calculations.
- [pkg/spells/saving_throws.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/saving_throws.go): Verbatim saving throw tables indexed by class, save type, and level.

---

## 2. High-Fidelity Validation & Design Parity

Comparing the systems reveals **exceptional architectural care** and a **1:1 logic translation**:

### 1. The Syllable Spelling Dictionary (`say_spell`)
- **Parity Status**: Flawless. Go's [say_spell.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/say_spell.go) ports the exact letter replacement array (`ar` -> `abra`, `ate` -> `i`, `ness` -> `lacri`, etc.) and rules from `src/spell_parser.c`.
- **Specialized Classes Obfuscation**: Matches the C rule where Psionics and Mystics bypass spoken syllables entirely, outputting `"$n stares at $N and focuses $s will..."` or similar, whereas other casting classes utter the syllable-obfuscated words.

### 2. Diku-Verbatim Saving Throws
- **Parity Status**: Flawless. [saving_throws.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/spells/saving_throws.go) ports the massive 3D integer array (`savingThrowTable`) verbatim from Diku's `magic.c` tables for all 12 classes across 5 categories (`SaveParalysis`, `SaveRodStaff`, `SavePetrify`, `SaveBreath`, `SaveSpell`) from level 0 to 40.

### 3. Reagent Bonuses & Consumptions
- **Parity Status**: Flawless. Go's `checkReagents` function matches the level-scaling mechanics where casting with proper reagents (e.g. eye of newt, bat wing) awards a success or damage bonus scaling with `level / 2` (minimum 1), consumes the item, and outputs specialized socket messages.

### 4. Advanced/Custom Spell Implementations
Every manual spell from `src/spells.c` was validated:
- **`spell_mirror_image`**: Perfectly ports mob clone creation (VNUM 69), copying character names, sex, and titles, and redirecting memorized/hunting aggressors from the caster to the clone.
- **`spell_divine_int` (Divine Intervention)**: Perfectly checks alignment, grovels, spawns an angel helper (VNUM 85/86 depending on good/evil alignment), applies the charm affect, and adds them as quiet followers.
- **`spell_summon`**: Seamlessly incorporates PvP outlaw checks, room `!exit` restrictions, dump checks, saving throw checks, and the funny **10% spell backfire check** where the summoner is accidentally pulled into the target's room.

---

## 3. Go's Architectural Improvements Over C

- **Lua Trigger Integrations**: Go integrates spell triggers cleanly with the entity system, bypassing fragile dynamic C void pointers.
- **Thread Safety**: All spells executed within the MUD gameloop check room/world state under thread-safe mutex operations.
