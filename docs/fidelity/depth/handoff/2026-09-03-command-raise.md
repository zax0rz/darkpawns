# Depth-fidelity handoff — `raise`

Date: 2026-09-03

Feature branch: `glm/depth-raise`
Feature PR: #1310 (merged green)
Feature commit: `8fe834835`
Feature merge commit: `2c2b63780`

## Queue position and result

The special-procedure inventory remains exhausted. The one allowed
`objmagic.sleep-entry-gates` cast-sleep vehicle attempt remains blocked, and
the completed `peer` handoff is not to be repicked. The next dedicated
un-manifested interpreter-table family after `peer` was `raise` at
`src/interpreter.c:635`; `pick` was already owned by the shared
`door.tsv` family, and `quest`/ `rest` were already covered by existing
manifests.

The C row is `{ "raise", POS_RESTING, do_action, 0, 0 }`. The live path is
`src/act.social.c:102-151`: `find_action`, the `PLR_NOSHOUT` early
return, conditional `one_argument`, no-argument handling, visible-room
target lookup, not-found and self branches, the victim-position check, and
actor/observer/victim audience dispatch. The authored record at
`lib/misc/socials:1322-1330` is `raise 0 5`: C hide flag 0,
POS_RESTING victim minimum, and all eight message fields. The existing Go
handler and social record already matched this path; no Go behavior change
was confirmed or made. This preserves R1/R2/R3/R4/R5e and delegates shared
gates/audience machinery under R5b/R5c.

Added `docs/fidelity/depth/raise.tsv` with eleven durable rows: the C entry
gate, delegated command-position and `PLR_NOSHOUT` gates, no-argument output,
target success and audience topology, first-token parsing, NPC target, self,
missing target, and sleeping-target rejection. Added the annotated vehicle
`cmd/dp-oracle-diff/scenarios/raise-depth.txt` and
`TestRaiseRegistrationUsesCEntryGateAndRecord` in
`pkg/session/raise_depth_test.go`.

## Proof and gates

This was a pure-coverage round: there was no source divergence to fix, so no
Go behavior changed. The focused metadata test initially caught and corrected
a draft-only interpretation error: C social value 5 is POS_RESTING, not
POS_STANDING. The final scenario ran with
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` for seeds 1, 2, 3,
5, and 8. Seed 1 used `--show-oracle` and showed the intended C blocks for
no-argument, player, NPC, self, missing-target, and sleeping-target probes;
all five runs exited 0 with `result: no normalized divergence`.

Local gates passed: `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
`gofumpt -l .` (clean), and `git diff --check`. Hosted CI run
`33767184741` was green for lint, security, and test; build-and-push and
deploy were skipped by workflow policy. The initial CI trigger did not report
checks, so the one permitted exact retry was used:
`gh workflow run "Dark Pawns CI/CD" --ref glm/depth-raise`. PR #1310 was
self-merged only after the retry checks were green.

## Frontier and continuation

Before this slice, main reported 4,100 total, 3,996 proven/delegated, 48
blocked, and 56 excluded; actionable completion was 3,996/4,044 = 98.8%.
This slice added eleven proven/delegated rows. Main now reports 4,111 total,
4,007 proven/delegated, 48 blocked, and 56 excluded; actionable completion is
4,007/4,055 = 98.8%.

The next fresh main session must checkout/pull main, rerun the frontier, read
`DEPTH_TESTING.md` and the newest handoff, re-check open fidelity PRs, then
re-audit `src/interpreter.c` against all
`docs/fidelity/depth/*.tsv` family command fields. The next presently
visible unmanifested row is `reel` at `src/interpreter.c:637`, but claim it
only after that fresh audit; do not repick any already-owned shared family.
