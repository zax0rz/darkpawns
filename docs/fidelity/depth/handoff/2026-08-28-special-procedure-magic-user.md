# Depth-fidelity handoff — 2026-08-28 — special procedure `magic_user`

## Frontier and queue position

- Started from `main` at `567070a6e`, pulled before work, and refreshed the
  frontier with `make fidelity-depth`; the merged slice is now `d93da6c71`.
- The special-procedure census remains 113 `SPECIAL` definitions across
  `src/spec_procs.c`, `src/spec_procs2.c`, and `src/spec_procs3.c`; there are
  233 active `ASSIGNMOB` registrations, 228 unique active mob VNUMs, and 66
  final assigned procedure names after later-registration wins.
- Before this slice: 539 total cases, 527 proven/delegated, 1 blocked, and 11
  excluded. After this slice: 544 total, 532 proven/delegated, 1 blocked, and
  11 excluded; actionable completion is 532/533 (99.8%).
- `guild`, `snake`, and `thief` are already claimed by dated handoffs. This
  slice claims `magic_user`; the next unclaimed source-definition item is
  `fighter` at `src/spec_procs.c:509`, assigned to VNUMs 4914, 5200, 7901,
  7902, 12111, 14407, 12850, 20002, 20011, 20019, 20020, 20036, and 20042.

## C path and reachability

R5e was verified from the actual call path before changing Go: autonomous
`src/mobact.c:68-93` skips mobs already fighting, player-command dispatch in
`src/interpreter.c:1407-1456` supplies a nonzero command, and combat
`src/fight.c:1898-2032` invokes `MOB_SPEC` after the NPC's ordinary attack
loop. `src/spec_procs.c:409-500` then performs the `magic_user` target probe,
spell-roll selection, conditional dispel, self invulnerability, and outdoor
spell gates. The native cast path was checked through
`src/spell_parser.c:827-909` (`cast_spell` → `say_spell` → `call_magic`).

## Proof and confirmed divergences

- RED on the pre-fix implementation: the live `spec-proc-magic-user` vehicle
  reached ordinary combat but emitted no NPC spell; C emitted the verbal
  component and spell effect. Source inspection also confirmed Go's wrong
  `number(0,5)` target probe, off-by-one spell selection range, unconditional
  `DispelGood`, wrong teleport gate, victim-targeted invulnerability, and
  missing `OUTSIDE` guards.
- GREEN on `spec-proc-magic-user`, seeds 1, 2, 3, 5, and 8, with no normalized
  divergence. The vehicle uses assigned mob 20014, strips its script, and
  reaches the native special through the combat seam.
- Focused proofs cover entry gates, self-targeted invulnerability, the neutral
  victim dispel gate, and indoor/outdoor meteor-swarm behavior. The tests also
  pin C's NPC verbal-component capitalization and zero-damage skill-message
  path needed by the live vehicle.

## Change and gates

- Go-only changes wire native combat-time `MOB_SPEC` dispatch, align the
  `magic_user` procedure and native cast path, and preserve C spell/combat
  output. Added scenario, focused tests, and five manifest rows. Neither
  `src/` nor `darkpawns-c-oracle/` was edited.
- PR #694 (`glm/spec-magic-user`) was self-merged only after hosted lint,
  security, and test checks all passed; build/deploy jobs were skipped as
  expected for this fidelity-only change.
- Passed `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test
  ./...`, `golangci-lint run ./...` (0 issues), `gofumpt -l .` (empty), and
  `git diff --check`.
- Fidelity rules exercised: R1, R3, R4, R5c, and R5e.

## Next action

Checkout and pull `main`, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this handoff, then map and prove the next
source-ordered active special, `fighter`. The blocked
`objmagic.sleep-entry-gates` row is untouched.
