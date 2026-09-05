# Depth-fidelity handoff — `snarl`

Date: 2026-09-03

Branch: `glm/depth-snarl`

Feature PR: #1258 (open; hosted checks pending)

Feature commit: `5fcaa1315`

## Queue position and result

This session checked out `main`, pulled with `--ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter table. The special-procedure
inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed cast-sleep
outlaw/reagent vehicle and was not repicked.

The next source-order row was `snarl` at `src/interpreter.c:718`; earlier
`smell`, `smirk`, `sneeze`, `sneer`, and `snicker` rows have open feature and
handoff PRs and must not be repicked. The generic Go `do_action` path was
already byte faithful, so this is a pure-coverage slice. PR #1258 is open
because hosted CI fired and remains pending; do not merge it until every
required check is green. Continue with `sneak` at `src/interpreter.c:719`.

Main frontier at handoff: 3,752 total, 3,651 proven/delegated, 48 blocked,
and 53 excluded; actionable completion 3,651/3,699 = 98.7%.

Feature-branch evidence frontier: 3,763 total, 3,662 proven/delegated, 48
blocked, and 53 excluded; actionable completion 3,662/3,710 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:718 */
{ "snarl"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` finds the social, rejects
`PLR_NOSHOUT`, parses the first target token with `one_argument`, and dispatches
no-argument, target-not-found, self, minimum-victim-position, or target-found
actor/room/victim branches.

The `snarl` record in `lib/misc/socials:745-753` is `snarl 0 0`: zero social
level metadata, zero hide flag, zero minimum victim position, and eight
authored messages for the animal no-argument snarl, target self-control trio,
`Eh? Who? Not here, my friend.` miss, and schizophrenia self pair.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/snarl-depth.txt`;
- `docs/fidelity/depth/snarl.tsv`; and
- `pkg/session/snarl_depth_test.go`.

The clean-main `snarl-depth` vehicle reported no normalized divergence at
seeds 1, 2, 3, 5, and 8. Seed 1 was inspected with `--show-oracle` and
confirmed all intended C blocks: no argument, first-token target with trailing
words ignored, target audiences, self target, and missing target. No Go
behavior fix was confirmed or made; inventing one would violate R4. Shared
position, noshout, visibility, and metadata behavior is delegated under
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

PR #1258's hosted CI fired normally and was pending at handoff. No workflow
retry was used because CI did fire. The PR was not merged because all checks
were not green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed matrix and audience topology), R4 (no invented behavior), R5/R5e
(verify the actual C path and let C win), and R5b/R5c (delegate shared social
gate, lookup, visibility, and metadata behavior).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and claim/audit
`sneak` at `src/interpreter.c:719`. Do not repick `snarl`, `snicker`, `sneer`,
`sneeze`, `smirk`, `smell`, `smackheads`, `slap`, `slowns`, `slug`, or the
shared `smile` social proof. Leave PR #1258 open until its hosted checks
resolve; never merge it while non-green.
