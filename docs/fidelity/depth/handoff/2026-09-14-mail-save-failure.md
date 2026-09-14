# Mail recipient-save failure proof handoff — 2026-09-14

## Disposition

**CAUSE ESTABLISHED — repair deferred.** This bounded proof establishes that a
short message delivered through the current Go mail path can make the
recipient's subsequent PostgreSQL player save fail with `22P05`. It does not
repair the value, change a production caller, change a schema or save format,
or begin Phase 6.4 ownership injection. Human review is required before a
repair PR is started.

The proof branch is `glm/audit-mail-save-failure`, based on fresh
`origin/main` at `f51eb840a`, which contains merged #1462. The primary
checkout was not used for edits. #1463 remains a separate open,
documentation-only PR; its recommendation was corrected in place to make
this save proof the priority, and it remains unmerged.

## Exact diagnosis

The failing value is the delivered object's `Runtime.MailText`, not a raw
mail-file block and not a failure of Go JSON marshaling:

1. `pkg/game/mail.go:393-483` reads a 512-byte header, decodes it into a
   `[488]byte` text field, then appends `string(header.Text[:])` at
   `readDelete`. A short body is followed by fixed-block zero padding.
2. `pkg/game/mail.go:543-560,614-626` passes that string through
   `PostmasterReceiveMail` into a synthetic note's
   `ObjectRuntimeState.MailText`.
3. `pkg/game/object.go:381-425` includes the non-empty runtime field as
   `state["mail_text"]`. `pkg/db/convert.go:11-19,175-191` includes that
   state in the inventory and `encoding/json` successfully emits each NUL as
   the JSON escape `\u0000`.
4. `pkg/db/player.go:435-456` sends the resulting bytes as `$24` in
   `UPDATE players ... inventory=$24 ...`; `players.inventory` is JSONB
   (`pkg/db/player.go:135-160`). PostgreSQL rejects the JSONB value with
   `pq: unsupported Unicode escape sequence (22P05)`.

The displayed note does not disprove the defect: `pkg/game/look.go:932-936`
trims trailing NULs for player-facing output only. The in-memory value used by
save remains padded.

## R5 ownership and call-path trace

| State or boundary | Current owner/lifetime | Actual readers/writers | Existing named proof | Remaining gap for the smallest repair |
| --- | --- | --- | --- | --- |
| Header index, free list, file cursor | Package globals in `pkg/game/mail.go`, protected for file operations by `mailGlobalMu`; lifetime is mail-system boot to process shutdown | `scanFile`, `storeMail`, `readDelete`, `hasMail`, and block writes/readers | #1462 restart/once-only lifecycle evidence; `mail_save_failure_test.go` verifies receipt consumes the header and leaves a 512-byte deleted block | Not the cause of this save failure. Multi-block/free-list/concurrency fidelity remains open. |
| Identity hooks and persistent resolver | Package hooks installed by #1462 boot initialization; persistent `players.id`/`players.name` is the authority | `GetIDByName`/`GetNameByID`, send header construction, receive header formatting | #1462 boot/restart identity evidence and the preserved sender/recipient IDs in this proof | No identity movement is justified by this proof; ownership injection remains deferred. |
| Disabled state | Package-level `mailDisabled`, configured at boot and cleared only by successful initialization | Boot and recognized mail dispatch | #1462 disabled-dispatch proof | Not involved after successful boot; no change proposed. |
| Mail-file locking | `mailGlobalMu` owns seek/read/write sequencing for the process lifetime | `storeMail` and `readDelete` | #1462 locking/restart evidence; this proof observes intact before/after 512-byte blocks | Lock ordering and global ownership are outside this defect proof. |
| Composition map and locking | Package `mailWriteEntries`, owned by `mailWriteMu` while a session composes | `PostmasterSendMail`, `HandleMailInput`, `CancelMailWriting` | #1462 send and disconnect cleanup coverage | Not involved after `storeMail`; no composition change proposed. |
| Boot, send/check/receive, disconnect, shutdown callers | Existing server/session owners; receive mutates the player inventory, then session cleanup/linkdead paths call `PlayerToRecord` and `SavePlayer` | `cmd/server/main.go`, `pkg/game/mail.go`, `pkg/session/manager.go:1196-1203,1237-1243`, `pkg/db/convert.go`, `pkg/db/player.go` | This proof reaches real send, restart, receive, second empty check, note read, direct save failure, reload, and shutdown error logging | The save path needs a corrected semantic text value before a repair can claim recipient persistence. |

The proof answers the two required questions separately. Existing lifecycle
proof supports the current file/index transitions and once-only receipt, but it
does not support moving state into a new owner. The focused proof establishes
the C-semantic text defect and its persistence consequence; it does not claim
that all current Go mail output, object fields, file format, or broader
lifecycle behavior matches C.

## Durable proof record

The focused characterization tests are:

- `pkg/game/mail_save_failure_test.go:13-65` receives through the native Go
  mail path and asserts the short body is followed by exactly
  `MailHeaderDataSize-len(body)` NULs in `Runtime.MailText`.
- `pkg/db/mail_save_failure_test.go:18-63` isolates serialization. The padded
  value and a C-style terminated control both produce valid JSON; only the
  padded value contains the JSON `\u0000` escape.
- `tests/e2e/mail_save_failure_test.go:24-241` runs the real server path with
  a disposable mail directory and PostgreSQL: pre-receipt save/reload,
  one send, restart, check/receive, deleted mail block, second empty check,
  `read letter`, post-receipt serialization, expected 22P05 save failure,
  unchanged recipient reload, terminated-value control save/reload, and
  shutdown log assertions.

The final focused run was:

```text
DP_TEST_DB_URL='host=/var/run/postgresql user=zach dbname=dp_mail_review_20260914 sslmode=disable' \
DP_MAIL_SAVE_FAILURE_PRESERVE_DIR=/home/zach/dp-mail-save-failure-evidence-2026-09-14/production-final-rerun \
/usr/local/go/bin/go test ./tests/e2e -run '^TestMailProductionRecipientSaveFailure$' -count=1 -v
```

Durable output is in
`/home/zach/dp-mail-save-failure-evidence-2026-09-14/production-final-rerun/`:

- `proof-run.log` records PASS, pre-receipt save success, a 512-byte sent
  block, a 615-byte runtime value with 473 NULs at offset 142, the sanitized
  JSON payload, expected `db.SavePlayer` error
  `pq: unsupported Unicode escape sequence (22P05)`, unchanged recipient
  reload (`inventory=[]`, room `1204`), successful terminated control
  save/reload, and matching shutdown log assertions.
- `proof-summary.json` records the sanitized payload, error, NUL count,
  unchanged recipient inventory, and successful control.
- `mail-before-receive.bin` and `mail-after-receive.bin` are both 512-byte
  snapshots. The latter has the existing deleted marker; the proof does not
  alter the mail-file representation.
- `server-save-failure-receive.log:86,88` records the real process's
  `linkdead save error` and `DB save error` with the same 22P05.

The earlier production record independently reports the same error at
`/home/zach/dp-mail-review-fix-evidence-2026-09-14/production/server-receive.log:86,88`.
The prior #1462 lifecycle test established receipt but did not attempt or
reload a successful recipient save; this proof does.

The proof used the existing dedicated database
`dp_mail_review_20260914` because the test role could not create another
database (`permission denied to create database`). It started empty, used
unique seeded players, and cleanup returned the player count to zero. This is
a setup limitation, not a behavior result; no live player data or existing
mail file was used.

## Persistence consequence

The recipient's save succeeds and reloads with `inventory=[]` before receipt.
After receipt, the direct `db.SavePlayer` operation fails at PostgreSQL's
JSONB input boundary. Reloading the recipient shows the prior inventory and
room unchanged, so the failed statement did not partially persist the mail
object in this experiment. The mail header has already been marked deleted
and the in-process recipient has the note; this proof does not claim message
loss, rollback of delivery, or recovery semantics beyond those observations.

The terminated control differs only in removing the fixed-block tail before
serialization. Its save succeeds and reload contains the mail body. That
minimal control establishes causality for this value: the JSONB-invalid NUL
escape, not the presence of a mail object or the body text, is the condition
that separates failure from success.

## C comparison and smallest repair boundary

The C source treats fixed-block text as a C string. In `src/mail.c:295-302`
and `323-327,353-355`, it zeroes the block, copies bounded text, and writes an
explicit terminator. In `src/mail.c:435-464`, message construction uses
`strcpy`/`strcat`, so fixed-block padding never enters the semantic message.
The Go writer's on-disk 512-byte block and the C `etc/plrmail` format are not
being unified here.

The smallest repair PR should terminate fixed-block text at the Go mail read
boundary: add one narrowly scoped C-string conversion for fixed mail text and
use it for the decoded header and data-block text in `readDelete` (the sibling
data-block conversion must be audited in the same change). This preserves the
existing mail-file bytes, offsets, markers, identity resolution, lock
ownership/order, JSON keys, PostgreSQL schema, and save format while ensuring
all consumers receive the intended C-backed text. It should not strip NULs
indiscriminately from player JSON or make the serializer guess which strings
are mail text.

That repair must remain separate from ownership injection. The owner remains
the current mail package for file/index/free-list/locking state; the current
boot/session/database owners remain unchanged until a later, explicitly
reviewed Phase 6.4 slice is justified.

Required repair regressions are:

- the focused native receive test sees the body with no fixed padding;
- `PlayerToRecord` contains no `\u0000` for the delivered note;
- the real disposable PostgreSQL path saves and reloads the recipient with
  the mail object and exact body;
- the mail file remains byte-compatible before/after the semantic conversion;
- a second check/receive remains empty, proving once-only delivery; and
- the existing pre-receipt control and the focused serialization control stay
  green.

## Explicit limits and newly recorded defects

This proof does not launch the broader multi-block/free-list matrix, change
the mail format, repair C/Go output or object-field differences, change the
save/schema contract, repair any other JSON value, or begin ownership
injection. Known C/Go format, output, object-field, and broader lifecycle debts
remain explicit. The production log's shutdown success cannot substitute for
the failed save/reload checks above.

The sibling data-block conversion is a newly confirmed analogous risk when a
multi-block message is read, but it is recorded for the repair boundary only;
it is not fixed or expanded into a matrix here.

## Census provenance reused for context

The completed census was not treated as mail proof. Its durable checkpoint is
`/home/zach/dp-mail-review-fix-evidence-2026-09-14/census/checkpoint.txt`:
`d6b64449b4f66c48b2567f22f241f778bab9357c`. That checkpoint is an ancestor of
the fresh `origin/main`; the only changes from it to `origin/main` are
documentation files, including the preserved primary-checkout edit, and
there is no non-documentation production/oracle difference. The durable final
tally in `census/stdout.log:957` is:

```text
scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The only unpinnable case is the established human-cleared
`accuse-noarg-depth` baseline. Retry dispositions are visible in the durable
log (the six infrastructure-shaped retry identities are
`love-sleeping-depth`, `mortal-batch19`, `quaff-bless`, `redit-entry-depth`,
`room-desc-exits`, and `wiz-verb-usage`); the final tally has no failed,
infra, timed-out, or stale scenarios. There is no mail scenario in that
census, so it supplies no evidence of recipient persistence.

Manual inspection of the preserved retry logs found the bounded startup
symptoms reported by the runner (readiness bind collisions and an existing
zone target-object startup message), with no unexpected content-red result.
`census/exit-code.txt` is `2` because the runner reports the one unpinnable
human-cleared baseline; the aggregate counters, rather than that exit code
alone, are the health record.

## Validation and review stop

The proof branch must retain the required documentation-PR gate record:

```text
gofumpt -l .                         PASS
git diff --check                     PASS
go build ./...                       PASS
go vet ./...                         PASS
go test ./...                        PASS
go test ./pkg/game/...               PASS
golangci-lint run ./...              PASS
make fidelity-depth                  PASS
make expected-divergences-check      PASS
focused PostgreSQL proof             PASS (expected isolated 22P05)
```

The durable gate transcript is
`/home/zach/dp-mail-save-failure-evidence-2026-09-14/validation/gates.log`.

Do not merge this proof, merge #1463, change `src/` or
`darkpawns-c-oracle/`, implement the proposed conversion, change production
callers, or begin ownership injection until human review completes.
