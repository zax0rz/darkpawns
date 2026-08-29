# 2026-08-29 — `carrion` depth slice

## Frontier and queue

- Started from `main`, pulled `0c1f07914`, ran `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff before
  selecting the next unclaimed registered special.
- The pre-slice frontier was 1,095 total cases: 1,050 proven/delegated, 13
  blocked, and 32 excluded. This slice adds nine proven/delegated rows, for
  1,104 total: 1,059 proven/delegated, 13 blocked, and 32 excluded;
  actionable completion is 1,059/1,072 (98.8%).
- The next source-order definitions after `carrion` are `bat_room` and `bat`,
  but neither is registered in the actual assignment tables. `no_move_east`
  and `key_seller` are already claimed. The next active, unclaimed special is
  `mindflayer`, defined at `src/spec_procs2.c:1972` and registered for mob
  vnum 14414 at `src/spec_assign.c:383`.

## C call path and branch census

- `SPECIAL(carrion)` is defined at `src/spec_procs2.c:1726-1754` and assigned
  as a room special by `ASSIGNROOM(14305, carrion)` at
  `src/spec_assign.c:612`.
- Room command dispatch is `src/interpreter.c:1407-1416`: the room special is
  called first with `(ch, world + room, cmd, argument)` and a TRUE return
  stops ordinary command handling. There is no room-special pulse path.
- The entry gate returns FALSE for a sleeping positive-HP caller or mini-mud,
  then requires a nonzero command and a nonblank argument containing the
  case-sensitive substring `corpse`, `corpses`, or `pile`. A nonmatching
  command falls through to the ordinary command handler.
- On the matching branch, C calls `read_mobile(14308, VIRTUAL)`, sets the
  spawned stalker's level and damroll to the caller's level, moves it into the
  caller's room, sends the exact room Act
  `Suddenly $n skitters from out of a corpse!`, calls `hit(i, ch,
  TYPE_UNDEFINED)`, and returns TRUE. The stalker's hit is shared combat
  behavior, not a second carrion-specific matrix.
- The Go implementation now preserves the caller-room spawn and C room Act,
  carries both level and direct damroll overrides on the mob instance without
  mutating the shared prototype, and routes the opener through the canonical
  mobile-special combat seam. `StartCombatFromMob` defers defender enrollment
  until after the C NPC switcheroo draw, while ordinary command combat keeps
  its existing eager enrollment.

## Vehicle evidence and disposition

- Clean-main RED was captured with
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff --scenario spec-proc-carrion --show-oracle`:
  the C room special emitted the stalker Act while clean-main Go fell through
  to ordinary `say` output.
- GREEN gate evidence is `spec-proc-carrion` at seed 1. The matching success
  vehicle `spec-proc-carrion-success` matches with `--show-oracle` at seeds 1,
  2, and 8. The peer confirms that the Act is delivered to both room players
  before the canonical hit transcript, and the focused tests pin the hidden
  spawn/damroll state.
- The nine manifest rows are `room.carrion-commandless-gate`,
  `room.carrion-argument-gate`, `room.carrion-sleeping-gate`,
  `room.carrion-keyword-gate`, `room.carrion-return-fallthrough`,
  `room.carrion-spawn-audience`, `room.carrion-spawn-state`,
  `room.carrion-hit-delegation`, and `room.carrion-return-intercept`.

## Verification and integration

- PR #770 (`glm/spec-carrion`) passed lint, security, and test checks and was
  squash-merged to `main` as `0c1f07914`.
- `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check` all passed.
- No file under `src/` or `darkpawns-c-oracle/` was edited.

This slice applies R1/R2 (exact room bytes and command interception), R3
(seeded combat draw parity and state ordering), R4 (no invented fallthrough or
combat output), R5b/R5c (shared combat ownership), and R5e (verified C
dispatch/call path).

## Next queue item

Continue the special-procedure inventory with `mindflayer` in source and
registration order. Do not repick `carrion` or any procedure claimed by an
earlier handoff.
