# Depth-fidelity handoff — `nuzzle`

Date: 2026-09-03

Feature branch: `glm/depth-nuzzle`
Feature PR: #1301 (merged green)
Feature commit: `03862fab4`
Feature merge commit: `cb210dc91`

## Queue position and result

`nuzzle` was the next dedicated un-manifested interpreter-table family after `nudge`; the `murder` alias is owned by `combat-entry`, while `order`, `orgasm`, and `offer` are already covered by their existing handler families. The special-procedure inventory remains exhausted, and `objmagic.sleep-entry-gates` remains blocked after its one allowed cast-sleep outlaw/reagent attempt. Do not repick those surfaces. The separate `nudge` handoff PR #1300 remains open after its single permitted CI retry because no checks became green; that PR is intentionally not merged or repicked.

The C row is `src/interpreter.c:584`:
`{ "nuzzle", POS_RESTING, do_action, 0, 0 }`.
The C call path is `src/act.social.c:102-151`: `find_action`, PLR_NOSHOUT early return, one-argument parsing, visible-room target lookup, self handling, victim-position gate, and actor/room/victim audience dispatch. The authored record at `lib/misc/socials:540-548` is `nuzzle 1 5`: hide flag 1, victim-position minimum POS_RESTING (5), a `#` no-argument room branch, target-success messages, a record-specific missing-target response, an impossible self-target message, and all eight authored fields. No Go behavior change was confirmed or made; no `src/` or C-oracle file was edited. This preserves R1/R2/R3/R4/R5e and shared boundaries R5b/R5c.

Added `docs/fidelity/depth/nuzzle.tsv` with eleven durable rows: entry gate, delegated position and PLR_NOSHOUT gates, no-argument output, player target success and audience topology, first-token parsing, NPC target, self, missing target, and sleeping-target rejection. Added the annotated vehicle `cmd/dp-oracle-diff/scenarios/nuzzle-depth.txt` and `TestNuzzleRegistrationUsesCEntryGateAndRecord` in `pkg/session/nuzzle_depth_test.go`.

## Proof and gates

The scenario was run with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` for seeds 1, 2, 3, 5, and 8. Seed 1 used `--show-oracle` and showed the intended C blocks for no-arg, player target, NPC target, self, missing, and sleeping-target probes; every run exited 0 with `result: no normalized divergence`.

Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`. The shell PATH did not expose `go`, so the installed `/usr/local/go/bin/go` toolchain was used explicitly; this was an environment detail, not a gate failure. Hosted CI run `33759199911` was green for lint, security, and test; build-and-push and deploy were skipped by workflow policy. The PR was self-merged only after all applicable checks were green.

## Frontier and continuation

Before this slice, main reported 4,049 total, 3,945 proven/delegated, 48 blocked, and 56 excluded. This slice added eleven proven/delegated rows. Current main reports 4,060 total, 3,956 proven/delegated, 48 blocked, and 56 excluded; actionable completion is 3,956/4,004 = 98.8%.

The next fresh main session must checkout/pull main, rerun the frontier, read `DEPTH_TESTING.md` and the newest handoff, re-check open fidelity PRs, then claim the next dedicated source-order family: `pace` at `src/interpreter.c:595`. Do not repick `nuzzle`, `nudge`, or the already-owned `murder` alias.
