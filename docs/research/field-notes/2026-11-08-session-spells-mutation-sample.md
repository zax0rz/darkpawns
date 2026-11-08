# Session and spells mutation sample

## Research track and question

**Track:** legacy-port fidelity. **Question:** how often does the full differential census detect a preregistered one-byte mutation in randomly sampled player-output functions from `pkg/session` and `pkg/spells`, and how well do predicted scenarios identify the mutation’s coverage?

## Time window and sources

Work ran 2026-11-06 through 2026-11-08 in the `luna/round3-mutation-sample` worktree, based on Part 1 commit `aa867bf4dfa25c0fcafe337c11071d56ed7aee40`. Sources were the repository’s frozen populations, prediction and probe TSVs, the `scripts/census.sh` manifests/results/transcripts under `~/Archives/darkpawns/oracle-runs/2026-11-06` through `2026-11-08`, and the cited C source under `src/`. No outside sources or additional agents were used. The census tool was the repository `scripts/census.sh` wrapper, with run manifests retaining oracle SHA, seed, worker count, and immutable input hashes. The exact Codex model build identifier was not recorded in those artifacts.

## Method and observations

The round-two population collector was corrected to read argument 0 for one-argument selector calls such as `DreamContext.SendToChar` and `SendToRoom`, while retaining argument 1 for ordinary two-argument calls. Regenerating round two at `5e9556a8789c4125bed8913d34d16e6f2ed68fac` added **2 function rows** (`Dream`, `DreamTravelFn`) and **29 eligible literals**. Its frozen sample and 40-probe denominator were unchanged. Round-three populations for session and spells were separately frozen and hashed before each sample was drawn; the details and hashes are in `docs/research/idiom-experiment/round3/PREDICTIONS.md`.

All 80 sampled mutations received a full census, with no overlapping mutation census. There were **24 mutation-specific catches out of 40** for session and **20/40** for spells. Both counts are inside the preregistered 80% predictive interval of 17–28/40 (22.9 expected). Raw 95% Wilson intervals are 44.6%–73.7% for session and 35.2%–64.8% for spells.

Thirty-six misses had unrelated failed census scenarios. Each received one full recheck; each recheck remained red only on unrelated failures, so those rows are marked `infra` and excluded from the conclusive-rate denominator. The conclusive rates are 24/24 and 20/20; that conditional result is selected because every excluded row was a miss, so the raw 24/40 and 20/40 counts remain the direct comparison with the forecasts.

First-pass miss classes were session A6/B6/C4/D0 and spells A10/B6/C4/D0. The C witnesses and per-probe explanations are in `round3/RESULTS.md` and `round3/run/probes.tsv`. A visible repeated pattern is that misses include both uncovered C-backed state branches (A) and Go fallback paths that production dispatch bypasses or C never emits (B/C). The final spells probe, `castConjureElemental`, was a Go-only spawn-error message; C calls `create_mobile` and has no matching message.

## New observations versus repeated context

This repeats PF-102’s random-function mutation method on two additional packages. It adds package-specific catch observations and first-pass A/B/C/D diagnoses; it does not repair the uncovered branches. Both observed raw catch counts fell within the same prediction interval, though a large share of misses were inconclusive under the full-census/recheck rule. The Part 1 blind-spot audit separately found two unmutated fixture divergences, documented in its PR; this Part 2 sample did not run unmutated control censuses for all 80 probes.

## Proposed claim and state

**PF-108 — verified:** In the frozen samples, the census produced 24/40 mutation-specific failures for session and 20/40 for spells; after excluding 16 and 20 recheck-persistent unrelated-red rows, the conclusive samples were 24/24 and 20/20. Both raw counts fall inside the preregistered 17–28/40 predictive interval. Evidence: round-three `run/probes.tsv`, `RESULTS.md`, `PREDICTIONS.md`, and the retained census directories listed in `RESULTS.md`.

## Contradictions, limits, and artifacts

No contradiction to the frozen expected catch count was observed. The main limitation is that all uncaught probes were excluded from the conclusive rate after unrelated failures persisted, so the 100% conditional rates should not be interpreted as unbiased package estimates. This remains a probe-level census measurement, not a completeness claim.

Artifacts created or updated: round-three `run/probes.tsv`, `RESULTS.md`, this field note, and the `PF-108` ledger row. No Part 2 game code or tests were changed.
