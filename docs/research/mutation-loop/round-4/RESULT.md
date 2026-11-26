# Round 4 result

## Measurement

- Draw SHA: `f3d46e951e8c73f154dfecf38770dd4ae130ada2`.
- Mutant commit: `4e6e24cba8da5b6bd186491d0c50bed348b9af4d`, parented directly to the draw SHA and never pushed.
- Sample seed: `20261123`; 100 functions, one sampled message literal per function, from 187 filtered functions and 976 literals (648 raw functions and 3,182 literals before filtering).
- Of the 100 sampled functions, 99 occur in the frozen Round 4 sweep pool and 1 is new (row 78). This does not split the headline.
- Headline: **88 / 93 catchable mutants = 94.6%**. The uncorrected two-sided 95% Wilson interval is **88.0%–97.7%**. The four-look corrected one-sided Wilson lower bound (z = 2.50) is **85.5%**, above the 80% stop bar.
- All-sample count, including rows excluded from the denominator: **88 / 100**.
- Frozen classes: 93 `c-reachable` (88 killed, 5 survived); 7 `c-unreachable` (excluded). There are no `go-only-in-game`, `go-only-surface`, `harness-blind`, or `ask-zach` rows.
- Survivor rows: **0, 6, 27, 35, 80**. Each had a one-mutant targeted census with draw parity. Rows 0, 6, 27, 35, and 80 were CLEAN; row 23's isolated run reproduced its kill.

The sample classes and denominator remain those committed before the census. A post-census C call-path audit found that rows 27 and 80 do not reach their sampled C lines through the registered command path. Those corrections are recorded in `CLASSIFICATION-CORRECTIONS.md`; the pre-registered classes and denominator remain frozen, so both rows stay in the measurement as non-kills. Rows 0 and 6 likewise have clean isolated runs where the sampled Go branch is not reached by the exercised command, while row 35's error branch is not reached because object prototype 20 exists in the fixture. These are retained as survivors, not silently removed from the denominator.

## Census evidence

- Control: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-round4-control` — 2,524 scenarios, `CLEAN`; passed=2,519, expected=4, infra=0, timed_out=0, unstable=1. Draw parity: 2,524 scenarios, 0 divergent.
- All-mutants run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-round4-mutant-100` — 2,524 scenarios, `NOT_CLEAN`; passed=2,292, expected=3, failed=227, infra=1, timed_out=0, unstable=1. Draw parity: 2,524 scenarios, 0 divergent.
- Both full-run summaries report `probe_setup_refusal_scenarios=0` and `probe_setup_refusal_steps=0`.
- The mutant run retained `l1-constants-4-depth` as INFRA after the wrapper's second attempt. It is outside the sampled function set and is not an unknown sampled row. The run retains both attempts.
- The ordinary full-census runs retain per-attempt `c-draws.log` and `go-draws.log` files under `initial-logs`; their summary draw tallies are zero-divergent. The ordinary full wrapper does not produce the aggregate `draw-results.tsv` emitted by combined runs.

## Isolated probes

| Row | Scenario | Result | Run |
|---:|---|---|---|
| 0 | `cast-sitting-last-survivor-depth`, `lua-cast-spell-gates-depth` | CLEAN; no mutated Go default-branch output | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-round4-row-00-mutant` |
| 6 | `movement-directions` | CLEAN; normal no-exit path uses a separate literal | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-round4-row-06-mutant` |
| 23 | `use-basic` | NOT_CLEAN; C prints `You can't seem to figure out how to use it.` and Go prints `Qou can't seem to figure out how to use it.` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-round4-row-23-mutant` |
| 27 | `position-fighting` | CLEAN; the interpreter gate prevents the sampled handler line | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-round4-row-27-mutant` |
| 35 | `switched-body-door-depth` | CLEAN; puddle creation succeeds | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-round4-row-35-mutant` |
| 80 | `clan-callsite-authorization-gates-depth` | CLEAN; the empty-value prompt is not reached | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-round4-row-80-mutant` |

Every isolated summary reports zero draw divergence and zero probe setup refusals. Every kill has a retained paired block with the original C line and the mutated Go bytes in `probes.tsv`'s `killed_by` scenario.

## Follow-up

The corrected bar is met. No later repair changes this frozen Round 4 number. The survivors remain the input for the mutation-loop repair work after review.

## Decisions for Zach

No ruling is requested. The NULL `%s` and overlapping-`sprintf` cases follow Zach's 2026-11-25 decisions recorded before the census. The post-census rows 27 and 80 call-path corrections preserve the frozen class and denominator.

## For Claude

Review the frozen classes, the two post-census call-path corrections, and all 88 paired kill witnesses. This report branch includes current `origin/main` `31146bf1d4307a6055c9e89fc9931c15b0ac99a2`; the measurement itself remains pinned to draw SHA `f3d46e951e8c73f154dfecf38770dd4ae130ada2`.
