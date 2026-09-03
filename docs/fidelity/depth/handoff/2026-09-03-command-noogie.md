# Depth-fidelity handoff — `noogie`

Date: 2026-09-03

Feature branch: `glm/depth-noogie`
Feature PR: #1297 (merged green)
Feature commit: `1dd62fc9e`
Feature merge commit: `4a379fa62`

## Queue position and result

`noogie` was the next dedicated un-manifested interpreter-table family after `mumble`; the `murder` alias is owned by `combat-entry`, and earlier `nibble`/`nod` families are already manifested. The special-procedure inventory remains exhausted, and `objmagic.sleep-entry-gates` remains blocked after its one allowed cast-sleep outlaw/reagent attempt. Do not repick those surfaces.

The C row is `src/interpreter.c:580`:
`{ "noogie", POS_RESTING, do_action, 0, 0 }`.
The C call path is `src/act.social.c:102-151`: `find_action`, PLR_NOSHOUT early return, one-argument parsing, visible-room target lookup, self handling, victim-position gate, and actor/room/victim audience dispatch. The authored record at `lib/misc/socials:1441-1449` has hide 0, victim-position minimum 0, a `#` no-argument room branch, three target-success messages, an arm-failure self branch, a record-specific missing-target message, and all eight authored fields. No Go behavior change was confirmed or made; no `src/` or C-oracle file was edited. This preserves R1/R2/R3/R4/R5e and shared boundaries R5b/R5c.

Added `docs/fidelity/depth/noogie.tsv` with eleven durable rows: entry gate, delegated position and PLR_NOSHOUT gates, no-argument output, player target success and audience topology, first-token parsing, NPC target, self, missing target, and sleeping-target behavior. Added the annotated vehicle `cmd/dp-oracle-diff/scenarios/noogie-depth.txt` and `TestNoogieRegistrationUsesCEntryGateAndRecord` in `pkg/session/noogie_depth_test.go`.

## Proof and gates

The scenario was run with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` for seeds 1, 2, 3, 5, and 8. Seed 1 used `--show-oracle` and showed the intended C blocks for no-arg, player target, NPC target, self, missing, and sleeping-target probes; every run exited 0 with `result: no normalized divergence`.

Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`. The shell PATH did not expose `go`, so the installed `/usr/local/go/bin/go` toolchain was used explicitly; this was an environment detail, not a gate failure. Hosted CI run `33756229237` was green for lint, security, and test; build-and-push and deploy were skipped by workflow policy. The PR was self-merged only after all applicable checks were green.

## Frontier and continuation

Before this slice, main reported 4,027 total, 3,923 proven/delegated, 48 blocked, and 56 excluded. This slice added eleven proven/delegated rows. Current main reports 4,038 total, 3,934 proven/delegated, 48 blocked, and 56 excluded; actionable completion is 3,934/3,982 = 98.8%.

The next fresh main session must checkout/pull main, rerun the frontier, read `DEPTH_TESTING.md` and the newest handoff, re-check open fidelity PRs, then claim the next dedicated source-order family: `nudge` at `src/interpreter.c:582`. Do not repick `noogie` or the already-owned `murder` alias.
