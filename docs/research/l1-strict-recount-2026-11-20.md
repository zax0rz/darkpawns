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
| mobprog.c:577 | settled-witnessed (mob-sound-mime-box-depth; see the hollow-retain note below) |
| constants.c:691 | open (L4: transient descriptor state, DP-2918) |
| constants.c:712, 714, 716 (3) | open (L4: identd socket / same-pass staging, DP-2918) |
| constants.c:713, 715 (2) | settled-unreachable (fall-through ident states held only within one ident_check pass; matches informative.tsv excluded rows) |
| mobact.c:252 | open (race-hate staging) |

**Open after finish-L1: 5 of the 20.** Exact open list: constants.c:691,
712, 714, 716 (four transient-descriptor rows, DP-2918 class), and
mobact.c:252 (race-hate staging). Settled: the other 15 (3 witnessed,
4 unreachable, 8 unit-exception).

## Corrections found while finishing

- **mob-sound-mime-emote's retained blocks are hollow.** The shipped mime
  (vnum 8079) carries MOB_AGGRESSIVE|MOB_RANDZON (80.mob act word 1048648 =
  ISNPC\|AGGRESSIVE\|RANDZON); spawn-mob enters through C's normal 'M' reset
  path, so RANDZON relocates the body at db.c:2132-2145 before the observer
  ever sees it, and the aggressive body removes the fresh one. No retained
  census dump anywhere contains the mime line. The new
  mob-sound-mime-box-depth scenario clears both flags on the fixture mob and
  witnesses the line with the observer co-located
  (glm-l1-final-mime3/census-dump/mob-sound-mime-box-depth.txt:16,20;
  mutant red glm-l1-final-mut-mime, parity MATCH). The existing manifest row
  `mob.mp-sound-emote-line` (mobprog.tsv) keeps its green status on the
  emote-rendering claim but its retained proof does not contain the case
  bytes; flagged for Zach.
- **constants.c:713/715 are same-pass ident states.** CON_IDCONED and
  CON_IDREAD are fall-through assignments inside ident_check's switch and are
  left within the same pass; the waiting states are CON_IDCONING (712) and
  CON_IDREADING (714), which need an in-flight ident socket (DP-2918). This
  aligns the finish TSV with the already-excluded manifest rows.

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
