# Handoff: persisted synthetic mail-object reload repair

Date: 2026-09-15

## Decision and review boundary

This is one bounded, unmerged production repair on branch
`glm/fix-mail-reload`. PR #1465 was verified merged before implementation;
fresh `origin/main` was `29aa29d96e95fe224489bdf366bf2aa7efd166bd`. No open PR
overlapped the branch, and the primary checkout was preserved.

The implementation checkpoint is
`cbe87c8b7e67b000d0c116b22e87d094b8946294`. The branch stops for human review.
It does not merge, deploy, inject ownership, alter the save/schema/mail-file
format, add fields, migrate data, refactor general inventory ownership, or
edit `src/` or `darkpawns-c-oracle/`.

## R5 call-path finding

Returning-player login in `pkg/session/session_login.go` calls
`db.RecordToPlayer`. In the new-format inventory path, `RecordToPlayer` only
constructed entries whose VNum resolved through `world.GetObjPrototype`.
Production mail is intentionally synthetic: `CreateMailObject` gives it VNum
`-1`, no prototype, note type, mail keywords/description, pickup behavior, and
runtime `MailText`. `ObjectInstance.GetSaveState` already persists that text as
`state.mail_text`. Therefore PostgreSQL retained the mail object, but reload
skipped it as an unknown prototype.

The repair follows the actual call path and uses the existing APIs. A VNum
`-1` inventory entry is considered mail only when its existing state contains
a non-empty string `mail_text`. `World.CreateMailObject` reconstructs the
canonical identity/type fields, the saved text is passed unchanged, and
`Inventory.RestoreItem` plus `LocInventoryPlayer` establish inventory and
location invariants. Invalid/missing mail state is warned and skipped. Normal
prototype objects and unrelated synthetic objects remain on their prior
paths. No VNum-only reinterpretation or invented prototype is involved.

## Focused proof

The independent persisted-JSON tests in `pkg/db/convert_test.go` prove valid
mail, exact sender/body and runtime text, synthetic identity/type fields,
unrelated synthetic preservation, ordinary prototype inventory, malformed
state handling, and location validation. The focused conversion suite passed
under normal and race execution. Existing mail unit/lifecycle coverage also
passed under race execution.

The measured implementation delta is three files and 235 insertions:

- `pkg/db/convert.go` — 31 insertions;
- `pkg/db/convert_test.go` — 141 insertions; and
- `tests/e2e/mail_lifecycle_test.go` — 63 insertions.

## Production lifecycle: PASS

Against disposable PostgreSQL `dp_mail_reload_20260915` and isolated mail
storage, the real server/session path proved:

1. Persistent sender sends one short letter to an offline persistent
   recipient.
2. Recipient logs in after a real server restart, checks and receives exactly
   one letter, reads it, and sees one inventory mail object.
3. Recipient performs a second check and receive; both are empty.
4. The actual recipient shutdown save succeeds. The persisted inventory has
   exactly one synthetic mail object with exact sender, recipient, and body.
5. The server starts again on the same storage. Recipient logs in, sees one
   readable mail object with exact sender/body, and subsequent check/receive
   are empty.

The durable production rerun artifacts are in
`/home/zach/dp-mail-reload-evidence-2026-09-15/production-rerun/`, including
the complete e2e log, sender/receiver/reload server logs, native mail
snapshots, and `recipient-inventory.json`. The recipient-save and both server
logs explicitly report zero DB-save/linkdead-save errors. A separate
`-race` run in `production-race/` repeats the lifecycle and exits 0.

This is the exact proven boundary: one production receive → save → restart →
login → read path, with one object and once-only empty receipt afterward. It
does not prove all synthetic objects or all mail corruption/free-list/
concurrency behavior.

## Full validation and oracle census

At checkpoint `cbe87c8b7`, gofumpt, diff check, build, vet, full tests, game
tests, lint, fidelity-depth, expected-divergences-check, focused unit tests,
and focused race checks passed. The full oracle regression was run from frozen
branch inputs with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`.
No mail scenario exists in the corpus, so no mail oracle scenario was
invented; the full run is the unrelated regression guard.

The aggregate is:

    scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0

Exit 2 is expected from the established human-cleared
`accuse-noarg-depth` unpinnable baseline. The preserved runner evidence
contains 956 attempt logs for 941 identities. Five infrastructure-shaped
first attempts recovered to PASS: `clan-depth`, `consider-depth`, `kyo-depth`,
`mortal-batch17`, and `peek-success-depth`. The other 10 repeated identities
are the nine pinned expected pairs plus the two-attempt unpinnable baseline
(15 additional attempts total). Manual reconciliation found no missing,
unexpected, duplicate, stale, failed, final-infra, or timed-out scenario.

Complete output and available runner files are under
`/home/zach/dp-mail-reload-evidence-2026-09-15/full-census-cbe87c8b7/`.

## Remaining debts and next recommendation

Retain the remaining C/Go format, output, object-field, free-list, corruption,
and concurrency debts. This PR does not broaden synthetic-object reload, add
mail fields, change ownership, or close Phase 6.4. Phase 6.4 mail ownership
injection is explicitly deferred until separately authorized.

The next non-mail roadmap slice I recommend is the already evidenced,
bounded Phase 6.3 board authority candidate (`boards-remove-authority-depth`,
tracked separately under #1451). It has an existing oracle scenario and a
clear authority-boundary question; it can be reviewed independently of this
mail work. Do not start it as part of this handoff.

Stop for human review of the reload PR.
