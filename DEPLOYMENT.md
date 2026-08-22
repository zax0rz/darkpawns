# Dark Pawns Production Deployment

This is the canonical deployment runbook. Production is CT 120 at
`192.168.1.121` (`linux/amd64`). Do not use deprecated hosts or standalone site
clones mentioned in historical notes.

## Production topology

| Component | Production location |
|---|---|
| Game service | `dark-pawns.service` |
| Game binary / previous binary | `/opt/darkpawns/darkpawns-server`, `/opt/darkpawns/darkpawns-server.bak` |
| HTTP/WebSocket / Telnet | `localhost:4350`, `localhost:7777` |
| Static web root | `/srv/hugo/` |
| Caddy service/config | `caddy.service`, `/etc/caddy/Caddyfile` |
| Cloudflare tunnel | `cloudflared.service` |
| World/scripts | `/opt/darkpawns/lib/` |
| PostgreSQL / Redis | `localhost:5432`, `localhost:6379` |

Caddy serves the static site and proxies `/ws`, `/api/*`, `/openapi.json`,
`/admin/*`, `/health`, and `/metrics` to Go. Cloudflare fronts Caddy; verify
both the origin and public URL after a change.

## Preflight

Deploy only a reviewed commit from `main`. Run the repository gates first:

```bash
git switch main
git pull --ff-only origin main
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

Check production before restarting. A restart disconnects players:

```bash
ssh root@192.168.1.121 \
  "systemctl is-active dark-pawns.service caddy.service cloudflared.service && \
   ss -Htn state established '( sport = :7777 or sport = :4350 )'"
```

If connections are listed, announce maintenance or choose another window.

## Deploy the Go server

Build an explicit Linux x86-64 artifact and verify its format:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -o darkpawns-server ./cmd/server
file darkpawns-server
# Must report: ELF 64-bit LSB ... x86-64
```

Stage without replacing the running executable:

```bash
scp darkpawns-server \
  root@192.168.1.121:/opt/darkpawns/darkpawns-server.new
ssh root@192.168.1.121 "file /opt/darkpawns/darkpawns-server.new"
```

Back up, swap, restart, and wait for systemd. Do not assume a fixed
three-second startup or retry a timed-out restart blindly—inspect service state.

```bash
ssh root@192.168.1.121 "\
  cp /opt/darkpawns/darkpawns-server \
     /opt/darkpawns/darkpawns-server.bak && \
  mv /opt/darkpawns/darkpawns-server.new \
     /opt/darkpawns/darkpawns-server && \
  chmod 0755 /opt/darkpawns/darkpawns-server && \
  systemctl restart dark-pawns.service"

ssh root@192.168.1.121 "\
  systemctl is-active dark-pawns.service && \
  curl --fail --silent http://localhost:4350/health && echo && \
  journalctl -u dark-pawns.service --no-pager -n 30"
```

Confirm startup logs include a successful database connection. Healthy HTTP
alone does not prove persistence is available.

## Deploy Caddy configuration

Never overwrite the live Caddyfile without preserving its security headers,
redirect import, and production-only routes. First retrieve the live file,
merge the reviewed routing change into it, then stage and validate that complete
candidate:

```bash
scp root@192.168.1.121:/etc/caddy/Caddyfile Caddyfile.production
# Merge the reviewed website/deploy/Caddyfile change into Caddyfile.production.
scp Caddyfile.production root@192.168.1.121:/etc/caddy/Caddyfile.new
ssh root@192.168.1.121 "\
  /usr/local/bin/caddy validate --adapter caddyfile \
    --config /etc/caddy/Caddyfile.new && \
  cp /etc/caddy/Caddyfile /etc/caddy/Caddyfile.bak && \
  mv /etc/caddy/Caddyfile.new /etc/caddy/Caddyfile && \
  /usr/local/bin/caddy reload --config /etc/caddy/Caddyfile && \
  systemctl is-active caddy.service"
```

Do not replace production with the reduced repository baseline verbatim.

## Deploy the website

The public files currently in `/srv/hugo/` include the newer generated site and
machine-readable Markdown siblings. A full `rsync --delete` from a different
generator can erase valid routes. Before a full site deployment:

1. Run `make build-site` successfully.
2. Preview `website/public/` and compare its route inventory with `/srv/hugo/`.
3. Back up `/srv/hugo/` on CT 120.
4. Only then run the guarded target:

```bash
make deploy-site \
  DEPLOY_USER=root \
  DEPLOY_HOST=192.168.1.121 \
  DEPLOY_PATH=/srv/hugo/
```

For a small machine-readable change, stage and copy only the named files rather
than replacing the visual site. Keep HTML and Markdown representations paired.

## Required verification

```bash
# Origin
ssh root@192.168.1.121 "\
  systemctl is-active dark-pawns.service caddy.service cloudflared.service && \
  curl --fail --silent http://localhost:4350/health"

# Public HTML and error behavior
curl -sS -o /dev/null -w '%{http_code} %{content_type}\n' \
  https://darkpawns.labz0rz.com/
curl -sS -o /dev/null -w '%{http_code} %{content_type}\n' \
  https://darkpawns.labz0rz.com/path-that-does-not-exist

# Public agent surfaces
curl -sS -o /dev/null -w '%{http_code} %{content_type}\n' \
  https://darkpawns.labz0rz.com/openapi.json
curl -sS -D - -o /dev/null -H 'Accept: text/markdown' \
  https://darkpawns.labz0rz.com/
curl -sS -o /dev/null -w '%{http_code} %{content_type}\n' \
  -H 'Accept: text/markdown' \
  https://darkpawns.labz0rz.com/path-that-does-not-exist
```

Expected: health `OK`; HTML home `200`; missing HTML and Markdown paths `404`;
OpenAPI `200 application/json`; negotiated pages `text/markdown`; negotiated
responses include `Vary: Accept`.

## Rollback

Binary:

```bash
ssh root@192.168.1.121 "\
  cp /opt/darkpawns/darkpawns-server.bak \
     /opt/darkpawns/darkpawns-server && \
  chmod 0755 /opt/darkpawns/darkpawns-server && \
  systemctl restart dark-pawns.service"
```

Caddy:

```bash
ssh root@192.168.1.121 "\
  cp /etc/caddy/Caddyfile.bak /etc/caddy/Caddyfile && \
  /usr/local/bin/caddy validate --config /etc/caddy/Caddyfile && \
  /usr/local/bin/caddy reload --config /etc/caddy/Caddyfile"
```

Use the timestamped site backup made before a full static deployment to restore
`/srv/hugo/`. Report what was rolled back and retain failed artifacts for diagnosis.
