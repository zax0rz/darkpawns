# Race-hate mobile dispatch — approved witness — 2026-11-11

## C path and fixture boundary

`src/db.c:549-594,653-665` loads the mob files named by C's mob index. The
index contains `55.mob` but not `85.mob`: #5502 is an indexed Race-18 mobile in
`55.mob`, while #8552 is a Race-18 prototype in unindexed `85.mob`. The M reset
in `85.zon` cannot make #8552 load. These source and world paths were checked
in this session (R5e/R5g).

The shared path is vnum-independent. `src/handler.c:205-238` fills and clears a
player's `APPLY_RACE_HATE` slots as equipment changes; `src/mobact.c:236-258`
scans room occupants and calls `hit` when a non-shopkeeper mobile's race
matches one of the player's slots. Stock dagger #5200 applies race hates 18
and 20. The scenario loads indexed #5502 with the real `load mob` command and
has a peer receive and wield #5200 through real commands.

The scenario's `onpulse_pc` fixture runs before the race-hate scan
(`src/mobact.c:181-195`). It sets the wearer to POS_STUNNED and gives the mobile
a violence timer, then returns false so mobile activity continues into the
race-hate branch. This keeps the target output focused on the dispatch and
avoids a player counterattack round.

## Census and revert proof

The targeted seed-1 census is CLEAN with no draw divergence:
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-race-hate-isolated-dispatch`.
Its retained transcript shows both engines delivering the indexed Taltos's
attack to the wearer.

R5h revert proof: changing one byte in Go from
`vict.RaceHates[i]` to `vict.RaceHates[1]` makes the scenario miss the Race-18
slot; C's two attack lines are absent from Go and the census is NOT_CLEAN at
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-race-hate-slot-revert-proof`.
The source was restored immediately after the census snapshot.

## Ruling

Zach approved the generic `APPLY_RACE_HATE` witness scope on 2026-11-11. The
depth row now claims the indexed #5502 witness at seed 1. #8552 remains
unloaded by C and is not claimed. This is the same shared dispatch class, with
the specific #8552 fixture boundary recorded above (R1/R2/R4/R5e/R5g/R5h).
