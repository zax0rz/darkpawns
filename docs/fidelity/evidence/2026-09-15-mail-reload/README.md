# 2026-09-15 persisted synthetic mail-object reload repair

## Status and provenance

This evidence records one bounded, unmerged reload repair on branch
`glm/fix-mail-reload`. The branch was created from fresh `origin/main` at
`29aa29d96e95fe224489bdf366bf2aa7efd166bd`, which contains merged PR #1465
(`fix: terminate fixed mail text at read boundary`, merged 2026-09-15).
There were no open PRs to overlap when the branch was created. The primary
checkout was left untouched.

The implementation checkpoint is
`cbe87c8b7e67b000d0c116b22e87d094b8946294`. The final documentation/PR head
is reported in the closing handoff and review report. This repair does not
merge, deploy, add ownership injection, change the schema or save format, or
edit `src/` or `darkpawns-c-oracle/`.

## R5 finding and reconstruction contract

Returning-player login calls `db.RecordToPlayer` from
`pkg/session/session_login.go`. Its new-format inventory path decoded each
`SaveItemData`, looked up `item.VNum` in world prototypes, and skipped an
unknown VNum. `game.CreateMailObject` deliberately creates synthetic mail
with VNum `-1`, so a persisted mail object was present in PostgreSQL but was
not reconstructed on login.

The repair adds a narrow branch in `pkg/db/convert.go`:

- VNum `-1` is only a candidate, not the discriminator. Existing persisted
  `state.mail_text` must be a non-empty string;
- the existing `World.CreateMailObject` constructor re-establishes the
  established synthetic identity and type fields (`VNum -1`, no prototype,
  mail keywords/description, note type, and pickup behavior);
- the persisted mail text is passed through unchanged, including its sender,
  recipient, body, and existing bytes;
- the object is inserted with `Inventory.RestoreItem` and its location is set
  through the existing `LocInventoryPlayer` invariant;
- ordinary prototype-backed objects continue through their existing path;
  unrelated VNum `-1` synthetic objects do not become mail; and
- missing, empty, non-string, or otherwise malformed `mail_text` state is
  skipped with an explicit warning. No prototype is invented and no other
  synthetic state is reinterpreted.

This is the smallest safe contract available in the existing persisted data:
`ObjectInstance.GetSaveState` already writes `mail_text`, while mail identity
and type are canonical constructor fields rather than new save fields.

## Focused unit coverage

`pkg/db/convert_test.go` uses independent literal persisted JSON and verifies:

- valid persisted mail reconstructs as exactly one inventory object;
- exact `Runtime.MailText`, sender/body text, VNum, prototype absence,
  keywords, short description, note type, pickup behavior, and inventory
  location survive reconstruction;
- an unrelated synthetic VNum `-1` object remains skipped;
- missing state, missing/null/non-string `mail_text`, and empty text are
  explicitly skipped; and
- an ordinary prototype-backed inventory item retains its normal prototype,
  VNum, state override, and type path.

The valid-mail test does not serialize an object and then read the same bytes;
its persisted JSON and expected mail text are independent fixtures. The
location validator is also exercised directly.

## Production lifecycle proof

The dedicated disposable PostgreSQL database was
`dp_mail_reload_20260915`, owned by the test role, with isolated native mail
storage under each temporary server fixture. Unique disposable identities
were used. Durable artifacts are under:

- `/home/zach/dp-mail-reload-evidence-2026-09-15/production-rerun/`; and
- `/home/zach/dp-mail-reload-evidence-2026-09-15/production-race/`.

The real session/server/persistence path proved this sequence:

1. A persistent sender logged in and composed one short letter to an offline
   persistent recipient.
2. A real server restart preserved the native mail store; the recipient
   logged in, checked mail, received one letter, read it, and saw exactly one
   `a piece of mail` inventory object.
3. The second check and second receive were empty.
4. The recipient quit through the live server path. Shutdown cleanup saved the
   recipient through `PlayerToRecord` and `DB.SavePlayer`; the test then
   verified the persisted row contained exactly one synthetic mail object with
   exact sender, recipient, and body text and no NUL.
5. A new server instance on the same database and isolated mail storage let
   the recipient log in again. Inventory contained exactly one mail object;
   `read letter` returned the exact sender/body; subsequent `check` and
   `receive` were empty.

The rerun log records sender `MailSender545033097`, offline recipient
`MailRcpt545033097`, `production_recipient_save completed=true`,
`production_relogin completed=true`, and zero DB-save/linkdead-save errors in
both receive and reload server logs. The preserved
`recipient-inventory.json` contains one item and the exact persisted
`mail_text`. The separate race-backed lifecycle run records the same behavior
with sender `MailSender704540728` and exits 0.

The ordinary test suite skips the PostgreSQL vehicle when `DP_TEST_DB_URL` is
absent; that skip is not being used as proof. The explicit runs above used a
live dedicated PostgreSQL instance and the real server.

## Validation checkpoint and census

At implementation checkpoint `cbe87c8b7`, the following gates passed; full
transcripts are in
`/home/zach/dp-mail-reload-evidence-2026-09-15/validation-cbe87c8b7/`:

- `gofumpt -l .` and `git diff --check`;
- `/usr/local/go/bin/go build ./...`;
- `/usr/local/go/bin/go vet ./...`;
- `/usr/local/go/bin/go test ./... -count=1`;
- `/usr/local/go/bin/go test ./pkg/game/... -count=1`;
- `golangci-lint run ./...`;
- `make fidelity-depth`;
- `make expected-divergences-check`;
- focused unit and production lifecycle tests; and
- focused `-race` checks for the conversion package, mail game tests, and
  the real production lifecycle.

No mail scenario exists in the frozen C/Go oracle corpus, so no affected mail
oracle scenario was invented. The fresh full `make oracle-regression` used
frozen Makefile/runner/divergence files, oracle binary, and 941 scenario
inputs. Complete stdout, the frozen-input manifest, and available runner
attempt logs are preserved under
`/home/zach/dp-mail-reload-evidence-2026-09-15/full-census-cbe87c8b7/`.

Final census:

    scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0

The command exited 2 solely because the established human-cleared
`accuse-noarg-depth` baseline remains unpinnable. The nine expected results
were pinned baseline divergences. There were 956 preserved attempt logs for
941 scenario identities: the extra 15 attempts are the nine expected pairs,
the two attempts for the one unpinnable case, and five infrastructure-shaped
retries that recovered to PASS (`clan-depth`, `consider-depth`, `kyo-depth`,
`mortal-batch17`, and `peek-success-depth`). Manual inspection found no
unexpected content-red, stale, final infra, or timeout result. The preserved
full stream and result reconciliation contain zero missing or unexpected
scenario identities.

## Changed-file coverage and measured delta

The implementation checkpoint changes exactly three files:

- `pkg/db/convert.go`: 31 insertions;
- `pkg/db/convert_test.go`: 141 insertions; and
- `tests/e2e/mail_lifecycle_test.go`: 63 insertions.

Measured checkpoint delta: **235 insertions, 0 deletions**. No ordinary
prototype reload path, unrelated synthetic object path, mail file bytes,
mail format, JSON/schema/save representation, ownership model, oracle input,
or C/oracle file changed.

## Remaining debts and stop boundary

This proves the receive → recipient save → server restart → recipient login
→ read sequence for one persisted production mail object, plus exact once-only
empty subsequent receipt. It does not claim:

- Phase 6.4 mail ownership injection (explicitly deferred);
- broad synthetic-object reconstruction;
- C/Go format, output, object-field, free-list, corruption, or concurrency
  fidelity;
- mail migration, schema/save-format changes, or writer changes; or
- broader mail lifecycle coverage beyond this bounded production contract.

Stop here for human review. Do not merge, deploy, or begin the deferred
ownership-injection slice from this PR.
