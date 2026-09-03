# Depth-fidelity handoff — `muhaha`

Date: 2026-09-03

Feature branch: `glm/depth-muhaha`
Feature PR: #1293 (merged green)
Feature commit: `f1ef4e5ee`
Feature merge commit: `22a6f66e1`

## Queue position and result

`muhaha` was the next dedicated un-manifested interpreter-table family after the already-claimed `smackheads` slice; the `:` alias remains owned by the existing emote/communication proof. The special-procedure inventory remains exhausted, and `objmagic.sleep-entry-gates` remains blocked after its one allowed cast-sleep outlaw/reagent attempt. Do not repick either surface.

The C row is `src/interpreter.c:560`:
`{ "muhaha", POS_RESTING, do_action, 0, 0 }`.
The C call path is `src/act.social.c:102-151`: `find_action`, PLR_NOSHOUT early return, then the `char_found`-absent branch at lines 121-124 forces an empty argument and emits only the no-argument actor/room pair. The authored record at `lib/misc/socials:495-498` is `muhaha 0 0`, `MUHAHAHAHA!!!!!!!`, `$n throws $s head back and laughs with draconian terror.`, `#`. Therefore visible-target lookup, not-found, self-target, sleeping-target, and victim-success branches are unreachable for this family; typed inputs were explicitly proven to be ignored.

Added `docs/fidelity/depth/muhaha.tsv` with seven durable rows: entry gate, delegated shared position gate, no-argument actor/observer audience, ignored visible argument, ignored missing target, ignored self-looking argument, and delegated PLR_NOSHOUT gate. Added the annotated vehicle `cmd/dp-oracle-diff/scenarios/muhaha-depth.txt` and `TestMuhahaRegistrationUsesCEntryGateAndRecord` in `pkg/session/muhaha_depth_test.go`. No Go behavior change was confirmed or made; no `src/` or C-oracle file was edited. This preserves R1/R2/R3/R4/R5e and the shared-class boundaries R5b/R5c.

## Proof and gates

The scenario was run with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` for seeds 1, 2, 3, 5, and 8. Seed 1 used `--show-oracle` and showed the intended C blocks; every run exited 0 with `result: no normalized divergence`.

Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`. The shell PATH did not expose `go`, so the installed `/usr/local/go/bin/go` toolchain was used explicitly; this was an environment detail, not a gate failure. Hosted CI run `33752972896` was green for lint, security, and test; build-and-push and deploy were skipped by workflow policy. The PR was self-merged only after all applicable checks were green.

## Frontier and continuation

Before this slice, main reported 4,009 total, 3,905 proven/delegated, 48 blocked, and 56 excluded. This slice added seven proven/delegated rows. Current main reports 4,016 total, 3,912 proven/delegated, 48 blocked, and 56 excluded; actionable completion is 3,912/3,960 = 98.8%.

The next fresh main session must checkout/pull main, rerun the frontier, read `DEPTH_TESTING.md` and the newest handoff, then claim the next dedicated source-order family: `mumble` at `src/interpreter.c:561`. Do not repick `muhaha`.
