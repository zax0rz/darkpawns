# Depth-fidelity handoff — `cheer`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the newest prior handoff, `2026-09-02-command-chide.md`. The special-procedure
inventory remains exhausted. The one-time blocked `objmagic.sleep-entry-gates`
row was already attempted through the cast-sleep outlaw/reagent vehicle and
was not repicked. The interpreter sweep consumed the next genuinely
unmanifested family, `cheer`, at `src/interpreter.c:379`.

The pre-slice frontier was 3,592 total cases, with 3,491 proven/delegated, 48
blocked, and 53 excluded. The cheer manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,603 total cases
- 3,502 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,502/3,550 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:379 */
{ "cheer"    , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument`, then selects the no-argument, visible-target,
self-target, not-found, or target-position branch. The cheer record in
`lib/misc/socials:1158-1166` has hide=0, a `POS_STANDING` victim-position
minimum, and all eight authored messages. Its visible-target path sends
distinct actor, observer, and victim bytes; its self and not-found paths use
their authored records; and a sleeping target reaches only C's canonical
position-failure line.

## Evidence and verification

The C-first vehicles are `cmd/dp-oracle-diff/scenarios/cheer-depth.txt` and
`cmd/dp-oracle-diff/scenarios/cheer-sleeping-depth.txt`. They use named actor,
observer, target, and sleeper peers and probe no argument, visible target,
fill-word/trailing parsing, self-target, not-found, and sleeping-target
branches. Both vehicles were run on clean pre-change `main` with
`--show-oracle` at seed 1 and at seeds 2, 3, 5, and 8. Every run reported
`result: no normalized divergence`, and the shown C blocks confirm the exact
audience bytes and sleeping-target gate. No Go source change was needed: the
generic `DoAction` and existing cheer record were already byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/cheer-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/cheer-sleeping-depth.txt`;
- `docs/fidelity/depth/cheer.tsv`; and
- `pkg/session/cheer_depth_test.go`.

The manifest delegates the shared POS_RESTING command gate, PLR_NOSHOUT
boundary, and social lookup/visibility behavior to their existing owners
under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-cheer`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `22c4812b2` (`test: add cheer depth fidelity coverage`).

Feature PR: #1225 (`glm/depth-cheer`). Hosted lint, security, and test checks
completed green; conditional build-and-push and deploy jobs were skipped. CI
fired normally, so no workflow retry was used. The PR was self-merged only
after all applicable checks were green. The resulting `main` merge commit is
`34f67ac15`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The live interpreter-table-versus-manifest sweep skips covered command rows
through `clap`, then finds `credits` as the next genuinely unmanifested
family:

```c
/* src/interpreter.c:396 */
{ "credits"  , POS_DEAD    , do_gen_ps   , 0, SCMD_CREDITS },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`credits` before advancing in interpreter-table order.

