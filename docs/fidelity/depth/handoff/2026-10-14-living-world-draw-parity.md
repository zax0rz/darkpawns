# Living-world draw-parity train (DP-1371 / DP-1427)

Date: 2026-10-14
Branch: `codex/luna-dp-1364`
Base: `2c11916b` (`origin/main`, PR #1968 merge)
Reference oracle SHA-256: `bd6c548c6fbc41d35eede384e4ce2c5a858ff2ce0f568a379d7fb366a9051b07`

Train size: five fixes across 533 changed production lines (299 added, 234 removed). The train stops at the shared combat, mobile-aggression, room-activity, and session-capture family boundaries.

## Fixes landed

Five fixes are split into five commits:

1. `48e1e9b58 fix: restore C pulse iteration order` — adds one newest-first sequence across players and mobiles for `MobileActivity`, `RoomActivity`, `PointUpdate`, and `AffectUpdate`, and walks live objects by descending creation ID in `ObjectActivity`. C prepends both `create_char()` and `read_mobile()` to `character_list` (`src/db.c:1716-1717,1745-1746`), prepends new objects to `object_list` (`src/db.c:1892-1893,1950-1951`), and prepends room arrivals in `char_to_room()` (`src/handler.c:532-550`). The pulse loops walk these lists in `src/comm.c:700-702,758-765`. Tests: `TestMobileActivityWalksNewestMobFirst`, `TestCOrderedCharactersMergesPlayerAndMobileInsertions`, and `TestLiveObjectsInCOrder`.
2. `4227878ca fix: run mob greet triggers after wandering` — after a successful `perform_move`, runs `mp_greet` and room greet scripts after `entry_prog` when the mover is not sneaking, matching `src/act.movement.c:265-284`. The dog check in `src/mobprog.c:62-88` consumes `number(0,5)`. Test: `TestMobileActivityRunsMobGreetAfterWander`.
3. `2fbe85689 fix: preserve default mobile sound draws` — lets `mp_sound` reach its demon and dog tails when a mobile VNum has no sound-switch case. C runs these tails in `src/mobprog.c:612-645`; `src/mobact.c:143-147` calls `mp_sound` after the unconditional `number(0,15)` gate. The Go path now retains the demon alignment transition and uses the mobile say/emote paths. Tests: `TestMpSoundRunsDefaultMobileTail` and `TestMpSoundDemonAdvancesStateAfterTailDraw`.
4. `c1da6de0e fix: match cleric pulse draw bounds` — C uses `number(1,healperc+2)<3` at `src/spec_procs.c:1487`; Go had an equivalent branch probability with the wrong zero-based bounds. The implementation now uses `clericNumber(1, healPerc+2) >= 3`; the focused tests assert the exact range. Direct red control returns to stream 4116; fixed seed 1 matches through stream 4131.
5. `39ad131f8 fix: preserve butler container draw parity` — `SPECIAL(butler)` calls `do_gen_door` on state-changing opens and closes (`src/spec_procs3.c:144-196`), whose successful path reaches `ok_pick` and consumes `number(1,101)` (`src/act.movement.c:541`). Go’s helper now preserves that draw at the same state transition; the focused test pins three close draws and the `1..101` range. Direct red control returns to stream 4109; fixed seed 1 matches through stream 4120.

## R5h controls and first-mismatch movement

Seed-1 Haven evidence is retained under `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-soaks-seed1/diagnosis/`.

| Control / state | First draw result |
|---|---|
| Before the greet fix | `stream=68836 C Number(0,5)=1; Go Number(0,15)=4` |
| Remove `MpGreetActor` | Same first mismatch: `stream=68836 C Number(0,5)=1; Go Number(0,15)=4` (`haven-seed1-red-no-mpgreet.txt`) |
| With greet fix, before the sound-tail fix | `stream=68844 C Number(0,25)=22; Go Number(0,18)=16` (`haven-seed1-mpgreet-fix.txt`) |
| Restore `MpSound`'s default early return | Same first mismatch: `stream=68844 C Number(0,25)=22; Go Number(0,18)=16` (`haven-seed1-red-no-mpsound-tail.txt`) |
| With all three fixes | Haven advances to `stream=81523 C Dice(1,1)=1; Go Number(0,20)=8` (`haven-seed1-roomactivity-investigation.txt`) |
| Revert the cleric range | `stream=4116 C Number(1,2)=2; Go Number(0,1)=1`; with the fix, `spec-proc-cleric` matches 4,088 entries through stream 4,131 |
| Revert the Butler transition draws | `stream=4109 C Number(1,101)=8; Go Number(0,3)=0`; with the fix, `spec-proc-butler` matches 4,120 entries through stream 4,120 |

The cleric and Butler direct reports are retained outside census folders at `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-dp1364-fix-controls/`.

The corpus draw census after fix 1 reduced DRAW scenarios from 61 to 45. All 22 DP-1364 mobile-order mismatch instances disappeared; 16 scenario rows left the DRAW set, with no new DRAW rows. The retained post-fix results are at `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-dp1364-corpus1/`. The original 61-row corpus triage is at `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/draw-parity-corpus-1/`.

## Populated-world soak findings

The three new scenarios omit `quiet-mobs`, use the stock reset corpus, and run thousands of clock pulses. The seed-1 census at `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-soaks-seed1-after-mobile-fixes/` classified all three as content `FAIL`; its `draw_divergent=0` is only the census classification count (draw-only divergences are counted as DRAW when bytes match). The direct reports below preserve the first draw even when transcript bytes already diverged.

| Scenario / seed | First draw after mobile fixes | Source path and owner boundary |
|---|---|---|
| Haven / 1 | `stream=81523 C Dice(1,1)=1; Go Number(0,20)=8`, during `~dpclock pulse 4600` | The C range maps to `skill_message`'s message selector (`src/fight.c:1035`, reached from `damage()` at `:1534-1541`); Go's next draw is the `loudMobs` noise gate (`pkg/game/room_activity.go:423`). The report verifies these call sites but does not isolate a particular mobile arm; filed at the mobile/combat pulse boundary in [DP-1435](https://linear.app/labz0rz/issue/DP-1435/mobile-activity-room-acts-aggr24-grins-evilly-memory-warning-and-helper-assist-arms). |
| Dungeon entrance / 1 | Same `stream=81523` mismatch during `~dpclock pulse 6100` | Same verified C message-selector / Go noise-gate call-site pair; the report does not identify the triggering mobile arm. Retained report: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-soaks-seed1/diagnosis/dungeon-seed1-after-mobile-fixes.txt`; follow-up is filed in [DP-1435](https://linear.app/labz0rz/issue/DP-1435/mobile-activity-room-acts-aggr24-grins-evilly-memory-warning-and-helper-assist-arms). |
| Orc burrow / 1 | `stream=81146 C Number(1,20)=15; Go Number(0,18)=14`, during `~dpclock pulse 3600` | C's aggressive mobile arm calls synchronous `hit()` (`src/mobact.c:198-229`); its hit roll is `number(1,20)` in `src/fight.c:1815`. Go's corresponding branch only calls `StartCombat` (`pkg/game/mobact.go:320-329`) and proceeds to the wander-door draw at `pkg/game/mobact.go:223`. This verified mobile-combat call-path gap is tracked in [DP-1435](https://linear.app/labz0rz/issue/DP-1435/mobile-activity-room-acts-aggr24-grins-evilly-memory-warning-and-helper-assist-arms). Report: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-soaks-seed1/diagnosis/orc-seed1-after-mobile-fixes.txt`. |
| All three / 2 | `stream=60976 C Number(1,20)=17; Go Number(0,18)=15`, during setup | Same C mobile `hit()` roll (`src/mobact.c` attack arms → `src/fight.c:1815`) versus Go mobile door draw (`pkg/game/mobact.go:223`), tracked in [DP-1435](https://linear.app/labz0rz/issue/DP-1435/mobile-activity-room-acts-aggr24-grins-evilly-memory-warning-and-helper-assist-arms). Reports: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-seed2-diagnostics/`. |

These later mismatches cross the mobile-combat pulse-order family and the long-pulse session/output-capture stop tier. The direct orc chain is tracked in DP-1435; the repeated `Dice(1,1)` / `Number(0,20)` signatures are attached there as call-site evidence without assigning them to a particular mobile arm. This train stops at that family boundary (R1/R3/R5e). No harness, C source, oracle binary, save format, or governing document was changed.

The seed-1 differential reports also show why these soak cases remain census `FAIL`: at long pulse steps, C emits many room/combat bytes while the Go actor capture has no bytes for that same step. For example, Haven pulse 4600 has 245 C actor lines and 0 Go actor lines; its later `look` output matches again. This is a session/output-delivery symptom after thousands of pulses, within the goal’s explicit session-lifecycle/transport stop tier. The reports are retained in the soak diagnosis directory above and filed as [DP-1439](https://linear.app/labz0rz/issue/DP-1439/fidelity-r1r3-long-dp-clock-pulses-lose-go-actor-output-in-census); this train does not change session lifecycle, output transport, or the harness to mask it.

## Corpus draw triage

The retained corpus after the first three fixes is `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-dp1364-corpus2/`. It captured `f38081940` and reported 1,172 scenarios, 1,125 byte passes, 3 new soak `FAIL` rows, 1 `INFRA`, and 43 `DRAW` rows. The draw-only reports for those 43 rows are retained in `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-dp1364-corpus2-diagnostics/`. The first two later fixes below remove `spec-proc-cleric` and `spec-proc-butler` from this inventory; the post-fix corpus run will give the final count.

The original `TRIAGE.md` and `mismatch-clusters.txt` in `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/draw-parity-corpus-1/` cover the earlier 61-row inventory, not these first-draw results.

| Corpus-2 DRAW row(s) | First draw (C / Go) | Verified call path and disposition |
|---|---|---|
| `spec-proc-cleric` | `4116 Number(1,2)` / `Number(0,1)` | Fixed: C `SPECIAL(cleric)` at `src/spec_procs.c:1487`; Go `specCleric` now preserves the same branch and exact range. R3/R5h controls are retained above. |
| `spec-proc-butler` | `4109 Number(1,101)` / `Number(0,3)` | Fixed: C `SPECIAL(butler)` calls `do_gen_door`; C `ok_pick` draws at `src/act.movement.c:541`. Go’s transition helper now consumes that draw on successful state changes. R3/R5h controls are retained above. |
| `mortal-obj-batch` | `4064 Number(1,101)` / `Number(0,3)` | Filed as [DP-1436](https://linear.app/labz0rz/issue/DP-1436/fidelity-r3-do-palm-skips-its-entry-draw-on-no-argument-input): C `do_palm` draws before its no-argument return (`src/new_cmds.c:2112`); Go's `CmdPalm` returns before `DoPalm` (`pkg/command/skill_commands.go:2046-2058`). The retained scenario is bare `palm`; `DoPalm` also defers its draw until after item/weight checks (`pkg/game/skill_special.go:710-742`). This is outside the mobile/special pulse family. |
| `peek-depth` | `4102 Number(1,101)` / `Number(1,200)` | Filed as [DP-1437](https://linear.app/labz0rz/issue/DP-1437/fidelity-r3-do-peek-omits-its-percent-draw-for-immortal-actors): for a valid non-self target, C consumes `number(1,101)` before its immortal-level test (`src/act.other.c:1688-1692`); Go's short-circuit at `pkg/game/other_utility.go:45-47` skips that draw, so its next draw comes from `ImproveSkill` (`pkg/game/combat_helpers.go:51-65`). |
| `mudlog-crash-load-entry` | `81529 Dice(1,1)` / `Number(0,20)` | Same first-draw call-site pair as the Haven/dungeon soaks: C's `skill_message` message-selector draw (`src/fight.c:1035`, called by `damage()` at `:1534-1541`) versus Go's `loudMobs` noise gate (`pkg/game/room_activity.go:423`). The report does not identify a particular mobile arm; filed with [DP-1435](https://linear.app/labz0rz/issue/DP-1435/mobile-activity-room-acts-aggr24-grins-evilly-memory-warning-and-helper-assist-arms) for the shared mobile/combat pulse boundary. Retained report: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-dp1364-corpus2-diagnostics/mudlog-crash-load-entry.txt`. |
| `combat-entry-newbie-gate`, `combat-entry-peaceful-wait`, `combat-entry-shopkeeper` | respectively `4106`, `4087`, `4081`: C `Number(1,20)` / Go log end or `Number(0,3)` | Repeated root filed as [DP-1438](https://linear.app/labz0rz/issue/DP-1438/fidelity-r3-combat-entry-refusals-skip-cs-hit-roll). C `hit()` rolls at `src/fight.c:1815` before `damage()`'s protection block (`src/fight.c:1335-1367`); Go `CmdHit` preflights `DamageRefused` in `pkg/session/combat_cmds.go:142` before `PerformInitialAttack`. |
| `combat-entry-charm-relations` | `4152 Number(1,20)` / Go log end | Same repeated entry-draw class, filed in [DP-1438](https://linear.app/labz0rz/issue/DP-1438/fidelity-r3-combat-entry-refusals-skip-cs-hit-roll): C `do_assist` calls `hit()` immediately (`src/act.offensive.c:95`); Go `cmdAssist` preflights `DamageRefused` before its initial attack (`pkg/session/cmd_combat_basic.go:111`). |
| `shop-live-inventory-control`, `shop-stack-list-live`, `show-shops-detail` | `59247 Number(1,20)` / `Number(0,18)` | Filed to the mobile-aggression/shared-combat owner. The C mobile arm calls synchronous `hit()` (`src/mobact.c:198-229`) and reaches `src/fight.c:1815`; Go reaches the wander draw at `pkg/game/mobact.go:223` after `StartCombat` in `pkg/game/mobact.go:320-325`. |
| `shop-live-inventory-filters`, `shop-live-inventory-gate` | `59093 Number(1,20)` / `Number(0,18)` | Same verified normal-aggression call path and owner as the preceding row. |
| `spec-proc-no-get`, `spec-proc-no-get-bare`, `spec-proc-no-get-palm`, `spec-proc-no-get-take` | `4118 Number(1,20)` / Go log ends | Filed to shared mobile-combat/session timing owner. The C draw is the `hit()` to-hit draw (`src/fight.c:1815`); the Go log ends after the corresponding command step, so this report does not prove a `no_get` procedure draw. |
| `carve-depth`, `sharpen-fighting-depth` | respectively `4098`, `4125`: C `Number(1,20)` / Go `Number(0,3)` | Filed to shared combat/pulse ordering. C’s first event is a `hit()` to-hit roll (`src/fight.c:1815`); Go’s first event at the corresponding probe boundary is the session command gate (`pkg/session/commands.go:670`). The scenario reports retain the exact prior warmup draws. |
| `checkload-depth` | `65101 Dice(1,5)` / `Number(0,3)` | Filed to shared combat messaging, not `checkload`: C `do_checkload` has no RNG (`src/act.wizard.c:3819-3862`); `skill_message()` selects among `fight_messages[i].number_of_attacks` at `src/fight.c:1035`, with this report’s count of five. Go’s next draw is the command gate for the following probe step. |
| `shoot-mob-miss-depth`, `shoot-pc-miss-depth` | `4136`, `4158`: C `Dice(1,2)` / Go `Number(0,3)` | Filed to shared combat messaging: C’s variable `Dice(1,2)` is the `skill_message()` message selector (`src/fight.c:1035`); Go next reaches the command gate. |
| `mudlog-attitude-loot-depth`, `spec-proc-dragon-breath-combat` | `4279`, `4172`: C `Number(0,99)` / Go log ends | Filed to shared combat damage tail: C reaches the mounted-victim dismount roll after damage (`src/fight.c:1548-1558`); Go equivalent is `pkg/combat/fight_core.go:469`. |
| `backstab-aware-trio` | `4158 Number(0,18)` / Go log ends | C reaches mobile wander (`src/mobact.c:120`) after the matching hit/damage draws; Go has no later draw entry after the same probe pulse. Filed to the shared combat/death/session boundary; the report does not establish a missing backstab draw. |
| `act-messages` | `64811 Number(0,18)` / `Number(0,3)` | C mobile wander (`src/mobact.c:120`) versus Go command-interpreter gate (`pkg/session/commands.go:670`) during setup; the scenario uses different setup actions on C and Go. Filed to setup/pulse scheduling. |
| `spec-proc-key-seller`, `spec-proc-key-seller-non-owner` | `4036 Number(0,18)` / `Number(1,6)` | C mobile wander versus Go character-stat generation (`src/class.c:392`, `pkg/game/character.go:196-205`) during setup. Filed to the setup/login boundary, not the key-seller special. |
| `house-depth`, `object-door-messages` | `4058`, `4063`: C `Number(0,3)` / Go `Number(1,6)` | C command-interpreter gate (`src/interpreter.c:889`) versus Go character-stat generation (`src/class.c:392`, `pkg/game/character.go:196-205`) during setup/warmup. Filed to setup/login. |
| `observation` | `4059 Number(1,6)` / `Number(0,3)` | Same setup/stat-generation versus command-gate boundary as the preceding group. |
| `object-doors` | `4059 Number(0,3)` / `Number(0,2)` | First divergence occurs during the warmup after matching command-gate draws. Filed to warmup/session sequencing; no object door call path is claimed from this signature alone. |
| `object-inventory` | `4061 Number(0,3)` / `Number(1,101)` | C command-interpreter gate precedes C’s `ok_pick` draw (`src/interpreter.c:889`, `src/act.movement.c:541`). Go’s `open pack` reaches the lock-pick draw without a preceding `Number(0,3)` in the captured stream (`pkg/game/act_movement.go:761`). Filed to the session-command gate owner. |
| `look-start-room`, `object-give-mob`, `object-give-nobody`, `observation-roomflags`, `prompt-infobar-status-depth`, `prompt-playing-depth`, `socials-command-depth` | C `Number(0,3)` / Go log ends | The C event is the command-interpreter gate (`src/interpreter.c:889`); each Go report ends before a corresponding draw entry. Filed to command/session transcript lifecycle; no missing gameplay draw is inferred from a log-end marker. |
| `position-gates`, `security-gates` | C log ends / Go `Number(0,3)` | Go reaches the command-interpreter gate (`pkg/session/commands.go:670`) after the C draw log has ended. Filed to session/transport capture boundary. |

### Repeated-root audit (R5b/R5c)

The protection-before-hit-draw finding repeats in the peaceful, newbie, and shopkeeper controls. I audited the related reachable Go call sites with `rg DamageRefused`: player `hit` (`pkg/session/combat_cmds.go:142`), `assist` (`pkg/session/cmd_combat_basic.go:111`), and recurring combat (`pkg/combat/engine.go:764`) all preflight the shared gate; C’s `hit()` can consume `number(1,20)` before `damage()` reaches the same protection block. This is one shared combat-entry/round class, not three isolated command defects; its initial-command draw gap is filed as DP-1438, related to the completed DP-1327/DP-1045 protection work.

The mobile-aggression mismatch repeats across the populated orc soak and five shop corpus rows. The C arm class uses `hit()` in `src/mobact.c:198-344` (normal aggression, race hate, memory, helper, and AGGR24); Go counterparts call `StartCombat` in `pkg/game/mobact.go:320-482`. DP-1435 is the active owner record and now carries this train's direct and call-site evidence. The Haven/dungeon/mudlog `Dice(1,1)` versus `Number(0,20)` rows point to combat message selection versus `loudMobs`; the reports do not prove the triggering arm, so no room self-damage or DP-1426 cause is claimed.

Proposed rulebook amendment for Zach; no governing document was edited:

> **R5b. Repeat reds indict rules.** When two independent scenarios or seeds reproduce the same verified call-path cause, stop file-level fixes in that class, audit every reachable member of the class, and report the audit plus exact proposed rule text to Zach before continuing. A repeated signature without a verified shared call path is not enough.

Incident motivating this clarification: the same attack-before-protection draw order appears in three combat-entry rows, while normal mobile aggression repeats across five shop rows and the populated orc soak. These belong to shared classes; treating each scenario as a separate local exception would have produced conflicting fixes.

The three new soaks remain transcript `FAIL` rows in the combined full census because of long-pulse Go output capture. They are not added to an expected-divergence allowlist. The combined census is complete; the post-fix full draw-parity corpus and final-tip seed-1/seed-2 soak checks remain pending.

## Gates and final census

| Gate / evidence | Result |
|---|---|
| `make fmt` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./...` | PASS |
| `go test ./pkg/game/...` | PASS |
| `golangci-lint run ./...` | PASS, 0 issues |
| `make deadcode-ratchet` | PASS, 52 baselined / 0 new |
| `make fidelity-depth` | PASS, 5,561 cases; 5,470 proven/delegated |
| `make fidelity-units` | PASS, 1,449 rows / 1,343 symbols / 1,641 claims |
| `make string-census` | PASS, 93 Go-only / 3,046 C-missing / 28 data-sourced / 522 unresolved; ratchet 98 known / 0 unreviewed; reports stale |
| `gosec -severity=high -quiet ./...` | PASS |
| `go test -race ./pkg/game/...` | PASS |
| Direct cleric and Butler controls | PASS fixed seed-1 draw reports; each revert reproduces its recorded first mismatch |
| Corpus-2 draw-parity corpus | 1,172 scenarios; 43 DRAW; 3 populated-world FAIL; 1 INFRA; NOT_CLEAN; run and diagnostics paths above |
| Post-fix full draw-parity corpus | Pending; at this update, another train's combined census holds the shared lock. Start after that run releases it. |
| Final combined census | `NOT_CLEAN`: 3,415 scheduled pairs / 1,028 deduplicated; full 1,172 scenarios: 1,169 pass, 3 fail, 0 infrastructure, 0 timeout, 0 stale, 0 unpinnable; claims 3,271/3,271 pass (`CLEAN`). The only full failures are `living-world-haven-square` (53), `living-world-dungeon-entrance` (52), and `living-world-orc-burrow-entrance` (52), the known long-pulse output-capture stop-tier findings tracked in DP-1439. Source HEAD was `39ad131f8`; the sole dirty path was this then-untracked handoff file. Retained run: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-dp1364-final-combined/` |
| Final-tip soak draw parity, seeds 1 and 2 | Pending after final rebase |
