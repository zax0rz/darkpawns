# Converting a PostgreSQL Dark Pawns database to SQLite

SQLite is Dark Pawns' sole long-term database: a fresh installation needs no
external service. This document is the one-time conversion procedure for an
existing PostgreSQL installation that already holds live characters.

The conversion is performed by `dp-db-migrate`, an operator tool in this
repository. It reads PostgreSQL, writes a SQLite file, verifies the copy
independently, and installs the result atomically. It never writes to the
PostgreSQL database, and it never connects anywhere you did not name.

```bash
go build -o dp-db-migrate ./cmd/dp-db-migrate
./dp-db-migrate --help
```

**Do not run the tool on a live instance without reading the preflight section.**
The tool is safe to run repeatedly, but the cutover itself is a service change.

## What is converted

Exactly five tables, because they are the ones the game and its moderation
surface read and write:

| Table | Rows | Identity |
|---|---|---|
| `players` | characters: stats, conditions, resources, inventory, equipment, character data, lockout state | `id` (autoincrement) |
| `abuse_reports` | player reports, review state, resolution | `id` (autoincrement) |
| `admin_log` | administrator action trail | `id` (autoincrement) |
| `player_penalties` | mutes and bans, issue/expiry/expired state | `(player_name, penalty_type, issued_at)` |
| `word_filters` | moderation word and regex filters | `id` (autoincrement) |

Everything else the instance owns lives outside this database and is **not**
touched by the conversion:

- mail, boards, houses, clans, aliases, shops and the admin store live under the
  instance's `data/` directory;
- the world tree (`lib/world`) and Lua scripts (`-scripts`) are files;
- the research sidecar (`decision_log`, `combat_log`) is a separate, separately
  enabled database with its own lifecycle.

Back those up as files if you back them up at all. A database migration is not a
backup strategy.

If the PostgreSQL schema holds a table outside the five above, the tool **stops**
and names it. That table may be production data with no SQLite home yet, so the
decision to leave it behind is yours, not the tool's: pass
`--drop-extra-tables` only after you have decided those tables are obsolete. The
receipt always records what was left behind.

## Schema differences and canonicalization

The destination schema is not written by hand. It is created by the same two
constructors the server runs at boot — `pkg/db.New` for the game store and
`moderation.NewManager` for the moderation tables — so a converted database and
a freshly created one cannot drift apart. Where the dialects disagree, the tool
records the difference instead of hiding it.

| Concern | PostgreSQL | SQLite | How it is compared |
|---|---|---|---|
| `id` columns | `SERIAL` | `INTEGER PRIMARY KEY AUTOINCREMENT` | integers, exact |
| Booleans (`is_admin`, `is_regex`) | `boolean` | `BOOLEAN` (stored 0/1) | both normalized to `0`/`1` |
| Game-store timestamps | `timestamptz` (an instant) | `TIMESTAMP` (an instant, offset preserved) | UTC instants, nanosecond precision |
| `admin_log.duration` | `interval` (rendered `01:30:00`) | `INTERVAL` (untyped storage) | text, exactly as stored |
| JSON columns (`inventory`, `equipment`, `character_data`) | `json` (exact text) | `JSON` (text) | semantic JSON; raw-byte differences reported separately |
| `room_vnum`, `hometown`, `olc_zone` | signed integer | integer | exact, including the `-1` load-room sentinel |

Two deliberate canonicalizations, both reported per column when they fire:

1. **Timestamps are compared as instants, not wall clocks.** A lockout written
   as `20:05-04` and the same instant written as `00:05Z` are equal — they are
   the same moment — and the comparison says so. A literal with no zone at all
   is read as UTC, matching SQLite's `CURRENT_TIMESTAMP`, and that reading is
   only ever used for comparison.
2. **JSON is compared semantically, byte differences still reported.** PostgreSQL
   `jsonb` (an install that predates the game store's `json` columns) re-spaces
   and re-orders its input, so a byte comparison would flag a difference where
   the value is identical. When the value matches but the bytes moved, the
   receipt lists the column under `json_reformatted_columns`. Numeric literals
   are *not* normalized: `1` and `1.0` are different content, not formatting.

Nothing else is canonicalized. Text, password hashes, names, room numbers,
levels, experience, statistics, conditions and counters are compared exactly.

## Schema differences you may find in an old installation

`pkg/db` and `pkg/moderation` create their schema with `CREATE TABLE IF NOT
EXISTS`, so a table created by an older build keeps its older shape. Two of
those shapes are known:

- `scripts/init-moderation-db.sql` is a legacy script (no longer referenced by
  anything) that defines extra columns the runtime never reads —
  `abuse_reports.severity`, `abuse_reports.evidence`, `admin_log.details`,
  `word_filters.is_active` — plus three tables the runtime does not own
  (`admin_users`, `chat_logs`, `player_notes`). An installation that loaded it
  will hit both stop conditions above.
## How the tool works

Understanding four behaviours makes the procedure below readable.

**Atomicity.** The database is built at a temporary sibling path
(`.<name>.migrating-<pid>` next to the destination), verified there, and only
then renamed over the destination. The destination path is never partially
written: a run that dies in the copy, the integrity check, the verification or
the finalization leaves no file where a result should be, and the temporary file
is removed. The file is `0600` and the directory holding it should be
operator-only.

**The rename is the point of no return, and the receipt says which side of it a
run ended on.** The file is fsynced before the rename and the destination
directory is fsynced after it, because a rename is only durable once its
directory entry is. If that last sync fails, the destination **already holds the
verified database**; that is a different fact from an install that never landed,
and it is reported as one:

```
"destination": { "installed": true, "durability_uncertain": true }
"failure":     { "phase": "install-durability", ... }
```

and the summary prints it on its own line:

```
  destination: /var/lib/darkpawns/darkpawns.db
    state:     installed, directory entry NOT synced (a crash could lose the rename)
```

Read that state rather than the exit status alone. The file in place is the
verified database — check it, then re-run with `--replace` if you want a fresh
install — but a crash before the next successful sync could still lose the
rename. Every failure **before** the rename genuinely leaves the destination
untouched, and the receipt records `"installed": false` with the summary line
`state: unchanged (nothing was renamed into place)`.

**Transaction.** All five tables are copied inside one SQLite transaction, and
the destination is opened through the production schema initialisation first.
Because WAL would leave the newest commits in a `-wal` sidecar that a rename does
not carry, the tool checkpoints and switches the journal mode to `DELETE` before
the rename: the artifact is one self-contained file. The server turns WAL back on
when it opens the file, so runtime behaviour is unchanged.

**The source is read-only, once.** Every source read — the catalog, the copy and
the verification — happens inside a single read-only, repeatable-read
transaction. That is what makes "the source is not modified" a property of the
connection rather than of the tool's discipline, and it means the verification
compares the copy against the exact snapshot it was copied from. No `pg_dump` and
no table locks are involved; PostgreSQL only has to keep the snapshot alive for
the duration.

**A refusal is a result.** A non-empty destination, an unknown table, an
unmappable column, a reversed source/destination, an in-memory destination and a
missing source table all stop the run with a message naming the problem, exit
non-zero, and leave the destination untouched.

### Verification

Verification never trusts the copy. Both sides are re-read through their own
connection and digested separately, and a table passes only when all of the
following agree:

- row count;
- the primary-key set (order-independent digest of every key);
- the whole-table content digest (order-independent over per-row digests, so a
  change in physical row order is not a change);
- the null shape (null count per column, plus the nulls' position in each row);
- per-column content digests, which is what names the column when something
  differs;
- the highest generated id;
- `PRAGMA integrity_check` (and `PRAGMA foreign_key_check`) on the destination;
- the destination's unique and named indexes, and that no two player names
  collide when folded to lower case.

`--verify-only` runs exactly this comparison against two existing databases and
writes nothing, which is how you re-check a conversion later. It performs the
same source-schema inventory a conversion performs, because a comparison that
quietly narrowed its own scope would report a partial result as OK:

- a source table outside the migrated five stops it, unless the receipt of the
  conversion being verified proves that conversion was explicitly told to leave
  that exact table behind;
- a source column with no destination column stops it, on the same terms — a
  table is not "verified" while part of its data has nowhere to go;
- `--drop-extra-tables` and `--drop-extra-columns` are **refused** in this mode.
  A verification and a conversion are different commands with different
  authority, so a verification must not be able to widen what it ignores with an
  argument on its own command line. The only proof it accepts is
  `--conversion-receipt <path>`;
- a receipt proves something only when it is the receipt of a **successful
  conversion** that records the flag granting the allowance. A verification
  receipt, a failed conversion, a conversion that left nothing behind and a file
  that is not a receipt at all each prove nothing, and each is refused.

```bash
./dp-db-migrate --from "$DP_MIGRATE_FROM" --to /var/lib/darkpawns/darkpawns.db   --verify-only --conversion-receipt /var/lib/darkpawns/migration-receipt.json
```

Both modes emit a JSON receipt (`--report path.json`) and a terminal summary. The
receipt carries counts, column names, digests, install state and timings — never
a password hash, a character's payload or a DSN credential. The receipt also
records which allowances the run used and what proved them, so a later reader can
follow the chain from an explicit decision to a passing verification.

### Generated ids

SQLite's `AUTOINCREMENT` keeps a sequence above the highest id ever inserted, and
the copy inserts explicit ids, so the sequence lands on the migrated maximum. The
tool does not rely on that inference: it sets the sequence explicitly, reads it
back, and **fails the run** if any table's sequence is not at or above its
highest migrated id. The receipt prints `max`, `sequence` and
`sequence_above_max` per table. An id is therefore never reused, and a character
created after the cutover sorts after every migrated character.

## Downtime

Measured on a developer workstation against a disposable PostgreSQL, converting
the full five-table set with `--verify` (the number that matters is the sum of
copy, integrity check and verification):

| Dataset | Rows | SQLite file | Wall clock |
|---|---|---|---|
| Typical | ~7,500 | 2.9 MB | ~1.3 s |
| Large | ~52,500 | 24 MB | ~12 s |

Scale your own with a restore of a recent dump. The conversion is not the
downtime: stopping the service, taking the filesystem backup and changing the
service configuration dominate, and the tool should be run **while the service is
stopped** so that no write lands between the snapshot and the cutover.

- An installation that predates the folded-name index
  (`players_name_folded_key`, unique on `lower(name)`) can hold two characters
  whose names differ only in case. The real schema refuses that, the runtime
  refuses to arbitrate it (`ErrAmbiguousPlayerName`), and so does the
  conversion: the copy fails rather than pick an account. Resolve the collision
  in PostgreSQL first — that is a player-visible decision.

If the source holds a **naive** timestamp column (`timestamp without time zone`)
rather than `timestamptz`, boot the current binary once against PostgreSQL before
converting. The server's own startup migration converts those columns in place,
and it knows the database's timezone while the converter can only see wall-clock
text.

## Preflight

Do all of this before stopping anything.

1. **Announce and empty the instance.** Zero connected players: a character saved
   while you convert is a character lost. Use the shutdown path you normally use
   and confirm nobody is logged in.
2. **Record the source row counts** so you have an independent expectation:

   ```sql
   SELECT (SELECT count(*) FROM players)          AS players,
          (SELECT count(*) FROM abuse_reports)   AS abuse_reports,
          (SELECT count(*) FROM admin_log)       AS admin_log,
          (SELECT count(*) FROM player_penalties) AS player_penalties,
          (SELECT count(*) FROM word_filters)    AS word_filters;
   ```

   Inventory every table too (`\dt`). Anything outside the five stops the tool,
   and you want to know that now rather than at cutover.
3. **Take a `pg_dump -Fc`** of the database, and verify it restores into a
   scratch database. A dump you have not restored is a hope.
4. **Take a filesystem backup** of the instance directory (world tree, `data/`,
   `lib/logs/`, the current SQLite file if one exists) and of the current server
   binary and service unit. The conversion does not need any of it, but the
   rollback does.
5. **Check free disk space.** Budget three times the source size on the
   destination filesystem: the new SQLite file, its temporary copy, and a spare.
6. **Confirm the source schema is current.** Boot the *current* binary once
   against PostgreSQL (that converts older timestamp and JSON columns in place),
   then stop it. Pass `-telnet-port 0` if you want to be certain nobody logs in.
7. **Choose the destination path and ownership.** The file must live on local,
   durable storage — not on NFS, not in a container's ephemeral layer, not in a
   directory that a deploy step prunes. Decide its mode (`0600`, owner = the
   service account) and its directory (`0700`).
8. **Build the tool from the revision you are deploying** and record its
   revision: `go build -o dp-db-migrate ./cmd/dp-db-migrate`.

## Conversion

Run this with the service **stopped**. Substitute your own paths. Passing the
source DSN in `argv` exposes it to anything that can read the process list, so
prefer the environment:

```bash
# export DP_MIGRATE_FROM='postgres://...'   # or DATABASE_URL
# export DP_MIGRATE_TO='/var/lib/darkpawns/darkpawns.db'

./dp-db-migrate \
  --from "$DP_MIGRATE_FROM" \
  --to   "$DP_MIGRATE_TO" \
  --verify \
  --report /var/lib/darkpawns/migration-receipt.json
```

**Keep the receipt.** It is not a log: it is the artifact that proves what this
conversion was explicitly allowed to leave behind, and a later `--verify-only`
needs it if the source holds a table or column the destination cannot hold. Store
it beside the database and keep it with the backups.

`--verify` is on by default; it is written out here so the command reads as what
it does. Read the summary before doing anything else. You want `result: OK`, the
five `copied` rows with the counts from preflight, `sqlite integrity_check: ok`,
an `id watermarks` block with `above_max=true` for `players`, `abuse_reports`,
`admin_log` and `word_filters`, and five `ok` lines under `verification`.

The tool writes only `$DP_MIGRATE_TO` and its temporary sibling. If a run is
interrupted, re-run it: it is idempotent against an absent or empty destination,
and it cleans up its own temporary file.

If the destination already exists and holds data (for example a previous
attempt, or the instance's current empty SQLite file), the tool refuses. Add
`--replace` to build a verified replacement through the temporary file and the
atomic rename; the old file is replaced, not merged.

Under `systemd`, run the tool as the service account while the unit is stopped:

```bash
systemctl stop darkpawns
sudo -u darkpawns   ./dp-db-migrate --from "$DP_MIGRATE_FROM" --to /var/lib/darkpawns/darkpawns.db   --report /var/lib/darkpawns/migration-receipt.json
chown darkpawns:darkpawns /var/lib/darkpawns/darkpawns.db
chmod 600 /var/lib/darkpawns/darkpawns.db
```

### Point the service at SQLite

Point the service at the file you just verified, and at nothing else. The
supported spellings are the `-db` flag and `DATABASE_URL`; the value may be a
bare path or a `sqlite://` URL:

```
ExecStart=/opt/darkpawns/darkpawns-server -world /opt/darkpawns/lib/world \
  -telnet-port 7777 -db /var/lib/darkpawns/darkpawns.db
```

```
Environment=DATABASE_URL=sqlite:///var/lib/darkpawns/darkpawns.db
```

Do not rely on the default path (an embedded SQLite file created beside the
world data): it makes the converted file a second database nobody looks at. And
remove the `postgres://` URL — while a `postgres://` value is present, the
service uses PostgreSQL and your conversion is simply not in use. If the unit is
templated or generated, change the generator and record the change; a hand-edit
that the next deploy reverts is how an instance silently returns to PostgreSQL
(with the characters created in the meantime missing).

## Post-cutover verification

1. **Service state.** `systemctl is-active` is `active`, and the restart counter
   is stable — a crash-looping service with a valid database is a different bug,
   but you must see it as one.
2. **The right file is open.** Confirm the running process holds the verified
   file (`ls -l /proc/<pid>/fd | grep darkpawns.db`). If the mtime changed
   immediately after start, or the boot log says it created a database, the
   service is on a fresh empty file — stop and fix the configuration. A migrated
   database is distinguishable from a fresh install by its rows:

   ```bash
   sqlite3 /var/lib/darkpawns/darkpawns.db \
     'SELECT (SELECT count(*) FROM players) AS players,
             (SELECT count(*) FROM abuse_reports) AS reports,
             (SELECT count(*) FROM word_filters) AS filters;'
   sqlite3 /var/lib/darkpawns/darkpawns.db 'PRAGMA integrity_check;'
   ```

   The counts must match preflight. `integrity_check` must print `ok`.
3. **Boot log.** No PostgreSQL connection attempt, no fallback message, no
   "created" or "initialised database" line for the destination.
4. **A known character logs in.** Use an account you can log into as yourself:
   the password verifies, the character enters the room the character record
   says, and the inventory and equipment are the ones it had. Check a character
   with a lockout if you can, and let it expire on schedule.
5. **Save and reload.** Change something trivial (the title), save, log out, log
   back in, confirm it is still there. Then confirm the file's `-wal` sidecar is
   being written, which is the runtime's WAL behaviour rather than the artifact's.
6. **Moderation.** An abuse report written now appears in the admin view; a word
   filter from before the cutover still matches; an active mute is still in
   force. A penalty with an expiry set before the cutover should still expire.
7. **A new character.** Create one and confirm its `id` is above every migrated
   id. This is the single check that catches a broken id sequence.
8. **Restart and log in again.** Stop and start the service, then log in once
   more. The second boot is the one that proves the file is durable and the
   configuration is not a one-shot.
9. **Re-verify, read-only, after the fact.**

   ```bash
   ./dp-db-migrate --from "$DP_MIGRATE_FROM" --to /var/lib/darkpawns/darkpawns.db \
     --verify-only --report /tmp/verify-receipt.json \
     --conversion-receipt /var/lib/darkpawns/migration-receipt.json   # only if that conversion dropped something
   ```

   Run this **before** the first post-cutover save if you want a strict result:
   once the instance has written new rows, `--verify-only` will (correctly)
   report them as differences. A late run is still useful for the row counts and
   the integrity check, not for the digests.

## Rollback

The conversion does not modify PostgreSQL, so rollback is a configuration
change, not a data restore.

1. Stop the service.
2. Restore the previous binary and the previous service unit from the preflight
   backup, including the PostgreSQL DSN.
3. **Do not restore the PostgreSQL dump.** The source was never written: it holds
   exactly the characters it held before the cutover. Restoring the dump would
   roll back anything that happened between the dump and the cutover for no
   reason. Restore it only if you independently changed PostgreSQL, and then
   accept that the SQLite side is now ahead.
4. **Preserve the SQLite file.** Move it aside rather than deleting it — it is
   the evidence for whatever went wrong, and the receipt names the digests it
   was verified against. Do not delete the receipt either.
5. Start the service and repeat the post-cutover checks that apply to PostgreSQL:
   active, connected to the right database, a known character logs in, a new
   character appears, and the counts match preflight.
6. Anything players did between the cutover and the rollback exists **only** in
   the SQLite file. Decide explicitly whether to replay it by hand before you
   discard the file; a rollback after live play is not a lossless operation.

## Stop and escalate instead of pushing through

The tool refuses, and you should too, when:

- the source has a table outside the five;
- a source column has no destination column (the tool names it; extending the
  runtime schema is a code change, not an operator decision);
- a source table is missing entirely, which means PostgreSQL never had it
  created — boot the current binary once against PostgreSQL and retry;
- two characters' names collide when folded to lower case;
- a verification refuses because the source has grown a table or column with no
  destination column, and no conversion receipt proves the decision — that is a
  question for you, not a flag to add;
- timestamp or JSON data cannot be compared without transformation the receipt
  does not describe;
- the migration would need to run while the instance is live;
- `origin/main` has moved and the build you tested is no longer what you are
  deploying.
