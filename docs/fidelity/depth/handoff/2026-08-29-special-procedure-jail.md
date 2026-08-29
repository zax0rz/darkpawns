# Dated Handoff: 2026-08-29 — `jail` depth slice

## Frontier and queue

- Started from synchronized `main` after the required checkout, fast-forward
  pull, `make fidelity-depth`, and reread of
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-29-special-procedure-take-to-jail.md`.
- The pre-slice frontier was 1079 total cases: 1036 proven/delegated, 13
  blocked, and 30 excluded. The `jail` slice adds one proven case and one
  excluded case, yielding 1081 total: 1037 proven/delegated, 13 blocked, and
  31 excluded. Actionable completion is 1037/1050 = 98.8%.
- `SPECIAL(jail)` is assigned to room 8118 at `src/spec_assign.c:585,609`
  and defined at `src/spec_procs2.c:1470-1493`. The next actual registered
  source-order procedure is `medusa` at `src/spec_procs2.c:1552`, assigned to
  mob vnums 14101 and 14102.

## C call path and branch census

- `src/interpreter.c:1407-1416` invokes a room special during player command
  dispatch; `src/act.movement.c:115` invokes the same special for movement
  dispatch. Both pass a nonzero command index for every reachable player
  command. The first C gate, `if (cmd || mini_mud)`, therefore returns FALSE.
- The latent body at `src/spec_procs2.c:1476-1493` would handle a
  commandless, mortal, non-hunting player with zero jail timer, emitting the
  time-up/throw-out acts, moving to room 8117, and looking. The repository has
  no room-heartbeat call to `special()`; the only C call sites are the command
  and movement paths above. This body is consequently D5 excluded from the
  player-facing registered call surface (R2/R4/R5e).
- The Go placeholder previously invented a `say release` command, deducted
  gold and movement, and attempted to use a nil room-special mob. The faithful
  command path now falls through to the ordinary command handler.

## RED/GREEN evidence and port result

- Clean `main` RED was established by the registered room-8118 vehicle: C
  fell through `say release` to `You say 'release'`, while the old Go
  `specJail` intercepted it as a release operation and could dereference the
  nil room-special mob.
- The corrected vehicle is
  `cmd/dp-oracle-diff/scenarios/spec-proc-jail.txt`. It reports no normalized
  divergence at seed 1 and proves the live command gate and ordinary say
  fallthrough.
- Focused coverage is in `pkg/game/spec_jail_test.go`; it pins no handling,
  no output, no relocation, and no gold/movement mutation for `say release`.

## Verification and integration

- Local gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #766 (`glm/spec-jail`) passed lint, security, and test checks;
  build-and-push/deploy were skipped by workflow policy. It was squash-merged
  to `main` as `30fa78c0b`.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Manifest

The durable rows added by this slice are:

- `room.jail-command-gate` (oracle-green)
- `room.jail-commandless-body` (D5 excluded; no reachable C dispatcher)

This handoff applies R1 (exact ordinary-command bytes), R2 (registered room
special command surface), R4 (no invented release behavior), and R5/R5e
(actual dispatcher call-path verification).
