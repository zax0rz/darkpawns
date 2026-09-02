# Depth-fidelity handoff — `bonk`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest available prior handoff,
`2026-09-02-command-blush.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next genuinely unmanifested family,
`bonk`, at `src/interpreter.c:362`.

The pre-slice frontier was 3,506 total cases, with 3,405 proven/delegated, 48
blocked, and 53 excluded. The bonk manifest contributes 11 fully
proven/delegated cases. The resulting frontier is:

- 3,517 total cases
- 3,416 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,416/3,464 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:362 */
{ "bonk"     , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, parses the first target token
with `one_argument` when `char_found` exists, then selects the no-argument,
visible-target, self-target, not-found, or target-position branch. The `bonk`
record in `lib/misc/socials:46-54` has hide=0, a zero victim-position
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
behavior change was needed: the generic `DoAction` implementation was
already byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/bonk-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/bonk-sleeping-depth.txt`;
- `docs/fidelity/depth/bonk.tsv`; and
- `pkg/session/bonk_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup, and
Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-bonk`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `600c98115`.

Feature PR: #1207 (`glm/depth-bonk`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `54f982b483db560637c9e123e36c9de1af83c099`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep finds `boink` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:363 */
{ "boink"    , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`boink` before advancing in interpreter-table order.

