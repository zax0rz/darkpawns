# Depth-fidelity handoff — `bless`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest available prior handoff,
`2026-09-02-command-bleed.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next genuinely unmanifested family,
`bless`, at `src/interpreter.c:359`.

The pre-slice frontier was 3,457 total cases, with 3,356 proven/delegated, 48
blocked, and 53 excluded. The bless manifest contributes 11 fully
proven/delegated cases, giving the feature branch frontier:

- 3,468 total cases
- 3,367 proven/delegated
- 48 blocked
- 53 excluded

During the merge, `main` also advanced with previously claimed bellow and bird
evidence. The final checked-in frontier on `main` is therefore:

- 3,490 total cases
- 3,389 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,389/3,437 = 98.6%. The additional 22 cases are
bellow and bird work from their existing claims, not a change to the bless
slice.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:359 */
{ "bless"    , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument` when `char_found` exists, then selects the no-argument,
visible-target, self-target, not-found, or target-position branch. The `bless`
record in `lib/misc/socials:1526-1534` has hide=0, a zero victim-position
minimum, and all eight authored message slots. A sleeping target therefore
reaches the target branch; C's `SENDOK` delivery suppresses the victim line
while actor and awake observer output remain.

## Evidence and verification

The direct three-player vehicle proves no argument, visible target, observer,
one-token parsing, self-target, and not-found behavior. The sleeping-target
vehicle proves that a sleeping target is admitted, that actor and observer
receive their authored target messages, and that the sleeping target receives
no victim bytes. Both vehicles were run on pre-change `main` first at seeds 1,
2, 3, 5, and 8; seed 1 for each used `--show-oracle` to confirm the intended
C blocks. Every run reported `result: no normalized divergence`. No Go
behavior change was needed: the generic `DoAction` implementation was already
byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/bless-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/bless-sleeping-depth.txt`;
- `docs/fidelity/depth/bless.tsv`; and
- `pkg/session/bless_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup, and
Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-bless`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `97972e968`.

Feature PR: #1201 (`glm/depth-bless`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `fc157a7839127b45fae27b3ea25a1a9345e2f5d3`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep finds `blink` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:360 */
{ "blink"    , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`blink` before advancing in interpreter-table order.

