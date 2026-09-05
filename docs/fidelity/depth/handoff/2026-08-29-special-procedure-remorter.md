# 2026-08-29 — `remorter` depth slice

## Frontier and queue

- Started this slice on `glm/spec-remorter` from the synchronized `main`
  frontier after the required `make fidelity-depth` confirmation and reread of
  `docs/fidelity/DEPTH_TESTING.md` plus the newest dated handoff.
- The pre-slice frontier was 1011 total cases: 974 proven/delegated, 12
  blocked, and 25 excluded. This slice adds eight remorter cases, yielding
  1019 total: 982 proven/delegated, 12 blocked, and 25 excluded. Actionable
  completion is 982/994 = 98.8%.
- `SPECIAL(remorter)` is the next reachable source-order procedure after the
  `pissedalchemist` registration exclusion. The next source-order procedure
  after this slice is `assassin`; its registration audit is still pending.
  Do not repick `remorter`, `pissedalchemist`, `rescuer`, `stableboy`,
  `tipster`, `couch`, `whirlpool`, or earlier claimed procedures.

## C call path and branch census

- `SPECIAL(remorter)` is defined at `src/spec_procs2.c:682-843` and assigned
  to mob vnum 4 at `src/spec_assign.c:183`. The player-command dispatch is
  `src/interpreter.c:1407-1456`; the C mobile also carries `MOB_RANDZON`, so
  the vehicle follows its source-derived landing room 8044 after clearing
  only that placement flag in disposable copies.
- The procedure first rejects an NPC actor, trims command arguments, and
  handles every `buy` and `list` form with the direct `do_tell` instruction.
  Only `remort` continues to the gates: level below `LEVEL_IMMORT-1`, level at
  or above `LEVEL_IMMORT`, insufficient 60000 gold, and any equipped item.
  Each gate returns TRUE with its own native bytes; unrelated commands return
  FALSE to the normal interpreter.
- The successful body clears PLR_IT/vampire/werewolf state and active affects,
  deducts 60000 gold, maps the old class/race through `find_remort_class`,
  resets level/experience/wimp, draws the 30–40 HP and 20–30 mana maxima,
  performs the two remort stat adjustments, sets PLR_REMORT, zeroes skills,
  seeds class and racial skills, sets ten practices, restores tattoo affects,
  calls `advance_level`, tells the actor through the remorter, and emits the
  two final direct actor lines. C's `advance_level` increases maxima without
  healing current pools; the remorter path restores the pre-call current
  pools after the shared Go helper to preserve that call-path behavior.

## RED/GREEN evidence and port result

- The C-first gate vehicle used the real registered vnum 4, isolated competing
  specials with `quiet-mobs`, and cleared only `MOB_RANDZON` in disposable
  copies. RED on main showed Go falling through to generic `Huh?!?` for the
  native list/buy tells and immortal remort rejection.
- The GREEN vehicles cover the exact list/buy tell forms and FALSE fallthrough,
  level 29 rejection, level 30/zero-gold rejection, funded equipped rejection,
  and a funded naked successful remort with a peer. The success score proves
  the class transition to Human Paladin, level/experience/gold reset, current
  versus maximum HP/mana/move state, and the final direct output. All five
  vehicles report `result: no normalized divergence` with
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` and seed 1.
- Focused unit proof in `pkg/game/spec_remorter_test.go` covers the full
  class mapping, state/flag/affect reset, class skill seeding, gate ordering,
  command fallthrough, and nil autonomous entry.
- The disposable scenario harness gained a narrowly scoped
  `clear-mob-flag <vnum> RANDZON` fixture in
  `internal/oraclediff/scenario.go` and `cmd/dp-oracle-diff/main.go`; it never
  changes the shipped world or oracle source.
- No `src/` or `darkpawns-c-oracle/` file was edited. The unrelated
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

## Verification

- Green local gates: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`.
- This slice applies R1 (exact bytes), R2 (command surface and FALSE
  fallthrough), R3 (draw/state/current-pool parity), R4 (no invented output),
  and R5/R5e (C-first actual registered call path). The shared affect unwind
  keeps the legacy compatibility state from leaking into the successful
  transition without using a deprecated helper.

## Manifest

The durable rows are in `docs/fidelity/depth/spec-procs.tsv`:

- `mob.remorter-command-gates`
- `mob.remorter-buy-list-tell`
- `mob.remorter-fallthrough`
- `mob.remorter-low-level-gate`
- `mob.remorter-gold-gate`
- `mob.remorter-equipment-gate`
- `mob.remorter-success-state`
- `mob.remorter-success-audience`

## Next queue item

Continue the active special-procedure inventory with `assassin` at
`src/spec_procs2.c:845`. First grep the complete dispatch registration tables;
if no `ASSIGNMOB(..., assassin)` registration reaches a `MOB_SPEC` prototype,
record that exclusion before advancing to `tattoo1` at
`src/spec_procs2.c:945`, registered at `src/spec_assign.c:296`.
