# Dated Handoff: 2026-08-29 — `take_to_jail` depth slice

## Frontier and queue

- Started from synchronized `main` after the required checkout, fast-forward
  pull, `make fidelity-depth`, and reread of
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-29-special-procedure-identifier.md`.
- The pre-slice frontier was 1071 total cases: 1030 proven/delegated, 12
  blocked, and 29 excluded. The `take_to_jail` slice adds five proven cases,
  one delegated case, one blocked case, and one excluded case, yielding 1079
  total: 1036 proven/delegated, 13 blocked, and 30 excluded. Actionable
  completion is 1036/1049 = 98.8%.
- `SPECIAL(take_to_jail)` is assigned to mob vnums 8027, 8059, 8001, 8002,
  and 8020 at `src/spec_assign.c:285-291`; its body is at
  `src/spec_procs2.c:1427-1468`. The scriptless 8001 vehicle was the first
  registration that reached the autonomous path reliably.
- The next actual registered source-order procedure is `jail` at
  `src/spec_procs2.c:1470`, assigned to room 8118 in
  `src/spec_assign.c`.

## C call path and branch census

- `src/mobact.c:68-93` filters to awake, non-fighting MOB_SPEC mobiles and
  invokes the special with an empty command. `take_to_jail` first rejects
  commands and sleeping mobs, then delegates an already-fighting guard to
  `fighter`.
- The room scan at `src/spec_procs2.c:1432-1442` selects the first visible
  outlaw, emits the exact no-comma warning, calls canonical `hit()`, and
  returns the native fighter result. The same loop calls shared
  `breed_killer`; that branch remains with the existing blocked
  `mob.cityguard-breed-killer` owner and is not duplicated here (R5b/R5c).
- The protection scan at `src/spec_procs2.c:1443-1466` selects the lowest
  alignment visible combatant with a nonnegative-alignment fight target and
  valid NPC topology, emits the native warning, calls `hit()`, and delegates
  the shared cityguard matrix.
- `src/fight.c:1370-1400` owns the jail redirect: reciprocal combat cleanup,
  HP=1, memory/hunting cleanup, victim/room audience bytes, unmount,
  relocation to room 8118, destination look, and
  `MAX(2, GET_LEVEL(victim)/2)` jail timer.
- The direct fighting branch is recorded as D5 excluded: mobile_activity
  filters fighting mobs before special dispatch, while player-command
  dispatch supplies nonzero `cmd` and fails the first gate. This is a C path
  exclusion, not a synthetic vehicle (R2/R4/R5e).

## RED/GREEN evidence and port result

- Clean `main` RED on the registered 8001 vehicle showed the old Go
  placeholder silently skipping the C outlaw intervention; state remained at
  the pre-jail room/HP instead of entering the C redirect.
- The corrected vehicle is
  `cmd/dp-oracle-diff/scenarios/spec-proc-take-to-jail.txt`. Its GREEN
  transcript matches C at seeds 1, 2, 3, 5, and 8, including the warning
  punctuation, victim/room audience split, exact room-look ordering, HP=1,
  room 8118, and score output.
- Focused coverage is in `pkg/game/spec_take_to_jail_test.go`. It pins entry
  gates, outlaw warning/hit boundary, protection/fallthrough contract, and
  jail redirect state including reciprocal combat and hunting cleanup.
- The byte correction in `pkg/game/combat_wire.go` uses canonical `Act`
  templates and passes the direct jail sentence without a duplicated CRLF;
  the previous renderer-wide room behavior was left unchanged (R1/R5e).

## Verification and integration

- Local gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #765 (`glm/spec-take-to-jail`) passed lint, security, and test checks;
  build-and-push/deploy were skipped by workflow policy. It was squash-merged
  to `main` as `683137fad`.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Manifest

The durable rows added by this slice are:

- `mob.take-to-jail-autonomous-entry`
- `mob.take-to-jail-outlaw-warning`
- `mob.take-to-jail-subdue-audience`
- `mob.take-to-jail-subdue-state`
- `mob.take-to-jail-return-contract`
- `mob.take-to-jail-fighting-branch` (D5 excluded)
- `mob.take-to-jail-breed-killer` (D5 blocked; shared owner)
- `mob.take-to-jail-protection-intervention` (D4 delegated to
  `mob.cityguard-protection-intervention`)

This handoff applies R1 (exact player-facing bytes), R2 (registered command
surface), R3 (combat/jail state and draw-preserving boundaries), R4 (no
invented unregistered or shared-owner behavior), and R5/R5b/R5c/R5e (actual C
call path, shared ownership, and call-path verification).
