# Depth-fidelity handoff — `blush`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest available prior handoff,
`2026-09-02-command-blink.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next genuinely unmanifested family,
`blush`, at `src/interpreter.c:361`.

The pre-slice frontier was 3,498 total cases, with 3,397 proven/delegated, 48
blocked, and 53 excluded. The blush manifest contributes 8 fully
proven/delegated cases. The resulting frontier is:

- 3,506 total cases
- 3,405 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,405/3,453 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:361 */
{ "blush"    , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, and only parses an argument when
the social has a `char_found` message. The `blush` record in
`lib/misc/socials:41-45` has hide=0, a zero victim-position minimum, the two
authored no-argument messages, and `#` in the `char_found` slot.
Therefore named, missing, and self-named arguments all follow the no-argument
actor/room pair; there is no target lookup, target-position, self-target, or
not-found branch for this record.

## Evidence and verification

The three-player vehicle proves no argument plus named-target, missing-target,
and self-name inputs, with actor and room audiences. It was run on pre-change
`main` at seeds 1, 2, 3, 5, and 8; seed 1 used `--show-oracle` to confirm
the intended C blocks. Every run reported `result: no normalized divergence`.
No Go behavior change was needed: the generic `DoAction` implementation was
already byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/blush-depth.txt`;
- `docs/fidelity/depth/blush.tsv`; and
- `pkg/session/blush_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup, and
Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-blush`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `f68d5bc98`.

Feature PR: #1205 (`glm/depth-blush`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `3427b86107b354ea680ccdb7d14ff6f022c90256`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep finds `bonk` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:362 */
{ "bonk"     , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`bonk` before advancing in interpreter-table order.

