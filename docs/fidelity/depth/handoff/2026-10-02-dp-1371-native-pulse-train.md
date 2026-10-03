# Phase 4 native pulse train

Base: origin/main 4591db664 after #1758. Zach live-playtested switch/return,
both reconnect identities and occupied-original return before merging. This
train ends at the native-special family boundary; reset-engine coverage is next.
No production access, oracle repair, harness change or governing-document edit.

## Janitor autonomous dispatch

`mob.janitor-pulse-dispatch` uses `spec-proc-janitor-pulse@1,2,3,5,8`.
C: src/mobact.c:68-93, src/spec_assign.c:292,505 and
src/spec_procs.c:750-768. The earlier fixture's janitor was absent because
its shipped action flags include RANDZON: src/db.c:2131-2144 relocates it at
boot. The new paired disposable fixture clears only RANDZON, leaves the native
assignment intact and removes exits. It observes presence before dropping bread,
then both actor/witness pickup bytes and floor removal after the pulse.
TestJanitorAutonomousAssignedDispatch traverses World.MobileActivity for both
real assigned vnums, checks the exact object's mob-inventory location and proves
TRUE consumption prevents the subsequent movement/sound draws. Clearing both
assignments gives an assertion-failing 0/1/0 control.

Other readers: room object index, typed ObjectLocation, mob inventory ordering,
observer delivery, native registration and mobile-activity fallthrough are
covered. No production code changes for this case. The carried-object proof is
unit state; full NPC stat-report bytes are not claimed. Earlier presence/stat
experiments and failed fixture attempts remain retained; the final pulse vehicle
is separately CLEAN. A nonexistent native cannot certify autonomous dispatch.

## Take-to-jail breed caller

C src/spec_procs2.c:1427-1468 interleaves outlaw, breed_killer and protection
checks in one world[].people loop. Go omitted the breed call and used a separate
all-outlaws-first scan. The fix restores the one loop, using existing runtime
RoomEntrySequence to merge the two Go stores into C's newest-first arrival order.
The ordering change stays local to take_to_jail, not the shared cityguard helper.

`mob.take-to-jail-breed-killer` is delegated to the already-proven
`mob.cityguard-breed-killer` owner. The new narrow
`mob.take-to-jail-breed-caller` row claims
`spec-proc-take-to-jail-breed@1,2,3,5,8`: assigned mob 8001 reaches the nested
special on a real autonomous pulse (src/spec_assign.c:289-291;
src/mobact.c:68-93). The C transcript actually emits Hans's nightbreed speech
and a hit block. The existing outlaw/subdue vehicle remains CLEAN.

TestTakeToJailBreedCallerAndRoomOrder fails on the original production code;
then proves an earlier breed intervention consumes before a later outlaw, and
an outlaw at the list head keeps priority. TestTakeToJailBreedCallerEntryGates
covers sleeping and nonzero-command rejection. Removing the handoff and removing
the arrival-order merge each independently fails assertions (0/1/0).

Other readers: awake/fighting gates, outlaw warning and synchronous hit/fighter
boundary, protection selection and its existing tests, room arrival sequences,
shared breed visibility/attack/state effects are accounted for. The helper's
internal full matrix remains owned by its cityguard manifest. Go-only admin,
transport, persistence and editor code do not consume this local candidate list.
No store schema, authentication identity or lifecycle lock changes.

## Evidence and gates

Retained proofs: ~/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-e4-proofs/.
Targeted runs: dp-1371-janitor-presence, dp-1371-janitor-fixed-placement,
dp-1371-janitor-native-stat (failed exploratory vehicles retained),
dp-1371-janitor-pulse-final (CLEAN), dp-1371-jail-breed-caller (2/2 CLEAN).
Final five-seed assertions are gated by the tip's combined census manifest.
Every commit runs normal gates separately. The first janitor gate run hit an
entry-disconnect retry timeout and the retained E2E port-bind race; the complete
janitor-gates-retry suite passed. These failures are not proof controls.

## Retained frontier

Continue in batch order with zreset's state/RNG matrix, then the separate
teleport combat recheck. Retain browser name routing, identity consumers,
storedPlayer and PlayerStoreEdit temporary loaded-inventory ownership,
legal-quit rented-object cleanup, harness port-bind race and users Login@
wall-clock. Entry login restrictions still wait for D6. Full NPC stat-report
comparison is outside the janitor pulse claim and was not repaired here.
