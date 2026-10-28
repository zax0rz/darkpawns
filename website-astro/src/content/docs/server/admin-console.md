---
title: "Admin console"
description: "Sign in, inspect the running game, edit world records, and manage saved characters."
section: "server"
audience: "operator"
order: 15
sourcePath: "website-astro/src/content/docs/server/admin-console.md"
updated: 2026-10-28
draft: false
---

The admin console runs at `/admin/` on the game's HTTP listener. Sign in with an
immortal character's name and password. The server checks the character's current
access level on authenticated requests. Keep this listener behind your configured
HTTPS proxy. The session uses an HttpOnly cookie and a CSRF header.

See [Dashboard and logs](/docs/server/dashboard-and-logs/) for history coverage,
metric definitions, live logs and filtered downloads.

## Find the right tool

| Page | Use |
| --- | --- |
| Dashboard | Current server status, world counts, pending zone saves, process measurements, and recent server log lines |
| Terminal | Connect to the game through the shared browser client |
| Zones & rooms | Browse zone records, inspect rooms, and open world editors |
| Mobs, Objects, Shops | Search prototypes and open their editors |
| Workshop & saves | Inspect editing claims and changes waiting to be saved to zone files |
| Builder guide | Read the claim, draft, commit, and save workflow |
| Players | Edit a disconnected character's saved stats, equipment, inventory, password, flags, and identity; HIGOD access required |
| [MOTD](/docs/server/motd-editor/) | Edit mortal and immortal welcome messages with a terminal preview; builders can read, GRGOD (level 38) can save |
| Notifications | Configure every outbound channel: event subscriptions, the gossip mirror, the Healthchecks heartbeat, credentials and test sends; HIGOD access required |
| Operations | Inspect process metrics, online players, and recent logs; reset actions require their server-side permission |
| Configuration | Edit every runtime setting with live/restart marking, backup and restore, and change history; implementor access required |
| Updates | Check for releases and run the launcher's update transaction with automatic rollback; implementor access required |

Use the search button or `Ctrl+K` (`⌘K` on macOS) to find a page or world record.
The account menu contains the builder guide and sign-out action. Narrow screens
provide a menu button for the full navigation.

Notifications has its own reference page: [Notifications](/docs/server/notifications/)
lists the events, the credential model, delivery semantics and troubleshooting.

## Save deliberately

World editing uses claims to prevent conflicting edits. Edit a draft, review it,
then commit it to the live world. Save the zone's files through the workshop to
retain committed world changes across restarts. Live script controls identify
changes applied directly to the running prototype.

Saved-player editing requires the character to be fully disconnected. Stat,
item, and password actions have separate save boundaries. A stale record is
rejected: reload it before trying again. File editors also use conditional writes
with ETags so another editor's changes cannot be silently overwritten.

For the rename audit, C-compatible deletion behavior, and flag level gates, see
[Saved character management](/docs/server/player-management/).

## Read status accurately

The connection indicator checks the server's `/health` liveness endpoint. It shows
whether HTTP is responding; it does not prove every game subsystem is healthy.
Process metrics and cumulative counters describe the running process. Current
counts and totals are not login trends or historical time series. The console
foundation does not add historical charts.

Errors stay beside the operation or appear as dismissible notifications. Check
Operations for recent server log lines and the terminal for game-side inspection.

## Build the console

From the repository root:

```sh
npm ci --prefix admin-ui
make admin-api
npm run build --prefix admin-ui
npm run lint --prefix admin-ui
```

`make admin-api` exports the registered Huma operations without starting the game,
then generates TypeScript types used by the console's API client. To compare with
a running server, set `ADMIN_OPENAPI_URL` to its `/api/openapi.json` URL.
The source-owned components follow `admin-ui/DESIGN.md` and the site's Haunted
Paperback tokens. Recharts support is available for subsequent dashboard work.
