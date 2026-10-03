# E4b reset matrix: shared-state design and scope stop

Base: origin/main 0f26c977f, after #1759. Design only: no game, harness,
manifest status, oracle binary or governing document changes. No production
access. Existing reset greens are not a complete reset_zone proof.

## Concrete gap and decision

The next case, zreset.reset-zone-state, includes C's final age assignment
(src/db.c:2285). Go's parsed Zone has definitions only; neither World nor
Spawner owns runtime age. cmdZreset calls World.ResetZone, which calls
Spawner.ExecuteZoneReset; no age assignment occurs.

The live producer is also different: cmd/server/main.go starts
StartPeriodicResets(60*time.Second); spawner.go:719-765 runs every empty zone
on each wall-clock minute, disregarding lifespan and reset mode, and disables
itself under DP_CLOCK. The existing engine OnZoneUpdate callback is unwired;
its PULSE_ZONE is 60 seconds, while src/structs.h:632 is 10 seconds.
C comm.c:805-808 calls zone_update after pending extraction. C db.c:1967-2045
increments age once per minute, enqueues eligible zones in table order with
age=999, and scans the queue every ten seconds, removing at most one eligible
entry. Reset mode 2 bypasses occupancy; mode 1 waits; mode 0 does not age/queue.

This is reachable through normal production startup, not a guessed future
consumer. Closing the age part needs shared runtime state and a real producer.
A test that writes a synthetic age into a new map and resets it would not prove
that call path (R5e/R5h). The aggregate row stays blocked.

**Decision requested:** expand E4b to port this zone-clock class faithfully,
using the design below, rather than adding a reset-only age placeholder.
This is a broader shared-path change than the manual reset matrix: it changes
automatic spawning and RNG during pumped/live time. The goal's design-first
rule and shared-path stop apply. If the expansion is deferred, proceed with
manual command subcases and keep the age/automatic-clock remainder explicitly
blocked; do not call the aggregate complete.

## Proposed representation and execution

1. World owns runtime zone state separately from parser.Zone: age per zone,
   minute accumulator, FIFO reset queue keyed by zone number, and a snapshot
   API for future show zones. Runtime state is not written to world files or
   player saves. Definitions retain C table order through GetAllZones.
2. One zone-update heartbeat runs at C's PULSE_ZONE=10 seconds in both live
   and DP_CLOCK modes. Remove the server's standalone periodic reset driver
   so there is no second schedule. Keep startup boot resets in table order.
   Age increments after six updates, with C's exact queue/sentinel behavior.
3. All boot/manual/automatic reset entries use one serialized execution path.
   A reset execution mutex precedes short runtime-state/world snapshots;
   release world/manager locks before executing commands or emitting output.
   No lifecycle lock. Review every admin ResetZone caller for lock inversion.
   Automatic queue manipulation and manual age-zero assignment share that
   serialization. C manual reset does not silently delete a queued entry;
   preserve that behavior, including its next automatic reset.
4. Occupancy is descriptor state, not World player membership.
   src/db.c:2290-2300 walks playing descriptors only. Add a narrow manager
   callback returning occupied zones from descriptor-attached bodies, including
   PC/NPC switching, excluding linkdead holders and entry/menu descriptors.
   Snapshot under the manager lock; take no lifecycle lock; release manager
   before world zone resolution. Existing identity registry stays unchanged.
5. Cross-instance effects remain canonical: equipment uses typed object
   movement; mobile removal marks deferred extraction and retains global
   counts until the existing extraction pass; door resets publish room state
   through existing world snapshots. No fake mobile/player adapters.

No player-facing strings or deliberate divergence are proposed. These are
implementation choices for C's semantics, with explicit scope approval needed
for the shared automatic producer and occupancy callback.

## Reset command proof/fix matrix

Read C src/db.c:2074-2285, handler.c:1194-1208, act.wizard.c:2035-2076.
Existing spawner tests already prove useful O/P/G draw-order and cap branches;
retain them and add individual assertion-failing 0/1/0 controls rather than
counting their names as new proof.

| Surface | Required state/RNG assertions |
|---|---|
| M | successful load/cap/failure and retained last mob; zone79 vs RANDZON constraints, retry order and draws; conditional effects |
| O | room/floating object, global cap, init_rare before percent, success/failure extraction, front insertion |
| P | global newest matching container, missing-container floating object, successful ownership, percent failure, draw order |
| G/E | missing/retained last mob, cap and percent, inventory prepend; equipped slot/location/affect boundary; invalid/occupied-slot C behavior |
| R object | first room match, possessions/contents cleanup, missing target and conditional result |
| R mobile | newest matching room occupant, skip fighting/already-marked; destroy possessions now, mark extraction, retain count until drain; last-mob pointer effects |
| D | invalid/missing exits; states 0/1/2 and invalid state; preserve unrelated exit bits; clear ROOM_SECRET_MARK on a valid door even if state is invalid |
| L/if_flag | start/end iterations, conditional skip, failed conditional leaves prior success, unconditional reset, comments/unknown command disabling |
| age/queue | manual/boot age zero; mode 0/1/2, lifespan boundary, age=999 sentinel, FIFO order, blocked head bypass, one reset per update, queued manual reset |

Source-read gaps requiring failing-first confirmation include E's direct
Equipment map assignment without canonical ObjectLocation, D's missing secret
mark clear, and R mobile's immediate removal/oldest-first spawner list. They
are candidates for local C repairs within this matrix, not claimed fixed or
unit/oracle-proven by this note.

Disposable paired zone-reset fixtures may be needed to expose E/R/L command
combinations absent from shipped files. First use existing fixture controls;
any required fixture extension gets parser/application/cleanup controls and
retains exact transcripts. Never edit src/ or the shared oracle checkout.

## Implementation trains and gates after approval

One commit per command/state case with C citations, failing-first tests where
needed, assertion-failing revert triples and other-reader audits. Split at
about eight cases or 500 production diff lines, and at family boundaries:
first runtime clock/occupancy if approved; then reset command repairs/proofs;
then the separate teleport combat recheck. Keep the aggregate blocked until
its full matrix is evidenced, delegating to proven subrows when complete.

One combined census per implementation train at its committed tip, verifying
all claimed seeds 1,2,3,5,8. Adding automatic zone updates can change old
long-pump scenarios: establish paired C output at the exact pulse and explain
newly matching blocks; do not repin, drop seeds or normalize away changes.
Investigate any PASS-to-FAIL; attribute pre-existing failures on origin/main.
Run all normal gates and race checks for reset/admin/update concurrency.

Other-reader audit includes startup, wizard zreset, admin ResetZone endpoints,
zone definition editing/deletion, future show zones snapshots, global object/
mobile counts, pending extraction, room snapshots, Lua object/character
lookups, descriptor switching/menus/linkdead occupancy, and DP_CLOCK cadence.
This does not authorize the broader C1 dispatcher or C3 Lua extchar repair.

## Retained frontier

Browser name routing, identity consumers, storedPlayer/PlayerStoreEdit loaded
inventory ownership, legal-quit rented-object cleanup, harness port-bind race
and users Login@ wall-clock remain retained. Entry restrictions wait for D6.
No new completeness claim is made. The teleport combat recheck remains next
in batch order after the reset matrix; no combat repair is folded into this
shared-state design.
