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
| 2 | `World.DoPour` (152 lines) | `pour`, `fill` | `pour-basic`, `pour-multi`, `pour-out-refind` | Strong command coverage; extraction candidates are small, but item mutation order needs careful review. |
| 3 | `World.doPut` (112 lines) | `put` | `put-basic`, `put-all-success`, `put-fillword` | Direct byte coverage and a moderate block; preserve inventory traversal and transfer ordering. |
| 4 | `World.doGive` (120 lines) | `give`, with mob recipients delegating into mob-give logic | `give-basic`, `give-fillword`, `give-gold`, `object-give-mob`, `object-give-nobody`, `mp-give-dog-food`, `mp-give-dog-junk`, `mp-give-demon-nonsoul`, `mp-give-demon-soul`, `mp-give-janitor` | Many visible cases and good payoff; defer until the current transfer and mob-trigger call path is read end to end. Avoid files excluded by open PR #2271. |
| 5 | `doDoorcmd` (141 lines), then `World.mobGenDoor` (155) | `open`, `close`, `lock`, `unlock`, `pick`; NPC door actions | `door-basic`, `door-fillword`, `door-lock-pick-depth`, `object-door-messages`, `object-doors`, `restart-door-state`, `retreat-closed-door` | Strong coverage. Keep command subcode checks, direction lookup, and act order intact. |
| 6 | `World.doSimpleMove` (200 lines), `World.doSimpleMoveMob` (121) | Player direction movement; NPC movement/retreat | `enter-depth`, `leave-success-depth`, `retreat-closed-door`, `force-mob-move` | Visible but topology, random direction selection, and wait-state behavior make broad extraction risky. |
| 7 | `CmdShoot` (110 lines) | `shoot` | `shoot-entry-depth`, `shoot-fighting-depth`, `shoot-item-gates-depth`, `shoot-lookup-depth`, `shoot-mob-hit-depth`, `shoot-mob-lethal-depth`, `shoot-mob-miss-depth`, `shoot-no-bow-depth`, `shoot-no-target-depth`, `shoot-pc-hit-depth`, `shoot-pc-lethal-depth`, `shoot-pc-miss-depth`, `shoot-pc-window-depth`, `shoot-peaceful-depth`, `shoot-projectile-direction-order-depth`, `shoot-sentinel-depth`, `shoot-target-depth`, `shoot-unlearned-depth`, `shoot-closed-exit-depth` | Excellent depth coverage; RNG and projectile order make this a high-risk target. |
| 8 | `DoBackstab` (126 lines), `DoCircle` (106) | `backstab`, `circle` | `backstab-depth`, `backstab-aware-trio`, `combat-backstab-opener`, `backstab-skill-message-{hit,miss,death,god}-depth`; `circle-depth`, `circle-aware-depth`, `circle-behead`, `circle-skill-message-{hit,miss,death,god}-depth` | Excellent coverage and high payoff; defer unless the exact draw sequence is left untouched and locally reviewable. |
| 9 | `DoKujiKiri` (112 lines) | `rin` | `rin-depth` | Strong named coverage; branch-specific random draws make extraction require close call-order inspection. |
| 10 | `DoScrounge` (105 lines) | `scrounge` | `scrounge-default-depth`, `scrounge-desert-depth`, `scrounge-failure-depth`, `scrounge-field-depth`, `scrounge-forest-depth`, `scrounge-hills-depth`, `scrounge-mountain-depth`, `scrounge-mounted-depth`, `scrounge-no-skill-depth`, `scrounge-water-depth` | Excellent environment matrix; preserve all roll gates and draw order. |
| 11 | `World.HandleDeath`, `handleMobDeath`, `handlePlayerDeath`; `World.AwardMobKillXP` | Combat death and XP award call paths | `combat-death`, `death-autogold-kick`, `death-autoloot-audience`, `death-corpse-decay-{npc,pc}`, `death-corpse-loot`, `death-followers-depth`, `death-group-*`, `death-neutral-room-save`, `death-ungrouped-*`, `kill-body-second-death-depth`, `flee-death-exit` | Wide coverage; lifecycle, audience, and state order make these stop-tier for this pass absent a very local extraction. |
| 12 | `World.PointUpdate`, `World.decayObject`, `World.CheckIdling` | Periodic world pulse, object decay, idle handling | `tick-*`, `object-decay-*`, `idle-*` scenario families | High line reduction potential, but timing and random gates are hard to prove from a single command census. |
| 13 | `CombatEngine.processCombatPair`, `performOneHit`, `takeDamageFrom`, `getMinusDam` | Shared combat loop and damage calculations | `combat-*`, `fightmsg-*`, and skill-message death/hit/miss scenario families | Very visible but directly touches RNG and arithmetic; defer as unsure. |
| 14 | `World.executeZoneResetLocked`, `World.NewWorld`, `NewMob` | Boot and zone reset | `zone-reset-*`, `entry-bootstrap-*`, plus every scenario that boots a world | Broad exposure with substantial initialization and draw-order risk. |
| 15 | `World.mobileActivityForMob`, `MpGreetActor`, `MpSound`, `specCleric`, `specStableboy`, `specRoach`, other nested spec-proc functions | NPC autonomous and script-triggered paths | `mob-sound-*`, `spec-proc-*`, `mp-give-*`, `force-mob-move` | Several behaviors are well covered; each callee needs an individual call-path audit before selection. |
| 16 | `sendSkillResult` (446 lines), `CmdListSkills` (111) | All player skill outcomes; `skills`/skill-list commands | `*-skill-message-*`, `*-depth`, `abilities-depth`, `skill-list-*` | High payoff, shared strings and audience order make a large extraction hard to review. |
| 17 | `performAct` (121 lines), `World.huntVictim` (106), `RaceHelpEntries` (105), `Dream` (132), `World.doGenTog` (113) | Shared `act()` output; `hunt`; race help; dream; toggles | `act-messages`, `act-informative-sweep`, `act-obj-sweep`, `act-arena-broadcast`; `hunt-*`; `help-*`; `dream-*`; toggle-specific scenarios | Reachability varies by branch; choose only after the exact caller and recipient paths are mapped. |
| 18 | Remaining long/deep functions in `target_inventory.tsv` | Shared setup, lifecycle, world helpers, validation | Mapped by their direct callers; several have only unit or boot coverage | Lowest priority until a command scenario and call path are confirmed. |

## Other metric targets

`target_inventory.tsv` gives file and line for every underscore-bearing
identifier and empty interface, as well as every long/deep function. For an
identifier or type occurrence, census reachability belongs to its containing
function and its callers; it is not an independent command entry. The specific
containing-function/caller path must be traced before editing. Do not rename
package-visible exported API. Do not narrow an empty interface to a concrete
type based only on an apparent current caller: the concrete values and external
callers must be checked first.

## Skipped: unsure

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
- Each underscore-bearing declaration and empty-interface occurrence in
  `target_inventory.tsv` that is not inside a selected target: its true callers,
  concrete runtime values, or command surface are not yet established by this
  census-oriented pass. Keep these in the measured inventory, unmodified, and
  rank them after direct command handlers.

The open PR exclusion list is in `open-pr-files.md`; none of its package files
is selected for the initial target.
