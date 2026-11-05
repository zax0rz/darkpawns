# Refactor blueprint

Base: `dce50f5bf398ec858a5f3bf92e8b4b61b2664357` (`origin/main`, fetched 2026-11-04).
The source inventory is `target_inventory.tsv`: 48 functions over 100 lines,
22 functions nested at least five control-flow levels, 126 underscore-bearing
declarations, and 88 empty-interface occurrences. These counts match
`baseline.tsv`. The non-test `dupl` findings are four overlapping spans in the
syllable tables in `pkg/game/act_comm.go`; they are language data with different
replacement values, not identical executable blocks, so they are not a safe
collapse target. Lint also reports test-only duplicates, which are out of scope.

Order below is by direct scenario visibility first, then payoff. The first row
is the initial target. Names of scenario files omit `.txt` in census commands.

| Rank | Target | Entry commands / call path | Census scenarios | Payoff and risk |
|---:|---|---|---|---|
| 1 | `World.DoDrink` (121 lines) | `drink`, `sip`, plus `taste` delegating to sip | `drink-basic`, `drink-eat-lookup`, `vampire-consume-day`, `vampire-consume-sunset`, `vampire-consume-dark` | Directly covered output and vampire branches; can extract the sustenance decision without moving any statement or draw. C path read at `src/act.item.c:953-1030`. |
| 2 | `World.DoPour` (152 lines) | `pour`, `fill` | `pour-basic`, `pour-multi`, `pour-out-refind` | C: `src/act.item.c:1238-1263`. Strong command coverage; the pour-out extraction preserves the object mutation order. |
| 3 | `World.doPut` (112 lines) | `put` | `put-basic`, `put-all-success`, `put-fillword` | C: `src/act.item.c:127-148`. Direct byte coverage; preserve the inventory snapshot and transfer order. |
| 4 | `World.doGive` (120 lines) | `give`, with mob recipients delegating into mob-give logic | `give-basic`, `give-fillword`, `give-gold`, `object-give-mob`, `object-give-nobody`, `mp-give-dog-food`, `mp-give-dog-junk`, `mp-give-demon-nonsoul`, `mp-give-demon-soul`, `mp-give-janitor`, `mp-bribe-gremlin-tunnel`, `mp-bribe-cityguard-sleeps`, `mp-bribe-room-depth`, `mp-bribe-cityguard-refusal`, `mp-bribe-cell-eject`, `mp-bribe-gremlin-eject`, `lua-ongive-highpriest-alignment`, `lua-ongive-healer-runestone`, `lua-ongive-blacksmith-tooth`, `lua-ongive-healer-rubbish` | C: `src/act.item.c:767-830`, `700-730`, `733-763`; read object, coin, mob, and Lua-trigger paths end to end. Avoid files excluded by open PR #2271. |
| 5 | `doDoorcmd` (141 lines), then `World.mobGenDoor` (155) | `open`, `close`, `lock`, `unlock`, `pick`; NPC door actions | `door-basic`, `door-fillword`, `door-lock-pick-depth`, `object-door-messages`, `object-doors`, `restart-door-state`; `retreat-closed-door` reaches movement handling | C: `src/act.movement.c:477-533`. Keep command subcode checks, direction lookup, and act order intact; confirm the exact NPC door caller before editing `mobGenDoor`. |
| 6 | `World.doSimpleMove` (200 lines), `World.doSimpleMoveMob` (121) | Player direction movement; NPC movement/retreat | `enter-depth`, `leave-success-depth`, `retreat-closed-door`, `force-mob-move` | Visible but topology, random direction selection, and wait-state behavior make broad extraction risky. |
| 7 | `CmdShoot` (110 lines) | `shoot` | `shoot-entry-depth`, `shoot-fighting-depth`, `shoot-item-gates-depth`, `shoot-lookup-depth`, `shoot-mob-hit-depth`, `shoot-mob-lethal-depth`, `shoot-mob-miss-depth`, `shoot-no-bow-depth`, `shoot-no-target-depth`, `shoot-pc-hit-depth`, `shoot-pc-lethal-depth`, `shoot-pc-miss-depth`, `shoot-pc-window-depth`, `shoot-peaceful-depth`, `shoot-projectile-direction-order-depth`, `shoot-sentinel-depth`, `shoot-target-depth`, `shoot-unlearned-depth`, `shoot-closed-exit-depth` | Excellent depth coverage; RNG and projectile order make this a high-risk target. |
| 8 | `DoBackstab` (126 lines), `DoCircle` (106) | `backstab`, `circle` | `backstab-depth`, `backstab-aware-trio`, `combat-backstab-opener`, `backstab-skill-message-{hit,miss,death,god}-depth`; `circle-depth`, `circle-aware-depth`, `circle-behead`, `circle-skill-message-{hit,miss,death,god}-depth` | Excellent coverage and high payoff; defer unless the exact draw sequence is left untouched and locally reviewable. |
| 9 | `DoKujiKiri` (112 lines) | `rin` | `rin-depth` | Strong named coverage; branch-specific random draws make extraction require close call-order inspection. |
| 10 | `DoScrounge` (105 lines) | `scrounge` | `scrounge-default-depth`, `scrounge-desert-depth`, `scrounge-failure-depth`, `scrounge-field-depth`, `scrounge-forest-depth`, `scrounge-hills-depth`, `scrounge-mountain-depth`, `scrounge-mounted-depth`, `scrounge-no-skill-depth`, `scrounge-water-depth` | Excellent environment matrix; preserve all roll gates and draw order. |
| 11 | `World.HandleDeath`, `handleMobDeath`, `handlePlayerDeath`; `World.AwardMobKillXP` | Combat death and XP award call paths | `combat-death`, `death-autogold-kick`, `death-autoloot-audience`, `death-corpse-decay-{npc,pc}`, `death-corpse-loot`, `death-followers-depth`, `death-group-*`, `death-neutral-room-save`, `death-ungrouped-*`, `kill-body-second-death-depth`, `flee-death-exit` | Wide coverage; lifecycle, audience, and state order make these stop-tier for this pass absent a very local extraction. |
| 12 | `World.PointUpdate`, `World.decayObject`, `World.CheckIdling` | Periodic world pulse, object decay, idle handling | `point-depth`, `dream-point-update-wake-depth`, `point-update-global-object-timers`, `lifecycle-idle-command-reset`, `lifecycle-idle-force-rent-*`, `lifecycle-idle-rent-*`, `lifecycle-idle-terminal-weather*`, `lifecycle-idle-void`, `death-corpse-decay-{npc,pc}` | High line reduction potential, but timing and random gates are hard to prove from a single command census. |
| 13 | `CombatEngine.processCombatPair`, `performOneHit`, `takeDamageFrom`, `getMinusDam` | Shared combat loop and damage calculations | `combat-*`, `fightmsg-*`, and skill-message death/hit/miss scenario families | Very visible but directly touches RNG and arithmetic; defer as unsure. |
| 14 | `World.executeZoneResetLocked`, `World.NewWorld`, `NewMob` | Boot and zone reset | `zone-reset-*`, `entry-bootstrap-*`, plus every scenario that boots a world | Broad exposure with substantial initialization and draw-order risk. |
| 15 | `World.mobileActivityForMob`, `MpGreetActor`, `specCleric`, `specStableboy`, `specRoach`, other nested spec-proc functions | NPC autonomous and script-triggered paths | `mobact-aggr24-weaker-mobile-grin`, `mob-wander-flying`, `mob-wander-waterwalk`, `spec-proc-*`, `force-mob-move` | Several behaviors are well covered; each callee needs an individual call-path audit before selection. `MpSound` is deferred because its `pkg/game/mobprogs.go` file is touched by open PR #2271. |
| 16 | `sendSkillResult` (446 lines), `CmdListSkills` (111) | All player skill outcomes; `skills`/skill-list commands | `*-skill-message-*`, `*-depth`, `abilities-depth`, `skill-list-*` | High payoff, shared strings and audience order make a large extraction hard to review. |
| 17 | `performAct` (121 lines), `World.huntVictim` (106), `RaceHelpEntries` (105), `Dream` (132), `World.doGenTog` (113) | Shared `act()` output; `hunt`; race help; dream; toggles | `act-messages`, `act-informative-sweep`, `act-obj-sweep`, `act-arena-broadcast`; `sethunt-depth`, `mob-hunt-items`; `help-depth`, `help-immortal-depth`, `help-system`; `dream-depth`, `daydream-depth`; `toggle-cluster`, `god-toggles` (`nosummon` is outside the cluster because its wait state blocks the next probe) | C for toggles: `src/act.other.c:1142-1278`. Reachability varies by branch; map callers before selecting the remaining functions. |
| 18 | Remaining long/deep functions in `target_inventory.tsv` | Shared setup, lifecycle, world helpers, validation | Mapped by their direct callers; several have only unit or boot coverage | Lowest priority until a command scenario and call path are confirmed. |

## Lower-priority long/deep reachability notes

| Target | Command or caller | Known scenario coverage / decision |
|---|---|---|
| `World.wanderMobWithDoor`, `World.mobGenDoor`, `World.doSimpleMoveMob` | NPC autonomous movement and scripted/forced movement | `mob-wander-flying`, `mob-wander-waterwalk`, `mobact-aggr24-weaker-mobile-grin`, `force-mob-move`, `spec-proc-movement`; defer the door-specific NPC branch until a fixture is confirmed to reach it. |
| `World.ExtractPendingPlayers` | Pending-player/session cleanup, no direct command | No named command scenario confirmed; skip until a caller-specific lifecycle scenario is identified. |
| `World.WireCombatCallbacks` | Server startup callback wiring | Combat entry and death scenarios exercise downstream callbacks, but not the wiring function as a command body; skip as shared setup. |
| `createMoneyDesc` | Death loot/autogold formatting | `death-autogold-kick`, `death-group-ungrouped-leader-gold`, `death-ungrouped-follower-reward`; owned by the death/XP cluster at rank 11. |
| `Player.AdvanceLevel` | `advance` and new-character level initialization | `advance-depth`, `entry-bootstrap-cleric`, `entry-bootstrap-mage`, `entry-bootstrap-ninja`, `entry-bootstrap-psionic`, `entry-bootstrap-thief`, `entry-bootstrap-warrior`; defer because level-up calls include conditional RNG draws. |
| `World.HcontrolBuildHouse` | `hcontrol` | `hcontrol-depth`, `hcontrol-capacity-depth`, `house-depth-v2`, `house-entry-gates-depth`, `house-contents-not-saved`; C: `src/house.c:386-469`. Selected as target 7: the ordered direction scan was extracted without moving validation, state, or output. |
| `ObjectLocation.Validate` | Object invariant validation, not a player command | No oracle scenario directly reaches this validator; unit-only coverage, skip. |
| `World.attachObjectLocked` | Shared object transfers used by get, drop, put, give, eat/drink, and pour paths | `get-*`, `drop-*`, `put-*`, `give-*`, `drink-*`, `pour-*`; the specific lock helper has no player entry of its own, so leave it to a separately proven transfer target. |
| `NewMob`, `World.NewWorld`, `World.AddPlayer` | World boot, fixture mob loading, and player entry | `entry-bootstrap-cleric`, `entry-bootstrap-mage`, `entry-bootstrap-ninja`, `entry-bootstrap-psionic`, `entry-bootstrap-thief`, `entry-bootstrap-warrior`, `combat-zone-equipped-mobile`, and `zone-reset-remove-object`; broad setup call paths, not a single command. |
| `World.doGenTog` | Shared toggle commands (`brief`, `compact`, channel and immortal preferences) | `toggle-cluster` covers the mortal preference cluster; `god-toggles` covers quest and immortal toggles. `nosummon` remains outside because its wait state blocks subsequent commands in the current probe. |
| `playerToSaveData` | Logout/save paths | Lifecycle scenarios reach persistence boundaries, but changing save representation is forbidden; skip. |
| `shopEvaluateExpression` | Shop pricing/trade evaluation | `shop-trade-expression` and `shop-trade-expression-00` through `shop-trade-expression-09`; shop-specific, lower priority. |
| `DoSpike` | `spike` | `spike-depth`; RNG and damage order make it lower priority. |
| `specCleric`, `specStableboy`, `specRoach` | NPC special commands / periodic behavior | `spec-proc-cleric`, `spec-proc-stableboy`, `spec-proc-stableboy-success`, `spec-proc-roach`; use the owning spec-proc depth rows before selecting. |
| `specSummoner`, `specCouch`, `specCastleGuardEast` | NPC special commands | No exact live scenario was confirmed for these three handlers; skip pending a call-path check. Similar-looking summoning, couch, and other castle-guard scenarios are not proof of these exact branches. |
| `World.MpSound` | NPC default `mp_sound` tail | `mob-sound-mime-emote`, `mob-sound-zealot-say`; skip because open PR #2271 edits `pkg/game/mobprogs.go` and both scenarios. |
| `weatherChangeLocked` | Weather pulse and weather spells | `weather-month-rollover`, `weather-month16-rollover`, `weather-sky-transitions`, `weather-sun-cycle`, `weather-year-rollover`, plus `spell-control-weather-invalid-depth`; time and RNG order make it lower priority. |
| `Whod.BuildWhoList` | `who` | `who-color-levels-depth`, `who-no-who-room`, `info-who-pair`, `mudlog-whod-depth`; visible but not above the directly selected handlers. |
| `CombatEngine.stopCombatLocked` | Combat teardown after flee, death, or target removal | `combat-death`, `flee-death-exit`, `death-followers-depth`, and the combat outcome scenario families; shared state order makes it lower priority. |

## Execution outcome

Targets 1–5 above and the separately audited lower-risk World.doGenTog and
World.HcontrolBuildHouse paths were the seven changes attempted. Each passed
its local gates and clean targeted real census; every one-byte probe was
caught. Exact scenarios, run paths, and tested isolated SHAs are recorded in
telemetry.tsv. The branch keeps each target as its own commit. The remaining
ranked paths are skipped under the reasons below; no behavior fix was folded
into a refactor.

## Other metric targets

`target_inventory.tsv` gives file and line for every underscore-bearing
identifier and empty interface, as well as every long/deep function. For an
identifier or type occurrence, census reachability belongs to its containing
function and its callers; it is not an independent command entry. The specific
containing-function/caller path must be traced before editing. Do not rename
package-visible exported API. Do not narrow an empty interface to a concrete
type based only on an apparent current caller: the concrete values and external
callers must be checked first. The snake-identifier inventory is the metric's
candidate superset; exported identifiers are excluded from rename targets.

## Skipped: unsure

- Rank 6, `World.doSimpleMove` and `World.doSimpleMoveMob`: movement topology,
  follower order, wait state, and random direction choices make an extraction
  hard to prove from the available movement scenarios.
- Rank 7, `CmdShoot`: projectile selection and combat outcomes share RNG and
  state transitions across many branch-specific scenarios.
- Ranks 8–10, `DoBackstab`, `DoCircle`, `DoKujiKiri`, and `DoScrounge`: their
  skill, damage, environment, and message branches depend on conditional draws.
- Ranks 11–14, death/XP handling, periodic update/decay/idling, combat damage,
  and boot/zone-reset logic: they can reorder persistence, recipients, timing,
  arithmetic, or initialization draws.
- Rank 15, autonomous mob activity and the remaining special procedures: the
  exact call path and per-mob RNG gates were not fully established. `MpSound`
  also remains excluded by open PR #2271.
- Rank 16, `sendSkillResult` and `CmdListSkills`: shared outcome audience and
  string order make `sendSkillResult` broad. `CmdListSkills` is not
  registered in the current command table; the live command is C's `practice`
  path, so it is not a direct refactor target.
- Rank 17 leftovers, `performAct`, `huntVictim`, `Dream`, and the static
  `RaceHelpEntries` catalog: shared act bytes, random hunter/dream branches, or
  static source text offer no small safe logic extraction in this pass. The
  `doGenTog` flag branch was selected separately as target 6.
- Lower-priority `shopEvaluateExpression`: its operator stack, precedence, and
  short-circuit pop order mirror C's parser; leave it for a branch-level audit.
- `World.PointUpdate`, `World.executeZoneResetLocked`, `World.NewWorld`, and
  combat damage/attack functions: timing, hidden state, or conditional RNG draws
  can change even when visible command transcripts stay green.
- `World.HandleDeath`, `handleMobDeath`, `handlePlayerDeath`, and
  `World.AwardMobKillXP`: extraction can reorder recipients, persistence, group
  state, or teardown operations.
- `DoBackstab`, `DoCircle`, `DoKujiKiri`, `DoScrounge`, `CmdShoot`, and
  `sendSkillResult`: they contain RNG or shared outcome messaging; a safe split
  needs more branch-level proof than the current first target.
- The four `dupl` blocks in syllable tables: the repeated shapes carry distinct
  language mappings, so their equivalence is uncertain and collapsing them may
  change player-visible text.
- Each remaining unexported underscore-bearing identifier and empty-interface
  occurrence in `target_inventory.tsv`: its containing-function callers,
  concrete runtime values, or command surface are not yet established by this
  pass. Skip each until its exact owner and consumers are traced; exported
  identifiers are never rename targets.

The open PR exclusion list is in `open-pr-files.md`; none of its package files
is selected for the initial target.
