# Dark Pawns — Next Session Prompt

**Branch:** main  
**Working dir:** `~/darkpawns/`  
**Plan:** `docs/PORT_COMPLETION_PLAN.md` (read this first — has full scope, session log, and remaining items)  
**Audit doc:** `docs/SOURCE_ACCURACY_AUDIT.md` (now stale — most items already fixed)

## Context

Wave 19 is in progress — closing out the remaining 45 TODOs to complete the C-to-Go port. Prior session killed 11 TODOs (equipment regen, veteran, kill counter, autosplit, KK_JIN/KK_ZHEN, PRF_INACTIVE, AFF_FLESH_ALTER, idle handling, dreams, autowiz, char creation, gender pronouns, tattoo spawn). All committed, CI green.

**Run `git log --oneline -5` and `grep -rn 'TODO' pkg/ --include='*.go' | grep -v '_test.go\|.bak' | wc -l` at session start.**

## Priority: Continue PORT_COMPLETION_PLAN.md

### Batch 3: Scripting Engine TODOs (~600 lines)
8 TODOs in `pkg/scripting/engine.go`. These are good subagent candidates — read the file, match each TODO against the C source, implement. Key items:
- Zone broadcast (2 TODOs) — send to all players in a zone
- Mob command execution — mobs can run game commands
- Soul leech heal — caster heals dam/3
- Move points restoration
- Death handling in scripts
- Object placement (room vs char inventory)
- Full can_see (PLR_INVISIBLE, room DARK, AFF_BLIND)
- Global channel broadcast

### Batch 4: Wizard Commands (~800 lines)
8 TODOs in `pkg/session/wizard_cmds.go`. Good subagent candidate — read C's act.wizard.c, implement:
- cmdLoad, cmdPurge, character switch/return, cmdReload, object stat, wizlist/last login, combat-stop broadcast

### Batch 5: Spell System Cleanup (~400 lines)
- `pkg/spells/spells.go` — Cast() default case, damage spell stubs
- `pkg/spells/say_spell.go:327` — room iteration for zone messages
- `pkg/spells/damage_spells.go:316` — same
- `pkg/game/death.go:368` — death attack type constants
- `pkg/combat/formulas.go:565` — backstab function (not just TODO stub)

### Tier 2: Go Modernization (~800 lines)
- Database error handling + retry (2 TODOs in optimization/)
- Board room echo (2 TODOs)
- House storage (2 TODOs)
- Clan string_write, weather graph, zone mob AI, world target resolution, world death handling, death reconnection, other settings

## Rules
- Read before write. `go build ./...` after every change. `go test ./...` before commits.
- slog not log/fmt. No globals. C source (`src/`) is authoritative.
- Import cycle: spells→game. Use interface assertions.
- Use subagents for Batch 3 and 4 — they're large and self-contained. Feed them the C source + Go file + specific TODOs.
- Commit after each batch. Update PORT_COMPLETION_PLAN.md session log.

## Post-Completion
Once all TODOs are gone:
1. `grep -rn 'TODO' pkg/ --include='*.go' | grep -v '_test.go\|.bak'` — should return 0
2. `go vet ./...` + `go test ./...`
3. Remove all .bak files
4. Update PORT_COMPLETION_PLAN.md with "PORT COMPLETE" status

Then dispatch reviews:
- **Sonnet QA:** Split into 4 passes pairing Go subsystems with C source (see plan)
- **Opus security:** Input validation, overflow, race conditions, auth, Lua injection
