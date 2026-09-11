# Modernization Phase 5.2 audit — 2026-09-11

## Recommendation

Blocked for a whole-Phase 5.2 verification claim. The active registered
tattoo, castle-guard, fighter, and paladin behavior is verified within the
named proof boundaries below. The two undead-knight procedures cannot be
called verified: their C procedures are declared but unassigned, their Go
names are registered only in `SpecRegistry` and have no `MobSpecAssign`
vehicle, and the extracted Go body does not match the C source if it is
called directly. This is an audit blocker, not a claim that the dead path is
currently player-reachable.

No production code, C oracle source, normalizer, fixture, manifest, pin, or
tested input was changed by this audit.

## Tested state and history

- Audit branch: `glm/audit-phase5-2` in an isolated worktree.
- Audit PR: [#1442](https://github.com/zax0rz/darkpawns/pull/1442), left
  unmerged for review.
- Tested commit: `18b5a911ce75702e5ab71df4aa46e87baa00c076` (current
  `origin/main` on 2026-09-11), which includes merged PR #1441.
- Audited merge: PR #1400, `ad3a9e7db70169fd02e4be51183b6f33ab97976f`, with
  first parent `ecafde5302596e29d525617513bbd033d2c78727`.
- `git diff ad3a9e7db..origin/main -- pkg/game/spec_procs.go
  pkg/game/spec_procs2.go` is empty, and the path log has no later commits.
  The complete #1400 diff was three files: the two production files and the
  historical handoff.
- The primary checkout's unrelated edit to
  `docs/specs/tui-setup-wizard.md` was left untouched.

### Measured #1400 delta

The original refactor diff, measured with `git diff --numstat` and verified by
physical line totals, is:

| scope | before | after | diff |
|---|---:|---:|---:|
| production `pkg/game/spec_procs.go` | 1,299 | 1,292 | +34 / −41, net −7 |
| production `pkg/game/spec_procs2.go` | 2,519 | 2,403 | +77 / −193, net −116 |
| production total | 3,818 | 3,695 | **+111 / −234, net −123** |
| tests and oracle scenarios | unchanged | unchanged | **0** |
| #1400 handoff documentation | absent | 46 lines | **+46** |

The physical production totals also changed by 9 and 107 nonblank lines
respectively (1206→1197 and 2327→2220). These figures describe the original
#1400 refactor. There are no later changes to those production paths on the
current `origin/main`; this audit adds documentation only.

## Per-family disposition matrix

The C call path for command specials is `special()`'s room/equipment/inventory/
mobile order in `src/interpreter.c:1407-1465`. Autonomous specials enter from
`src/mobact.c:68-93`; combat-time specials enter after the ordinary attack
loop in `src/fight.c:1898-2032`. Go command dispatch reaches
`pkg/game/act_movement.go:469-485` for movement blockers, autonomous dispatch
is `pkg/game/mobact.go:170-197`, and combat dispatch is wired by
`pkg/session/manager.go:721-736` into `pkg/combat/engine.go:598-606`.

### Tattoo ×4

| proc | C / Go sites and registration | #1400 branches and retained exceptions | Exact manifest cases; proving scenarios/seeds; focused tests | Proof boundary |
|---|---|---|---|---|
| tattoo1 | C `src/spec_procs2.c:945-1008`, assigned vnum 8086 at `src/spec_assign.c:296`; Go `sendTattooList`/`parseTattooChoice` at `pkg/game/spec_procs2.go:1130-1154`, wrapper at `:1156-1189`, `giveTattoo` at `:1191-1206`. | Shared list bytes and leading-space/empty/nondigit/range parsing. Local `me == nil` gate precedes ownership and parsing; five-offer table, owner tell, price tell, `giveTattoo`, return values remain local. List and buy gates emit actor-only bytes; success retains actor pain/blackout, room-only work/scream, and zone-limited shout audiences; no authored RNG draw. | `mob.tattoo1-list`, `mob.tattoo1-entry-gates`, `mob.tattoo1-owned-gate`, `mob.tattoo1-price-gate`, `mob.tattoo1-success-audience`, `mob.tattoo1-success-state`, `mob.tattoo1-fallthrough` (`spec-procs.tsv:237-243`). Scenarios `spec-proc-tattoo1`, `spec-proc-tattoo1-owned`, `spec-proc-tattoo1-price`, `spec-proc-tattoo1-success`, seed 1. Tests `TestSpecTattoo1ListAndEntryGates`, `TestSpecTattoo1PriceAndSuccessAudience`, `TestSpecTattoo1AppliesEveryOfferedTattoo`, `TestSpecTattoo1AutonomousEntry`. | The success unit pins all five offer identities/effects, gold, tattoo state, and healthy-player position recovery. No separate randomness exists in the extracted branches; `giveTattoo` was not changed by #1400. |
| tattoo2 | C `src/spec_procs2.c:1010-1072`, assigned vnum 18213 at `src/spec_assign.c:404`; Go wrapper `pkg/game/spec_procs2.go:1220-1255`. | Same shared list/parser. Local buy-only NPC fallthrough occurs before parsing, parsing occurs before the local `me == nil` check, and the four-offer table, Confucius tell, price gate, success path, and returns remain local. | `mob.tattoo2-list`, `mob.tattoo2-entry-gates`, `mob.tattoo2-owned-gate`, `mob.tattoo2-price-gate`, `mob.tattoo2-success-audience`, `mob.tattoo2-success-state`, `mob.tattoo2-fallthrough` (`spec-procs.tsv:244-250`). Scenarios `spec-proc-tattoo2`, `spec-proc-tattoo2-owned`, `spec-proc-tattoo2-price`, `spec-proc-tattoo2-success`, seed 1. Tests `TestSpecTattoo2ListAndEntryGates`, `TestSpecTattoo2PriceAndSuccessAudience`, `TestSpecTattoo2AppliesEveryOfferedTattoo`, `TestSpecTattoo2AutonomousEntry`. | Actor-only list/gates and shared success audiences/state are covered. No extracted RNG draw or untested success ordering was added. |
| tattoo3 | C `src/spec_procs2.c:1075-1137`, assigned vnum 21244 at `src/spec_assign.c:507`; Go wrapper `pkg/game/spec_procs2.go:1269-1304`. | Same shared list/parser. Local buy-only NPC fallthrough and parse-before-`me == nil` order remain; four-offer table, Polywig's intentionally garbled owner tell, price gate, success path, and returns remain local. | `mob.tattoo3-list`, `mob.tattoo3-entry-gates`, `mob.tattoo3-owned-gate`, `mob.tattoo3-price-gate`, `mob.tattoo3-success-audience`, `mob.tattoo3-success-state`, `mob.tattoo3-fallthrough` (`spec-procs.tsv:251-257`). Scenarios `spec-proc-tattoo3`, `spec-proc-tattoo3-owned`, `spec-proc-tattoo3-price`, `spec-proc-tattoo3-success`, seed 1. Tests `TestSpecTattoo3ListAndEntryGates`, `TestSpecTattoo3PriceAndSuccessAudience`, `TestSpecTattoo3AppliesEveryOfferedTattoo`, `TestSpecTattoo3AutonomousEntry`. | The manifest's entry row explicitly proves only the listed C-first bare/nonnumeric branches; out-of-range behavior is not claimed beyond the shared parser's source comparison. |
| tattoo4 | C `src/spec_procs2.c:1282-1340`, assigned vnum 2766 at `src/spec_assign.c:213`; Go wrapper `pkg/game/spec_procs2.go:1438-1466`. | Same shared list/parser. The outer `ch == nil || NPC || me == nil` gate remains before command dispatch; three-offer table, sleazy artist tell including its trailing space, price gate, success path, and returns remain local. | `mob.tattoo4-list`, `mob.tattoo4-entry-gates`, `mob.tattoo4-owned-gate`, `mob.tattoo4-price-gate`, `mob.tattoo4-success-audience`, `mob.tattoo4-success-state`, `mob.tattoo4-fallthrough` (`spec-procs.tsv:268-274`). Scenarios `spec-proc-tattoo4`, `spec-proc-tattoo4-owned`, `spec-proc-tattoo4-price`, `spec-proc-tattoo4-success`, seed 1. Tests `TestSpecTattoo4ListEntryOwnedAndPriceGates`, `TestSpecTattoo4SuccessAudienceAndState`, `TestSpecTattoo4AppliesEveryOfferedTattoo`. | Three offer identities/effects, actor/room/zone audiences, gold, tattoo and position state are covered; no authored RNG draw is present. |

The tattoo extraction is therefore **verified within the manifest and focused
unit boundaries**. It follows R1/R2 for bytes and command fallthrough, R3 for
unchanged state/order and draw-free shared branches, R4 by retaining every
shop exception, and R5e by using the assigned vnums and actual dispatch path.

### Directional castle guards ×3

| proc | C / Go sites and registration | Shared branches and local exceptions | Exact manifest cases; proving scenarios/seeds; focused tests | Proof boundary |
|---|---|---|---|---|
| castle_guard_down | C `src/spec_procs2.c:2134-2174`, assigned vnums 19626/19627/19640/19641 at `src/spec_assign.c:453-456`; Go `specCastleGuard`/wrapper `pkg/game/spec_procs2.go:2221-2301`. | Shared awake/position, mortal target, matching direction, owner fallthrough, grouped-owner actor/peer allowance, block actor/peer/room bytes and TRUE return; commandless scan preserves first assigned same-special fighting peer and canonical `mobHit`. Down keeps owner and grouped-owner offset +2 and name `castle_guard_down`. No RNG. | `mob.castle-guard-down-entry-gates`, `mob.castle-guard-down-block-audience`, `mob.castle-guard-down-return-intercept`, `mob.castle-guard-down-owner-fallthrough`, `mob.castle-guard-down-group-owner-audience`, `mob.castle-guard-down-autonomous-peer`, `mob.castle-guard-down-autonomous-mob-target` (`spec-procs.tsv:343-349`). Scenario `spec-proc-castle-guard-down`, seed 1. Tests `TestSpecCastleGuardDown_EntryGatesAndBlockAudience`, `TestSpecCastleGuardDown_AutonomousSecondGuardTarget`, `TestSpecCastleGuardDown_AutonomousSecondGuardTargetsMob`, `TestSpecCastleGuardDown_DoesNotTreatUnregisteredMobAsPeer`, plus `TestSpecCastleGuardDown_NilChDoesNotPanic`. | Unit proof covers player and mob targets, audience split, no movement, and assignment-name filter. The dormant C `mini_mud` gate (`src/spec_procs2.c:2139`) has no Go runtime mode and is not claimed as a live Go branch. |
| castle_guard_up | C `src/spec_procs2.c:2176-2216`, assigned vnums 19650/19651 at `src/spec_assign.c:457-458`; same Go helper/wrapper at `:2303-2315`. | Same shared behavior. Up retains owner offset +1 but grouped-owner offset 0, matching C's asymmetric `world[room+1]` vs `world[room]` checks, and name `castle_guard_up`. No RNG. | `mob.castle-guard-up-entry-gates`, `mob.castle-guard-up-block-audience`, `mob.castle-guard-up-return-intercept`, `mob.castle-guard-up-owner-fallthrough`, `mob.castle-guard-up-group-owner-asymmetry`, `mob.castle-guard-up-autonomous-peer`, `mob.castle-guard-up-autonomous-mob-target` (`spec-procs.tsv:350-356`). Scenario `spec-proc-castle-guard-up`, seed 1. Tests `TestSpecCastleGuardUp_EntryGatesAndBlockAudience`, `TestSpecCastleGuardUp_AutonomousSecondGuardTarget`, `TestSpecCastleGuardUp_AutonomousSecondGuardTargetsMob`, `TestSpecCastleGuardUp_DoesNotTreatUnregisteredMobAsPeer`. | The offset asymmetry is directly unit-proven. Duplicate-name/pointer identity beyond the tested target shapes is not claimed; the shared code preserves the pre-existing name-resolution seam. |
| castle_guard_north | C `src/spec_procs2.c:2218-2258`, assigned vnums 19510/19601/19602/19690/19691/19675/19676 at `src/spec_assign.c:447,450-451,459-462`; same Go helper/wrapper at `:2317-2327`. | Same shared behavior. North retains owner and grouped-owner offsets +2 and name `castle_guard_north`. No RNG. | `mob.castle-guard-north-entry-gates`, `mob.castle-guard-north-block-audience`, `mob.castle-guard-north-return-intercept`, `mob.castle-guard-north-owner-fallthrough`, `mob.castle-guard-north-group-owner-audience`, `mob.castle-guard-north-autonomous-peer`, `mob.castle-guard-north-autonomous-mob-target` (`spec-procs.tsv:357-363`). Scenario `spec-proc-castle-guard-north`, seed 1. Tests `TestSpecCastleGuardNorth_EntryGatesAndBlockAudience`, `TestSpecCastleGuardNorth_AutonomousSecondGuardTarget`, `TestSpecCastleGuardNorth_AutonomousSecondGuardTargetsMob`, `TestSpecCastleGuardNorth_DoesNotTreatUnregisteredMobAsPeer`. | Same boundary as down. `castle_guard_east` remains local at `pkg/game/spec_procs2.go:2014-2048` / `src/spec_procs2.c:1934-1970`; it was not part of #1400 and was not folded into the helper. |

The castle extraction is **verified within active registrations and named
command/autonomous tests**. The helper does not introduce a draw, changes no
movement or combat ordering, and retains the up offset exception. This is the
R1/R2/R3/R4/R5c/R5e boundary; the east proc and dormant `mini_mud` mode are
not generalized by inference.

### Fighter and paladin combat gates, target picking, and native arms

The additional combat-gate extraction actually present in #1400 is
`mobCombatSpecialTarget` at `pkg/game/spec_procs.go:437-450`. It is used only
by `specFighter` (`:452-470`) and `specPaladin` (`:472-490`). The helper checks
commandless entry, `POS_FIGHTING`, nonnegative HP, nonempty FIGHTING state,
zero mob wait, and the resolved current target. It consumes no RNG. Target
resolution is the existing combat-engine target first, then the mob target or
name fallback (`:130-153`); the C special receives the actual FIGHTING pointer.

| proc | C / Go registrations and call path | Local branches and fidelity points | Exact manifest cases; proving scenarios/seeds; focused tests | Unsupported branches |
|---|---|---|---|---|
| fighter | C `src/spec_procs.c:509-535`; assignments at `src/spec_assign.c:240,246,260-261,348,379,427,469,476,479-480,488,490`; Go map entries `pkg/game/spec_assign.go:76,81,93-94,171,202,245,286-287,293,296-297,305,307`. The live oracle uses vnum 20020. C `mobile_activity` skips fighting mobs, then `perform_violence` calls the special after the ordinary attack loop; Go uses `SetMobSpecialFunc` → `MobSpecialFunc` after `performOneHit`. | After the shared zero-draw gate, local `number(0,10)` is preserved. Cases 1/2/3/4 call native headbutt/parry/bash/berserk with subcmd=1; default returns FALSE. Focused tests pin combat target, native messages/audiences, recoil/damage, victim position/wait, parry marker, affects, and no invented mob wait mutation. | `mob.fighter-combat-action`, `mob.fighter-entry-gates`, `mob.fighter-headbutt`, `mob.fighter-bash`, `mob.fighter-parry`, `mob.fighter-berserk` (`spec-procs.tsv:32-37`). Scenario `spec-proc-fighter`, seeds 1,2,3,4,5,6. Tests `TestSpecFighter_Golden`, `TestSpecFighter_NativeSkills`. `--show-oracle` blocks showed green combat transcripts; per-arm state/message proof comes from the named unit. | No separate C-first vehicle exists for each additional assigned fighter vnum, so the audit does not claim each vnum's prototype/equipment/script combination. The shared function and native arms are covered; no other combat-gate family was extracted by #1400. |
| paladin | C `src/spec_procs.c:537-568`; assignments vnums 71 and 7915 at `src/spec_assign.c:258,265`; Go map `pkg/game/spec_assign.go:31,98`. Live oracle uses Death Dealer vnum 71, with the same combat seam. | After the shared gate, local `number(0,8)` is preserved. Cases 0/1/2/3/5 call parry/bash/charge/alignment-selected dispel/disarm; default still returns TRUE without bytes. Paladin's spell choice remains local (`paladinDispelSpell`). | `mob.paladin-entry-gates`, `mob.paladin-default-dispatch`, `mob.paladin-parry`, `mob.paladin-bash`, `mob.paladin-charge`, `mob.paladin-dispel-alignment`, `mob.paladin-disarm`, `mob.paladin-combat-action` (`spec-procs.tsv:38-45`). Scenario `spec-proc-paladin`, seeds 1,2,3,5,8. Tests `TestSpecPaladin_Golden`, `TestSpecPaladin_DefaultDispatch`, `TestSpecPaladin_ChargeNativeArithmetic`, `TestSpecPaladin_DispelAlignment`, `TestSpecPaladin_DisarmNativeAudienceAndState`, `TestSpecPaladin_DisarmFailure`; parry/bash are delegated to the fighter rows as the manifest states. | No separate oracle vehicle exists for paladin vnum 7915. The direct native tests cover the local arms, including failure posture and state; the audit does not infer unrelated class-specific setup. |

Fighter and paladin are **verified within the shared gate, target-picker, native
arm, and named live-vehicle boundaries**. The helper preserves gate order,
target resolution before the one local dispatch draw, inclusive bounds, arm
return conventions, native bytes/audiences, state changes, and R3 draw order.

### Undead knights — blocked, not excluded

| proc | C / Go sites | Registration and reachability | Evidence / unsupported branches |
|---|---|---|---|
| black_undead_knight | C `src/spec_procs.c:1144-1205`; Go wrapper/helper `pkg/game/spec_procs.go:1152-1190`. C hates `RED_UNDEAD 18402`, taunts while fighting with `number(1,20)`, and scans room-list order with `!number(0,2)`. Go passes hated vnum **11471** and the shared hunt draw is `number(0,3)`; its taunts use `randRange(1,20)`. | C declarations are present at `src/spec_assign.c:108-109`, but `rg` over `assign_mobiles`, `assign_objects`, and `assign_rooms` finds no assignment for the procedure or vnums 18401/18402. Go `RegisterSpec` entries at `pkg/game/spec_procs.go:1288-1289` do not create a VNum dispatch vehicle; `GetMobSpec` returns only from `MobSpecAssign` (`pkg/game/spec_assign.go:421-427`), and neither 11471 nor 18402 is assigned. `mobileActivityForMob` (`pkg/game/mobact.go:170-197`) and movement special lookup (`pkg/game/act_movement.go:469-485`) therefore cannot reach it in the current world. | No `spec-procs.tsv` case ID, proving scenario, or focused unit test exists. There is source and unit-level registration evidence only, plus the dated 2026-08-29 handoff's explicit no-assignment finding. All taunt arms, wake/NPC/HP gate, opposing-knight scan, probability, room message, `hit`, state, and TRUE return remain unsupported. |
| red_undead_knight | C `src/spec_procs.c:1208-1263`; Go wrapper/helper `pkg/game/spec_procs.go:1152-1169,1192-1210`. C hates `BLACK_UNDEAD 18401`, with the same `number(1,20)` taunt and `!number(0,2)` scan. Go passes hated vnum **11470** and the shared `number(0,3)` hunt draw. | Same no-assignment result on both sides; `SpecRegistry` registration alone is not live reachability. | No manifest case, proving scenario, or focused test. No oracle-green claim is made. |

Finding U-1 is **P2 audit-blocking / currently non-live**: if the Go helpers
are directly called or a future assignment makes them reachable, they are not
source-faithful on opposing VNums or hunt probability, and the C
`AWAKE`/`IS_NPC` gate is only represented by the current caller boundary. This
must not be “fixed” by inventing a registration or by promoting an exclusion
to green under R2/R4/R5e.

Smallest next action: first establish from the authoritative world/assignment
source whether either undead procedure is intended to be live. If yes, add a
C-first disposable vehicle and focused tests for the real vnums, both taunt
tables, wake/command/NPC/HP gates, room iteration order, `number(1,20)`,
`!number(0,2)`, battle-cry audience, canonical `hit`, state, and return. Then
correct or re-scope the helper against that evidence in a separate fidelity
change; do not combine it with this audit PR. If no, retain the procedures as
source-only and document them as unassigned rather than adding a synthetic
coverage case.

## Oracle and validation evidence

### Focused oracle result set

The frozen run used `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`
and `--show-oracle` for every case. The result set was computed from
`/home/zach/dp-phase5-2-audit-20260911/focused-summary.tsv`:

- 30 total scenario/seed rows: 19 tattoo/castle rows at seed 1, fighter
  seeds 1–6, and paladin seeds 1/2/3/5/8.
- 21 unique scenario files; 30 unique scenario/seed keys.
- 30 green, 0 non-green; failed=0, infra=0, timed_out=0, stale=0.
- Missing rows: none. Duplicate rows: none.
- Content/branch blocks inspected in
  `spec-proc-tattoo1-success--seed-1.log`,
  `spec-proc-castle-guard-down--seed-1.log`, all six fighter logs, and all
  five paladin logs. The tattoo block showed actor-only pain/blackout and
  peer room/shout bytes; castle showed actor/peer block and room statement;
  combat blocks showed the native combat transcripts. A green transcript is
  not treated as proof of every local arm; those arms are covered by the
  focused unit tests listed above.

### Commands and results

All commands ran from the audit worktree at the tested commit, with
`PATH=/usr/local/go/bin:$PATH` and the oracle environment above:

| command | result | durable log |
|---|---|---|
| focused family unit tests: `go test ./pkg/game -run 'TestSpec(Tattoo|CastleGuard|Fighter|Paladin)' -count=1` | PASS | `/home/zach/dp-phase5-2-audit-20260911/focused-family-tests.log` |
| `gofumpt -l .` | PASS, 0 files | `/home/zach/dp-phase5-2-audit-20260911/gofumpt.log` |
| `go build ./...` | PASS | `/home/zach/dp-phase5-2-audit-20260911/go-build.log` |
| `go vet ./...` | PASS | `/home/zach/dp-phase5-2-audit-20260911/go-vet.log` |
| `go test ./...` | PASS | `/home/zach/dp-phase5-2-audit-20260911/go-test-all.log` |
| `go test ./pkg/game/...` | PASS | `/home/zach/dp-phase5-2-audit-20260911/go-test-game.log` |
| `golangci-lint run ./...` | PASS, 0 issues | `/home/zach/dp-phase5-2-audit-20260911/golangci-lint.log` |
| `make fidelity-depth` | PASS; 4,798 total / 4,679 proven-delegated / 68 blocked / 51 excluded | `/home/zach/dp-phase5-2-audit-20260911/fidelity-depth.log` |
| `make expected-divergences-check` | PASS; expected pins OK; 26 unresolved rows across 10 scenarios | `/home/zach/dp-phase5-2-audit-20260911/expected-divergences-check.log` |

The full `make oracle-regression` census was **not run** because U-1 is a
material unresolved audit finding. A full census cannot create live proof for
unassigned procedures and would not clear the blocker; no census tally is
claimed. No pin or exclusion was created.

## Applicable rules and review boundary

- R1: compared exact actor, peer, room, and zone bytes in the active blocks and
  retained local exception text.
- R2: checked the actual command fallthrough and the C/Go registration and
  dispatch tables; unassigned undead code is not promoted to live behavior.
- R3: checked local inclusive RNG ranges and ordering; shared tattoo/castle
  branches are draw-free, and combat gates draw only before local dispatch.
  The undead mismatch is recorded rather than changed.
- R4: no generalized fallback, new registration, normalized message, or
  collapsed exception was introduced.
- R5c/R5e: audited the whole changed behavior class and actual call paths,
  including the combat callback after the ordinary NPC attack and the
  commandless mobile path. Unit or caller evidence is not substituted for
  coverage of a missing caller.

The original historical handoff is preserved. Its stale local-preparation
status is corrected by a dated note in that file linking this audit.
