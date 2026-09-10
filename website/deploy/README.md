# Website Deployment Files

Use the root `make deploy-site` workflow and the dry-run instructions in
[AGENTS.md](../../AGENTS.md). Official host access and service configuration live
in the private ops repository. For the game server, see
[Running Dark Pawns](../../DEPLOYMENT.md).

- `Caddyfile` is the repository baseline for static files, reverse-proxy routes,
  content negotiation, and agent-friendly errors.
- Validate a complete staged configuration with production Caddy before reload.
- The Docker Caddy recipe is retired; the Caddy reference configuration remains.
- Do not run `rsync --delete` until routes have been compared with production
  and `/srv/darkpawns/` has been backed up. Read the dry run's `*deleting` lines and
  account for every one: only stale build artifacts belong there. A real page or
  image in that list means its source is missing from the branch being deployed,
  and the fix is to commit the source, not to drop `--delete`. The full check is
  in the root runbook.
