# Commit Diff Sentinel — 2026-05-17

**Period:** 2026-05-16 04:30 ET → 2026-05-17 04:30 ET  
**Commits reviewed:** 12  
**Build:** clean (`go build ./...` + `go vet ./...` pass)

## Commits Reviewed

| Hash | Message | Files |
|---|---|---|
| `0e9f902` | fix: DP-162 memory hook body on first attempt + DP-161 graceful shutdown | 2 |
| `72e3395` | feat: DP-155 Phase 1 — unified affect data model | 18 |
| `1529154` | feat: DP-155 Phase 2+3 — spells + session migrated to new affect API | 4 |
| `33d637a` | feat: DP-155 Phase 4+5 — save format + deprecated alias cleanup | 2 |
| `f0266d2` | docs: update stale numbers and port status after spell system completion | 3 |
| `5117d25` | Fix formatting in architecture diagram | 1 |
| `9a00071` | docs: Lua script deployment vnum mapping plan | 1 |
| `6977da5` | fix: Lua trigger flag bitmask — match C source (DP-166) | 1 |
| `24cd0b4` | feat: wire death script trigger — ScriptDeathFunc (DP-167) | 4 |
| `e63bf8c` | feat: deploy Phase 1 Lua scripts (DP-165) | 7 |
| `451b3fa` | feat: deploy combat AI scripts — Phase 2 (DP-165) | 30 |

## Findings

### MEDIUM-001: `spellStackKey()` produces unreadable stack IDs

**File:** `pkg/engine/affect.go:133-135`

**What:** `spellStackKey(spellID)` uses `string(rune(spellID))` which converts the integer to a single Unicode code point instead of a decimal string. For spellID=42, the key is `"spell_*"` (asterisk U+002A) instead of `"spell_42"`.

**Why it matters:** Stack IDs are used for affect deduplication in `AffectManager.AddAffect()`. While the keys are technically unique per spellID, they're unreadable in logs/debugging. If any external system or save file references StackID, it will be indecipherable. The old code used `strconv.Itoa()` correctly — `strconv` import was removed in the same commit.

**Suggested fix:** Add `"strconv"` back to imports, change to `return "spell_" + strconv.Itoa(spellID)`.

### LOW-001: Debug `print()` calls in deployed Lua script (hisc.lua)

**File:** `lib/world/scripts/mob/144/hisc.lua:4,6,13`

**What:** Three `print()` calls remain in the deployed `hisc.lua` script — two in `oncmd` and one in `sound`.

**Why it matters:** `print()` in the Lua VM writes to stdout, clogging server console/logs with debug output. These are clearly leftover from development. Should use `log()` in production, or remove the calls entirely.

**Suggested fix:** Replace `print(...)` with `log(...)` or remove the debug lines.

### Correct Changes (no issue needed)

- **DP-166 fix:** Trigger flag bitmask corrected to match C `1<<N` shift — `MS_FIGHTING` is now `256`, not `128`. All mob data files use correct bitmask values. ✓
- **DP-167:** Death script trigger wired from `combat.engine.handleDeath()` through `World.FireMobDeathScript()` to Lua VM. Read lock held during lookup, released before script execution — correct deadlock avoidance. ✓
- **DP-161:** `sync.WaitGroup` tracks zone reset goroutine; `wg.Wait()` called before `SaveWorld()` on shutdown. Prevents concurrent world state writes. ✓
- **DP-162:** Memory hook body now set on every HTTP attempt, not just retries (`attempt > 0` guard removed). First attempt no longer sends empty body. ✓
- **DP-155 Phase 1-5:** Affect system unification compiles clean. APPLY_* constants match `structs.h`, AFF_* bit flags match C. Save format backward-compatible with legacy Type field fallback in `restoreAffects()`. ✓
- **Lua scripts (Phase 1+2):** All 11 deployed scripts are faithful ports of original archive scripts. No logic errors detected. ✓
- **Mob data:** 30 mob files updated with correct `Script: <file>.lua 256` references. ✓
- **Architecture docs:** Whitespace formatting fix only. ✓

**0 critical, 0 high, 1 medium, 1 low.**
