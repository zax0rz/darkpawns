# Depth-fidelity handoff — `bow`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the newest prior handoff, `2026-09-02-command-boggle.md`. The special-procedure
inventory remains exhausted. The one-time blocked `objmagic.sleep-entry-gates`
row was already attempted through the cast-sleep outlaw/reagent vehicle and
was not repicked. The interpreter sweep consumed the next genuinely
unmanifested family, `bow`, at `src/interpreter.c:365`.

The pre-slice frontier was 3,539 total cases, with 3,438 proven/delegated, 48
blocked, and 53 excluded. The bow manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,550 total cases
- 3,449 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,449/3,497 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:365 */
{ "bow"      , POS_STANDING, do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument` when `char_found` exists, then selects the no-argument,
visible-target, self-target, not-found, or target-position branch. The
`bow` record in `lib/misc/socials:76-84` has hide=5, a `POS_STANDING` victim
minimum, and all eight authored message slots. A sleeping target therefore
takes C's position failure line and receives no target or observer social
output; trailing words after the first target token are ignored by C.

## Evidence and verification

The direct three-player vehicle proves no argument, visible standing target,
observer, one-token parsing, self-target, and not-found behavior. The
sleeping-target vehicle proves the standing target-position gate and the
absence of target and observer bytes. Both vehicles were run on clean
pre-change `main` at seeds 1, 2, 3, 5, and 8; seed 1 for each used
`--show-oracle` to confirm the intended C blocks. Every run reported
`result: no normalized divergence`. No Go behavior change was needed: the
generic `DoAction` implementation was already byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/bow-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/bow-sleeping-depth.txt`;
- `docs/fidelity/depth/bow.tsv`; and
- `pkg/session/bow_depth_test.go`.

The manifest delegates shared POS_STANDING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c. The focused test
also pins the C entry registration and the authored social record, including
the C hide flag and victim-position gate.

## Integration and gates

The required local gates passed on `glm/depth-bow`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `fde82d75a` (`test: add bow depth fidelity coverage`).

Feature PR: #1213 (`glm/depth-bow`). Hosted lint, security, and test checks
completed green; conditional build-and-push and deploy jobs were skipped. CI
fired normally, so no workflow retry was used. The PR was self-merged only
after all applicable checks were green. The resulting `main` merge commit is
`b3adc03a0`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep finds `brief` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:366 */
{ "brief"    , POS_DEAD    , do_gen_tog  , 0, SCMD_BRIEF },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`brief` before advancing in interpreter-table order.

