# Round 4 population sweep classifications

- Source SHA: `c28a41e9d427c6b034aee40eca0c4a4c2199765f` (the frozen sweep SHA).
- Input: all 998 literal tokens in `literals.tsv`; row numbers in `classifications.tsv` are zero-based and follow that file exactly.
- Classes are committed before any sweep census. This file and the TSV are pre-registered classification evidence; no census result informed a class.
- The C call sites were checked in `src/` for exact text, C format strings, adjacent string fragments, action-table output, and helper-based output. `c-reachable` means C can emit the corresponding player-facing text; `go-only-in-game` stays in the denominator. No row is a `go-only-surface` or `harness-blind` case.

## Class counts

| Package | c-reachable | c-unreachable | go-only-in-game | go-only-surface | harness-blind | ask-zach | Total |
|---|---:|---:|---:|---:|---:|---:|---:|
| `pkg/boards` | 8 | 0 | 0 | 0 | 0 | 0 | 8 |
| `pkg/combat` | 13 | 0 | 0 | 0 | 0 | 0 | 13 |
| `pkg/command` | 54 | 0 | 0 | 0 | 0 | 0 | 54 |
| `pkg/game` | 711 | 1 | 7 | 0 | 0 | 2 | 721 |
| `pkg/session` | 197 | 0 | 5 | 0 | 0 | 0 | 202 |
| `pkg/spells` | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| **Total** | **983** | **1** | **12** | **0** | **0** | **2** | **998** |

## Rows pending Zach's class ruling

- **Row 148, `*World.doClanInfo`, `"(null)"`:** Go emits this when a clan plan is nil. C passes `clan[i].plan` to `%s` in `do_clan_info` (`src/clan.c:774`); a null `%s` argument is undefined behavior, so the reference bytes depend on the C runtime. Zach ruled 2026-11-22: keep the row `unknown`, out of kills and the denominator. This remains an R1a question about the intended output for a planless clan; attribution is settled.
- **Row 472, `*World.npcRemoveDirect`, `"all."`:** this source token is the remove-all parser prefix inside an output expression; it is not itself player-facing text (`pkg/game/npc_command_items.go:251`). The collector's same-function dataflow includes it in the frozen population, but C has no corresponding output token. The row remains `ask-zach` and stays in the denominator.

## c-unreachable evidence

- **Row 697, `specTurnUndeadObject`:** C declares `SPECIAL(turn_undead)` in `assign_objects` but has no `ASSIGNOBJ` for it (`src/spec_assign.c:525`; a full search of that assignment table finds no such assignment). The C body at `src/spec_procs3.c:543-581` is therefore not reachable in the game world. Keep the row excluded from the denominator.

## Freeze

`classifications.tsv` is paired with the pool freeze in `FREEZE.md`. Do not edit a class after a census. If later source-path evidence requires a correction, append it to `CLASSIFICATION-CORRECTIONS.md` and preserve this pre-census row and denominator.
