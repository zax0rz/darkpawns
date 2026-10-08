# Arena broadcast prerequisite to DP-1402

Stop tier: this repairs the shared Act output path for room messages and affects recipients across sessions. Zach explicitly approved this prerequisite on 2026-10-08. Claude reviews and gates; Zach merges.

Base: `origin/main` at `ec44271c6` (includes D7 PR #1858). No D7 files or producers change. Authority: `src/comm.c:2529-2535,2539-2556`, read in this checkout. R1/R3/R5c/R5e/R5g/R5h.

## Behavior

When the actor's room has ROOM_ARENA, TO_ROOM/TO_NOTVICT invokes the shared arena hook before its ordinary room audience. The hook wraps the unexpanded act format in `&RBroadcast: %s&n` and runs the existing per-recipient substitutions and visibility rules. It walks the registered players and mobs, excludes the actor, skips PRF_NOBROAD, permits sleepers, rejects writing recipients, and honors hide_invisible. The victim is included in the global broadcast even when TO_NOTVICT excludes them from the normal room act. The normal room act still follows, so an eligible local observer can receive both lines.

Color codes remain raw player-facing text here. `&R...&n` only renders after DP-1403 adds recipient-specific color expansion at the terminal funnel; color-off players retain the codes as C does.

**C's `static char buf[128]` plus unbounded `sprintf` (`src/comm.c:2543-2545`) can overflow; that is undefined behaviour (R1a).** The approved Go port emits the full text, with no 128-byte truncation or simulated overflow. The unit test uses more than 128 bytes; oracle probes use short formats to avoid that C UB.

## No-broadcast representation and other readers

The registered `nobroadcast` command goes through `wrapToggle` -> `ExecGenTog` -> `doGenTog`, whose toggle map assigns `PrfNoBroad` and mutates `Player.Flags` via `SetPlrFlag`. This matches `src/act.other.c:1266-1267`. The arena hook reads that flag, not the legacy `Player.NoBroadcast` bool. The proof exercises the real toggle twice and intentionally makes the bool disagree, so it cannot accidentally certify the wrong representation.

`Player.GetNoBroadcast`, the separate wizard broadcast handler and the saved `NoBroadcast` bool remain independent existing readers. The `Flags` field is already saved/restored and shown by preference/stat reports. No persistence format or other broadcast command is changed by this train.

Act and `channelAct` share `actDeliver`, so all callers get the hook once. `channelAct` retains its existing delivery callback and out-of-band mirror for each delivered line. The direct helper does not bypass World.MessageSink, telnet/WebSocket text events, GMCP or agent-state infrastructure. Mobile output retains existing MobInstance.SendMessage semantics; no synthetic descriptor is introduced. No RNG is consumed.

Class audit: the hook sits after TO_CHAR/TO_VICT early returns and room resolution, before the local room loop. It is not invoked for private acts or actor-less object acts. `rg` found the custom `actDeliver` readers in channel delivery, skull-room acts and mobile equipment display: the shared hook uses each supplied delivery callback, and the private mobile equipment TO_CHAR path remains outside its boundary.

## Locks

World.GetAllPlayers/GetAllMobs each snapshot their registry under World.mu.RLock and release it before delivery. Recipient flag and visibility reads take only the existing body/accessor locks. No world or body lock is held across performAct, the delivery callback, session output, or out-of-band observation. There are no new lifecycle/identity/session-manager lock acquisitions and no state mutations in the hook.

## Proofs and gates

- TestArenaActRemoteAndLocal: remote output, exact wrapper bytes, both room kinds, global victim inclusion, local double-delivery ordering, actor exclusion.
- TestArenaActNoBroadcastToggle: actual flag representation and no broadcast while enabled; toggled-off positive control despite legacy bool true.
- TestArenaActInvisibleActor: hidden remote output followed by detect-invisible positive control.
- TestArenaActNonArenaGate: ordinary local-only output followed by arena positive control.
- TestArenaActSleepWritingAndFullText: sleeper, writing filter, full text beyond 128 bytes, private-act exclusion.

The negative proofs include positive controls because removing the entire path would otherwise keep their negative assertion green. `2026-10-08-arena-broadcast-controls.py` uses a Go overlay to remove the hook while compiling, requires assertion failures in every named test, and checks restoration (0 -> 1 -> 0).

Oracle vehicle `act-arena-broadcast` uses shipped arena 8152, a remote mortal in 8004, the real nobroadcast toggle, and a sleeping recipient. The targeted census could not start because the shared slot is held by `dp-1371-mudlog-1b`; no second census was started. Full gates and combined census remain pending until that run releases the slot. Final evidence/verdicts will be appended before the PR.

DP-1402 remains in its separate worktree. After this prerequisite is clean, merged and available, rebase that branch onto it, re-run its real neutral-room spell probe and combined census. The approved INACTIVE regeneration gate remains a separate subsequent train.

Focused test and removal/restoration control completed successfully for all five symbols, with every removed-path failure occurring at a behavior assertion. Retained output: `~/Archives/darkpawns/oracle-runs/2026-10-08/arena-broadcast-unit-proofs/controls.txt`. `make fmt`, `git diff --check` and `make fidelity-depth` passed (5424 cases, 23 still blocked); `make hooks` installed the clone's hook configuration. A bounded census wait confirms `dp-1371-mudlog-1b` is still running; heavy gates and oracle execution are deferred to avoid disrupting the shared slot. No commit or PR yet.

## Heavy gates completed (2026-10-08)

After Zach confirmed the shared census slot was free, each required gate was run separately and its exit status checked: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, `golangci-lint cache clean`, `golangci-lint run ./...`, `git diff --check`, `make fidelity-depth`, `make fidelity-units` and `make string-census` all exited 0. Lint reported zero issues; fidelity-depth still reports 23 blocked cases. Fidelity unit evidence: `~/Archives/darkpawns/oracle-runs/2026-10-08/dp-1371-units-104312` (1405/1405 claims PASS). The string ratchet has zero unreviewed segments; its existing stale-generated-report warning was retained, with no baseline changes.

The next step is a combined census at the committed arena tip from this worktree. Its `go-head.txt` must equal that tip. The oracle scenario remains an unclaimed vehicle until its C transcript and census result have been checked; the manifest currently records the fail-capable unit proof. This supersedes the earlier pending-gates/slot note. DP-1402 remains queued on top of this prerequisite.

## Census finding and approved emote repair

The first committed tip (`99901e06b`) combined run was NOT_CLEAN: only the new `act-arena-broadcast` vehicle failed, missing the remote broadcast on both ordinary and sleeping probes. Claims were CLEAN_AFTER_RECHECK; `invis-depth` was infrastructure and its automatic isolated recheck passed. Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-08/arena-broadcast-combined`. The retained C transcript confirms the two broadcast lines and the empty NOBROAD block.

R5e live-path audit found `cmdEmote` sending a private JSON emote event through Manager.BroadcastToRoom instead of C do_echo's shared act (`src/act.wizard.c:144-151`). Zach explicitly approved routing repair after the report. The command now calls shared Act TO_ROOM before TO_CHAR, with C's NOREPEAT `Okay.\r\n` branch. Existing moderation/input gates remain. TestEmoteCommandArenaBroadcast executes command dispatch against registered sessions and a numeric ROOM_ARENA flag, drains the terminal sink, and checks remote output, local broadcast-before-normal order, real NOBROAD toggle, sleeper delivery and NOREPEAT. The control script independently removes the shared hook and restores the old command route; each compiles and fails on the missing remote bytes, then restoration passes. Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-08/arena-broadcast-unit-proofs/emote-controls.txt`.

Other readers: emote now uses World.MessageSink's existing text event envelope, terminal/WebSocket output, heartbeat staging and snoop delivery. No repository browser frontend reader specifically consumes the former emote event type. No persistence or GMCP state changes. `cmdEmote` holds no world, manager, lifecycle or body lock across Act; the existing registry snapshot and MessageSink locks apply. No new RNG draw (command dispatch retains its existing draw).

Further bypass audit found `cmdEcho` (the other C do_echo subcommand) still using Manager.BroadcastToRoom (`pkg/session/wiz_communication.go:60`). This pre-existing sibling needs a separate approved follow-up with C do_echo:144-151 as authority; its length/repeat/rendering differences are outside this approved emote repair. Other BroadcastToRoom callers span movement and wizard object paths, and are not claimed repaired here. No D7 producers change.

The repaired `act-arena-broadcast` targeted census is CLEAN (1/1 PASS), retained at `~/Archives/darkpawns/oracle-runs/2026-10-08/arena-emote-route-probe`. Its C transcript was read: ordinary and sleeping probes emit full broadcast wrappers; NOBROAD suppresses them. This is evidence of the intended branch, not only a no-diff verdict. No oracle-green claim is added here; the manifest names the independently fail-capable unit proof.

All required gates passed again after the approved routing repair: make fmt, go build ./..., go vet ./..., go test ./..., go test ./pkg/game/..., golangci-lint cache clean and run (zero issues), git diff --check, make fidelity-depth, make fidelity-units, and make string-census. The string ratchet remains zero unreviewed, with the existing generated-report warning and no baseline edits. Retained gate outputs are under `~/Archives/darkpawns/oracle-runs/2026-10-08/arena-broadcast-unit-proofs/emote-*.txt`. A fresh combined run at the repair commit is required before opening the stop-tier PR; the original NOT_CLEAN run remains retained.
