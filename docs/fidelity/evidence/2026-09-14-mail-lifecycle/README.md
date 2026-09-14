# Mail initialization and lifecycle proof — 2026-09-14

## Scope and provenance

This is a bounded proof of the mail boot boundary and one short
send→restart→receive attempt. It does not claim whole-mail fidelity, Phase
6.4 completion, production initialization, or readiness for injection.

The worktree started from fresh `origin/main` at `d5328ce1b`, which contains
the weather repair from #1457 (`eac85ac30379879633167611b858016414287f97`).
The open-PR query returned no rows. The primary checkout's pre-existing
`docs/specs/tui-setup-wizard.md` edit was not touched. `src/` and
`darkpawns-c-oracle/` were read-only.

The governing R5 trace was recorded in:

`/home/zach/dp-mail-lifecycle-evidence-2026-09-14/call-path-2026-09-14.log`

## Initialization boundary

Go's only production implementation of mail boot is
`pkg/game/mail.go:95-100`:

```text
InitMailSystem(nameFunc, idFunc) -> assign worldNameFunc/worldIDFunc -> scanFile()
```

The non-test search has exactly the definition and no caller. `cmd/server`
boots `ResetTime`, parses the world, constructs `World`, constructs the
session `Manager`, wires other manager/world hooks, and starts zone resets;
there is no `InitMailSystem` call. The earliest sensible ownership point is
after `session.NewManager(gameWorld, dbIface)` has established the live world
identity context and before long-lived listeners accept players. This proof
does not add that call: the correct ID/name universe (online world versus
persistent database identities) is still a fidelity decision.

The focused fallback test proves the uninitialized state directly:

```text
GetIDByName("Recipient") == -1
GetNameByID(202) == "Player(202)"
```

Go postmaster registration exists (`pkg/game/postmaster.go:7-32`) and VNum
3010 is assigned in `pkg/game/spec_assign.go`. Session dispatch checks the
room spec cache and invokes the assigned mob procedure in
`pkg/session/commands.go:694-712`. Composition completion is intercepted by
`pkg/session/session_login.go:363-378`; disconnect cleanup cancels in-progress
writing. The live telnet probe reached that real path and observed the exact
stamp-affordability response before lookup/composition.

The second boundary is in `pkg/game/mail.go:226-237`: `scanFile` reads into a
temporary `marshalMailHeader(&nextBlock)` byte slice but never calls
`unmarshalMailHeader`. On a new process, `nextBlock.BlockType` and `To` stay
zero, so a valid header is not indexed. The helper restart output is the
durable characterization of this known defect, not an unexplained failing
test.

## C comparison

C boot calls `scan_file()` from `src/db.c:366-371`. `src/mail.c:221-251`
opens `MAIL_FILE`, creates it if absent, indexes `HEADER_BLOCK`, records
`DELETED_BLOCK` positions, records the end offset, and returns failure for a
non-block-aligned file; boot then sets `no_mail=1`. The C postmaster at
`src/mail.c:476-500` rejects non-player/no-descriptor callers, accepts only
`mail`, `check`, and `receive`, and emits the technical-difficulties message
when `no_mail` is set. C send/check/receive are at `:503-595`, with
`string_write` completion, `store_mail`, `has_mail`, `read_delete`, note
creation, and one receive loop.

The production C and Go storage contracts are not interchangeable:

| boundary | C | Go | proof consequence |
|---|---|---|---|
| file | `etc/plrmail` | `data/mail` | separate native fixtures only |
| block size | `BLOCK_SIZE=100` | `MailBlockSize=512` | no cross-read is attempted |
| markers | header `-1`, last `-2`, deleted `-3` | header `1`, last `-2`, deleted `2` | marker semantics are not assumed compatible |
| send level / price | level `2` / `25` | level `5` / `50` | production probe uses the current Go gate |
| max message | `4096` | `4096` | not exercised beyond one short block |
| boot failure | `no_mail=1` and player-facing technical-difficulties path | `scanFile` returns `bool`, but no production caller or dispatch wiring is present | Go failure behavior is not claimed C-equivalent |

A native C layout probe in the disposable fixture directory reports
`sizeof(header_block_type)=104` and `sizeof(data_block_type)=104` on this
host while the source contract remains `BLOCK_SIZE=100`; C `store_mail`
would hit its size guard here. The C file is therefore a native-format ABI
fixture/probe, not a claimed successful C runtime lifecycle.

## Finite result table

| lifecycle stage | actual path exercised | C evidence | Go evidence | result | smallest next action |
|---|---|---|---|---|---|
| boot scan and failure boundary | Go source trace; explicit helper `InitMailSystem` in a child process | `db.c:366-371` calls `scan_file`; `mail.c:221-251` sets the C failure boundary | no non-test `InitMailSystem` caller; explicit reopen logs `512 bytes read` and `0 messages` | **blocked/divergent** | authorize one Go boot-owner + scan-decode repair, preserving the current Go format until separately decided |
| postmaster registration/dispatch | real server boot/login/telnet room 1204; helper assigned VNum 3010 | `spec_assign.c:214`; `mail.c:476-500`; interpreter entries are `do_not_here` | production room rendered “The postman” and `mail` returned the stamp gate; helper `GetMobSpec(3010)` invoked the registered procedure | **proven to dispatch; lifecycle blocked** | keep the next repair focused on initialization/indexing, not dispatch refactoring |
| send preconditions | production `mail Recipient body`; helper with level/coins supplied | `mail.c:509-531` level, argument, price, lookup order | production stopped at exact current Go price/affordability output; helper passed level 5 and 51 coins | **production blocked at affordability** | use a reviewed production fixture/identity setup in the next repair proof |
| composition completion | helper `PostmasterSendMail` → `HandleMailInput(body)` → `HandleMailInput("@")` | `mail.c:533-545` → `string_write`; completed text reaches `store_mail` | `Mail sent`, one short body, 512-byte isolated `data/mail` | **proven helper-only** | retain as regression while repairing the production owner |
| persist one block | disposable native-format fixture | `store_mail` writes fixed 100-byte blocks under its ABI guard | Go fixture hash `44d934313abb0429bb59053b2a9396efb5b549397e4f01335444fd4420755e1d`, size 512 | **proven helper-only** | no storage-format change in the next task |
| real restart/reopen | separate child process with the same isolated Go CWD | C scan indexes header blocks on boot | `InitMailSystem` returns; `scanFile` reports `0 messages`; recipient `check` and `receive` both report no mail | **blocked at scan/index** | decode the existing Go header before claiming restart readiness |
| recipient check/receive | same-process helper with explicit initialization | `has_mail` → `read_delete` → note object; receive loops until empty | exact check waiting output, exact receive output, body present in one inventory item | **proven helper-only** | rerun through production after initialization repair |
| consume once | same-process helper then `hasMail` | C `read_delete` removes/deletes indexed blocks | `consumed_once=true`, `mail_remaining=false` | **proven helper-only** | preserve the once-only assertion in the repair proof |

Overall disposition: the requested production lifecycle is **blocked**, not
proven. The narrow helper vehicle proves same-process storage, composition,
delivery, and one-time consumption, but it does not prove production boot or
restart. The real process restart/reopen vehicle proves the Go scan/index
boundary is not ready.

## Durable proof outputs

The focused run and isolated native fixtures are outside self-cleaning test
directories:

- `/home/zach/dp-mail-lifecycle-evidence-2026-09-14/go-helper-trace-2026-09-14.log`
- `/home/zach/dp-mail-lifecycle-evidence-2026-09-14/production-mail-boundary-2026-09-14.log`
- `/home/zach/dp-mail-lifecycle-evidence-2026-09-14/call-path-2026-09-14.log`
- `/home/zach/dp-mail-lifecycle-evidence-2026-09-14/go-native/data/mail`
- `/home/zach/dp-mail-lifecycle-evidence-2026-09-14/c-native/etc/plrmail`

The Go test sets `DP_FIXED_TIME` for the vehicle, but the current
`storeMail` timestamp is `time.Now().Unix()`; the proof therefore asserts
body, file size, receipt output, and consumption state, not a timestamp.

## Oracle/depth boundary

No oracle scenario, fixture, manifest, or runner input changed. The current
production sources and oracle inputs are identical to the tested #1457
checkpoint `eac85ac30379879633167611b858016414287f97`; the established census
is reused under the handoff rule, not rerun for this test-only/docs-only
proof. Its healthy aggregate is:

```text
scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The only unpinnable row is the established human-cleared
`accuse-noarg-depth` baseline. Fresh focused mail tests and the validation
gates below are the evidence for this PR's changed surface; the census is
not a claim of mail coverage.

Final gate records are preserved in the same evidence directory:

```text
gofumpt -l .                         pass
git diff --check                     pass
go build ./...                       pass
go vet ./...                         pass
go test ./...                        pass
go test ./pkg/game/...               pass
golangci-lint run ./...              pass
make fidelity-depth                  pass (4816 total; 4697 proven/delegated, 68 blocked, 51 excluded)
make expected-divergences-check      pass (26 pins across 10 scenarios; pins OK)
```

The final command logs are `gofumpt-final-2026-09-14.log`,
`git-diff-check-final-2026-09-14.log`, `go-build-final-2026-09-14.log`,
`go-vet-final-2026-09-14.log`, `go-test-all-final-2026-09-14.log`,
`go-test-game-final-2026-09-14.log`, `golangci-lint-final2-2026-09-14.log`,
`fidelity-depth-2026-09-14.log`, and
`expected-divergences-check-2026-09-14.log`.
