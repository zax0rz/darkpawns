# Depth-fidelity handoff — elements_master_column

Date: 2026-08-30  
Slice: special procedure `elements_master_column`  
C declaration/registration: `src/spec_assign.c:85-180,622`  
C definition: `src/spec_procs3.c:936-1002`  
Go registration: `pkg/game/spec_assign.go:378`; `pkg/game/spec_procs3.go:90`  
PR: #784, merged as `d1ec0f08b`  
Hosted checks: run `33294026767` (`test`, `security`, `lint` green; optional build/deploy skipped)

## Queue position

The fresh session began on `main` at the `hisc` handoff frontier (1,194 total;
1,148 proven/delegated; 13 blocked; 33 excluded). The next active registration
in C file-and-registration order was room 1315. The next active special after
this slice is `elements_platforms`, registered at rooms 1326, 1337, 1348, and
1359 in `src/spec_assign.c:623-626`.

## C call path and branch inventory

The room special is reached from the current-room player-command dispatcher in
`src/interpreter.c:1407-1456`; it ignores the command text. Movement also
checks room specials from `src/act.movement.c:115`. The active room assignment
is `ASSIGNROOM(1315, elements_master_column)`.

For each player in `world[ch->in_room].people`, C scans carried object VNums
1300-1303, walks the earth/air/fire/water prefix, and selects destinations
1320/1331/1342/1353/1372. It emits the exact no-talisman, partial-prefix, or
four-talisman direct message, the departure `TO_NOTVICT` Act, moves through
`char_from_room`/`char_to_room`, renders `look_at_room`, and emits the arrival
`TO_NOTVICT` Act. `has_object[]` is initialized once per special invocation
and reset only for the processed prefix; the stale carry-state behavior is
therefore player-order observable and was treated as part of R1/R3, not
normalized away.

## RED → GREEN

Added three C-first vehicles:

- `spec-proc-elements-master-column-none` — no talismans, room look, actor and
  peer audience, and relocation to 1320.
- `spec-proc-elements-master-column-stale-state` — actor air-only followed by
  peer earth-only, proving the C stale `has_object[1]` state and 1342 result.
- `spec-proc-elements-master-column-all` — all four talismans for the chamber
  branch and a no-talisman peer.

The matrix was green at seeds 1, 2, and 3, and `--show-oracle` confirmed the
intended C blocks. Focused tests cover nil entry, all prefix/destination
branches, exact VNum classification, stale carry state, room audience, and
canonical transfer. The Go room path now preserves C's actor-first vehicle
ordering, direct message CRLF framing, room look, and first-player God's
`PRF_AUTOEXIT`-off state (C's first-player `init_char` skips `do_start`).

No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .
git diff --check
```

All local gates passed. The manifest now reports:

```text
1202 total; 1156 proven/delegated, 13 blocked, 33 excluded
Actionable completion: 1156/1169 = 98.9%
```

## Fidelity rulings

This slice follows R1 (exact player-facing bytes), R2 (the active command and
room registration surface), R3 (deterministic multi-player ordering and stale
state), R4 (no invented talisman or destination behavior), and R5e (the C
dispatch and movement call paths were verified before the port change).

## Next action

Start the next session from `main`, pull, run `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this handoff, then map and prove
`elements_platforms` in C registration order. Continue the special-procedure
inventory, then attempt `objmagic.sleep-entry-gates` once via the cast-sleep
vehicle before sweeping the remaining un-manifested interpreter command
families. Leave one dated handoff for the next session.
