# Mail initialization/restart repair handoff — 2026-09-14

## Disposition

One reviewable, unmerged repair PR is being prepared on
`glm/fix-mail-initialization`. PR #1461 was verified merged before work began;
its merge is `1dd4794ac`. The implementation checkpoint is
`411ad4b20` (`fix: initialize persistent mail and rebuild index`). The final
head will be the documentation-only follow-up commit after validation; the
production code, tests, fixtures, scenarios, and runner inputs remain frozen
at the checkpoint.

The primary checkout's pre-existing `docs/specs/tui-setup-wizard.md` edit was
not touched. There are no edits to `src/` or `darkpawns-c-oracle/`. The open
PR overlap query was empty.

## Identity authority and boot owner

R5 tracing establishes the existing PostgreSQL player row as the Go
name↔ID authority: `pkg/db/player.go:270-324` supplies case-insensitive
`GetPlayer` plus `ListPlayerNames`, while `pkg/session/session_login.go:164-172`
loads and preserves the row's stable ID/canonical name. The C call path is
`src/db.c:470-528` (`player_table`) and `src/db.c:2313-2337`
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
existing empty store, while unusable storage or identity preflight failures
abort startup. `DP_ALLOW_NO_DB=1` remains explicitly non-persistent and logs
mail disabled; it cannot safely provide restart-stable offline identities and
is not used for the production vehicle.

## Exact repair and proof

`pkg/game/mail.go:96-105` returns the initialization result and resets stale
index state. `pkg/game/mail.go:217-286` only creates an absent file, reads full
512-byte blocks, unmarshals the current Go header before checking its marker
and recipient, and fails closed on open/create/read/stat/partial/corrupt
storage. Existing Go markers, offsets, block size, encoding, and written bytes
are unchanged. The sibling audit found `readDelete` already explicitly
decodes header/data blocks after `readFromFile`; no sibling discarded-decode
repair was broadened here.

The helper regression at `pkg/game/mail_lifecycle_test.go:43-82,173-199`
keeps the same-process control and now proves a distinct-process restart:
one indexed message, waiting check, one receipt with body, one inventory item,
then empty second check/receive and no remaining mail.

The production vehicle at `tests/e2e/mail_lifecycle_test.go:67-204` seeds a
level-34 sender and persistent level-1 recipient before boot, leaves the
recipient offline, uses the real telnet login/menu path, funds with live
`set`, composes one short single-block body through the editor, sends SIGTERM,
restarts the server against the same temporary `data/mail`, logs the recipient
in, checks/receives, and repeats check/receive to prove no duplicate. It
asserts persistent sender ID/name and exact body `proof-mail-body` before and
after receipt. Header assertions are Go-byte preservation evidence, not C
parity evidence.

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
```

The full census command was run fresh after the production behavior change
with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, four jobs, and
frozen scenario/fixture/runner inputs. Its final aggregate is
`scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0
timed_out=0`; the only unpinnable case is the established human-cleared
`accuse-noarg-depth` baseline. Seven bounded infrastructure-shaped first
attempts recovered to PASS. The preserved full-run log is
`oracle-regression-final/oracle-regression.log`; its result-name reconciliation
had no unexpected or duplicate names, and the one final `yuball-depth` result
preservation race was closed by the tight-poll same-input recovery under
`oracle-yuball-recovery-2026-09-14-tight/`. Execution coverage remains the
runner's complete 941/941 aggregate with `failed=0`.

## Review stop

Do not merge, deploy, change the Go/C mail format, add schema or save fields,
or begin Phase 6.4 injection. Human review is required at this bounded repair
boundary.
