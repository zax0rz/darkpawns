# Strict L1 recount — 2026-11-20 (finish-L1)

## Method

Redo of the 2026-11-19 strict recount (PF-132): classify every C segment the
coverage tool reports uncovered, joined with the current depth manifests and
the batch TSVs. This pass joins FOUR batch TSVs (batch one #2861, batch two
#2891, batch three #2914, finish-L1 this PR) instead of the two the 11-19
recount had, and uses Claude's combined census on main 43cbca21b
(`/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/claude-integ-1120-full`)
for the coverage join.

**Status at writing:** Claude's combined census was still running
(36/36 workers active, no summary yet), so the coverage-based recount below
is recorded as PENDING and the row-level recount is taken from the four batch
TSVs plus this PR's new evidence. When the run finishes, rerun
`cmd/dp-census-coverage` on its `census-dump` and re-run
`docs/research/l1-strict-2026-11-19.md`'s join to confirm.

## Row-level recount (the 20 rows the finish-L1 brief carried)

| Rows | Ending after finish-L1 |
|---|---|
| new_cmds.c:2088 | settled-witnessed (compare-band-depth@2) |
| spells.c:191 | settled-unit (unit exception, DP-1202 class; C-cited test + mutant-red revert proof) |
| dream.c:76,78,83,85x2,91,93 (7) | settled-unit (unit exception; oracle cannot age the real-time stamp) |
| interpreter.c:2227 x2 | settled-unreachable (free_rent=YES + static world ⇒ Crash_load return-2 arms cannot fire) |
| zedit.c:677 | settled-unreachable (reset disables non-MOEPDRL letters before zedit's display; fixture + empty-list evidence) |
| limits.c:629 | settled-witnessed (batch three, corpse-decay-depth@2) |
| constants.c:691 | open (L4: transient descriptor state) |
| constants.c:712-716 (5) | open (L4: identd socket states) |
| mobact.c:252 | open (race-hate staging) |
| mobprog.c:577 | open (RANDZON locator) |

**Open after finish-L1: 8 of the 20.** Exact open list: constants.c:691,
712, 713, 714, 715, 716 (six transient-descriptor rows, DP-2918 class),
mobact.c:252 (race-hate staging), mobprog.c:577 (RANDZON locator).
Settled: the other 12 (2 witnessed, 2 unreachable, 8 unit-exception).

## Coverage-based recount

PENDING the claude-integ-1120-full run. When its census-dump lands:

```bash
GOEXPERIMENT=nojsonv2 go run ./cmd/dp-census-coverage \
  --dump /home/zach/Archives/darkpawns/oracle-runs/2026-11-20/claude-integ-1120-full/census-dump
```

and re-run the 11-19 classification (uncovered segments joined with the
current top-level manifests and the four batch TSVs). Expected movements vs
the 11-19 baseline: the four say_spell segments covered (DP-2898's matcher
fix), plus every segment witnessed by batches two/three/final.
