# Mail recipient-save failure proof — 2026-09-14

This evidence package characterizes the PostgreSQL save failure after a short
mail receipt. It is deliberately separate from the #1463 ownership-readiness
documentation and contains no production repair.

## Result

**Cause established; repair deferred.** The current Go `readDelete` path turns
the fixed 488-byte header text into a Go string without stopping at its first
NUL. A short message therefore reaches `Runtime.MailText` with 473 trailing
NULs. `encoding/json` emits valid JSON containing `\u0000`; PostgreSQL rejects
that value while parsing the `players.inventory` JSONB parameter in
`DB.SavePlayer`, with `pq: unsupported Unicode escape sequence (22P05)`.

The minimal terminated-value control saves and reloads successfully. The
repair boundary is the mail fixed-block read conversion, not blanket NUL
stripping in player JSON. No repair is included here.

## Tests

- `pkg/game/mail_save_failure_test.go` proves the native Go mail path places
  fixed-block padding in `Runtime.MailText`.
- `pkg/db/mail_save_failure_test.go` proves the serializer boundary and the
  terminated control without requiring PostgreSQL.
- `tests/e2e/mail_save_failure_test.go` is explicit about missing
  `DP_TEST_DB_URL`: it skips with a “skip is not proof” message. With the
  dedicated database URL it proves pre-receipt save/reload, real send and
  restart receipt, post-receipt serialization, the expected 22P05, unchanged
  failed-save reload, and successful control save/reload.

## Durable run

Final command:

```text
DP_TEST_DB_URL='host=/var/run/postgresql user=zach dbname=dp_mail_review_20260914 sslmode=disable' \
DP_MAIL_SAVE_FAILURE_PRESERVE_DIR=/home/zach/dp-mail-save-failure-evidence-2026-09-14/production-final-rerun \
/usr/local/go/bin/go test ./tests/e2e -run '^TestMailProductionRecipientSaveFailure$' -count=1 -v
```

Preserved artifacts:

- `/home/zach/dp-mail-save-failure-evidence-2026-09-14/production-final-rerun/proof-run.log`
- `/home/zach/dp-mail-save-failure-evidence-2026-09-14/production-final-rerun/proof-summary.json`
- `/home/zach/dp-mail-save-failure-evidence-2026-09-14/production-final-rerun/mail-before-receive.bin`
- `/home/zach/dp-mail-save-failure-evidence-2026-09-14/production-final-rerun/mail-after-receive.bin`
- `/home/zach/dp-mail-save-failure-evidence-2026-09-14/production-final-rerun/server-save-failure-receive.log`

The run records a 512-byte mail block, 615-byte runtime text, 473 NULs,
expected direct `db.SavePlayer` failure, recipient reload with unchanged
`inventory=[]` and room `1204`, successful terminated control save/reload, and
matching shutdown error logs. The earlier production observation is retained
at `/home/zach/dp-mail-review-fix-evidence-2026-09-14/production/server-receive.log:86,88`.

The test used the existing empty dedicated database
`dp_mail_review_20260914` because the test role could not create a new
database. It seeded unique rows and cleaned them up; no live data or existing
mail file was used.

The final formatting, build, vet, full-test, game-test, lint,
fidelity-depth, and expected-divergences transcript is preserved at
`/home/zach/dp-mail-save-failure-evidence-2026-09-14/validation/gates.log`.

## C semantics and repair contract

`src/mail.c:295-302,323-327,353-355` zero-terminates fixed text before
`strcpy`/`strcat` message construction at `src/mail.c:435-464`. A repair PR
should apply that semantic conversion to decoded Go header and data-block
text at `readDelete`, preserving on-disk bytes, current locks, identity,
JSON/schema/save format, and once-only delivery. Its regressions must include
receipt, save success, reload with exact body, unchanged mail bytes, and a
second empty check/receive.

Known C/Go format, output, object-field, and broader lifecycle debts remain
open. The analogous data-block padding risk is recorded but not tested as a
broader matrix or fixed here.

See the dated [handoff](../../depth/handoff/2026-09-14-mail-save-failure.md)
for the full R5 trace, persistence consequence, census provenance, limits, and
review stop.
