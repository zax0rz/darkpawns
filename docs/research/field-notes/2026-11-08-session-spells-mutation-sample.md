# Session and spells mutation sample

> **Correction added 2026-11-08:** The first filed version mislabeled the 36 uncaught probes as infrastructure and reported selected 100% rates. The census evidence shows these were baseline-red rows. The corrected account and preserved correction history appear below.

## Research track and question

**Track:** legacy-port fidelity. **Question:** how often does the full differential census detect a preregistered one-byte mutation in randomly sampled player-output functions from `pkg/session` and `pkg/spells`, and how well do predicted scenarios identify the mutation’s coverage?

## Time window and sources

Work ran 2026-11-06 through 2026-11-08 in the `luna/round3-mutation-sample` worktree, based on Part 1 commit `aa867bf4dfa25c0fcafe337c11071d56ed7aee40`. Sources were the repository’s frozen populations, prediction and probe TSVs, the `scripts/census.sh` manifests/results/transcripts under `~/Archives/darkpawns/oracle-runs/2026-11-06` through `2026-11-08`, and the cited C source under `src/`. No outside sources or additional agents were used. The census tool was the repository `scripts/census.sh` wrapper, with run manifests retaining oracle SHA, seed, worker count, and immutable input hashes. The exact Codex model build identifier was not recorded in those artifacts.

## Method and observations

The round-two population collector was corrected to read argument 0 for one-argument selector calls such as `DreamContext.SendToChar` and `SendToRoom`, while retaining argument 1 for ordinary two-argument calls. Regenerating round two at `5e9556a8789c4125bed8913d34d16e6f2ed68fac` added **2 function rows** (`Dream`, `DreamTravelFn`) and **29 eligible literals**. Its frozen sample and 40-probe denominator were unchanged. Round-three populations for session and spells were separately frozen and hashed before each sample was drawn; the details and hashes are in `docs/research/idiom-experiment/round3/PREDICTIONS.md`.

All 80 sampled mutations received a full census, with no overlapping mutation census. There were **24 mutation-specific catches out of 40** for session and **20/40** for spells. Both counts are inside the preregistered 80% predictive interval of 17–28/40 (22.9 expected). Raw 95% Wilson intervals are 44.6%–73.7% for session and 35.2%–64.8% for spells.

Thirty-six misses had failed census scenarios unrelated to their mutation. The one full recheck required by the original procedure retained red rows, but an unmutated control at the same source base would have shown those rows were already red. In the 2026-11-07 seed-1 rehearsal at Go HEAD `b7fdd6ef58eb48ac407387ab62cdb0ed44a58863`, the 21 distinct failing scenarios were 20 `PASS` and one ledger-backed `EXPECTED` (`spellmatrix-wand-olc-08-depth`), with no `FAIL`. These 36 outcomes are therefore `baseline-red`, not infrastructure, and remain in the raw denominators. Only the raw rates, 24/40 and 20/40, are reported.

Corrected first-pass miss classes are session A6/B8/C2/D0 and spells A10/B9/C1/D0 (overall A16/B17/C3/D0). The five changed verdicts were session probes 21 and 34, and spells probes 4, 18 and 19; they are Class B. The C witnesses and per-probe explanations are in `round3/RESULTS.md` and `round3/run/probes.tsv`. A visible repeated pattern is that misses include both uncovered C-backed state branches (A) and Go fallback paths that production dispatch bypasses or C never emits (B/C). The final spells probe, `castConjureElemental`, was a Go-only spawn-error message; C calls `create_mobile` and has no matching message.

## New observations versus repeated context

This repeats PF-102’s random-function mutation method on two additional packages. It adds package-specific catch observations and first-pass A/B/C/D diagnoses; it does not repair the uncovered branches. Both raw catch counts fell within the prediction interval. The 36 baseline-red misses show that the recheck rule did not distinguish pre-existing source-base failures from mutation-specific outcomes. The Part 1 blind-spot audit separately found two unmutated fixture divergences, documented in its PR; this Part 2 sample did not run unmutated control censuses for all 80 probes.

## Proposed claim and state

**PF-117 — verified:** In the frozen samples, the census produced 24/40 mutation-specific failures for session (60.0%; 95% Wilson 44.6%–73.7%) and 20/40 for spells (50.0%; 95% Wilson 35.2%–64.8%). Both raw counts fall inside the preregistered 17–28/40 predictive interval. The other 36 outcomes were baseline-red misses, not infrastructure exclusions, and remain in the raw denominators. Corrected miss classes are session A6/B8/C2/D0 and spells A10/B9/C1/D0. Evidence: round-three `run/probes.tsv`, `RESULTS.md`, `PREDICTIONS.md`, the retained mutation censuses, and the seed-1 rehearsal at Go HEAD `b7fdd6ef58eb48ac407387ab62cdb0ed44a58863` (`/home/zach/Archives/darkpawns/oracle-runs/2026-11-07/claude-rehearsal-1107`).

## Contradictions, limits, and artifacts

No contradiction to the frozen expected catch count was observed. The main limitation is that no unmutated control was run before the original mutation sample, so baseline-red scenarios were only recognized after the 80 censuses and 36 rechecks. Future mutation rounds should first run one unmutated control across all planned scenarios. This remains a probe-level census measurement, not a completeness claim.

Artifacts created or updated: round-three `run/probes.tsv`, `RESULTS.md`, this field note, and the `PF-117` ledger row. No Part 2 game code or tests were changed.

## Correction history

On 2026-11-08, the results review in PR #2402 corrected two errors in the original note: (1) the 36 misses were not infrastructure failures, and the selected 24/24 and 20/20 rates were removed; the reported rates are 24/40 and 20/40; (2) five first-pass classes changed from C to B (session 21 and 34; spells 4, 18 and 19), changing totals to session A6/B8/C2/D0 and spells A10/B9/C1/D0. The original wording is retained in Git history at the pre-correction PR head `1a26da667948e0e1c1abd9ad8298c3dce98fdb77`. DP-1512 records the five source-path verdict corrections. The full changed row list and rehearsal comparison are in `round3/RESULTS.md` and `round3/run/probes.tsv`.
