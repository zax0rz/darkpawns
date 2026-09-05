# 2026-08-29 — `identifier` depth slice

## Frontier and queue

- Started from synchronized `main` after the required checkout, fast-forward
  pull, `make fidelity-depth`, and reread of `docs/fidelity/DEPTH_TESTING.md`
  plus the newest handoff. The pre-slice frontier was 1052 total cases: 1014
  proven/delegated, 12 blocked, and 26 excluded.
- This slice adds nine proven cases, yielding 1061 total: 1023
  proven/delegated, 12 blocked, and 26 excluded. Actionable completion is
  1023/1035 = 98.8%.
- `SPECIAL(identifier)` is assigned to mob vnum 8087 at
  `src/spec_assign.c:297`. The next active source-order procedure is
  `tattoo4` at `src/spec_procs2.c:1282`, assigned to mob vnum 2766 at
  `src/spec_assign.c:213`.

## C call path and branch census

- `SPECIAL(identifier)` is defined at `src/spec_procs2.c:1193-1280`.
  Player commands enter through `src/interpreter.c:1407-1456`; the registered
  vnum-8087 mob is the actual dispatch vehicle. Its prototype is Ferrenx in
  `lib/world/mob/80.mob`, reset into room 8116; the disposable scenario strips
  `identifier.lua` and removes room exits so only the C special is exercised.
- `list` sends the actor the direct tell `Just read the sign!` through the
  identifier mob and returns TRUE.
- `value` skips leading spaces, handles the empty argument with `Value what?`,
  searches only the actor's carried visible objects, handles a miss with
  `You don't seem to have that.`, and otherwise quotes `val_cost(obj)`.
  `val_cost` is `src/spec_procs2.c:1178-1191`: cost/10 below 5000, cost*.14
  at or above 5000, plus cost/20 for ITEM_MAGIC, with a minimum of one.
- `give` returns FALSE for a bare/incomplete command, a nonmatching recipient,
  or a missing carried visible object, leaving normal `do_give` to produce its
  own response. A matching recipient and object uses the same price gate and
  emits the two exact shortage tells when underfunded.
- A funded give deducts `val_cost`, emits the C actor/victim/room act sequence,
  returns the object in the narrative without moving it, sends the blank line,
  and calls `spell_identify`. The C formatter in `src/spells.c:476-621` emits
  the short description, uppercase item type, C extra/affect bit lines,
  encumbrance/value, and type-specific details.

## RED/GREEN evidence and port result

- RED on `main` with the registered vnum-8087 and carried bread showed Go
  falling through for `list`/`value`, targeting the identifier incorrectly for
  incomplete `give bread`, and using the unrelated invented `identify`
  command/fee/output. The RED also exposed an empty-name bug in the shared Go
  `FindMobInRoom`, where an empty recipient matched the first mob.
- GREEN vehicles are:
  `cmd/dp-oracle-diff/scenarios/spec-proc-identifier.txt` for list, value,
  give entry/fallthrough, and ordinary command fallthrough;
  `spec-proc-identifier-price.txt` for the shortage branch; and
  `spec-proc-identifier-success.txt` for the funded audience and identify
  output. All three report no normalized divergence with
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`; the success vehicle
  was inspected with `--show-oracle`.
- Focused tests in `pkg/game/spec_identifier_test.go` cover exact entry/value
  tells, FALSE fallthrough, price/state preservation, successful audience and
  inventory retention, and the `val_cost` threshold/magic surcharge.

## Verification and integration

- Green local gates after the final correction: `make fidelity-depth`,
  `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, `gofumpt -l .`, and gosec 2.29.0 high-severity
  scan.
- PR #763 (`glm/spec-identifier`) passed the initial lint/test jobs after one
  authorized CI workflow retry; the initial security job caught the new G115
  conversion and was corrected with an explicit range guard. The retry's
  lint, security, and test checks were all green; the PR was squash-merged to
  `main` as `bdfd54844`. Build-and-push and deploy were skipped by workflow
  policy.
- No `src/` or `darkpawns-c-oracle/` file was edited. The empty-name guard is
  a confirmed shared call-path fix required to preserve C's normal give
  fallthrough, not a change to the oracle.

This slice applies R1 (exact player-facing bytes), R2 (registered special and
FALSE command fallthrough), R3 (gold and object state), R4 (no invented
identifier behavior), and R5/R5e (actual registration, C call path, and
shared `spell_identify` audit).

## Manifest

The durable rows are in `docs/fidelity/depth/spec-procs.tsv`:

- `mob.identifier-list`
- `mob.identifier-value-entry-gates`
- `mob.identifier-value-success-tell`
- `mob.identifier-give-entry-gates`
- `mob.identifier-give-fallthrough`
- `mob.identifier-price-gate`
- `mob.identifier-success-audience`
- `mob.identifier-success-state`
- `mob.identifier-fallthrough`

## Next queue item

Continue the special-procedure inventory with `tattoo4` in source and
registration order. Do not repick `identifier`, `eviltrade`, `couch`,
`whirlpool`, or earlier claimed procedures.
