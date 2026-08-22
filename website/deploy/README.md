# Website Deployment Files

The canonical production procedure is [`../../DEPLOYMENT.md`](../../DEPLOYMENT.md).

- `Caddyfile` is the repository baseline for static files, reverse-proxy routes,
  content negotiation, and agent-friendly errors.
- Validate a complete staged configuration with production Caddy before reload.
- Production serves `/srv/hugo/` on CT 120 (`192.168.1.121`). References to
  frankendell, `192.168.1.15`, `/opt/darkpawns/hugo-site/`, or Docker Caddy are
  obsolete.
- Do not run `rsync --delete` until routes have been compared with production
  and `/srv/hugo/` has been backed up.
