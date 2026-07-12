# Brief: DP-1046 mob combat redirects

## Issue

`damage()` in `src/fight.c:1370-1440` has three mob-initiated control-flow redirects that were not on the live Go melee path:

- jail guards 8102/8103 subdue eligible PCs and send them to jail instead of killing them
- NPCs about to damage charmed NPC followers can retarget to the follower's master
- level 21+ NPCs can switch targets to another room occupant currently fighting them

The partial `combat.TakeDamage` port was not used by `CombatEngine.processCombatPair`, so live mob melee skipped these behaviors.

## Fix

- Added narrow combat callbacks for room combatant snapshots, follower/master lookup, and jail-guard subdue side effects.
- Added `CombatEngine.applyMobCombatRedirects` before melee damage is calculated.
- Wired game-layer jail subdue behavior: set victim HP to 1, clear mount state, clear guard memory/hunting for the victim, move victim to room 8118, show the room, and set jail timer to `max(2, level/2)`.
- Added regression tests for all three redirects.

## Fidelity notes

- The high-level NPC switch follows the cited C source: iterate room occupants and switch on `!number(0,80)` when the occupant is fighting the NPC. The Linear title says "highest-damage attacker", but the cited block does not track damage.
- The jail intercept uses room vnum 8118 because C calls `char_to_room(victim, real_room(8118))`.
- Redirects run in the per-round damage path, not `StartCombat`, preserving the shared combat enrollment funnel.

## Verification

- `go test ./pkg/combat`
- `go test ./pkg/game/...`
