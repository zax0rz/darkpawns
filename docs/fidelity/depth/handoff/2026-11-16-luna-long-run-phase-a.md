# Luna long-run port completion: Phase A handoff

## Phase A disposition

Phase A of `PROMPT-2026-11-15-luna-long-run-port-completion.md` is complete.
The DP-1535 branch includes the previously merged DP-1547 direct-call fix and
is based on `origin/main` `86598452bf308f67fbaeb9ef1b79cf32c9291a2f`.

Settled 45 manifest rows: 43 previously blocked rows now have C-first oracle
witnesses, and the backstab target-fighting and aware rows moved from
unit-only to oracle-backed proof. The `tag` case is recorded as
`tag.switched-mobile-npc-target`; C `do_tag` has no skill gate. The batch also
rechecked the direct `do_look` path from `SPECIAL(start_room)`, the breed-killer
direct `do_spike` call, switched waits, and the closed-door retreat branch.

The targeted candidate census covers every registered scenario that types a
touched command, every scenario that types `switch`, and every `spec-proc-*`
scenario. It ran 500 scenarios in the primary set and an 11-scenario
non-overlapping supplement (511 selected scenarios total). The selector audit
found one eligible scenario it did not capture,
`switcheroo-retarget-keeps-round-depth`; it contains switcheroo retargeting,
not the command routing changed here. Both runs used `--draw-parity`:

- `luna-dp1535-phase-a-targeted-candidate-500`: CLEAN, 499 pass, one existing
  expected-unstable `spellmatrix-staff-olc-09-depth`, no failed/infra/timeout/
  stale/unpinnable scenarios, and zero draw-divergent scenarios.
- `luna-dp1535-phase-a-targeted-candidate-supplement-11`: CLEAN, 11/11 pass,
  no hollow steps, and zero draw-divergent scenarios.
- `luna-dp1535-phase-a-final-revert-witnesses-r2`: retained revert proof for
  52 selected witnesses; 31 fail and 21 pass as expected, with zero draw
  divergence. The focused actor-wait revert is 0/4, also with zero draw
  divergence.

The primary candidate census reports seven scenarios with ten hollow prompt
steps. These are the deliberate actor commands queued during `WAIT_STATE`:
the immediate command has no C output, and the deferred room output appears on
the following pulse block. The separate actor-wait run likewise passes 4/4
with seven such prompt-only markers; its retained revert fails 4/4. These
markers do not hide the success output or queued-command evidence.

Two C/Go divergences were found and fixed, each with an isolated retained
revert that removes the matching output while preserving draw parity:

- C's plain `char` is signed for the smackheads percentage path
  (`src/structs.h:703`); Go now applies the same signed-byte interpretation.
  The isolated C-byte revert fails
  `switched-body-combat-skills-batch4-smackheads-success-depth`.
- C sends solo-kill XP text to a switched killer's descriptor but gates
  player-only XP state updates separately (`src/fight.c:1634-1653`,
  `src/comm.c:597-612`). The isolated XP revert fails the switched ambush
  witness.

The breed-killer witness is clean at 40 pulses and proves the registered
`SPECIAL(breed_killer)` directly calls `do_spike` with `SCMD_SPIKE`
(`src/spec_procs2.c:1679-1717`). The retained revert misses the stake output;
both runs have draw parity. Earlier 32-pulse runs were hollow and are not used
as evidence.

No C or oracle source was changed. No rulebook amendment is proposed. Zero new
PRs were opened; this work updates the existing DP-1535 PR #2712. The open PR
count was already four, so no additional PR was created.

## Verification

- `make fmt`
- `go build ./...`
- `go vet ./...`
- `GOEXPERIMENT=nojsonv2 go test ./...`
- `GOEXPERIMENT=nojsonv2 go test ./pkg/game/...`
- `golangci-lint run --concurrency 4 ./...` (0 issues)
- `make deadcode-ratchet` (50 baselined, 0 new)
- `make db-migrate-test`
- `GOEXPERIMENT=nojsonv2 make fidelity-depth` (7,598 proven/delegated; one
  unrelated pre-existing blocked case)

## Phase B next step

Regenerate the L1 recount from the newest full census dump at
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-15/deepseek-dp1542-full/census-dump`.
Subtract sites already cited in depth manifests using the frozen-list README
method, commit `docs/research/l1-recount-2026-11-16.tsv`, and update the
evidence ledger before working the largest uncovered source files in batches
of about 20 sites.
