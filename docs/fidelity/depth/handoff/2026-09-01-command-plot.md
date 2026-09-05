# Depth-fidelity handoff — `plot`

Date: 2026-09-01
Branch: `glm/depth-plot`
Feature commit: `97d99c31a`
Feature PR: #1063 (merged to `main` as `514af77a5`)

## Frontier

The clean-main frontier before this slice was 2,765 total cases: 2,694
proven/delegated, 22 blocked, and 49 excluded. After merging `plot`, it is
2,776 total: 2,705 proven/delegated, 22 blocked, and 49 excluded. Actionable
completion is 2,705/2,727 (99.2%). The special-procedure inventory remains
exhausted, and `objmagic.sleep-entry-gates` remains the single explicitly
blocked vehicle.

## C-first call path

The registration at `src/interpreter.c:607` is `{ "plot", POS_RESTING,
do_action, 0, 0 }`. `src/act.social.c:102-151` applies the shared
`PLR_NOSHOUT` gate, selects the `plot` social record, parses the first target
with `one_argument`, and emits the no-argument, not-found, self, or target
actor/room/victim audience branch. The exact eight authored messages are in
`lib/misc/socials:1098-1106`; `plot 1 0` means a target-aware record with no
invisible hiding and no minimum victim position.

## RED and confirmed fixes

The first live vehicle was GREEN across all reachable plot branches. No
confirmed Go divergence was found, so this slice is evidence-only: it adds
the source-order manifest, an exact command/social registration test, and a
named peer/NPC oracle vehicle. No oracle or C source was edited and no
player-facing behavior was invented.

## Evidence

- Scenario: `cmd/dp-oracle-diff/scenarios/plot-depth.txt`, covering no
  argument, visible player and NPC targets, named self, `self`, missing
  target, and fill-word/trailing-argument parsing with actor/room/victim
  captures.
- Manifest: `docs/fidelity/depth/plot.tsv` (11/11 proven or delegated rows).
- Focused test: `pkg/session/plot_depth_test.go`.
- Oracle matrix: `plot-depth@1,2,3,5,8`, all `result: no normalized divergence`;
  `--show-oracle --seed 1` confirmed every intended C social block.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`.
- Hosted checks for PR #1063 were green: lint, security, and test passed;
  build/deploy were skipped by the workflow. CI fired normally, so no retry
  was required.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (the registered command
surface), R3 (multi-seed audience and parser parity), R4 (no invented
behavior), and R5/R5e (verify the actual C call path). Shared social gates,
visibility, and audience machinery remain delegated under R5b/R5c.

## Next queue position

Return to clean `main`, pull, run `make fidelity-depth`, reread the depth guide
and this newest handoff, then resweep `src/interpreter.c` against all depth
manifests. `plot` is claimed; `peer` is a shared social, `pick` is owned by
the door manifest, and `players` is already claimed. The next unclaimed
command family in table order is `pinch` at `interpreter.c:605`; do not
re-pick any command already owned by a manifest or delegated boundary.
