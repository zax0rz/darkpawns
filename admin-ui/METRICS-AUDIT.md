# Dashboard data audit, 2026-10-27

Audited before B+C UI implementation against origin/main `4fb40c8f0`.

| Signal | Existing producer and exposure | Limitation / B+C action |
| --- | --- | --- |
| Online now | `World.GetPlayerCount`, `/admin/server`, `/admin/metrics` | World-registered characters. Prometheus sampler used `Manager.SessionCount` (connections, including login screens); align with world count. |
| Online history | None in server | Record 10-second observations in minute buckets, retain seven days. Hourly/day chart uses latest observation and observed peak; gaps stay null. |
| Connections | `telnet/listener.go`, `session/session_pump.go` feed active/total counters | Includes both transports despite historic WebSocket help text; connections are not logins. |
| Logins | No event history. `RecordLoginSuccess` clears lockout; `last_logon` is last save | Add atomic count after successful returning-character authentication, including duplicate/reconnect authentication. Guests and new character creation excluded. |
| New characters | `players.created_at` exists in SQLite but is not exposed as history | Imports/replacements are not necessarily gameplay creation. Record only successful accepted-character persistence in `char_creation.go`. |
| Deaths | `metrics.Death` in `game/death.go` player and mob paths | Existing cumulative total includes players and mobs. Add observer count at this existing chokepoint; label combined total. |
| Combat / damage | `combat/engine.go`, `game/player_affects.go`, `game/mob.go`, `game/shoot.go` | Real cumulative counters, no retained series. Not used as activity substitutes. |
| Commands | `session/commands.go` records class/duration | Real cumulative counters/histograms; no historical query service. |
| Generic errors | `darkpawns_errors_total` registered, no production callers | Do not chart. Count actual emitted WARN/ERROR structured server records in admin slog handler instead. |
| Database queries | Counters/histogram registered, no production callers | Do not chart. |
| Runtime / uptime | `metricsSnapshot` reads runtime and `processStartTime` | Real current snapshot; no historical heap series. |
| Rooms / mobs | Ten-second sampler in `cmd/server/main.go` | Room gauge counts loaded rooms, despite help saying rooms with players. Not an activity signal. |
| Release / last update | No operator release API. Portable manifest includes version/commit; Go build includes VCS when available | Read running executable identity and bundle metadata. Record observed executable changes durably. First observation has unknown last update; never substitute build time or uptime. Development builds say development. |
| Server log | `LogBuffer`, `/admin/logs`, 1000-line volatile ring | No full-file archive. Downloads cover the retained filtered ring only; .log preserves stored lines, CSV has time/level/type/message. |
| Mudlog | Tap → feed queue/ring → `/admin/mudlog`, catalog, SSE | 1000 events, nonblocking and potentially dropped. Type is OFF/BRF/NRM/CMP, level is minimum immortal visibility, neither is WARN/ERROR. Preserve actual-caller ceiling in list and exports. |

B+C records no player identities in telemetry. Counters are observed every ten
seconds; event deltas belong to the observation interval ending at that sample,
so an event near midnight can land in the next UTC day. Minute buckets persist
once a minute and on context shutdown; abrupt termination can lose the latest
minute. Offline intervals are gaps, not zero traffic. Coverage is explicit.
All event hooks perform atomic increments only, without changing game output,
random draws, persistence formats or command decisions (R1/R3/R4).
