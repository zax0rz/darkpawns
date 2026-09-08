# Entry lifecycle proof expansion — 2026-09-08

Scope: the repaired entry lifecycle only. All PostgreSQL checks used an
isolated schema on a disposable loopback cluster at `127.0.0.1:55439`; no
production database or record was touched.

## Proof commands and outcomes

| Boundary | Command | Outcome |
|---|---|---|
| PostgreSQL state machine | `DP_ENTRY_TEST_DATABASE_URL='postgres://zach@127.0.0.1:55439/postgres?sslmode=disable' /usr/local/go/bin/go test ./pkg/session -run '^TestEntry' -count=1 -v` | PASS: case-folded lookup, N retry, accepted-stats persistence, menu disconnect/reconnect, password retries, real unique collision, lookup/count/entry-save failures, legacy ambiguity, two concurrent creation sessions, and God/mortal draw boundaries. |
| God/mortal PostgreSQL persistence | Same DSN; `/usr/local/go/bin/go test ./pkg/session -run '^TestEntryDatabasePersistsGodAndMortalEntry$' -count=1 -v` | PASS: the first persisted character is God, the second is mortal, IDs differ, God level persists, and the mortal row is level zero at acceptance then level one after menu entry. |
| WebSocket JSON boundary | Same DSN; `/usr/local/go/bin/go test ./pkg/session -run '^TestEntryWebSocket(SavedIdentityAndMenuResume|NewCharacterPersistsAtMenu)$' -count=1 -v` | PASS: real local WebSocket; `aiko`/`AIKO` resolve the same saved row, and a new mortal's accepted level-zero stats survive a menu disconnect and reconnect without a duplicate row. |
| Telnet boundary | Same DSN; `/usr/local/go/bin/go test ./pkg/telnet -run '^TestEntryTelnetSavedIdentityAndMenuResume$' -count=1 -v` | PASS: real local TCP listener; case-variant lookup, password, MOTD/menu, room output, and one persisted row. |
| Browser-owned protocol behavior | `node --test scripts/entry-client.test.mjs` | PASS: 4 tests. The shipped handler waits for server name lookup, keeps secret input out of terminal output, renders exact server prompts, and handles secret backspace. This is terminal/WebSocket-double proof, not visual browser proof. |
| Depth manifest | `make fidelity-depth` | PASS: 4,790 cases; 4,673 proven/delegated, 66 blocked, 51 excluded. Entry surface: 15/29 proven; remaining rows are explicit gaps. |
| Creation/retry oracle | `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /tmp/dp-oracle-diff-entry --scenario character-creation-name-retry --seed {1,2,3,5,8}` | PASS at all five seeds: no normalized divergence. |
| Oracle branch inspection | Same scenario with `--seed 1 --show-oracle` | PASS; the retry block ran and showed the C `Aiko` confirmation, MOTD, menu, and first-entry transcript. |
| Assembled telnet smoke | `DP_ALLOW_NO_DB=1 /usr/local/go/bin/go test ./tests/e2e -run '^TestTelnetSmoke_(CharacterCreation|GuestEntersWorld)$' -count=1 -v -timeout 150s` | PASS. The initial run without `DP_ALLOW_NO_DB=1` was a reproducible harness/configuration failure because the server correctly refuses a dead DB by default. |
| God smoke oracle | `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /tmp/dp-oracle-diff-entry --scenario god-harness-smoke --seed {1,2,3,5,8}` | PASS at all five seeds; no normalized divergence. Seed 1 was also run with `--show-oracle`. |

## Full oracle census

After the focused checks, the 935-scenario corpus was run with:

```text
PATH=/usr/local/go/bin:$PATH \
ORACLE_REGRESSION_GO=/usr/local/go/bin/go \
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
ORACLE_REGRESSION_JOBS=16 ORACLE_REGRESSION_TIMEOUT=240s \
make oracle-regression
```

The driver completed with 920 passed, 9 expected ledger-backed divergences, 1
unpinnable ledger-backed divergence, 5 actionable failures, and 0
infrastructure failures or timeouts. None of the entry-lifecycle scenarios
were among the five failures.

The census ran before the later narrow pre-world registration repair described
below. The final code was revalidated through the focused creation and God
oracle seeds, PostgreSQL transport tests, the full Go suite, and the lint and
race gates; the census was not rerun because that repair is confined to the
transport/session cleanup path.

The five reproducible failures were (the original unrelated classification was
corrected by the cleanup review below):

- `color-depth` — color level says `complete` in Go versus `off` in C
  (fingerprint `bc4362589df9a4e5dcc4f03ffd44ac7bcad16fa6aec147907346a12b2eaec954`).
- `informative-residual-depth` — the same color-level text differs in the
  `toggle` report (fingerprint `f15a76bc55350a15333a555f16d83850a3b0623ecd394072654a290ae0528e2d`).
- `mortal-prefs-batch` — the standalone color command has the same color-level
  mismatch (same `color` fingerprint as `color-depth`).
- `toggle-cluster` — both toggle reports differ only in color level
  (fingerprints `42c3f6b3bbff3472390104b19075715bb74d4467cab6685e0a9de48a95ab4753`
  and `f065021928f8babe26949d120c9788ac982fd3de92cfa099ca182ef98357ca39`).
- `wizard-valid-reports-depth` — Go reports `C1 C2` in the peer's `PRF`
  field where C reports none (fingerprint
  `4223dd8055cf00959660df57ab01f442b24f307fc7e53e663aad86ba43c91e70`).

The census also reported `accuse-noarg-depth` as unpinnable because its
divergence contains run-varying bytes; it remains a human-clearance item in
the existing ledger. At this checkpoint these findings were classified as unrelated rather than
repaired. That classification was incorrect: see the cleanup review below.

## C call-path anchors

- `src/interpreter.c:1743-1815`: `CON_GET_NAME` validates, calls
  `load_char`/`find_name`, and retries through the complete lookup path on N.
- `src/db.c:2342-2405`: `load_char` reads the saved identity using the C
  player-table path; `src/db.c:2366-2398` is the corresponding `save_char`
  persistence path.
- `src/interpreter.c:1860-1939`: existing identities enter password handling,
  then MOTD/menu after password and duplicate/restriction checks.
- `src/interpreter.c:2123-2132`: accepted stats run `init_char` and `save_char`
  before MOTD.
- `src/interpreter.c:2165-2217`: menu option 1 enters the world and invokes
  `do_start` only when the loaded level is zero.
- `src/class.c:501-565`: `do_start` supplies the level-zero mortal bootstrap
  and its class/gear effects.
- `src/db.c:3006-3048`: `init_char` crowns the first player and consumes the
  sex-dependent body draws.
- `src/utils.c:104-120`: `str_cmp` is case-insensitive.

## Confirmed Go lifecycle repair

The transport test exposed a real pre-world-entry gap: after accepted stats,
the Go session had a persisted level-zero player but no registered
`playerName`, so a disconnect at MOTD/menu could bypass the manager cleanup
save path. The repair registers the accepted candidate before MOTD/menu, makes
same-session registration idempotent, and removes the registered candidate
through normal cleanup on later entry failure; `TestEntryWebSocketNewCharacterPersistsAtMenu`
now proves the resulting disconnect/reconnect path against PostgreSQL. This
does not broaden the proof to the still-open menu, restriction, duplicate, or
visual-browser matrices.

## Boundaries that remain open

This proof does not certify name-validation combinations, deleted-name reuse,
all password policy branches, site/wizlock restrictions, duplicate-session
reconnect/usurp/unswitch behavior, full creation-choice matrices, exact
oracle stats/gear values, frozen/unhealthy entry, every identity consumer, or
real visual browser rendering. The focused no-database oracle cannot certify
PostgreSQL persistence; the transport and database tests above are the
separate evidence for those boundaries (R5f).

The proof is therefore complete at the tested boundaries, with the gaps above
remaining explicit rather than inferred away.

## Cleanup review correction

The follow-up review reproduced the five color failures as stale setup input:
old port fixtures sent an extra Y between password confirmation and the intended
N color answer. The repaired server correctly interpreted Y as color-on. The
fixture sweep removes that obsolete input; it does not change color semantics or
normalization. See cleanup-review.md for the audit and regression results.

The original telnet test above was renamed
`TestEntryTelnetSavedIdentityEntersWorld`: it only proves one saved login.
`TestEntryTelnetNewCharacterDisconnectResumes` now separately proves actual TCP
EOF at MOTD and menu, same-ID/stat reconnect, and playing-only linkdead retention.
The review also found cleanup retried a transiently failed first-entry save with
an already-advanced candidate; that candidate is now discarded before teardown.
