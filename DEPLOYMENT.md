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
CWD-relative `data/` directory (shops, aliases, mail, admin store). Back up **both**
Postgres and `data/` if you care about the world's state.

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

Copy [`.env.example`](.env.example) and fill in your own values:

```bash
DATABASE_URL=postgres://user:pass@localhost:5432/darkpawns?sslmode=disable
AI_API_KEY=<generate a real key — see below>
```

**Agent API keys:** generate real keys with `go run ./cmd/agentkeygen`. The server
rejects the old example default and any key containing `example`/`test`/`REPLACE_WITH`
(`pkg/db/player.go` → `ValidateAgentKey`), so a placeholder will not authenticate.

## Run

```bash
./darkpawns-server \
  -world ./lib \
  -web ./web \
  -telnet-port 7777
# DATABASE_URL is read from the environment (or pass -db).
```

The process anchors its working directory to the parent of `-world`, so the `data/`
tree resolves next to it. Confirm the startup logs show a successful DB connection —
a healthy `/health` alone does not prove persistence is available.

`DP_ALLOW_NO_DB=1` runs without a database for ephemeral/dev/oracle use only — never
in production; DB-init failure otherwise stops startup by design.

## Docker Compose (local dev)

```bash
docker compose up
```

Brings up Postgres (initialized from [`scripts/init-db.sql`](scripts/init-db.sql)) and
the server. Intended for local development.

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
