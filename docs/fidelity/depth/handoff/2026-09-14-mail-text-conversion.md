# Handoff: bounded mail text-conversion repair

Date: 2026-09-14

## Decision and review boundary

The fixed-block NUL-padding defect proven in merged PR #1464 is repaired at
the Go mail read boundary. This handoff covers one bounded, unmerged
production repair PR on branch glm/fix-mail-text-conversion. It stops for
human review and does not begin ownership injection.

The implementation checkpoint is
ec34175862827fec6d497354b1228de44f4b6ec6, based on fresh origin/main
f3c1bfbf0 containing merged #1463 and #1464. No open PR overlapped this
branch when work began. The primary checkout remained untouched.

## R5 call-path finding

The C writer creates fixed-size mail blocks with zero-filled bounded text and
an explicit terminator. The C read path uses C-string operations when joining
the header and continuation text. The Go production caller is readDelete in
pkg/game/mail.go: it decodes the fixed header, appends header text, walks
header.NextBlock, decodes each continuation block, and appends its text before
creating the delivered mail object.

The repair adds fixedMailText, which stops at the first NUL and preserves a
full no-terminator field. readDelete uses it for both the header field and
each continuation field. This is the minimum R1/R5 correction. It does not
trim tails, strip arbitrary player strings, alter JSON serialization, change
the writer, rewrite stored mail, change offsets or markers, inject ownership,
or change the schema or save representation.

## Evidence ledger

### Live receipt: PASS

The native Go unit regressions prove:

- empty text;
- short text followed by zero padding;
- embedded terminator followed by nonzero bytes;
- full field without a terminator;
- short single-block receive;
- bounded full header plus continuation receive with exact join and no
  padding; and
- actual receipt creating one Runtime.MailText value with no NUL.

The sender/recipient production vehicle composes the letter through the real
session path. After a real server restart, the offline recipient checks,
receives, and reads the letter. It verifies exact sender and body, one mail
inventory object, no NUL in player-facing text, and an empty second
check/receive.

### Database persistence: PASS

The production vehicle was rerun against a newly created dedicated database,
dp_mail_text_conversion_20260914_a, owned by the test role and dropped after
the successful run. It used isolated native storage and unique test
identities. SIGTERM on the live recipient server exercised the actual
shutdown cleanup path, including PlayerToRecord and DB.SavePlayer. Direct
database reload then verified exactly one persisted VNum -1 mail object with
the exact sender, recipient, body, and no NUL or \u0000 escape. Server logs
contained no DB save error or linkdead save error.

The ordinary go test ./... run skips this test if DP_TEST_DB_URL is absent.
That skip is not proof; the explicit disposable-database run is the
persistence evidence. Durable artifacts are under
/home/zach/dp-mail-text-conversion-evidence-2026-09-14/production-disposable-proof.

### Recipient restart/reload: BLOCKED outside this repair

A diagnostic run continued after the successful save and performed a fresh
recipient login. The reload showed no inventory object:

    reloaded recipient inventory mail count = 0, want exactly one readable mail object

The PostgreSQL row itself contains the saved synthetic VNum -1 mail object.
RecordToPlayer reconstructs only inventory entries with a known world
prototype, so it skips this mail object. This is a distinct mail-object
rehydration/ownership-boundary failure. It is not silently fixed here.

Concrete follow-up scope: separately authorize a mail-object rehydration
slice that defines the VNum -1 compatibility contract, adds an explicit
RecordToPlayer reconstruction/ownership path, and proves save/restart/relogin,
one readable object, exact text, and empty subsequent receipt. Keep that
slice separate from first-NUL conversion and Phase 6.4 ownership injection.

## Storage proof

The disposable production run captured native data/mail before and after
receipt. Both files were 512 bytes. cmp -l reported one difference:

    offset 1: 1 -> 2

All bytes after the pre-existing header live/deleted marker were equal. The
semantic conversion therefore changed no serialized text, block layout,
offset, marker, or writer output. The before/after snapshots and proof
summary are preserved in the production evidence directory.

## Validation checkpoint

At ec34175862827fec6d497354b1228de44f4b6ec6:

- formatting, diff check, build, vet, full tests, game tests, lint,
  fidelity-depth, and expected-divergences-check all passed;
- focused game and database race checks passed;
- the database test was explicitly executed, not merely skipped; and
- the full oracle census ran with frozen driver scripts and scenario inputs.

The first lint command had a status-3 environment failure because Go was not
on PATH. The corrected command with /usr/local/go/bin on PATH reported
0 issues. Both records are preserved in the validation evidence.

The fresh oracle aggregate was:

    scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0

The only unpinnable case is the established human-cleared
accuse-noarg-depth baseline. Nine infrastructure-shaped attempts recovered
to PASS: bash-peaceful-depth, chide-depth, flee-no-exits, purge-npc-depth,
socials-depth, spec-proc-carrion, spec-proc-castle-guard-north (two
attempts), and tug-depth. The preserved first-attempt logs show C-oracle
readiness failures; recovered attempts report no normalized divergence.
There were no unexpected content-red results.

No mail scenario exists in the oracle corpus, so no mail oracle scenario was
invented or claimed. Stream reconciliation against the 941 frozen input
identities found 941 unique identities, zero missing, and zero unexpected
identities. The one duplicate stream line is the expected repeated
UNPINNABLE accuse-noarg-depth summary. The durable result watcher missed
only the last yuball-depth file during self-cleanup, but complete stdout
contains PASS yuball-depth and the aggregate is complete. Full output,
result snapshot, fingerprints, and available attempt logs are preserved at
/home/zach/dp-mail-text-conversion-evidence-2026-09-14/full-census-ec3417586.

## Changed-file coverage and measured delta

The implementation checkpoint changes exactly four files:

- pkg/game/mail.go: 14 insertions, 2 deletions;
- pkg/game/mail_save_failure_test.go: 135 insertions, 11 deletions;
- pkg/db/mail_save_failure_test.go: 48 insertions, 43 deletions; and
- tests/e2e/mail_save_failure_test.go: 72 insertions, 100 deletions.

Total checkpoint delta: 269 insertions and 156 deletions. The writer,
storage constants, identity resolution, locks, markers, offsets, schema,
serializer, and save representation are unchanged. src/ and
darkpawns-c-oracle/ are unchanged.

## Remaining debts

The following remain open and explicitly outside this handoff:

- persisted synthetic mail-object rehydration through RecordToPlayer;
- C/Go native format and output/object-field fidelity;
- broader multi-block, free-list, corruption, and concurrency lifecycle
  coverage;
- mail migration, writer redesign, schema/save-format change, and global
  JSON sanitization; and
- Phase 6.4 ownership injection.

Stop here for human review. Do not merge, deploy, or expand this PR into the
rehydration or ownership work without a new scope decision.
