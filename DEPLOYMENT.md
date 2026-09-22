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
| SQLite (embedded) | `lib/data/darkpawns.db` | Default game persistence — no setup, no external services |
| PostgreSQL | `-db` / `DATABASE_URL` | Opt-in — agent/research sidecar corpus only |

It also reads/writes on-disk state: the world/scripts tree (`-world`, `-scripts`) and a
CWD-relative `data/` directory (shops, aliases, mail, admin store). With the checkout
layout below, the server changes its working directory to `lib/`, so that state
is in **`lib/data/`**, not the repository-root `data/`. The admin audit trail follows
the same rule: the log is **`lib/logs/audit.log`** (mode `600`, in a `750` directory it
creates on first boot). Back up the SQLite database and the instance's `lib/` tree, including any
edited world files (plus Postgres if you run the research sidecar).

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
- No database server: with no `-db` flag and no `DATABASE_URL`, the server
  creates an embedded SQLite database at `<world>/../data/darkpawns.db` and
  boots against it.
- PostgreSQL is the opt-in backend for scaled deployments: pass a
  `postgres://` URL via `-db` or `DATABASE_URL`. The research corpus
  (decision_log, combat_log) is a separate, separately-enabled database — see
  [The research corpus](#the-research-corpus-optional) below.
- Optionally a reverse proxy (Caddy, nginx, …) terminating TLS in front of `:4350`.

## Build

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server
file darkpawns-server        # ELF 64-bit LSB … x86-64
```

Drop `GOOS`/`GOARCH` to build natively for your own platform.

## Configure

PostgreSQL is opt-in. To use it, provision a role and database; for example, run
these as a PostgreSQL administrator (the commands prompt for the new role's
password):

```bash
createuser --pwprompt darkpawns
createdb --owner=darkpawns darkpawns
```

The application creates and migrates its schema at startup. The role must own
the database or have equivalent schema creation privileges. Do not load the
legacy `scripts/init-db.sql` into a new native installation.

Export your connection string and a signing secret:

```bash
export DATABASE_URL='postgres://darkpawns:YOUR_PASSWORD@localhost:5432/darkpawns?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
```

`postgres://user:password@localhost:5432/darkpawns` connects over TCP and asks
for the role's password; the three-slash form with `host=/var/run/postgresql`
connects over the Unix socket, where the server trusts your operating-system
identity instead. If you created the database as yourself, the socket form needs
no password and no role name.

Replace `YOUR_PASSWORD` with the role's password (URL-encode special characters).
Keep the signing secret private and stable across restarts. Startup
rejects a missing or short `JWT_SECRET` unless `ENVIRONMENT=development`, which
uses an ephemeral secret. The `AI_API_KEY` is for agent access, not human telnet
login.

Alternatively, copy [`.env.example`](.env.example), edit it, and export it from
your shell. The native binary **does not load `.env` automatically**:

```bash
set -a
. ./.env
set +a
```

**Agent API keys:** generate real keys with `go run ./cmd/agentkeygen`. The server
rejects the old example default and any key containing `example`/`test`/`REPLACE_WITH`
(`pkg/db/player.go` → `ValidateAgentKey`), so a placeholder will not authenticate.

### The research corpus (optional)

Decision capture — the corpus behind the research tracks — has its own
database, named by `DP_RESEARCH_URL`, and is **not** the game database:
choosing PostgreSQL for the game does not record anything. The corpus uses
PostgreSQL-native range partitioning, so the URL must be a `postgres://` one;
provision it like the game database (own role and database recommended).

```bash
export DP_RESEARCH_URL='postgres://darkpawns:YOUR_PASSWORD@localhost:5432/darkpawns_research?sslmode=disable'
```

With `DP_RESEARCH_URL` set, capture is **available but off** at boot. It is
turned on deliberately, per run, through the admin console:

```bash
curl -X POST -H 'Authorization: Bearer <builder-token>' \
  -d '{"enabled": true}' http://localhost:4350/admin/research/capture
```

`GET` on the same endpoint reports the current state. The control names what
it records because it should be said where it is operated: **the literal
command text of every player, tells and says included**, written verbatim to
`decision_log.raw_input` (player names are salted hashes; see the boot-time
salt warning). Disabling flushes what is buffered, so the last second of a
run lands; re-enabling works without a restart. An unreachable research
database does not stop the game — boot logs an error and capture stays
unavailable for that run.

Retention is whole monthly partitions: set `DP_LOG_RETENTION_MONTHS` to have
expired partitions dropped automatically; without it the corpus is retained
indefinitely. Until it is enabled, no connection to the research database is
made beyond schema/partition setup.

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
# DATABASE_URL is read from the environment (or pass -db).
```

The `-world` directory must contain `wld/`, `mob/`, `obj/`, `zon/`, and `shp/`.
In this checkout that is `lib/world/`; passing `lib/` is rejected at boot with
the reason, instead of failing later as a parser error about `lib/wld`. The
default script directory is `<world>/scripts`, that is `lib/world/scripts/`. The
process anchors its working directory to the parent of `-world` (`lib/`).
Confirm the startup logs show a successful DB connection —
a healthy `/health` alone does not prove persistence is available.

Two flag notes: `-static <dir>` serves a built static site at `/` and takes
precedence over `-web`, and `-hugo` is a deprecated alias for it that still
works and warns. Hugo is no longer part of this repository.

Connect with `telnet localhost 7777` or open `http://localhost:4350`. On an empty
database, the first character becomes the administrator; create that character
before opening access to other players. A second new character follows the normal
mortal entry flow. Save, restart the server, and log back in to verify persistence.

`DP_ALLOW_NO_DB=1` permits a failed database connection for ephemeral/dev/oracle
use only; a nonempty database URL is still required. Never use this bypass in
production. Database initialization failure otherwise stops startup by design.

The native build, first-character creation, mortal creation, and saved login after
a full restart were checked on a clean checkout. See the
[verification record](docs/maintenance/native-install-check.md) for scope and commands.

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

Run the native binary with PostgreSQL, using a service manager such as systemd
for an unattended instance. Docker, Compose, Kubernetes manifests, and image
publishing are retired; they are not maintained installation options. CI still
uses disposable service containers to test database-backed behavior.

Official host-specific service definitions and deploy/rollback procedures belong
in the private ops repository. See the [retirement record](docs/maintenance/container-retirement.md).

## Behind a reverse proxy

Terminate TLS at the proxy and forward the HTTP routes to `:4350`; expose telnet
(`:7777`) directly since it isn't HTTP. A reference Caddyfile lives under
[`website/deploy/`](website/deploy/). For a TLS telnet port, terminate TLS in front of
the telnet listener (a dedicated port — not STARTTLS).

## Entry-identity migration note

A migration adds a unique index on `lower(players.name)`. If your database predates it,
resolve any case-folding name collisions first — the migration deliberately fails rather
than choosing or deleting a character:

```sql
SELECT lower(name) AS identity, array_agg(id ORDER BY id) AS ids
FROM players GROUP BY lower(name) HAVING count(*) > 1;
```

## Game-store column migration note

The first boot after upgrading also rewrites `players.inventory` and
`players.equipment` from `jsonb` to `json`. `jsonb` canonicalizes whatever is
written to it (keys reordered by length, spacing normalized), which made
byte-identical save→load impossible; `json` stores the input text exactly and
still validates it as JSON on write. Legacy values already canonicalized by
`jsonb` come across in that form — the original byte layout of old rows is not
recoverable — and everything written after the change is preserved verbatim.
It happens on the same first boot, takes the same one-time table lock, and
logs the same way (`converted game-store column to json column=players.inventory`).

The game store's timestamp columns (`players.created_at`, `players.updated_at`,
`players.locked_until`, `agent_keys.created_at`) are declared `timestamptz`. A
database created by an earlier build holds them as naive `timestamp`, which
records a wall clock with no zone attached, so lockout and expiry comparisons
land hours off on a host that is not UTC. **Restarting is the whole procedure.**
The first boot after the upgrade rewrites those columns and says so, one line
per column:

```
INFO converted game-store column to timestamptz column=players.locked_until
```

Values are reinterpreted in the database session's zone (`ALTER COLUMN ... TYPE
timestamptz USING column AT TIME ZONE current_setting('TimeZone')`), which keeps
the wall clock that is stored and attaches that zone to it. Rows written before
the upgrade therefore read back with the same wall clock in that zone, and every
row written after it is an exact instant. The stored value was a wall clock with
no zone attached, so the writing zone had to be supplied from somewhere: the
session's zone is the one the provisioning in this document sets up, and naming
it instead of assuming UTC is what keeps a lockout that is still in flight in
force through the restart. A row written while the database ran in a different
zone is read in the current one — the writing zone is not recorded anywhere, so
no conversion can recover it.

The rewrite takes an exclusive lock on the table once, at that first boot (a
few thousand rows in `players`). Every boot after it finds the columns already
zone-aware and does nothing, and a conversion interrupted part way through
finishes the remaining columns on the next boot. If the database role lacks
`ALTER` on the table, boot fails with `migrate naive timestamps: ...` instead
of starting against a schema it cannot trust — grant `ALTER` and restart.
SQLite needs none of this: it has no zone-aware type, and the DDL translation
already folds `TIMESTAMPTZ` back to `TIMESTAMP` there.
