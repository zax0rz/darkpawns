# Dark Pawns — Next Session Prompt

**Branch:** main  
**Working dir:** `~/darkpawns/`  
**Plan:** `docs/PORT_SCOPE.md` (read this first — has full gap analysis, priorities, remaining work)

## Context

C-to-Go MUD port. Core game loop is functional. We just did a thorough function-by-function audit of every C file against its Go counterpart and documented the real remaining gaps in PORT_SCOPE.md.

**Run `git log --oneline -5` at session start.**

## Current State

**~2,400 C lines remaining (~1,700-2,400 Go lines).** 6 commits in the last session:
- wave19 batch 5: spell system routing + death.go TYPE_/SKILL_ constants
- wave19 tier2 batch A: error handling with retry/backoff
- wave19 tier2 batch B: game infrastructure fixes
- wave19: clear remaining TODOs — zero actionable TODOs left

All TODOs are resolved (converted to docs or implemented). The remaining work is filling in missing functions.

## What's Next (in priority order)

### Priority 1: Core gameplay blockers (~480 C lines)
1. **limits.c** — `exp_needed_for_level`, `find_exp`, `mana_gain`, `hit_gain`, `move_gain`. XP table + regen formulas. Port to `pkg/game/limits.go`.
2. **handler.c helpers** — `remove_follower`, `set_hunting`, `obj_from_room`, `obj_to_obj`, `obj_from_obj`, `get_number`, `affect_to_char2`, `get_obj_num`, `get_char_num`, `free_char`. Spread across `pkg/game/inventory.go`, `movement.go`, `follow.go`.
3. **Shared utility functions** — `remove_follower`, `set_hunting`, `stop_follower`, `can_speak`, `add_follower`, `num_followers`, `circle_follow`, `die_follower`. Some exist as MobInstance methods — need centralized versions in `follow.go`.

### Priority 2: Persistence (~750 C lines)
4. **objsave.c** — Binary save/load format. `Obj_to_store`, `Crash_load`, etc. Go may use JSON instead. Logic matters more than format. `pkg/game/objsave.go` partially done.
5. **house.c** — File I/O for houses: `House_load`, `House_save`, `House_crashsave`, etc. Structure exists (1086 lines), file persistence missing.
6. **events.c + queue.c** — Custom event queue. May already be handled by Go goroutines/tickers. **Verify first** before porting.

### Priority 3: Commands & features (~550 C lines)
7. **act.wizard.c** — Immortal commands: `do_stat_room`, `do_stat_object`, `stop_snooping`, visibility toggles, zone info.
8. **act.social.c** — Social file loading: `fread_action`, `find_action`, `boot_social_messages`.
9. **ban.c** — Ban persistence: `load_banned`, `isbanned`, `write_ban_list`.

### Priority 4: Polish (~350 C lines)
10. **act.informative.c** — Display helpers (target room, class bitvector, print_object_location).
11. **oc.c** — Object editor commands.
12. **mapcode.c** — ASCII map rendering.
13. **graph.c** — BFS helpers.
14. **scripts.c** — Lua table operations.
15. **boards.c** — Board persistence.
16. **weather.c** — Weather effects.

## Rules
- Read C source before writing Go. `src/` is authoritative.
- `go build ./...` after every change. `go test ./...` before commits.
- slog not log/fmt. No globals.
- Import cycle: spells→game. Use interface assertions.
- Use subagents for self-contained tasks. `deepseek-v4/deepseek-v4-pro` works well for bounded tasks. GLM-5.1 for medium-scope. **NOT** for large files (>1500 lines combined context).
- Commit after each priority batch. Update PORT_SCOPE.md as functions are ported.
- When a subagent fails, check the actual file state before retrying.

## Subagent Learnings
1. **deepseek-v4-pro crashes on large files.** Don't give it >1500 lines of combined C+Go context. Split into smaller tasks.
2. **Automated function-name matching overcounts wildly.** The `docs/C_FUNCTIONS.json` script reports 12K lines missing — that's wrong. Forward declarations, duplicate entries, and missed renames inflate it 5x. The manual audit (~2,400 lines) is accurate. Use the JSON as a starting point, not gospel.
3. **Many "missing" C functions exist in Go with different names.** Always grep for camelCase equivalents before declaring something missing.
4. **comm.c functions are mostly Go networking layer.** `game_loop`, `new_descriptor`, `process_input`, `close_socket` etc. are handled by Go's net/websocket/session layer. Don't port them — verify they're covered.
5. **handler.c has the highest density of genuinely missing functions.** The object movement helpers (obj_from_room, obj_to_obj, etc.) and follow-chain management are the real gaps.
6. **improved-edit.c is the text editor.** Check if `pkg/game/text_editor.go` or `pkg/game/editor.go` covers it before porting.
7. **spec_assign.c functions run at boot only.** `assign_mobiles`, `assign_objects`, `assign_rooms` — check if Go boot sequence already handles spec assignment.

## Post-Completion
When all functions are ported:
1. Run full `go vet ./...` + `go test ./...`
2. Remove `pkg/game/act_item_stubs.go` (dead code — stubs no longer referenced)
3. Remove any remaining `.bak` files
4. Update PORT_SCOPE.md with "PORT COMPLETE" status
5. Then: Sonnet QA review (4 passes pairing Go with C source) + Opus security review
