# Website Deployment Files

The canonical production procedure is [`../../DEPLOYMENT.md`](../../DEPLOYMENT.md).

- `Caddyfile` is the repository baseline for static files, reverse-proxy routes,
  content negotiation, and agent-friendly errors.
- Validate a complete staged configuration with production Caddy before reload.
- Production serves `/srv/hugo/` on CT 120 (`192.168.1.121`). References to
  frankendell, `192.168.1.15`, `/opt/darkpawns/hugo-site/`, or Docker Caddy are
  obsolete.
- Do not run `rsync --delete` until routes have been compared with production
  and `/srv/hugo/` has been backed up. Read the dry run's `*deleting` lines and
  account for every one: only stale build artifacts belong there. A real page or
  image in that list means its source is missing from the branch being deployed,
  and the fix is to commit the source, not to drop `--delete`. The full check is
  in the root runbook.
