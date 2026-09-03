# Depth-fidelity handoff — `sneak`

Date: 2026-09-03

Branch: `glm/depth-sneak`

Feature PR: #1260 (merged green)

Feature commits: `4e2715aa8`, `fee79421e`

Main merge: `d4061875b`

## Queue position and result

This round checked out `main`, pulled with `--ff-only`, ran `make
fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest dated
handoff, and audited the interpreter table in source order. The special-
procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its single permitted
cast-sleep outlaw/reagent vehicle and was not repicked.

The next unclaimed interpreter row was `sneak` at `src/interpreter.c:719`, a
dedicated `do_sneak` handler. It is now claimed and merged. The next
unclaimed source-order family is `sniff` at `src/interpreter.c:720`; the next
session must confirm that claim from a fresh `main` checkout before touching
it. Do not repick `sneak`.

## Frontier

Fresh post-merge `main` reports 3,760 total cases, 3,659 proven/delegated, 48
blocked, and 53 excluded. Actionable completion is 3,659/3,707 = 98.7%.
The sneak manifest contributes eight proven/delegated cases; the counts on
`main` include them after merge.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:719 */
{ "sneak"    , POS_STANDING, do_sneak    , 1, 0 },
```

`do_sneak` in `src/act.other.c:214-244` first rejects a mounted actor with
`Dismount first!`, then always sends `Okay, you'll try to move silently for a
while.`. If already affected, it removes both the ordinary `SKILL_SNEAK`
affect and the `SKILL_STEALTH` affect and clears `AFF_SNEAK`. It draws exactly
one `number(1,101)` and compares it with `GET_SKILL(SNEAK)` plus the C dex
skill bonus. Failure emits no additional bytes; success installs a
`SKILL_SNEAK` affect with `GET_LEVEL(ch)` duration and `AFF_SNEAK`.

The reachable branches are mounted early return, ordinary attempt message,
ignored trailing arguments, roll failure, roll success, affect replacement,
and the no-roll mounted boundary. The registered POS_STANDING entry gate is
covered locally; shared dispatcher position behavior is delegated to
`flip.position-gate` under R5b/R5c.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/thief-sneak.txt`;
- `cmd/dp-oracle-diff/scenarios/sneak-mounted-depth.txt`;
- `docs/fidelity/depth/sneak.tsv`;
- `pkg/session/sneak_depth_test.go`; and
- `pkg/game/skill_stealth_test.go`.

Both oracle vehicles reported no normalized divergence at seeds 1, 2, 3, 5,
and 8, with seed 1 inspected using `--show-oracle`. The ordinary vehicle was
already green on clean `main`, so this was an honest pure-coverage round and
made no behavior change; the mounted vehicle proved the dedicated early
return. Focused tests prove success affect duration/replacement, failed
reroll clearing, and no RNG draw on the mounted gate. No file under `src/` or
`darkpawns-c-oracle/` was edited.

## Gates and review

The feature branch passed `make fidelity-depth`, `go build ./...`, `go vet
./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .` (clean), and
`git diff --check`. PR #1260's hosted lint, security, and full test checks
were green; conditional build-and-push and deploy were skipped. CI fired
normally, so the one-time workflow retry was not used. The PR was self-merged
only after all applicable checks were green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed matrix and draw boundaries), R4 (no invented output), R5/R5e (actual
C call-path authority), and R5b/R5c (shared position-gate ownership).
