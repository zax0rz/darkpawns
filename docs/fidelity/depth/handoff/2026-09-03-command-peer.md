# Depth-fidelity handoff — `peer`

Date: 2026-09-03

Feature branch: `glm/depth-peer`
Feature PR: #1308 (merged green)
Feature commit: `fd981a62e`
Feature merge commit: `41dd95f94`

## Queue position and result

The special-procedure inventory remains exhausted. The one allowed
`objmagic.sleep-entry-gates` cast-sleep vehicle attempt remains blocked, and
the completed `pat` handoff is not to be repicked. The next dedicated
un-manifested interpreter-table family after `pat` was `peer` at
`src/interpreter.c:603`; `page`, `palm`, `parry`, `pardon`, and
`peek` were already claimed in source order.

The C row is `{ "peer", POS_RESTING, do_action, 0, 0 }`. The live path is
`src/act.social.c:102-151`: `find_action`, the `PLR_NOSHOUT` early
return, conditional `one_argument`, no-argument handling, visible-room
target lookup, not-found and self branches, the victim-position check, and
actor/observer/victim audience dispatch. The authored record at
`lib/misc/socials:560-568` is `peer 1 0`: C hide flag 1, no victim-position
restriction, and all eight message fields. The existing Go handler and
social record already matched this path; no Go behavior change was confirmed
or made. This preserves R1/R2/R3/R4/R5e and delegates shared gates/audience
machinery under R5b/R5c.

Added `docs/fidelity/depth/peer.tsv` with eleven durable rows: the C entry
gate, delegated command-position and `PLR_NOSHOUT` gates, no-argument output,
target success and audience topology, first-token parsing, NPC target, self,
missing target, and sleeping-target behavior. Added the annotated vehicle
`cmd/dp-oracle-diff/scenarios/peer-depth.txt` and
`TestPeerRegistrationUsesCEntryGateAndRecord` in
`pkg/session/peer_depth_test.go`.

## Proof and gates

This was a pure-coverage round: there was no source divergence to fix, so no
Go behavior changed. The scenario ran with
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` for seeds 1, 2, 3,
5, and 8. Seed 1 used `--show-oracle` and showed the intended C blocks for
no-argument, player, NPC, self, missing-target, and sleeping-target probes;
all five runs exited 0 with `result: no normalized divergence`.

Local gates passed: `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
`gofumpt -l .` (clean), and `git diff --check`. Hosted CI run
`33765493984` was green for lint, security, and test; build-and-push and
deploy were skipped by workflow policy. PR #1308 was self-merged only after
the applicable checks were green.

## Frontier and continuation

Before this slice, main reported 4,089 total, 3,985 proven/delegated, 48
blocked, and 56 excluded; actionable completion was 3,985/4,033 = 98.8%.
This slice added eleven proven/delegated rows. Main now reports 4,100 total,
3,996 proven/delegated, 48 blocked, and 56 excluded; actionable completion is
3,996/4,044 = 98.8%.

The next fresh main session must checkout/pull main, rerun the frontier, read
`DEPTH_TESTING.md` and the newest handoff, re-check open fidelity PRs, then
claim the next dedicated un-manifested interpreter-table family after
`peer` in source order. Re-audit `src/interpreter.c` and
`docs/fidelity/depth/*.tsv` at that time; do not infer the next item from
this note alone.
