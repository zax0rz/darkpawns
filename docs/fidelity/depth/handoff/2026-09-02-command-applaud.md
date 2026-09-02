# Depth-fidelity handoff — `applaud`

Date: 2026-09-02

## Queue position and result

This round started from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest prior handoff,
`2026-09-02-command-apologize.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next source-order family, `applaud`, at
`src/interpreter.c:334`.

The pre-slice frontier was 3,364 total cases, with 3,263 proven/delegated, 48
blocked, and 53 excluded. The applaud manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,375 total cases
- 3,274 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,274/3,322 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:334 */
{ "applaud"  , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT and actor-position gates, parses the
first target token with `one_argument` when `char_found` exists, then selects
the no-argument, visible-target, self-target, not-found, or target-position
branch. The `applaud` record in `lib/misc/socials:21-29` has hide=0, no
victim-position minimum, and all eight authored messages. Because the victim
minimum is zero, a sleeping target reaches the target arm, while C's `SENDOK`
delivery suppresses the `TO_VICT` line for the sleeping victim while actor and
awake observer output remains.

## Evidence and verification

The direct three-player vehicle proves no argument, visible target, observer,
one-token parsing, self-target, and not-found behavior. The sleeping-target
vehicle proves actor and observer delivery plus the silent sleeping victim.
Both vehicles are green at seeds 1, 2, 3, 5, and 8; seed 1 for each was run
with `--show-oracle` to verify the intended C block. The vehicles were run on
pre-change `main` first and then on the feature branch. Every run reported
`result: no normalized divergence`. The Go implementation was already
byte-faithful, so no behavior change was made.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/applaud-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/applaud-sleeping-depth.txt`;
- `docs/fidelity/depth/applaud.tsv`; and
- `pkg/session/applaud_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-applaud`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `042c82615`.

Feature PR: #1179 (`glm/depth-applaud`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `76caa21dd73c08c0920b175cd2280be7a61b6aa0`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order audit now finds `anguish` as the next unclaimed interpreter
token:

```c
/* src/interpreter.c:335 */
{ "anguish"  , POS_RESTING , do_action   , 0, 0 },
```

Its social record is at `lib/misc/socials:1287-1290`; map that record's actual
field population before choosing the proof matrix. The next session must
return to clean `main`, pull, rerun `make fidelity-depth`, reread the guide and
this handoff, then map and prove `anguish` before advancing in
interpreter-table order.
