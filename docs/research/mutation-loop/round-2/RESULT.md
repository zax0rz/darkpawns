# Round 2 result

## Measurement

- Draw SHA: `2d5166cf0de2023e4eefa7af9272d58219feb6bc`.
- Mutant commit: `836c1a4c867ca34e1ea197f7f866415441939850`, with the draw SHA as its parent.
- Sample seed: `20261121`; 100 functions, one sampled message literal per function, from a frozen population of 388 functions / 1,926 literals.
- Headline: **61 / 94 catchable mutants = 64.9%**. The uncorrected two-sided 95% Wilson interval is **54.8%–73.8%**. The four-look corrected one-sided Wilson lower bound (z = 2.50) is **52.0%**, below the 80% stop bar.
- All-sample count, including excluded rows: **61 / 100**.
- Frozen classes: 88 `c-reachable` (61 killed, 27 survived); 6 `go-only-in-game` (0 killed, 6 survived); 6 `c-unreachable` (excluded). There are no `go-only-surface`, `ask-zach`, or `harness-blind` rows.
- Survivor rows: 3, 5, 6, 8, 9, 11, 15, 19, 20, 22, 28, 35, 38, 39, 41, 43, 44, 51, 54, 57, 59, 60, 62, 69, 70, 71, 76, 77, 81, 82, 84, 88, 98.

## Census evidence

- Control: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-mutloop-round2-control` — 2,400 scenarios, `CLEAN_AFTER_RECHECK`; draw parity 2,400, 0 divergent. Summary: passed=2,393, expected=4, infra=2, unstable=1.
- All-mutants run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-mutloop-round2-mutant` — 2,400 scenarios, `NOT_CLEAN` (275 failed); draw parity 2,400, 0 divergent. Summary: passed=2,120, expected=4, failed=275, unstable=1.
- Both summaries report `probe_setup_refusal_scenarios=0` and `probe_setup_refusal_steps=0`; no refusal scenarios need listing.
- The control and mutant censuses were run at the frozen draw SHA, before `enforceProbeSetupRefusals` was enabled. Per Zach's follow-up instruction, the enforcement flag was not a prerequisite because both retained summaries report zero refusals. The result branch now merges current `origin/main` `89f0feca578018b99c841ea16a6de42efb6cd363`, where `cmd/dp-oracle-diff/main.go` sets `enforceProbeSetupRefusals = true`; the measurement and denominator remain those of the draw SHA. Current main also includes DP-1607 guard removals after the draw, so this result describes the earlier frozen code as required.
- Every kill requires the sampled Go mutation in the diff and the original C transcript line in that scenario. Full per-row offsets, class evidence, witness lines, and run paths are in `probes.tsv`.
- All 33 survivors were checked in isolation when a covering scenario existed: 27 singleton runs were `CLEAN`; six rows had no covering scenario. Seven additional singleton runs reproduced kills to settle attribution (rows 0, 18, 31, 37, 74, 78, 92). Row 0 is therefore a kill: `spec-proc-butler` prints the C line `The Servant gets a loaf of bread.` while Go drops the subject. Row 44 is a survivor: its singleton was `CLEAN`, and the matching `Huh?!?` output is present in both engines; the full-run clan-status witness belongs to row 92.
- Row 76's frozen class remains `c-reachable`, but its post-census audit finds the C function has no call site. This correction does not rewrite the frozen class or denominator; see `CLASSIFICATION-CORRECTIONS.md`.
- Row 28's frozen class also remains unchanged in the sampled TSV; Zach's DP-1610 audit reclassifies its `doClanQuit` guard as `go-only-in-game` because C `do_clan_quit` reaches an R1a out-of-bounds write before printing different bytes. It leaves the repair list; the frozen mutation result and 61/94 denominator do not change.

## Follow-up

The corrected bar is not met. Repair the catchable survivors in batches of about six; keep the frozen measurement unchanged. The six `go-only-in-game` rows remain denominator misses under the loop method.

## Decisions for Zach

DP-1610 asks Zach to rule on the R1a handling of C `do_clan_quit`'s `clan_num == -1` write. Row 28 is reclassified as `go-only-in-game` and omitted from repairs; no `doClanQuit` code change is included before that ruling. Other open decisions are listed in [`../DECISIONS.md`](../DECISIONS.md).

## For Claude

Review the frozen classifications, kill attribution, and the current-main dependency above. **ROUND 2 READY FOR CLAUDE**.
