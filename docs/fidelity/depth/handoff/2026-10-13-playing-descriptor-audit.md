# Playing-descriptor audit — DP-1371 (R1, R3, R4, R5b, R5c, R5e, R5g, R5h)

Base: `bbf8fee77` (fresh `origin/main`, includes merged PR #1932). Branch: `codex/luna-playing-descriptor-audit`.

C defines `CON_PLAYING` as state 0 (`src/structs.h:353-385`). The shared Go predicate `Session.isConPlaying` now keeps `Session.IsPlaying` intact for idle/lifecycle callers and excludes menu, character creation, OLC/TEDIT, superseded, linkless, and closed descriptors. It does not exclude PLR_WRITING/PLR_MAILING: those are C character flags while the descriptor remains in `CON_PLAYING`.

## Communication family

| C site / exact gate | Go equivalent / gate | Verdict |
|---|---|---|
| `src/act.comm.c:1122-1124` `do_page`: `!d->connected && d->character` | `cmdPage` all: `isConPlaying` | Fixed; exact dispatch proof covers actor and peer bytes plus menu, creation, REDIT, TEDIT, linkless. |
| `src/act.comm.c:1279-1294` `do_gen_comm`: `!connected`, not actor descriptor, has character, channel enabled, not PLR_WRITING, not soundproof; shout also same zone and awake | `DoChannel`/`mobGlobalGossip`: `playingPlayers` plus the C channel/writing/room/position checks | Fixed; exact gossip bytes and all reachable excluded states covered. |
| `src/act.comm.c:1345-1359` `do_qcomm`: `!connected`, not actor, not PLR_WRITING, quest participant | `broadcastQuest`: `playingPlayers` plus quest/writing filters | Fixed; exact actor/recipient bytes and excluded states covered. |
| `src/act.comm.c:1540-1560` `do_ctell`: `!connected`, same clan/rank, not actor, not NOCTELL | `doCTell`: `playingPlayers` plus clan/rank/no-tell checks | Fixed; exact command-path bytes and excluded states covered. |

The Go-only `[Broadcast]` command in `pkg/session/wiz_system.go:745-782` has no C `ACMD(do_broadcast)` or interpreter registration. C `do_broadcast` is the internal `perform_act` fanout helper in `src/comm.c:2539`, not a player command. Recorded as Go-only under R4; unchanged.

## Broadcast-helper family

| C site / exact gate | Go equivalent / gate | Verdict |
|---|---|---|
| `src/comm.c:2342-2350` `send_to_all`: `!connected` | `World.SendToAll`, `Manager.SendToAll`, script global echo: `isConPlaying` | Fixed; playing string writers still receive as in C. |
| `src/comm.c:2353-2364` `send_to_outdoor`: `!connected && character && AWAKE && OUTSIDE` | `Manager.SendToOutdoor`, script outdoor echo: `isConPlaying && awake && outside` | Fixed; exact recipient bytes tested. |
| `src/comm.c:2366-2374` `send_to_room`: `character->desc` and position greater than sleeping; no `connected` state gate | Room-character output | Out of scope for this state audit: C tests descriptor presence, not connection state. |
| `src/comm.c:2376-2387` `send_to_zone`: `!connected && character && AWAKE && different room && same zone` | `World.SendToZone`, mob/script zone helpers: `playingPlayers` plus awake, different-room and zone checks | Fixed; same-room and sleeping controls are asserted. |
| `src/utils.c:230-240` `mudlog`: `!connected && !PLR_WRITING` | `Manager.EachSession` now supplies `isConPlaying`; `MudLog` retains its PLR_WRITING filter | Fixed; the existing D7 proof did not exercise menu/editor recipients, so this round adds REDIT/TEDIT state cases. |

## Counts and shared state family

| C site / exact gate | Go equivalent / gate | Verdict |
|---|---|---|
| `src/comm.c:903-917` `record_usage`: count every descriptor, then count `!connected` as playing | `Manager.CountSessions`: `hasDescriptor` for connected, `isConPlaying` for playing | Partly fixed: live registered descriptors and playing states now match. Pre-auth sockets are not registered in `Manager.sessions`, so the connected total cannot include them without changing session registration/lifecycle (stop tier). |
| `src/db.c:2290-2302` `is_empty`: return occupied if any `!connected` character is in the zone | `Manager.occupiedZoneRooms`: playing descriptor sessions only | Fixed; unit proof covers menu, creation, editor and linkless bodies. |
| `src/utils.c:691-704` `get_char_by_id`: resolve only under `!d->connected` | Go `huntVictim` player-ID resolution consults `PlayerHasPlayingDescriptor` | Fixed; linkless ID stays unresolved. |
| `src/utils.c:754-768` `is_playing`: exact `connected == CON_PLAYING`; C callers at `src/clan.c:261-272,326-340,1131-1148` | Clan destroy/applicant/rank paths use `World.players` plus store reads/writes; they do not distinguish retained linkless bodies from playing descriptors | Stop-tier: matching the split requires changing clan persistence selection and online/offline record handling. No persistence edits in this round. |
| `src/db.c:2511-2519` untag-on-entry scans only `connected == CON_PLAYING` | No corresponding Go login-entry untag hook found | Stop-tier / missing login lifecycle hook; no login change. |
| `src/objsave.c:1211-1223` `Crash_save_all`: `connected == CON_PLAYING && !NPC` | `Manager.CrashSaveAll` excludes menu/creation/linkless but not editor state | Stop-tier: persistence path; no save scheduling/write change. |
| `src/fight.c:1297-1309` death reward: `!connected && descriptor != actor`, then heal and bless | `counter_procs` uses `playingPlayers` and preserves actor exclusion | Fixed; exact message and HP state tested. |

## R5e call path and lock check

Telnet `Session.TerminalLine` → `terminalCommand` → `sendClientMessage` → `handleMessage(MsgCommand)` → `handleCommand` → `executeCommandRaw` dispatches without `playerLifecycleMu`; `ExecuteCommand` also enters `executeCommandRaw` directly without that lock. The only `playerLifecycleMu` acquisition on the relevant manager side is registration/reconnect/disconnect lifecycle (`Manager.Register`, `HandleTransportDisconnect`, etc.). Manager callbacks snapshot sessions and release `Manager.mu` before calling `Session.isConPlaying`, which takes a short read lock to inspect state; command dispatch does not hold `Manager.mu` at those call sites. `occupiedZoneRooms` evaluates the same predicate with its locked helper while snapshotting switched/player bodies, avoiding races with menu and switch transitions without taking `playerLifecycleMu`. `World.playingPlayers` snapshots/releases `World.mu` before consulting the manager callback. OLC state is an atomic marker because `mudlog` can run while an editor mutex is held.
