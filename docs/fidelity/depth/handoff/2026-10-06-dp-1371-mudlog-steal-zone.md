# DP-1371 D7: steal producers and zone-editor defensive tails

**Stop tier.** Adds a steal-specific result marker and post-message producer boundary, plus C's missing zone-editor discard/cleanup defaults. These exceed the local mudlog-only exception. Fresh main `704463ae6` (merge #1821); five cases, one commit per case, one combined census at the tip.

## C-first contracts and actual paths

| Case | C site | Producer / boundary | Proof |
|---|---|---|---|
| carried steal | src/act.other.c:471-478 | PC victim only; transfer, CMP/IMMORT/file log, robbed affect, acknowledgement | TestStealZoneMudlogInventory; independent observer vehicle |
| equipment steal | src/act.other.c:429-439 | PC victim only; unequip/transfer, same log, robbed affect, acknowledgement | TestStealZoneMudlogEquipment; independent observer vehicle |
| caught steal | src/act.other.c:452-456,531-539 | All caught audiences, OUTLAW, CMP/IMMORT/file log, WAIT_STATE | TestStealZoneMudlogFailure; independent room observer |
| ARG3 default | src/zedit.c:1208-1214 | cleanup_olc(CLEANUP_ALL), BRF/BUILDER/file diagnostic | TestStealZoneMudlogZoneArg3; defensive state injection |
| parser default | src/zedit.c:1269-1273 | same cleanup then exact BRF/BUILDER/file diagnostic | TestStealZoneMudlogZoneDefault; defensive state injection |

`DoSteal` has one non-test caller, `command.CmdSteal`, the ordinary subcmd==0 command. Success producers run in the actual inventory/equipment transfer helpers before applyRobbedAffect. C spells `unsuccessfuly` with one l; the failure payload retains it. `applyStealFailure` keeps its existing OUTLAW assignment and marks only its PC arm. `sendSkillResult` emits that case's log after all literal audiences and before WAIT_STATE. Other skill results leave the marker false; no generic deferred logging API, new registry, command routing change or RNG draw is introduced. The ordinary PC-steal admission already requires OUTLAW; the existing assignment is idempotent.

`handleZeditInput` dispatches both defensive states. Their defaults now call existing finishZeditLocked(false): flush pending output, clear writing, emit the existing stop-OLC act, discard the editor and release its reservation. Only then log. No save or new cleanup implementation. Valid modes and numeric-refusal branches retain their existing behavior. Unit injections prove defensive contracts, not valid-play reachability or telnet oracle receipt for corrupt modes.

## Locks acquired and held at each producer

| Producer | Acquisitions before log | Held at log / delivery audit |
|---|---|---|
| steal inventory | room read, actor/target/object getters, inventory removal/addition, player crash setters | none; helper locks returned; names/description read locks finish before MudLog |
| steal equipment | body/equipment/inventory helpers and player crash setter | none; player equipment staging transfer completes before log |
| steal failure | body/room/message getters, world audience snapshot, ordinary message sink; prior OUTLAW setter in game helper | none; sendSkillResult retains no body/world/manager/lifecycle lock at its post-message boundary |
| zone ARG3 / parser defaults | textEditMu; writing/player helper; stop Act world/body helpers; OLC registry Release | textEditMu only; every cleanup helper releases its own locks before MudLog |

MudLog file write precedes manager EachSession's snapshot. Manager locks are released before recipient traversal/output. Player flags/level/color getters release player locks; Player.SendMessage releases player/world read locks before the sink; attached-body and snoop/heartbeat/send helpers do not acquire textEditMu. Thus delivery has no reverse acquisition of the retained editor lock (same audited boundary as #1815/#1820). No new lifecycle lock. Reproduce the path/other-reader sweep:

```sh
rg -n 'DoSteal\(|StealCaughtPlayer|applyStealFailure|applyRobbedAffect' pkg --glob '!**/*test.go'
rg -n 'handleZeditInput|finishZeditLocked|parseZedit.*Locked|releaseZoneEdit|textEditMu' pkg/session/zedit.go pkg/session/olc_registry.go
rg -n 'MudLog|EachSession|SendMessage|MessageSink|textEditMu' pkg/game/logging.go pkg/game/player_affects.go pkg/session/manager.go pkg/session/session_send.go pkg/session/switch_ownership.go pkg/session/heartbeat_output.go
```

## Fail-capable proofs and evidence

Tests invoke the real command/editor and manager provider. They assert exact observer/file bytes, minimum31 vs30, CMP vsNRM or BRF vsOFF, no invis threshold, transfer before success log before robbed affect/ack, caught actor/victim/room messages before failure log before wait, and cleanup/reservation release/discard before zone log. NPC steal checks use the direct game boundary to complement the live PC command; early missing-target refuses without logging. No assertion of full NPC-command dispatch parity.

Ten compiling controls cover five producer/cleanup reverts, two success-NPC classifiers, premature failure logging and the two cleanup removals; each requires its named assertion failure. Reproduce:

```sh
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-steal-zone-controls.py --output /tmp/mudlog-steal-zone-controls
go test ./pkg/session -run '^TestStealZoneMudlog' -count=1
go test -race ./pkg/session -run '^TestStealZoneMudlog' -count=1
```

Retained evidence: `~/Archives/darkpawns/oracle-runs/2026-10-06/dp-1371-mudlog-steal-zone-proofs/`. C captures at an uncommitted tree are preliminary; committed-tip combined census is authoritative. The class sweep also retains ZEDIT ARG1/ARG2 defaults (src/zedit.c:1070-1071,1135-1136) in the separately inventoried olc-zone-seams group; this train does not claim all ZEDIT diagnostics. The aggregate D7 row stays blocked. No whole-handler claim, oracle edit, repin, dropped seed or added expected divergence.

Oracle seed1 comparisons reach both theft successes and all failure audiences. Initial captures were false greens on the peaceful-room refusal; a later draft queued equipment theft behind WAIT_STATE. The final paired fixture clears ROOM_PEACEFUL and explicitly pumps 20 clock pulses between thefts. Both discarded drafts remain in evidence; no old vehicle, pin or seed was removed. Existing steal-depth uses the same peaceful room and requires a separate proof-frontier audit, rather than claiming that earlier vehicle proves a transfer.

Final integration gates pass: fmt, build, vet, all tests, game tests, lint (0 issues), selected session/game race tests, fidelity-depth, fidelity-units, string-census and diff check. Each case also passed its required gates before committing. All ten compiling revert controls passed. Combined census runs at the committed integration tip; its full and claims verdicts remain the merge gate.
