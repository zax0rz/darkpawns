---
title: "The admin console"
description: "Find the admin console page for each task, understand its access gates, and save world or character changes without losing another operator’s work."
type: explainer
audience: operators
verified: v0.1.7
---

The admin console runs at `/admin/` on the game's HTTP listener. Sign in with an
immortal character's name and password. The server checks the character's current
access level on every authenticated request. Keep this listener behind your
configured HTTPS proxy. The session uses an HttpOnly cookie and a CSRF header.

![The dashboard charts show synthetic players online and daily arrivals.](/media/docs/admin-dashboard.png)

## Find the right tool

| Page | Use |
| --- | --- |
| [Dashboard and logs](/docs/host/dashboard-and-logs/) | Current server status, world counts, pending zone saves, process measurements, player activity, and log exports |
| Terminal | Connect to the game through the shared browser client |
| Zones & rooms | Browse zone records, inspect rooms, and open world editors |
| Mobs, Objects, Shops | Search prototypes and open their editors |
| [Workshop & saves](/admin/workshop) | Inspect editing claims and changes waiting to be saved to zone files |
| [Builder guide](/admin/workshop/help) | Read the claim, draft, commit, and save workflow |
| Players | Edit a disconnected character's saved stats, equipment, inventory, password, flags, and identity; HIGOD (staff level 36) access required |
| [MOTD](/docs/host/motd-editor/) | Edit mortal and immortal welcome messages with a terminal preview; builders can read, GRGOD (staff level 38) can save |
| Notifications | Configure every outbound channel: event subscriptions, the gossip mirror, the Healthchecks heartbeat, credentials and test sends; HIGOD (staff level 36) access required |
| Operations | Inspect process metrics, online players, and recent logs; reset actions require their server-side permission |
| Configuration | Edit every runtime setting with live/restart marking, backup and restore, and change history; implementor access required |
| Updates | Check for releases and run the launcher's update transaction with automatic rollback; implementor access required |

Use the search button or `Ctrl+K` (`⌘K` on macOS) to find a page or world record.
The account menu contains the builder guide and sign-out action. Narrow screens
provide a menu button for the full navigation.

For event selection, credentials, delivery status, and troubleshooting, see
[Notifications](/docs/host/notifications/).

## Save deliberately

World editing uses claims to prevent conflicting edits. Edit a draft, review it,
then commit it to the live world. Save the zone's files through the workshop to
retain committed world changes across restarts. Live script controls identify
changes applied directly to the running prototype.

Saved character editing requires the character to be fully disconnected. Stat,
item, and password actions have separate save boundaries. A stale record is
rejected: reload it before trying again. File editors also use conditional writes
with ETags so another editor's changes can't be silently overwritten.

For the rename audit, C-compatible deletion behavior, and flag level gates, see
[Saved character management](/docs/host/player-management/).
