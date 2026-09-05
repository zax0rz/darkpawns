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

> **The restart can block for up to ~90s.** If the outgoing binary does not exit
> on SIGTERM, systemd waits out `TimeoutStopSec` (~90s) and then SIGKILLs the
> process group (`Main process exited, code=killed, status=9/KILL`; the *stop*
> records `Result=timeout`). The combined swap+restart `ssh` above will stay open
> for that whole window, so a client-side timeout shorter than ~90s will cut the
> connection **while the restart is still finishing** — this is expected, not a
> failure, and the new process still comes up. Do **not** re-run the restart.
> Instead re-`ssh` and inspect, distinguishing "still finishing" from
> "crash-looping":
>
> ```bash
> ssh root@192.168.1.121 "\
>   systemctl show dark-pawns.service \
>     -p ActiveState,SubState,MainPID,NRestarts,Result,ExecMainStartTimestamp"
> ```
>
> A healthy result is `ActiveState=active`, `Result=success`, `NRestarts=0`, and a
> `MainPID`/`ExecMainStartTimestamp` dated *after* you triggered the deploy. A
> climbing `NRestarts` or a start timestamp that keeps moving means it is
> crash-looping — roll back. A forced SIGKILL of the *outgoing* process is safe
> only when no players are connected (world state is file-loaded and persistence
> is in Postgres); with players online, prefer a graceful window.

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

1. **Service is active:** `systemctl is-active dark-pawns.service` → `active`
2. **Health endpoint:** `curl http://localhost/health` → `OK`
3. **Game port:** `curl http://localhost:4350/health` → `OK`
4. **Logs are alive:** `journalctl -u dark-pawns.service --no-pager -n 5` — expect world-loading warnings (missing mob scripts are normal), no fatal errors

## Website deployment

The public site is Astro. Hugo has been removed from the repository; only the
shared generated assets, the parse scripts and the Caddy configuration remain
under `website/`.

Production runs on CT 120 at `192.168.1.121`. Caddy serves `/srv/hugo/`; the
directory name is historical and does not mean the deployed site uses Hugo.

```bash
# Build shared map/database data, run site checks, build Astro, and generate
# the Caddy redirect table.
make build-site

# Review the destructive sync before deployment. Read the *deleting lines:
# --delete makes the docroot match dist/ exactly, so anything production
# serves that this branch does not build is removed. Only stale build
# artifacts are expected. Real content in that list means stop.
rsync -azn --delete --itemize-changes \
  website-astro/dist/ root@192.168.1.121:/srv/hugo/ | tee /tmp/dryrun.txt
grep '^\*deleting' /tmp/dryrun.txt

# Deploy static files. DEPLOY_PATH defaults to /srv/hugo/.
make deploy-site DEPLOY_USER=root DEPLOY_HOST=192.168.1.121
```

When redirects or Caddy policy change, validate and install both configuration
files before reloading Caddy:

```bash
scp website/deploy/Caddyfile website/deploy/redirects.caddy \
  root@192.168.1.121:/tmp/
ssh root@192.168.1.121
mv /tmp/Caddyfile /tmp/Caddyfile.astro
caddy validate --config /tmp/Caddyfile.astro
install -m 644 /tmp/Caddyfile.astro /etc/caddy/Caddyfile
install -m 644 /tmp/redirects.caddy /etc/caddy/redirects.caddy
caddy validate --config /etc/caddy/Caddyfile
systemctl reload caddy
```

Deploying more than one branch to this single docroot is what makes the dry
run load-bearing: production can hold pages whose source is not in the branch
you are shipping. See `website/deploy/README.md` for the failure this caught
and what to do about it.

Before a major cutover, archive `/srv/hugo/` under
`/opt/darkpawns/site-backups/`. A static-site rollback restores that archive
and the corresponding backed-up Caddy configuration; it does not require a
game-server rollback.

## Website contact form

The public form posts to `/api/contact`, which Caddy already proxies to the Go
server. Install these values as private systemd service environment variables:

```text
CONTACT_TO=
CONTACT_SMTP_HOST=smtp.gmail.com
CONTACT_SMTP_PORT=587
CONTACT_SMTP_USER=
CONTACT_SMTP_PASSWORD=
CONTACT_TURNSTILE_SECRET=
CONTACT_TURNSTILE_HOSTNAMES=darkpawns.labz0rz.com
```

`CONTACT_TO` is the private destination. `CONTACT_SMTP_PASSWORD` should be a
Google app password, not the account password. The static Astro build also
requires the public `PUBLIC_TURNSTILE_SITE_KEY` value in the ignored
`website-astro/.env` file. Production loads the private values from
`/etc/dark-pawns/contact.env` through a systemd override. The form remains
disabled when the public key is absent, and the endpoint returns 503 when any
private delivery setting is missing.

## Notes

- **Don't deploy from `go run` or `go build` with no output flag.** The binary must be explicitly named and cross-compiled.
- **The `.bak` file is overwritten each deploy.** If you need to keep a specific snapshot, copy it elsewhere first.
- **Systemd restart kills active connections.** Players get disconnected. Deploy during low-traffic windows or announce first.
- **For Go-only changes that don't touch config or scripts**, a simple binary swap + restart is sufficient. For config/script changes, sync those separately.

Use the timestamped site backup made before a full static deployment to restore
`/srv/hugo/`. Report what was rolled back and retain failed artifacts for diagnosis.
