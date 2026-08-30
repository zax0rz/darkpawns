# 2026-08-30 — depth-fidelity handoff: SPECIAL(prostitute)

## Starting frontier

- Starting main: 3adedebf4 (merged glm/spec-mirror, PR #808).
- Queue slice consumed: first active special-procedure registration after
  mirror: ASSIGNMOB(8023, prostitute) at src/spec_assign.c:283.
- C body: src/spec_procs3.c:670-705.
- Next queue item: SPECIAL(roach), defined at src/spec_procs3.c:707 and
  first registered as ASSIGNMOB(23, roach) at src/spec_assign.c:185.

## C call-path audit

The active vnum-8023 registration is reached by the player-command
special() dispatcher in src/interpreter.c:1407-1456, after mob scripts
with oncmd triggers and before ordinary command handling. Autonomous
mobile_activity() calls the same registered procedure with cmd == 0
(src/mobact.c:68-93), which returns false before any output. The procedure
does not inspect arg: CMD_IS("buy") and CMD_IS("list") are exact command
identity tests.

The reachable branch matrix is:

- commandless, fighting actor, and sleeping actor: false, silent;
- unrelated command: false, preserving ordinary command handling;
- exact buy/list while CAN_SEE(mobile, ch) is false: two ordered
  act(..., TO_ROOM) lines, delivered to the actor and room peers except the
  mobile;
- visible buy/list: do_tell(mobile, ...), direct to the actor only;
- CAN_SEE uses the mobile's blindness gate, room LIGHT_OK/infravision, and
  target invisibility gate (src/utils.h:515-530).

## RED, fix, and proof

The first valid RED vehicle initially had a real vnum-8023 mob but the old Go
procedure broadcast service text to the room, omitted the actor's C tell
format, and used mobCanSee(me) instead of CAN_SEE(mobile, ch). After the
fixture's RANDZON flag was cleared, the old Go player-command path produced
no direct list/buy tell while the oracle did. The authored
elven_prostitute.lua was stripped in both disposable worlds; no source or C
oracle files were edited.

The Go fix:

- uses a procedure-local World-aware CAN_SEE predicate for the C blindness,
  darkness, infravision, and target-invisibility gates;
- routes the hidden-target pair through canonical Act with C's exact
  two-line TO_ROOM audience and ordering;
- routes visible buy/list through the existing direct tellFromMob helper
  with the exact C message bodies;
- adds nil-safe entry handling without changing the autonomous call contract.

Live vehicles:

- cmd/dp-oracle-diff/scenarios/spec-proc-prostitute.txt: visible list/buy,
  unrelated say fallthrough, and registered player-command dispatch;
- cmd/dp-oracle-diff/scenarios/spec-proc-prostitute-hidden.txt: real
  invisibility spell, wait drain, and hidden list/buy room audience.

Both vehicles are GREEN for seeds 1, 2, 3, 5, 8. Focused tests in
pkg/game/spec_prostitute_test.go cover commandless/fighting/sleeping and
exact-command gates, direct tell audience, invisible-target audience,
darkness/infravision, and blind-mob behavior.

## Frontier counts

After the slice, make fidelity-depth reports:

- 1,313 total cases;
- 1,259 proven/delegated;
- 14 blocked;
- 40 excluded;
- actionable completion: 1,259/1,273 (98.9%).

The special-procedure inventory and the blocked
objmagic.sleep-entry-gates row remain tracked; this slice did not alter
either. Rules applied: R1/R2 for exact bytes and command surface, R3 for the
multi-seed vehicle, R4 for no invented behavior, R5e for the verified C call
path, and R5c for the canonical visibility/audience class.
