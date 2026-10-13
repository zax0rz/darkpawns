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

## R5e call path and lock check

Telnet `Session.TerminalLine` → `terminalCommand` → `sendClientMessage` → `handleMessage(MsgCommand)` → `handleCommand` → `executeCommandRaw` dispatches without `playerLifecycleMu`; `ExecuteCommand` also enters `executeCommandRaw` directly without that lock. The only `playerLifecycleMu` acquisition on the relevant manager side is registration/reconnect/disconnect lifecycle (`Manager.Register`, `HandleTransportDisconnect`, etc.). Manager callbacks snapshot sessions and release `Manager.mu` before calling `Session.isConPlaying`, which takes a short read lock to inspect state; command dispatch does not hold `Manager.mu` at those call sites. `World.playingPlayers` snapshots/releases `World.mu` before consulting the manager callback. OLC state is an atomic marker because `mudlog` can run while an editor mutex is held.
