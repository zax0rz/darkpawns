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
voice checks, builds Astro, and generates `redirects.caddy`.

### Read the dry run before you sync

The dry run is required, and reading it is the point. `--delete` makes the
docroot match your `dist/` exactly, so anything on production that your branch
does not build is removed. Scan the output for `*deleting` lines and account
for every one:

```bash
rsync -azn --delete --itemize-changes \
  website-astro/dist/ root@192.168.1.121:/srv/hugo/ | grep '^\*deleting'
```

Expect only stale build artifacts: superseded `_astro/*.css` hashes, `.md`
twins for routes that no longer emit them, old `.bak` files. Anything that
looks like real content is a stop.

This is not hypothetical. On 2026-09-04 a deploy from `website-deslop` was
about to delete `/blog/the-long-middle/` and all of `/images/blog/`. The post
was live and published; its source existed only as untracked files in the
`darkpawns-astro-search` worktree, which is where production had last been
deployed from. Nothing in git protected it.

If a `*deleting` line names real content, do not deploy and do not reach for
`--delete`-less rsync as a habit. Find the source, commit it to the branch you
are deploying, rebuild, and dry run again until the only deletions are
artifacts. Deploying different branches to one docroot is what creates this,
so the fix is to make the branch complete, not to make the sync gentler.

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
