# Round 3 result

## Measurement

- Draw SHA: `cc94e6d25b1ea099421a12a0a88c6a08d20d7ae0`.
- Mutant commit: `b4bdebc407e8db23c5ebf887373b5d5f1392aaa9`, parented directly to the draw SHA.
- Sample seed: `20261122`; 100 functions, one sampled message literal per function, from 285 filtered functions and 1,406 literals (654 raw functions and 3,239 literals before filtering).
- Headline: **63 / 98 catchable mutants = 64.3%**. The uncorrected two-sided 95% Wilson interval is **54.4%–73.1%**. The four-look corrected one-sided Wilson lower bound (z = 2.50) is **51.7%**, below the 80% stop bar.
- All-sample count, including excluded rows: **63 / 100**.
- Frozen classes: 96 `c-reachable` (63 killed, 33 survived); 2 `go-only-in-game` (0 killed, 2 survived); 2 `c-unreachable` (excluded). There are no `go-only-surface`, `ask-zach`, or `harness-blind` rows.
- Survivor rows: 4, 10, 14, 15, 19, 20, 24, 25, 27, 28, 30, 31, 34, 38, 39, 41, 46, 50, 53, 54, 55, 59, 64, 65, 67, 72, 78, 80, 84, 88, 91, 94, 95, 96, 98. Each available covering scenario was run alone with only its mutant; row 14 has no covering scenario after the call-path correction. The other 34 singleton runs were CLEAN.

## Census evidence

- Control: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-round3-control` — 2,433 scenarios, `CLEAN`; passed=2,428, expected=4, infra=0, timed_out=0, unstable=1. Draw parity: 2,433, 0 divergent.
- All-mutants run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-round3-mutant-100` — 2,433 scenarios, `NOT_CLEAN`; passed=2,044, expected=4, failed=379, infra=5, timed_out=0, unstable=1. Draw parity: 2,433, 0 divergent.
- Both full-run summaries report `probe_setup_refusal_scenarios=0` and `probe_setup_refusal_steps=0`. No refusal scenarios need listing. All 35 singleton summaries also report zero setup refusals and zero draw divergence.
- The mutant full run retained five infrastructure rows after the wrapper's recheck: `advance-depth`, `l1-constants-4-depth`, `observation-commands-basic`, `shops-tattoo1-offer3`, and `shops-tattoo1-offer4`. The isolated row 26 run at `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-round3-isolate-r26` is `NOT_CLEAN`: C prints `You rise a level!` and the Go diff prints `Qou rise a level!`. This confirms row 26 as a kill despite the full run classifying `advance-depth` as infrastructure.
- The other 34 singleton runs were CLEAN with draw parity and are recorded by row in `probes.tsv`; they leave the survivors unchanged. Every kill has both the original C transcript line and the sampled mutation in the Go diff. Full-run hits and the isolated row 26 hit are distinguished in the probe ledger.
- Row 14's frozen class remains `go-only-in-game`, but a post-census call-path correction finds the special unassigned in C and absent from world assignments in the draw tree. The frozen class and denominator remain unchanged; it has no covering census scenario. See `CLASSIFICATION-CORRECTIONS.md`.
- This result branch is rebased on current `origin/main` `b2e884aca1c9947cd9b51db7ca41de4982e01799`. The measurement itself remains pinned to draw SHA `cc94e6d25b1ea099421a12a0a88c6a08d20d7ae0`, before the intervening merges #2916, #2917, and #2919; the result describes the frozen draw code. In particular, #2919 subsequently changed Go's `turn_undead` handler registration, while the object special remains unassigned in world data. No census result was recomputed against the later main code.

## Follow-up

The corrected bar is not met. At this checkpoint, no repair batch is included; the frozen survivor list is the input for review and subsequent repair work. The Round 3 measurement remains 63/98 regardless of later code changes.

## Decisions for Zach

Round 3 row 50 was frozen as `go-only-in-game` at the measurement commit. Zach later ruled to remove the unreachable `w.Bans == nil` fallbacks, and PR #2934 merged that change; the frozen measurement and denominator are unchanged.

## For Claude

Review the frozen classes, kill attribution, row 14 correction, census evidence, and the current-main dependency above. **ROUND 3 READY FOR CLAUDE**.
