# Deployment Guide

## Quick Reference

| What | Value |
|------|-------|
| CT / IP | **CT 120 — 192.168.1.121** |
| OS | Linux x86\_64 (Proxmox LXC) |
| Service | `dark-pawns.service` (systemd) |
| Binary | `/opt/darkpawns/darkpawns-server` |
| Backup | `/opt/darkpawns/darkpawns-server.bak` |
| Listen ports | 4350 (HTTP/WS game), 7777 (telnet) |
| Proxy | Caddy on :80 → :4350 |
| Tunnel | Cloudflare (`cloudflared.service`) |
| DB | Postgres — `127.0.0.1:5432` |
| Cache | Redis — `127.0.0.1:6379` |
| Web root | `/opt/darkpawns/web` |
| Scripts | `/opt/darkpawns/lib/scripts` |

## Build Platform

Production (CT 120) is **linux/amd64**. The build artifact must be an
`ELF 64-bit LSB x86-64` binary — anything else fails with `status=203/EXEC`.

### Native build (Linux dev workstation) — preferred

On a linux/amd64 workstation the build is native; no cross-compile flags:

```bash
cd darkpawns
go build -o darkpawns-server ./cmd/server
```

### Cross-compile (mac-mini fallback)

When building on the mac-mini (`192.168.1.196`, darwin/arm64) — e.g. during the
dev-environment transition — you **must** cross-compile, or the binary is Mach-O
and won't run on CT 120:

```bash
cd /Users/zach/darkpawns
GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server
```

### Verify the binary (either path)

```bash
file darkpawns-server
# Expected: "ELF 64-bit LSB executable, x86-64, ..."
# Wrong:   "Mach-O 64-bit executable arm64"  ← you built on the mac without cross-compile flags
```

## Deploy Steps

```bash
# 1. Build. On Linux/amd64: `go build -o darkpawns-server ./cmd/server`
#    On the mac-mini, cross-compile (see Build Platform above):
GOOS=linux GOARCH=amd64 go build -o darkpawns-server ./cmd/server

# 2. Copy to CT 120 as .new (so the old binary stays in place until we're ready)
scp darkpawns-server root@192.168.1.121:/opt/darkpawns/darkpawns-server.new

# 3. On CT 120: backup current binary, swap, restart, verify
ssh root@192.168.1.121 "\
  cp /opt/darkpawns/darkpawns-server /opt/darkpawns/darkpawns-server.bak && \
  mv /opt/darkpawns/darkpawns-server.new /opt/darkpawns/darkpawns-server && \
  chmod +x /opt/darkpawns/darkpawns-server && \
  systemctl restart dark-pawns.service && \
  sleep 3 && \
  systemctl is-active dark-pawns.service && \
  echo '=== Health ===' && \
  curl -s http://localhost/health && \
  echo
"

# 4. Clean up local binary
rm darkpawns-server
```

## Rollback

```bash
ssh root@192.168.1.121 "\
  cp /opt/darkpawns/darkpawns-server.bak /opt/darkpawns/darkpawns-server && \
  systemctl restart dark-pawns.service
"
```

## Verification

After deploy, check:

1. **Service is active:** `systemctl is-active dark-pawns.service` → `active`
2. **Health endpoint:** `curl http://localhost/health` → `OK`
3. **Game port:** `curl http://localhost:4350/health` → `OK`
4. **Logs are alive:** `journalctl -u dark-pawns.service --no-pager -n 5` — expect world-loading warnings (missing mob scripts are normal), no fatal errors

## Notes

- **Don't deploy from `go run` or `go build` with no output flag.** The binary must be explicitly named and cross-compiled.
- **The `.bak` file is overwritten each deploy.** If you need to keep a specific snapshot, copy it elsewhere first.
- **Systemd restart kills active connections.** Players get disconnected. Deploy during low-traffic windows or announce first.
- **For Go-only changes that don't touch config or scripts**, a simple binary swap + restart is sufficient. For config/script changes, sync those separately.
