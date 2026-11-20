# Round 1 result

## Measurement

- Draw SHA: `8590800fe1d41b0c57daf606a9dbee97dbce963c` (DP-1596 merge).
- Sample seed: `20261120`; 100 functions, one sampled literal per function.
- Headline: **65 / 97 catchable mutants = 67.0%**. The uncorrected two-sided 95% Wilson interval is **57.2%–75.6%**. The four-look corrected one-sided Wilson lower bound (z = 2.50) is **54.4%**, below the 80% stop bar.
- All-sample count, including excluded rows: **65 / 100**.
- Classes: 95 `c-reachable` (65 killed, 30 survived); 2 `go-only-in-game` (0 killed, 2 survived); 3 `c-unreachable` (excluded).
- Survivor rows: 1, 2, 3, 10, 12, 14, 15, 18, 21, 23, 24, 26, 27, 28, 34, 36, 38, 41, 54, 55, 58, 59, 60, 61, 65, 72, 74, 76, 82, 88, 96, 97.
- Mob-side survivor audit: **17 / 32** (rows 1, 2, 10, 12, 14, 15, 23, 24, 26, 34, 36, 54, 58, 59, 65, 72, 76), counting mobile command handlers and NPC specials, including shop/postmaster outputs and the force notice to the mobile. This is just over half.

## Census evidence

- Control: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-mutloop-round1-control` — 2,381 scenarios, `CLEAN_AFTER_RECHECK`; draw parity 2,381, 0 divergent. The one infrastructure row was rechecked by the wrapper.
- All-mutants run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-mutloop-round1-mutant` — 2,381 scenarios, `NOT_CLEAN` (351 failed); draw parity 2,381, 0 divergent.
- The score counts only rows where the Go diff contains the sampled mutation and the C transcript prints the original line. Duplicate-literal attribution was resolved with singleton probes: rows 36 and 96 were CLEAN on the exact matching scenarios and are survivors; rows 39 and 80 were NOT_CLEAN on the matching scenarios and are kills. Rows 83, 92, and 99 also reproduced as kills in their singleton runs. All 32 survivors have a targeted singleton run with draw parity.
- Full per-row evidence, including class evidence, byte offset and mutation, verdict, transcript line and run directories, is in `probes.tsv`.

## Frozen predictions

1. **Held.** The bar is not met; the uncorrected interval's lower bound is below 80%.
2. **Failed.** The point estimate is 67.0%, below 80%.
3. **Failed.** There are 3 exclusions, below the predicted 15–25.
4. **Held.** 17 of 32 survivors are on mobile command or NPC-special paths, which is more than half.
5. **Pending repair audit.** No survivor has yet been confirmed and filed as a Go divergence; rows 28 and 34 remain open in `../DECISIONS.md` for Zach's DP-1596 follow-up.
6. **Pending Claude spot check.**
7. **Held so far.** 0 harness-blind rows identified.

## Follow-up

This round does not pass the corrected bar. Repair the 32 survivors in batches of about six. Keep rows 28 and 34 parked for Zach's ruling; record any verified Go divergence in a DP issue. The measurement remains fixed at this draw SHA and is not changed by repairs.
