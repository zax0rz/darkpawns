# Modernization Phase 4.6 — Shared XP and level helpers

Date: 2026-09-06  
Status: implementation prepared locally; serial handoff gate is PR #1396 landing

## Scope

Pure refactor only. The session infobar now calls the existing `game.FindExp`
implementation instead of carrying a second `findExp` copy. The XP-share
arithmetic now lives in exported `combat.CalcXPShare`; the combatant wrapper
retains `CalcLevelDiff`, while the world party path passes its existing XP cap
explicitly. This preserves the pre-refactor distinction between the combat and
world caps while removing the duplicate formula.

The spell package's `lvlImmort` alias now reads `combat.LVL_IMMORT`, removing
the import-cycle workaround's duplicate literal without making `pkg/spells`
depend on `pkg/game`.

## Call-path evidence

- `src/class.c:1089-1187` — `find_exp()` and `exp_needed_for_level()`.
- `src/fight.c:660-705,1637-1653` — `calc_level_diff()` and its solo/group
  callers.
- `src/spell_parser.c:414-450` — the `LVL_IMMORT` spell gate.

The affected call sites are reachable through the infobar, combat death, group
XP, and spell dispatch paths (R5e). No player-facing bytes, RNG draws, or state
transitions are intentionally changed (R1/R3/R4).

## Verification

- Focused packages: `go test ./pkg/combat ./pkg/game ./pkg/session ./pkg/spells`
- Oracle: `levels-depth`, `levels-arguments-depth`, `infobar-mortal-depth`,
  `combat-death`, and `combat-entry-gates` — 5 passed, 0 failed, 0 infra,
  0 timed out.
- Repository gates: `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, `gofumpt`, `git diff --check`, expected-divergence
  pin check, and `make fidelity-depth` all pass.

The full census remains the evidence boundary recorded by the phase 4.5
closure note: 934 scenarios, 922 passed, 9 expected, 1 unpinnable, 0 failed,
2 infra, and 0 timed out. This phase is not pushed or opened as a follow-on PR
until PR #1396 is merged, per the serial modernization rule.
