# APPLY_SPELL removal quirk — vehicle post-mortem and blocked arm — 2026-10-29 (GLM)

This note finishes the round the 2026-10-29 equipment handoff deferred: C's
APPLY_SPELL *removal* arm and what the port does with it. No game code changed.
Every C citation below was re-read in `src/` this round (R5g).

## The quirk

`src/handler.c:240-270` implements APPLY_SPELL inside `aff_apply_modify`:
NPCs are skipped; a positive modifier sets the named AFF bit in the persistent
`AFF_FLAGS`, a negative one clears it, and a **timed affect whose bitvector
equals the modifier suppresses the operation** (`src/handler.c:249-266`). The
suppression check is the only provider check C performs — the scan sees
`ch->affected` records, nothing else.

The consequence the round was asked to document: *removing an item clears its
affect bit unless a timed affect provides it* — so the bit is cleared even
when something else, such as a real nightbreed form, had set it
independently. A real werewolf or vampire (`transform` sets the bit with no
timed record, `src/act.other.c:1441`, `:1503`) who wears and then removes an
item granting their form bit loses the form with the item
(`src/structs.h:337-338`: AFF_WEREWOLF 27, AFF_VAMPIRE 28).

## Why the cape vehicle was vacuous — corrected diagnosis

The vampiric cape #7901 (`lib/world/obj/79.obj`) applies `A 29 19` —
APPLY_SPELL 19 = AFF_HIDE (`src/structs.h:329`). The earlier round attributed
its vacuous demo to zone 79's resets having consumed the cape's max-existing
count. This round found a second, sufficient cause: **the cape is
ITEM_INVISIBLE**. Its extra-flags field is 1106016, whose bit 5 is
`ITEM_INVISIBLE` (`src/structs.h:473`), and `CAN_SEE_OBJ`
(`src/utils.h:536`) refuses it to everyone without AFF_DETECT_INVIS — the
wearer included.

Observed directly on the reference oracle: an implementor's `load obj 7901`
succeeds (`act.wizard.c:1322-1371` puts the object in inventory, cost 3500
passes the level-31 gate at `:1336-1360`), and the creator then reports
`You are carrying: Nothing.` with `Carried: weight: 5, items: 1` — the cape
is there and unseeable; `get cape`, `wear cape` and `stat cape` all refuse by
visibility. Retained: `oracle-runs/2026-10-29/glm-apply-spell-removal/cape-invis-manual-boot.log`
and `-server.log`. No APPLY_SPELL 19 item in the world is visible (the scan
below lists the inventory), so hide has no reachable item vehicle.

## The vehicle that works: robes #4331

A full scan of `lib/world/obj/*.obj` A-records at location 29 found 62
APPLY_SPELL records; the complete list with visibility is retained in this
round's evidence note in the PR description. Exactly one visible item grants a
nightbreed bit: object 4331, the dragon Emperor robes (`lib/world/obj/43.obj`),
which applies both `A 29 28` (vampire) and `A 29 27` (werewolf) alongside stat
modifiers. The room-list fangs/form strings are
`src/act.informative.c:420-427` (`look <char>`) and the self-report lines are
`src/act.informative.c:1374-1378` (`score`).

`cmd/dp-oracle-diff/scenarios/egon-equipment-apply-spell-removal.txt` drives
the full wear/remove cycle on the robes with both observables at each state:
score reports `You're a lycanthrope!` / `You're a vampire!` and the mortal
observer's `look Robebearer` reports the fangs line while worn; both
disappear after removal and return on re-wear. **Both engines match through
every arm** (seed 1, retained at
`oracle-runs/2026-10-29/glm-apply-spell-removal/robes-observables-run8.log`):
the #2216 equipment overlay reproduces C's persistent-bit set on equip and
clear on unequip for everything a scenario can reach. The plain removal arms
are *not* divergent post-#2216 — that is a negative result, and the manifest
records it as live proof.

## The arm that stays blocked: the real nightbreed wipe

The quirk's bite needs a real nightbreed: `transform` with the persistent bit
set and no timed record, then an unequip of a bit-granting item. C wipes the
form bit (the timed-affect scan finds no provider, `src/handler.c:259-266`);
the Go overlay recomputes from the worn set and leaves the persistent bit
alone — a predicted divergence, player-visible through the same form strings
and mechanically through the nightbreed state consumers.

No harness-reachable path creates that state. `transform` refuses without
`PLR_VAMPIRE`/`PLR_WEREWOLF` (`src/act.other.c:1417`), and the reference
oracle's immortal `set` refuses the flag on a PC (`Can't set that!`,
observed on the retained manual boot). The only granting path in the game is
the werewolf mob's bite chain (`src/spec_procs3.c:427-453`), which is
combat-RNG-dependent. A bite-chain vehicle is possible but is its own round.

The manifest row `equipment.apply-spell.nightbreed-unequip-wipe` records this
as `blocked`, per the depth-testing rules: a real proof gap, not an exclusion.

## Deliverables this round

- `cmd/dp-oracle-diff/scenarios/egon-equipment-apply-spell-removal.txt` — the
  robes arms, green on both engines, with the observables above.
- Manifest rows in `docs/fidelity/depth/equipment.tsv`:
  `equipment.apply-spell.removal-arms` (oracle-green) and
  `equipment.apply-spell.nightbreed-unequip-wipe` (blocked).
- Evidence: `oracle-runs/2026-10-29/glm-apply-spell-removal/` — the retained
  green run, the earlier cape-refusal runs, and the manual boot that
  demonstrates the ITEM_INVISIBLE cape.
- This note. No fix, no rulebook edit.

## Decision for Zach

Whether to commission the bite-chain vehicle (werewolf mob 5510 bites the
actor, then the robes unequip) to turn the blocked arm red, or to ratify the
blocked row as the standing record. The predicted C-vs-Go divergence is stated
above and has not been observed on live engines.
