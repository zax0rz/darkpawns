# Dark Pawns website deployment

## Architecture

- Astro builds the site from `website-astro/`.
- Generated map, database, discovery, and compatibility assets remain in
  `website/static/`, which Astro uses as its public directory.
- CT 120 (`192.168.1.121`) runs Caddy as a systemd service.
- Caddy serves Astro output from `/srv/hugo/`. The path is a historical name.
- Cloudflare Tunnel publishes `darkpawns.labz0rz.com` from CT 120.
- `/ws`, `/api/*`, `/admin/*`, `/health`, and `/metrics` proxy to the Go server
  on port 4350.

## Build and deploy

From the repository root:

```bash
make build-site
rsync -azn --delete --itemize-changes \
  website-astro/dist/ root@192.168.1.121:/srv/hugo/
make deploy-site DEPLOY_USER=root DEPLOY_HOST=192.168.1.121
```

`make build-site` regenerates world and database data, runs the content and
voice checks, builds Astro, and generates `redirects.caddy`. The dry run is
required before a destructive sync.

## Caddy configuration

Static content updates do not require a reload. When Caddy policy or redirects
change, install both files together:

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

The generated redirect table turns Astro's static redirect documents into
HTTP 301 responses. The documents remain in the static build as fallbacks.

## Rollback

Before a major deployment, archive `/srv/hugo/` and back up
`/etc/caddy/Caddyfile` plus `/etc/caddy/redirects.caddy`. Restore the static
archive and matching configuration, validate Caddy, then reload it.
