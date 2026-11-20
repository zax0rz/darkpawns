# Round 2 classification corrections

- **Row 76:** the frozen class is `c-reachable`; the post-census correction is `c-unreachable after all`. C defines `npc_rescue` at `src/mobprog.c:435-448`, including the sampled room-output line, but a C-source search for `npc_rescue(` finds only the definition at line 435 and no caller. The Go implementation comment at `pkg/game/mobprogs.go:395` records the same dead-code status. The correction does not rewrite `sample.tsv`, the frozen class, or the denominator.
