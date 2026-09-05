# Depth-fidelity handoff — `beg`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest available prior handoff,
`2026-09-02-command-bounce.md`; the immediately preceding beckon handoff was
also read from its unmerged handoff branch. The special-procedure inventory
remains exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was
already attempted through the cast-sleep outlaw/reagent vehicle and was not
repicked. The interpreter sweep consumed the next genuinely unmanifested
family, `beg`, at `src/interpreter.c:351`; `ask` and `auction` remain covered
by their existing shared communication manifests.

The pre-slice frontier was 3,413 total cases, with 3,312 proven/delegated, 48
blocked, and 53 excluded. The beg manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,424 total cases
- 3,323 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,323/3,371 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:351 */
{ "beg"      , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument` when `char_found` exists, then selects the no-argument,
visible-target, self-target, not-found, or target-position branch. The `beg`
record in `lib/misc/socials:31-39` has hide=0, no victim-position minimum,
authored target and self branches, and an absent `others_auto` room line.
Consequently a sleeping target reaches the target arm; C's SENDOK delivery
suppresses only the sleeping victim line while actor and awake observer output
remain.

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

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/beg-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/beg-sleeping-depth.txt`;
- `docs/fidelity/depth/beg.tsv`; and
- `pkg/session/beg_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-beg`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `cf7737815`.

Feature PR: #1189 (`glm/depth-beg`). Hosted lint, security, and test checks
completed green; conditional build-and-push and deploy jobs were skipped. CI
fired normally, so no workflow retry was used. The PR was self-merged only
after all applicable checks were green. The resulting `main` merge commit is
`511859d3fd68042c85991e1ebf7118b22b6fbaf5`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented self-target output), R5/R5e (the
actual C call path), and R5b/R5c (shared social gate, lookup, and audience
ownership).

## Continuation

The source-order sweep finds `bellow` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:352 */
{ "bellow"   , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`bellow` before advancing in interpreter-table order.
