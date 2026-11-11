# Entry browser routing and identity audit — 2026-11-11

## Browser name routing

`entry-name-gates` drives initial invalid names, the `Aiko` confirmation
decline, and the accepted `Biko` creation through C's `CON_GET_NAME` and
`CON_NAME_CNFRM` path (`src/interpreter.c:1743-1818,1819-1858`). The scenario
uses only the `quiet-mobs` fixture and typed creation input; it has no setup
commands that either engine could refuse. Its retained C block reaches the
name prompt, repeated invalid-name prompts, confirmation retry, new-character
password flow, and first world entry.

The target was run with `DP_ORACLE_GO_TRANSPORT=ws`. The Go side used the real
`web/public/mud-client.js` loaded by `internal/oraclediff/wsdriver/driver.mjs`,
the same module imported by `website-astro/src/pages/play.astro`. The census
was CLEAN_AFTER_RECHECK at
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-entry-browser-routing`.
`entry-name-gates` passed on its first attempt. The name-retry scenario's
initial content fingerprints were unstable, so the wrapper classified it as
INFRA and rechecked it; the retained recheck passed. The attempt logs record
the actual `ws://` browser-client transport. The mode remains a probe outside
the standard telnet expected-divergence pins.

The row remains blocked. The approved proof spelling was
`entry-name-gates@1[ws]`, but `make fidelity-depth` rejects it as invalid seed
syntax (`1[ws]`). The claims enumerator records scenario and seed, but not
transport. Per Zach's instruction, the checker was not changed; this matching
browser transcript remains an advisory probe and the row stays blocked.

The family revert proof is
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-entry-browser-revert-proof`.
For that run only, `writeOutput` changed the initial prompt's final `?` to `!`
in `web/public/mud-client.js`; `entry-name-gates` failed against C on two
content attempts. The browser module was restored immediately after the census
snapshot. No production client change is included in this PR.

## Identity consumers

The source uses several distinct identifiers:

- `player_table.name` is lowercased when loaded and created, and `find_name`
  uses case-insensitive comparison to locate the player-file record
  (`src/db.c:519-523,2651-2678`; `src/interpreter.c:1492-1498`). That name
  indexes `load_char` and crash-file cleanup (`src/db.c:2340-2353`;
  `src/objsave.c:263-269`).
- The index maps names to persistent numeric IDs, while duplicate-session
  teardown, mail, and house access compare those IDs
  (`src/db.c:2313-2336`; `src/interpreter.c:1528-1563`;
  `src/mail.c:553-585`; `src/house.c:620,734-738`).
- C persists `GET_BAD_PWS` on the character and separately applies its
  three-failure disconnect threshold to `descriptor_data.bad_pws`
  (`src/interpreter.c:1880-1894`).
- Go's world registry is name-keyed (`pkg/game/world.go:120,424-435,578`) and
  login adopts the stored record spelling (`pkg/session/session_login.go:186`).
  Go also has per-IP login admission/rate controls; its per-account budget is
  opt-in and disabled by default (`pkg/session/manager.go:346-354`).
- The current Go tests state that the former `X-Agent-Key` admission path is
  gone (`pkg/session/session_test.go:511-527`); no active agent-key consumer
  was found in the current Go source search.

These findings do not establish one identity key that can be applied across
name lookup, durable player IDs, and connection-level failure accounting.
Zach approved splitting the umbrella on 2026-11-11. The depth manifest now has
five separately blocked consumers: C name/index lookup, persistent ID for
duplicate sessions, persistent ID for mail, persistent ID for house access,
and the password-retry counter. Each remains blocked pending its own proof.
Go-only per-IP admission/rate controls and the retired agent-key path are
recorded in this audit but are not added as C identity consumers. No lockout or
identity behavior changed.
