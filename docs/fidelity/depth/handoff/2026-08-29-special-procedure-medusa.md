# Dated Handoff: 2026-08-29 — `medusa` depth slice

## Frontier and queue

- Started from synchronized `main` after the required checkout, fast-forward
  pull, `make fidelity-depth`, and reread of
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-29-special-procedure-jail.md`.
- The pre-slice frontier was 1081 total cases: 1037 proven/delegated, 13
  blocked, and 31 excluded. The `medusa` slice adds six proven/delegated
  cases, yielding 1087 total: 1043 proven/delegated, 13 blocked, and 31
  excluded. Actionable completion is 1043/1056 = 98.8%.
- The next actual registered procedure in source-and-registration order is
  `eq_thief`, implemented at `src/spec_procs2.c:1613-1646` and assigned to
  mob vnums 7979, 12118, and 14225 at `src/spec_assign.c:266,350,366`.

## C call path and branch census

- `SPECIAL(medusa)` is implemented at `src/spec_procs2.c:1552-1590` and
  registered for mobs 14101 and 14102 at `src/spec_assign.c:355-356`.
- `src/interpreter.c:1407-1456` supplies player commands to the mob special;
  the procedure accepts only `look`/`examine` whose argument matches
  `mobile->player.name` through C `isname()`.
- The command path evaluates only the actor with `mag_savingthrow(ch,
  SAVING_PETRI)`. A save returns FALSE for ordinary look/examine. A failed save
  sends the stone act to `TO_NOTVICT`, the horror act to `TO_CHAR`, increments
  `GET_DEATHS(ch)`, applies `gain_exp(ch, -(level*level*level))`, then calls
  `raw_kill(ch, SPELL_PETRIFY)`.
- The commandless fighting arm calls the shared `magic_user(mobile, mobile,
  0, NULL)` path. That branch is delegated to the existing
  `mob.magic-user-combat-cast` owner; no duplicate combat matrix was added
  (R5b/R5c).

## RED/GREEN evidence and port result

- Clean `main` RED was established with the registered 14101 vehicle in room
  14112. The old Go handler invented room-wide gaze text and ordinary attacks
  against every player instead of running the actor-only C save/petrify path.
- The corrected vehicle is
  `cmd/dp-oracle-diff/scenarios/spec-proc-medusa.txt`. Failed-save seeds 1, 2,
  and 8 are green, including the actor/witness audience split and death cry.
  Seeds 3 and 5 save in C and expose the existing shared ordinary
  look-at-character rendering difference; the medusa handler now correctly
  returns FALSE and that command-owner gap is represented by the unit-green
  save-fallthrough row rather than duplicated here (R5b/R5c).
- Focused coverage is in `pkg/game/spec_medusa_test.go`; it pins all entry
  gates, the shared fighting delegation boundary, save fall-through, exact
  actor/witness output, death count, level-cubed XP loss, dead position, and
  the raw-kill extraction seam.

## Verification and integration

- Local gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #767 (`glm/spec-medusa`) passed test, lint, and security checks; the
  build-and-push job was skipped by workflow policy. It was squash-merged to
  `main` as `3437357816042f50cf3ed23525fbf0d0755ee093`.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Manifest

The durable rows added by this slice are:

- `mob.medusa-entry-gates` (unit-green)
- `mob.medusa-look-target-gate` (oracle-green)
- `mob.medusa-save-fallthrough` (unit-green)
- `mob.medusa-petrify-audience` (oracle-green-multiseed)
- `mob.medusa-petrify-state` (unit-green)
- `mob.medusa-magic-user-delegation` (delegated)

This handoff applies R1 (exact petrification bytes), R2 (registered medusa
command surface), R3 (save draw and failed-arm parity), R5/R5e (actual
dispatcher and call-path verification), and R5b/R5c (shared look and
magic-user ownership).
