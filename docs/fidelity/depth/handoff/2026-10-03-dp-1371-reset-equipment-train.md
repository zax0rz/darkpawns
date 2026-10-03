# DP-1371: reset equipment boundary and loop train

Base: origin/main 45bd1994a, after #1763. This train proves three narrow
reset subcases. The aggregate remains blocked: these are not full equip_char
semantics. No oracle, harness, reference hash, save format, governing documents,
pins or expected-divergence rows change. No production access.

## Skipped shared equipment remainder: proposed next scope

Fresh C read: src/handler.c:609-795, src/db.c:2201-2219,
src/class.c:737-764, src/structs.h:390-412. Reset E calls the full equip_char:
alignment refusal, take-name description, slot-weighted armor, room light,
all equipment modifiers/flags, then check_for_bad_stats. It does not merely
attach an object and recompute attributes.

The local slot/ownership boundary below is within #1760's approved matrix.
The remaining class is wider than reset: Mob.GetAC reads only prototype or
runtime override; hitroll and affect readers similarly have independent state.
Mob.EquipItem, generic movement attachment, movement detachment and extraction
do not share full C equip/unequip effects. Mob.HitModifiers reads Go SlotWield=6,
while reset/mobile storage uses C WEAR_WIELD=16. Player equipment has its own
C-to-Go slot mapping. A reset-only armor adjustment would silently disagree
with later movement, Lua writeback and combat readers (R5b/R5c).

Recommendation for the next approved scope: one shared mobile equip/unequip
boundary, preserving C slot indices, with an explicit base/effective audit for
armor, points, modifiers, flags and Lua overrides; canonical removal must undo
exactly the same effects. Gate alignment before attach, route descriptor-owned
actor output through existing switched-body routing, and preserve room acts.
Implement light/name restoration on every movement/extraction path; port the
zero-stat priority, self damage and pestilence hunting through existing combat
and hunting paths, with fail-capable tests. Audit HitModifiers and every other
mobile slot reader together. No fake Player or deliberate divergence proposed.

That shared stat/combat/output class needs a design decision under the goal's
2026-10-01 shared-path stop. It is skipped here while the independent loop and
unknown-command cases finish, as the train rule requires. Retain the aggregate
blocked pending that scope; do not treat this as equipment parity.

## Case 1: E ownership, occupied slot and attribute boundary

C: db.c:2201-2219; handler.c:690-695,731-748; attribute bounds
handler.c:314-373. Successful attach records canonical LocEquippedMob using
the existing mobile C-slot representation. Occupied slots preserve the old
object and leave the new instance floating and counted. The E caller still
records success, enabling its conditional successor. The private locked
attribute computation is the existing AffectTotal computation, with no new
stat semantics; its public wrapper still takes the same mobile lock.

Tests: TestZoneResetEquipmentOwnershipAndOccupiedSlot,
TestZoneResetEquipmentAttributeBoundary, TestZoneResetEquipmentLoadGates.
They assert extraction detachment and attribute restoration, missing mobile,
negative/upper slot bounds, global cap and percent failure. The unchanged main
implementation fails the occupied-slot assertion. Ownership, refusal and
attribute mutations each produce assertion-failing 0/1/0 triples.

Other readers: canonical object extraction and movement use Location to detach
from mobile maps; R-mobile destruction, corpse/drop and Lua object lookups see
the same registered instance. Effective attribute readers keep the existing
mobile lock and cache. Combat/armor, light/name, anti-alignment output and
mobile slot consumers remain in the explicit shared remainder above. Player
slot numbering, player inventory and player equipment are untouched.

## Case 2: L and last_cmd matrix

C: db.c:2077-2104. Tests exercise negative/zero/one/two/five counts, nested
starts replacing C's single saved index/counter, conditional failed caps and
comments retaining prior success, unconditional failures/comments/end commands
clearing it, and initially skipped loop starts not changing control state.
No game code change is needed for these branches. Start count, end comparison,
unconditional reset and conditional skip each have assertion-failing 0/1/0
controls. No loop-stack redesign or malformed-table convenience exclusion.

Other readers: the reset-local counter and last_cmd have no external readers.
Every conditional M/O/G/E/P/D/R/unknown command uses the same existing precheck;
spawn counts and draw hooks assert effects rather than just control variables.

## Case 3: unknown command parsing and persistent disable

C: db.c:1575-1589,2085-2095,2278-2281. Unknown commands are reachable through
zone files: C parses three integers and preserves if_flag, then disables only
an actually executed command. Go previously zeroed those parsed fields and
never disabled the command. New parser and runtime tests fail on those two
assertions before the fix.

Each reset works on a private command copy, so an executed unknown becomes *
immediately inside a loop. World publishes a cloned live zone table for later
resets; old zone/parsed-world snapshots remain unchanged. Publication checks
the current command against the executed command, preserving an editor's
changed replacement. A skipped unknown stays enabled. The same C diagnostic
is logged once across loop iterations and later resets. No player-facing text
is added. Parser, same-pass disable, persistent disable and editor guard each
have assertion-failing 0/1/0 controls.

Other readers: GetZone/GetAllZones/SnapshotZone, zone clock, zreset and admin
reset all see the newly published definition. Existing ZEDIT clone/commit
behavior and disk filtering of * remain; no save-file mutation occurs.
Definition edits/reloads can install a new table, just as their normal paths
do. TestZoneResetUnknownConcurrentEditorAndReaders races resets, editor
replacement, old-zone snapshot reads and mobile effective-attribute reads.

## Locks

- World.ResetZone: zoneResetMu, brief World RLock for definition, release;
  execute with zoneResetMu held. Boot/manual/clock retain this serialization.
- Equipment attach: zoneResetMu -> World Lock -> Mob Lock. Only raw IDs,
  canonical constructors, map fields and affectTotalLocked run while held;
  no session/manager/lifecycle lock, callbacks or output. Release Mob then World.
- Unknown publication: zoneResetMu -> World Lock; clone/publish then release.
  Private loop table has no extra lock. Editor only holds World Lock and never
  asks for zoneResetMu. Readers release World before mobile getters.
- Huma single/all-zone reset (huma_completion.go:235,312) and legacy HTTP
  single/all-zone reset (handlers.go:1077,711) hold no manager, lifecycle,
  mobile or world lock across ResetZone. GetAllZones releases its read lock
  before iteration. Audit logging and HTTP output occur after ResetZone returns.
- No new lifecycle acquisition. No admin endpoint or reset scheduling change.

## Evidence and validation

Retained root:
~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-reset-equipment-proofs/.
Each case's normal gate directory contains individually checked fmt/build/vet/
test/lint-cache/lint/diff/depth/units/string-census logs. Controls are named by
case and fixed/reverted/restored; all counted reverted runs fail assertions.
The corrected conditional-skip mutation retains a read of last_cmd so its red
is an assertion rather than a compile failure; the initial invalid mutation
is not evidence. race.log holds ten repeated race runs of equipment, loops,
conditional, unknown parser and concurrent editor/reset/readers tests.

The one combined census is run at the committed train tip; its result and
retained directory will be recorded in the PR body. No automatic reset timing
or random draw implementation changes are made. Any census red still requires
investigation against paired C bytes rather than a repin or dropped seed.

Retained frontier: browser name routing, identity consumers, storedPlayer and
PlayerStoreEdit loaded-inventory ownership, legal-quit rented-object cleanup,
port-bind race and users Login@ wall-clock. Entry restrictions await D6.
Teleport combat recheck remains separate, after the reset matrix remainder.
