# Depth-fidelity handoff — `agree`

Date: 2026-09-02

## Queue position and result

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md`, the
2026-08-27 brief amendment, and the newest handoff,
`2026-09-02-command-accuse.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next source-order family, `agree`, at
`src/interpreter.c:331`.

The pre-slice frontier was 3,342 total cases, with 3,241 proven/delegated, 48
blocked, and 53 excluded. The agree manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,353 total cases
- 3,252 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,252/3,300 = 98.5%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:331 */
{ "agree"    , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record and PLR_NOSHOUT gate, parses the first target token with
`one_argument` when `char_found` exists, and then selects no-argument,
visible-target, self-target, not-found, or target-position branches. The
`agree` record in `lib/misc/socials:11-19` has hide=0, no victim-position
minimum, the actor/room no-argument pair, actor/room/victim target templates,
the exact `Really now?` not-found line, and the self-target pair. Because
the victim minimum is zero, a sleeping target reaches the target arm, but C's
`TO_VICT` delivery is suppressed by `SENDOK` while actor and awake observer
outputs remain.

## Evidence and verification

The direct three-player vehicle proves the no-argument, target, observer,
one-token, self-target, and not-found branches. The sleeping-target vehicle
proves actor and observer delivery plus the silent sleeping victim. Both
vehicles are green at seeds 1, 2, 3, 5, and 8; seed 1 for each was run with
`--show-oracle`. The Go implementation was already byte-faithful for this
generic `do_action` record, so no behavior change was made.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/agree-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/agree-sleeping-depth.txt`;
- `docs/fidelity/depth/agree.tsv`; and
- `pkg/session/agree_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c. The focused test
pins the C entry gate and all eight authored social messages/metadata.

## Integration and gates

The required local gates passed on `glm/depth-agree`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `71a948237`.

Feature PR: #1175 (`glm/depth-agree`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `04062025a`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate/lookup ownership and whole-class
review).

## Continuation

The source-order audit now finds `apologize` as the next unclaimed
interpreter token:

```c
/* src/interpreter.c:333 */
{ "apologize", POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`apologize` before advancing in interpreter-table order.
