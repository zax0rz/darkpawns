# Mail initialization and restart/index repair — 2026-09-14

## Scope and provenance

This is one bounded production repair for the two defects identified in PR
#1461: the missing production `InitMailSystem` call and `scanFile`'s discarded
header decode. It does not claim Phase 6.4 ownership injection, C/Go mail
format convergence, player-save changes, or whole-mail fidelity.

The branch `glm/fix-mail-initialization` started from fresh `origin/main` at
`1dd4794ac`, the merge commit for #1461. PR #1461 was verified merged before
implementation (`2026-09-14T11:55:37Z`). The open-PR overlap query returned no
open PRs. The primary checkout's pre-existing
`docs/specs/tui-setup-wizard.md` edit was preserved and untouched. No file in
`src/` or `darkpawns-c-oracle/` was edited.

The original implementation checkpoint is `411ad4b20`. Review corrections are
in `7992c3b0d`:

```text
fix: keep boot alive when mail is unavailable
```

The final documentation commit is intentionally separate and must remain
production/test-input identical to the corrected checkpoint.

## R5 identity authority and boot ownership

The persistent name↔ID authority was established from the actual call paths:

- C boot builds `player_table` from persisted player records at
  `src/db.c:470-528`; `get_id_by_name` and `get_name_by_id` read that table at
  `src/db.c:2313-2337`.
- Go returning-player login reads the authoritative PostgreSQL row through
  `pkg/db/player.go:270-304` and preserves its `id`/canonical `name` through
  `pkg/session/session_login.go:164-172`.
- Go's existing `db.Database` API exposes `ListPlayerNames` and `GetPlayer`
  (`pkg/db/interface.go:15-18`; `pkg/db/player.go:307-324`). The new
  `cmd/server/mail_identity.go:11-119` uses only those APIs: boot enumerates
  all persistent records, validates positive stable IDs and canonical names,
  resolves names case-insensitively through `GetPlayer`, and refreshes the
  reverse map on an ID miss. It does not consult the online `World` lookup.

This supports an offline recipient and sender-name resolution after restart.
Lookup errors, missing records, invalid IDs/names, and contradictory ID/name
records fail the boot preflight or return the existing mail lookup failure
sentinel; no ID is invented and no incomplete online fallback is substituted.
The identity tests cover case matching, offline reverse lookup, a post-boot
record, list failure, record lookup failure, and a listed-but-missing record:
`cmd/server/mail_identity_test.go:11-108`.

The boot owner is `cmd/server/main.go:208-231`: after `session.NewManager`
has its world/database context and before zone/listener acceptance, the
PostgreSQL-backed path constructs the identity resolver and calls
`game.InitMailSystem`. A missing `data/mail` is created as the existing empty
Go store. A non-ENOENT open error, creation error, short/corrupt block,
stat error, or failed identity preflight disables mail and logs the failure;
the server continues booting under the explicit C-compatible `no_mail`
availability policy. `cmd/server/mail_boot.go:11-26` makes that boundary
testable: it clears partial mail state, returns the initialization error, and
leaves the caller responsible for the nonfatal log. `DP_ALLOW_NO_DB=1` remains
an explicit no-persistence development/oracle configuration: it logs mail
disabled and does not wire an online-only identity source. The JSON player-save
path is not substituted for the production login authority.

## Exact production repair

`pkg/game/mail.go:96-120` now clears stale in-memory index/identity state,
returns the existing scan result from `InitMailSystem`, and disables mail on a
failed scan rather than leaving partial hooks installed.
`pkg/game/mail.go:217-286` now:

1. creates only a genuinely absent `data/mail`;
2. fails closed for other open/create/read/stat errors;
3. reads complete 512-byte blocks with `io.ReadFull`;
4. calls `unmarshalMailHeader` before inspecting `BlockType` and `To`;
5. discards any partial rebuilt index on failure; and
6. retains the current Go markers, offsets, block size, encoding, and written
   bytes unchanged.

The sibling-read audit found no second discarded decode in this repair class:
`readDelete` explicitly unmarshals both header and data blocks after the raw
`readFromFile` helper; the helper itself intentionally only performs I/O.
Short-read/write behavior outside this scan boundary remains a separate
storage-matrix finding and was not changed.

## Lifecycle proof

The Go-native control in `pkg/game/mail_lifecycle_test.go:43-82` now runs send
and restart phases in distinct child processes. It proves the same-process
control remains green and the restart phase now reports:

```text
512 bytes read
Mail file read -- 1 messages
check=... 'You have mail waiting.'
receive=... 'gives you a piece of mail.'
received_body=true
second_check=... 'Sorry, you don't have any mail waiting.'
second_receive=... 'Sorry, you don't have any mail waiting.'
receive_inventory_items=1 mail_remaining=false
```

The production vehicle in `tests/e2e/mail_lifecycle_test.go:67-238` uses:

- disposable native Go fixture storage rooted at a temporary `lib/data/mail`;
- existing PostgreSQL-backed sender and recipient rows seeded before boot;
- a level-34 sender and level-1 recipient, with the recipient never logged in
  during send;
- the live telnet login/menu path, live `set <sender> gold 51`, postmaster
  assignment VNum 3010, and one short body completed with the line editor and
  `@`;
- a real SIGTERM shutdown and a fresh server process against the same mail
  file; and
- recipient login, `check`, `receive`, a second `check`, and a second
  `receive`, followed by `read note` and `inventory` on the delivered object.

The production run observed one 512-byte header with Go marker `1`, the
expected persistent sender/recipient IDs, and exact body `proof-mail-body`
before restart. After restart and receipt the same header had marker `2`,
preserved IDs/body, and PostgreSQL returned the same sender ID/name. The
actual telnet output was one waiting-mail response, one receipt response, and
no-mail responses for both second operations. The delivered note was then read
through the live observation path and asserted to contain the expected sender
and exact body; inventory output contained exactly one `a piece of mail`. The
`CreateMailObject`/note observation additions make that assertion inspect the
player-facing object instead of proxying receipt with the mail file. Header
inspection remains explicitly Go-format preservation evidence; it is not a
claim that the C format or C runtime can cross-read it.

Durable production outputs from the final run are under:

`/home/zach/dp-mail-initialization-repair-evidence-2026-09-14/production-correction-final/`

The complete test transcript is
`production-correction-final.log`; the process logs and pre/post-receive native
mail files are retained in that directory. An earlier TCP-credential attempt
and two bounded note-target debugging attempts were authentication/setup
failures; they were recovered by rerunning with the local PostgreSQL peer DSN
and the unambiguous `read note` target. The final supported persistence run
passed.

## C/Go boundary and remaining gaps

C and Go remain separate native systems:

| boundary | C | Go |
|---|---|---|
| file | `etc/plrmail` | `data/mail` |
| block size | `BLOCK_SIZE=100` | `MailBlockSize=512` |
| markers | header `-1`, last `-2`, deleted `-3` | header `1`, last `-2`, deleted `2` |
| level / stamp | level 2 / 25 | level 5 / 50 |
| message bound | 4096 | 4096 |

The host C ABI probe reports `sizeof(header_block_type)=104` against the
source's 100-byte contract, so the reported C size guard limits runtime
comparison here. No C file, Go constant, normalization, pin, or exclusion was
changed to manufacture parity. Multi-block, corruption/free-list/concurrency
matrices, C mail repair, output/object fidelity, and the remaining player
identity/storage cases remain open. Phase 6.4 global→struct ownership
injection is explicitly not started.

## Validation and measured delta

At corrected checkpoint `7992c3b0d`, `git diff origin/main...7992c3b0d --stat`
measured 13 changed files, `+1154/-32` lines. The final documentation commit
only changes the evidence/roadmap/handoff prose below; production code, tests,
fixtures, scenarios, and runner inputs remain identical to this checkpoint.
The changed-file coverage is:

| file | coverage citation |
|---|---|
| `cmd/server/main.go` | production boot owner and failure gate, `:208-231` |
| `cmd/server/mail_boot.go` | nonfatal C-compatible mail disposition, `:11-26` |
| `cmd/server/main_test.go` | nonfatal unusable-store boundary, `:121-140` |
| `cmd/server/mail_identity.go` | persistent identity resolver, `:11-119` |
| `cmd/server/mail_identity_test.go` | authority and lookup-failure tests, `:11-108` |
| `pkg/game/mail.go` | init result, disabled failure state, and decoded scan, `:96-120`, `:217-286` |
| `pkg/game/mail_test.go` | unusable-store non-overwrite/disabled-hooks test, `:27-57` |
| `pkg/game/look.go` | delivered note observation, `:918-947` |
| `pkg/game/mail_lifecycle_test.go` | helper restart and once-only regression, `:43-82`, `:173-199` |
| `tests/e2e/mail_lifecycle_test.go` | production lifecycle and direct note proof, `:67-238` |

The following final gates passed from the checkpoint, with logs preserved in
`/home/zach/dp-mail-initialization-repair-evidence-2026-09-14/`:

```text
gofumpt -l .                         pass
git diff --check                     pass
go build ./...                       pass
go vet ./...                         pass
go test ./...                        pass
go test ./pkg/game/...               pass
golangci-lint run ./...              pass
make fidelity-depth                  pass
make expected-divergences-check      pass
focused mail races                   pass
not-here-depth seeds 1,2,3,5,8       pass; no normalized divergence
corrected production lifecycle        pass; sender/body read from note, inventory count 1
```

The corrected focused logs are `go-helper-correction-final.log`,
`go-server-correction-final.log`, and `production-correction-final.log`.

The fresh full `make oracle-regression` census on corrected checkpoint
`7992c3b0d` completed with this aggregate:

```text
scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The sole unpinnable case was the established human-cleared
`accuse-noarg-depth` baseline. Five infrastructure-shaped first attempts
(`disarm-depth`, `donate-basic`, `home-depth`, `lines-depth`, and
`spec-proc-tattoo2`) recovered within the runner's bounded retry budget; the
final aggregate has `infra=0` and no flaky-red content result.

The complete corrected run log is
`oracle-regression-correction-final.log`, with its mirrored result/attempt
files under `oracle-regression-correction-final/preserved/`. Reconciliation
found 941 unique result files for 941 scenario inputs, with missing=0,
unexpected=0, and duplicate=0. The preserved status tally is exactly 931
`PASS`, 9 `EXPECTED`, and 1 `UNPINNABLE`; execution coverage is complete and
the only nonzero exit disposition is the established human-cleared baseline.


## Final review correction — disabled dispatch and keywords

At implementation checkpoint `d6b64449b`, failed initialization explicitly
marks mail disabled. Recognized postmaster commands emit C's exact technical
failure diagnostic and return false; unrelated commands remain silent. A
successful initialization clears the disabled state. The state is configured
at boot, not through a supported runtime reload operation.

Mail keywords now use C's `mail paper letter`; the production regression uses
`read letter`. Required local gates and focused race checks passed. The new
isolated PostgreSQL production lifecycle passed, including actual delivered
sender/body and one inventory object after a second empty receive. Evidence:
`/home/zach/dp-mail-review-fix-evidence-2026-09-14/production-isolated.log`,
`production/`, and `race.log`. The dated handoff records the sibling object-field
debts and setup-only failed attempts, separately from game behavior.

This checkpoint supersedes the earlier implementation for final review.
The fresh full census completed before #1462 merged. Its durable checkpoint,
identity reconciliation, retry inspection, final tally, and exit-2 baseline
disposition are recorded in the [mail ownership-injection readiness
evidence](../2026-09-14-mail-ownership-readiness/README.md). The final tally is
`931 PASS / 9 EXPECTED / 1 UNPINNABLE`, with `failed=0`, `infra=0`,
`timed_out=0`, and `stale=0`; prior census tallies above remain historical
evidence for their named checkpoints.
