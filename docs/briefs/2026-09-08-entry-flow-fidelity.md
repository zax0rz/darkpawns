# Entry-flow fidelity: verified failure and implementation contract

Status: diagnosis complete; implementation pending. R1/R3/R4/R5c/R5e.

**Cite:** `nanny`, `find_name`, `load_char`, `str_cmp`, `init_char`,
`perform_dupe_check` in `src/interpreter.c`, `src/db.c`, `src/utils.c`.

This corrects `/home/zach/dp-bug-aiko-2026-09-07.md`. Its double-finalize
explanation is contradicted by production timestamps and a database-backed
reproduction. Do not implement a fix based on that explanation.

## Verified production evidence

Read-only inspection on 2026-09-07 EDT / 2026-09-08 UTC:

- Running binary build metadata: `11a11ead41af01553ec06598919742a1319cf12b`,
  `vcs.modified=false`; matches the audited checkout.
- Binary SHA-256: `42beaa48eacca120214ba3f9b06b2ef950d49e0faf4b7be6c47e8a078cb07096`.
- Deployed `/srv/hugo/js/client.js` and checked-in client have the same SHA-256:
  `597776e1405c5054f5cc400498ed7ea8250ff7adbf2940e533a93b3580caeb46`.
- `players` uses `name character varying` with exact-name unique index
  `players_name_key`. There is no folded-name unique index.
- `Aiko`, ID 61, level 1, created/updated `2026-05-27 14:48:58.665649`
  (database timestamp without time zone).
- `aiko`, ID 69, level 1, created `2026-09-07 22:43:29.364812`, updated
  `2026-09-07 22:44:51.839083` (same database timestamp type).
- Logs: `22:40:59.118-04:00` duplicate creation rejection for `Aiko`;
  `22:43:29.365-04:00` successful world entry and state clearing for `aiko`.
- Three case-folded identities have two records each: `aiko`, `test`, `brenda69`.

No production records, service settings, binaries, or site files were changed.
No password material was queried. Existing collisions require explicit data
resolution before enforcing a folded unique constraint. Never pick an arbitrary
row, merge inventories, delete a character, or silently transfer ownership.

## Confirmed failure chain

1. The shipped browser collects password/new-character/confirmation locally
   before sending a login message. Telnet does a database lookup first instead.
2. `DB.GetPlayer("aiko")` misses a saved `Aiko` under the actual schema.
3. `handleCharInput`'s `get_name` branch accepts `Aiko` after N without a saved
   player lookup. `ValidName` only checks format/bans/online presence.
4. Stats acceptance shows the MOTD without saving. First menu 1 makes exactly
   one `CreatePlayer` call. PostgreSQL rejects the already-existing `Aiko`.
5. `completeCharCreation` assigned `s.player` before that failed insert. A
   second menu 1 takes `enterReturningPlayer` and adds this unsaved candidate
   to the world. The session remains unauthenticated. This is a confirmed
   invalid lifecycle state, not evidence of authenticated account takeover.

Local reproduction used synthetic records/passwords in isolated PostgreSQL
schemas. It drove `MsgCharInput` through `handleMessage`, including both menu
attempts. Counts: zero inserts before menu 1, one after, still one after retry.

## C contract and implementation boundaries

- `CON_GET_NAME` validates then calls `load_char -> find_name -> str_cmp`.
  `str_cmp` folds case. `CAP` uppercases the first character in-place, including
  the name-confirmation text. Preserve C display behavior separately from the
  identity key; do not assume C lowercases every display name.
- Existing live records go to `CON_PASSWORD`; unknown/deleted records to name
  confirmation. N returns through the complete name lookup, not a creation-only
  shortcut. New-password collection begins only after Y.
- Wrong existing passwords retry, then close on the configured attempt limit.
  Empty input closes. Current Go closes immediately on a wrong password.
- Successful existing authentication applies site/wizlock/duplicate checks,
  then the appropriate MOTD and menu. Audit `perform_dupe_check` separately for
  reconnect/usurp/unswitch; a same-name world lookup is not sufficient proof.
- `CON_ROLLABL2` Y calls `create_entry` as needed, `init_char`, and `save_char`
  before MOTD. A character must survive disconnect from the menu. World entry
  happens at menu 1. `do_start` runs only for level zero; first-player God skips
  it. Preserve the exact RNG order and God/mortal distinction (R3d).
- Database failures must not be treated as unknown names or leave a candidate
  eligible for returning-player entry. Existing `RecordToPlayer` failure also
  falls back to creation and belongs to this audit.
- All player transports should consume the same server-owned entry states.
  Audit browser, telnet, structured/agent callers, secret input, and message
  dispatch. Browser rendering currently adds options and `>` beyond server text.
- Persistence identity must be consistent across lookup, uniqueness, sessions,
  world maps, lockouts, saved-name lists, agent-key ownership, and updates.
  Do not broaden case-insensitive matching to passwords.

## RED evidence and reproducible commands

Evidence is in `docs/fidelity/evidence/entry-2026-09-08/`. These are pre-fix
failures, not accepted divergences or green coverage. The branch inventory is
`docs/fidelity/depth/entry.tsv`; all unproven cases remain blocked.

Use a disposable local PostgreSQL cluster. The Go tests reject non-loopback
URLs, create a unique schema per test, and remove their own schema at teardown.

```bash
DP_ENTRY_TEST_DATABASE_URL='postgres://zach@127.0.0.1:55439/postgres?sslmode=disable' \
  go test ./pkg/session -run '^TestEntry' -count=1 -v
node --test scripts/entry-client.test.mjs
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario character-creation-name-retry --seed 1 --show-oracle
```

The four PostgreSQL tests fail on identity lookup, saved-name retry,
save-before-menu, and failed-insert candidate entry. The browser test executes
the actual shipped handler with terminal/WebSocket doubles and fails because
no name message is sent before the browser prints `Password:`. It is not a
full browser E2E test.

The fresh-name oracle retry reaches C's full creation, menu, room, and birth
sequence. Its seed-1 diff exposes lowercase initial confirmation and raw `&c`
MOTD markup on the Go side. It cannot prove saved-name collision behavior:
the standard Go oracle vehicle lacks PostgreSQL persistence. Existing matching
creation input streams do not test the browser's independent preamble.

Diagnosis checkpoint validation: `go test ./...` passes with the PostgreSQL
opt-in unset (the four integration contracts skip); the explicit PostgreSQL run,
browser contract, and oracle retry are RED as recorded. The depth-manifest
validator passes. No implementation or commit has been made at this checkpoint.

## Next bounded goals

1. Repair shared entry state, name identity, and creation persistence/entry
   boundaries; turn focused tests green. Adapt the incident test to the corrected
   interactive route rather than retaining obsolete keystrokes as a golden flow.
2. Expand the branch inventory into oracle, database, transport and browser
   proof; seeds 1,2,3,5,8; God/mortal draw counts; full census and repository gates.
3. Review collision resolution with the owner using concrete proposed record
   changes, then deploy reviewed main per `DEPLOYMENT.md`. Website source changes
   use the documented Astro asset pipeline, build, deletion dry-run and target.

A green telnet oracle cannot certify browser-authored output. Add proof at each
actual entry boundary; never normalize away differing dialogues or teach a
compatibility shim the old broken flow. Track broader findings rather than
silently enlarging one repair goal indefinitely (R5b/R5c).

## Repair checkpoint — 2026-09-08

The bounded repair is implemented locally on `codex/entry-flow-fidelity`.
Browser and telnet name entry now route through the shared session state machine.
Name retry repeats persistence lookup; saved identities resolve case-insensitively,
and ambiguous legacy identities fail closed. Password bytes retain whitespace.
Creation persists at stat acceptance, before MOTD/menu, and failed inserts cannot
leave an enterable candidate. Saved level-zero characters resume first entry;
restore no longer consumes constructor RNG draws (R1/R3/R5e).

The browser renders server prompts and waits for each entry reply before draining
pasted input, preserving secret echo suppression. Its authored client now follows
the Astro asset pipeline. Existing pinned xterm dependencies are bundled locally:
the built-page check found the CDN scripts blocked by the server's self-only CSP.

Validation passed: build, vet, all Go tests, lint (0 issues), formatting, explicit
local PostgreSQL TestEntry suite, four Node handler contracts, site build, and
character-creation-name-retry oracle seeds 1,2,3,5,8 with no normalized divergence.
A built-page browser smoke test created Aiko through N/name retry, entered the
world, reloaded, and authenticated as AIKO into the same character. This disposable
world's first player was God; it is not mortal browser depth proof. Historical
RED evidence above is retained; green evidence is in the same evidence directory.

This is not full entry certification. The manifest preserves unproven validation,
restrictions, duplicate-session, deletion, menu, failure-injection, identity-consumer,
color matrix and RNG/gear branches. Full oracle census has not run. Browser smoke
and terminal doubles do not establish byte equality across both transports.
The local smoke also displayed repeated bulletin boards in the fixture room;
its cause is uninvestigated and outside this repair.

Production data and deployment are unchanged. The folded unique index will reject
the three known colliding identities (`aiko`, `test`, `brenda69`); no automated
winner selection or deletion was implemented. Next: bounded proof expansion,
then a concrete owner-reviewed collision proposal and deployment review.

## Lifecycle proof and cleanup checkpoint

The follow-up proof and cleanup are recorded in
[cleanup-review.md](../fidelity/evidence/entry-2026-09-08/cleanup-review.md).
Real TCP tests now cover disconnect at MOTD and menu, same-ID/stat reconnect,
and playing-only linkdead retention. One-shot entry-save failure leaves the
accepted level-zero row intact. The 1,757 stale extra-Y setup inputs in 932
oracle fixtures were removed, resolving all five color-report failures.

The complete census rerun reports 925 passes, nine expected divergences, one
standing unpinnable case, and zero unexpected failures across 935 scenarios.
The exit-2 clearance requirement remains; it has not been waived. Other blocked
entry branches remain explicit in the manifest. Production collision resolution
and deployment remain separate from this repair.
