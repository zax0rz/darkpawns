# Depth-fidelity handoff: `werewolf`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `e5e41bc2c`  
Merged PR: `#802` (`3f2a98df2`)

## Scope and source audit

This slice covered `SPECIAL(werewolf)` in `src/spec_procs3.c:427-448`.
The actual C path is the ordinary combat loop in `src/fight.c:1898-2032`,
which invokes registered mobile specials after the mob's normal attack. The
procedure is actively registered as `ASSIGNMOB(5510, werewolf)` in
`src/spec_assign.c:251`. The audited world mob is vnum 5510 in
`lib/world/mob/55.mob`; its authored Lua script was stripped in the disposable
fixture while the native SPEC assignment was retained.

The C body has commandless/fighting/positive-HP gates, a 0..9 howl draw, and a
0..3 bite draw. Howl uses room `act()` plus `send_to_zone`; C's latter sends
only to awake players in other rooms of the same zone. Bite uses separate
`TO_VICT` and `TO_NOTVICT` acts, `TYPE_BITE` damage, and a level*1.5 integer
movement reduction floored at zero. The Go fix uses canonical `Act` for the
same audiences and pronouns, a procedure-local zone sender matching C's
`src/comm.c:2377-2387` audience, the combat target boundary, and the player
movement state update. The broad `World.SendToZone` helper was not changed;
R5c keeps this C-specific audience rule local to the procedure.

## Proof and disposition

`cmd/dp-oracle-diff/scenarios/spec-proc-werewolf.txt` reaches the registered
mob-5510 fighting path with same-room victim/observer and remote same-zone
observer. The live oracle vehicle is green for seeds 1, 2, 3, 5, and 8, with
`--show-oracle` confirming the fighting special block executes. Focused tests
cover entry gates, howl room/zone/sleeping audiences, bite victim/room
audiences and C pronouns, TYPE_BITE movement reduction, and the explicit
zero floor.

The manifest adds seven cases: one D1 entry gate, one D3 live howl audience,
two D3 bite-audience claims, two D4 movement-state claims, and one D5 live
combat-dispatch claim. No `src/` or `darkpawns-c-oracle/` file was edited.
The work enforces R1/R2/R3/R4/R5c/R5e: player-facing bytes and command
surface remain C-authoritative, deterministic draws and state are preserved,
the real dispatch path is used, and the audience fix is scoped to the audited
behavior class.

## Verification

All required gates pass on the merged slice: `make fidelity-depth` reports
1,291 total cases, 1,241 proven/delegated, 14 blocked, and 36 excluded
(1,241/1,255 actionable, 98.9%); `go build ./...`; `go vet ./...`;
`go test ./...`; `golangci-lint run ./...` (`0 issues`); `gofumpt -l .` (no
output); and `git diff --check`. Hosted PR checks for lint, security, and
test passed; build/deploy were skipped by workflow conditions.

## Next queue item

Continue special-procedure source order with `SPECIAL(field_object)` at
`src/spec_procs3.c:456`, auditing its active object registrations
`ASSIGNOBJ(50, field_object)` and `ASSIGNOBJ(52, field_object)` before building
the object-special vehicle.
