# Depth-fidelity handoff — `blame`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest available prior handoff,
`2026-09-02-command-bitch.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next genuinely unmanifested family,
`blame`, at `src/interpreter.c:357`; bellow and bird remain claimed by their
open feature handoffs after their one-time CI retries.

The pre-slice frontier was 3,435 total cases, with 3,334 proven/delegated, 48
blocked, and 53 excluded. The blame manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,446 total cases
- 3,345 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,345/3,393 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:357 */
{ "blame"    , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument` when `char_found` exists, then selects the no-argument,
visible-target, self-target, not-found, or target-position branch. The
`blame` record in `lib/misc/socials:1341-1349` has hide=0, a
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

- `cmd/dp-oracle-diff/scenarios/blame-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/blame-sleeping-depth.txt`;
- `docs/fidelity/depth/blame.tsv`; and
- `pkg/session/blame_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-blame`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `b6631e846`.

Feature PR: #1197 (`glm/depth-blame`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `30df27207e6075c525ba3124829de04f182065c4`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep finds `bleed` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:358 */
{ "bleed"    , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`bleed` before advancing in interpreter-table order.
