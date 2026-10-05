# D2: faithful shoot target path

**Stop tier — design note.** This is the next Phase 4 family after #1791.
No behavior, scenario, manifest status, or reference binary changes here.
All nine shoot cases remain blocked. The existing D2 authorization chooses
C parity; this note specifies the implementation boundaries for that repair.

## C contract and confirmed gaps

The registered path is `shoot` → `command.CmdShoot` → `game.DoShoot` →
`sendSkillResult`. The current result supplies invented messages, generic
skill damage and a wait, and the command relocates both player and mob targets.
C `src/act.offensive.c:746-998` instead does the following:

- Reject an unskilled or already-fighting shooter before argument/object lookup
  (:760-770), then run the existing projectile/bow/direction/door/peaceful gates.
- Resolve `get_char_room(arg3, targ_room)`, falling back to the room's first
  person (:862-872). `src/handler.c:865-883` matches numbered abbreviated names
  in room-list order **without CAN_SEE**, without a player-only `0.` special
  case, and without the generic resolver's `self`/`me` shortcut.
- Reject PC victims below 10 or above 30, fighting victims, then sentinel mobs
  (:881-900). No projectile or RNG change occurs on these refusals.
- Emit the origin act and actor twang, remove the projectile from inventory,
  draw `number(1,101)`, and compare **strictly less** than skill plus shooter
  dexterity missile adjustment minus victim dexterity reaction (:904-920).
- On a hit, damage is DAMROLL plus projectile dice plus bow dice, in that order.
  Print the roar, extract the projectile, then improve shoot (:922-930).
  There is no `damage()` or `skill_message()` call and no WAIT_STATE.
- NPC hit: destination act, direct HP subtraction and `update_pos`; if dead,
  `die()` and return. Otherwise relocate to the shooter, emit arrival acts,
  then synchronous `hit(to, ch, TYPE_UNDEFINED)` (:932-955).
- PC hit: victim line, destination act, direct HP subtraction and `update_pos`,
  then `die()` if dead. No relocation or combat initiation (:959-973).
- Miss: victim and destination acts, then prepend the projectile to the
  victim's room (:978-987). No damage, improvement, relocation, retaliation or wait.

`src/fight.c:186-203` supplies update_pos thresholds. `die()` at :625-631
calls `gain_exp(ch, -(GET_EXP(ch)/3))` and `raw_kill(TYPE_UNDEFINED)`. It bypasses
`damage()`'s kill rewards, death counter, PK bookkeeping, and killer-aware
special/script hook. The existing `World.HandleNonCombatDeath` delegates to a
player helper that increments `Deaths`; it cannot certify this C boundary.

## Proposed boundaries

1. Keep the command's entry/object/exit gates, correcting ordering only where
   the actual C read requires it. Replace the targeted `SkillResult` route with
   a C-shaped shoot executor. Remove the unused simplified DoShoot after the
   command is migrated; do not leave an alternate callable implementation.
2. Add a narrowly named shoot target lookup over actual PC and NPC bodies,
   merged by descending RoomEntrySequence. This runtime sequence already models
   char_to_room's prepend in movement/spawn and Lua RoomFields. Apply C's
   GetNumber/name matcher and fallback there. Do not alter the generic visible
   resolver or its other callers as part of shoot.
3. Use the existing dex_app table through a small accessor for its missile and
   reaction fields. Avoid a copied table or fitted probability constants.
   Keep draw/arithmetic calculation independently testable, with actual object
   values, integer HP deltas, and the existing ImproveSkill call at C's boundary.
4. Direct projectile HP loss goes through a narrowly named legacy ranged death
   boundary when update_pos reaches DEAD. Compose C's gain_exp and raw_kill
   semantics. Preserve the existing general combat/non-combat callers; add no
   killer credit, PK flag, death-counter increment, CON loss, or damage messages.
   Audit raw_kill cleanup/deferred extraction for both real body types. Any
   missing shared effect gets its own demonstrated failing test before a fix.
5. NPC survivors use canonical silent relocation, then the existing hit engine
   for exactly one synchronous swing. `StartCombatFromMob` defers defender
   enrollment, while `PerformUnenrolledInitialAttack` currently has a different
   subsequent-turn contract. Neither may be selected merely because its name
   resembles hit. Prove the retaliation's protection checks, draw order,
   pre-fight victim posture, and both ensuing fighter memberships against C
   damage (:1314-1458) before choosing/reusing the seam. If needed, add a direct
   hit entry that shares performOneHit and registers only the fighters C's
   damage actually enrolls. Existing hit callers keep their current behavior.

No synthetic players, new registry, new output buffering, save-format change,
or new lifecycle lock. Snapshot world lists under their existing locks, release
before querying bodies, sending output, moving objects/bodies, or invoking
combat/death. The implementation PR lists the concrete acquisitions after the
code exists; no lock-order assertion is claimed from this design.

## Trains, proofs and retained boundaries

First train: target lookup/fallback, PC level window, and target-fighting refusal
(one commit per case). Refusal scenarios can terminate at sentinel/PC/fighting
arms, so these gates do not need a fake successful target path to earn proof.
Second train: mob outcomes, player outcomes, actor/audience output, projectile
and zero-wait contract, and no skill-message path (six cases). End earlier if
production changes reach about 500 lines. Dependent rows remain blocked until
all their required outcomes are proven. Both implementation trains are stop
tier if they touch the death/extraction or session boundary; classify from the
actual diff, not this estimate.

Proofs use real issuing, victim/destination, and origin observer connections.
Inspect retained C output for every claimed branch at seeds 1,2,3,5,8. Add focused
units for invisible/numbered/fallback room ordering, mixed PC/NPC arrival order,
PC levels 9/10/30/31, fighting-before-sentinel ordering, exact hit/miss draws and
next draw, equipment dice, improvement ordering, HP thresholds, player immobility,
NPC relocation and synchronous retaliation, and zero wait. Lethal tests pin
ordinary corpse/deferred extraction, EXP boundary and absence of damage-path
kill bookkeeping. Every newly claimed case gets a compiling assertion-based
revert triple, including dex terms, comparison, fallback, each refusal, PC move,
projectile extraction/drop, wait insertion, direct death and retaliation removal.

Other readers audit: generic targeting, equipment getters and dex_app consumers,
combat rounds/enrollment/protection callbacks, following/mount/memory cleanup,
corpse generation and deferred extraction, autosave/logout persistence, syslog,
transports/switch-owned output, kill counters/XP/PK state. Trace shared combat
failures before expanding the batch; reproduce a proposed pre-existing failure
on fresh origin/main with the reference oracle. No repins or dropped seeds.

One combined census at each implementation tip, plus normal gates,
fidelity-depth and fidelity-units before opening. This note earns no oracle
or unit claim and needs no census. No divergence is proposed.

## Verification

Docs-only at origin/main `0821e3d32` (#1791 merged). `make fmt`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint cache clean`,
`golangci-lint run ./...`, `git diff --check`, `make fidelity-depth`,
`make fidelity-units`, and `make string-census` each exited 0. The inventory
remains 27 blocked, including these nine cases. Gate logs are retained at
`~/Archives/darkpawns/oracle-runs/2026-10-05/dp-1371-shoot-design/`.
No oracle execution or census is claimed for this design-only change.
