# DP-1371 users Login@ harness repair

Stop tier: shared oracle normalization, approved after #1772 and before #1770 Train B. No game, oracle, session or transport output changes.

## Scope

C `src/act.informative.c:2061-2063` formats descriptor login_time as host-clock HH:MM:SS. Go uses connectedAt.Format at the corresponding users field. DP_CLOCK does not freeze either connection's host time. Under 36-worker load, the retained pre-fix main capture shows C 20:24:58 and Go 20:24:59 in informative-residual-depth while gold and toggle match. This is the same flake that left #1772's otherwise passing combined census NOT_CLEAN.

The shared comparison recognizes the exact users table header and divider, then masks only each valid row's Login@ token as <LOGIN@>, keeping the eight-byte width. It ends at the first non-row. Both engines use the same function. Ordinary, keep-ANSI and keep-prompts comparisons share this metadata exception; framing and controls are preserved by the latter two modes. Invalid time formats and clocks outside that table are not masked. R5e/R5g/R5h.

## Required controls

Focused tests compare differing connection seconds across long names, short names, switched and nonplaying rows. Negative controls mutate descriptor number, level/class, name, state, idle, host and footer count, and must remain differences. Exact raw tests preserve all other bytes, including ANSI and CRLF/prompt framing. Table-context and malformed-clock controls prevent broad clock masking. Reverting the helper must fail assertions; restoration must pass.

The old 36-worker main reproduction is retained at ~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-hourly-driver-proofs/baseline-loaded-attempts/attempt.DJoMFk/informative-residual-depth.attempt1.log, with baseline source and run manifests. The implementation PR records the repaired 36-worker load control, all gates, and a combined census. Claude's #1771 census takes priority: no testing/load or census starts while it runs.

## Other readers

The comparison helper affects only users tables in captured blocks. Common oracle comparison, ANSI-preserving comparison and prompt-preserving comparison all retain their existing policies for every other byte. Raw retained server transcripts remain actual server output. Production users command, saved/login timestamps, GMCP and browser output are unchanged. No locks or goroutines are introduced. Train B follows only after this stop-tier PR is reviewed and merged.

## Validation completed

Source tip `3bb229a52`, based on merged #1771 (`d3d7f59f6`). The old normalizer at that base has the same SHA-256 as the retained 36-worker reproduction's base (`29469d8b5`): `23bb1868be78e9a366a29aba39931d74580020fc374798f261d79329ae03fe46`.

Revert/restore control: replacing only normalize.go with the base version makes the new tests fail on clock equality and exact raw output assertions (exit 1); restoring the fix passes, including 100 repetitions. Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-users-clock-proofs/{reverted-tests,restored-tests}.txt`.

The same 36-scenario, 36-worker load selection passes all 36 on the fixed source in 41.184 seconds, without rechecks: `~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-users-clock-load-control/`. Attempt logs are retained in the proofs directory's `fixed-loaded-attempts/`.

Combined census: `~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-users-login-clock/`, clean source `3bb229a52`, reference oracle SHA verified, 36 workers, 3,194 pairs, 977 duplicates removed, 1,816.907 seconds (30m17s). Full **CLEAN**, claims **CLEAN**, no infrastructure rechecks. Existing expected divergences remain classified by their existing ledger entries; no pins, seeds, or ledger rows changed.

Normal gates passed: make fmt, go build ./..., go vet ./..., go test ./..., golangci-lint (after cache clean), make fidelity-depth, make fidelity-units (1,322 claims), make string-census, and git diff --check.
