# Depth-fidelity handoff — `bird`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest available prior handoff,
`2026-09-02-command-beg.md`; the immediately preceding bellow handoff was
also read from its unmerged handoff branch. The special-procedure inventory
remains exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was
already attempted through the cast-sleep outlaw/reagent vehicle and was not
repicked. The interpreter sweep consumed the next genuinely unmanifested
family, `bird`, at `src/interpreter.c:354`; bellow remains claimed by its open
feature handoff after its CI retry.

The pre-slice `main` frontier was 3,424 total cases, with 3,323
proven/delegated, 48 blocked, and 53 excluded. The bird feature branch adds
11 fully proven/delegated cases, but its PR is intentionally not merged: CI did
not fire on the initial PR creation, the one permitted workflow retry was
consumed, and the standing operational rule treats that PR as not-green.
Therefore the checked-in `main` frontier remains 3,424 total / 3,323
proven/delegated / 48 blocked / 53 excluded until the open PR can be handled
under that rule.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:354 */
{ "bird"     , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument` when `char_found` exists, then selects the no-argument,
visible-target, self-target, not-found, or target-position branch. The `bird`
record in `lib/misc/socials:1536-1544` has hide=0, no victim-position minimum,
all eight authored message slots, a literal `him` in its actor target line,
and an absent `others_auto` room line. A sleeping target therefore reaches the
target arm; C's SENDOK delivery suppresses only the sleeping victim line while
actor and awake observer output remain.

## Evidence and verification

The direct three-player vehicle proves no argument, visible target, observer,
one-token parsing, self-target, and not-found behavior. The sleeping-target
vehicle proves that the zero victim-position minimum admits a sleeping target,
with actor and observer delivery and no sleeping-victim bytes. Both vehicles
were run on pre-change `main` first and on the feature branch at seeds 1, 2,
3, 5, and 8; seed 1 for each used `--show-oracle`. Every completed run
reported `result: no normalized divergence`, and the shown C blocks confirmed
the intended branches. One seed-5 direct attempt hit a transient oracle bind
collision and was rerun alone successfully. No Go behavior change was needed:
the generic `do_action` implementation was already byte-faithful.

Durable evidence is on the open feature branch:

- `cmd/dp-oracle-diff/scenarios/bird-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/bird-sleeping-depth.txt`;
- `docs/fidelity/depth/bird.tsv`; and
- `pkg/session/bird_depth_test.go`.

## Integration and gates

The required local gates passed on `glm/depth-bird`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `7653eb181`.

Feature PR: #1193 (`glm/depth-bird`). Initial CI did not fire; the exact
one-time retry `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-bird` was
issued. Per the standing operational rule, the PR remains open and is treated
as not-green; it was not merged.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented self-target output), R5/R5e (the
actual C call path), and R5b/R5c (shared social gate, lookup, and audience
ownership).

## Continuation

The source-order sweep finds `bitch` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:355 */
{ "bitch"    , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`bitch` before advancing in interpreter-table order.
