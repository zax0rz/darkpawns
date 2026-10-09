# DP-1371 D7: mudlog PR 2 — admission, ban and close_socket (stop tier)

**Local edits, stop tier.** Base `origin/main` `c70c598b2`. R1/R2/R4/R5e/R5h.

| Commit | Contract |
|---|---|
| `31e0e4398` | C's `d->host` (the padded quad) at every consumer; `nameserver_is_slow` gates the lookup; `DNS lookup failed on %s.` |
| `0716a569b` | `comm.c:1571-1575` refusal bytes, close, then `Connection attempt denied from [%s]` |
| `1317cb0b0` | `ban.c:205-209` and `ban.c:237-244`, each in C's order relative to the acknowledgement |
| `38b5b000e` | `close_socket`'s two non-playing arms (`Losing player:` / `Losing descriptor without char.`) |

## 1. The host-string class

C's `new_descriptor` writes **either** a resolved name **or** the zero-padded
dotted quad into `d->host` (`src/comm.c:1534-1563`). Because
`nameserver_is_slow` ships `YES` (`src/config.c:206`), C never calls
`gethostbyaddr` (`src/comm.c:1527-1529`) and the padded quad is every
connection: `001.002.003.004`, not `1.2.3.4`. The port printed the raw IP at
every admission site — an R1 class, not a single line.

- `session.CConnectionHost` is the one padding rule; `MudHost()` is C's
  `d->host` for a session (the stored resolved name, else the padded quad).
  `usersHost` (act.informative.c:2093-2094) and `formatLastHost` now share it
  instead of padding a second and third time.
- Every consumer of `d->host` moved: the two reconnect lines
  (`interpreter.c:1640`, `:1654`), the entry refusals (`:1825`, `:1837`,
  `:1878`, `:1902`, `:1911`), has connected (`:1924`) and new player (`:2154`).
- **Ban matching changed with it.** C checks `isbanned(newd->host)`,
  `wildhost` (`%03u.%03u.%03u.*`) and `double_wild` (`%03u.%03u.*.*`) at
  accept (`src/comm.c:1541-1548`, `:1567-1569`), and `isbanned(d->host)` alone
  at each nanny boundary (`src/interpreter.c:1822`, `:1896`). `isbanned` is a
  `strstr` substring test, so `ban all 127.000.000.*` bans the whole /24 and
  `ban all 127.0.0.1` matches nothing. The port had it the other way round.
  `session.AcceptBanLevel` implements the accept-time union; the session's
  entry identity is the `d->host` string only.
- **Rule 7: no mechanism was invented.** Go still has no `dns_cache`
  (`add_dns_host`, `src/comm.c:1511-1517`), no ident worker and no WHOD
  daemon. The resolver keeps its own `dnsLookupTimeout`; a timeout is a
  failure, exactly as a NULL `gethostbyaddr` is.
- **R1a, C's undefined behaviour.** `wildhost` and `double_wild`
  (`src/comm.c:1470-1471`) are uninitialised stack arrays filled **only** on
  the resolution-failure branch. When a name resolves, C runs `isbanned` on
  garbage. The port compares the host alone in that branch and records the
  omission here instead of reproducing the garbage.

**Deploy note.** A ban written in raw-IP form (`127.0.0.1`) no longer matches
anything, because C's strings are the padded quad and its two wildcards. The
repo ships no `data/badsites`, so no fixture changes; ops must read prod's ban
file and rewrite any raw-IP entries into the padded spelling before deploying.

## 2. Site contracts

| Site | Contract | Position |
|---|---|---|
| `comm.c:1552-1555` | `DNS lookup failed on %s.` at CMP / LVL_GOD / file TRUE, `%s` = the padded quad | Before the ban check, inside resolution; only when `nameserver_is_slow` is clear and the lookup failed. With the shipped default it cannot fire. |
| `comm.c:1571-1575` | `Sorry, your site is banned.\r\n` written to the descriptor, then close, then `Connection attempt denied from [%s]` at CMP / LVL_GOD / file TRUE | Both telnet BanAll arms. TLS keeps its bare close (a Go-only transport, dropped before the handshake) and still logs; WebSocket keeps its policy-violation close frame and logs. |
| `ban.c:205-209` | `%s has banned %s for %s players.` at NRM / MAX(LVL_GOD, invis) / file TRUE | After the ban is added, **before** the acknowledgement and before `write_ban_list()`. `%s` is the parsed site — the first non-fill-word token, already lowercased by `one_argument`; only the stored copy is truncated to 50 bytes. |
| `ban.c:237-244` | `%s removed the %s-player ban on %s.` at NRM / MAX(LVL_GOD, invis) / file TRUE | **After** the acknowledgement, before `write_ban_list()`. `%s` is `ban_node->site`, the stored spelling. |
| `comm.c:2136-2138` | `Losing player: %s.` at CMP / LVL_IMMORT / file TRUE, `%s` = `GET_NAME(d->character)` or `<null>` | The `else` of `if (d->connected == CON_PLAYING)`, for a descriptor that holds a character but never played. |
| `comm.c:2140-2143` | `Losing descriptor without char.` at CMP / LVL_IMMORT / file TRUE | The `else` of `if (d->character)`: no character at all. |

`GET_NAME` is NULL-safe only because the caller guards it
(`src/utils.h:275`), and C creates `d->character` on the **first input at the
name prompt**, before it checks for an empty line (`src/interpreter.c:1743-1752`);
a fresh `clear_char` zeroes `player.name` (`src/db.c:2976-2989`), so an empty
name line prints C's literal `<null>`. `session.descriptorBound` records that
input and is set by `terminalName` (every telnet name line, including the empty
one) and by `handleLogin` (the browser transport's name-prompt message).

## 3. The close_socket branch audit

| Go path that closes a non-playing descriptor | C branch |
|---|---|
| Telnet read fails before any name-prompt line (EOF, error, pre-auth idle) | `!d->character` → `Losing descriptor without char.` |
| Telnet read fails after the name (password, creation, menu) | `d->character && !CON_PLAYING` → `Losing player: <name>.` |
| Empty line at the name prompt (`interpreter.c:1751`) | → `Losing player: <null>.` |
| A login refusal closes the send channel (bad PW ×3, wizlock, site ban, `abortEntry`) | → `Losing player: <name>.` (the refusal's own producer still fires first) |
| `UnregisterSession` for any session that is not playing (WebSocket pre-auth drop or timeout, WebSocket refused login, guest teardown) | `d->character` state decides the arm |
| Menu option `0` after `quit` (`interpreter.c:2168-2170`) | → `Losing player: <name>.`; the arm clears `menuActive` before closing, so it calls the producer directly |
| Frozen self-delete (`interpreter.c:2322-2325`) | → `Losing player: <name>.` (its own `has self-deleted.` line is separate) |
| `perform_dupe_check` displacing a menu/creation descriptor, an unswitch, a usurp or an OLC editor (`interpreter.c:1551-1576`) | C NULLs `k->character` first, so all four take `Losing descriptor without char.` |
| A linkless RECON target | no descriptor in `descriptor_list`; C never reaches `close_socket`, so nothing is logged |

A **playing** descriptor still logs only `Closing link to: %s.` on its linkdead
transition (DP-1323). Two Go-only paths have no C state and are rendered as the
descriptor's own C state: the pre-auth rate limiter/lockout arms (the name was
never parsed, so `<null>`) and `abortEntry` (a store failure). The
input-overflow close is the same. The guest login is admitted as a playing
descriptor and closes with `Closing link to:`; only its *name* has no C analogue
(DP-1378/1379).

Census consequence: every syslog-complete immortal now sees these lines in any
scenario that disconnects a non-playing descriptor. Each moved row has to be a
new *match with C*, never a new diff.

## 4. Lock audit

| Producer | Locks held when `MudLog` is called |
|---|---|
| `DNS lookup failed on %s.` (`identifyConnection`) | None. The lookup runs in a per-connection goroutine or in the accept loop, and the accept-loop counters are released (`releaseConnectionSlot`) before any refusal or log. Never `connMu`. |
| `Connection attempt denied from [%s]` | None, by the same rule: `refuseBannedConnection` is called after the slot is released and before `handleConn`; the WebSocket arm runs before the session exists. |
| `has banned` / `removed the %-player ban` | None. `DoBan`/`DoUnban` take `BanManager.mu` only inside `AddBan`/`RemoveBan`/`WriteBanList`, each released before the producer; `cmdBan`/`cmdUnban` read the acting body under `manager.mu.RLock` and release it before calling in (`pkg/session/mudlog_actor.go`), which is required because delivery takes `manager.mu.RLock` per session. |
| `Losing player:` / `Losing descriptor without char.` | None. `LoseDescriptor` is called from the transport loop (no locks), from `UnregisterSession` under `playerLifecycleMu` only — delivery takes `manager.mu`, per-session `sendMu` and the world's read lock, never `playerLifecycleMu` — and from the menu and dupe-check paths, which hold no lock at the call. |

MudLog delivery itself takes `world.mu.RLock`; no producer here is called with
any world, player, manager or zone write lock held.

## 5. Classifications

| Site | Class | Evidence |
|---|---|---|
| `ident.c:235` | `blocked-reachability` | `ident_check` is only reached from ident.c's ident reply path, and Go has no ident worker (rule 7): the `ident` toggle is state only (`pkg/game/other_settings.go`). Go refuses a banned connection from comm.c's own `isbanned` path. |
| `objsave.c:489` | `blocked-representation` | No rent file: C's `fopen` fails, so it fires on **every new character's first entry**. Go stores the inventory in the SQLite row (`pkg/db`), so there is no rent file, no rent code and no selectable arm. |
| `objsave.c:504` | `blocked-representation` | `RENT_RENTED` or `RENT_TIMEDOUT` and the accrued cost exceeds gold plus bank — the rent-then-log-in-poor arm. |
| `objsave.c:519` | `blocked-representation` | `RENT_RENTED` — **every ordinary re-login after renting** in C. |
| `objsave.c:524` | `blocked-representation` | `RENT_CRASH`, written by C's `Crash_crashsave`. |
| `objsave.c:528` | `blocked-representation` | `RENT_CRYO`, which only C's receptionist sets. |
| `objsave.c:534` | `blocked-representation` | `RENT_FORCED` or `RENT_TIMEDOUT` — C's force-rent and timeout codes. |
| `objsave.c:539` | `blocked-representation` | The default arm, for a rent code this codebase never wrote. |
| `objsave.c:1186` | `blocked-representation` | The receptionist's accepted-rent and cryo arms. Go has no receptionist or cryogenicist command path; `quit`'s `RentOut` is the quit path only and is not evidence of reachability. |

Go stores one inventory blob per player row and no `RENT_*` code, so a player in
ordinary play sees none of the eight rent lines. Zach decides the representation
in DP-1404 (a save-format question, not a producer).

## 6. Notes, including what this PR did not touch

1. **Stored data.** `db.c:2377` copies `d->host` into the player file. Go's
   player record has **no host column at all** (`pkg/db`), so there is nothing to
   convert and nothing was changed; the wizard `last` view renders the live
   session's `RemoteIP()` through the shared padding rule.
2. **Not in scope, for the record:** the connection-limit refusal
   `Sorry, Dark Pawns is full right now...` (`src/comm.c:1508`) is not a mudlog
   site and the port writes nothing there; and a *kicked* playing descriptor
   (C's `do_dc` → `close_socket` → `Closing link to:`) still closes without its
   linkdead line because `SendClosed` short-circuits the linkdead transition.
   Both need their own ruling.
3. **Parsing.** Both ban commands parse through `oneArgument`
   (`src/interpreter.c:1267-1284`): the first non-fill-word token, lowercased.
   `DoBan` logs that parsed site (untruncated) and stores at most 50 bytes of it,
   as C does, so a >50-character site re-banned verbatim is accepted in both —
   C's duplicate check compares the truncated stored site. `DoUnban` was
   trimming and lowercasing the whole argument, so `unban foo bar` looked for
   "foo bar" where C looks for `foo`; fixed in review, with tests for a
   trailing word, a leading filler word and a filler-only argument.
4. **One read of `nameserver_is_slow` per connection.** The accept loop used to
   read the flag twice — once for the precheck and once as the goroutine's
   branch selector — so a `slowns` toggle between the two could hand the session
   an empty identity and skip its ban check. `identifyConnection` now takes the
   flag as an argument.
5. **Pre-existing flake, not from this change:** `pkg/session`
   `TestEntryWebSocketSavedIdentityAndMenuResume` asserts `!IsLinkless()` after
   its journey's deferred `conn.Close()`, so the assertion races the transport's
   linkdead transition. It failed once in a full-package run here and then passed
   5/5 on this tree and 4/4 on `origin/main`.
6. **Oracle vehicles.** Ban refusal: the harness cannot open a connection it
   expects to be refused, so the three-transport refusal test is unit-green only.
   The `Losing player:` and `Losing descriptor without char.` arms are driven
   through real transports and real command dispatch in the unit tests; the
   oracle corpus gained no new vehicle in this PR, so the parity evidence is the
   combined census's moved rows.

## 7. Reproduce

```sh
go test ./pkg/telnet -run 'TestIdentifyConnection|TestBannedConnection|TestTLSBanned|TestDropBeforeName|TestEmptyNameLine|TestDropDuringCreation' -count=1
go test ./pkg/session -run 'TestBanMudlog|TestMudlogActor|TestLose|TestMudHost|TestCConnectionHost|TestAcceptBanLevel|TestWildBanHosts|TestWebSocketBannedRefusalLogs|TestEntryBanLevelUsesMudHost' -count=1
python3 docs/fidelity/depth/handoff/2026-10-09-dp-1371-mudlog-admission-controls.py --output /absolute/evidence/path
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites-check.py
```

The control script prints `1 -> 0 -> 1` for all nine cases (`dns-failure-log`,
`dns-slow-gate`, `refusal-bytes`, `refusal-log`, `ban-producer`, `unban-order`,
`losing-descriptor`, `losing-player`, `host-padding`), asserting the named
failure line in each revert and never a build failure. `ban-producer` removes the
call and leaves `_ = rawSite`; `losing-player` leaves `_ = fmt.Sprintf(...)`,
because both files would otherwise lose an import and fail to build instead of
failing on the assertion.

