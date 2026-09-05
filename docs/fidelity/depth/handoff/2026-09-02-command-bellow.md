# Depth-fidelity handoff — `bellow`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest available prior handoff,
`2026-09-02-command-beg.md` from its handoff branch. The special-procedure
inventory remains exhausted. The one-time blocked `objmagic.sleep-entry-gates`
row was already attempted through the cast-sleep outlaw/reagent vehicle and
was not repicked. The interpreter sweep consumed the next genuinely
unmanifested family, `bellow`, at `src/interpreter.c:352`.

The pre-slice `main` frontier was 3,424 total cases, with 3,323
proven/delegated, 48 blocked, and 53 excluded. The bellow feature branch adds
11 fully proven/delegated cases, but its PR is intentionally not merged: CI did
not fire on the initial PR creation, the one permitted workflow retry was
consumed, and the standing operational rule treats that PR as not-green.
Therefore the checked-in `main` frontier remains 3,424 total / 3,323
proven/delegated / 48 blocked / 53 excluded until the open PR can be handled
under that rule.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:352 */
{ "bellow"   , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument` when `char_found` exists, then selects the no-argument,
visible-target, self-target, not-found, or target-position branch. The
`bellow` record in `lib/misc/socials:1063-1072` has hide=1, no victim-position
minimum, all eight authored message slots, and a populated self-target room
line. A sleeping target therefore reaches the target arm; C's SENDOK delivery
suppresses only the sleeping victim line while actor and awake observer output
remain. The hide-invisible audience behavior is delegated to the shared
socials-depth proof under R5b/R5c.

## Evidence and verification

The direct three-player vehicle proves no argument, visible target, observer,
one-token parsing, self-target, and not-found behavior. The sleeping-target
vehicle proves that the zero victim-position minimum admits a sleeping target,
with actor and observer delivery and no sleeping-victim bytes. Both vehicles
were run on pre-change `main` first and on the feature branch at seeds 1, 2,
3, 5, and 8; seed 1 for each used `--show-oracle`. Every completed run
reported `result: no normalized divergence`, and the shown C blocks confirmed
the intended branches. No Go behavior change was needed: the generic
`do_action` implementation was already byte-faithful.

Durable evidence is on the open feature branch:

- `cmd/dp-oracle-diff/scenarios/bellow-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/bellow-sleeping-depth.txt`;
- `docs/fidelity/depth/bellow.tsv`; and
- `pkg/session/bellow_depth_test.go`.

## Integration and gates

The required local gates passed on `glm/depth-bellow`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `537aacd36`.

Feature PR: #1191 (`glm/depth-bellow`). Initial CI did not fire; the exact
one-time retry `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-bellow` was
issued. Per the standing operational rule, the PR remains open and is treated
as not-green; it was not merged.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, hide-invisible audience, and
Act ownership).

## Continuation

The source-order sweep finds `bird` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:354 */
{ "bird"     , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`bird` before advancing in interpreter-table order.
