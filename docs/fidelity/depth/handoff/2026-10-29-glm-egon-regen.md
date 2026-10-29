# Item regen verification and the mobile gain tail — 2026-10-29 (GLM)

## Correction to the #2216 audit conclusion

The #2216 equipment handoff listed APPLY_HIT_REGEN / APPLY_MANA_REGEN /
APPLY_MOVE_REGEN as a "potential fidelity divergence" because
`affect_modify_ar`'s cases 26-28 are empty (`src/handler.c:175-180`). That
conclusion was wrong about the game: the regen consumers are the gain
functions, not affect_modify. `mana_gain` adds a positive item modifier only
while sleeping and a negative one in any position, before the class
multipliers (`src/limits.c:86-104`); `hit_gain` adds any item modifier inside
its sleeping case, after the half bonus (`src/limits.c:149-172`); `move_gain`
adds any item modifier inside its sleeping case, before the half bonus
(`src/limits.c:217-235`). The Go port already implements all of this
faithfully (`pkg/game/limits_gain.go`, `sumEquipAffect` in
`pkg/game/limits_misc.go`), and **nothing was removed**. This round added the
missing proof instead:

- `regen-equipment-stat` — object 1317 (+3 to all three locations) worn:
  standing shows the item suppressed (stat gain +20/+14/+20), sleeping shows
  hit +33 (half bonus then item), mana +31 (double then item, warrior), move
  +34 (item then half bonus). Green, seeds 1-3, draw parity MATCH.
- `regen-equipment-tick` — explicit score deltas across 700-pulse DP_CLOCK
  blocks (one game hour = SECS_PER_MUD_HOUR 63 × PASSES_PER_SEC 10): two
  sleeping hours apply +33/+31/+34, the second hour caps move at max via C's
  MIN, and an awake hour drops back to the no-item gains. Green, seeds 1-3,
  draw parity MATCH.

## The one real mismatch found and fixed

C runs the gain tail after the PC/NPC branch for **every** character: each of
AFF_POISON / AFF_FLAMING / AFF_CUTTHROAT quarters the gain and
ROOM_REGENROOM adds half again (`src/limits.c:107-125,176-192`). Go's
`MobHitGain` and `ManaGainNPC` returned the raw branch value, so a mobile
carrying one of those affects, or standing in a regen room, showed and
regenerated at the wrong rate — directly observable in `stat <mob>`'s gain
fields (`src/act.wizard.c:802-821`). Mobiles carry conditions of −1
(`src/db.c:1098-1101`), so the hunger/thirst quarter never applies to them;
`move_gain` returns early for NPCs (`src/limits.c:210-212`), so its NPC gain
correctly stays raw.

Both functions are now `(w *World)` methods applying the shared tail; the
regen tick and the stat display call the same implementation. Proof:
`regen-mob-regenroom-stat` (trainee in a REGENROOM room, `Hit p.: [25/25+3]`,
move still +1 proving the early return) is green here and **red on
origin/main** (+2 versus C's +3; retained under
`oracle-runs/2026-10-29/glm-egon-regen/gates/`), plus
`TestMobGainSharedTail` covering poison and regenroom arithmetic.

## Verified-correct boundaries (no change)

- Player `ManaGain`/`HitGain`/`MoveGain`: base, veteran, KK_JIN/KK_ZHEN,
  position switches, item gates and ordering, class multipliers (including
  the magus/avatar else-chain that only matters for single classes), poison/
  flaming/cutthroat quarters, hunger/thirst, regenroom — each step matches C,
  now pinned by the two equipment scenarios above plus the existing
  regen-position-gains rows.
- NPC mana regen: C's tick line exists (`src/limits.c:500`) but no C code
  path ever decrements a mobile's mana (grep over `src/` finds no
  `GET_MANA` consumer for NPCs), so `mana < max` never holds for a mobile and
  the line is unobservable; Go's tick omits it, recorded as a boundary.
- `ManaGainNPC`'s regenroom tail is now applied for stat parity even though
  the tick line is unobservable, because `stat <mob>` prints mana_gain.

## Census and gates

Targeted census with draw parity and per-gate logs are linked in the PR
description; `make fidelity-depth` passes with the new rows registered.
