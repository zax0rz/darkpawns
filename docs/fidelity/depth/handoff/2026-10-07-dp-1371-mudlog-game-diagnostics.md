# DP-1371 D7 game-diagnostic train

Stop tier: the spell-world consumer bridge and the RawKillVerb result field exceed the mudlog-only exception. Six sites repaired at their live boundaries; two retained frontiers below. Fresh main 9a0a0f343. R1/R3/R5e/R5g/R5h.

## Death-cry NOWHERE frontier (case 7)

`src/fight.c:506-516` logs BRF/31/file TRUE and places the body in C room index 0. However, `raw_kill` guards NOWHERE before any work (`534-539`), so this producer cannot be justified by simply calling raw_kill with an unplaced body. This matches Go's guard; it is not a demonstrated raw-kill divergence.

The other direct callers are `src/act.movement.c:293,298`. The rider has NOWHERE checks after entry and greet (`266-287`). A retained mount is moved to the destination before those callbacks (`193-203`); the mount pointer survives their execution. Lua teleport validates the destination (`src/scripts.c:1527-1536`), while extchar requests deferred extraction (`480-489`). This is not yet a complete proof that every nested callback preserves a valid mount room, and the default remains blocked. Do not inject an invalid descriptor/body and call it a valid-play oracle proof.

Next: finish the whole callback/retained-mount audit. If no legal path exists, add a fail-capable bounded caller proof and exclude this producer from valid play. If a legal path exists, demonstrate it in unchanged C first and design the body-placement boundary across combat/world death-cry readers, using the existing room-index conversion and body ownership. No second registry, guessed vnum 0, approximate log, or proposed divergence.

Retained C caller sweep: `dp-1371-mudlog-game-diagnostics-proofs/C-deathcry-sites.txt`.
