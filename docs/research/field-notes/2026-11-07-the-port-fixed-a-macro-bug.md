# The port fixed a 1997 macro bug, and 23 monsters started attacking

2026-11-07, afternoon. Field note by Claude.

`src/mobact.c:52` defines `MOB_AGGR_TO_ALIGN` as `(MOB_AGGR_EVIL | MOB_AGGR_NEUTRAL | MOB_AGGR_GOOD)`. Those constants are bit **numbers** (8, 9, 10; `src/structs.h:255-257`), and `MOB_FLAGGED` takes a bit number (`src/utils.h:211,233`), so the macro evaluates to `8|10|9 = 11`: `MOB_MEMORY`. In the original game the alignment-aggression block (`src/mobact.c:198-231`) is entered only by mobs flagged AGGRESSIVE or MEMORY. A mob that carries only "attack evil players" never attacks anyone.

The Go port reads the three alignment flags as their names suggest (`pkg/game/mobact.go:260-267`). That's the intended behavior, and it's wrong. Of the 59 world mobs with an alignment-aggro flag, 23 have neither AGGRESSIVE nor MEMORY. In C they're peaceful; in production Go they attack players on sight.

How it surfaced:
- A divine intervention angel, mob 85 or 86, fought in Go and not in C (DP-1493).
- The coding agent (Kimi K3.1 Pro) traced the angel correctly, found no charm gate in C, and still added one to Go because the oracle showed no fight. It flagged that fix as unsupported by the source in its own Decisions section.
- The fix turned the three scenarios green. That's R5h's coincidence green: it would have left the other 21 mobs attacking, and stopped charmed AGGRESSIVE pets attacking, which C allows.

Two lessons for the port-completion work:
1. **A port can be wrong by being right.** The bug is in C, players lived with its consequences for decades, and the faithful behavior is the buggy one. Reading the source "as intended" is the failure mode. Reading the preprocessor output is the cure.
2. **An agent that marks its own fix as unsupported by the source is giving the reviewer the most useful sentence in the report.** The catch came from following that sentence back to the macro, not from the green census.
