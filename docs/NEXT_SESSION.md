# Dark Pawns — Next Session Prompt

**Branch:** main  
**Working dir:** `~/darkpawns/`  
**Plan:** `docs/PORT_COMPLETION_PLAN.md` (read this first — has full scope, session log, and remaining items)  
**Audit doc:** `docs/SOURCE_ACCURACY_AUDIT.md` (stale — most items already fixed)

## Context

Wave 19 is in progress — closing out the remaining TODOs to complete the C-to-Go port.  
Previous sessions killed Batch 1 (core loop gaps), Batch 2 (player systems), Batch 3 (scripting engine, partial), and Batch 4 (wizard commands). All committed, CI green.

**Run `git log --oneline -5` and `grep -rn 'TODO' pkg/ --include='*.go' | grep -v '_test.go\|.bak' | wc -l` at session start.**

## Current State

**TODOs remaining: 26** (started session at 45)

### Batch 3 leftovers (scripting — 0 remaining, ALL DONE)
All 8 scripting engine TODOs resolved. The Batch 3 subagent (GLM-5.1) got 5/8 in its 2m40s run — the remaining 3 were stale comments that got cleaned manually.

### Batch 4 (wizard commands — 0 remaining, ALL DONE)
Done manually after GLM-5.1 subagents failed (crashed at 14s and 83s — file too large for their context). All 9 TODOs implemented directly.

### What's Next

#### Batch 5: Spell System Cleanup (~4 code TODOs + header comments)
- `pkg/spells/spells.go:190` — Cast() default case is a stub. Needs to route 60+ damage/effect spells through `pkg/spells/call_magic.go`'s existing `CallMagic`/`MagDamage`/`MagAffects` dispatch.
- `pkg/spells/say_spell.go:327` — `sendToRoom()` is a stub returning true. Needs real room iteration. **Challenge:** spells package uses `interface{}` for all params due to import cycle (spells→game). Need type assertions or a shared interface.
- `pkg/spells/damage_spells.go:316` — `sendToZone()` is a no-op stub. Same interface{} challenge.
- `pkg/game/death.go:368` — `attackTypeToCorpseAttack` missing TYPE_/SKILL_ constants (SLASH, PIERCE, CRUSH, BASH, KICK, etc.). C ref: `src/fight.c:283-370` and `src/structs.h`.

**Note:** The large TODO header comments in `spells.go:8` and `spells.go:40` are documentation, not code. They can be removed once the Cast() switch is wired.

#### Tier 2: Go Modernization (~22 TODOs)
Mostly infrastructure polish. Good subagent candidates. Key files:
- `pkg/optimization/database.go:244` + `websocket.go:209` — error handling/retry
- `pkg/game/houses.go` — 4 TODOs (DB lookup, obj store wiring)
- `pkg/game/clans.go:1202` — string_write
- `pkg/game/boards.go:335,489` — room echo (BoardSystem has no World ref)
- `pkg/game/graph.go:179` — weather penalty
- `pkg/game/zone_dispatcher.go:126` — mob AI
- `pkg/game/world.go` — 5 TODOs (mob command routing, death, target resolution)
- `pkg/game/other_settings.go:104` — just a doc comment, not real TODO

#### Tier 3: Reviews (post-completion)
- **Sonnet QA review** — 4 passes pairing Go subsystems with C source
- **Opus security review** — input validation, overflow, race conditions, auth, Lua injection

## Subagent Learnings (IMPORTANT)

1. **GLM-5.1 fails on large files.** Both Batch 4 attempts cratered (14s, 83s) on `wizard_cmds.go` (1,574 lines) + `act.wizard.c` (3,863 lines). **Too much combined context.** For large files, implement manually or split into smaller scoped tasks.

2. **DeepSeek V4 Pro is NOT in the allowlist as `zai/deepseek-v4-pro`.** The correct model ID is `deepseek-v4/deepseek-v4-pro` (provider/model format). It's already in `agents.defaults.models` in `openclaw.json`. Cost: $0.25/$0.40 per M tokens (90% sale through May).

3. **GLM-5.1 works well for medium-scope tasks.** Batch 3 (scripting, 8 TODOs across engine.go) completed in 2m40s and got 5/8 done. The 3 it missed were stale comments, not real code.

4. **When subagents add interface methods**, they often forget to update test mocks. The `ExecuteMobCommand` addition to `ScriptableWorld` broke `integration_test.go` — had to add mock implementations manually. Always check test compilation after interface changes.

5. **Unexported→exported method renames cascade.** Exporting `extractMob`, `sendToZone`, `sendToAll` broke internal callers in `spec_procs2.go` and `spec_procs3.go`. If renaming, grep the entire codebase.

6. **`parser.World` uses `Objs` not `Objects`.** The Obj slice field is `Objs`. `ObjectInstance` (game package) uses `GetShortDesc()` method, while `parser.Obj` uses `ShortDesc` field directly. Easy to confuse.

## Rules
- Read before write. `go build ./...` after every change. `go test ./...` before commits.
- slog not log/fmt. No globals. C source (`src/`) is authoritative.
- Import cycle: spells→game. Use interface assertions.
- Use subagents for Tier 2 — they're infrastructure polish, self-contained. Use `deepseek-v4/deepseek-v4-pro` or `zai/glm-5.1` (NOT `zai/deepseek-v4-pro`).
- Batch 5 should be done directly — the interface{} plumbing in spells is too fiddly for subagents.
- Commit after each batch. Update PORT_COMPLETION_PLAN.md session log.

## Post-Completion
Once all actionable TODOs are gone:
1. Remove stale TODO comments (header docs in spells.go can become regular docs)
2. `go vet ./...` + `go test ./...`
3. Remove all .bak files
4. Update PORT_COMPLETION_PLAN.md with "PORT COMPLETE" status

Then dispatch reviews:
- **Sonnet QA:** Split into 4 passes pairing Go subsystems with C source (see plan)
- **Opus security:** Input validation, overflow, race conditions, auth, Lua injection
