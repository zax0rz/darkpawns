# VULN-042: bounded pending session commands

Stop tier: security divergence and session input/transport behavior. Zach approved the concrete policy on 2026-10-07; [DP-1400](https://linear.app/labz0rz/issue/DP-1400/approved-divergence-bound-pending-session-command-queue-vuln-042).

At most 128 commands may await the heartbeat drain. The 129th command, or an alias remainder that would exceed this bound, discards the queue, permanently closes admission for that session and silently closes its transport. Alias remainder admission is all-or-nothing. Below the bound, existing wait delays, raw arguments, FIFO and alias priority/marker are preserved. No new player-facing string, RNG draw or clock path.

## C correction (R4/R5e)

The security brief incorrectly says C's OS socket buffer bounds this queue. `src/comm.c:575-581` reads ready descriptors before the wait-gated drain at 597-603. `process_input` queues complete lines at 2008, and `write_to_q` at 1210-1228 allocates unbounded linked nodes. The raw input overflow at 1867 logs only to the server and returns failure; its 512-byte buffer does not bound queued complete lines. This repair is an approved security divergence, not C parity.

## Queue and lock audit (R5c)

All insertions are in `tryExecuteNow`, test helper `enqueueInput` and `prependAliasedInputs`; all use the same inputMu-protected admission check. No second queue or registry. Alias dispatch no longer takes inputMu outside the helper; the helper owns it and releases it before closing. The admission latch prevents input arriving after the close request from entering the fast path or queue. Removing a head zeros its slot and releases an empty backing slice, avoiding stale retained command strings/args.

Funnel: inputMu, then the existing player wait-state lock; release both before Session.Close. Alias insertion: inputMu only, released before Session.Close. Transport close consults existing heartbeat outputBatch.mu and invokes existing transport closure with inputMu released. Drain's existing manager read lock then player wait lock/inputMu remains unchanged; command execution occurs after manager unlock. No new lifecycle acquisition. An already-dequeued concurrent job retains existing execution semantics; this change does not resolve the separate VULN-021 drain/fast-path synchronization finding.

## Proofs

`TestInputQueueCapWaitFlood` drives 300 manually stepped pulses, rearming wait each time, checks the bound on every pulse, one transport close outside inputMu, no output and no retained queue. It fails on original main at depth 129. `TestInputQueueCapAliases` fills to 127, admits exactly one alias at the head with its aliased marker, then overflows a two-entry batch; old code fails to close. Existing TestInputTiming cases prove below-cap immediate execution, wait delay, one-per-pulse and FIFO/raw argument preservation. Mutation control disables only the shared admission predicate, requires compiling assertion failures, then restores and verifies both tests.

Build, vet, full tests, game tests, lint, diff-check, fidelity-depth and fidelity-units (PASS=1386) pass. Focused -race covers existing timing proofs and concurrent eight-producer admission; TestInputQueueCapConcurrentAdmission requires exactly one close and no retained queue. The string ratchet passes with zero unreviewed bytes; eight pre-existing stale baseline entries and generated report warnings are retained unchanged. Evidence: ~/Archives/darkpawns/oracle-runs/2026-10-07/vuln-042-proofs/. The combined census is required before opening the PR; its final verdict and exact committed HEAD belong in the PR.
