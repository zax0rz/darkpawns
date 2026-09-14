# Mail ownership-injection readiness evidence — 2026-09-14

This package records the validation of the completed #1462 mail lifecycle
repair against the Phase 6.4 ownership requirement. It contains no
production, test, scenario, fixture, runner, CI, deploy, website, oracle, or
save-format change.

## Source identity and census artifact

The audit worktree is `/home/zach/darkpawns-audit-mail-ownership-readiness`,
branch `glm/audit-mail-ownership-readiness`, based on fresh `origin/main`
`f51eb840a`. The base contains merged #1462 (`4af6d9fad`). The census was run
before that merge at implementation checkpoint
`d6b64449b4f66c48b2567f22f241f778bab9357c`, recorded in:

`/home/zach/dp-mail-review-fix-evidence-2026-09-14/census/checkpoint.txt`

The census directory's `run.py` captures that checkpoint before invoking
`make oracle-regression` in `/home/zach/darkpawns-fix-mail-initialization`.
The command used the required `/usr/local/go/bin` Go path,
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, four jobs, the
240-second per-scenario timeout, and frozen repository inputs.

## Durable census reconciliation

The final line of `census/stdout.log` is:

```text
oracle-regression: scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0 elapsed=7057.558s started=2026-09-14T13:17:34-0400 finished=2026-09-14T15:15:12-0400
```

`census/exit-code.txt` contains `2`. This is the expected runner disposition
for the established human-cleared `accuse-noarg-depth` unpinnable baseline,
not an unexplained success signal.

The durable file/identity checks are:

| check | result |
|---|---:|
| result files | 941 unique names |
| fingerprint directories | 941 unique names |
| tested `.txt` scenario inputs at checkpoint | 941 |
| missing result/fingerprint identities | 0 |
| unexpected result/fingerprint identities | 0 |
| duplicate result identities | 0 |
| final PASS | 931 |
| final EXPECTED | 9 |
| final UNPINNABLE | 1 |
| final FAIL / INFRA / TIMEOUT / STALE | 0 |

The nine final EXPECTED identities are `accuse-depth`, `force-mob`,
`medit-entry-depth`, `medit-session-depth`, `redit-entry-depth`,
`redit-session-depth`, `sedit-entry-depth`, `sedit-session-depth`, and
`shoot-target-depth`. Their 18 content-red attempts match the checked-in
ledger pins on both attempts. `accuse-noarg-depth` has two different
run-varying fingerprints and no pin, exactly the established human-cleared
baseline.

There are 958 preserved attempt logs. Manual retry inspection found only bind
contention (`SYSERR: bind: Address already in use`) in the seven
infrastructure-shaped attempts:

| scenario | retry disposition |
|---|---|
| `love-sleeping-depth` | attempts 1–2 bind contention; attempt 3 PASS |
| `mortal-batch19` | attempt 1 bind contention; attempt 2 PASS |
| `quaff-bless` | attempt 1 bind contention; attempt 2 PASS |
| `redit-entry-depth` | attempt 1 bind contention; attempts 2–3 identical pinned content red; final EXPECTED |
| `room-desc-exits` | attempt 1 bind contention; attempt 2 PASS |
| `wiz-verb-usage` | attempt 1 bind contention; attempt 2 PASS |

No final failed, infrastructure, timeout, stale, unexpected, or flaky content
result remains. The full durable attempt/result material is under
`census/preserved/dp-oracle-regression.LTOtnd/`.

## Equivalence to merged #1462

The exact checkpoint-to-current-main diff is:

```text
M  docs/fidelity/depth/handoff/2026-09-14-mail-initialization-repair.md
M  docs/fidelity/evidence/2026-09-14-mail-initialization-repair/README.md
M  docs/specs/tui-setup-wizard.md
```

`git diff --quiet d6b64449b4f66c48b2567f22f241f778bab9357c origin/main -- ':!docs/'`
passes. Production Go, tests, e2e tests, scenarios, world/fixture inputs,
runner scripts, expected-divergence inputs, and oracle inputs are unchanged
between the tested checkpoint and merged `origin/main`. The census is reusable
as a regression record, but the repository has no mail oracle scenario, so it
does not prove mail behavior or C parity.

## Repair proof carried forward

The #1462 evidence is preserved at
`/home/zach/dp-mail-review-fix-evidence-2026-09-14/` and in
`docs/fidelity/evidence/2026-09-14-mail-initialization-repair/README.md`.
The final production vehicle proves a PostgreSQL-backed sender, an offline
recipient, current Go 512-byte storage, real SIGTERM/restart, one waiting-mail
check, one receipt, a second empty check/receive, the persistent sender/body
inside `read letter`, and exactly one inventory object. The helper vehicle
proves the same-process control. `race.log` reports clean selected game and
server race runs.

That evidence supports only the crossed paths. It does not prove free-list or
multi-block behavior, empty/max/disconnect composition behavior, or concurrent
checker/store/delete access. Those are ownership proof gaps, not census
failures.

## Scope boundary and disposition

The finite reconciliation and concrete next-task specification are in
[`2026-09-14-mail-ownership-readiness`](../../depth/handoff/2026-09-14-mail-ownership-readiness.md).
The supported decision is **PROOF-FIRST**, with recipient-save
characterization preceding ownership injection. The smallest next task is a
future test-only disposable-PostgreSQL proof that saves/reloads the recipient
before receipt, delivers one short message, attempts the recipient save after
receipt, captures the exact serialized payload/offending bytes and SQL error,
checks reload persistence, and compares a minimal no-mail-object control. It
is not launched here.

The previously proposed `-race` mail-owner boundary remains deferred until
that save proof passes or produces a reviewed blocker. It covers the current Go
format, a multi-block message whose receipt creates deleted/free blocks,
reopen/free reuse, composition cancellation, concurrent check versus
send/receive, a serial control, and exact-byte/once-only assertions. Its
result then decides whether the existing storage lock can move unchanged into
one owner or requires a separate lock repair.

The proposed owner would be the sole runtime authority for the mail index,
free list, file cursor, identity callbacks, disabled state, file/index lock,
composition map, and composition lock. Only immutable format constants and
stateless serialization helpers remain package-level. The persistent resolver
continues to use PostgreSQL `players.id` and canonical `players.name`; the
current Go path and storage format remain unchanged.

Known C/Go format, output, object-field, and broader lifecycle debts remain
explicit in the handoff. The passing receive run also records a separate
recipient DB-save error (`server-receive.log:86,88`, PostgreSQL `22P05`); no
cause is inferred and no fix is included.

## Review stop

This is preserved evidence and documentation only. Do not implement, merge,
change `src/`, alter the mail format, or claim Phase 6.4 complete from this
package.
