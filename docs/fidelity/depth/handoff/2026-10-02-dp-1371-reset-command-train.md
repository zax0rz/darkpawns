# E4b reset commands: eight-case train

Base: origin/main f0e8546bf after #1761. Implements the next part of the
approved #1760 design. Eight narrow unit rows; the aggregate
`zreset.reset-zone-state` remains blocked. Equipment (`E`) and the remaining
`L`/if_flag matrix are next. No complete reset-engine parity claim is made.

## Cases and other readers (R5b/R5c/R5e)

1. **D door state / secret mark.** `src/db.c:2246-2274` clears
   ROOM_SECRET_MARK for every valid exit, including an unknown state. One
   World mutation publishes flags and exit state together. Tests preserve
   unrelated bits and prior snapshots; invalid/missing exits fail the next
   conditional. Movement/search, redit, web room readers and atomic topology
   snapshots see the same canonical publication. No editor definitions or
   player-facing strings change.
2. **P global target / ownership.** `src/db.c:2167-2184`,
   `src/handler.c:851-860,939-954` select the newest matching global object,
   including instances outside the spawner. World assigns increasing IDs in
   `newObjectInstanceLocked`; restores and synthetic-object creation use that
   registry. P permits non-container item types, as C obj_to_obj does, and
   prepends canonical container ownership. The player put command retains its
   existing item-type gate. `GetObjNum` had no production callers; P is now its
   caller. Spawner's old lookup remains test-only. Global caps, recursive
   extraction, inventory persistence and Lua's world registry resolve the same
   object IDs. These are ownership proofs, not a complete weight-consumer audit.
3. **R mobile deferred removal.** `src/db.c:2228-2242`,
   `src/handler.c:1194-1208`: newest eligible canonical room body, skip fighting
   or already marked, destroy possessions now, mark and retain global counts
   until heartbeat drain. R replaces the reset-local last-mob pointer, including
   nil on a failed search. Private spawner caches release their references;
   they do not supply global caps. `HasPendingExtractions`, session extraction,
   NPC display filters, Lua lookup and later M caps use the existing marked
   body / canonical registry. Tests cover a later G using R's selected pointer
   and a missing R clearing that pointer. Broader C3 extraction is unclaimed.
4. **R object contents.** `src/db.c:2220-2227`,
   `src/handler.c:1006-1031`: first canonical room match, all children extracted,
   conditional success only on a found target. A three-child failing-first
   test exposed sibling skipping in shared extraction. Detaching the contents
   list before recursion fixes the class without changing locks. Other callers
   include Lua extraction/cleanup, consumables, money pickup and donation;
   ownership and global counts still clear through the same helper. Descendant
   spawner placement caches are not canonical identity/count readers; no cache
   redesign is included.
5. **M loading / placement.** `src/db.c:2107-2146` retains the last body on a
   capped load, counts world-owned bodies outside the spawner, and runs zone79
   rejection before RANDZON rejection. A cross-zone M exposed Go choosing the
   initial room's zone; C chooses the owning reset zone. The local fix uses
   `zone.Number`. Tests assert all restricted flags, city/zone163 distinctions,
   six exact placement draws, >5 retries, cap behavior and conditional effects.
   Room entry sequences, descriptor occupancy, specials and combat see the
   existing canonical relocated body; no new dispatcher or room-selection RNG
   is introduced. The old uppercase-RANDZON fixture now supplies its owning
   zone explicitly.
6. **O state / draw order.** `src/db.c:1899-1925,2148-2165`,
   `src/handler.c:897-910`: global caps, canonical prepended room ownership,
   floating objects without percent_load, rare initialization before percent,
   and failure extraction. Existing draw-range/sign tests are retained. Other
   readers are room listings, global caps, Lua lookup and extraction. Proof-only
   case; production behavior unchanged.
7. **G state / draw order.** `src/db.c:2186-2200`,
   `src/handler.c:559-575`: reset-local body required, global cap without extra
   draws, canonical inventory prepend, and a failed conditional preserving
   prior success. Existing rare/percent extraction tests are retained. The
   actual mob Inventory and ObjectLocation feed inventory views, ownership,
   extraction and Lua references. Proof-only case; no equipment or fake PC
   adapter is introduced.
8. **P self target.** `src/db.c:2167-2184`,
   `src/handler.c:939-946`: loading the target prototype itself makes the new
   object the newest global match. C refuses that self-link while reset_zone
   still records percent success. The object remains floating and counted;
   its conditional successor executes. Tests assert both percent calls and
   no contents link. Floating-object counts and later global lookup are the
   other readers. Proof-only case using case 2's implementation.

## R5h controls

All controls execute Go assertions (not compilation errors), recording fixed,
reverted and restored exits **0/1/0**. Logs are under
`~/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-reset-command-proofs/`.
The label is the log prefix; `*-fixed.log`, `*-reverted.log` and
`*-restored.log` retain the outcomes. The original D failing-first and
`door-fixed.log`, plus `door-control-reverted.log` and
`door-control-restored.log`, are its triple.

| Labels | Perturbation that fails |
|---|---|
| door | omit clearing ROOM_SECRET_MARK |
| put-newest, put-self-target | change highest-ID selection to lowest-ID |
| put-target-type | restore an ITEM_CONTAINER gate |
| remove-mobile-deferred | omit MOB_EXTRACT marking |
| remove-mobile-fighting, remove-mobile-marked | remove each eligibility gate |
| remove-mobile-pointer | discard R's returned pointer |
| remove-object-children | traverse the parent slice while children remove themselves |
| mobile-retained-pointer | clear lastMob on a capped M |
| mobile-global-cap | use spawner-local counts |
| mobile-owning-zone | use the initial room's zone |
| mobile-zone79-city | accept city rooms in the zone79 sampler |
| object-room-order | append room placement instead of prepend |
| object-global-cap | use spawner-local counts |
| object-floating-percent | call percent_load for NOWHERE O |
| object-init-before-percent | omit init_rare |
| give-missing-mob | invent lastMob from the world registry |
| give-global-cap | use spawner-local counts |
| give-prepend | append to the mob inventory |
| give-conditional-success | reset lastCmd even on conditional commands |
| give-percent | bypass the G percent gate |

Each commit has separate successful normal gates, retained under its
`*-gates/` directory: fmt, build, vet, all tests, clean lint cache, lint,
diff check, fidelity-depth, fidelity-units and string census. Final race
validation is `race.log`, including ten repetitions of the concurrent reset /
clock / canonical reader test. The combined census's separate full/claims
verdicts and evidence directory are supplied in the PR after it completes.

An unrelated normal-suite failure, the immediate Freshwire reconnect name
refusal in `TestEntryTelnetNewCharacterDisconnectResumes/menu=false`,
reproduced twice in 50 repetitions on untouched f0e8546bf. Retained evidence:
`remove-object-gates/test.log` and `main-entry-transport-repeat.log`. The later
normal gates pass. This is a retained entry follow-up, not repaired here.

## Lock acquisitions

No reset path acquires the player lifecycle lock. The train retains #1761's
single reset mutex and changes no admin caller's outer locking.

- Boot: `zoneResetMu` over the sorted boot pass, then command-specific locks.
- Wizard/manual/admin single zone: `zoneResetMu`, brief World read lock for
  zone lookup, release World lock, then execute. Admin all-zones endpoints
  snapshot GetAllZones before entering ResetZone individually. Reviewed Huma
  reset-zone/reset-all-zones (`pkg/admin/huma_completion.go:235,312`) and legacy
  single/all routes (`pkg/admin/handlers.go:1077,711`): no outer manager,
  lifecycle or World lock at the ResetZone call; audit logging follows it.
- Automatic heartbeat: `zoneResetMu`; occupancy's brief Manager read lock is
  released before body/world resolution and command execution.
- M/O/G creation: existing Spawner write lock then World write lock; placement
  uses the existing Spawner -> World -> Mob order. G movement uses World and
  the existing mobile ownership helper. No added lifecycle acquisition.
- P: separate World read lock for newest lookup, released before the percent
  gate; World write lock for canonical attachment. No Spawner lock spans it.
- D: World write lock for clone/update/publication; publication is atomic and
  adds no lock.
- R mobile: World read snapshot released before Mob reads and room ordering;
  Mob read snapshot of possessions released before extraction's World write
  lock. Mark under Mob write lock, release, then private-cache Spawner write
  lock (which briefly takes Mob read locks). No Mob -> World nesting is added.
- R object/shared extraction: World write lock, with existing owner-specific
  Inventory/Equipment/Player/Mob locks where applicable. The traversal repair
  adds no lock or acquisition-order change.

Retained frontier remains browser name routing, identity consumers,
storedPlayer/PlayerStoreEdit loaded-inventory ownership, legal-quit rented
object cleanup, harness port bind race and users Login@ wall-clock behavior.
