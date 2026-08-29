# 2026-08-29 — `tattoo3` depth slice

## Frontier and queue

- This slice began from synchronized `main` after the required `git checkout
  main`, `git pull --ff-only`, `make fidelity-depth`, and reread of
  `docs/fidelity/DEPTH_TESTING.md` plus the newest dated handoff.
- The pre-slice frontier was 1037 total cases: 1000 proven/delegated, 12
  blocked, and 25 excluded. The slice adds seven tattoo3 proofs and one
  source-order unreachable exclusion, yielding 1052 total: 1014
  proven/delegated, 12 blocked, and 26 excluded. Actionable completion is
  1014/1026 = 98.8%.
- `SPECIAL(tattoo3)` is assigned to mob vnum 21244 at `src/spec_assign.c:507`.
  The following source definition, `eviltrade`, has no C dispatch
  registration and is durably excluded. The next reachable source-order
  procedure is `identifier` at `src/spec_procs2.c:1193`, assigned to mob vnum
  8087 at `src/spec_assign.c:297`.

## C call path and branch census

- `SPECIAL(tattoo3)` is at `src/spec_procs2.c:1075-1137`; player commands
  enter through `src/interpreter.c:1407-1456` and dispatch to the registered
  mob special. The live vnum 21244 prototype is Polywig in
  `lib/world/mob/212.mob`, reset into room 21281 by
  `lib/world/zon/212.zon`; its `tattoo.lua` script is stripped only in the
  disposable oracle/port copies so the registered C special is isolated.
- `list` emits the direct instructions, header, and four indexed offers from
  `src/constants.c:174-193,1416-1433`: open eye (18000), crossed swords
  (20000), ship (11000), and the word MOM (15000), with their source names
  and descriptions.
- `buy` trims leading spaces before the NPC early return, then handles bare
  and nonnumeric arguments. Out-of-range choices are rejected, an existing
  tattoo receives Polywig's exact garbled `do_tell`, an underfunded player
  receives the exact “cash, hot stuff” tell, and a funded player enters the
  shared `give_tat` helper. Unrelated commands return FALSE.
- The shared `give_tat` path is at `src/spec_procs2.c:927-943`: deduct the
  selected price, set the tattoo, emit two `TO_NOTVICT` room acts and one
  `TO_VICT` act, send the two-line pain text, use the canonical shout path,
  send the blackout line, assign `POS_STUNNED` followed by C `update_pos`, and
  apply tattoo affects. The offered tattoo effects are pinned against
  `src/tattoo.c:104-186`: eye and ship have no direct modifier, crossed swords
  adds hitroll and damroll, and MOM adds wisdom.
- `SPECIAL(eviltrade)` is at `src/spec_procs2.c:1139-1191`, but the actual
  `src/spec_assign.c` tables contain no registration for it. Under the C
  `special()` and `mobile_activity` call paths it is unreachable, so no
  synthetic player-facing vehicle is valid (R2/R4/R5e).

## RED/GREEN evidence and port result

- RED on `main` with the real vnum 21244 vehicle showed Go falling through to
  `Sorry, but you cannot do that here!` while C emitted Polywig's list, buy
  gates, and price tell.
- GREEN vehicles cover list and argument gates, the already-tattooed and
  insufficient-cash tells, the successful actor/room/shout audiences, every
  offered tattoo number and effect, and ordinary command fallthrough. Four
  isolated vehicles (`spec-proc-tattoo3`, `-owned`, `-price`, and `-success`)
  all report no normalized divergence with
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` and seed 1. The
  success vehicle was also inspected with `--show-oracle` and reached the
  shared give_tat block with the 18000-gold deduction and open-eye state.
- Focused proof in `pkg/game/spec_tattoo3_test.go` covers exact list/gate
  bytes, Polywig's direct tell output, audience ordering, healthy-player
  position recovery, every offered tattoo number, and each offered tattoo's
  stat/combat effect.

## Verification

- Green local gates: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, clean `gofumpt -l .`, and clean
  `git diff --check`.
- PR #762 (`glm/spec-tattoo3`) passed lint, security, and test; build-and-push
  and deploy were skipped by workflow policy. It was squash-merged into
  `main` as `6d7a3192d`. No CI retry was required. No `src/` or
  `darkpawns-c-oracle/` file was edited.
- This slice applies R1 (exact bytes), R2 (command surface, reachability, and
  FALSE fallthrough), R3 (gold, position, and tattoo-effect parity), R4 (no
  invented output), and R5/R5e (actual vnum registration, C call path, and
  shared helper audit).

## Manifest

The durable rows are in `docs/fidelity/depth/spec-procs.tsv`:

- `mob.tattoo3-list`
- `mob.tattoo3-entry-gates`
- `mob.tattoo3-owned-gate`
- `mob.tattoo3-price-gate`
- `mob.tattoo3-success-audience`
- `mob.tattoo3-success-state`
- `mob.tattoo3-fallthrough`
- `mob.eviltrade-unassigned`

## Next queue item

Continue the special-procedure inventory with `identifier` at
`src/spec_procs2.c:1193`, registered to mob vnum 8087 at
`src/spec_assign.c:297`. First map its complete C call path and actual object
lookup/identify branches, then build the registered-mob oracle vehicle in
source order.
