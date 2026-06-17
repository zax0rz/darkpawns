# Deployment Guide

How to deploy Dark Pawns server changes to production.

## Server Architecture

- **Host:** CT 120 (192.168.1.121) — bare Debian (dark-pawns VM)
- **Binary:** `/opt/darkpawns/darkpawns-server` (directly on host, no container)
- **World data:** `/opt/darkpawns/lib/`
- **Web assets:** `/opt/darkpawns/hugo-site/` (Hugo site, served by Caddy)
- **Database:** PostgreSQL on localhost (darkpawns:darkpawns-ct120-pg@localhost/darkpawns)
- **Cache:** Redis on localhost
- **Public URL:** https://darkpawns.labz0rz.com (Cloudflare Tunnel)
- **Services:** dark-pawns.service, caddy.service, cloudflared.service, postgresql, redis-server

## Database connection (IMPORTANT)

The server reads its PostgreSQL DSN **only from the `-db` flag** — it does
**not** read `DATABASE_URL` (or any env var) for the database. The flag's
default is `postgres://postgres:postgres@localhost/darkpawns?sslmode=disable`,
which is the wrong role for this host.

If the DSN is missing or wrong, the server does **not** fail — it logs
`Database connection failed, continuing without persistence` and runs with no
database. Returning-player logins, character saves, and moderation all silently
stop working. So a broken `-db` looks like a healthy server.

The `dark-pawns.service` unit's `ExecStart` MUST pass `-db` explicitly, e.g.:

```
ExecStart=/opt/darkpawns/darkpawns-server -world /opt/darkpawns/lib \
  -db "postgres://darkpawns:darkpawns-ct120-pg@localhost/darkpawns?sslmode=disable"
```

After deploying, confirm persistence is actually live (not the silent
fallback):

```bash
ssh root@192.168.1.121 "journalctl -u dark-pawns.service -n 50 --no-pager | grep -i 'database'"
# want: "Database connected."   NOT: "continuing without persistence"
```

## Deploy Steps

### 1. Build (cross-compile for Linux)

The server runs on x86_64 Debian. The mac-mini is arm64. You MUST cross-compile:

```bash
cd darkpawns_repo
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server
```

**Do NOT skip `CGO_ENABLED=0`.** A dynamically-linked binary will crash with missing shared library errors.

**Do NOT skip `GOOS=linux GOARCH=amd64`.** A native build produces an ARM binary that won't run on the x86_64 server.

### 2. Copy to server

SCP to `/tmp/` first, then move into place:

```bash
scp darkpawns-server root@192.168.1.121:/tmp/
ssh root@192.168.1.121 "cp /opt/darkpawns/darkpawns-server /opt/darkpawns/darkpawns-server.bak && mv /tmp/darkpawns-server /opt/darkpawns/darkpawns-server && chmod +x /opt/darkpawns/darkpawns-server"
```

Always keep the previous binary as `.bak` before overwriting.

### 3. Restart the service

```bash
ssh root@192.168.1.121 "systemctl restart dark-pawns.service"
```

### 4. Verify

```bash
# Check service is running
ssh root@192.168.1.121 "systemctl status dark-pawns.service --no-pager | head -10"

# Check logs for startup
ssh root@192.168.1.121 "journalctl -u dark-pawns.service -n 30 --no-pager"

# Health check
curl -s https://darkpawns.labz0rz.com/health
```

Look for: "World loaded", no panics, listening on port 4350.

## Rollback

```bash
ssh root@192.168.1.121 "cp /opt/darkpawns/darkpawns-server.bak /opt/darkpawns/darkpawns-server && systemctl restart dark-pawns.service"
```

## Website Deploy

The Hugo site deploys via Makefile:

```bash
cd darkpawns_repo
make deploy-site
```

This runs: parse world data → Hugo build → rsync to CT 120 `/opt/darkpawns/hugo-site/`.

Caddy serves the Hugo site directly — no restart needed (Caddy auto-detects file changes).

## World Data Changes

If you need to update world files (`.wld`, `.mob`, `.obj`, `.zon`, `.shp`):

```bash
scp -r lib/world/ root@192.168.1.121:/opt/darkpawns/lib/world/
ssh root@192.168.1.121 "systemctl restart dark-pawns.service"
```

World files are reloaded on server start.

## Help Files

Help files live at `/opt/darkpawns/lib/text/help/` on the server. To update:

```bash
scp lib/text/help/* root@192.168.1.121:/opt/darkpawns/lib/text/help/
```

Help files are hot-reloaded — no restart needed.

## Lua Scripts

Behavioral scripts live at `/opt/darkpawns/lib/scripts/mob/`. To update:

```bash
scp lib/world/scripts/mob/*.lua root@192.168.1.121:/opt/darkpawns/lib/scripts/mob/
```

Scripts are hot-reloaded on next mob reset.

## Prerequisites for Agents

To deploy, an agent needs:

1. **SSH access to CT 120** — `root@192.168.1.121` with ed25519 key
2. **Go toolchain** — for cross-compilation
3. **The repo checked out** — `darkpawns_repo/` in workspace

Without SSH access, hand off to someone who has it (Brenda, The Architect).

## Services on CT 120

| Service | Port | Purpose |
|---------|------|---------|
| dark-pawns.service | 4350 (API), 7777 (telnet) | Game server |
| caddy.service | 80 | Reverse proxy (routes /game, /ws, /api, /admin, /health to 4350; serves Hugo site) |
| cloudflared.service | — | Tunnel to Cloudflare → https://darkpawns.labz0rz.com |
| postgresql | 5432 | Game database |
| redis-server | 6379 | Cache/session store |

## Common Failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| `exec format error` | Binary compiled for wrong arch | Rebuild with `GOOS=linux GOARCH=amd64` |
| Service won't start | Binary link error (missing glibc) | Rebuild with `CGO_ENABLED=0` |
| Port already in use | Old process didn't stop | `systemctl stop dark-pawns.service && systemctl start dark-pawns.service` |
| Health check fails | Cloudflare Tunnel issue | Check `systemctl status cloudflared` |
| Mob spawn errors (room not found) | Pre-existing world data issue (zone 142) | Known — not a deployment problem |
