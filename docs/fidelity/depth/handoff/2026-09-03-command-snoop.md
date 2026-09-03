# Depth-fidelity handoff — snoop

Date: 2026-09-03

Branch: glm/depth-snoop

Feature PR: #1267 (merged green)

Feature commit: 153e585f5

Main merge: 095fea211

## Queue position and result

This round returned to main, ran git pull --ff-only, confirmed the frontier
with make fidelity-depth, reread docs/fidelity/DEPTH_TESTING.md and the newest
dated handoff, and then audited the interpreter table against the depth
manifests. The special-procedure inventory remains exhausted. The one blocked
row, objmagic.sleep-entry-gates, remains blocked after its one allowed
cast-sleep outlaw/reagent vehicle and was not repicked.

The next genuinely unclaimed source-order row after snoop is the shared social
snuggle at src/interpreter.c:724. The preceding snore and snowball rows are
covered by their manifests, and snoop is covered by snoop.tsv; do not repick
them.

Pre-slice frontier: 3,783 total, 3,680 proven/delegated, 48 blocked, and 55
excluded. The snoop manifest adds 15 proven/delegated cases. Post-slice
frontier: 3,798 total, 3,695 proven/delegated, 48 blocked, and 55 excluded;
actionable completion is 3,695/3,743 = 98.7%.

## C call path and observable contract

The registered C row at src/interpreter.c:723 is:

    { "snoop"    , POS_DEAD    , do_snoop   , LVL_GOD, 0 },

do_snoop in src/act.wizard.c:1120-1170 begins with the descriptor early
return, parses one argument with one_argument, and then follows the actual
branch order:

- no argument calls stop_snooping, emitting either You aren't snooping
  anyone. or You stop snooping. and clearing both links;
- an unresolved name emits No such person around.;
- a visible NPC or linkless player emits There's no link.. nothing to snoop.;
- naming the snooper itself calls stop_snooping;
- a target whose descriptor already has snoop_by emits Busy already. ;
- a target already snooping this descriptor emits Don't be stupid.;
- the target's switched original, when present, supplies the level comparison;
  equal-or-higher targets emit You can't.;
- success emits Okay., clears a prior target link, and installs the
  bidirectional snooping/snoop_by descriptor links.

src/comm.c:1646-1651 forwards flushed victim output to the snooper as
percent-prefix plus output plus percent-terminator; src/comm.c:1992-1998
forwards raw victim input with percent-prefix and CRLF before command routing.
These are D5 descriptor paths, not optional UI behavior.

## Evidence and implementation boundary

The durable evidence is:

- cmd/dp-oracle-diff/scenarios/snoop-depth.txt;
- docs/fidelity/depth/snoop.tsv; and
- pkg/session/snoop_depth_test.go.

The clean-main vehicle was RED on the confirmed Go divergences: invented
online-only responses, missing NPC/no-link resolution, toggle-on-repeat
instead of C's busy branch, missing self-stop and level gates, and absent
snooped output delimiters. After the Go-only fix, the scenario reported
result: no normalized divergence at seeds 1, 2, 3, 5, and 8, with seed 1
inspected using --show-oracle. The focused tests pin the reciprocal state
branches, switched-original level comparison, and raw input framing. No file
under src/ or darkpawns-c-oracle/ was edited.

## Gates and review

The final local gates passed on the feature branch:

- make fidelity-depth
- go build ./...
- go vet ./...
- go test ./...
- golangci-lint run ./... — 0 issues
- gofumpt -l . clean
- git diff --check

PR #1267's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was needed. The PR was self-merged only after the applicable
checks were green, per the 2026-08-27 amendment.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (descriptor ordering and multi-seed proof), R4 (no invented behavior), and
R5/R5e (verify the actual C path and let C win), with R5b/R5c applied to the
shared session output and cleanup behavior.

## Continuation

The next session must checkout main, pull with --ff-only, rerun
make fidelity-depth, reread the guide and newest handoff, and audit/claim
snuggle at src/interpreter.c:724 before touching implementation. Do not repick
snoop or its shared descriptor relay.
