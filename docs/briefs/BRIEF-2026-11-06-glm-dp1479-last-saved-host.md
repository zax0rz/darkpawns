# Brief for GLM-5.3: DP-1479, `last` reads the saved host, as C does

Linear: DP-1479. Rulebook: R1 (bytes), R4 (no invention), R5b and R5c (find the class), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail), R5i (a fixture that silences a surface).

Work in a worktree: `git worktree add ~/wt/glm-dp-1479 -b glm/dp-1479-last-saved-host origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

**Ordering:** MiMo's PR #2311 marks `last-depth` as an expected divergence citing DP-1479. If #2311 hasn't merged when you're ready to open the PR, stop and report rather than building on its branch.

## Zach's decision (made 2026-11-06)

Save the host in the player store, as C saves it in the player file. Zach approved this persistence change, an additive column, for this issue only. Don't make any other schema change.

## Ground truth (verified by Claude at `origin/main` 5d472424c)

**C**
- `do_last` (`src/act.wizard.c:1834-1854`) loads the **saved** record and prints `"[%5ld] [%2d %s] %-12s : %-18s : %-20s\r\n"`, filled with idnum, level, `class_abbrevs[class]`, name, `chdata.host`, and `ctime(&chdata.last_logon)`.
- `save_char` (`src/db.c:2366-2381`) copies `ch->desc->host` into the record **only when the character has a descriptor**. A save without one, such as a linkdead player, writes an **empty** host. Keep that quirk.
- The host's text comes from `src/comm.c:1536`: `"%03u.%03u.%03u.%03u"` when there's no DNS name. The port already has this as `CConnectionHost` (`pkg/session/mud_host.go`). Also read `src/comm.c:1555-1563` to see when C stores a resolved name instead, and say what the port does there.
- `last_logon` is set in `char_to_store` (`src/db.c:2592`); Go already saves it as `LastLogon`.

**Go**
- `cmdLast` (`pkg/session/wiz_system.go:336`) has two branches:
  - **The store branch**, the one production runs, prints the invented `"[%d] [%2d] %-12s : Level %d\r\n"`.
  - **The no-store branch** (`onlinePlayerForLast`, `formatOnlineLast`) rebuilds C's line from the *live session*. It existed only because the census used to run without a store.
- `db.PlayerRecord` (`pkg/db/player.go:80`) has `ID`, `Class` and `LastLogon`, but no host. Additive columns go through the list near `pkg/db/player.go:212` (SQLite has no `ADD COLUMN IF NOT EXISTS`). Follow how `last_logon` was added.

## Do

1. **Add the column.** A `host TEXT` column on `players`, added through the existing migration list. Existing rows read as empty: `COALESCE(host, '')`.
2. **Save it.** On every save, write the host from the session's connection, formatted like `CConnectionHost`, or empty when the player has no live descriptor. Find every save path (normal save, quit, rent, autosave, linkdead, the admin editor) and list them in the PR, each with the host it writes.
3. **Print it.** The store branch of `cmdLast` prints C's line from the saved record: `%5d` idnum, `%2d` level, the class abbreviation, `%-12s` name, `%-18s` host, and the `ctime` of `LastLogon` (C's `ctime` ends in `\n`; check what `%-20s` does with it and keep it).
4. **Delete the no-store branch** (`onlinePlayerForLast`, `formatOnlineLast`) if nothing else calls them. If a no-database production mode still exists and reaches `last`, report it rather than deciding.
5. **The `players` sibling (R5b).** MiMo reported that `cmdPlayers` lists names `ORDER BY name` while C walks `player_table` order. But Claude's grep finds no `ACMD(do_players)` in `src/`. Find what C command, if any, `cmdPlayers` ports. If it's a Go-only command (check the approved Go-only list from PR #1646), leave its order alone and say so. Then list every other command whose output reads saved players (finger, whois, `set file`, clan listings) in `docs/fidelity/evidence/dp-1479/store-readers.tsv` with columns `command`, `Go function`, `C function`, `matches C? (yes/no/unchecked)`. Report mismatches; fix only `last` (and `players`, if it turns out to port a C command).

## Proof

1. **Unit tests** in `pkg/db`, `pkg/session` or both:
   - the host round-trips through save and load;
   - a save without a descriptor stores empty;
   - an old row without the column reads as empty;
   - `last` prints C's exact bytes from a saved record.

   Revert proof: with the old format string restored, the format test fails. Paste the failure.
2. **`last-depth` turns green.** Its expected-divergence pin goes STALE. Remove the pin, set the manifest row back to `oracle-green-multiseed`, and show the scenario PASS at seeds 1, 2, 3, 5 and 8. Revert proof: the old format string turns it FAIL.
3. **A targeted census:** `last-depth`, every scenario that sends `last` or `players`, and every scenario whose fixture saves and reloads a character (grep for relogin and quit-and-return fixtures). Name the set and why. `scripts/census.sh start --name glm-dp1479-last-host --scenarios …`, then `wait --run <dir>`; don't poll.
4. **The migration converter:** `make db-migrate-test`, because `tools/db-migrate` imports `pkg/db`. Say whether `DP_MIGRATE_TEST_DB_URL` was set.

## Rules

- Cite C only from `src/`, grepped in this session.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`. Propose any rule text in the PR description.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt`, `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`, `make db-migrate-test`.

## Done means

A PR against `main` containing:
- the column, the save paths and the C-format `last`;
- the TSV;
- the tests and revert proofs;
- the census directory;
- a **Decisions for Zach** section, including how the host column appears in the admin user editor, if it does, and anything about retaining it.

Then stop and report. Claude reviews in Zach's session, and Zach merges. This PR is stop tier (persistence).
