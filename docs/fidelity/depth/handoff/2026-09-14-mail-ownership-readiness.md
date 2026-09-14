# Mail ownership-injection readiness — 2026-09-14

## Decision

**PROOF-FIRST — recipient-save characterization takes priority.** The
completed #1462 repair clears the previous production boot, persistent-
identity, header-decode, restart-delivery, disabled-dispatch, and C-keyword
blockers, but its passing receive vehicle records a PostgreSQL `22P05` save
error while the recipient is shutting down. Receipt is therefore established;
recipient persistence is not. The smallest next task is a separate,
test-only disposable-PostgreSQL proof of that exact post-receipt save and
reload consequence. Until that proof is complete, the ownership-boundary
experiment below is deferred: a green single-message receipt cannot authorize
ownership injection under R5a/R5c/R5f.

The save proof must first save and reload the recipient successfully, deliver
one short message through the existing path, attempt the recipient save after
receipt, capture the exact failing operation, offending serialized bytes, SQL
error, and reload result, and compare it with a minimal control that uses the
same save path without the delivered object. It must use a dedicated database
and disposable mail storage, and must not change production code, schema,
mail-file bytes, or ownership. The expected isolated `22P05` characterization
is not a repair or a C-parity experiment.

The previously proposed test-only **mail owner boundary** remains the next
ownership experiment after the save proof: under `-race`, start from the
current Go 512-byte format with one multi-block message whose receipt creates
deleted/free blocks, then exercise send, check, receive, free-block reuse after
reopen, composition cancellation, and concurrent index observation against a
serial control. It must report exact file bytes, recipient IDs, once-only
consumption, and no race. Do not launch either experiment or an implementation
from this handoff.

## Provenance and scope

This handoff was prepared in the isolated worktree
`/home/zach/darkpawns-audit-mail-ownership-readiness`, branch
`glm/audit-mail-ownership-readiness`, from fresh `origin/main` at
`f51eb840a`. That base contains merged #1462 at merge `4af6d9fad`; its tested
implementation checkpoint is `d6b64449b4f66c48b2567f22f241f778bab9357c`.
The open-PR query returned no rows. The primary checkout was not modified.
`src/` and `darkpawns-c-oracle/` remain read-only.

This is documentation and preserved-evidence work only. It does not begin
Phase 6.4, alter production/tests/scenarios/fixtures/runner inputs, change a
save or storage format, reopen weather/ban/spec/Phase 6.3 work, or claim mail
fidelity or Phase 6.4 completion.

## Census validation-record closure

The durable census directory is
`/home/zach/dp-mail-review-fix-evidence-2026-09-14/census/`. Its
`checkpoint.txt` contains the exact tested checkpoint
`d6b64449b4f66c48b2567f22f241f778bab9357c`, and `run.py` records that it wrote
the checkpoint before invoking `make oracle-regression` in the repair
worktree. `stdout.log` ends with:

```text
oracle-regression: scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0 elapsed=7057.558s started=2026-09-14T13:17:34-0400 finished=2026-09-14T15:15:12-0400
make: *** [Makefile:105: oracle-regression] Error 2
```

`exit-code.txt` is `2`, which is explained by the one established
human-cleared `accuse-noarg-depth` unpinnable baseline; it is not the evidence
of success by itself. The preserved `results/` directory contains 941 unique
result files and the preserved fingerprint directory set contains 941 unique
names. Both sets exactly match the 941 `.txt` scenario inputs at the tested
checkpoint: missing identities `0`, unexpected identities `0`, and duplicate
identities `0`.

The final result identities are:

| result | count | exact non-PASS identities |
|---|---:|---|
| PASS | 931 | all remaining tested scenario names |
| EXPECTED | 9 | `accuse-depth`, `force-mob`, `medit-entry-depth`, `medit-session-depth`, `redit-entry-depth`, `redit-session-depth`, `sedit-entry-depth`, `sedit-session-depth`, `shoot-target-depth` |
| UNPINNABLE | 1 | `accuse-noarg-depth` |
| FAIL / INFRA / TIMEOUT / STALE | 0 | none |

There are 958 preserved attempt logs. The seven infrastructure-shaped retry
markers and their manually inspected dispositions are:

| scenario | attempts | disposition |
|---|---|---|
| `love-sleeping-depth` | 1–2 bind failure; 3 pass | recovered to PASS |
| `mortal-batch19` | 1 bind failure; 2 pass | recovered to PASS |
| `quaff-bless` | 1 bind failure; 2 pass | recovered to PASS |
| `redit-entry-depth` | 1 bind failure; 2–3 identical normalized reds | final `EXPECTED`; both fingerprints equal the checked-in pins |
| `room-desc-exits` | 1 bind failure; 2 pass | recovered to PASS |
| `wiz-verb-usage` | 1 bind failure; 2 pass | recovered to PASS |

Every inspected infrastructure attempt ends with `SYSERR: bind: Address already
in use` after the server reaches `WHOD port opened`; it does not contain a
normalized content result. The 18 content-red attempts for the nine pinned
EXPECTED identities match their pins on both attempts. The two
`accuse-noarg-depth` attempts have different run-varying fingerprints and no
pin, matching the established human-cleared unpinnable baseline. No unexpected
or flaky content-red remains.

The census runner returned exit 2 only for that baseline. The census is now
complete and must no longer be described as running or being collected.

## Checkpoint equivalence

The census checkpoint is an ancestor of current `origin/main`. The exact
checkpoint-to-main diff contains only:

```text
M  docs/fidelity/depth/handoff/2026-09-14-mail-initialization-repair.md
M  docs/fidelity/evidence/2026-09-14-mail-initialization-repair/README.md
M  docs/specs/tui-setup-wizard.md
```

The non-document diff is empty. Therefore the merged #1462 production
implementation, Go tests, e2e test, scenario files, world/fixture inputs,
oracle inputs, and census runner are byte-identical to the tested checkpoint.
The census evidence is reusable as a regression record, not as mail behavior
proof: the current depth inventory has no mail oracle scenario.

## Finite ownership reconciliation

The inventory below is the existing #1455/Phase 6.4 mail inventory, checked
against the current post-#1462 call paths. “Proof” means the named evidence
actually crosses that boundary; it does not promote a helper seam into a
whole-family claim.

| state | current owner/lifetime → actual readers/writers | existing named proof | gap relevant to moving it |
|---|---|---|---|
| `mailIndex` | Package-global `*MailIndex`, process lifetime. Readers: `findCharInIndex` from `hasMail`, `readDelete`, and `indexMail`. Writers: `scanFile`, `indexMail`, and `readDelete`. | `TestMailLifecycleAcrossProcessRestart_HelperLevel` and `TestMailProductionLifecycleAcrossRestart` now prove one header is indexed after restart, checked, delivered, and removed once. | No multi-message/multi-block index proof; `hasMail` reads without `mailGlobalMu` while writers mutate under it. |
| `freeList` and `fileEndPos` | Package-global linked free list and allocation cursor. `scanFile` rebuilds them; `popFreeList` reads them; `pushFreeList` and `writeToFile` mutate them. They are memory-only and reconstructed at boot. | The production run proves a 512-byte file and deleted header marker after one receive. | No deleted/free-block fixture, multi-block chain, restart reuse, or concurrent cursor/free-list proof. |
| identity hooks + persistent resolver | `worldNameFunc`/`worldIDFunc` are package-global callbacks set by `InitMailSystem`; `GetNameByID`/`GetIDByName`, `readDelete`, and `PostmasterSendMail` read them. `cmd/server/mailIdentity` is a DB-backed process object captured by those callbacks. | `TestMailIdentityUsesPersistentRecordsForOfflineAndRestartLookups`, the failure tests, and the production offline-recipient restart vehicle prove stable DB IDs/names and sender/body delivery. | Hook lifetime is process-global and has no owner/reset contract after boot. Resolver refresh/error semantics must stay DB-backed and must not fall back to the online `World`. |
| disabled state | Package-global `mailDisabled`; `InitMailSystem`/`DisableMailSystem` write it and `postmaster` reads it for `mail`, `check`, and `receive`. | `TestMailBootFailureIsNonFatalAndDisablesMail` and `TestPostmasterDisabledDispatch` prove the boot failure boundary and exact C diagnostic/fallthrough. | It is boot-configured in production but directly mutable in package tests; no owner publication/lifetime is explicit. |
| file/index lock | Package-global `mailGlobalMu`. `storeMail` and `readDelete` hold it across seek/read/write plus index/free-list changes. `scanFile`, `hasMail`, and lookup callbacks do not take it. | `race.log` proves the selected #1462 mail/server tests report no race; those tests are sequential lifecycle tests, not concurrent send/check/receive access. | The untested `hasMail` read versus mutation boundary is the smallest missing ownership experiment. Do not move or widen the lock based on the single-message run. |
| composition map + lock | Package-global `mailWriteEntries` keyed by player ID and `mailWriteMu`, process lifetime. `PostmasterSendMail` creates; `HandleMailInput` reads/deletes/appends; `CancelMailWriting` deletes. Session input calls `HandleMailInput` at `pkg/session/session_login.go:363-378`; disconnect calls `CancelMailWriting` at `pkg/session/manager.go:1191-1194`. | Production send→`@` proves one completed composition; the manager call path is present and the focused race command is clean. | No empty-`@`, max-length, disconnect-cancellation, or concurrent composition proof. A per-World owner must not retain a second package map. |
| persistence contract | `MailFile`, block constants, marker constants, and marshal/unmarshal helpers are immutable package state. Current Go storage is `data/mail`, 512-byte blocks, markers `1/-2/2`; completed writes are immediate and no mail flush occurs at shutdown. | `TestMailReadWriteSharedFilePositioning` and the production header/file assertions preserve the current Go bytes; the restart test proves the decoded header path. | This is not C parity. The ownership task must not change path, block size, markers, encoding, timestamps, or shutdown persistence. |

### Path-level proof split

Existing proof supports a behavior-preserving plumbing change for the named
production route: `cmd/server/main.go:208-230` constructs the DB-backed
identity path; `postmaster` dispatches through the assigned VNum 3010;
`PostmasterSendMail` uses the identity hook and composition state;
`session_login.go:363-378` drains composition input; `manager.go:1191-1194`
cancels it on disconnect; and `PostmasterCheckMail`/
`PostmasterReceiveMail` read and delete the indexed message. The helper and
production vehicles prove the short send→restart→receive path, including the
delivered object and once-only result.

The shutdown caller at `cmd/server/main.go:545-595` stops world activity,
drains sessions, and saves dynamic world state; it does not flush mail state.
Completed mail is already written synchronously, while an in-progress
composition is discarded by the disconnect cleanup path. That unchanged
shutdown contract is part of the ownership boundary, not evidence that every
mail state path is concurrency-safe.

That proof does not support moving the entire mutable group yet. The
free-list/cursor, multi-block, composition-abort/cancel, and concurrent access
paths stop at the gaps in the table. The full census is only a non-mail
regression gate and cannot fill them.

## C comparison, kept separate

The current Go behavior still does not match C as a whole. C uses
`src/db.c:366-374` → `src/mail.c:221-251` for boot, `etc/plrmail`, 100-byte
blocks, markers `-1/-2/-3`, minimum level 2, and stamp price 25. Go now has a
production initializer and restart decode, but still uses `data/mail`,
512-byte blocks, markers `1/-2/2`, minimum level 5, and price 50. The host C
ABI probe reports `sizeof(header_block_type)=104` against the source 100-byte
contract, so the existing C runtime fixture remains unsuitable for claiming a
cross-runtime mail-file comparison.

The #1462 correction aligns the disabled diagnostic with C
`src/mail.c:484-487` and the created-object keywords with
`src/mail.c:574-576`. It does not establish C-equivalent object fields: Go's
synthetic note still lacks C's room description, hold flag, weight, cost, and
load. The mail activity-output family remains the 11-call-site blocked row in
`docs/fidelity/depth/surface-inventory.tsv:27`; no mail oracle scenario exists.

One separate newly observed persistence symptom is retained without a fix:
the passing production receive run's
`/home/zach/dp-mail-review-fix-evidence-2026-09-14/production/server-receive.log:86,88`
records `DB save error` with PostgreSQL `unsupported Unicode escape sequence
(22P05)` during recipient shutdown. The lifecycle receipt/object assertions
passed, but recipient save success did not. The evidence does not establish
whether the cause is mail-object serialization or a broader player-save path;
it is outside this ownership decision and must not be “fixed” by injection.

## Next-task specification: proof-first mail owner boundary

Do not launch this task from the current PR. A future task may proceed in this
order:

1. Freeze the current Go production, test, scenario, fixture, and runner inputs
   at the #1462 implementation checkpoint. Run the single `mail-owner-boundary`
   vehicle described above under `-race`, with a serial control and durable
   output. Stop on any race, lost/cyclic chain, cross-recipient leak, changed
   block bytes, duplicate receipt, or unexplained save/shutdown error.
2. If that proof clears, introduce one `MailSystem` owner in `pkg/game` with
   the following mutable fields: `mailIndex`, `freeList`, `fileEndPos`, the
   injected ID/name callbacks, disabled availability state, the file/index
   mutex, `mailWriteEntries`, and its composition mutex. There must be no
   package-global mutable mail authority after the change.
3. Construct the owner in `cmd/server/main.go` after the PostgreSQL identity
   resolver is available and before session/listener acceptance. Attach it to
   the live `World`; a no-DB boot or initialization failure attaches a disabled
   owner and preserves the existing nonfatal boot/logging boundary.
4. Keep `postmaster(w, ...)` as the C-shaped registry callback, but make the
   `World` postmaster methods delegate to its owner. Change the exact session
   callers at `session_login.go:363-378` and `manager.go:1191-1194` to use the
   same owner for input and cancellation. Convert the current package-level
   `InitMailSystem`, `DisableMailSystem`, `GetNameByID`, `GetIDByName`,
   `HandleMailInput`, and `CancelMailWriting` entry points into owner methods;
   repository search found no external production callers requiring a default
   singleton. Same-package helper tests must use an explicit owner. A temporary
   compatibility wrapper is acceptable only as a direct delegate with no
   package-global backing state.
5. Preserve the current persistence and identity contracts exactly: relative
   `data/mail`, 512-byte blocks, current markers/encoding/timestamp behavior,
   DB `players.id` plus canonical `players.name`, case-insensitive DB lookup,
   reverse-map refresh/error sentinels, immediate completed writes, and
   discard-on-disconnect/in-progress-shutdown semantics. Do not add a reload,
   save field, schema, or C-format conversion.
6. Keep locking explicit. One owner mutex must cover `scanFile`, all index/free
   list/cursor reads and writes, and file I/O; use private locked/unlocked
   helpers to avoid recursive locking. The composition mutex covers only the
   buffer map and is released before `storeMail` acquires the storage mutex.
   Do not acquire `World.mu` or `Player.mu` while holding the storage mutex; do
   not invoke a runtime identity refresh while holding it unless the proof
   documents the lock order. No shutdown flush is added.

### Required before/after acceptance set

Capture the #1462 baseline, then run the same set after injection:

- `pkg/game`: `TestMailReadWriteSharedFilePositioning`,
  `TestMailInitializationFailsClosedWithoutOverwritingUnusableStore`,
  `TestPostmasterDisabledDispatch`, `TestMailObjectCKeywords`,
  `TestMailLookupFallbackWithoutInitialization`,
  `TestMailLifecycleAcrossProcessRestart_HelperLevel`,
  `TestMailHelperLifecycleSameProcess`, plus the new scan/free-list,
  multi-block, abort/max, disconnect, and concurrent boundary tests;
- `cmd/server`: `TestMailIdentityUsesPersistentRecordsForOfflineAndRestartLookups`,
  `TestMailIdentityRejectsIncompletePersistentAuthority`, and
  `TestMailBootFailureIsNonFatalAndDisablesMail`;
- `tests/e2e`: `TestMailProductionBootBoundary` and
  `TestMailProductionLifecycleAcrossRestart`, including persistent offline
  recipient, SIGTERM/restart, exact sender/body, `read letter`, one inventory
  object, and second empty check/receive;
- repository gates: formatting, build, vet, full tests, game tests, lint,
  `make fidelity-depth`, and `make expected-divergences-check`; and
- the full census only as a frozen-input regression gate. No current mail
  oracle scenario exists, so no C-parity claim may be attached to its green
  aggregate.

The implementation is ready for human consideration only if the boundary
experiment and every before/after item pass with no new content, byte, lock,
identity, or persistence discrepancy. This handoff itself does not launch or
authorize it.

## Review stop

This PR contains documentation/evidence only and remains unmerged. Do not
implement the owner, alter the mail format, repair the separate C/Go fidelity
debts, or claim Phase 6.4 complete. Stop for human review.
