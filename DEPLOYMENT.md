# Running Dark Pawns

This guide covers running your own Dark Pawns instance — building the server, wiring
up persistence, and putting it behind a reverse proxy. It is deliberately
infrastructure-agnostic: it describes the software, not any particular deployment.

> **Operating the official `darkpawns.org` instance** (host access, deploy/rollback
> mechanics, backup targets) is documented separately in the private ops repo, not here.

## What you're running

A single Go binary (`cmd/server`) that serves three surfaces:

| Surface | Default | Notes |
|---|---|---|
| HTTP + WebSocket | `:4350` (`-port`) | Web client, `/ws`, `/api/*`, `/openapi.json`, `/admin/*`, `/health`, `/metrics` |
| Telnet | `:7777` (`-telnet-port`, `0` disables) | Classic MUD access |
| SQLite (embedded) | `lib/data/darkpawns.db` | The game database — one file, no setup, no external services |

It also reads/writes on-disk state: the world/scripts tree (`-world`, `-scripts`) and a
CWD-relative `data/` directory (shops, aliases, mail, admin store). With the checkout
layout below, the server changes its working directory to `lib/`, so that state
is in **`lib/data/`**, not the repository-root `data/`. The admin audit trail follows
the same rule: the log is **`lib/logs/audit.log`** (mode `600`, in a `750` directory it
creates on first boot). Back up the SQLite database and the instance's `lib/` tree, including any
edited world files. There is no database server to back up: the game state is that
one SQLite file.

## Quickstart

A fresh clone to a running server. No external services are required: the
server boots against an embedded SQLite database by default. Build and run:

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns

export JWT_SECRET="$(openssl rand -hex 32)"

go build -o server ./cmd/server
./server
```

`./server` with no flags works from the repository root: it loads the world from
`lib/world` (the directory holding `wld/`, `mob/`, `obj/`, `zon/` and `shp/`),
creates the SQLite database at `lib/data/darkpawns.db` on first boot, serves the
browser client from `web/public` on `:4350`, and accepts telnet on `:7777`
(`-telnet-port 0` disables it). Every flag is listed in `./server -h`; none of
the defaults need to be passed.

A stable `JWT_SECRET` is required outside development; a missing or short one
is rejected at boot with the command to fix it. Connect with
`telnet localhost 7777`, or open <http://localhost:4350>.

## Prerequisites

- Go (see `go.mod` for the version); a C toolchain is not required (`CGO_ENABLED=0`).
- No database server and no cache: with no `-db` flag and no `DP_SQLITE_PATH`,
  the server creates an embedded SQLite database at
  `<world>/../data/darkpawns.db` and boots against it. PostgreSQL is not a
  runtime backend of this build: a `postgres://` value stops the boot with a
  message naming the conversion tool, rather than being ignored.
- Optionally a reverse proxy (Caddy, nginx, …) terminating TLS in front of `:4350`.

## Build

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server
file darkpawns-server        # ELF 64-bit LSB … x86-64
```

Drop `GOOS`/`GOARCH` to build natively for your own platform.

## Configure

There is nothing to provision. The database is a file the server creates on
first boot at `lib/data/darkpawns.db` (beside the world data, from the `-world`
directory's parent), and the schema is created with it. Export a signing secret,
and name a different database path only if you want one:

```bash
export JWT_SECRET="$(openssl rand -hex 32)"
# Optional: put the database somewhere owned by the service account.
export DP_SQLITE_PATH=/var/lib/darkpawns/darkpawns.db
```

The file is the whole state of the game. Put it on local durable storage — not
NFS, not a container layer, not a directory a deploy step prunes — with the
directory at mode `700` and the file at `600`, owned by the service account.
Back it up by copying the file while the server is stopped, or with SQLite's
own online backup (`VACUUM INTO`).

Keep the signing secret private and stable across restarts. Startup rejects a
missing or short `JWT_SECRET` unless `ENVIRONMENT=development`, which uses an
ephemeral secret.

Moving an existing PostgreSQL installation to SQLite is a one-time operator
procedure with its own preflight, verification and rollback steps:
[docs/operational/SQLITE-CUTOVER.md](docs/operational/SQLITE-CUTOVER.md).

Alternatively, copy [`.env.example`](.env.example), edit it, and export it from
your shell. The native binary **does not load `.env` automatically**:

```bash
set -a
. ./.env
set +a
```

### TLS telnet (optional)

Plain telnet sends passwords in the clear. To also serve telnet over TLS,
point the server at a certificate and pick a port:

```bash
export TELNET_TLS_CERT_FILE=/path/to/fullchain.pem
export TELNET_TLS_KEY_FILE=/path/to/privkey.pem
./server -telnet-tls-port 7778
```

These are separate from `TLS_CERT_FILE`/`TLS_KEY_FILE`, which switch the HTTP
server to HTTPS; behind a TLS-terminating proxy, set only the telnet pair. The
server exits if the port is requested without a readable certificate. It
re-reads the files when they change, so a renewal (for example a Let's Encrypt
certificate the reverse proxy already manages) needs no restart; the server
user must be able to read them. Only TLS 1.2 and newer are accepted.

While the TLS port runs, MSSP advertises it (`TLS`, plus `HOSTNAME` from the
certificate), and Mudlet offers the encrypted port to players who connect in
plaintext. The login banner is unchanged.

### The Mudlet package (optional)

The telnet listener speaks GMCP (see [`docs/gmcp.md`](docs/gmcp.md)), so
Mudlet players get gauges, a map, and a chat window from the package in
[`mudlet/`](mudlet/). To have Mudlet install it automatically on connect,
publish `mudlet/darkpawns.xml` at a public URL and set:

```bash
export DP_MUDLET_PACKAGE_URL='https://darkpawns.org/darkpawns.xml'
```

That is the URL Dark Pawns itself uses, serving the file from the game deploy
next to the binary. A self-hosted server sets its own URL: pointing at another
server's copy offers players a package built for that server's version.
Keep the file name `darkpawns.xml`, since Mudlet names the installed package
after it.

The server also serves the whole world as a Mudlet map at
`/darkpawns-map.xml`, generated from the live world. Proxy that path to the
game from your front door, and set its public URL to have Mudlet load it:

```bash
export DP_MUDLET_MAP_URL='https://darkpawns.org/darkpawns-map.xml'
```

The server announces the package version compiled into it (`mudlet/VERSION`), and
Mudlet reinstalls the package whenever that version changes, so publish the
file from the same commit you deploy. Unset, nothing is offered and players
can still import the file by hand.

## Run

From the repository root, the defaults are already correct:

```bash
./server
```

The equivalent explicit form, for a layout that is not a checkout or a service
unit that prefers every path spelled out:

```bash
./darkpawns-server \
  -world ./lib/world \
  -web ./web/public \
  -telnet-port 7777
# -db (or DP_SQLITE_PATH) names the SQLite database; the default is lib/data/darkpawns.db.
```

The `-world` directory must contain `wld/`, `mob/`, `obj/`, `zon/`, and `shp/`.
In this checkout that is `lib/world/`; passing `lib/` is rejected at boot with
the reason, instead of failing later as a parser error about `lib/wld`. The
default script directory is `<world>/scripts`, that is `lib/world/scripts/`. The
process anchors its working directory to the parent of `-world` (`lib/`).
Confirm the startup logs show a successful DB connection — a healthy `/health`
alone does not prove persistence is available. A boot that had to *create* the
database file logs a warning saying so, which is how a fresh install is told
apart from the database you expected it to open.

Two flag notes: `-static <dir>` serves a built static site at `/` and takes
precedence over `-web`, and `-hugo` is a deprecated alias for it that still
works and warns. Hugo is no longer part of this repository.

Connect with `telnet localhost 7777` or open `http://localhost:4350`. On an empty
database, the first character becomes the administrator; create that character
before opening access to other players. A second new character follows the normal
mortal entry flow. Save, restart the server, and log back in to verify persistence.

`DP_ALLOW_NO_DB=1` suppresses the embedded default and permits startup without
persistence for ephemeral development and oracle use only. Never use this bypass
in production. Database initialization failure otherwise stops startup by design.

The native build, first-character creation, mortal creation, and saved login after
a full restart were checked on a clean checkout. See the
[verification record](docs/maintenance/native-install-check.md) for scope and commands.

## Audit trail

`<game root>/logs/audit.log` is the append-only (0600) record of admin
world edits and security events — lockouts, bad-password probes, name
validation rejections. Rejected or unvalidated name input is never written
(players often type a password at the name prompt). The IP address is stored
pseudonymously — a truncated, unsalted SHA-256 — which is pseudonymous, not
anonymous. Boot logs loudly if the file cannot be opened; nothing is
silently unrecorded. Back up this file with the game data.

## Optional admin interface

Human telnet and browser play do not require Node.js. To include the React admin
interface, build it and put its output under the server's working directory:

```bash
npm --prefix admin-ui ci
npm --prefix admin-ui run build
rm -rf lib/admin-ui-dist
mkdir -p lib/admin-ui-dist
cp -R admin-ui/dist/. lib/admin-ui-dist/
```

Clear the directory first: vite fingerprints its bundles, so copying over
the top leaves every superseded `index-*.js` and `index-*.css` behind. Two
builds are enough to start serving a directory of orphans, and a stale
bundle sitting next to a live one is easy to read by mistake.

Then restart the server and visit `/admin/`. Both the admin router and `/assets/`
expect this layout by default. The frontend build is verified; administrative
authentication and operations are outside the installation check.

## Supported deployment model

Run the native binary with its state in a local SQLite file, using a service
manager such as systemd for an unattended instance. Docker, Compose, Kubernetes
manifests, and image publishing are retired; they are not maintained
installation options, and CI starts no service containers either: the tests run
against the same embedded SQLite database a fresh installation creates.

Official host-specific service definitions and deploy/rollback procedures belong
in the private ops repository. See the [retirement record](docs/maintenance/container-retirement.md).

## Behind a reverse proxy

Bind the game's HTTP surface to loopback with `-http-bind 127.0.0.1` so
`/ws`, `/api/*` and `/admin/*` are reachable only through the proxy (empty
`-http-bind` keeps the historical all-interfaces listen).

Terminate TLS at the proxy and forward the HTTP routes to `:4350`; expose telnet
(`:7777`) directly since it isn't HTTP. A reference Caddyfile lives under
[`website/deploy/`](website/deploy/). For a TLS telnet port, terminate TLS in front of
the telnet listener (a dedicated port — not STARTTLS).

The server learns each browser player's real address from the proxy's
`X-Forwarded-For`, which it believes only from trusted proxies:
`TRUSTED_PROXIES` (comma-separated CIDRs). Unset, it trusts loopback
(`127.0.0.0/8`, `::1/128`), which covers a proxy on the same machine; set a
proxy on another host explicitly, and set it empty to trust none. Without a
trusted proxy, every web player looks like the proxy: one shared per-address
connection limit, one shared login rate limit, and IP bans that can't tell
players apart.

Per-address connection caps are flood protection, not the multiplay rule
(three characters per player, which immortals enforce, as in the original):
`TELNET_MAX_CONNS_PER_IP` and `WEBSOCKET_MAX_CONNS_PER_IP`, 8 each by
default, room for two players sharing a connection at three characters each.
`TELNET_MAX_CONNS` (default 200) caps telnet overall.

## Entry-identity migration note

A migration adds a unique index on `lower(players.name)`. C's `find_name`
compares names case-insensitively, so the store must too, and the index is
created (if it is missing) on every boot. If your database predates it and holds
two names that differ only in case, the index cannot be built and the boot fails
rather than choosing or deleting a character. Resolve the collision — rename one
of the characters — and restart.

Find the collisions with:

```sql
SELECT lower(name) AS identity, group_concat(id) AS ids
FROM players
GROUP BY lower(name)
HAVING count(*) > 1;
```

## Game-store column migration note

The game store's schema is created and extended by the server itself, on every
boot, from `pkg/db`. The `players` table is created when it is missing, and
columns added since the original schema are appended to an existing table with
`ALTER TABLE ... ADD COLUMN` — skipped when a `pragma_table_info` lookup shows
the column is already present, because SQLite's `ALTER TABLE` has no `IF NOT
EXISTS`. A fresh database and an old one converge on the same schema, and no
stored value is rewritten.

Those columns are declared `TIMESTAMP` (`created_at`, `updated_at`,
`locked_until`) and `JSON` (`inventory`, `equipment`, `character_data`). SQLite
has no zone-aware timestamp type and no JSON canonicalizer, so there is no type
conversion to perform, and the stored JSON payloads round-trip byte for byte.

> **Historical: the retired PostgreSQL backend.** It stored JSON as `jsonb` and
> timestamps as `timestamptz`, and an upgrade used to rewrite those columns on
> first boot. None of that applies to SQLite. Moving an old PostgreSQL database
> across is the one-time procedure in
> [docs/operational/SQLITE-CUTOVER.md](docs/operational/SQLITE-CUTOVER.md).

## Durable player and rent reports migration note

Restoring the player and rent reports added two items to that boot-time schema
work. Both are automatic on the next boot, additive, and idempotent, and the
existing save data is not changed.

- **`players.last_logon`**, a nullable column. It is filled once from
  `players.updated_at` — the best timestamp the existing rows carry — so an
  existing character reports the time of its most recent save. A row with no
  `updated_at` stays `NULL` until that character next saves, and the save then
  writes the current time into it.
- **`object_saves`**, a table holding the C crash/rent-file identity and object
  snapshot. Entering the game never creates a row: a snapshot is written only at
  a real object-save boundary, so a character has no entry until its objects are
  next saved.

## Empty password hashes

Rows created before password hashing was persisted — and any `NULL` carried
across a migration — can hold an empty `password_hash`. Login answers such a
row exactly like a wrong password — same reply bytes, same failure counting
and lockout — so the state cannot be probed from outside and the row can
never authenticate.

Find how many rows are affected, and repair one with an operator-set bcrypt
hash (`$2a$`/`$2b$`, cost 10 or higher):

```sql
SELECT count(*) FROM players WHERE password_hash IS NULL OR password_hash = '';
UPDATE players SET password_hash = '<bcrypt hash>' WHERE name = '<name>';
```

