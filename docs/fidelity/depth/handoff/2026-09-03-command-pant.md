# Depth-fidelity handoff — `pant`

Date: 2026-09-03

Feature branch: `glm/depth-pant`
Feature PR: #1305 (merged green)
Feature commit: `97c37e118`
Feature merge commit: `4c4fb98ef`

## Queue position and result

The special-procedure inventory remains exhausted. The one allowed
`objmagic.sleep-entry-gates` cast-sleep vehicle attempt remains blocked, and
the completed `nudge` handoff is not to be repicked. The next dedicated
un-manifested interpreter-table family after `pace` was `pant`.

The C row is `src/interpreter.c:596`:
`{ "pant", POS_RESTING, do_action, 0, 0 }`.
The live path is `src/act.social.c:102-151`: `find_action`, the
`PLR_NOSHOUT` early return, first-token parsing, visible-room target lookup,
self handling, the victim-position gate, and actor/observer/victim audience
dispatch. The authored record at `lib/misc/socials:1217-1225` is `pant 0 5`:
hide flag 0, POS_RESTING victim-position minimum, all eight authored fields,
and the exact self-target spelling `You quitely pant to yourself.`. No Go
behavior change was confirmed or made; no `src/` or C-oracle file was edited.
This preserves R1/R2/R3/R4/R5e and delegates shared behavior under R5b/R5c.

Added `docs/fidelity/depth/pant.tsv` with twelve durable rows: the C entry
gate, delegated command-position and `PLR_NOSHOUT` gates, no-argument output,
target success and audience topology, first-token parsing, NPC target, self,
missing target, and sleeping-target rejection. Added the annotated vehicle
`cmd/dp-oracle-diff/scenarios/pant-depth.txt` and
`TestPantRegistrationUsesCEntryGateAndRecord` in
`pkg/session/pant_depth_test.go`.

## Proof and gates

The scenario used
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` for seeds 1, 2, 3,
5, and 8. Seed 1 used `--show-oracle` and showed the intended C blocks for
no-argument, player, NPC, self, missing-target, and sleeping-target probes;
all five runs exited 0 with `result: no normalized divergence`.

Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
`git diff --check`. Hosted CI run `33762588022` was green for lint, security,
and test; build-and-push and deploy were skipped by workflow policy. PR #1305
was self-merged only after the applicable checks were green.

## Frontier and continuation

Before this slice, main reported 4,067 total, 3,963 proven/delegated, 48
blocked, and 56 excluded. This slice added eleven proven/delegated rows. Main
now reports 4,078 total, 3,974 proven/delegated, 48 blocked, and 56 excluded;
actionable completion is 3,974/4,022 = 98.8%.

The next fresh main session must checkout/pull main, rerun the frontier, read
`DEPTH_TESTING.md` and the newest handoff, re-check open fidelity PRs, then
claim the next dedicated source-order family: `pat` at
`src/interpreter.c:597`. Its C record is `pat 0 0` at
`lib/misc/socials:550-558`: hide flag 0, no victim-position restriction, and
a full target-social matrix. Map that call path before choosing the vehicle.
