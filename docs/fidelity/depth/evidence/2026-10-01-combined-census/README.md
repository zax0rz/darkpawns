# Combined full + claims census

`census.sh start --combined --name <run> --jobs <N>` executes the union of every
scenario at seed 1 and every manifest claim, once per distinct pair (R5h).
The existing full, claims and targeted modes remain available.

One global pool limits active workers across all seeds. Completed-run timings
schedule slow pairs first; unseen pairs fall back to a scenario's timing at
another seed, then the existing static scenario cost. `pairs.tsv` and
`scheduling-timings.tsv` retain the exact schedule and estimates.
`CENSUS_TIMINGS_FILE` freezes estimates for reproducible worker-count comparisons;
it affects scheduling only, never classifications or claim membership.

The unchanged `oracle_regression_worker.sh` retains its three-attempt budget,
expected-divergence pins, fingerprints, timeouts and infrastructure detection.
Each seed keeps initial/recheck dumps, worker logs, all attempt logs,
fingerprints, results and per-seed summaries. Shared prebuilt binaries are retained
once. Full normalized C dumps also remain at `census-dump/`.

`results.tsv` and `recheck-results.tsv` remain full-census projections;
`claims-results.tsv` and `claims-recheck-results.tsv` remain claims projections.
Their columns are unchanged. Per-seed `results.tsv` keeps only claimed pairs;
`combined-results.tsv` additionally records the entire union. Initial results
remain initial results; infrastructure rechecks are separate, never overwritten.

`full-verdict` and `claims-verdict` are independent. Content failures never retry.
Full infra/timeouts recheck once at concurrency 1 only if full has no content
failure; claims infra/timeouts recheck once regardless of another claim's failure.
A shared pair's one recheck supplies both projections when both request it.
Missing, duplicate, unknown or malformed results fail closed. Aggregate verdict
is clean only when both projections are clean (allowing CLEAN_AFTER_RECHECK).
Subset summary elapsed times describe the shared run, not independent seed runs.

Tests: `python3 scripts/test_combined_census.py`, `scripts/test_census.sh`, and
`python3 -m unittest discover -s scripts -p 'test_*.py'`.
Replay the seven assertion-only 0/1/0 triples using `revert_proofs.py <checkout>
<archive-log-directory>`. The checked-in table names each proof.

Benchmark all four invocations on one fixed tooling commit: full at 36, claims
at 36, combined at 24, combined at 36. Freeze the same estimates for both combined
runs using the completed baseline durations. Compare final per-pair verdicts
(apply only the recorded infra rechecks), not elapsed-seconds columns. The
benchmark driver and retained census manifests record commands, source revision,
reference SHA, classifications, and wall times. No game, harness, scenario,
manifest claim, ledger, pin or oracle change belongs to this PR.
