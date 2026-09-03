# Depth-fidelity handoff — `pace`

Date: 2026-09-03

Feature branch: `glm/depth-pace`
Feature PR: #1303 (merged green)
Feature commit: `49b18c563`
Feature merge commit: `80e334a77`

## Queue position and result

After the special-procedure inventory was exhausted and the one allowed
`objmagic.sleep-entry-gates` cast-sleep vehicle attempt remained blocked, the
next dedicated un-manifested interpreter-table family was `pace`. The earlier
`nudge` handoff PR #1300 initially had no checks; its one permitted workflow
retry was dispatched, and the checks later completed green, so it was merged
as `6fa74d82a` before this handoff. Do not repick either surface.

The C row is `src/interpreter.c:595`:
`{ "pace", POS_RESTING, do_action, 0, 0 }`.
The live path is `src/act.social.c:102-151`: `find_action`, the
`PLR_NOSHOUT` early return, the missing-`char_found` self-only branch, and its
actor/room dispatch. The authored record at `lib/misc/socials:1227-1231` is
`pace 0 5`: hide flag 0, victim-position metadata 5, actor line
`You pace back and forth.`, room line `$n paces back and forth.`, and a `#`
terminator. Since `char_found` is absent, C ignores visible, missing, and
self-named arguments and emits only the no-argument pair. No Go behavior
change was confirmed or made; no `src/` or C-oracle file was edited. This
preserves R1/R2/R3/R4/R5e and delegates shared behavior under R5b/R5c.

Added `docs/fidelity/depth/pace.tsv` with seven durable rows: the C entry
gate, delegated command-position and `PLR_NOSHOUT` gates, no-argument output,
and the visible-target, missing-target, and self-target argument-ignored
branches. Added the annotated vehicle
`cmd/dp-oracle-diff/scenarios/pace-depth.txt` and
`TestPaceRegistrationUsesCEntryGateAndRecord` in
`pkg/session/pace_depth_test.go`.

## Proof and gates

The scenario used
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` for seeds 1, 2, 3,
5, and 8. Seed 1 used `--show-oracle` and showed the intended C self-only
block for every probe; all five runs exited 0 with `result: no normalized divergence`.

Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
`git diff --check`. Hosted CI run `33761089952` was green for lint, security,
and test; build-and-push and deploy were skipped by workflow policy. PR #1303
was self-merged only after the applicable checks were green.

## Frontier and continuation

Before this slice, main reported 4,060 total, 3,956 proven/delegated, 48
blocked, and 56 excluded. This slice added seven proven/delegated rows. Main
now reports 4,067 total, 3,963 proven/delegated, 48 blocked, and 56 excluded;
actionable completion is 3,963/4,011 = 98.8%.

The next fresh main session must checkout/pull main, rerun the frontier, read
`DEPTH_TESTING.md` and the newest handoff, re-check open fidelity PRs, then
claim the next dedicated source-order family: `pant` at
`src/interpreter.c:596`. Its C record begins at `lib/misc/socials:1217` and
has a full target-social message matrix, so map that call path and enumerate
its target, audience, self, missing-target, position, and argument branches
before choosing the vehicle.
