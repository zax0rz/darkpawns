# 2026-08-29 — `tattoo2` depth slice

## Frontier and queue

- This slice began from synchronized `main` after the required `git checkout
  main`, `git pull --ff-only`, `make fidelity-depth`, and reread of
  `docs/fidelity/DEPTH_TESTING.md` plus the newest dated handoff.
- The pre-slice frontier was 1037 total cases: 1000 proven/delegated, 12
  blocked, and 25 excluded. The slice adds seven tattoo2 cases, yielding
  1044 total: 1007 proven/delegated, 12 blocked, and 25 excluded. Actionable
  completion is 1007/1019 = 98.8%.
- `SPECIAL(tattoo2)` is assigned to mob vnum 18213 at `src/spec_assign.c:404`.
  The next source-and-registration-order procedure is `tattoo3` at
  `src/spec_procs2.c:1075`, assigned to mob vnum 21244 at
  `src/spec_assign.c:507`.

## C call path and branch census

- `SPECIAL(tattoo2)` is at `src/spec_procs2.c:1010-1072`; player commands
  enter through `src/interpreter.c:1407-1456` and dispatch to the registered
  mob special. The live vnum 18213 prototype is Confucius in
  `lib/world/mob/182.mob`, reset into room 18252 by
  `lib/world/zon/182.zon`; its `tattoo.lua` script is stripped only in the
  disposable oracle/port copies so the registered C special is isolated.
- `list` emits the direct instructions, header, and four indexed offers from
  `src/constants.c:1416-1433`: tiger (14000), heart (17000), star (17000),
  and Jyhad (19000), with their source names and descriptions.
- `buy` trims leading spaces before the NPC early return, then handles bare,
  nonnumeric, and out-of-range arguments. A player with an existing tattoo
  receives Confucius's exact `do_tell`; an underfunded player receives the
  exact wisdom tell; a funded player enters the shared `give_tat` helper.
  Unrelated commands return FALSE.
- The shared `give_tat` path is at `src/spec_procs2.c:927-943`: deduct the
  selected price, set the tattoo, emit two `TO_NOTVICT` room acts and one
  `TO_VICT` act, send the two-line pain text, use the canonical shout path,
  send the blackout line, assign `POS_STUNNED` followed by C `update_pos`, and
  apply tattoo affects. The offered tattoo effects are pinned against
  `src/tattoo.c:104-186`: tiger dexterity/movement, heart hit points, star
  mana, and Jyhad damroll.

## RED/GREEN evidence and port result

- RED on `main` with the real vnum 18213 vehicle showed Go falling through to
  `Sorry, but you cannot do that here!` while C emitted Confucius's list,
  buy gates, and price tell.
- GREEN vehicles cover list and argument gates, already-tattooed and
  insufficient-gold tells, the successful actor/room/shout audiences, every
  offered tattoo effect, and ordinary command fallthrough. Four isolated
  vehicles (`spec-proc-tattoo2`, `-owned`, `-price`, and `-success`) all report
  no normalized divergence with
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` and seed 1. The
  success vehicle was also inspected with `--show-oracle` and reached the
  shared give_tat block with the 14000-gold deduction and tiger effects.
- Focused proof in `pkg/game/spec_tattoo2_test.go` covers exact list/gate
  bytes, direct tell output, audience ordering, healthy-player position
  recovery, every offered tattoo number, and each offered tattoo's
  stat/resource effect.

## Verification

- Green local gates: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, clean `gofumpt -l .`, and clean
  `git diff --check`.
- PR #761 (`glm/spec-tattoo2`) passed lint, security, and test; build-and-push
  and deploy were skipped by workflow policy. It was squash-merged into
  `main` as `e75e39ffc`. No CI retry was required. No `src/` or
  `darkpawns-c-oracle/` file was edited.
- This slice applies R1 (exact bytes), R2 (command surface and FALSE
  fallthrough), R3 (gold, position, and tattoo-effect parity), R4 (no
  invented output), and R5/R5e (actual vnum registration, C call path, and
  shared helper audit).

## Manifest

The durable rows are in `docs/fidelity/depth/spec-procs.tsv`:

- `mob.tattoo2-list`
- `mob.tattoo2-entry-gates`
- `mob.tattoo2-owned-gate`
- `mob.tattoo2-price-gate`
- `mob.tattoo2-success-audience`
- `mob.tattoo2-success-state`
- `mob.tattoo2-fallthrough`

## Next queue item

Continue the special-procedure inventory with `tattoo3` at
`src/spec_procs2.c:1075`, registered to mob vnum 21244 at
`src/spec_assign.c:507`. First map its complete C call path and tattoo offer
table, then build the registered-mob oracle vehicle in source/registration
order.
