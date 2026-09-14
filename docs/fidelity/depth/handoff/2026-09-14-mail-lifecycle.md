# Mail initialization and send→restart→receive proof handoff — 2026-09-14

## Disposition and stop boundary

This is one bounded, unmerged proof PR from fresh `origin/main` at `d5328ce1b`
(#1457, weather repair checkpoint `eac85ac30`). It does not wire mail
initialization, change production mail behavior, change a storage format,
inject mail ownership, edit `src/` or `darkpawns-c-oracle/`, or expand into
the remaining mail matrix. Human review is required before any repair.

The production lifecycle is **blocked**. The real server reaches the room
containing assigned postmaster VNum 3010 and dispatches `mail`, but the fresh
character has zero coins and stops at the current Go stamp gate. Source/R5
evidence separately establishes that the live boot sequence has no caller of
`InitMailSystem`. The explicit helper vehicle passes the fee and proves a
same-process send, check, receive, and one-time consume; a separate child
process reopens the resulting 512-byte Go file but indexes zero messages, so
restart/reopen is blocked by `scanFile`.

## Findings

1. `pkg/game/mail.go:95-100` is the only production implementation of
   `InitMailSystem`; a non-test search finds no caller. Its hooks consequently
   retain the fallback behavior (`GetIDByName` → `-1`, `GetNameByID` →
   `Player(<id>)`) in the live boot path.
2. The natural future insertion point is after `session.NewManager` has
   established the live identity context and before listener acceptance. The
   identity authority still needs an explicit decision, so this PR does not
   insert it.
3. Go postmaster registration and command dispatch are present. The production
   telnet vehicle reached room 1204 and observed the exact current Go
   affordability output (`50` coins) before recipient lookup/composition.
4. The helper child process proves a short single-block message completes and
   writes an isolated native Go file. On a new process, `scanFile` reads the
   bytes but never unmarshals them into `nextBlock`, so no recipient index is
   rebuilt. This is a precise known defect, characterized with a passing test.
5. C remains the comparison authority: boot scan/no-mail handling is in
   `src/db.c:366-371` and `src/mail.c:221-251`; postmaster assignment and
   send/check/receive are in `src/spec_assign.c:214` and `src/mail.c:476-595`.
   C and Go use separate native paths and formats (`etc/plrmail`/100-byte
   blocks versus `data/mail`/512-byte blocks); no fixture was cross-read.

## Result table

The finite stage table, including actual paths, C evidence, Go evidence, and
smallest next actions, is in
[`2026-09-14-mail-lifecycle`](../../evidence/2026-09-14-mail-lifecycle/README.md).
The short result is:

```text
production boot/init: blocked — no InitMailSystem caller
production dispatch: reached — stopped at stamp affordability
helper compose/store: proven with explicit initialization
helper restart/reopen: blocked — 512 bytes read, 0 messages indexed
helper check/receive/consume once: proven same-process only
overall production lifecycle: blocked; no injection readiness claim
```

## Validation and census reuse

The focused helper trace and production telnet trace are preserved under
`/home/zach/dp-mail-lifecycle-evidence-2026-09-14/`. The full validation gate
record will be appended to the evidence package before the PR is opened.
No oracle scenario, fixture, manifest, or runner input changed, so a fresh
`make oracle-regression` is not required. Reuse is tied to the tested #1457
checkpoint `eac85ac30379879633167611b858016414287f97`; the established healthy
census is `941` scenarios, `931` passed, `9` expected, `1` human-cleared
unpinnable, `0` failed, `0` infra, `0` timed out, and `0` stale.

## Concrete next task

Take one bounded initialization/fidelity repair: give the production boot
owner an explicit `InitMailSystem` call with a reviewed identity-hook source,
and fix `scanFile` to decode the existing Go 512-byte header before indexing.
Keep the current Go path/constants/format unchanged in that slice; keep the C
100-byte/`etc/plrmail` contract separate for a later fidelity decision. The
repair's acceptance proof should rerun this exact production send attempt with
a disposable sender fixture that can pay the current stamp, then perform a
real process restart and verify recipient receipt/once-only consumption.

STOP HERE for human review. Do not merge or begin the repair in this task.
