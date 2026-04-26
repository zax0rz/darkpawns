# Dark Pawns — Next Session Prompt

**Context:** Wave 18 (Spell System Porting) is **COMPLETE** — all 24 manual spells implemented and pushed (10 commits, CI green).

**Branch:** main  
**Working dir:** `~/darkpawns/`  
**GBrain page:** `gbrain get_page darkpawns/wave18-spell-port` (full session state)  
**Audit doc:** `docs/SOURCE_ACCURACY_AUDIT.md` (8 systems audited, priority fixes listed)

## Priority: Equipment Regen Bonuses

Per SOURCE_ACCURACY_AUDIT.md Section 4b/4c, these are the next Phase 3 items:

1. **APPLY_MANA_REGEN** — equipment bonus to mana regen tick
2. **APPLY_HIT_REGEN** — equipment bonus to HP regen tick  
3. **APPLY_MOVE_REGEN** — equipment bonus to move regen tick

### Where to look
- **C source:** `src/limits.c` — `mana_gain()`, `hit_gain()`, `move_gain()` — equipment regen bonuses are applied inside these functions after position modifiers
- **Go source:** `pkg/game/limits.go` — regen functions with TODO comments for equipment bonuses
- **Affect system:** `pkg/engine/affect.go` — `AffectType` enum needs regen affect types added
- **Equipment:** `pkg/game/equipment.go` — `GetAffects()` already returns per-slot affects

### Approach
1. Read `src/limits.c` to find the exact equipment regen formula for each stat
2. Check what APPLY_* constants C uses (likely APPLY_MANA_REGEN, APPLY_HIT_REGEN, APPLY_MOVE_REGEN in `src/structs.h`)
3. Add corresponding `AffectType` values in `pkg/engine/affect.go`
4. Wire equipment regen bonuses into `pkg/game/limits.go` regen functions
5. Add regen affect types to `AffectTotal()` recalculation
6. `go build ./...` after every change

### Also worth fixing (from audit priority list)
The audit doc lists higher-priority items if equipment regen goes fast:
- **wis_app table** (System 2a) — completely wrong values, affects practice economy
- **Position damage multiplier** (System 1a) — wrong formula in `CalculateDamage`
- **getMinusDam high-AC entries** (System 1b) — missing -170 to -290

### Rules
- Read before write. Check interfaces before calling them.
- `go build ./...` after every significant change.
- Use `slog` not `log` or `fmt` for logging.
- No globals — pass dependencies through params.
- C source is authoritative: `src/` directory. Fix C bugs noted during port.
- Import cycle: spells→game creates a cycle. Use interface assertions.
- For Claude Code subagents: always prepend `unset ANTHROPIC_API_KEY &&` to use subscription OAuth.
