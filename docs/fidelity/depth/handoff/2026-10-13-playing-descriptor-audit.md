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

## Wizard and information family

| C site / exact gate | Go equivalent / gate | Verdict |
|---|---|---|
| `src/act.wizard.c:341-359` transfer-all: `!connected && character && character != actor`, then lower level | transfer-all session selection uses `isConPlaying` before level/room work | Fixed; exact room output and state tested. |
| `src/act.wizard.c:1690-1707` gecho: `!connected && character && character != actor` | `cmdGechoText`: `isConPlaying` before actor exclusion | Fixed; real dispatch proof pins exact actor and peer bytes. |
| `src/act.wizard.c:1894-1906` force-all: skip connected/no-character/equal-or-higher; then force | `forceTargets` global branch uses `isConPlaying` before level comparison; same-room branch stays room-character based, like C | Fixed; real dispatch proof executes forced command only for eligible playing descriptors. |
| `src/act.wizard.c:1948-1980,2008-2020` wiznet listing/broadcast: `!connected` plus C visibility/level/flags | existing `wiznetRecipientEligible`/`isPlaying` plus transport gate | Already fixed by merged PR #1932; left untouched. Existing `wizard-wiznet-depth` includes the menu-state recipient scenario. |
| `src/act.informative.c:2026-2112` users: includes descriptor rows by `connected`/`playing`/`deadweight` selection | `cmdUsers`: descriptor rows use `hasDescriptor`, state text uses `isConPlaying` | Fixed; exact fixture confirms menu, creation, REDIT, TEDIT rows, descriptor footer, and linkless omission. |
| `src/act.informative.c:2254-2279` where/no argument: only `!connected` | `cmdWhere` no-argument branch uses `isConPlaying` | Fixed; playing/string writers included, menu/creation/OLC/linkless omitted. |
| `src/new_cmds.c:2216-2225` tag: target linkless check is `!vict->desc`; “it” scan is `!connected && character` | `DoTag` uses descriptor presence for target gate and playing descriptor list for “it” | Fixed; editor target remains descriptor-valid, linkless target keeps exact C refusal. |
| `src/spec_procs.c:365-380` summoner memory scan: `!connected`, candidate before `number(0,3)` | `specSummoner` requires a playing descriptor before candidate/RNG selection | Fixed; next RNG draw test proves no linkless draw. |
| `src/new_cmds2.c:771-786` `hunt_items`: `!connected && character` | No Go counterpart found | Unported C effect; no behavior invented. |
| `src/new_cmds.c:1310-1329` `full_moon`: `!connected && character`, vampire/werewolf transform and direct messages | Go has a full-moon global weather message but no equivalent transformation | Unported C effect; this round only fixes the shared `send_to_all` audience. No transformation/RNG added. |
| `src/new_cmds.c:2661-2682` `lunar_hunter`: `!connected && character`, then per-player RNG, mob spawn, and tell | No Go counterpart to the C hunter loop; Go's lunar-hunter announcement is a separate Go-only event | Unported effect; left unchanged to preserve the C draw boundary (R3/R4). |
| `src/boards.c:417-421` board removal guard: `!connected && d->str == msg_storage[slot]` | `BoardSystem.RemoveMsg` checks `writingSlots` | Stop-tier: a disconnected writer can leave stale slot ownership; exact mapping requires writer cleanup/reconnect lifecycle work. No board lifecycle change. |

## OLC editor-state loops

| C site / exact gate | Go equivalent / gate | Verdict |
|---|---|---|
| `src/oedit.c:327-331`, `src/medit.c:322-325`: `connected == CON_SEDIT` while renumbering edited shop references | Go registry claims use stable VNUM references; no shop RNUM shift of an open descriptor reference | Match-by-design; no descriptor connection-state filter needed. |
| `src/olc.c:148-160`: reject matching `connected == olc_scmd_info[subcmd].con_type` and edited OLC number | `pkg/olc.Registry` holds typed editor claims on stable VNUMs | Match-by-design; duplicate editor state is claimed by the registry, not descriptor traversal. |

## State-machine loops reviewed but outside recipient scope

| C site / exact gate | Go equivalent / gate | Verdict |
|---|---|---|
| `src/comm.c:594-630` command queue loop: `!connected` restores a playing body from void; `connected != CON_PLAYING` routes to `nanny` instead of `command_interpreter` | Telnet `TerminalLine`/`handleCommand`: menu, creation, pager, string editor, each OLC editor, then ordinary command dispatch | Match-by-design; the generic state routing is already covered by entry/editor tests. |
| `src/comm.c:2170-2187` `check_idle_passwords`: only `CON_PASSWORD`, `CON_GET_NAME`, `CON_MENU` | `Manager.CheckIdlePasswords`: only unauthenticated sessions | Finite lifecycle gap for authenticated menu timeout; session timeout/registration is stop-tier. |
| `src/ban.c:264-269` `Valid_Name`: when same character is attached, report duplicate only if `STATE(d) == CON_PLAYING` | Entry-name reservation/retry and session registration | Lifecycle stop-tier; no duplicate-login/name reservation edits. |
| `src/interpreter.c:1541-1573` duplicate login: only a playing descriptor is selected as the usurped body; editor/menu descriptors follow cleanup paths | `Manager.register` / reconnect switch resolution | Lifecycle stop-tier; no reconnect or transport edits. |

Other candidate loops were read and have no connection-state gate: `src/act.informative.c:1750` do_who, `src/act.wizard.c:1746` descriptor-number disconnect, `src/act.other.c:147` duplicate-connection cleanup, `src/handler.c:1108` extraction descriptor-pointer transition, and `src/utils.c:586-596` `num_pc_in_room` (a room character list, not a descriptor loop). `send_to_room` is likewise descriptor-presence based, not connection-state based.

## R5e call path and lock check

Telnet `Session.TerminalLine` → `terminalCommand` → `sendClientMessage` → `handleMessage(MsgCommand)` → `handleCommand` → `executeCommandRaw` dispatches without `playerLifecycleMu`; `ExecuteCommand` also enters `executeCommandRaw` directly without that lock. The only `playerLifecycleMu` acquisition on the relevant manager side is registration/reconnect/disconnect lifecycle (`Manager.Register`, `HandleTransportDisconnect`, etc.). Manager callbacks snapshot sessions and release `Manager.mu` before calling `Session.isConPlaying`, which takes a short read lock to inspect state; command dispatch does not hold `Manager.mu` at those call sites. `occupiedZoneRooms` evaluates the same predicate with its locked helper while snapshotting switched/player bodies, avoiding races with menu and switch transitions without taking `playerLifecycleMu`. `World.playingPlayers` snapshots/releases `World.mu` before consulting the manager callback. OLC state is an atomic marker because `mudlog` can run while an editor mutex is held.

## R5h controls: reverted gate → red, restored gate → green

Controls were run as compiling source mutations and each mutation was restored before proceeding:

| Reverted gate | Red output | Restored proof |
|---|---|---|
| Remove `conEditorActive` from `Session.isConPlaying` (old `IsPlaying` semantics) | `go test ./pkg/session` failed on OLC/TEDIT recipients in gecho, send-all/outdoor/script echoes, gossip, ctell, page-all, mudlog provider, count/zone occupancy, clan who, force-all, transfer-all, users, where, and tag's “it” search. Representative assertions: `OLC editor descriptor received gecho`; connected 8 / playing 6; TEDIT moved during transfer. | Same proof group passes with the editor gate restored; `go test ./pkg/session ./pkg/game -run '^(TestConPlaying|TestScriptableOutdoor|TestClanTell|TestClanWho|TestQuestBroadcast|TestPageAll|TestForceAll|TestTransferAll|TestTagDescriptor|TestManagerEachSession|TestRecordUsage|TestMudLogCombines|TestUsersListsDescriptors|TestWhereNoArgument|TestZoneOccupancy|TestCounterProcsBlessing|TestCombatHealAllCallback|TestMobGlobalGossip|TestSummonerIgnores|TestHuntVictim)' -count=3` → both packages `ok`. |
| Make `World.playingPlayers` return every world player (old unfiltered world scans) | `go test ./pkg/game ./pkg/session` failed: counter reward healed menu/editor/linkless bodies and sent blessing; NPC global gossip reached those bodies; script/global and other fanouts included menu, creation, OLC, linkless; clan who/tag found non-playing bodies. | The same named game/session proofs pass with the callback filter restored. |
| Count any registered session as a descriptor (omit `hasTransport`) | `TestRecordUsageCountsDescriptorsAndConPlayingOnly`: connected 9 / playing 4 instead of 8 / 4. `TestUsersListsDescriptorsButOmitsLinklessCharacters`: listed Linkless and footer reported 9. | Both pass with `Session.hasDescriptor` checking transport. |
| Treat every world player as having a descriptor (revert tag target descriptor test) | `TestTagDescriptorSearchAndLinklessGateMatchC`: `You tap Linkless and scream, 'TAG! You're it!'` instead of `No way! They're linkless!` | Passes with `Manager.playerHasDescriptor` matching a registered live transport. |
| Remove send-to-zone awake/same-room filters | `TestConPlayingDescriptorFanoutMatchesCStates`: same-room actor and sleeping player each received the horn line. | Passes with C's awake/different-room checks restored. |
| Admit a linkless memory target to summoner before its draw | `TestSummonerIgnoresNonPlayingMemoryTargetWithoutRNGDraw`: `summoner selected a target without a CON_PLAYING descriptor`. | Passes with the pre-selection gate restored; next RNG draw remains unchanged. |
| Resolve a linkless PC hunting ID without a playing descriptor | `TestHuntVictimLeavesLinklessHuntingIDUnresolved`: hunting state became empty instead of retaining C's unresolved ID. | Passes with the descriptor resolution gate restored. |

The editor marker transition control also passes: `TestConPlayingEditorMarkersFollowEditorTransitions` enters real REDIT/TEDIT, confirms both are non-playing, then checks clean/abort exit restores playing eligibility.

## Depth proof rows

Passing unit rows were added to the matching command manifests: `comm.tsv`, `channels.tsv`, `qsay.tsv`, `ctell.tsv`, `page.tsv`, `gecho.tsv`, `clan.tsv`, `force.tsv`, `wizard.tsv`, `informative.tsv`, `mudlog.tsv`, `zone-reset.tsv`, `combat-entry.tsv`, and `spec-procs.tsv`; `descriptor-state.tsv` holds internal `record_usage`/provider/callback/editor-transition and concurrent occupancy proofs. No row is marked green for a stop-tier or unported site.
