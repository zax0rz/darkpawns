# Dark Pawns Research Log

[DESIGN] [2026-04-24] Initial thought on the game...

## 2026-04-24 — Wave 5 Complete: Character Management & Affect Lifecycle

[RESULT] [DESIGN]

**Status:** All 30 C functions from handler.c, limits.c, magic.c mapped to Go equivalents.

**What was written:**
- `pkg/game/char_mgmt.go` — `HasLight`, `UpdateCharObjects`, `UpdateCharObjectsAR`, `ExtractChar`, `ExtractPendingChars`, `UpdateObject`
- `pkg/game/affect_update.go` — `AffectUpdate`, `SpellWearOffMsg`

**Already existed:**
- All 11 C affect helpers (handler.c:136-499) → `pkg/engine/affect_helpers.go`
- Gain functions (limits.c:30-197) → `pkg/game/limits.go`
- `init_char` → `NewPlayer`/`NewCharacter` constructors in `player.go`

**Key design decisions (deviations from C):**
- `init_char` → Go constructors, not "lazy init + blank fields". `NewPlayer()` for loading from DB, `NewCharacter()` for new creation (with `do_start()` equivalent).
- `free_char`/`clear_char` → handled by Go GC and struct zero-value patterns. No explicit free.
- `stop_follower`/`add_follower`/`remove_follower`/`set_hunting` → C linked-list follower system replaced by `Player.Following` string + simplified design. WIP for 6+.
- `obj_from_obj`/`object_list_new_owner` → C linked-list pattern doesn't match Go slice pattern. Not ported.
- `PLR_EXTRACT` bit = 21, `NOWHERE` = -1, `ITEM_LIGHT` type = 1 — constants defined inline in char_mgmt.go.
- `AffectUpdate` uses `AffectFromChar` (from affect_helpers.go) instead of `affect_remove` — same semantics.

**Build:** clean, vet clean.

---

## 2026-04-24 — Wave 6.5: Wire act.other.c Commands

[DESIGN] [RESULT]

**Status:** 22 player-facing commands from act.other.c now registered and usable. Game logic (`World.doXxx`) existed in `pkg/game/act_other.go` but had zero session-level wiring — players couldn't type `save`, `report`, `afk`, etc.

**The bridge pattern:**
- Problem: `World.doXxx` methods are unexported (lowercase), so `pkg/session/` can't call them directly.
- Solution: Created `pkg/game/act_other_bridge.go` with 21 exported `ExecXxx` one-liners that delegate to the corresponding `doXxx`, passing `nil` for the mob instance (player sessions don't have one). Session wrappers call `s.manager.world.ExecSave()` etc.
- Alternative considered: Exporting all `doXxx` methods (capitalize). Rejected — would pollute the game package surface and the task explicitly said not to change the game package.

**Wired commands by priority:**
- P0 (must-have): save, report, peek, recall, afk
- P1 (important): split, wimpy, display, visible, inactive, gentog, bug/typo/idea/todo
- P2 (nice): transform, ride, dismount, yank, stealth, appraise, scout, roll, auto

**NOT wired:** `do_not_here` — it's a stub/placeholder for `cmd_not_here`, not intended for direct player use.

**Observations:**
- The `auto` command registration overlapped with the existing `autoexit` internal flag — checked during input parsing, no collision (different command strings).
- `cmdAFK` had to be named carefully — original plan called it `cmdAfk` but registration used `cmdAFK`. Caught during build.
- Bug/typo/idea/todo all share one `doGenWrite` backend — each wrapper passes a different command string to distinguish the report type.

**Build:** `go build ./... && go vet ./...` clean. All 22 registered.
**Commit:** `2098e2c`

---

## 2026-04-24 — Wave 7: Spell System (magic.c, spells.c, spell_parser.c)

[DESIGN] [RESULT] [OBSERVATION]

**Status:** 4,843 C lines ported to 8 Go files (1,846 lines). `go build ./...` and `go vet ./...` both pass clean.

**Go files written:**
- `call_magic.go` (202 lines) — CallMagic central dispatch, SpellInfo, CastType/MagRoutine/TarFlags types, magSavingThrow, magicManaCost, checkMaterials/checkReagents
- `damage_spells.go` (327 lines) — MagDamage switch with 20+ spell damage formulas, dice() helper, all breath weapons
- `affect_spells.go` (277 lines) — MagAffects (bless, armor, blindness, curse, invisible, sanctuary, sleep, poison, haste, slow, fly, detect spells, infravision, water breathe, etc.), MagPoints, MagUnaffects, plus group/mass/area/summon/creation/alter-obj stubs
- `affect_effects.go` (60 lines) — Existing 5 affect spells (blindness, curse, poison, sleep, sanctuary)
- `spell_info.go` (213 lines) — SpellInfo struct, HasRoutine, GetSpellInfo, SpellLevel, MagAssignSpells with full template array
- `saving_throws.go` (244 lines) — Full sav_throws table (6 classes × 21 levels × 5 save types), GetSavingThrow, CheckSavingThrow
- `say_spell.go` (347 lines) — Syllable substitution table, ObfuscateSpellName, SaySpell with class-aware incantation display
- `object_magic.go` — MagObjectMagic for potion/wand/staff/scroll

**Design decisions (deviations from C):**
- **Interface duck-typing over type imports:** All spell functions use `interface{}` parameters and ad-hoc interfaces (`damageable`, `healer`, `affecter`, `sender`, `classer`, `poser`, `lever`) instead of importing `pkg/game` or concrete character types. This avoids the circular import problem (spells is imported by game, so spells can't import game).
- **MagRoutine constants renamed:** C macros `MAG_DAMAGE`, `MAG_AFFECTS`, etc. → Go `RoutineDamage`, `RoutineAffects`, etc. (the original MagXxx names conflict with MagDamage() function — Go is case-sensitive so both can't coexist).
- **Spell constants renamed for clarity:** `SPELL_ARMOR` → `SpellArmor`, `SPELL_TELEPORT` → `SpellTeleport`, etc. One gotcha: `SPELL_CURE_CRIT` → `SpellCureCritic` (not SpellCureCritical, matching C name).
- **sav_throws as package-level var:** Ported the C 3D array exactly but as `[NumClasses][SavingThrowTableSize][SavingThrowCount]int`.
- **SaySpell as package-level function:** Not on a struct — it's a stateless port of the C function that reads character info via duck-type interfaces.
- **Kimi K2.5 subagents broke badly:** Repeatedly started subagents on the same task, they returned after ~10s with "Let me read the files" and no actual work. Subagent model (deepseek-chat) lacks the follow-through for complex multi-file tasks. Direct editing was much more effective.

**Key lesson for Wave 8:** Spawn subagents on `zai/glm-5.1` or `deepseek-v4-flash` for tasks >500 lines — step-3.5-flash and deepseek-chat don't have the agentic follow-through for multi-file porting.

**What's stubbed (needs Wave 8):**
- MagGroups, MagMasses, MagAreas, MagSummons, MagCreations, MagAlterObjs — all have `// TODO` stubs
- ExecuteManualSpell — dispatch map created but all entries call TODO functions
- roomHasNoMagic / roomIsPeaceful — need World.GetRoom() wiring
- Cast() still uses old ApplySpellAffects path; needs to call CallMagic instead

**Build:** `go build ./... && go vet ./...` clean.
**Commit:** (pending)
