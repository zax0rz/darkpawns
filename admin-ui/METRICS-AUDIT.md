# Dashboard data audit, 2026-10-28

Initial audit before the dashboard implementation: `4fb40c8f0`. Eyeballs B
re-audited producers against `0fdf05bfc`, including entry, guarded death, raw kill
and death traps. The table records the earlier baseline; the correction below
supersedes its death definition and anonymous-only recording policy.

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

The original dashboard recorded no player identities in telemetry. Counters are observed every ten
seconds; event deltas belong to the observation interval ending at that sample,
so an event near midnight can land in the next UTC day. Minute buckets persist
once a minute and on context shutdown; abrupt termination can lose the latest
minute. Offline intervals are gaps, not zero traffic. Coverage is explicit.
Its original event hooks performed atomic increments only, without changing game output,
random draws, persistence formats or command decisions (R1/R3/R4).


## Eyeballs B: character activity and player deaths

| Signal | Confirmed producer before this change | Corrected recording and detail |
| --- | --- | --- |
| Returning logins | Atomic increment before clearing saved wrong-password state; no names | After authentication, restrictions and successful wrong-password reset, record character name, class, level, time and minimum staff visibility. Reconnect/duplicate authentication counts; guests, failed authentication and creation do not. This is authentication, not world entry. |
| New characters | Atomic increment after an accepted character is saved | Same accepted-save seam, once per successful creation, with name/class/level/time. Failed saves and repeated acceptance do not add entries. |
| Deaths | `metrics.Death` includes mobs and sits before player idempotency/HP guards; traps bypass it | Keep that legacy Prometheus/telemetry total intact. New `player_deaths` counts only actual player deaths after the guarded handler, raw-kill extraction and death-trap seams. Record name/class/level/time, room name/vnum, mob name/vnum or cause category. |

### Coverage and fidelity

`game/death.go` records accepted player deaths after extraction/CAS/HP guards.
The raw-kill bridge observes immediately before extraction, after the existing
body handling; the killer body is carried through solely for this observation.
Death traps are observed before their existing extraction. Guest deaths count,
with a generic Guest label. No game output, decisions, save format or random draws
change (R1/R3/R4). Source boundaries read during implementation:
`src/fight.c:534-581`, `src/fight.c:586-629`, and
`src/act.movement.c:288-301`.

Actual killer bodies distinguish mobs from players even when their names collide.
Player killers are labeled “another player”; their identity is not copied. Raw
kills and older spell adapters that supply no killer say “cause not recorded”;
self-inflicted damage says “other damage”. Current spell damage already routes
through `HandleDeath` with the caster; the older `HandleSpellDeath` bridge is
retired from that call path (`pkg/spells/damage_spells.go`). No guessed killer or trap label fills
missing context. The Lua raw-kill adapter preserves a supplied killer body.

### Storage, limits and authorization

A nonblocking 4,096-entry journal copies observations to the history sampler.
Game writers never wait for the sampler or storage. Contention/overrun increments
an explicit dropped-detail count; the aggregate counters still advance. History
saves at most 20,000 detail entries for seven days alongside existing minute
buckets. Capacity removals are reported separately. Detail loss and hidden names
can make aggregate totals exceed visible rows. Abrupt termination can still lose
the latest unsaved minute. Normal shutdown takes a final sample before saving.

The Huma `GET /admin/activity/events` operation uses the existing builder gate
and resolves the caller's current immortal level per request. Visibility follows
the recorded name's minimum level; no token role can raise it. It supports the
same UTC hour/day windows as the dashboard, event-kind and name filters, and
stable before-ID pagination. Responses are no-store. Events contain no host,
IP, credentials, account data, chat or arbitrary log fields. Character names and
world metadata are the only identities/details added.

`activity_since` marks the real start of named detail and separate player-death
recording. Loading an old file keeps its combined death counts in `deaths` and
starts the new coverage at upgrade time. The dashboard reads `player_deaths`;
it never relabels old mob-inclusive data or reconstructs names from logs. Aggregate
counts are assigned to the sample interval; details preserve observation time,
so events near an interval boundary can appear in adjacent bins/list windows.
