---
title: Dashboard and logs
description: Read recorded player activity, inspect server health, and export retained logs.
section: server
audience: operator
order: 16
sourcePath: website-astro/src/content/docs/server/dashboard-and-logs.md
draft: false
updated: 2026-10-28
---

The admin dashboard opens at `/admin/`. It puts server health and player
activity first. Every tile opens operations, an activity detail, or filtered logs.
The console requires a builder role for telemetry and logs.

## Read the dashboard

Choose **24 hours** or **7 days** for online history and the count tiles. The
online chart shows the latest observation and observed peak in each UTC hour
or day. Daily arrivals always shows seven UTC days of logins and new characters.
Activity detail includes charts for deaths, errors and warnings, plus a table
with sample coverage. Dates and daily boundaries use UTC.

- **Players online** counts characters registered in the world, including bodies
  still present after a connection loss. Login screens are not players online.
- **Logins** counts successful returning-character authentication, including
  authenticated duplicate/reconnect attempts. Guests and new characters are
  excluded. It is not a count of unique people.
- **New characters** counts accepted character creation after persistence succeeds.
- **Player deaths** counts players, including death traps. Mob deaths do not
  count in this tile. Guest deaths use a generic Guest name.
- **Errors and warnings** count emitted structured server log records. Mudlog's
  OFF/BRF/NRM/CMP settings are visibility types, not severity levels.
- **Uptime** belongs to the current process. **Current release** comes from the
  running portable bundle manifest or available Go build metadata. Development
  executables identify themselves as development builds.

The server samples every ten seconds, stores minute buckets, and retains seven
days in `DARKPAWNS_DATA_DIR/admin-history.json`. This file is separate from player
saves. It is written once a minute and during normal context shutdown. An abrupt
termination can lose the latest minute. Events are assigned to the observation
interval ending at the sample, so an event within ten seconds of midnight can
be counted in the next day.

A fresh installation has no historical data. Gaps are not zeroes. Totals cover
observed periods only; partial days do not imply full-day traffic. The detail
table shows sampling coverage. A storage failure is visible on the dashboard.
Preserve the history file across upgrades if you want continuity.

The last update time is recorded when a startup observes a changed executable
identity. The first observation has no evidence of a previous update, so it
shows unknown. Restarting the same executable does not count as an update.
A build timestamp is never presented as a deployment timestamp.

## Follow character activity

Open **Logins**, **New characters** or **Player deaths** to see the characters
behind the count. Find a character by name, choose a time window, or use
**Older entries** to page back. Each entry shows its time and class. Deaths also
show the room and what killed the character: a mob name and vnum, another player,
or a trap. If the game supplied no cause, the page says so.

Names of hidden staff follow your current immortal level. Counts can include
activity whose names you cannot see. Player killers are labeled **Another player**.
The list saves character names and game details, with no addresses, credentials
or chat. Guests do not count as returning-character sign-ins.

Details are saved for seven days, up to 20,000 entries. The page shows when this
recording began. Older releases saved counts without names and included mobs in
the death total; those totals are not reused as player deaths. A full recorder
can miss details while keeping the count. The page warns when entries were
missed or removed at the limit. Totals can therefore exceed the visible list.

### How it works

Activity details and separate player-death counts share the existing history
file and save schedule. Detail timestamps mark the observation; count buckets
mark the next sample. An event close to an hour or day boundary can fall into
adjacent chart and list windows. Normal shutdown samples once more before saving.

## Inspect and export logs

Open **Logs** in the sidebar or `/admin/operations/logs`. The original server
tail remains in Operations. The log view refreshes server records every three
seconds. Mudlog uses its existing live SSE stream to trigger filtered snapshots,
with periodic refresh as recovery. Pause and resume affect live refresh.

Choose **Server log** or **Mudlog**, then filter messages and UTC time bounds.
Server logs also filter by severity. Mudlog filters by syslog type, minimum
immortal level ceiling and exact site key. A requested level can narrow access;
it cannot grant access above the caller's actual immortal level. Negative-level
file-only mudlog events are excluded from the operator view and exports.

**Raw .log** downloads stored server lines or mudlog message text, followed by
newlines. **Filtered CSV** includes `time,level,type,message`. Both formats apply
the current filters on the server. CSV quotes embedded commas and newlines;
potential spreadsheet formulas receive a leading apostrophe. Raw message text
is unchanged.

Downloads cover the retained ring, up to 1,000 entries, rather than the full
on-disk log archive. Live events can arrive between viewing and downloading.
The mudlog observer reports dropped events; a nonzero count means the record is
partial. Daily dashboard error counts can exceed the available error log lines
because history outlives the log ring. No endpoint reads arbitrary server files.

### Which log should I read?

**Server log** shows startup, connections, storage and service errors.
**Mudlog** shows in-game events recorded for staff, limited to your immortal
level. They are separate sources, so the same lines need not appear in both.
