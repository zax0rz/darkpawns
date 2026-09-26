# 2026-09-26 — Restart Oracle, Lifecycle Corrections, and SQLite Cutover

## Scope

- **Tracks:** legacy-port fidelity; multi-agent software engineering
- **Window:** 2026-09-26
- **Primary artifacts:** PRs #1658, #1659, and #1660; oracle scenario corpus and regression summaries at their merged revisions
- **Status:** artifact-grounded field note. It records engineering evidence, not a paper conclusion.

## Real process boundaries changed the proof

PR #1658 extended the differential oracle with a true process-restart lifecycle. Each implementation is stopped and started against the same disposable durable files while transient process state is discarded. Six vehicles compare C and Go across dropped objects, gossip history, door state, mob state, reset-spawn duplication, and the durable-player boundary.

This is stronger than calling serialization functions in-process. It tests the same boundary an operator crosses during an actual reboot and distinguishes transient world state from durable player state. The restart implementation also became reusable infrastructure for future lifecycle questions.

The isolation harness demonstrated that 36 workers could run separate C and Go process pairs without shared ports, data directories, databases, surviving processes, or newly leaked scratch directories. The recorded full-corpus run completed 1,019 scenarios in 887.190 seconds. This is a dated throughput observation on one workstation, not a general scaling law.

## The oracle corrected the explanation, not only the code

The first restart vehicles exposed two returning-login CRLF differences. Initial reasoning attributed them to the C `process_output` trailing-CRLF clause. PR #1659's raw-byte investigation refuted that explanation:

1. The greeting frame receives a leading interruption CRLF because of the C game-loop accept/output ordering.
2. Echo restoration queues malformed telnet bytes containing bare CR and LF values, which become visible text bytes at specific nanny transitions.

That correction matters methodologically. A plausible source-level explanation survived until byte capture and exact call-path analysis contradicted it. The corrected implementation made the restart vehicles pass instead of merely classifying their output as expected divergence.

The same work exposed and corrected login-room selection at the level boundary around `LVL_IMMORT`, including the distinction between legal quit saves, entry/linkloss saves, selected load rooms, and start-room fallback. These are lifecycle contracts distributed across several C call sites rather than one obvious function.

## Artifact-grounded review caught a verifier blind spot

PR #1660 added PostgreSQL-to-SQLite migration tooling for the production cutover. Its first version copied and compared the five known runtime tables, but `--verify-only` did not inventory unknown source tables or extra source columns. It could therefore report success while data outside the known schema was omitted.

Review identified the gap before merge. The follow-up made conversion and verification share complete source-schema inventory, fail closed by default, and require an explicit successful conversion receipt for any deliberate omission. A second correction distinguished ordinary pre-install failure from a post-rename directory-sync failure where the database is installed but durability is uncertain.

The production cutover then encountered exactly the kind of legacy schema the fail-closed design was meant to reveal: active runtime tables coexisted with retired agent and research tables. The operator explicitly excluded those tables from active SQLite while preserving them in untouched PostgreSQL and a restore-tested backup. No private row contents or operational credentials belong in this public note.

## Multi-agent observation

DeepSeek produced both large infrastructure changes and responded accurately to review findings. GLM produced the login lifecycle corrections, while independent review found a PostgreSQL-backed E2E assertion that local default tests had skipped. The useful unit of analysis is not "agent succeeded" or "agent failed." It is the loop:

1. bounded brief;
2. agent implementation and receipts;
3. independent artifact review;
4. correction against repository, oracle, CI, or production evidence;
5. preserved account of the correction.

The corrections are evidence about the process. Erasing them would leave only a flattering and false success narrative.

## Limits

- Scenario count and runtime describe this corpus and workstation only.
- The migration cutover proves this five-row production database and schema transition, not arbitrary PostgreSQL-to-SQLite portability.
- No claim is made here about model-wide superiority from one successful DeepSeek or GLM session.
- Operational backup locations, credentials, and player data are intentionally omitted.
