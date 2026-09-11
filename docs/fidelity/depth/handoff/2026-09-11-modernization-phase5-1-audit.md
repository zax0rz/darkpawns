# Phase 5.1 audit — Elements teleport spine

Date: 2026-09-11
Starting head: `origin/main` at `e26ee2495` (merged PR #1437)
Existing Phase 5.1 merge: PR #1399, `ecafde530` (source branch commit
`b77d8b54f`)

This is the required pre-edit matrix for the seven active element special
procedures. The current main tree already contains the production refactor;
the new work at this milestone is an evidence and tracking audit, not a
duplicate extraction.

## Common call path

The C room procedures are reached by `special()` from the current-room branch
at `src/interpreter.c:1407-1416`, and by the movement pre-check at
`src/act.movement.c:112-116`. The C mobile procedures are reached by the
current-room mobile loop at `src/interpreter.c:1451-1456` and by autonomous
`MOB_SPEC` dispatch at `src/mobact.c:82-92`, which passes `cmd=0`. Registrations
are `src/spec_assign.c:195-196` for the two mobs and `src/spec_assign.c:622-632`
for the five room procedures. The corresponding Go registrations are
`pkg/game/spec_procs3.go:82-88` and assignments
`pkg/game/spec_assign.go:378-388`.

## Per-proc matrix

| C function and registration | Go implementation | Refactor branch / disposition | Exact manifest cases; proving vehicles and seeds | Bytes, audiences, state, and RNG | Gaps and limits |
|---|---|---|---|---|---|
| `SPECIAL(elements_master_column)`, `src/spec_procs3.c:936-1002`; `ASSIGNROOM(1315)` at `src/spec_assign.c:622` | `pkg/game/spec_procs3.go:1416-1475`; shared helper at `:1349-1414` | **Refactored in #1399.** Shared player snapshot/order, direct message, departure `Act`, checked `PlayerTransfer`, destination look, and arrival `Act`. Talisman scan, stale `has_object[]`, message selection, and five destinations remain local. | `room.elements-master-column-entry`, `-mappings`, `-stale-state`, `-no-talisman`, `-audience`, `-four-talisman`, `-relocation`, `-inventory-vnums`; unit symbols `TestSpecElementsMasterColumn_RejectsNilActor`, `TestSpecElementsMasterColumn_EntryAndTalismanDestinations`, `TestSpecElementsMasterColumn_PreservesCStaleCarryStateAndAudience`; oracle `spec-proc-elements-master-column-none`, `-all`, `-stale-state` at seeds **1,2,3** | `ToChar` no/partial/all-talisman text; `ToNotVict` departure and arrival. Destination `1320/1331/1342/1353/1372`; `look_at_room` stays between transfer and arrival. Per-player order is state-observable because C front-inserts and preserves stale carry state. No C RNG call in the changed branch. | The oracle matrix has seeds 1,2,3 only; the proc has no RNG. Transfer-error logging/continuation is retained but not forced by the vehicle. Synthetic empty-player fixtures do not certify unregistered `elemental_room` (correctly excluded elsewhere). |
| `SPECIAL(elements_platforms)`, `src/spec_procs3.c:1004-1024`; `ASSIGNROOM(1326/1337/1348/1359)` at `src/spec_assign.c:623-626` | `pkg/game/spec_procs3.go:1477-1494`; shared helper at `:1349-1414` | **Refactored in #1399.** Fixed destination and messages are proc-local; the player transfer sequence is shared. No look is invented. | `room.elements-platforms-entry`, `-direct-message`, `-audience`, `-relocation`; unit symbols `TestSpecElementsPlatforms_RejectsNilActor`, `TestSpecElementsPlatforms_EntryAudienceAndRelocation`; oracle `spec-proc-elements-platforms` at seeds **1,2,3,5,8** | Exact direct dizzy text to actor; departure and arrival `ToNotVict` Acts to peers; sequential checked transfer to `1314`; no destination look. No C RNG call. | No failure-injection vehicle for `PlayerTransfer`; no additional branch is proposed because all reachable procedure branches are covered by the manifest/unit boundary. |
| `SPECIAL(elements_load_cylinders)`, `src/spec_procs3.c:1026-1135`; `ASSIGNROOM(1360/1364/1380/1384)` at `src/spec_assign.c:627-630` | `pkg/game/spec_procs3.go:1496-1568` | **Already complete; no teleport spine applies.** Keep the get/drop gates, current-room cylinder cleanup, talisman/cylinder mapping, room-wide messages, and object insertion local. | `room.elements-load-cylinders-entry`, `-matching`, `-existing-cylinder`, `-audience`, `-cleanup`, `-room-list`; unit symbols `TestSpecElementsLoadCylinders_RejectsWrongTalismanAndExistingCylinder`, `TestSpecElementsLoadCylinders_MapsAllRegisteredPillars`, `TestSpecElementsLoadCylinders_GetRemovesOnlyCurrentRoomCylinder`, `TestRoomObjectLinesLeaveObjectFlagsUncolored`; oracle `spec-proc-elements-load-cylinders` at seeds **1,2,3,5,8** | `do_get`/`do_drop` output, room-wide cylinder create/sink lines, room object state and prepend ordering. No player transfer, destination look, arrival/departure sequence, or C RNG draw. | Refactoring this proc under a teleport abstraction would invent shared behavior and broaden into object-state work. No such change is in scope. |
| `SPECIAL(elements_galeru_column)`, `src/spec_procs3.c:1137-1182`; `ASSIGNROOM(1372)` at `src/spec_assign.c:631` | `pkg/game/spec_procs3.go:1570-1618`; shared helper at `:1349-1414` | **Refactored in #1399.** Four-room prerequisite scan and NPC exclusion remain local; the player-only beam transfer spine is shared. | `room.elements-galeru-column-entry`, `-nil-actor`, `-complete`, `-audience`, `-relocation`, `-destination-look`; unit symbols `TestSpecElementsGaleruColumn_RequiresAllExactTalismans`, `TestSpecElementsGaleruColumn_RejectsNilActor`, `TestSpecElementsGaleruColumn_AudienceLookAndRelocation`; oracle `spec-proc-elements-galeru-column` at seeds **1,2,3,5,8** | Raw direct beam buffer includes `\r\n\n` and goes only to actor; departure/arrival `ToNotVict`; destination `1389`; look follows each transfer and precedes arrival; NPCs stay in origin. No C RNG call. | No failure-injection vehicle; exact raw framing and NPC exclusion are explicit local exceptions. |
| `SPECIAL(elements_galeru_alive)`, `src/spec_procs3.c:1184-1215`; `ASSIGNROOM(1394)` at `src/spec_assign.c:632` | `pkg/game/spec_procs3.go:1620-1699`; shared ordering helper at `:1363-1385`, mixed transfer local at `:1633-1673` | **Refactored with retained exception in #1399.** Reuses only the player ordering helper. Do not force it through the player-only helper: C scans all characters, moves NPCs, gates on exact mob VNum `1315`, and uses Galeru-specific look framing. | `room.elements-galeru-alive-entry`, `-live-galeru`, `-fallthrough`, `-dead-branch`, `-audience`, `-relocation`, `-destination-look`; unit symbols `TestSpecElementsGaleruAlive_EntryAndExactMobGate`, `TestSpecElementsGaleruAlive_UsesExactVNumAndMovesNPCs`; oracle `spec-proc-elements-galeru-alive` and `-dead` at seeds **1,2,3,5,8** | Commandless/live-Galeru early returns; player direct raw buffer, NPC no direct descriptor; `ToNotVict` departure/arrival; all-character transfer to `1395`; player look after transfer with preserved framing. No C RNG call. | Mixed player/NPC ordering and special look framing remain local by design. The vehicle proves the registered room-command path, not every possible internal list mutation/error path. |
| `SPECIAL(elements_minion)`, `src/spec_procs3.c:1217-1240`; `ASSIGNMOB(1313)` at `src/spec_assign.c:195` | `pkg/game/spec_procs3.go:1701-1729` | **Already complete; no teleport spine applies.** Preserve the six ordered visible-keyword passes, room Act, extraction, and delegated cylinder cleanup. | `mob.elements-minion-entry`, `-command-fallthrough`, `-pulse-dispatch`, `-keyword-predicate`, `-destroy-audience`, `-extraction`; unit symbols `TestSpecElementsMinion_UsesOrderedVisibleKeywordPasses`, `TestSpecElementsMinion_UsesKeywordsNotVnumsAndSkipsInvisibleObjects`, `TestFindMobInRoomUsesAuthoredKeywords`; oracle `spec-proc-elements-minion` at seeds **1,2,3,5,8** | Room-wide destruction Act to actor and peer; each pass extracts one visible match; command path and autonomous `cmd=0` both return `FALSE`; no teleport and no C RNG draw. | The shared behavior here is `elements_remove_cylinders`, an object cleanup helper, not player teleport. Reopening it would be a separate object/fidelity change. |
| `SPECIAL(elements_guardian)`, `src/spec_procs3.c:1242-1287`; `ASSIGNMOB(1314)` at `src/spec_assign.c:196` | `pkg/game/spec_procs3.go:1731-1842` | **Already complete; no teleport spine applies.** Keep commandless gate, room order, eligibility, self-damage, exact Acts, and synchronous player hit boundary local. | `mob.elements-guardian-entry`, `-commandless`, `-solo-branch`, `-pair-branch`, `-pair-audience`, `-hit-boundary`; unit symbols `TestSpecElementsGuardian_CommandlessAndNilEntryGates`, `TestSpecElementsGuardian_PairUsesRoomOrderAudienceAndHit`, `TestSpecElementsGuardian_SoloUsesSelfDamageAndActPronouns`; oracle `spec-proc-elements-guardian` at seeds **1,2,3,5,8** | Solo branch draws C `number(10,50)` once before self-damage; pair branch emits three audiences then calls `hit`; exact combat/state ordering is local. No teleport. | Guardian RNG/combat proof is outside a teleport refactor. Any extraction would risk R3 draw/order and must be a separate, depth-backed milestone. |

## Matrix conclusion

The seven-proc family is covered honestly by four dispositions: three full
player-only spines refactored, one mixed-character proc sharing only its
proven ordering helper, and three non-teleport procs left local. No coverage
gap directly required a new fixture: all changed branches already have named
unit or oracle rows, and the affected oracle vehicles were green on the
starting head. No new production refactor is justified on this branch.

The unassigned `SPECIAL(elemental_room)` at `src/spec_procs.c:2021-2068` is
not one of the seven active registrations; its excluded manifest row
`mob.elemental-room-unassigned` records the R2/R4/R5e boundary. It is not a
valid teleport vehicle or a Phase 5.1 extraction target.

## Validation checkpoint

The final affected-family rerun covered all 44 recorded scenario/seed pairs:
the three master-column vehicles at seeds `1,2,3`, and the platforms,
cylinders, Galeru-column, both Galeru-alive vehicles, minion, and guardian
vehicles at seeds `1,2,3,5,8`. Every run reported `no normalized divergence`;
the proving blocks were inspected with `--show-oracle`. Durable logs are in
`/home/zach/dp-phase51-final-20260911/`. The initial parallel attempt is
preserved in `/home/zach/dp-phase51-final-20260911-path-misconfig/` and is
infrastructure-only (`Address already in use`).

Full `make oracle-regression` remains the final validation gate before PR
creation; no new fixture or production change is pending.

## Final validation and disposition

The full census was run from candidate head `e6c7e7312` on branch
`glm/modernize-elements-teleport`, with
`PATH=/usr/local/go/bin:$PATH`, `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`,
and `ORACLE_REGRESSION_JOBS=4`. The durable log is
`/home/zach/dp-phase51-oracle-regression-20260911.log`. Its final tally is:

| census result | count |
|---|---:|
| scenarios | 938 |
| passed | 928 |
| expected (ledger-backed, pinned shape) | 9 |
| unpinnable (human-cleared baseline) | 1 |
| stale | 0 |
| failed | 0 |
| infra | 0 |
| timed out | 0 |

The sole unpinnable result is the established `accuse-noarg-depth` baseline;
the harness exit 2 is solely the consequence of that human-clearance rule.
The bounded retry policy recovered `checkload-depth`, `flip-depth`,
`get-mounted-block`, `handbook-immortal-depth`, `hush-depth`, `look-basic`,
`tackle-depth`, `think-depth`, and `wizard-valid-reports-depth`; none remained
in the final infra count.

All 44 affected scenario/seed pairs were rerun with `--show-oracle` from the
same production candidate: master-column none/all/stale at seeds `1,2,3`,
and platforms, load cylinders, Galeru column, Galeru alive, Galeru alive
dead, minion, and guardian at seeds `1,2,3,5,8`. Every pair reported `no
normalized divergence`, and the proving blocks were inspected. The easy
gates also passed: `gofumpt`, `go build ./...`, `go vet ./...`,
`go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`,
`make fidelity-depth`, and `make expected-divergences-check`. The current
depth snapshot is `4798 total, 4679 proven/delegated, 68 blocked, 51
excluded`.

### Per-proc next actions

The three non-teleport procedures are explicit Phase 5.1 deferrals, not
missing work: `elements_load_cylinders` remains an object-state slice,
`elements_minion` remains an object-cleanup/pulse slice, and
`elements_guardian` remains a combat/RNG slice. No Phase 5.1 action remains
for them. If any is revisited, the smallest next action is a separate,
manifest-backed extraction using its listed rows, with fresh proof for object
ordering or combat draw order; do not route it through the player teleport
helper. The mixed NPC/player path in `elements_galeru_alive` likewise remains
local except for its proven player ordering helper.

This handoff stops at Phase 5.1. It does not start Phase 5.2, Phase 6, or
unrelated entry-depth work.

## Review handoff

Review PR: #1441
(`https://github.com/zax0rz/darkpawns/pull/1441`), intentionally unmerged.
Final branch/head identifier: `glm/modernize-elements-teleport`.
The production candidate exercised by the validation runs is the unchanged
`e6c7e7312` parent; the subsequent commits add only evidence text.
