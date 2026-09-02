# Depth-fidelity handoff — `chide`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the newest prior handoff, `2026-09-02-command-chuckle.md`. The special-procedure
inventory remains exhausted. The one-time blocked `objmagic.sleep-entry-gates`
row was already attempted through the cast-sleep outlaw/reagent vehicle and
was not repicked. The interpreter sweep consumed the next genuinely
unmanifested family, `chide`, at `src/interpreter.c:376`.

The pre-slice frontier was 3,582 total cases, with 3,481 proven/delegated, 48
blocked, and 53 excluded. The chide manifest contributes 10 fully
proven/delegated cases. The resulting frontier is:

- 3,592 total cases
- 3,491 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,491/3,539 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:376 */
{ "chide"    , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument`, then selects the no-argument, visible-target,
self-target, not-found, or target-position branch. The `chide` record in
`lib/misc/socials:1406-1414` has hide=0, no victim-position minimum, an
actor-only no-argument branch (`others_no_arg` is `#`), and all target,
observer, victim, self, and not-found message slots.

## Evidence and verification

The C-first vehicle is `cmd/dp-oracle-diff/scenarios/chide-depth.txt`. It uses
named actor, observer, and target peers and probes no argument, a visible
standing target, a fill-word/trailing-argument target, self-target, and a
missing target. The scenario was run on clean pre-change `main` with
`--show-oracle` at seed 1 and at seeds 2, 3, 5, and 8. Every run reported
`result: no normalized divergence`; the shown C blocks confirm the silent
no-argument room branch and each reachable target audience. No Go source
change was needed: the generic `DoAction` and existing chide record were
already byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/chide-depth.txt`;
- `docs/fidelity/depth/chide.tsv`; and
- `pkg/session/chide_depth_test.go`.

The manifest delegates the shared POS_RESTING command gate, PLR_NOSHOUT
boundary, and social lookup/visibility behavior to their existing owners
under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-chide`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `543d6b6c3` (`test: add chide depth fidelity coverage`).

Feature PR: #1223 (`glm/depth-chide`). Hosted lint, security, and test checks
completed green; conditional build-and-push and deploy jobs were skipped. CI
fired normally, so no workflow retry was used. The PR was self-merged only
after all applicable checks were green. The resulting `main` merge commit is
`5e33c8d00`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The live interpreter-table-versus-manifest sweep skips already covered
`charge` and `circle`, then finds `cheer` as the next genuinely unmanifested
family:

```c
/* src/interpreter.c:379 */
{ "cheer"    , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`cheer` before advancing in interpreter-table order.

