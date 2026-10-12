# DP-1371 C1 case 2: actor-aware native specials

## Slice

The shared special registry stores a `SpecEntry` with the legacy player callback and an optional actor-aware callback. `GetSpec`, `GetMobSpec`, `GetRoomSpec`, and the object compatibility adapter preserve legacy callback lookup. `DispatchCharacterSpecial` preserves the player callback lane, calls the actor lane for NPCs, and uses `Unsupported` for a missing/empty native entry or an unported legacy-only NPC special. `Declined` is reserved for a command that this C special is proven to reject.

The first actor conversion is `no_move_east`, assigned to mob 14410 in `pkg/game/spec_assign.go`. It uses the actual MobInstance body. `ActWithPrivateSink` changes only the final write for that same Actor after `actDeliver` has applied its real-body visibility, receive-position, room iteration, and identity checks. The existing player callback and ordinary `Act` path are unchanged.

## C evidence and limits

- `src/spec_procs2.c:1823-1842`: `no_move_east` checks nonzero `cmd`, `mini_mud`, `IS_MOVE(cmd)`, awake host, actor level `>= LVL_IMMORT`, `HUNTING(ch)`, east direction, and the second `< LEVEL_IMMORT` guard before the two acts. `src/structs.h:620-621` defines `LEVEL_IMMORT` as an alias of `LVL_IMMORT`. C sets `mini_mud` through its `-m` startup path at `src/comm.c:194-196`; the Go tree has no mini_mud runtime/config flag, so this Go branch cannot vary.
- The old player callback at `pkg/game/spec_procs2.go:1951-1962` still only rejects NPC actors and checks `cmd == "east"`; missing C guards (movement class, awake host, actor immortality/hunting, and the C mini_mud mode) remain a separate fidelity finding. This slice intentionally leaves player behavior unchanged.
- The pulse specials `citizen`, `cleric`, and `dragon_breath` reject nonempty command-time calls at `src/spec_procs.c:988`, `1430`, and `931`. Empty-command dispatch remains `Unsupported` because it may represent a pulse call; it is not classified as a C command rejection.
- Other narrow NPC rejection proofs are `src/spec_procs.c:208` (`guild`), `src/spec_procs2.c:1877` (`key_seller`), `src/mail.c:478` (`postmaster`), `src/spec_procs2.c:688` (`remorter`), and `src/spec_procs3.c:1294` (`fly_exit_up`). All other legacy-only NPC specials remain Unsupported.

This slice does not claim generic native-special parity. Actor-aware entries beyond `no_move_east` remain future work.

## Verification

Final-state checks passed independently: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...` (0 issues), `make deadcode-ratchet` (0 new), `make fidelity-depth` (5,478 cases; 5,384 proven/delegated), and `make fidelity-units` (1,489/1,489). The focused actor tests and actor-registration, real-body-sink, and missing-special-result mutation controls also passed; each mutant was killed and restored. Logs and mutant patches are retained at `~/Archives/darkpawns/oracle-runs/2026-10-11/dp1371-c1-foundation-audit/`.
