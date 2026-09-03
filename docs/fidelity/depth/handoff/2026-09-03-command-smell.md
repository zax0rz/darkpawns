# Depth-fidelity handoff — `smell`

Date: 2026-09-03

Branch: `glm/depth-smell`

Feature PR: #1250 (open; security and lint green, test still pending)

Feature commit: `ac2d4cb87`

## Queue position and result

This session checked out `main`, pulled with `--ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter command table. The special-procedure
inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed cast-sleep
outlaw/reagent vehicle and was not repicked.

The next unclaimed source-order row was `smell` at
`src/interpreter.c:712`. The generic Go `do_action` path was already byte
faithful, so this is a pure-coverage slice: the feature PR adds durable
record-specific evidence and no behavior change. PR #1250 is intentionally
unmerged because its hosted test check remained pending after CI fired
normally. Do not merge PR #1250 or repick this row; continue with the next
source-order family, `smirk`, at `src/interpreter.c:713`, while #1250 remains
open.

Main frontier at handoff: 3,752 total, 3,651 proven/delegated, 48 blocked,
and 53 excluded; actionable completion 3,651/3,699 = 98.7%.

Feature-branch evidence frontier: 3,763 total, 3,662 proven/delegated, 48
blocked, and 53 excluded; actionable completion 3,662/3,710 = 98.7%.

The preceding `smackheads` implementation slice remains represented by open
PR #1248 and handoff PR #1249; neither is green/merged, and neither should be
merged while hosted checks remain non-green.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:712 */
{ "smell"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` finds the social, rejects
`PLR_NOSHOUT`, parses the first target token with `one_argument` when
`char_found` exists, and dispatches no-argument, target-not-found, self,
minimum-victim-position, or target-found actor/room/victim branches.

The `smell` record in `lib/misc/socials:1128-1136` is `smell 1 0`: its social
level metadata is 1, its hide flag is 0, its minimum victim position is 0, and
its eight authored messages are the no-argument whiff/sniff pair, target
crotch trio, missing-target invisible-man line, and self armpit pair.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/smell-depth.txt`;
- `docs/fidelity/depth/smell.tsv`; and
- `pkg/session/smell_depth_test.go`.

The clean-main `smell-depth` vehicle reported no normalized divergence at
seeds 1, 2, 3, 5, and 8. Seed 1 was inspected with `--show-oracle` and
confirmed all intended C blocks: no argument, first-token target with trailing
words ignored, target audiences, self target, and missing target. The existing
generic social implementation already matched, so no Go behavior fix was
confirmed or made; inventing one would violate R4. Shared position, noshout,
visibility, and social metadata behavior is delegated to existing proofs under
R5b/R5c. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (with the installed Go binary on PATH)
- `gofumpt -l .` clean
- `git diff --check`

PR #1250's hosted CI fired normally. Security and lint completed green; test
remained pending during the wait. The workflow was not retried because it did
fire, and the PR was not merged because all checks were not green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed matrix and audience topology), R4 (no invented behavior), R5/R5e
(verify the actual C path and let C win), and R5b/R5c (delegate shared social
gate, lookup, visibility, and metadata behavior).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and claim/audit
`smirk` at `src/interpreter.c:713`. Do not repick `smell`, `smackheads`,
`slap`, `slowns`, `slug`, or the shared `smile` social proof. Leave PR #1250
open until its hosted test check resolves; never merge it while non-green.
