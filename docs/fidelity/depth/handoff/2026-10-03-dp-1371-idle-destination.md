# DP-1371 D5 idle destination

Local tier: only game room translation changes; the session lifecycle implementation and transports remain unchanged. The existing session tests receive a corrected room-table fixture.

## Resolved case

`idle.force-rent-lost-link` is oracle-green at seed 1 through `lifecycle-idle-force-rent-audience`. C `src/limits.c:438-443` passes RNUM 3 to char_to_room, then `src/comm.c:2131-2133` sends the lost-link act in that room. The shipped table in `lib/world/wld/0.wld` is VNUM 0, 1, 3, 4: world[3] is VNUM 4. Go now translates through the existing RoomVNumByIndex instead of passing VNUM 3 to PlayerTransfer. The binary-search world table is C's VNUM-ordered index (`src/db.c:3083-3097`). No hardcoded VNUM 4 is introduced. R1/R5e/R5g/R5h.

Independent peers in VNUM 4 and VNUM 3 reproduce both opposite audience errors on main and match after the fix. An inside-void fixture isolates the separately blocked terminal-weather output. Existing scenarios and seeds are unchanged. The unit proves the shipped prefix, shuffled parsed input, and a world whose index 3 instead holds VNUM 40; reverting to VNUM 3 fails on a destination assertion and restoration passes.

## Other readers and locking

The destination is observed by Act, rent/save, deferred extraction, the descriptor's lost-link retirement, room occupants, and admin/transport room views. Those consumers receive the translated destination through the existing PlayerTransfer path; their implementations are unchanged. Rent still occurs before extraction and does not drop inventory in the disconnect room. The existing session close/menu and rent-floor assertions use the actual destination in their corrected fixture.

No new lock or lock nesting: RoomVNumByIndex takes/releases world RLock, then PlayerTransfer runs after release using its existing world/player locking. There is no lifecycle lock in CheckIdling. The later session extraction retains its existing lifecycle lock.

Sibling numeric-room audit: idle's initial world[1] matches shipped VNUM 1; duplicate cleanup's C world[1] likewise matches Limbo; C death_cry's missing-room world[0] matches shipped VNUM 0. Existing teleport and guardian random-index readers already have explicit index translators. Their policies are outside this fix; no broader claim is made.

## Remaining D5 case

`idle.force-rent-terminal-weather-byte` stays blocked. Weather is delivered to independently running socket writers before idle extraction can retire the session; draining a channel afterward cannot retract bytes already written. Live point_update additionally uses an independent World ticker, while DP_CLOCK uses the heartbeat callback. A fresh seed-1 main-source run reproduces exactly C `<CLOSE>` versus Go sunset line plus `<CLOSE>`; retained `weather-main.log` contains both C blocks. The next work is a design-first queued-output repair with one faithful ordering in live and pumped modes; do not patch only the test seam or suppress weather by predicting the next idle threshold.

Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-idle-proofs/`. Normal gates and the one final-tip combined census are recorded in the PR.
