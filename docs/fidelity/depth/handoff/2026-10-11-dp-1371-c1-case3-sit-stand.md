# DP-1371 C1 case 3: actor-aware sit and stand callbacks

## Slice

The existing `command.Entry` keeps its player `Handler` and gains an optional
`ActorHandler`; aliases resolve to the same entry. `sit` and `stand` attach
actor callbacks there while preserving their player handlers. The callbacks
operate on the actual `MobInstance`, route private output through the supplied
descriptor sink, and use canonical room acts. They are callback implementations
only; they are not yet connected to NPC command resolution or interpretation.

## C evidence and limits

- `src/act.movement.c:696-731` defines `do_stand`: sitting/resting/default
  room acts pass `TRUE`; sleeping and fighting are private refusals. At
  Standing, `IS_MOUNTED(ch)` invokes `do_dismount` (`:701-706`). The Go callback
  returns Unsupported for Standing plus `AFF_MOUNT`, since generic NPC
  dismount is not implemented.
- `src/act.movement.c:734-766` defines `do_sit`: Standing room act passes
  `FALSE` (`:740-741`); Resting and default room acts pass `TRUE` (`:751-764`);
  mounted Standing emits only the private mounted refusal (`:738-745`). The
  Go default room-act flag matches the C source.
- The sitting branch calls `send_to_char("You're sitting already.\r\n", ch)`
  (`src/act.movement.c:747-749`); the actor callback routes the same exact
  bytes through `Act`'s private sink (`TestRegisteredActorPositionRefusalsAndMountedDistinction`).
- `AFF_MOUNT` is the actor's mounted state in this branch. `MobInstance` also
  has `MountRider`, which identifies a rider on that mob and is a different
  relationship; it is not used as a substitute for the actor affect.
- The C command interpreter applies position/authorization gates and ordered
  special handling before handler execution (`src/interpreter.c:883-949`,
  `:1407-1480`). This slice invokes registered callbacks directly in tests, so
  those tests prove callback bytes/state only. They do not prove command
  reachability, position-gate behavior, force semantics, Lua action semantics,
  specials order, or generic NPC command parity.

## Verification and remaining proof

The final slice passed `make fmt`, `go build ./...`, `go vet ./...`,
`go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...` (0 issues),
`make deadcode-ratchet` (52 baselined, 0 new), `make fidelity-depth` (5,478
cases), and `make fidelity-units` (1,489/1,489). The actor-registration,
default-sit visibility, and mounted-stand controls each failed their intended
assertion under a compiling mutant; sources were restored and focused actor
tests passed. Logs and mutant patches are retained at
`~/Archives/darkpawns/oracle-runs/2026-10-11/dp1371-c1-foundation-audit/case3/`.
The shared census was checked free; no census or commit was made for this slice.

Tests prove exact private/room bytes and state changes for direct registered
callbacks, including invisible actor output policy, position refusals,
mounted sit refusal, and Unsupported mounted stand. Remaining integration
proof must exercise the shared command resolver/interpreter and its minimum
position gates, existing NPC entrypoints, special ordering, force-targeted
NPCs, Lua `action()`, and private output for a real switched descriptor.

This is a bounded starter set. Native special dispatch, existing NPC say/tell/
emote/give/social behavior, other command families, mounted dismount, Force,
and Lua routing remain blocked; no generic C1 parity claim follows from these
callbacks.
