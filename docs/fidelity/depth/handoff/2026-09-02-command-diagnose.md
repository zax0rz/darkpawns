# Depth handoff — diagnose

Date: 2026-09-02
Branch: `glm/depth-diagnose`
Feature PR: #1231 (merged green)
Feature commit: `cc6d91907`
Main merge: `dc16382d9`

## Queue position

The special-procedure inventory is exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, was attempted through the cast-sleep
outlaw/reagent vehicle and remains blocked for the unreachable entry gates.
After refreshing `main`, pulling with `--ff-only`, running `make fidelity-depth`,
reading `DEPTH_TESTING.md`, and reviewing the dated handoffs, the next live
interpreter-table family in source order was `diagnose` at
`src/interpreter.c:412`. The older `show` handoff was stale and its proposed
frontier did not override the live manifest or source-order sweep.

Pre-slice frontier: 3,620 total, 3,519 proven/delegated, 48 blocked, 53
excluded.

Post-slice frontier: 3,635 total, 3,534 proven/delegated, 48 blocked, 53
excluded; actionable completion is 3,534/3,582 = 98.7%.

The next session must refresh `main`, rerun the frontier, reread the guide and
newest handoff, then repeat the interpreter-table sweep while honoring explicit
family claims in existing handoffs. Do not repick stale `show`/`shoot` claims
or already-covered shared families.

## C call path and observable contract

The command table registers:

```c
{ "diagnose" , POS_RESTING , do_diagnose , 0, 0 }
```

at `src/interpreter.c:412`. `glance` is an explicit registration for the same
handler at `src/interpreter.c:466`; `diag` is accepted as the interpreter's
prefix alias and is also registered by the Go command surface.

`src/act.informative.c:2433-2455` calls `one_argument`, uses
`get_char_room_vis` for an explicit target, sends `NOPERSON` on a failed lookup,
and otherwise calls `diag_char_to_char`. With no argument, C passes the
`FIGHTING(ch)` pointer directly; if it is null, C sends `Diagnose who?`.

`diag_char_to_char` at `src/act.informative.c:354-385` computes
`(100 * GET_HIT(victim)) / GET_MAX_HIT(victim)`, using -1 when max hit points
are non-positive, and sends only to the diagnosing character. Its exact
inclusive bands are: excellent at 100 or more; few scratches at 90-99; small
wounds/bruises at 75-89; quite a few wounds at 50-74; big nasty
wounds/scratches at 30-49; pretty hurt at 15-29; awful at 0-14; and bleeding
awfully below 0. `PERS`/`CAP` supply the target name.

The Go implementation already matched the explicit lookup, messages, and
thresholds. The confirmed divergence was the no-argument fighting fallback:
Go stored a fighting mob by its multi-word short description and incorrectly
sent that description back through keyword lookup. The fix in
`pkg/game/look.go` uses the existing `ResolveFightingTarget` helper, preserving
the C pointer semantics without inventing a new target rule. No `src/` or
oracle-tree file was edited.

## Proof artifacts

Scenarios:

- `cmd/dp-oracle-diff/scenarios/diagnose-depth.txt`
- `cmd/dp-oracle-diff/scenarios/diagnose-player-depth.txt`

The scenarios cover no argument, fighting fallback, explicit mob/player
targets, trailing arguments, case-folded lookup, not-found output, `diag` and
`glance`, the resting entry gate, private output, and the full condition
thresholds. They use the registered horse mob fixture and an explicit player
target fixture.

Manifest: `docs/fidelity/depth/diagnose.tsv` (15 rows).

Focused tests:

- `pkg/game/diagnose_depth_test.go`
- `pkg/session/diagnose_depth_test.go`

With `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, seed 1 with
`--show-oracle` and seeds 2, 3, 5, and 8 produced
`result: no normalized divergence` for both scenarios. The pre-fix focused
RED test captured `No-one by that name here.` where C emitted the fighting
target's condition sentence.

## Gates and review

All local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` (clean)
- `git diff --check`

PR #1231's hosted lint, security, and test checks were green; build/deploy were
correctly skipped. The PR was self-merged only after the checks were green.

This slice follows R1 (player-facing bytes), R2 (the registered command
surface), R3 (deterministic threshold behavior), R4 (no invented behavior),
R5 (actual C call path), R5e (the C source wins), and R5b/R5c (a repeated or
shared-family result must be audited at the class level rather than repicked).
