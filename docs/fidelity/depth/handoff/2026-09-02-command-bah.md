# Depth-fidelity handoff — `bah`

Date: 2026-09-02

## Queue position and result

This round started from clean `main` after the required pull attempt, a
successful `make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`,
and reading the 2026-08-27 brief amendment plus the newest prior handoff,
`2026-09-02-command-anguish.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next genuinely unmanifested family,
`bah`, at `src/interpreter.c:344`; `auction` remained covered by the shared
`channels.tsv` family.

The pre-slice frontier was 3,394 total cases, with 3,293 proven/delegated, 48
blocked, and 53 excluded. The bah manifest contributes 8 fully
proven/delegated cases. The resulting frontier is:

- 3,402 total cases
- 3,301 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,301/3,349 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:344 */
{ "bah"      , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. The loader at
`src/act.social.c:253-263` reads `lib/misc/socials:1396-1400`: `hide=0`, the
actor/room no-argument pair, and `#`. Thus `char_found` is NULL; C ignores
every argument and never reaches target lookup, not-found, self-target, or
victim-position branches. The command-level POS_RESTING gate and PLR_NOSHOUT
boundary are shared behavior.

## Evidence and verification

The actor/observer vehicle proves no input, a present target, a missing target,
and a self-named target. It was run on pre-change `main` first and on the
feature branch at seeds 1, 2, 3, 5, and 8; seed 1 used `--show-oracle`.
Every run reported `result: no normalized divergence`, and the shown C blocks
confirmed that all four inputs emit only the authored actor/room pair. The Go
implementation was already byte-faithful, so no behavior change was made.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/bah-depth.txt`;
- `docs/fidelity/depth/bah.tsv`; and
- `pkg/session/bah_depth_test.go`.

The manifest delegates shared POS_RESTING, PLR_NOSHOUT, visible-room lookup,
and Act behavior to their existing owners under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-bah`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `5dd93d5f6`.

Feature PR: #1185 (`glm/depth-bah`). Hosted lint, security, and test checks
completed green; conditional build-and-push and deploy jobs were skipped. CI
fired normally, so no workflow retry was used. The PR was self-merged only
after all applicable checks were green. GitHub reports the resulting main
merge commit as `031ee83521489898afcf423bfd4cdba6af0558b`; local main is
temporarily parked at its verified direct parent while the GitHub SSH transport
recovers, with no history rewritten or changes dropped.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep finds `beckon` as the next genuinely unmanifested
interpreter family:

```c
/* src/interpreter.c:349 */
{ "beckon"   , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`beckon` before advancing in interpreter-table order.
