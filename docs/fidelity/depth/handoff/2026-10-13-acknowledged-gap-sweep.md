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
