# Egon: asynchronous prompts and donation routing

Cite: `process_output`, `src/comm.c:1611-1644`; output dispatch and prompt sweep,
`src/comm.c:633-648`; `do_quit`, `src/act.other.c:133-137`; `do_drop`,
`src/act.item.c:529-575`; donation rooms, `src/config.c:159-160`.

## Confirmed prompt defect (R1, R5c)

`BroadcastToRoom` staged heartbeat output correctly, but its direct send bypassed
`notePlayerOutput`. A passive listener therefore received the leave line without
being eligible for the end-of-command/end-of-pulse prompt sweep. The sink now
uses `sendGuarded`, which records accepted terminal text; raw control, GMCP,
structured room state and entry frames do not request playing prompts.

Class inventory (repeat with `rg 'BroadcastToRoom\(' pkg/session --glob '!**/*test.go'`):

- `cmd_inventory.go`: quit's leave act; `manager.go`: cleanup leave fallback.
- `session_login.go`: login entry announcement.
- `movement_cmds.go`: shared position/room broadcast helper.
- `combat_cmds.go`, `act_offensive.go`: room combat/flee messages.
- `wiz_system.go`, `wiz_object.go`, `wiz_communication.go`: wizard broadcasts,
  object creation acts and room communication.
- `SetCombatBroadcastFunc` also used the guarded sink without output bookkeeping;
  it now benefits from the same correction.

Existing working routes: ordinary movement arrival/departure acts, gossip and
other channels use game act/Player.SendMessage -> World.MessageSink, which
already records player output. Weather uses Player.SendMessage as well. Tick,
zone and other heartbeat output is staged by BeginHeartbeatOutput and committed
by EndHeartbeatOutput, which already recognizes text and queues prompts.
TerminalLine and the main-loop OnFlushOutput invoke the shared prompt sweep.
The new direct-path bookkeeping fixes the remaining sink class without emitting
a prompt after each individual message.

## Donation verification (R3)

No routing change was needed. Go rolls randRange(0,3) before parsing, once per
command, exactly like C number(0,3): 0 junks; 1/2 select 8053; 3 selects 18204.
C initializes RDR to 0 (`src/act.item.c:537`), so junk is not the NOWHERE error.
Item actor/room text and destination act follow `perform_drop`
(`src/act.item.c:482-525`); gold follows `perform_drop_gold`
(`src/act.item.c:431-467`). The inaccurate Go comment saying the surviving
rooms were selected 50/50 was corrected.

The new fixture supplies twelve actual loaves, places passive observers in both
donation rooms, and donates each loaf separately. Inspecting C blocks confirmed
both destination messages and donations with neither destination receiving an
item. Draw streams match at seeds 1 and 2. This verifies existing behavior; it
is not counted as a behavior fix requiring a revert-red.

## Retained evidence

All directories below are under `/home/zach/Archives/darkpawns/oracle-runs/2026-10-29/`.

- `egon-prompt-passive-fixed`: prompt-preserving passive-observer probe CLEAN;
  the awake observer receives leave + prompt, the sleeper nothing.
- `egon-prompt-passive-reverted`: same scenario with production changes removed,
  FAIL solely on the passive observer's missing prompt; draw stream MATCH.
- `egon-prompt-targets`: all 32 selected quit, donation, movement/arrival and
  prompt scenarios CLEAN, zero draw divergences. The donation fixture in this
  frozen run was an early ineffective version; its proof is superseded below.
- `egon-donation-proven-seed1`, `egon-donation-proven-seed2`: corrected actual-item
  donation fixture, CLEAN, zero draw divergences, with both destination peers.

Earlier attempts are retained but are not proof: `egon-prompt-first` and
`egon-prompt-reverted` used force as the observer, whose own prompt masked the
bug; `egon-donation-outcomes-seed1` had an invalid exit fixture;
`egon-donation-all-seed1` used the wrong starting room and donated nothing.
The focused unit TestRoomBroadcastPromptsAcceptedText also fails when reverted
on heartbeat=false and passes with the fix; heartbeat=true remains correct.

## Decisions for Zach

Prompt timing is session stop tier: Zach merges after Claude's review. The
release batch still needs Claude's exact-commit combined release census. No
rulebook changes or donation behavior changes are proposed.
