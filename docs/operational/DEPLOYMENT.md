# Deployment Guide

How to deploy Dark Pawns server changes to production.

## Server Architecture

- **Host:** frankendell (192.168.1.15) — bare Debian
- **Container:** `darkpawns-server-1` (alpine:3.20)
- **Binary:** `/opt/darkpawns/darkpawns-server` on host → `/app/server` in container (bind mount, read-only)
- **World data:** `/opt/darkpawns/darkpawns/lib/` → `/app/lib/` (bind mount)
- **Web assets:** `/opt/darkpawns/hugo-site/` → `/app/hugo-site/` (bind mount)
- **Database:** PostgreSQL in `darkpawns-postgres-1` container
- **Cache:** Redis in `darkpawns-redis-1` container
- **Public URL:** https://darkpawns.labz0rz.com (Cloudflare Tunnel)

## Deploy Steps

### 1. Build (cross-compile for Linux)

The server runs in an Alpine container on x86_64. You MUST cross-compile:

```bash
cd darkpawns_repo
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server
```

**Do NOT skip `CGO_ENABLED=0`.** Alpine containers don't have glibc. A dynamically-linked binary will crash with `exec format error` or missing shared library errors.

**Do NOT skip `GOOS=linux GOARCH=amd64`.** The mac-mini is arm64. A native build produces an ARM binary that won't run on the x86_64 server.

### 2. Copy to server

SCP to `/tmp/` first, then move into place:

```bash
scp darkpawns-server root@192.168.1.15:/tmp/
ssh root@192.168.1.15 "mv /tmp/darkpawns-server /opt/darkpawns/darkpawns-server && chmod +x /opt/darkpawns/darkpawns-server"
```

**Do NOT SCP directly to `/opt/darkpawns/`.** The bind mount can cause permission issues. Use `/tmp/` as an intermediate.

### 3. Restart the container

```bash
ssh root@192.168.1.15 "docker restart darkpawns-server-1"
```

This picks up the new binary from the bind mount. No Docker rebuild needed.

### 4. Verify

```bash
# Check container is running
ssh root@192.168.1.15 "docker ps --format '{{.Names}} {{.Status}}' | grep darkpawns"

# Check logs for startup
ssh root@192.168.1.15 "docker logs darkpawns-server-1 --tail 30 2>&1"

# Health check
curl -s https://darkpawns.labz0rz.com/health
```

Look for: "World loaded", no panics, listening on port 4350.

## Rollback

```bash
ssh root@192.168.1.15 "cp /opt/darkpawns/darkpawns-server.bak /opt/darkpawns/darkpawns-server && docker restart darkpawns-server-1"
```

Always keep the previous binary as `.bak` before deploying.

## World Data Changes

If you need to update world files (`.wld`, `.mob`, `.obj`, `.zon`, `.shp`):

```bash
scp -r lib/world/ root@192.168.1.15:/opt/darkpawns/darkpawns/lib/world/
ssh root@192.168.1.15 "docker restart darkpawns-server-1"
```

World files are reloaded on server start.

## Help Files

Help files live at `/opt/darkpawns/lib/text/help/` on the server. To update:

```bash
scp lib/text/help/* root@192.168.1.15:/opt/darkpawns/lib/text/help/
```

Help files are hot-reloaded — no restart needed.

## Lua Scripts

Behavioral scripts live at `/opt/darkpawns/lib/world/scripts/mob/`. To update:

```bash
scp lib/world/scripts/mob/*.lua root@192.168.1.15:/opt/darkpawns/lib/world/scripts/mob/
```

Scripts are hot-reloaded on next mob reset.

## Prerequisites for Agents

To deploy, an agent needs:

1. **SSH access to frankendell** — `root@192.168.1.15` with ed25519 key
2. **Go toolchain** — for cross-compilation
3. **The repo checked out** — `darkpawns_repo/` in workspace

Without SSH access, hand off to someone who has it (Brenda, The Architect).

## Common Failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| `exec format error` | Binary compiled for wrong arch | Rebuild with `GOOS=linux GOARCH=amd64` |
| `Permission denied` on SCP | Bind mount permission issue | SCP to `/tmp/` first, then `mv` |
| Container exits immediately | Binary link error (missing glibc) | Rebuild with `CGO_ENABLED=0` |
| Port already in use | Old container didn't stop | `docker stop darkpawns-server-1 && docker start darkpawns-server-1` |
| Health check fails but container runs | Cloudflare Tunnel issue | Check `systemctl status cloudflared` on frankendell |
