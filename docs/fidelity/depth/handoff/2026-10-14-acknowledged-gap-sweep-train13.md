# Acknowledged Gap Sweep — Train 13 — 2026-10-14

## Train 13 closure

This train restores C's visibility gate for the AGGR24 + AGGRESSIVE full-health mobile scan. C checks `CAN_SEE(ch, tmp_ch)` before it selects a weaker NPC; Go previously selected the first weaker NPC regardless of visibility. The real-dispatch unit proof calls `World.MobileActivity()` with one NPC candidate, captures only the existing combat-engine handoff, and pins the four movement/sound draws. It leaves character-list ordering to DP-1364 and makes no attack, message, or combat-engine changes.

C call path rechecked in `src/`: `src/comm.c:817` calls `mobile_activity()`; `src/mobact.c:324-336` contains the branch, `CAN_SEE` candidate predicate, room grin act, and `hit()`; `src/utils.h:515-530` defines visibility. Go's heartbeat path is `cmd/server/main.go:623`, then `pkg/game/mobact.go:124`. The separate C room grin at `src/mobact.c:333` is omitted by Go and remains blocked under the shared attack/message owner boundary; this train does not change it.

The oracle fixture can spawn both stock NPCs, set the target's innate `AFF_INVISIBLE`, and enable the attacker's `MOB_AGGRESSIVE` flag. The scenario uses the `empty-players` fixture to crown its observer as God, toggles `NOHASSLE`, then moves the observer into room 8162. C's earlier ordinary aggressive and AGGR24 player scans skip that observer; the NPC scan must still skip the invisible weaker target. It pumps the actual heartbeat/mobile-activity path, so an ungated selection can enter combat and produce observable pulse output. Scenario: `mobact-aggr24-invisible-npc-target`. This preserves the shared `hit()` implementation and only uses it as the observable consequence of an incorrect target selection (R1/R3/R5e).

### R5h control

Removed the new `if !canSee(ch, vict) { continue }` gate and ran `go test ./pkg/game -run '^TestAggr24MobileTargetVisibilityGateUsesGlobalPulse$' -count=1`. It failed on the invisible candidate: `AGGR24 combat starts = 1, want 0; starts=[{attacker: defender:}]` (exit 1). Restored the gate; the focused visibility test plus existing AGGR24 skip/attack tests passed, as did `git diff --check`.

### Train 13 changed files

- `pkg/game/mobact.go`: apply the C candidate visibility check before the level comparison and combat-engine call.
- `pkg/game/mobact_test.go`: test invisible-candidate skip, visible weaker-candidate selection, actual `MobileActivity` dispatch, and pinned draw sequence.
- `docs/fidelity/depth/combat-entry.tsv`: mark the selection proof unit-green and track the missing pre-hit grin separately as blocked.
- `docs/fidelity/depth/mobile-activity.tsv`: add the invisible-NPC target proof row.
- `cmd/dp-oracle-diff/scenarios/mobact-aggr24-invisible-npc-target.txt`: set up a NOHASSLE God observer, the stock AGGR24 attacker, and the invisible weaker NPC.
- `docs/fidelity/depth/surface-inventory.tsv`: mark target visibility proven and add the blocked room-act surface.
- `docs/fidelity/depth/handoff/2026-10-14-acknowledged-gap-sweep-train13.md`: this report and carried-forward full inventory.

## Train 13 gates

All required gates passed, run separately. No race run was added because this change only adds a visibility predicate and changes no shared-state access.

- `make fmt` — PASS.
- `go build ./...` — PASS.
- `go vet ./...` — PASS.
- `go test ./...` — PASS.
- `go test ./pkg/game/...` — PASS.
- `go test ./cmd/dp-oracle-diff` — PASS after adding the targeted scenario.
- `golangci-lint run ./...` — PASS, 0 issues.
- `make deadcode-ratchet` — PASS, 52 baselined and 0 new.
- `make fidelity-depth` — PASS, 5,541 cases, 5,450 proven/delegated, 11 blocked, 68 excluded, 12 divergent-approved; 99.8% actionable completion.
- `make fidelity-units` — PASS, 1,433 rows, 1,324 symbols, 1,619 claims across 14 packages.
- `make string-census` — PASS, 98 known Go-only segments and 0 unreviewed; reports the pre-existing 24 baseline entries no longer produced and stale generated reports.
- `gosec -severity=high -quiet ./...` — PASS.

## Train 13 census

Targeted oracle census for `mobact-aggr24-invisible-npc-target` is pending release of the shared slot. After it passes, run the one full combined census for this train on the final tip and record both exact lines here.

## Current train 13 triage delta

| Item | Disposition | Evidence |
|---|---|---|
| AGGR24 aggressive-NPC target visibility | Fixed in Train 13; depth and surface rows point to the real-pulse unit proof, with a targeted oracle case pending. | `pkg/game/mobact.go`; `TestAggr24MobileTargetVisibilityGateUsesGlobalPulse`; `mobact-aggr24-invisible-npc-target`; R5h control above. |
| AGGR24 pre-hit room grin | Blocked and kept separate from the selection fix under the shared attack/message owner boundary. | C emits `act("$n grins evilly.", ..., TO_ROOM)` after selection at `src/mobact.c:333`; Go omits it before entering `StartCombat`. |
| Global `point_update` object timer pass | Fixed by Train 12 / PR #1964. | Train 12 implementation and retained target census in the prior handoff below; combined census completion is recorded there after it exits. |
| Player-equipment `APPLY_RACE_HATE`; Lua object-target spell/location gate; Kai-zai spell ID; mobile wounded/charmed, scavenger visibility, player-hunter gates; mobile `canget`; Lua follow | Fixed in trains 1–5. | Their exact C paths, proof tests, PRs, controls, and census lines are in the carried-forward train records; merged PRs #1953–#1955 cover the related script/spell and mobile actor fixes. |
| Detailed `show shops` | Fixed on main. | Commit `8aa563a332` (`fix: restore detailed show shops report and banking`); `cmd/dp-oracle-diff/scenarios/show-shops-detail.txt`. |
| Jail-guard visibility | Fixed and merged. | PR #1961 merged at `c50ab9da`; `pkg/combat/engine.go` visibility callback and train-10 evidence. |
| Water/flying mobile wandering | Fixed on main. | Commit `7a2f3b1049`; `pkg/game/ai.go` and `TestMobileActivityWaterAndFlyingSectorGates`. |
| `DoTurn` room-wide race behavior | Owned by open PR #1962; no overlapping edits in this sweep. | `src/new_cmds2.c:960-1030`; current PR state verified OPEN. |
| Corpse attack-type mapping | Fixed on main. | Commit `3694871a1`; `docs/fidelity/depth/corpse.tsv` row `death.corpse-attack-type-mapping` is unit-green. |
| Lua `iscorpse` predicate and leader-chain depth | Fixed and merged. | PR #1957 merged at `abe411ddd`; `docs/fidelity/depth/lua.tsv` rows `lua.bind-iscorpse-container-value` and `lua.bridge-leader-chain`. |
| Closed-door mobile wandering | Owned by open PR #1963; no overlapping movement-selection edits in this sweep. | PR #1963 current state verified OPEN. |
| Mobile memory warning and helper-assist arms | Inventory only; defer room acts and attacks to the shared combat/message owner. | Baseline A inventory cites `src/mobact.c:261-302`; these branches remain outside the selection-only fixes. |
| AGGR24 room grin act | Blocked under the shared attack/message owner; copy-ready Claude issue above. | `src/mobact.c:332-334`; no Go room act before `StartCombat`. |
| `DP-1426`, `DP-1419`, `DP-1423/1424/1425`, `DP-493`, `DP-1364`, generic NPC command interpreter, persistence/session/transport/save-format work | Owned or stop-tier; inventory only. | Exact owner assignments and evidence are preserved in the baseline triage and full A/B appendices below. |
| No-caller `get_obj`, ordinary-stock identify bitvector, legacy Lua fallback, and stale/accurate helper comments | Evidence-based exclusions or stale comments. | Baseline A inventory retains the call-site/world-data evidence and corresponding depth records. |
| Remaining assigned SPECIAL branches | No missing Go registration; existing branch statuses remain governed by the full 533-row special manifest. | All 89 assignment names, C definitions, and Go registrations are in Appendix B; retain its excluded/delegated/divergent/blocked statuses. |

## Copy-ready issue text for Claude: preserve AGGR24's room grin act

C's AGGR24 + AGGRESSIVE full-health mobile branch emits `act("$n grins evilly.", FALSE, ch, 0, 0, TO_ROOM)` after it selects the first visible weaker NPC and immediately before `hit()` (`src/mobact.c:324-336`). Go's branch in `pkg/game/mobact.go` selects a target and calls `StartCombat` without the room act. Please add the exact C room-audience message at the same point using the existing act/output path, with a fail-capable real-pulse proof and an oracle observer case. The `mobact-aggr24-invisible-npc-target` fixture shows how an empty-player God observer with `NOHASSLE` avoids the ordinary player-aggression branch. Preserve C candidate order, the `CAN_SEE` gate, existing RNG draws, and shared `hit()`/combat transcript ownership. Add an R5h revert control. Cite R1/R3/R5e and coordinate with the owner of `mob.teleport-victim-combat-transcript` before changing attack/message behavior.

---

## Carried-forward Train 12 handoff and full baseline inventories

The complete pre-edit A and B inventories, the full triage table, copy-ready issue texts, and earlier proof/census records are repeated below so this dated handoff is independently reviewable. The Train 12 census paragraph is a historical pending marker until the shared run completes; its exact result will be filled in here and in the Train 12 handoff before closure.

# Acknowledged Gap Sweep — Train 12 — 2026-10-14

## Train closure

This train fixes the silent global object timer pass in C `point_update()`. It is based on freshly fetched `origin/main` commit `259cca736`, after the mainline duplicate Lua depth-row cleanup. The implementation is commit `03d20b7d0` (`fix: restore point update object timers (R1 R3 R5e R5h)`). No governing document, C source, oracle source, save format, or reference binary was changed.

C call paths checked in `src/` during this train: `src/limits.c:525-686` (character pass then global `object_list` walk and timer branches); `src/db.c:1892-1893,1950-1951` (new objects prepend to `object_list`); `src/handler.c:907-948,995-1026,1046-1090` (room/container/carrier moves, extraction, and character-held object updates); `src/constants.c:1440-1448` (field-object table); `src/gate.c:343-380` (moon-gate countdown value); `src/spec_assign.c:521-551` (assigned moon gates); and `src/comm.c:2366-2384` (`send_to_room` audience and exact bytes). The port walks the registered objects newest-first after all character work, decays C's global timer branches at every location, spills corpse contents according to the C location branch, preserves the field-object `in_room > 0` guard and spawned-puddle timer, and stores SpellGate's moon-gate timer in `val[2]`.

The actual hourly oracle scenario is `point-update-global-object-timers`: it moves the witness into room 8162, sees the stock wall of ice, pulses 630 minutes to invoke the hourly update, and then sees the resulting puddle. A first diagnostic was a false green because the witness remained in room 8004; the fixture was corrected to `goto 8162` for both oracle and port before the visible scenario passed. Targeted retained census: `dp-1371-train12-global-object-timers @03d20b7d0: oracle-regression: scenarios=1 passed=1 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=0 elapsed=17.244s verdict=CLEAN`; the retained row is `PASS	point-update-global-object-timers	10` in `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train12-global-object-timers`. Full combined census: pending final handoff commit; it will be run on the final source tip for this train and its exact result appended here.

## R5h controls

Each mutation below was run against the named proof, failed with the stated assertion, then the original production file was restored byte-for-byte. The restored assertions passed.

| Reverted behavior | Expected red evidence |
|---|---|
| Remove the global object pass from `PointUpdate` | Carried puddle and carried/contained corpses survive; the RNG proof receives no corpse messages. Exit 1. |
| Empty `fieldObjs` | The field wall survives at C room index 1. Exit 1. |
| Disable the `moon_gate` branch | The assigned moon-gate survives after `val[2]` reaches zero. Exit 1. |
| Restore `SpellGate` to `SetTimer` | Red gate `val[2]` is 0, expected 2. Exit 1. |
| Move global object pass before NPC memory draws | Corpse decay messages differ from the pinned C global-list/RNG order. Exit 1. |

After every control, `limits_condition.go`, `limits.go`, or `gates.go` was byte-identical to its saved pre-control content. The combined restored assertion selection passed (`go test ./pkg/game -run '^(TestPointUpdateDecaysObjectsInCarriedAndContainedLocations|TestPointUpdateFieldObjectUsesCRealRoomIndex|TestPointUpdateExpiresAssignedMoonGateByValueThree|TestSpellGateStoresCountdownInObjectValueThree|TestPointUpdateRunsGlobalObjectPassAfterAllMemoryDraws)$' -count=1`).

## Gate record

All required gates passed on the final implementation source. `make string-census` passed its ratchet (98 known Go-only segments, 0 unreviewed); it also reported 24 baseline entries no longer produced and stale generated reports, a pre-existing maintenance notice. `make fidelity-depth` passed at 5,540 cases / 5,450 proven or delegated / 99.8% actionable completion. `make fidelity-units` passed with 1,433 rows, 1,328 symbols, 1,622 claims across 14 packages. The race run was included because the global registry pass reads shared world state.

- `make fmt` — PASS.
- `go build ./...` — PASS.
- `go vet ./...` — PASS.
- `go test ./...` — PASS.
- `go test ./pkg/game/...` — PASS.
- `go test -race ./pkg/game/...` — PASS.
- `golangci-lint run ./...` — PASS, 0 issues.
- `make deadcode-ratchet` — PASS, 52 baselined, 0 new.
- `make fidelity-depth` — PASS, 5,540 cases, 5,450 proven/delegated.
- `make fidelity-units` — PASS, 1,433 rows, 1,328 symbols, 1,622 claims.
- `make string-census` — PASS, 98 known Go-only segments, 0 unreviewed; stale generated-report notice as above.
- `gosec -severity=high -quiet ./...` — PASS.

## Current triage delta since the 2026-10-13 inventory snapshot

The appended A/B inventories are the requested pre-edit snapshot. These rows have moved since that snapshot and should be read with current mainline/open-PR state:

| Baseline item | Current disposition at train 12 | Evidence |
|---|---|---|
| `jail-guard redirect omits CAN_SEE` | Fixed in merged jail-visibility train; do not duplicate. | PR #1961 is in this branch's base ancestry; `pkg/combat/engine.go` visibility callback and train-10 handoff. |
| Water/flying mobile wandering gates | Already present on current main; no longer a gap. | Commit `7a2f3b1049`; `pkg/game/ai.go:120-131` and `TestMobileActivityWaterAndFlyingSectorGates` and related tests. |
| Detailed `show shops` | Already implemented and covered on current main. | Commit `8aa563a332`; `pkg/session/wiz_info.go:760-870`, `pkg/session/show_shops_depth_test.go`. |
| Corpse attack type mapping | Already matched; no mapping correction remains. | Commit `3694871a1`; `docs/fidelity/depth/corpse.tsv` has `death.corpse-attack-type-mapping` unit-green. |
| Lua `iscorpse` and deep leader chain | Owned by open PR #1957; leave to its author/review. | PR #1957 (`codex/luna-acknowledged-gap-sweep-6`). |
| Room-wide `DoTurn` | Owned by open PR #1962; leave to its author/review. | PR #1962 (`codex/luna-acknowledged-gap-sweep-11`). |
| Global `point_update` object timer pass | Fixed in this train; depth row and surface inventory now cite its unit and oracle proofs. | This train's implementation, controls, and retained targeted census above. |
| AGGR24 aggressive-NPC target visibility | Still blocked and remains a candidate for a separate pulse-selection train; C character-order dependency remains DP-1364. | `docs/fidelity/depth/surface-inventory.tsv`; `src/mobact.c:324-336`; `pkg/game/mobact.go:491-505`. |
| Closed-door mobile wandering | Open PR #1963; do not overlap its movement-selection change. | PR #1963 (`claude/dp-1428-wander-closed-doors`). |

## Full baseline handoff — inventories, triage, and issue text

The full 2026-10-13 baseline handoff follows verbatim. It contains the widened-search method, pulse and assigned-special inventory, recorded triage and exclusion evidence, copy-ready Claude issue texts, and earlier train evidence. The complete A comment/gap inventory and the complete B assigned-special crosswalk are also embedded in the appendices below rather than left as external-only references.

# Acknowledged Gap Sweep — 2026-10-13

## Scope and method

This inventory was captured before any production-code edit, on branch
`codex/luna-acknowledged-gap-sweep`, rebased onto `origin/main` at
`9b3ecaae3d02cb1f2b066e9dd27c81bb5ec350fc`. The broad comment search covered
non-test Go files under `pkg/` and `cmd/`, using the baseline expressions
`not (yet )?ported`, `unported`, `deliberately not`, `is not ported`,
`left unported`, `not implemented`, `no go counterpart/equivalent`, widened
with `stub`, `simplified`, `approximat*`, `placeholder`, `TODO`, `FIXME`,
`ignored`, and `skipped`. At the recorded `origin/main` revision, the baseline
expression set reproduced 21 hits. The widened set matched 175 lines total;
143 lines contained Go comment syntax, and 32 were other matches. This count
uses the original `grep -v _test` output filter. Most widened-only hits
describe ordinary parser, command, or error behavior; the admissions and the
additional gaps found by call-path review are inventoried in
`2026-10-13-acknowledged-gap-sweep-comments.tsv`.

For each behavior claim below, the named C site was checked in `src/` during
this session. The active C reference oracle and `src/` remain untouched. The
assigned-special crosswalk is the adjacent
`2026-10-13-acknowledged-gap-sweep-specials.tsv`; its branch-level evidence is
the existing `docs/fidelity/depth/spec-procs.tsv` (533 rows).

The comments TSV uses these triage letters: **a** tracked or owned, **b**
blocked depth row, **c** issue/stop-tier, **d** evidence-based exclusion or
stale comment, **e** fix candidate, and **f** fixed in this sweep. It records
each admitted source comment/gap; widened-only comments describing ordinary
behavior were reviewed but are not gaps.

## A. Acknowledged comments and triage

The TSV is the itemized inventory. Fixed in trains 1-5: player-equipment
`APPLY_RACE_HATE`, Lua object-target spell dispatch and location gate, Kai-zai
spell-ID gate, mobile activity/scavenger/player-hunter gates, live mobile-actor
`canget`, and Lua mobile follow. Current fix candidates: detailed `show shops`,
jail-guard visibility, water/flying mobile wandering, `DoTurn`'s room-wide
race behavior, corpse attack mapping, the Lua `iscorpse` binding, and the Lua
leader-table depth cap. Blocked depth rows
retain AGGR24 target visibility and global point-update object timers. Owned or
tracked: DP-1426, DP-1364, DP-493, generic NPC command dispatch, and
`mob.teleport-victim-combat-transcript`. Stop-tier: date-record persistence,
help usage logging, `do_write`'s editor/file path, tattoo persistence, and
house object persistence. Evidence-based exclusions include `get_obj`, which
has no callers, and comments on the legacy Lua table fallback, which is only
used by tests; production table construction already matches C's fields and
nil-vs-table behavior. `DoMindlink`, `DoDig`, display/door/math helpers, and old
stub headers are stale or accurate comments. `APPLY_RACE_HATE`'s proof and
census record are below.

Evidence notes:

- `src/act.other.c:557-591` gates `do_visible` with
  `affected_by_spell(ch, SKILL_KK_ZAI)`; Go's `pkg/game/other_character.go:19-33`
  searches an affect-source substring instead of spell ID 164.
- `src/spell_parser.c:291-320` emits the object-target phrase only when the
  object is in the caster's room or carried by the caster. Go's
  `pkg/spells/say_spell.go:235-236` currently has an unconditional-true stub;
  `src/scripts.c:1428-1489` also permits object targets in Lua `spell`, while
  `pkg/game/world_bridge.go:1109-1110` drops them.
- `src/fight.c:1374` checks `CAN_SEE(ch, victim)` before a jail-guard combat
  redirect. Go's `pkg/combat/engine.go:954` has no visibility callback. The
  TODO cites DP-1054, which is closed for a different jail-guard VNUM issue.
- `src/handler.c:205-238` implements `APPLY_RACE_HATE` slot toggle/insert/remove during equip/unequip. The stock world has 18 such affect records across 14 object prototypes (for example `lib/world/obj/162.obj:33,441` and `lib/world/obj/52.obj:15,17`). Go applies this affect only in `mobile_equipment.go`; player equipment paths do not update `Player.RaceHates`, despite pulse aggression reading those slots. The former exclusion claim in the initial search was false; this inventory corrects it.
- `src/new_cmds2.c:960-1030` `do_turn` scans all room occupants by `RACE_UNDEAD`/`RACE_VAMPIRE`, consumes mana and a random cost, and handles all eligible targets. Go `DoTurn` in `pkg/game/skills2.go:492-567` uses one command-resolved target and guesses undead status from its display name; this needs a dedicated issue/room-aware command change.
- `src/mobact.c:54-340` has no wounded/charmed early return after the two
  hunter calls; Go `pkg/game/mobact.go:176-179` adds one. The C scavenger gate
  uses `CAN_GET_OBJ` (`src/utils.h:535-549`), including `CAN_SEE_OBJ`; Go's
  candidate scan omits the room-light-aware visibility gate. C memory and
  helper branches also emit/perform arms not present in Go
  (`src/mobact.c:261-302`). The fully-healed AGGR24+AGGRESSIVE mob scan uses
  `CAN_SEE` at `src/mobact.c:324-336`; Go currently selects the first weaker
  mob without that gate. The C/Go global character-order difference remains
  tracked by DP-1364.
- `src/limits.c:525-686` traverses the global object list for timer effects;
  Go `pkg/game/limits_condition.go:305-363` drives `decayObjectsInRoom` over
  room-floor objects. The C pass also decays carried/contained objects, and
  corpse decay handles carried corpses specially. This silent point-update
  gap has no dedicated row yet.
- `src/new_cmds.c:1310-1329,2661-2682` transforms connected vampire/werewolf
  descriptors at full moon and makes per-descriptor RNG draws/spawns/tells
  for lunar hunters. Go `pkg/game/weather.go:603-630` broadcasts two global
  lines. Both are already recorded in the 2026-10-13 playing-descriptor audit
  and the blocked `lifecycle weather and time pulses` surface row; descriptor
  eligibility makes these stop-tier for this sweep.

## B. Silent pulse and assigned-special inventory

| Surface | C path checked | Go counterpart | Finding and triage |
|---|---|---|---|
| Heartbeat order | `src/comm.c:800-844` | game-loop heartbeat and `World` pulse methods | The mobile → room → object mobile-pulse order and character-order boundary are already represented by `lifecycle mobile room object pulse order` (blocked) and DP-1364. Hourly `weather_and_time`, `affect_update`, `point_update`, then `hunt_items` are sequenced in C; `hunt_items` is DP-493 (`pkg/session/wiz_system.go:740`). Autosave and usage/date writes have existing persistence rows. |
| `room_activity` | `src/comm.c:691-756` | `pkg/game/room_activity.go` | `flow_room`, `loud_mobs`, and CON<=0 croak are DP-1426 and owned by GLM. Character-list ordering is DP-1364. Other source arms are mapped in the function comment and pulse tests. |
| `object_activity` | `src/comm.c:758-797` | `pkg/game/room_obj_scripts.go:206-250` | The global object walk, drink/fountain/corpse skips, repair-weight branch, and object pulse script call are present. Object ordering is DP-1364. No other unowned branch omission was found in this pass. |
| `point_update` characters | `src/limits.c:460-525` | `pkg/game/limits_condition.go:87-303` | Conditions, tattoo/jail timers, dreams, regen, status damage, memory clear, character-object update and idle checks have existing surface/depth rows. `hunt_items` is DP-493. The global character-order difference is DP-1364. |
| `point_update` objects | `src/limits.c:525-686`; carried timers in `src/handler.c:1046-1090` | `pkg/game/limits_condition.go:305-363` | Go only runs timer effects over room-floor objects; C walks every object and expires carried/contained corpses, puddles, dust, CoC, gates, and field objects as applicable. Add a dedicated depth row before fixing; candidate for the point-update train. Preserve the C distinction between global timer branches and field-object `in_room > 0`. |
| `mobile_activity` | `src/mobact.c:54-340` | `pkg/game/mobact.go:154-507` | All 89 assigned special names have both C definitions and Go `RegisterSpec` entries; the exhaustive crosswalk is adjacent. The 533 branch rows retain their recorded statuses in `spec-procs.tsv`: 243 unit-green, 145 oracle-green, 105 oracle-green-multiseed, 26 excluded, 11 delegated, 2 divergent-approved, 1 blocked. New findings from the branch comparison are listed above: invented charmed/wounded gate, scavenger object visibility, missing memory/helper arms, and missing visibility on the AGGR24+AGGRESSIVE NPC candidate selection. Attack/message/RNG overlap owned by `mob.teleport-victim-combat-transcript` stays with that owner. |
| `weather.c` and hourly event bodies | `src/weather.c:52-123`; `src/new_cmds.c:1310-1329,2661-2682` | `pkg/game/weather.go:344-430,603-630` | Core hour/day/moon/weather progression is mapped. Date-record persistence is stop-tier and documented at `weather.go:185-190`. Full-moon transformation and lunar-hunter spawn/tell/RNG are already in the blocked lifecycle weather row and playing-descriptor audit; do not change in this sweep. |
| Assigned `SPECIAL`s | assignments in `src/spec_assign.c`; bodies in `src/spec_procs*.c` and other named C files | `pkg/game/spec_assign.go` and `RegisterSpec` calls | `2026-10-13-acknowledged-gap-sweep-specials.tsv` lists all 89 distinct assigned names with exact assignment, C definition and Go registration lines. None is missing a Go registration. Branch completeness remains governed by the 533-row ledger; retain its exclusions, delegations, approved divergences and the owned blocked combat transcript row rather than claiming registration proves behavior. |

## Train decisions

1. First train: finish player-equipment `APPLY_RACE_HATE`. The stock-data
   scenario uses `wield` and the actual mobile pulse; pair it with the real Go
   command-dispatch test, R5h revert control, targeted census, and required
   gates.
2. Next, fix Lua object-target spell dispatch and the exact object-location
   speech gate as one script/spell family. Use `lua-bind-actions` if its fixture
   can create and target the object, plus real Lua dispatch and R5h proofs.
   Confirm `src/scripts.c`, `src/spell_parser.c`, and `src/magic.c`; preserve
   object-target RNG order.
3. A pulse train may fix independent mobile-selection gates and point-update
   object traversal only after fail-capable tests pin the real dispatcher,
   ownership cases, and C list/RNG order. Do not alter attacks/messages owned
   under `mob.teleport-victim-combat-transcript`.
4. Treat Kai-zai and identify's object flags as independent small candidates;
   keep them in separate trains if they cross the family boundary. Defer jail
   guard visibility and mob memory/helper attack arms to their combat owner.
5. Do not touch DP-1426, DP-1419, DP-493, DP-1364, the generic NPC command
   interpreter, descriptor/session lifecycle, persistence, or save formats.
   DP-1423/1424/1425 and entry/nanny, login, transport, and session lifecycle
   remain stop-tier per the goal brief; report those gaps only.

The inventories above were recorded before the first production edit. The
first train implements player equipment `APPLY_RACE_HATE` slot mutation and
removes its obsolete exclusion comment. The actual `World.DoWield` and
`World.DoRemove` paths are covered by
`TestRaceHateEquipmentRebuildAndRemoval`, which then invokes the real mobile
activity dispatcher. `wear-race-hate-pulse` is the stock-data oracle vehicle.
The targeted census is CLEAN; the R5h control is RED with the rebuild removed
and GREEN after restoration. All per-train gates and the combined census are
recorded below; the combined run completed cleanly on code tip `e996c7e46`.
Slot rebuilding adds no RNG calls; the stock pulse vehicle exercises the
existing C race-hate speech/attack draws in order (R3).

## Copy-ready issue text for Claude

### DoTurn scans the whole room by race and preserves C mana/RNG order

`DoTurn` currently receives one command-resolved target and infers undead
status from its name (`pkg/game/skills2.go:492-567`). C `do_turn` in
`src/new_cmds2.c:960-1030` scans every room occupant for undead/vampire races,
charges mana, consumes a random mana cost, and applies outcomes to each eligible
target. For level differences -4 through 3, C applies no target effect after
the attempt message and mana charge; Go currently adds damage and messages.
Please add room-aware command dispatch that preserves C room-list order, race
checks, mana/RNG timing, and actor/target/room messages. Add fail-capable
real-command and pulse/oracle proofs; do not change player save fields or
formats. Cite R1/R3/R5e.

### Implement the detailed `show shops` report

The wizard command currently omits C's detailed report
(`pkg/session/wiz_info.go:237`; `src/shop.c:1413` and
`src/act.wizard.c:2414`). Please port the exact C-selected shop list, field
formatting, and ordering, with a transcript vehicle for empty and populated
shops. Keep it read-only and preserve the current command gates. Cite R1/R2/R5e.

### Match the `iscorpse` Lua binding

The production Lua table bridge follows C's `char_to_table` path.
`pkg/scripting/engine.go:1392-1397` treats type 18 as a corpse instead of
checking C's container type plus value[3]. Please match the C predicate with
real Lua dispatch proof and a revert control, retaining the already-matching
invalid-argument and nil return arms. C paths: `src/scripts.c:636-654`,
`src/utils.h:490-491`, and `src/structs.h:434`. `canget`'s acting-mobile
predicate is fixed in train 4 / PR #1955. Cite R1/R4/R5e.

### Preserve the full Lua leader chain

`pkg/scripting/bridge.go:293-326` stops nested `leader` tables at depth 32.
C `char_to_table` recursively includes each `ch->master` until the pointer is
nil or self (`src/scripts.c:1870-1873`); `circle_follow` rejects cycles
(`src/utils.c:365-375`). Please preserve every valid chain, with a real Lua
dispatch proof over 32 links and an R5h revert control. Add an oracle vehicle
if the current fixture can establish the chain. Cite R1/R4/R5e.

### Add the missing water-capability checks to mobile wandering

`pkg/game/ai.go:79` currently treats mobs as unable to swim/fly in water and
flying sectors. C `mobile_activity` checks `CAN_SWIM` and `IS_FLYING` at
`src/mobact.c:130-138`, using the shared capability predicates in `src/utils.h`.
Please identify the exact Go sources for those C predicates and port their
existing flag/equipment/affect inputs; do not invent new world-data flags. Add
real pulse proofs for swimmers, flyers, and ordinary mobs plus RNG parity. Cite
R1/R3/R5e.

### Separate jail-guard visibility from the shared combat transcript work

The jail-guard redirect in `pkg/combat/engine.go:954` lacks C's
`CAN_SEE(ch, victim)` gate (`src/fight.c:1374`). Its current TODO cites DP-1054,
which tracks a different jail-guard VNUM issue. Please add a world-aware
visibility input and a focused guard redirect proof after confirming this
change does not alter the combat transcript/RNG cases owned under
`mob.teleport-victim-combat-transcript`. Cite R1/R5e and retain that owner
boundary.

## Train 1 proof record

| Evidence | Result |
|---|---|
| Targeted `go test ./pkg/game -run '^TestRaceHate' -count=1` | PASS, restored fix: `ok github.com/zax0rz/darkpawns/pkg/game 0.004s` |
| `wear-race-hate-pulse` targeted census | CLEAN: 1 passed, 0 expected/failed/stale/infra/timeout/unstable; elapsed 19.344s (`2026-10-13T20:31:02-0400` to `20:31:22-0400`) |
| R5h revert control | RED as required after removing the `Player.RaceHates` rebuild: `RaceHateEquipmentRebuildAndRemoval` got `[-1 -1 -1 -1 -1]`, wanted `[18 20 -1 -1 -1]`; restored patch and rerun PASS |
| `make fmt` | PASS |
| `go build ./...`; `go vet ./...` | PASS separately |
| `go test ./...`; `go test ./pkg/game/...`; `go test -race ./pkg/game/...` | PASS separately |
| `golangci-lint run ./...` | PASS: 0 issues |
| `make deadcode-ratchet` | PASS: 52 baselined, 0 new |
| `make fidelity-depth` | PASS: 5,515 cases; 5,424 proven/delegated; report 99.8% actionable completion |
| `make fidelity-units` | PASS: 1,420 rows, 1,306 symbols, 1,601 claims |
| `make string-census` | PASS, ratchet 98 known go-only segments; generated reports were already stale, and the command warned to run `make string-census-update` |
| `gosec -severity=high -quiet ./...` | PASS |
| Full combined census (`dp-1371-train-1`, Go code tip `e996c7e46`) | `oracle-combined: pairs=3396 deduplicated=1016 full=CLEAN claims=CLEAN_AFTER_RECHECK elapsed=1994.722s verdict=CLEAN_AFTER_RECHECK`; `wear-race-hate-pulse` PASS |

## Train 10 — jail-guard damage visibility

Started from fresh `origin/main` at `d8e8024ddbfe52fbde36d16e0b02e2d7606fac71`
in the isolated `codex/luna-acknowledged-gap-sweep-10` worktree. The inventory
identified `src/fight.c:1374`'s `CAN_SEE(ch, victim)` condition missing from
both Go damage paths: the combat engine intercept in `pkg/combat/engine.go`
and legacy `takeDamageFrom` in `pkg/combat/fight_core.go`. The stale DP-1054
TODO was removed. A world-wired callback now mirrors the source predicate,
including its ordering: `GET_REAL_LEVEL` precedes HOLYLIGHT, then blindness,
room light/infravision, and invisibility/detect-invisibility; AFF_HIDE remains
outside this `CAN_SEE` predicate. This is separate from and does not change the
combat transcript/RNG owner boundary `mob.teleport-victim-combat-transcript`.

The fail-capable test control removed both visibility checks: the two focused
damage tests failed, with one taking the intercept and the other suppressing
ordinary damage. Restoring the checks made the focused tests pass. The oracle
vehicle `spec-proc-take-to-jail-visibility` uses assigned mob 8001, whose
prototype's affect value `1032` carries DETECT_INVIS bit 3
(`lib/world/mob/80.mob:1-8`; `src/structs.h:313`), and sets a level-1 victim's
invisibility level to 25 while the guard's level is 22. C's earlier level gate
therefore rejects visibility even though DETECT_INVIS is present. The victim
starts combat; then the `comm.c:822-823` -> `perform_violence()` -> `hit()` path
deals ordinary damage and leaves the player in the room. The registered
special is redispatched from the `perform_violence()` tail at
`fight.c:2030-2031`; `take_to_jail` reaches its fighting arm at
`spec_procs2.c:1430-1434`. `mobile_activity()` skips fighting mobs
(`src/mobact.c:68`), so the exercised path is combat update. Direct seed-1
differential and targeted census `dp-1371-train10-visibility-targeted` are
CLEAN. The targeted census ran on committed tip `3f90033f9`: 1 passed, 0
expected/failed/stale/infra/timeout/unstable; evidence is retained at
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train10-visibility-targeted/`.
The final combined census ran on committed tip `93e2db3cf` and completed
`CLEAN_AFTER_RECHECK`: 3,402 pairs, 1,018 deduplicated; full `CLEAN`, claims
`CLEAN_AFTER_RECHECK`, elapsed 1,978.534s. The scenario row is PASS; retained
evidence is at
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train10-final/`.

Remaining acknowledged candidates after Train 10 are room-wide undead `DoTurn`
(`src/new_cmds2.c:960-1030`), the live Lua character/mobile table parity gap
(`src/scripts.c:1812-1916`), and point-update object timers, which remain blocked
pending depth proof. Mobile memory/helper attack arms and AGGR24+AGGRESSIVE
visibility remain owned or blocked as recorded in the inventory; the shared
combat transcript owner boundary stays intact.

### Train 10 proof record

| Evidence | Result |
|---|---|
| Direct `spec-proc-take-to-jail-visibility`, seed 1 | CLEAN; ordinary damage output and post-combat room look match; the victim is not jailed |
| R5h revert control for both Go jail-guard checks | RED as required: `TestMobRedirect_JailGuardDoesNotSubdueInvisibleVictim` and `TestTakeDamage_JailGuardDoesNotSubdueInvisibleVictim` fail when their visibility checks are removed |
| Focused combat and world visibility tests | PASS |
| `make fmt`; `make check-fmt`; `git diff --check` | PASS |
| `go build ./...`; `go vet ./...` | PASS separately |
| `go test ./...`; `go test ./pkg/game/...`; `go test -race ./pkg/combat ./pkg/game` | PASS separately |
| `golangci-lint run ./...` | PASS: 0 issues |
| `make deadcode-ratchet` | PASS: 52 baselined, 0 new |
| `make fidelity-depth` | PASS: 5,522 cases; 5,432 proven/delegated; 99.8% actionable completion |
| `make fidelity-units` | PASS: 1,425 rows, 1,313 symbols, 1,608 claims |
| `make string-census` | PASS: ratchet 98 known go-only segments, 0 unreviewed; reports are stale and were not updated |
| `gosec -severity=high -quiet ./...` | PASS |
| Targeted census `dp-1371-train10-visibility-targeted` | CLEAN: 1 passed, 0 expected/failed/stale/infra/timeout/unstable |
| Final combined census `dp-1371-train10-final`, Go tip `93e2db3cf` | `oracle-combined: pairs=3402 deduplicated=1018 full=CLEAN claims=CLEAN_AFTER_RECHECK elapsed=1978.534s verdict=CLEAN_AFTER_RECHECK`; `spec-proc-take-to-jail-visibility` PASS; evidence at `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train10-final/` |

## Appendix A — Full acknowledged Go comment/gap inventory

This is the complete row-level inventory from the baseline TSV, including its triage column.

```tsv
source	admitting_comment_or_gap	triage	evidence_or_owner
pkg/game/room_activity.go:21	flow_room, loud_mobs, CON<=0 croak deliberately omitted	a	DP-1426; GLM owns all three
pkg/game/room_activity.go:54	character order differs from C	a	DP-1364; lifecycle mobile/room/object pulse order
pkg/game/npc_command.go:78,112,116,212	mobile command interpreter, mounted dismount and NPC give forms absent	a	lua.npc-command-unported / force.npc-command-interpreter; generic NPC command design is owned by Zach
pkg/game/actor_specials.go:7,54	legacy NPC-only special actor route is not implemented	a	generic NPC command interpreter / actor dispatch design is owned
pkg/game/world_bridge.go:1071	Lua follow of mobile leader ignored	f	src/scripts.c:542-569; fixed in train 5 with local get_char_room_vis resolution and real Lua/oracle proofs
pkg/game/world_bridge.go:1082	Lua set_hunt for player hunter ignored	f	src/scripts.c:1341-1363; fixed in train 4; PR #1955
pkg/game/world_bridge.go:1109	Lua object-target spell ignored	f	src/scripts.c:1428-1489; fixed in train 2; PR #1953
pkg/spells/say_spell.go:235-236	object target location predicate is unconditional true	f	src/spell_parser.c:291-320; fixed in train 2; PR #1953
pkg/game/other_character.go:19	Kai-zai gate searches affect source text instead of spell number	f	src/act.other.c:557-591; SKILL_KK_ZAI=164 at src/spells.h:212; fixed in train 3; PR #1954
pkg/combat/engine.go:954	jail-guard redirect omits CAN_SEE gate	e	src/fight.c:1374; TODO's DP-1054 citation is stale/unrelated; visibility callback required
pkg/game/mobact.go:176-179	Go has an extra wounded/charmed early return absent from C	f	src/mobact.c:54-105; fixed in train 4; PR #1955
pkg/game/mobact.go:203-230	scavenger scan omits room-aware CAN_SEE_OBJ in CAN_GET_OBJ	f	src/mobact.c:103-117; src/utils.h:535-549; fixed in train 4; PR #1955
pkg/game/mobact.go:376-405	memory special omits C room warning/growl before hit	a	C path verified at src/mobact.c:261-280; intersects owned shared attack/message transcript
pkg/game/mobact.go:414-458	helper branch omits charmed-master assist and C helper visibility/room-act gates	a	C path verified at src/mobact.c:284-302; assist attack bytes stay with shared combat owner
pkg/game/mobact.go:491-505	AGGR24+AGGRESSIVE mob target scan lacks C CAN_SEE predicate	b	src/mobact.c:324-336; add blocked depth row; target order also meets DP-1364
pkg/game/limits_condition.go:305-363	point-update timer effects only walk floor objects, unlike C object_list	b	src/limits.c:525-686 and src/handler.c:1046-1090; add blocked depth row; carried/contained coverage needed
pkg/game/weather.go:187	date_record override is absent	c	stop-tier persistence; lifecycle weather and time pulses row
pkg/session/wiz_system.go:740	hunt_items call absent after point_update	a	DP-493; do not implement
pkg/session/comm_cmds.go:187	do_write shortcut omits C item/permission checks and string editor	c	src/act.comm.c:1024-1103; stop-tier session/editor lifecycle
pkg/session/cmd_info.go:1151	help-miss usage-file append omitted	c	src/act.wizard.c help path; stop-tier file/persistence side effect
pkg/session/wiz_info.go:237	detailed `show shops` report omitted	e	src/shop.c:1413; src/act.wizard.c:2414; shop data/order already exist; implement `.` room selection, 1-based numeric selection, invalid input, and detailed transcript
pkg/session/string_cmd.go:20	C get_obj helper is unported but has no callers	d	rg found only its definition/use sites route through get_char_vis and get_obj_in_list_vis; docs/fidelity/depth/string.tsv records exclusion
pkg/game/deferred_fight_fns.go:36	APPLY_RACE_HATE slot mutation not separately ported	f	src/handler.c:205-238; 18 stock affect records across 14 object prototypes; fixed in train 1 via actual player wield/remove recomputation and covered by race-hate pulse dispatch proof
pkg/spells/affect_spells.go:2155	identify object bitvector currently prints NOBITS	d	src/spells.c:476-510; src/db.c:1372,1399-1457,1947-1951; ordinary stock object prototypes have no bitvector field or gameplay setter, so C prints NOBITS; nonzero legacy rent data is stop-tier persistence
pkg/game/ai.go:79	wandering treats every mob as unable to swim/fly in water/flying sectors	e	src/mobact.c:130-138; src/utils.h:148-153; Go roomActivityHasBoat already models AFF_WATERWALK/AFF_FLY/ITEM_BOAT for mobs; add wander predicates and pulse proofs
pkg/game/houses.go:56,94; pkg/game/house_save.go:33	house lookups and rent object load/save have incomplete adapter/persistence seams	c	lookup callbacks are registered in pkg/session/manager.go:258; house object persistence is stop-tier
pkg/session/tattoo.go:64	comment says skull tattoo is message-only	d	stale: game/use_tattoo_depth_test.go proves mob spawn, follow and charm; tattoo cooldown persistence remains stop-tier
pkg/session/tattoo.go	tattoo effects direct stats and tattoo/timer not serialized	c	C tattoo_af is effective-stat mutation; persistence is stop-tier
pkg/game/skills2.go:511-520	DoTurn guesses undead status from target name and accepts one target	e	src/interpreter.c:787; src/new_cmds2.c:960-1030; fixable: C spends level*3+number(1,55) mana and scans room order by undead/vampire race; diffs -4..3 get no target effect, while Go adds damage/messages; Go also omits C mana/RNG/room-wide flow and guesses race from names
pkg/game/skills2.go:274-275	DoMindlink called simplified	d	src/new_cmds2.c:254-317; src/utils.h:366,418; player targets take C's non-NPC return and valid NPC targets take the tested failure/RNG path, while the apparent success arm is unreachable
pkg/game/skills2.go:482	DoDig section says sector-based gameplay	d	stale/misplaced: C do_dig is builder exit creation (new_cmds2.c:818-881), implemented by session/wiz_movement.go
pkg/game/level.go:59	int_app TODO says display/practice wiring missing	d	stale: session/wiz_stats.go and game/spec_procs.go call IntAppLearn
pkg/game/act_display.go:54	score experience said estimated	d	stale: session/display_cmds.go uses game.FindExp
pkg/game/act_movement.go:943	door plural grammar described as rough approximation	d	C's strrchr/final-s test is equivalent to Go's final-byte-s check
pkg/game/limits_gain.go:154	integer approximation label	d	integer division truncation is C-equivalent
pkg/game/death.go:871	damage mapping described approximate	e	src/fight.c:258,578; src/spells.h:89,101,105,165,169; current mapping uses 21 for energy drain (C=25), 31 for shocking grasp (C=37), 34 for psiblast (C=100), and 35 for petrify (C=104), and omits meteor swarm (C=41); existing death_test.go asserts several wrong IDs
pkg/game/skill_advanced.go:256; pkg/game/graph.go:248; pkg/game/damage_stubs.go:38; pkg/session/informative_cmds.go:11-13; pkg/session/agent_vars.go:139	legacy stub wording	d	comments are historical or helpers delegate to live implementations; no current unported C branch identified
pkg/session/cmd_group.go:19	shadow TODO	d	stale: `shadow.tsv` and movement_commands.go prove success arm
pkg/game/actor_specials.go	legacy NPC dispatch limitation	a	owned with generic NPC command interpreter; not expanded in this sweep
pkg/scripting/engine.go:114-117	dangerous Lua library names intentionally replaced with sandbox error stubs	d	security sandbox behavior, not a missing gameplay routine
pkg/scripting/engine.go:806,828,1046,1096,1378,1421-1423,1527	stub headings, empty skills/wear tables, archived skill groups, stale canget comment	d	production RunScript always uses `charToTable` bridge (engine.go:453-468); bridge emits C's fields and nil-vs-table wear/carry (bridge.go:288-343); empty `skills` is absent from C char_to_table, skill-group scripts are archived with no C cmdlib entries (scripts.c:1609-1668), Tier API functions are registered, and the canget comment about the invalid-argument return is stale
pkg/scripting/engine.go:1392-1397	iscorpse uses the wrong object type and omits C's corpse value flag	e	src/scripts.c:636-654; src/utils.h:490-491; src/structs.h:434; C tests ITEM_CONTAINER=15 and value[3]==1, while Go checks type 18 only
pkg/scripting/engine.go:1411-1459	canget checks player `ch` and player carry capacity instead of acting mobile `me`	f	src/scripts.c:193-219; src/utils.h:535-549; fixed in train 4 / PR #1955 via live `me`/object references and `CAN_GET_OBJ` visibility proof (`lua.bind-canget-world-actor`, `TestLuaCanGetUsesMeAndLiveObjectVisibility`)
pkg/scripting/bridge.go:293-326	Lua leader table recursion stops at depth 32	e	src/scripts.c:1870-1873; src/utils.c:365-375; C recursively calls char_to_table for every master until nil, and its follow path rejects cycles
pkg/db/player.go:22	stub phrase	d	historical comment; current player store is SQLite implementation; not a gameplay C behavior omission
pkg/session/informative_cmds.go:11-13	file header says informative commands are stubs	d	stale: cmdTitle is a live command implementation
pkg/admin/huma_completion.go:272-274	admin zone reset placeholder	d	non-game admin API; outside requested pulse/command fidelity surface
```

## Appendix B — Full assigned SPECIAL crosswalk

This is the complete row-level crosswalk for every distinct assigned special, including C assignment, C definition, and Go registration evidence.

```tsv
special	c_assignments	c_definition	go_registration	branch_ledger
assassin	ASSIGNROOM 8114 @ src/spec_assign.c:608	src/spec_procs2.c:845	pkg/game/spec_procs2.go:30	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
backstabber	ASSIGNMOB 9151 @ src/spec_assign.c:309	src/spec_procs2.c:2003	pkg/game/spec_procs2.go:61	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
bank	ASSIGNOBJ 8034 @ src/spec_assign.c:535; ASSIGNOBJ 18224 @ src/spec_assign.c:561	src/spec_procs.c:2345	pkg/game/spec_procs4.go:37	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
beholder	ASSIGNMOB 11011 @ src/spec_assign.c:332	src/new_cmds2.c:328	pkg/game/spec_procs_missing.go:205	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
brain_eater	ASSIGNMOB 14420 @ src/spec_assign.c:386; ASSIGNMOB 14432 @ src/spec_assign.c:389	src/spec_procs3.c:198	pkg/game/spec_procs3.go:47	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
breed_killer	ASSIGNMOB 7900 @ src/spec_assign.c:259; ASSIGNMOB 7910 @ src/spec_assign.c:264	src/spec_procs2.c:1679	pkg/game/spec_procs2.go:51	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
butler	ASSIGNMOB 8092 @ src/spec_assign.c:300	src/spec_procs3.c:144	pkg/game/spec_procs3.go:46	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
carrion	ASSIGNROOM 14305 @ src/spec_assign.c:612	src/spec_procs2.c:1726	pkg/game/spec_procs2.go:52	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
castle_guard_down	ASSIGNMOB 19626 @ src/spec_assign.c:453; ASSIGNMOB 19627 @ src/spec_assign.c:454; ASSIGNMOB 19640 @ src/spec_assign.c:455; ASSIGNMOB 19641 @ src/spec_assign.c:456	src/spec_procs2.c:2134	pkg/game/spec_procs2.go:70	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
castle_guard_north	ASSIGNMOB 19510 @ src/spec_assign.c:447; ASSIGNMOB 19601 @ src/spec_assign.c:450; ASSIGNMOB 19602 @ src/spec_assign.c:451; ASSIGNMOB 19690 @ src/spec_assign.c:459; ASSIGNMOB 19691 @ src/spec_assign.c:460; ASSIGNMOB 19675 @ src/spec_assign.c:461; ASSIGNMOB 19676 @ src/spec_assign.c:462	src/spec_procs2.c:2218	pkg/game/spec_procs2.go:72	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
castle_guard_up	ASSIGNMOB 19650 @ src/spec_assign.c:457; ASSIGNMOB 19651 @ src/spec_assign.c:458	src/spec_procs2.c:2176	pkg/game/spec_procs2.go:71	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
chosen_guard	ASSIGNMOB 8096 @ src/spec_assign.c:302	src/spec_procs2.c:2113	pkg/game/spec_procs2.go:69	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
citizen	ASSIGNMOB 8062 @ src/spec_assign.c:293; ASSIGNMOB 18202 @ src/spec_assign.c:402	src/spec_procs.c:986	pkg/game/spec_procs.go:1282	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
cityguard	ASSIGNMOB 2747 @ src/spec_assign.c:212; ASSIGNMOB 18215 @ src/spec_assign.c:408; ASSIGNMOB 21200 @ src/spec_assign.c:493; ASSIGNMOB 21201 @ src/spec_assign.c:494; ASSIGNMOB 21202 @ src/spec_assign.c:495; ASSIGNMOB 21203 @ src/spec_assign.c:496; ASSIGNMOB 21227 @ src/spec_assign.c:503; ASSIGNMOB 21228 @ src/spec_assign.c:504	src/spec_procs.c:771	pkg/game/spec_procs.go:1279	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
cleric	ASSIGNMOB 1305 @ src/spec_assign.c:197; ASSIGNMOB 4102 @ src/spec_assign.c:225; ASSIGNMOB 7970 @ src/spec_assign.c:268; ASSIGNMOB 11023 @ src/spec_assign.c:321; ASSIGNMOB 11023 @ src/spec_assign.c:334; ASSIGNMOB 11024 @ src/spec_assign.c:335; ASSIGNMOB 11038 @ src/spec_assign.c:338; ASSIGNMOB 11039 @ src/spec_assign.c:339; ASSIGNMOB 14202 @ src/spec_assign.c:361; ASSIGNMOB 14206 @ src/spec_assign.c:362; ASSIGNMOB 14220 @ src/spec_assign.c:363; ASSIGNMOB 14309 @ src/spec_assign.c:373; ASSIGNMOB 12877 @ src/spec_assign.c:429; ASSIGNMOB 20005 @ src/spec_assign.c:472; ASSIGNMOB 20018 @ src/spec_assign.c:478; ASSIGNMOB 20029 @ src/spec_assign.c:485; ASSIGNMOB 20035 @ src/spec_assign.c:487; ASSIGNMOB 20041 @ src/spec_assign.c:489; ASSIGNMOB 21221 @ src/spec_assign.c:501	src/spec_procs.c:1425	pkg/game/spec_procs3.go:75	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
clerk	ASSIGNMOB 18228 @ src/spec_assign.c:407	src/spec_procs3.c:75	pkg/game/spec_procs3.go:45	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
con_seller	ASSIGNMOB 21246 @ src/spec_assign.c:508	src/spec_procs3.c:239	pkg/game/spec_procs3.go:49	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
conjured	ASSIGNMOB 81 @ src/spec_assign.c:186; ASSIGNMOB 82 @ src/spec_assign.c:187; ASSIGNMOB 83 @ src/spec_assign.c:188; ASSIGNMOB 84 @ src/spec_assign.c:189; ASSIGNMOB 85 @ src/spec_assign.c:190; ASSIGNMOB 86 @ src/spec_assign.c:191	src/spec_procs3.c:859	pkg/game/spec_procs3.go:63	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
cuchi	ASSIGNMOB 18306 @ src/spec_assign.c:415	src/spec_procs.c:1034	pkg/game/spec_procs.go:1283	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
dracula	ASSIGNMOB 7903 @ src/spec_assign.c:262; ASSIGNMOB 14110 @ src/spec_assign.c:432	src/spec_procs.c:1798	pkg/game/spec_procs4.go:25	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
dragon_breath	ASSIGNMOB 4209 @ src/spec_assign.c:229; ASSIGNMOB 4705 @ src/spec_assign.c:233; ASSIGNMOB 11000 @ src/spec_assign.c:325; ASSIGNMOB 11001 @ src/spec_assign.c:326; ASSIGNMOB 11002 @ src/spec_assign.c:327; ASSIGNMOB 20027 @ src/spec_assign.c:484	src/spec_procs.c:926	pkg/game/spec_procs.go:1281	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
dump	ASSIGNROOM 8085 @ src/spec_assign.c:610; ASSIGNROOM 21223 @ src/spec_assign.c:617	src/spec_procs.c:254	pkg/game/spec_procs.go:1268	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
elements_galeru_alive	ASSIGNROOM 1394 @ src/spec_assign.c:632	src/spec_procs3.c:1184	pkg/game/spec_procs3.go:70	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
elements_galeru_column	ASSIGNROOM 1372 @ src/spec_assign.c:631	src/spec_procs3.c:1137	pkg/game/spec_procs3.go:69	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
elements_guardian	ASSIGNMOB 1314 @ src/spec_assign.c:196	src/spec_procs3.c:1242	pkg/game/spec_procs3.go:72	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
elements_load_cylinders	ASSIGNROOM 1360 @ src/spec_assign.c:627; ASSIGNROOM 1364 @ src/spec_assign.c:628; ASSIGNROOM 1380 @ src/spec_assign.c:629; ASSIGNROOM 1384 @ src/spec_assign.c:630	src/spec_procs3.c:1058	pkg/game/spec_procs3.go:68	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
elements_master_column	ASSIGNROOM 1315 @ src/spec_assign.c:622	src/spec_procs3.c:936	pkg/game/spec_procs3.go:66	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
elements_minion	ASSIGNMOB 1313 @ src/spec_assign.c:195	src/spec_procs3.c:1217	pkg/game/spec_procs3.go:71	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
elements_platforms	ASSIGNROOM 1326 @ src/spec_assign.c:623; ASSIGNROOM 1337 @ src/spec_assign.c:624; ASSIGNROOM 1348 @ src/spec_assign.c:625; ASSIGNROOM 1359 @ src/spec_assign.c:626	src/spec_procs3.c:1004	pkg/game/spec_procs3.go:67	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
eq_thief	ASSIGNMOB 7979 @ src/spec_assign.c:266; ASSIGNMOB 12118 @ src/spec_assign.c:350; ASSIGNMOB 14225 @ src/spec_assign.c:366	src/spec_procs2.c:1613	pkg/game/spec_procs2.go:47	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
fido	ASSIGNMOB 8063 @ src/spec_assign.c:294; ASSIGNMOB 12115 @ src/spec_assign.c:349; ASSIGNMOB 15108 @ src/spec_assign.c:394; ASSIGNMOB 18203 @ src/spec_assign.c:403	src/spec_procs.c:724	pkg/game/spec_procs.go:1277	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
field_object	ASSIGNOBJ 50 @ src/spec_assign.c:528; ASSIGNOBJ 52 @ src/spec_assign.c:529	src/spec_procs3.c:456	pkg/game/spec_procs3.go:55	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
fighter	ASSIGNMOB 4914 @ src/spec_assign.c:240; ASSIGNMOB 5200 @ src/spec_assign.c:246; ASSIGNMOB 7901 @ src/spec_assign.c:260; ASSIGNMOB 7902 @ src/spec_assign.c:261; ASSIGNMOB 12111 @ src/spec_assign.c:348; ASSIGNMOB 14407 @ src/spec_assign.c:379; ASSIGNMOB 12850 @ src/spec_assign.c:427; ASSIGNMOB 20002 @ src/spec_assign.c:469; ASSIGNMOB 20011 @ src/spec_assign.c:476; ASSIGNMOB 20019 @ src/spec_assign.c:479; ASSIGNMOB 20020 @ src/spec_assign.c:480; ASSIGNMOB 20036 @ src/spec_assign.c:488; ASSIGNMOB 20042 @ src/spec_assign.c:490	src/spec_procs.c:509	pkg/game/spec_procs.go:1273	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
fly_exit_up	ASSIGNROOM 1389 @ src/spec_assign.c:635	src/spec_procs3.c:1289	pkg/game/spec_procs3.go:73	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
gen_board	ASSIGNOBJ 8064 @ src/spec_assign.c:536; ASSIGNOBJ 8065 @ src/spec_assign.c:537; ASSIGNOBJ 8096 @ src/spec_assign.c:538; ASSIGNOBJ 8097 @ src/spec_assign.c:539; ASSIGNOBJ 8098 @ src/spec_assign.c:540; ASSIGNOBJ 8099 @ src/spec_assign.c:541; ASSIGNOBJ 19652 @ src/spec_assign.c:542; ASSIGNOBJ 19601 @ src/spec_assign.c:543; ASSIGNOBJ 19627 @ src/spec_assign.c:553; ASSIGNOBJ 19640 @ src/spec_assign.c:554; ASSIGNOBJ 19666 @ src/spec_assign.c:555; ASSIGNOBJ 19677 @ src/spec_assign.c:556	src/boards.c:179	pkg/game/boards.go:94	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
guild	ASSIGNMOB 4813 @ src/spec_assign.c:236; ASSIGNMOB 4821 @ src/spec_assign.c:237; ASSIGNMOB 4825 @ src/spec_assign.c:238; ASSIGNMOB 8012 @ src/spec_assign.c:278; ASSIGNMOB 8013 @ src/spec_assign.c:279; ASSIGNMOB 8014 @ src/spec_assign.c:280; ASSIGNMOB 8015 @ src/spec_assign.c:281; ASSIGNMOB 8024 @ src/spec_assign.c:304; ASSIGNMOB 8026 @ src/spec_assign.c:305; ASSIGNMOB 21214 @ src/spec_assign.c:497; ASSIGNMOB 21215 @ src/spec_assign.c:498; ASSIGNMOB 21216 @ src/spec_assign.c:499; ASSIGNMOB 21217 @ src/spec_assign.c:500	src/spec_procs.c:201	pkg/game/spec_procs.go:1267	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
guild_guard	ASSIGNMOB 8014 @ src/spec_assign.c:271; ASSIGNMOB 8017 @ src/spec_assign.c:272; ASSIGNMOB 8016 @ src/spec_assign.c:273; ASSIGNMOB 8018 @ src/spec_assign.c:274; ASSIGNMOB 8019 @ src/spec_assign.c:275; ASSIGNMOB 8025 @ src/spec_assign.c:276; ASSIGNMOB 8083 @ src/spec_assign.c:306	src/spec_procs.c:569	pkg/game/spec_procs.go:1275	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
horn	ASSIGNOBJ 14415 @ src/spec_assign.c:558	src/spec_procs.c:2401	pkg/game/spec_procs4.go:38	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
identifier	ASSIGNMOB 8087 @ src/spec_assign.c:297	src/spec_procs2.c:1193	pkg/game/spec_procs2.go:37	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
jail	ASSIGNROOM 8118 @ src/spec_assign.c:609	src/spec_procs2.c:1470	pkg/game/spec_procs2.go:45	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
jailguard	ASSIGNMOB 8088 @ src/spec_assign.c:298	src/spec_procs.c:1781	pkg/game/spec_procs4.go:24	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
janitor	ASSIGNMOB 8061 @ src/spec_assign.c:292; ASSIGNMOB 21229 @ src/spec_assign.c:505	src/spec_procs.c:750	pkg/game/spec_procs.go:1278	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
key_seller	ASSIGNMOB 19610 @ src/spec_assign.c:452	src/spec_procs2.c:1870	pkg/game/spec_procs2.go:56	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
magic_user	ASSIGNMOB 1307 @ src/spec_assign.c:198; ASSIGNMOB 1315 @ src/spec_assign.c:199; ASSIGNMOB 2108 @ src/spec_assign.c:203; ASSIGNMOB 2716 @ src/spec_assign.c:207; ASSIGNMOB 2720 @ src/spec_assign.c:208; ASSIGNMOB 2732 @ src/spec_assign.c:209; ASSIGNMOB 2733 @ src/spec_assign.c:210; ASSIGNMOB 2734 @ src/spec_assign.c:211; ASSIGNMOB 3103 @ src/spec_assign.c:219; ASSIGNMOB 3113 @ src/spec_assign.c:220; ASSIGNMOB 3118 @ src/spec_assign.c:221; ASSIGNMOB 4101 @ src/spec_assign.c:224; ASSIGNMOB 4202 @ src/spec_assign.c:228; ASSIGNMOB 4704 @ src/spec_assign.c:232; ASSIGNMOB 4916 @ src/spec_assign.c:241; ASSIGNMOB 4919 @ src/spec_assign.c:242; ASSIGNMOB 5503 @ src/spec_assign.c:249; ASSIGNMOB 5507 @ src/spec_assign.c:250; ASSIGNMOB 7023 @ src/spec_assign.c:255; ASSIGNMOB 7969 @ src/spec_assign.c:267; ASSIGNMOB 8081 @ src/spec_assign.c:295; ASSIGNMOB 9903 @ src/spec_assign.c:313; ASSIGNMOB 9905 @ src/spec_assign.c:314; ASSIGNMOB 9911 @ src/spec_assign.c:315; ASSIGNMOB 11024 @ src/spec_assign.c:322; ASSIGNMOB 11029 @ src/spec_assign.c:323; ASSIGNMOB 11030 @ src/spec_assign.c:324; ASSIGNMOB 11005 @ src/spec_assign.c:328; ASSIGNMOB 11006 @ src/spec_assign.c:329; ASSIGNMOB 11007 @ src/spec_assign.c:330; ASSIGNMOB 11008 @ src/spec_assign.c:331; ASSIGNMOB 11016 @ src/spec_assign.c:333; ASSIGNMOB 11029 @ src/spec_assign.c:336; ASSIGNMOB 11030 @ src/spec_assign.c:337; ASSIGNMOB 11706 @ src/spec_assign.c:345; ASSIGNMOB 14219 @ src/spec_assign.c:364; ASSIGNMOB 14221 @ src/spec_assign.c:365; ASSIGNMOB 14314 @ src/spec_assign.c:371; ASSIGNMOB 14306 @ src/spec_assign.c:372; ASSIGNMOB 14406 @ src/spec_assign.c:378; ASSIGNMOB 14435 @ src/spec_assign.c:390; ASSIGNMOB 15804 @ src/spec_assign.c:419; ASSIGNMOB 15807 @ src/spec_assign.c:420; ASSIGNMOB 12848 @ src/spec_assign.c:426; ASSIGNMOB 18601 @ src/spec_assign.c:435; ASSIGNMOB 18603 @ src/spec_assign.c:436; ASSIGNMOB 18604 @ src/spec_assign.c:437; ASSIGNMOB 19412 @ src/spec_assign.c:443; ASSIGNMOB 20003 @ src/spec_assign.c:470; ASSIGNMOB 20008 @ src/spec_assign.c:473; ASSIGNMOB 20009 @ src/spec_assign.c:474; ASSIGNMOB 20010 @ src/spec_assign.c:475; ASSIGNMOB 20014 @ src/spec_assign.c:477; ASSIGNMOB 20023 @ src/spec_assign.c:481; ASSIGNMOB 20025 @ src/spec_assign.c:482; ASSIGNMOB 20026 @ src/spec_assign.c:483; ASSIGNMOB 20030 @ src/spec_assign.c:486	src/spec_procs.c:409	pkg/game/spec_procs.go:1272	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
medusa	ASSIGNMOB 14101 @ src/spec_assign.c:355; ASSIGNMOB 14102 @ src/spec_assign.c:356	src/spec_procs2.c:1552	pkg/game/spec_procs2.go:46	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
mindflayer	ASSIGNMOB 14414 @ src/spec_assign.c:383	src/spec_procs2.c:1972	pkg/game/spec_procs2.go:60	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
moon_gate	ASSIGNOBJ 4001 @ src/spec_assign.c:531; ASSIGNOBJ 4002 @ src/spec_assign.c:532; ASSIGNOBJ 19604 @ src/spec_assign.c:544; ASSIGNOBJ 19605 @ src/spec_assign.c:545; ASSIGNOBJ 19606 @ src/spec_assign.c:546; ASSIGNOBJ 19607 @ src/spec_assign.c:547; ASSIGNOBJ 19608 @ src/spec_assign.c:548; ASSIGNOBJ 19609 @ src/spec_assign.c:549; ASSIGNOBJ 19610 @ src/spec_assign.c:550; ASSIGNOBJ 19611 @ src/spec_assign.c:551	src/gate.c:242	pkg/game/spec_procs_missing.go:207	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
mortician	ASSIGNMOB 8095 @ src/spec_assign.c:301	src/spec_procs3.c:807	pkg/game/spec_procs3.go:62	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
never_die	ASSIGNMOB 14401 @ src/spec_assign.c:376; ASSIGNMOB 19119 @ src/spec_assign.c:440	src/spec_procs2.c:2075	pkg/game/spec_procs2.go:67	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
newbie_zone_entrance	ASSIGNROOM 16300 @ src/spec_assign.c:607	src/spec_procs.c:2262	pkg/game/spec_procs4.go:33	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
no_get	ASSIGNMOB 14416 @ src/spec_assign.c:385; ASSIGNMOB 14430 @ src/spec_assign.c:388	src/new_cmds2.c:552	pkg/game/spec_procs_missing.go:203	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
no_move_east	ASSIGNMOB 14410 @ src/spec_assign.c:380; ASSIGNMOB 12876 @ src/spec_assign.c:428	src/spec_procs2.c:1823	pkg/game/spec_procs2.go:55	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
no_move_south	ASSIGNMOB 16308 @ src/spec_assign.c:399	src/spec_procs2.c:2092	pkg/game/spec_procs2.go:68	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
no_move_west	ASSIGNMOB 2106 @ src/spec_assign.c:202; ASSIGNMOB 14421 @ src/spec_assign.c:387	src/spec_procs2.c:2033	pkg/game/spec_procs2.go:63	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
normal_checker	ASSIGNMOB 18301 @ src/spec_assign.c:411; ASSIGNMOB 18302 @ src/spec_assign.c:412; ASSIGNMOB 18303 @ src/spec_assign.c:413; ASSIGNMOB 18304 @ src/spec_assign.c:414	src/spec_procs2.c:162	pkg/game/spec_procs2.go:19	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
oro_quarters_room	ASSIGNROOM 18397 @ src/spec_assign.c:614	src/spec_procs.c:2302	pkg/game/spec_procs4.go:35	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
oro_study_room	ASSIGNROOM 18399 @ src/spec_assign.c:613	src/spec_procs.c:2321	pkg/game/spec_procs4.go:36	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
outofjailguard	ASSIGNMOB 8089 @ src/spec_assign.c:299	src/spec_procs.c:1763	pkg/game/spec_procs4.go:23	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
paladin	ASSIGNMOB 71 @ src/spec_assign.c:258; ASSIGNMOB 7915 @ src/spec_assign.c:265	src/spec_procs.c:537	pkg/game/spec_procs.go:1274	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
pet_shops	ASSIGNROOM 21235 @ src/spec_assign.c:618	src/spec_procs.c:1844	pkg/game/spec_procs4.go:26	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
pissedalchemist	ASSIGNMOB 15814 @ src/spec_assign.c:422	src/spec_procs2.c:546	pkg/game/spec_procs2.go:28	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
portal_to_temple	ASSIGNROOM 19658 @ src/spec_assign.c:615	src/spec_procs3.c:515	pkg/game/spec_procs3.go:56	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
postmaster	ASSIGNMOB 3010 @ src/spec_assign.c:214; ASSIGNMOB 21225 @ src/spec_assign.c:502	src/mail.c:476	pkg/game/postmaster.go:39	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
pray_for_items	ASSIGNROOM 8008 @ src/spec_assign.c:605	src/spec_procs.c:2068	pkg/game/spec_procs4.go:30	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
prostitute	ASSIGNMOB 8023 @ src/spec_assign.c:283	src/spec_procs3.c:670	pkg/game/spec_procs3.go:60	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
puff	ASSIGNMOB 1 @ src/spec_assign.c:182	src/spec_procs.c:611	pkg/game/spec_procs.go:1276	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
quan_lo	ASSIGNMOB 19405 @ src/spec_assign.c:444	src/spec_procs3.c:372	pkg/game/spec_procs3.go:52	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
recharger	ASSIGNMOB 8 @ src/spec_assign.c:184	src/new_cmds2.c:415	pkg/game/spec_procs_missing.go:204	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
recruiter	ASSIGNMOB 16300 @ src/spec_assign.c:398	src/spec_procs3.c:905	pkg/game/spec_procs3.go:65	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
remorter	ASSIGNMOB 4 @ src/spec_assign.c:183	src/spec_procs2.c:682	pkg/game/spec_procs2.go:29	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
rescuer	ASSIGNMOB 7909 @ src/spec_assign.c:263; ASSIGNMOB 15808 @ src/spec_assign.c:421	src/spec_procs2.c:523	pkg/game/spec_procs2.go:27	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
roach	ASSIGNMOB 23 @ src/spec_assign.c:185	src/spec_procs3.c:707	pkg/game/spec_procs3.go:61	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
snake	ASSIGNMOB 14103 @ src/spec_assign.c:357; ASSIGNMOB 14127 @ src/spec_assign.c:358; ASSIGNMOB 14415 @ src/spec_assign.c:384	src/spec_procs.c:330	pkg/game/spec_procs.go:1269	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
stableboy	ASSIGNMOB 8022 @ src/spec_assign.c:282	src/spec_procs2.c:315	pkg/game/spec_procs2.go:23	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
start_room	ASSIGNROOM 8099 @ src/spec_assign.c:606	src/spec_procs.c:2204	pkg/game/spec_procs4.go:32	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
suck_in	ASSIGNROOM 20073 @ src/spec_assign.c:616	src/spec_procs.c:2279	pkg/game/spec_procs4.go:34	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
take_to_jail	ASSIGNMOB 8027 @ src/spec_assign.c:285; ASSIGNMOB 8059 @ src/spec_assign.c:286; ASSIGNMOB 8001 @ src/spec_assign.c:289; ASSIGNMOB 8002 @ src/spec_assign.c:290; ASSIGNMOB 8020 @ src/spec_assign.c:291	src/spec_procs2.c:1427	pkg/game/spec_procs2.go:44	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
tattoo1	ASSIGNMOB 8086 @ src/spec_assign.c:296	src/spec_procs2.c:945	pkg/game/spec_procs2.go:31	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
tattoo2	ASSIGNMOB 18213 @ src/spec_assign.c:404	src/spec_procs2.c:1010	pkg/game/spec_procs2.go:32	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
tattoo3	ASSIGNMOB 21244 @ src/spec_assign.c:507	src/spec_procs2.c:1075	pkg/game/spec_procs2.go:35	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
tattoo4	ASSIGNMOB 2766 @ src/spec_assign.c:213	src/spec_procs2.c:1282	pkg/game/spec_procs2.go:38	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
teleport_victim	ASSIGNMOB 14405 @ src/spec_assign.c:377	src/spec_procs3.c:225	pkg/game/spec_procs3.go:48	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
teleporter	ASSIGNMOB 14411 @ src/spec_assign.c:381	src/spec_procs2.c:2019	pkg/game/spec_procs2.go:62	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
thief	ASSIGNMOB 3119 @ src/spec_assign.c:218; ASSIGNMOB 12127 @ src/spec_assign.c:351; ASSIGNMOB 18218 @ src/spec_assign.c:405; ASSIGNMOB 20004 @ src/spec_assign.c:471; ASSIGNMOB 21242 @ src/spec_assign.c:506	src/spec_procs.c:389	pkg/game/spec_procs.go:1271	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
troll	ASSIGNMOB 10029 @ src/spec_assign.c:318; ASSIGNMOB 14100 @ src/spec_assign.c:354; ASSIGNMOB 14311 @ src/spec_assign.c:369; ASSIGNMOB 14312 @ src/spec_assign.c:370; ASSIGNMOB 19900 @ src/spec_assign.c:465; ASSIGNMOB 19901 @ src/spec_assign.c:466	src/spec_procs3.c:347	pkg/game/spec_procs3.go:51	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
wall_guard_ns	ASSIGNMOB 8060 @ src/spec_assign.c:287	src/spec_procs2.c:2260	pkg/game/spec_procs2.go:73	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
werewolf	ASSIGNMOB 5510 @ src/spec_assign.c:251	src/spec_procs3.c:427	pkg/game/spec_procs3.go:54	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
whirlpool	ASSIGNMOB 12200 @ src/spec_assign.c:342	src/spec_procs2.c:244	pkg/game/spec_procs2.go:21	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
zen_master	ASSIGNMOB 18219 @ src/spec_assign.c:406	src/new_cmds2.c:658	pkg/game/spec_procs_missing.go:206	docs/fidelity/depth/spec-procs.tsv (branch-level statuses)
```
