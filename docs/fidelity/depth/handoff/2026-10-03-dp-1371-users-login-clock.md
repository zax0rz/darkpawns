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
