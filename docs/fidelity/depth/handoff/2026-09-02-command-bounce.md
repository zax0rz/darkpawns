# Depth-fidelity handoff — `bounce`

Date: 2026-09-02

## Queue position and result

This round started from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest prior handoff,
`2026-09-02-command-anguish.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next genuinely unmanifested family,
`bounce`, at `src/interpreter.c:342`; `ask` and `auction` were skipped because
their shared `do_spec_comm`/`do_gen_comm` families are already covered by
existing manifests.

The pre-slice frontier was 3,383 total cases, with 3,282 proven/delegated, 48
blocked, and 53 excluded. The bounce manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,394 total cases
- 3,293 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,293/3,341 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:342 */
{ "bounce"   , POS_STANDING, do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT and actor-position gates, parses the
first target token with `one_argument` when `char_found` exists, then selects
the no-argument, visible-target, self-target, not-found, or target-position
branch. The `bounce` record in `lib/misc/socials:66-75` has hide=0,
`min_victim_position=0`, and all eight authored messages. A sleeping target
therefore reaches the target arm; C's `SENDOK` delivery suppresses the
`TO_VICT` line for the sleeping victim while actor and awake observer output
remain.

## Evidence and verification

The direct three-player vehicle proves no argument, visible target, observer,
one-token parsing, self-target, and not-found behavior. The sleeping-target
vehicle proves actor and observer delivery plus the silent sleeping victim.
Both vehicles were run on pre-change `main` first and on the feature branch at
seeds 1, 2, 3, 5, and 8; seed 1 for each used `--show-oracle`. Every completed
run reported `result: no normalized divergence`, and the shown C blocks
confirmed the intended branches. One parallel baseline attempt hit an
`Address already in use` oracle startup collision at direct seed 3; that seed
was rerun alone with the configured oracle and passed. No Go behavior change
was needed: the generic `do_action` implementation was already byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/bounce-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/bounce-sleeping-depth.txt`;
- `docs/fidelity/depth/bounce.tsv`; and
- `pkg/session/bounce_depth_test.go`.

The manifest delegates shared POS_STANDING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-bounce`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `4a23221e8`.

Feature PR: #1183 (`glm/depth-bounce`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `fd1c4ac8e902a0181ad3824f212562c7029f299e`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep finds `bah` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:344 */
{ "bah"      , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`bah` before advancing in interpreter-table order.
