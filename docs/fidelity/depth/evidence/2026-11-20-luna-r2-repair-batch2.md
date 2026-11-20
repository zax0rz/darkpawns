# Round 2 repair batch 2 evidence

All oracle runs below use current main `72228a52e` and retain their source
snapshots under `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/`.

## Current-main control

- `luna-r2-repair-b2-control-v2` — CLEAN, 5/5 scenarios, zero setup refusals
  and zero hollow steps. It covers `split-depth`,
  `wizhelp-higher-level-target-depth`,
  `eat-werewolf-corpse-mobile-depth`,
  `switched-body-staff-zero-charge-depth`, and
  `switched-body-clan-who-depth`.

## Exact Round 2 mutants

- Row 20: `luna-r2-b2-mutant-row20-exact-v2` — NOT_CLEAN on
  `wizhelp-higher-level-target-depth`; the frozen source snapshot changes the
  sampled `cmdWizhelpBody` literal at `pkg/session/act_informative.go:275`.
- Row 22: `luna-r2-b2-mutant-row22` — NOT_CLEAN on
  `eat-werewolf-corpse-mobile-depth`.
- Row 38: `luna-r2-b2-mutant-row38` — NOT_CLEAN on
  `switched-body-staff-zero-charge-depth`.
- Row 44: `luna-r2-b2-mutant-row44` — NOT_CLEAN on
  `switched-body-clan-who-depth`.

All four mutant runs report zero setup refusals and zero hollow steps. The
earlier `luna-r2-b2-mutant-row20` is retained but is not valid row 20 evidence:
its source snapshot changed the duplicate `cmdCommandsForBody` literal, not the
sampled `cmdWizhelpBody` site. The exact-site rerun above supersedes it.

## Unit-only row 41

`TestDreamPointUpdateOutputBranches` passes on the control. Applied to the
exact row 41 one-byte mutation, that same test fails in
`death_three_to_five_days` with `Qou sleep uneasily...`, proving the sampled
player line. The exact mutant worktree is
`/home/zach/wt/luna-r2-b2-mutant-row41`.

## Row 19 unresolved site audit

The exact row 19 mutation changes the second `doSplit` insufficient-gold
literal after the actor lock (`pkg/game/other_economy.go:73-76`). The retained
`luna-r2-b2-mutant-row19` census is CLEAN on `split-depth`: the scenario reaches
the earlier precheck at lines 41-43, which remains unmutated. C
`do_split` (`src/act.other.c:836-841`) has only that precheck, and C's
`game_loop` processes descriptor commands serially (`src/comm.c:594-634`).
This is a live C-output witness but not a revert proof for the sampled Go
literal. The row's post-census classification is pending Zach's decision; no
`doSplit` code is changed in this batch.
