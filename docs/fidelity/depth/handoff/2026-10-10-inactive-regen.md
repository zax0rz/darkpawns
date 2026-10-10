# 2026-10-10 inactive regeneration handoff

## Result

`PointUpdate` now mirrors `src/limits.c:494-501`: for players at or above
`POS_STUNNED`, `PRF_INACTIVE` skips hit, mana, and movement regeneration.
Poison and cutthroat damage remain after that gate, matching
`src/limits.c:502-506`. Condition decay already skips inactive players at
`src/limits.c:476-480` and continues to use the same locked preference-bit
snapshot.

The gain implementations were audited in `pkg/game/limits_gain.go` against
`src/limits.c:59-253`. `HitGain`, `ManaGain`, and `MoveGain` make no RNG or
time-seam calls, so skipping their invocation consumes no draws. C walks character_list (players and NPCs); NPCs do not carry player
preference flags. Go's separate NPC loop remains unchanged.

## Proof

`TestPointUpdateInactiveRegen` covers four direct state outcomes:

- Inactive HP, mana, and movement stay unchanged. This is mutation-sensitive:
  removing the gate changes these values and fails the assertion.
- Active HP, mana, and movement all increase from a fixture with positive gains.
- Poison still reduces an inactive player's HP by 10.
- Cutthroat still reduces an inactive player's HP by 13.

The fixture's full, thirst, position, and class make the exact gains auditable:
with no affect, active hit/mana/move gains are 12/42/25. With poison, hit gain
is 3 before the 10 damage; with cutthroat, hit gain is 3 before the 13 damage.
Thus removing the gate makes the inactive health controls end at 93 (poison)
and 90 (cutthroat), instead of 90 and 87. The mutation script requires all
three inactive assertions (the all-vitals no-regen assertion plus both damage
assertions) to fail under gate removal. The active case correctly stays green
under that mutation because active players already pass the C gate. A second
mutation inverts the condition and requires the active-regeneration assertion
to fail while the inactive no-regen case passes. Both mutations compile and
must fail on their named assertions; the script restores source in a `finally`
block and requires a final green run.

## Readers and locks

`PointUpdate` is driven by the hourly game-loop callback
(`cmd/server/main.go:611-613`) and can also be invoked by the immortal `tick`
command (`pkg/session/wiz_system.go:739`). It snapshots the world player list
under `w.mu.RLock`, then reads `Position` and `Flags` together under
`p.mu.RLock`. Resource values and poison/cutthroat flags are read under a
separate `p.mu.RLock`; each resource write uses `p.mu.Lock`. The player lock is
not held while calling a gain function or applying damage.

Stored vitals are read through `Player.VitalsSnapshot` by display/prompt,
GMCP, observation, and agent-variable paths, and by persistence (`pkg/game/save.go`).
Wizard stat output separately calls the gain functions in
`pkg/session/wiz_stats.go`; it reports their rate and does not mutate stored
vitals. Commands such as spell casting read and spend mana through the locked
player accessors. The change affects only the values written by the point
update; all readers continue to use the existing accessors and locks.

## Validation status

See `2026-10-10-inactive-regen.controls.py` for baseline green, gate-removal
and gate-inversion assertion failures, source restoration, and final green.
No census was started. The most
recent `scripts/census.sh status` reported a completed clean combined census
from another worktree; there was no running census before focused validation.

Sol independently reviewed and integrated this patch. Full fmt/build/vet/tests/game tests/lint/fidelity-depth/fidelity-units/string-census and focused race gates pass (1467/1467 claims). Controls independently rerun and pass; combined census remains pending.

Gate/control logs retained at ~/Archives/darkpawns/oracle-runs/2026-10-10/inactive-prompt-gates/.
