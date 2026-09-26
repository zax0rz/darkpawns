# Mail initialization/restart repair handoff — 2026-09-14

## Disposition

PR #1462 is now merged. This handoff is retained as the historical repair
record for branch `glm/fix-mail-initialization`; its final implementation
checkpoint is `d6b64449b4f66c48b2567f22f241f778bab9357c`. PR #1461 was verified
merged before that work began; its merge is `1dd4794ac`. The original
implementation checkpoint was `411ad4b20` (`fix: initialize persistent mail
and rebuild index`), followed by correction `7992c3b0d` (`fix: keep boot alive
when mail is unavailable`) and the final disabled-dispatch/keyword correction
at `d6b64449b`. The ownership-readiness decision is recorded in the dated
follow-up handoff
[`2026-09-14-mail-ownership-readiness`](2026-09-14-mail-ownership-readiness.md).
The production code, tests, fixtures, scenarios, and runner inputs remain
frozen at the corrected checkpoint.

The primary checkout's pre-existing `docs/specs/tui-setup-wizard.md` edit was
not touched. There are no edits to `src/` or `darkpawns-c-oracle/`. The open
PR overlap query was empty.

## Identity authority and boot owner

R5 tracing establishes the existing PostgreSQL player row as the Go
name↔ID authority: `pkg/db/player.go:270-324` supplies case-insensitive
`GetPlayer` plus `ListPlayerNames`, while `pkg/session/session_login.go:164-172`
loads and preserves the row's stable ID/canonical name. The C call path is
`src/db.c:468-528` (`player_table`) and `src/db.c:2313-2337`
(`get_id_by_name`/`get_name_by_id`).

`cmd/server/mail_identity.go:11-119` builds a validated reverse map from
those existing DB APIs, resolves names through the DB, refreshes on reverse
misses, and rejects lookup failures, missing records, invalid identities, or
contradictory IDs. It never falls back to the online World map. Tests at
`cmd/server/mail_identity_test.go:11-108` cover offline/case/post-boot
resolution and failure boundaries.

`cmd/server/main.go:208-231` owns initialization immediately after
`session.NewManager` and before listener acceptance. PostgreSQL-backed startup
preflights identity and calls `game.InitMailSystem`; missing mail creates the
existing empty store. Unusable storage or identity preflight failures now
clear/disable mail, log the error, and continue booting, matching C's explicit
`no_mail` availability disposition. `cmd/server/mail_boot.go:11-26` owns the
recoverable initializer boundary and `cmd/server/main_test.go:121-140` proves
that an unusable store does not become a fatal server-start condition.
`DP_ALLOW_NO_DB=1` remains explicitly non-persistent and logs mail disabled;
it cannot safely provide restart-stable offline identities and is not used for
the production vehicle.

## Exact repair and proof

`pkg/game/mail.go:96-120` clears stale index/identity state and resets failed
initialization to disabled mail. `pkg/game/mail.go:217-286` only creates an
absent file, reads full 512-byte blocks, unmarshals the current Go header
before checking its marker and recipient, and fails closed on
open/create/read/stat/partial/corrupt storage. Existing Go markers, offsets,
block size, encoding, and written bytes are unchanged. The sibling audit found
`readDelete` already explicitly decodes header/data blocks after `readFromFile`;
no sibling discarded-decode repair was broadened here.

The helper regression at `pkg/game/mail_lifecycle_test.go:43-82,173-199`
keeps the same-process control and now proves a distinct-process restart:
one indexed message, waiting check, one receipt with body, one inventory item,
then empty second check/receive and no remaining mail.

The production vehicle at `tests/e2e/mail_lifecycle_test.go:67-238` seeds a
level-34 sender and persistent level-1 recipient before boot, leaves the
recipient offline, uses the real telnet login/menu path, funds with live
`set`, composes one short single-block body through the editor, sends SIGTERM,
restarts the server against the same temporary `data/mail`, logs the recipient
in, checks/receives, and repeats check/receive to prove no duplicate. It then
runs `read note` and `inventory` through the real session path, asserting the
delivered note's persistent sender name, exact body `proof-mail-body`, and
exactly one mail inventory item. Header assertions are Go-byte preservation
evidence, not C parity evidence.

## Remaining proof/fidelity gaps

C and Go mail remain intentionally separate: `etc/plrmail`/100-byte C blocks
and markers `-1/-2/-3` versus `data/mail`/512-byte Go blocks and markers
`1/-2/2`. Go's current level/price gate is 5/50 versus C's 2/25. The host C
ABI probe reports `sizeof(header_block_type)=104` against the source's
100-byte contract, so the C size guard limits runtime comparison. Multi-block,
corruption/free-list/concurrency matrices, C mail repair, output/object
fidelity, player-save changes, and Phase 6.4 global→struct injection remain
out of scope.

## Durable validation record

All final command/test output, process logs, result/attempt copies, and native
Go mail artifacts are preserved outside self-cleaning directories under:

`/home/zach/dp-mail-initialization-repair-evidence-2026-09-14/`

The final gate record is:

```text
gofumpt -l .                         PASS
git diff --check                     PASS
go build ./...                       PASS
go vet ./...                         PASS
go test ./...                        PASS
go test ./pkg/game/...               PASS
golangci-lint run ./...              PASS
make fidelity-depth                  PASS (4816 total; 4697 proven/delegated, 68 blocked, 51 excluded)
make expected-divergences-check      PASS (26 pins across 10 scenarios)
focused mail races                   PASS
not-here-depth seeds 1,2,3,5,8       PASS; no normalized divergence
production send→restart→receive      PASS; one receipt, second check/receive empty
production delivered-note proof      PASS; sender/body read from note, inventory count 1
```

The corrected focused logs are `go-helper-correction-final.log`,
`go-server-correction-final.log`, and `production-correction-final.log` under
the evidence directory.

The full census command was run fresh after the review corrections with
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, four jobs, and frozen
scenario/fixture/runner inputs. Its final aggregate is
`scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0
timed_out=0`; the only unpinnable case is the established human-cleared
`accuse-noarg-depth` baseline. Five bounded infrastructure-shaped first
attempts (`disarm-depth`, `donate-basic`, `home-depth`, `lines-depth`, and
`spec-proc-tattoo2`) recovered to PASS. The complete corrected run log is
`oracle-regression-correction-final.log`; its mirrored result/attempt
reconciliation found 941 unique results for 941 inputs, with missing=0,
unexpected=0, and duplicate=0. Execution coverage is complete with
`failed=0`, `infra=0`, `timed_out=0`, and `stale=0`.

## Historical review stop

This section records the pre-merge review boundary for #1462. That PR is now
merged; this handoff does not authorize deployment, Go/C mail-format changes,
schema or save fields, Phase 6.4 injection, or a claim of mail fidelity.


## Review correction: disabled dispatch and C object identity

Implementation checkpoint `d6b64449b` adds an explicit boot-configured disabled
state. Failed initialization leaves it set; successful initialization clears
it. For recognized `mail`, `check`, and `receive` commands, the postmaster
emits exactly `Sorry, the mail system is having technical difficulties.\r\n`
and returns false, matching `src/mail.c:484-487` and preserving ordinary
command fallthrough. Unrelated commands receive no diagnostic. This is distinct
from merely clearing the mailbox index or identity hooks. Configuration remains
a boot-only operation before session acceptance; it is not a runtime reload API.

The created mail object's keywords now match C `src/mail.c:574-576`:
`mail paper letter`. The production lifecycle reads it with `read letter`,
rather than inventing a `note` keyword for its test. Short description and
ITEM_NOTE type are independently pinned alongside the keywords. The sibling
field audit records remaining pre-existing discrepancies: the synthetic Go
object lacks C's room description, hold flag, weight 1, cost 30, and load 10.
Those fields and their persistence/interaction proof are separate debt; this
correction does not claim complete mail-object fidelity or modify save format.

Validation at this checkpoint: required formatting/build/vet/full tests/game
tests/lint/depth/divergence gates passed, as did focused mail/server race tests.
The production lifecycle passed in a newly created disposable database
`dp_mail_review_20260914`, with isolated storage and preserved logs under
`/home/zach/dp-mail-review-fix-evidence-2026-09-14/production/` and
`production-isolated.log`. It verifies `read letter` sender/body and exactly one
inventory object after the second empty receive. An initial attempt used an
invalid placeholder database credential and failed before setup; a subsequent
unprivileged database-creation attempt failed, after which the disposable
database was created by the local PostgreSQL administrator. Neither setup
failure exercised game behavior.

The fresh census for this correction completed before #1462 merged. Its
durable checkpoint and final reconciliation are recorded in the [mail
ownership-injection readiness evidence](../../evidence/2026-09-14-mail-ownership-readiness/README.md);
the earlier census above applies to its stated historical checkpoint only.
