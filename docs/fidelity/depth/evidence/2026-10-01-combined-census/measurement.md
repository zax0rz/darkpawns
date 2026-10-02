# Combined census measurement

All four invocations used fixed commit `59e523f123ded0dec45b749cd7cc5b6eeebfb671` with the reference oracle.
The two combined runs used identical frozen timing estimates. Commands, run
paths, captured HEADs, wall times and per-pair comparisons are in the retained
`benchmark-results.json`; initial and recheck result files remain separate.

| Run | Time | Verdict/counts |
|---|---:|---|
| baseline-full | 12m 37.9s | oracle-regression: scenarios=1058 passed=1050 expected=5 unpinnable=0 stale=0 failed=0 infra=2 timed_out=0 unstable=1 elapsed=757.870s started=2026-10-01T23:51:51-0400 finished=2026-10-02T00:04:29-0400 verdict=CLEAN_AFTER_RECHECK rechecked=informative-residual-depth,invisibility-assist-depth |
| baseline-claims | 43m 21.2s | oracle-claims: seed=all pairs=3020 expected=11 expected_unstable=0 fail=0 infra=0 pass=3009 stale=0 timeout=0 unpinnable=0 elapsed=2601.176s verdict=CLEAN_AFTER_RECHECK |
| workers-24 | 60m 34.7s | oracle-combined: pairs=3132 deduplicated=946 full=CLEAN claims=CLEAN_AFTER_RECHECK elapsed=3634.741s verdict=CLEAN_AFTER_RECHECK |
| workers-36 | 40m 18.6s | oracle-combined: pairs=3132 deduplicated=946 full=CLEAN claims=CLEAN elapsed=2418.592s verdict=CLEAN |

Final clean raw probe/audience counts (both engines): full {'blocks': 18740, 'empty_open': 4953, 'empty_closed': 5, 'pairs': 1058}; claims {'blocks': 62016, 'empty_open': 16255, 'empty_closed': 17, 'pairs': 3020}. Empty-closed is separate because EOF does not wait for the first-byte deadline. Setup and drain reads are outside these probe counts.

Baseline total: **55m 59.0s**.
Combined at 24: **60m 34.7s**, -8.2% less time.
Combined at 36: **40m 18.6s**, 28.0% less time.
These are single sequential measurements, not a statistical performance estimate.

The previous 300ms-only baseline took 2228.061s. With the new two-second first-byte allowance, the two-run baseline takes 3359.046s: an observed change of 1130.985s (50.8%). The previous 36-worker combined run failed one pair and is never counted as a clean benchmark.

The initial 24-worker slowdown came from concurrency: deduplication reduced worker-seconds from 74553 to 57901 (22.3%), but 24 workers reduce capacity by 33.3%. That run's 2426.657s was near 57901/24 = 2412.542s, confirming that the pool stayed busy.

Both combined runs cover the exact union and match every final baseline
classification: 1,058 full rows and 3,020 claimed pairs at their exact seeds.
Infrastructure rechecks are applied only from retained recheck files. Runtime
seconds are measurements, not part of the classification equality check.
The 946 overlapping seed-1 claims execute once. No scenario or seed is dropped.

Evidence: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-combined-benchmark/first-byte-formal/`, with each census's own MANIFEST.md at the paths
recorded in benchmark-results.json. Seven scheduler/verdict and three first-byte assertion-only revert triples pass. The real zreset CPU-contention oracle triple also passes fixed 0 -> legacy 1 -> fixed 0, with the exact failed census fingerprint under the legacy window.
All Go and fidelity gates pass; all Python and census-wrapper tests pass.
All gates pass on the final first-byte implementation. On the initial scheduling implementation, one Go all-tests invocation had a telnet admission timeout; its full rerun and a focused unchanged-base run passed. No game-code change was made for it.
A preliminary baseline was stopped before the formal experiment to ensure
all four measured invocations use the same final commit; it is not a timing result.
