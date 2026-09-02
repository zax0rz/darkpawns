# Depth-fidelity handoff — `anguish`

Date: 2026-09-02

## Queue position and result

This round started from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the 2026-08-27 brief amendment plus the newest available prior handoff,
`2026-09-02-command-applaud.md` from the open handoff branch. The special-
procedure inventory remains exhausted. The one-time blocked
`objmagic.sleep-entry-gates` row was already attempted through the cast-sleep
outlaw/reagent vehicle and was not repicked. The interpreter sweep consumed
the next source-order family, `anguish`, at `src/interpreter.c:335`.

The pre-slice frontier was 3,375 total cases, with 3,274 proven/delegated, 48
blocked, and 53 excluded. The anguish manifest contributes 8 fully
proven/delegated cases. The resulting frontier is:

- 3,383 total cases
- 3,282 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,282/3,330 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:335 */
{ "anguish"  , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. The loader at
`src/act.social.c:253-263` reads the record at `lib/misc/socials:1287-1290`:
`hide=0`, `min_victim_position=5`, then `char_no_arg` and `others_no_arg`
before `#`. Consequently `char_found` is NULL, so C ignores every argument
and always emits the no-argument actor/room pair; target lookup, not-found,
self-target, victim-position, and the remaining message slots are
unreachable. The command-level POS_RESTING gate and PLR_NOSHOUT boundary are
shared behavior.

## Evidence and verification

The actor/observer vehicle proves no input, a present target, a missing target,
and a self-named target. The vehicle was run on pre-change `main` first and on
the feature branch at seeds 1, 2, 3, 5, and 8; seed 1 on each branch used
`--show-oracle`. Every run reported `result: no normalized divergence`, and
the shown C blocks confirm that all four inputs emit only the authored
no-argument pair.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/anguish-depth.txt`;
- `docs/fidelity/depth/anguish.tsv`; and
- `pkg/session/anguish_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-anguish`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `bffb90607`.

Feature PR: #1181 (`glm/depth-anguish`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `4c5e779d4aa44a766cb361e4cb022490ca4d6d85`.

The separate applaud handoff PR #1180 did not report checks initially; its one
permitted workflow retry fired, but it remains open and was treated as
not-green per the operating rule. This round's handoff is therefore the
durable continuation record on the current main line as well as on its own
branch.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep finds `bounce` as the next genuinely unmanifested
interpreter family. `auction` is not next: it is already covered by the
shared `do_gen_comm` family in `docs/fidelity/depth/channels.tsv`, and `ask`
is likewise covered by existing depth evidence. The next session must return
to clean `main`, pull, rerun `make fidelity-depth`, reread the guide and this
handoff, then map and prove:

```c
/* src/interpreter.c:342 */
{ "bounce"   , POS_STANDING, do_action   , 0, 0 },
```

before advancing in interpreter-table order.
