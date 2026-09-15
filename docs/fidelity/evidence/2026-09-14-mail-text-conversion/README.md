# 2026-09-14 mail text-conversion repair

## Status

This evidence records the bounded production repair on branch
glm/fix-mail-text-conversion. The branch started from fresh origin/main at
f3c1bfbf0, which contains merged PRs #1463 and #1464. The implementation
checkpoint is ec34175862827fec6d497354b1228de44f4b6ec6. The repair is
intentionally unmerged and stops for human review.

The #1465 test-correction commit is checkpoint
`77635a46a149eca863a4f5db8ea8cdce1e513764`; that commit changes only
`pkg/game/mail_save_failure_test.go`. Production code, oracle code, Makefile,
scenarios, fixtures, and runner inputs remain identical to the tested
implementation checkpoint `ec34175862827fec6d497354b1228de44f4b6ec6`; the
full census is therefore reused for this test-only delta. The focused suite
and race variant, gofumpt, diff check, build, vet, full tests, game tests,
lint (with `/usr/local/go/bin` on PATH), fidelity-depth, and
expected-divergences-check all pass. The new regression drives a
continuation prefix/NUL/nonzero-tail field through `readDelete`, verifies
first-NUL termination, and verifies that receipt changes native bytes only
through the established deletion markers. The full-field/no-terminator
control remains covered.

The fixed-block padding defect is repaired within the tested boundary. Full
recipient restart/reload acceptance remains blocked by a separate existing
mail-object rehydration defect, described below. This document does not claim
that broader mail lifecycle acceptance or Phase 6.4 ownership injection is
complete.

## Repair and fidelity boundary

The C mail path treats each fixed text field as a C string. The relevant
writer paths zero-fill and terminate the bounded fields in src/mail.c, and
the read path joins them with strcpy/strcat semantics. The Go read path in
pkg/game/mail.go now applies one fixedMailText helper at the same semantic
boundary:

- the decoded header text is stopped at the first NUL;
- every decoded continuation-block text is stopped at the first NUL;
- a full field with no NUL is preserved in full; and
- no trailing-NUL trimming, global JSON sanitization, serializer change,
  writer change, stored-mail rewrite, format migration, identity change,
  locking change, schema change, or save-representation change was made.

The production change is limited to pkg/game/mail.go. The tests convert the
#1464 characterization into repair regressions in
pkg/game/mail_save_failure_test.go, pkg/db/mail_save_failure_test.go, and
tests/e2e/mail_save_failure_test.go. No file under src/ or
darkpawns-c-oracle/ changed.

## Focused regression coverage

The native Go conversion tests cover empty text, short text followed by zero
padding, an embedded terminator followed by nonzero bytes, and a full field
without a terminator. Native fixed-block fixtures cover a short single-block
receive and a bounded two-block receive whose full header and continuation
fields join without padding or truncation. The writer is used unchanged.

The database-package regression proves that a native received mail object
reaches PlayerToRecord without fixed-block padding or a JSON \u0000 escape.
A separately labeled serializer control still preserves an unrelated
embedded NUL as \u0000, proving that this repair is not general JSON
sanitization.

Focused commands, all PASS at the implementation checkpoint:

- go test ./pkg/game -run 'Test(FixedMailText|MailReceive)' -count=1 -v
- go test ./pkg/db -run 'Test(MailReceivedObject|SerializerControl)' -count=1 -v
- go test -race ./pkg/game -run 'Test(FixedMailText|MailReceive)' -count=1 -v
- go test -race ./pkg/db -run 'Test(MailReceivedObject|SerializerControl)' -count=1 -v

The durable focused transcripts are under
/home/zach/dp-mail-text-conversion-evidence-2026-09-14.

## Production persistence proof

The explicit proof used a newly created database named
dp_mail_text_conversion_20260914_a, owned by the test role, and dropped it
after the test completed successfully. Native mail storage was isolated under
the test fixture directory. The proof used unique sender and offline recipient
identities:

- sender SaveSender780798541, database id 1;
- recipient SaveRcpt780798541, database id 2; and
- exact body proof-mail-body.

The real server path proved the following sequence:

1. The sender logged in, composed and completed one letter.
2. The native 512-byte Go mail file was inspected before receipt.
3. The first server was stopped and a second server restarted on the same
   isolated storage.
4. The offline recipient logged in, checked mail, received the letter, read
   it, and showed exactly one mail inventory object.
5. A second check/receive was empty.
6. SIGTERM exercised the actual server session shutdown cleanup path,
   including PlayerToRecord and DB.SavePlayer for the live received object.
7. The PostgreSQL row was reloaded directly and contained exactly one
   synthetic mail object, with exact sender, recipient, and body text,
   no NUL byte, and no \u0000 escape.

The durable production output is in
/home/zach/dp-mail-text-conversion-evidence-2026-09-14/production-disposable-proof:
proof-run.log, proof-summary.json, both native mail snapshots, and both
server logs. The test exit record confirms test_exit=0 and that the dedicated
database was dropped. The server logs contain no DB save error or linkdead
save error. The ordinary full test suite skips this PostgreSQL test when
DP_TEST_DB_URL is absent; that skip is not being presented as persistence
proof.

### Reload boundary found during the proof

A diagnostic version of this same production test attempted a fresh recipient
login after the successful database save. It failed with an empty inventory:

    reloaded recipient inventory mail count = 0, want exactly one readable mail object

The direct row contains the correctly persisted VNum -1 mail object, but
RecordToPlayer only reconstructs inventory entries whose prototype exists in
the world. The synthetic mail object has no world prototype, so the object is
skipped. This is a separate production rehydration/ownership-boundary defect,
not a text-conversion defect. It is deliberately not fixed in this PR.

The concrete follow-up proposal is a separately authorized mail-object
rehydration slice: define the compatibility contract for VNum -1 mail state,
give RecordToPlayer an explicit mail reconstruction path (or an explicitly
owned persisted-mail representation), and add save/restart/relogin tests for
one readable object and empty subsequent receipt. That work must remain
separate from this first-NUL repair and from Phase 6.4 ownership injection.

## Native storage preservation

The disposable production proof captured the 512-byte file before and after
receipt. The measured cmp -l delta was exactly:

    offset 1: 1 -> 2

All non-marker bytes were equal. This is the pre-existing header transition
from live to deleted; there was no change to stored text, continuation layout,
offsets, block size, or markers beyond the expected receive/deletion
transition. SHA-256 snapshots:

- before: 1c653207433742084a5096a8d6feaee47aeb4c9ed81063453d049d55d9704727
- after:  be5f6ff4ad52e3bab924c45fd9e1bd854e04a92759310a4261b769aae30c7633

The different whole-file hashes are explained solely by that one marker byte.

## Validation

At implementation checkpoint ec34175862827fec6d497354b1228de44f4b6ec6:

- gofumpt -l .: PASS
- git diff --check: PASS
- go build ./...: PASS
- go vet ./...: PASS
- go test ./...: PASS
- go test ./pkg/game/...: PASS
- golangci-lint run ./...: PASS after rerun with PATH including
  /usr/local/go/bin
- make fidelity-depth: PASS
- make expected-divergences-check: PASS

The first lint invocation returned status 3 only because its environment
could not find Go. The corrected invocation returned 0 issues; both
transcripts are preserved under
/home/zach/dp-mail-text-conversion-evidence-2026-09-14/validation-ec3417586.

No mail scenario exists in the oracle scenario corpus, so there was no
affected mail scenario to run without inventing oracle coverage. The fresh
full make oracle-regression was nevertheless run with frozen driver scripts,
Makefile, oracle binary, and scenario inputs. Its complete output and
available attempt logs are preserved under
/home/zach/dp-mail-text-conversion-evidence-2026-09-14/full-census-ec3417586.

The final oracle aggregate was:

    scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0

The only UNPINNABLE identity is the established human-cleared
accuse-noarg-depth baseline. Nine infrastructure-shaped attempts recovered
to PASS: bash-peaceful-depth, chide-depth, flee-no-exits, purge-npc-depth,
socials-depth, spec-proc-carrion, two attempts for
spec-proc-castle-guard-north, and tug-depth. Their first-attempt logs show
C-oracle readiness failures; each recovered on a bounded retry with a
no-normalized-divergence result. There were no failed, infrastructure,
timed-out, stale, unexpected, or content-red scenarios.

The runner emitted a final duplicate UNPINNABLE line for
accuse-noarg-depth in its stream. Reconciliation of the complete stream
against the 941 frozen input identities found 941 unique identities, zero
missing, and zero unexpected identities. The watcher snapshot missed only
the final yuball-depth result because the self-cleaning runner directory was
removed immediately after the final stream line; the complete stdout contains
PASS yuball-depth and the final aggregate reports 941 scenarios. This
preservation race does not change the aggregate or scenario census.

## Remaining debts and review stop

This PR leaves the following explicitly open:

- synthetic VNum -1 mail objects are persisted but skipped by RecordToPlayer;
- C/Go native mail formats and broader mail output/object-field fidelity;
- multi-block/free-list/corruption/concurrency lifecycle coverage beyond the
  bounded fixture;
- any mail migration, writer redesign, schema/save-format change, or global
  JSON sanitization; and
- Phase 6.4 ownership injection.

The repair is one unmerged production PR and is stopped here for human review.
