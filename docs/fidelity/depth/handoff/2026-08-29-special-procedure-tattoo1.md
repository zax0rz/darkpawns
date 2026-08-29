# 2026-08-29 — `tattoo1` depth slice

## Frontier and queue

- This slice began from synchronized `main` after the required
  `git checkout main`, `git pull --ff-only`, `make fidelity-depth`, and reread
  of `docs/fidelity/DEPTH_TESTING.md` plus the newest dated handoff.
- The pre-slice frontier was 1030 total cases: 993 proven/delegated, 12
  blocked, and 25 excluded. The slice adds seven tattoo1 cases, yielding
  1037 total: 1000 proven/delegated, 12 blocked, and 25 excluded. Actionable
  completion is 1000/1012 = 98.8%.
- `SPECIAL(tattoo1)` is assigned to mob vnum 8086 at `src/spec_assign.c:296`
  and is the next reachable procedure after assassin. The next source-order
  procedure is `tattoo2` at `src/spec_procs2.c:1010`, assigned to mob vnum
  18213 at `src/spec_assign.c:404`.

## C call path and branch census

- The shared C helper `give_tat` is at `src/spec_procs2.c:927-943`; the
  `tattoo1` special is at `src/spec_procs2.c:945-1008`. Player commands enter
  through `src/interpreter.c:1407-1456`, then the registered mob special.
  The live 8086 prototype is the Berzerker tattooist in
  `lib/world/mob/80.mob`; its `tattoo.lua` script is stripped only in the
  disposable oracle/port copies so the registered special is isolated.
- `list` emits the direct instructions, header, and five indexed offers from
  `src/constants.c:1416-1433`: green dragon (30666), tribal (3000), screaming
  eagle (10000), fox (3000), and owl (3000). `buy` first returns FALSE for an
  NPC actor, then handles an already-set tattoo through the tattooist's
  `do_tell`, trims the argument, rejects empty/non-numeric/out-of-range
  choices, tells an underfunded player, or calls `give_tat`.
- `give_tat` deducts the selected price, sets the tattoo, emits two
  `TO_NOTVICT` room acts and one `TO_VICT` act, sends the two-line pain text,
  calls the canonical shout path, sends the blackout line, assigns
  `POS_STUNNED` and immediately calls C `update_pos`, then applies tattoo
  affects and totals. A healthy actor therefore ends standing. The tattoo
  affect matrix is the C `src/tattoo.c:104-186` switch; this slice proves all
  five offered tattoo state effects. Unrelated commands return FALSE.

## RED/GREEN evidence and port result

- RED on `main` with the real vnum 8086 vehicle showed Go falling through to
  `Sorry, but you cannot do that here!` while C emitted the list, buy gates,
  and `give_tat` transcript.
- GREEN vehicles cover the complete list, argument gates, already-tattooed
  tell, insufficient-gold tell, successful actor/room/shout audiences, and
  ordinary command fallthrough. The success vehicle reaches the C block under
  `--show-oracle` and matches the 30666 deduction, dragon tattoo, direct pain
  and blackout, room work/scream acts, and same-zone shout. All four isolated
  vehicles report no normalized divergence with
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` and seed 1.
- Focused proof in `pkg/game/spec_tattoo1_test.go` covers exact list/gate
  bytes, audience ordering, healthy-player position recovery, every offered
  tattoo number, and each offered tattoo's stat/max-move effect.

## Verification

- Green local gates: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, clean `gofumpt -l .`, and clean
  `git diff --check`.
- PR #760 (`glm/spec-tattoo1`) passed lint, security, and test; build-and-push
  and deploy were skipped by workflow policy. It was squash-merged into
  `main` as `49e279019`. No CI retry was required. No `src/` or
  `darkpawns-c-oracle/` file was edited.
- This slice applies R1 (exact bytes), R2 (command surface and FALSE
  fallthrough), R3 (gold, position, and tattoo-effect parity), R4 (no
  invented output), and R5/R5e (actual vnum registration, C call path, and
  shared helper audit).

## Manifest

The durable rows are in `docs/fidelity/depth/spec-procs.tsv`:

- `mob.tattoo1-list`
- `mob.tattoo1-entry-gates`
- `mob.tattoo1-owned-gate`
- `mob.tattoo1-price-gate`
- `mob.tattoo1-success-audience`
- `mob.tattoo1-success-state`
- `mob.tattoo1-fallthrough`

## Next queue item

Continue the special-procedure inventory with `tattoo2` at
`src/spec_procs2.c:1010`, registered to mob vnum 18213 at
`src/spec_assign.c:404`. First map its complete C call path and tattoo offer
table, then build the registered-mob oracle vehicle in source/registration
order.
