# Depth-fidelity handoff — `beckon`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest prior handoff,
`2026-09-02-command-bounce.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next genuinely unmanifested family,
`beckon`, at `src/interpreter.c:349`; `ask` and `auction` remain covered by
their existing shared communication manifests.

The pre-slice frontier was 3,402 total cases, with 3,301 proven/delegated, 48
blocked, and 53 excluded. The beckon manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,413 total cases
- 3,312 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,312/3,360 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:349 */
{ "beckon"   , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument` when `char_found` exists, then selects the no-argument,
visible-target, self-target, not-found, or target-position branch. The
`beckon` record in `lib/misc/socials:1208-1215` has hide=0, a
`POS_STANDING` victim-position minimum, and all eight authored message slots.
Thus a sleeping target takes C's position failure line and receives no target
or observer social output.

## Evidence and verification

The direct three-player vehicle proves no argument, visible target, observer,
one-token parsing, self-target, and not-found behavior. The sleeping-target
vehicle proves the standing target-position gate and the absence of target and
observer bytes. Both vehicles were run on pre-change `main` first and on the
feature branch at seeds 1, 2, 3, 5, and 8; seed 1 for each used
`--show-oracle`. Every completed run reported `result: no normalized
divergence`, and the shown C blocks confirmed the intended branches. No Go
behavior change was needed: the generic `do_action` implementation was
already byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/beckon-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/beckon-sleeping-depth.txt`;
- `docs/fidelity/depth/beckon.tsv`; and
- `pkg/session/beckon_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-beckon`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `08caebced`.

Feature PR: #1187 (`glm/depth-beckon`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `3780309323925b14cc697ef8ebe7f8d79190e47a`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented self-target output), R5/R5e (the
actual C call path), and R5b/R5c (shared social gate, lookup, and audience
ownership).

## Continuation

The source-order sweep finds `beg` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:351 */
{ "beg"      , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`beg` before advancing in interpreter-table order.
