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
| PostgreSQL | `DATABASE_URL` | **Required** — the server refuses logins without persistence |

It also reads/writes on-disk state: the world/scripts tree (`-world`, `-scripts`) and a
CWD-relative `data/` directory (shops, aliases, mail, admin store). With the checkout
layout below, the server changes its working directory to `lib/`, so that state
is in **`lib/data/`**, not the repository-root `data/`. Back up Postgres and the
instance's `lib/` tree, including any edited world files.

## Prerequisites

- Go (see `go.mod` for the version); a C toolchain is not required (`CGO_ENABLED=0`).
- PostgreSQL reachable via `DATABASE_URL`.
- Optionally a reverse proxy (Caddy, nginx, …) terminating TLS in front of `:4350`.

## Build

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server
file darkpawns-server        # ELF 64-bit LSB … x86-64
```

Drop `GOOS`/`GOARCH` to build natively for your own platform.

## Configure

Provision a PostgreSQL role and database. For example, run these as a PostgreSQL
administrator (the commands prompt for the new role's password):

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

## Run

```bash
./darkpawns-server \
  -world ./lib/world \
  -web ./web \
  -telnet-port 7777
# DATABASE_URL is read from the environment (or pass -db).
```

The `-world` directory must contain `wld/`, `mob/`, `obj/`, `zon/`, and `shp/`.
In this checkout that is `lib/world/`; passing `lib/` fails to parse the world.
The default script directory is `lib/world/scripts/`. The process anchors its
working directory to the parent of `-world` (`lib/` with this command).
Confirm the startup logs show a successful DB connection —
a healthy `/health` alone does not prove persistence is available.

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
mkdir -p lib/admin-ui-dist
cp -R admin-ui/dist/. lib/admin-ui-dist/
```

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
