# Depth-fidelity handoff — `snore`

Date: 2026-09-03

Branch: `glm/depth-snore`

Feature PR: #1263 (merged green)

Feature commit: `a40fef9ca`

Main merge: `2c93d8b7b`

## Queue position and result

This round checked out `main`, pulled with `--ff-only`, ran `make
fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest dated
handoff, and audited the interpreter table in source order. The special-
procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its single permitted
cast-sleep outlaw/reagent vehicle and was not repicked.

The next unclaimed interpreter row was `snore` at `src/interpreter.c:721`, a
`do_action` social with a POS_SLEEPING entry gate. It is now claimed and
merged. The next unclaimed source-order family is the dedicated immortal-only
`snowball` at `src/interpreter.c:722`; the next session must confirm that claim
from a fresh `main` checkout before touching it. Do not repick `snore`.

## Frontier

Fresh post-merge `main` reports 3,774 total cases, 3,671 proven/delegated, 48
blocked, and 55 excluded. Actionable completion is 3,671/3,719 = 98.7%.
The snore manifest contributes seven proven/delegated cases; the counts on
`main` include them after merge.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:721 */
{ "snore"    , POS_SLEEPING, do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social record, checks
`PLR_NOSHOUT`, and then sees that the `snore` record at
`lib/misc/socials:779-783` has no `char_found`, `others_found`, `vict_found`,
`not_found`, or self-target messages. It therefore ignores every typed
argument and always sends `Zzzzzzzzzzzzzzzzz.` to the actor followed by
`$n snores loudly.` to the room.

The reachable branches are the POS_SLEEPING entry gate, no-argument output,
argument-ignored output, and actor/room audience split. Target lookup, victim,
self, and missing-target branches are unreachable for this social record and
are explicitly excluded under R4/R5e. Shared position and noshout gates are
delegated to `group.position-gate` and `dance-noshout` under R5b/R5c.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/snore-depth.txt`;
- `docs/fidelity/depth/snore.tsv`; and
- `pkg/session/snore_depth_test.go`.

The oracle vehicle reported no normalized divergence at seeds 1, 2, 3, 5,
and 8, with seed 1 inspected using `--show-oracle`. Clean `main` was already
green, so this was an honest pure-coverage round and made no behavior change.
No file under `src/` or `darkpawns-c-oracle/` was edited.

## Gates and review

The feature branch passed `make fidelity-depth`, `go build ./...`, `go vet
./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .` (clean), and
`git diff --check`. PR #1263's hosted lint, security, and full test checks
were green; conditional build-and-push and deploy were skipped. The initial
check query returned no checks, so the one permitted
`gh workflow run "Dark Pawns CI/CD" --ref glm/depth-snore` retry was issued;
the resulting run completed green. The PR was self-merged only after all
applicable checks were green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed matrix), R4 (no invented output), R5/R5e (actual C call-path
authority), and R5b/R5c (shared gate and social ownership).
