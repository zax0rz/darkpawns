# Acknowledged Gap Sweep — Train 7 — 2026-10-14

## Scope and inherited inventory

Train 7 continues the finite inventory recorded before the first production
edit in [`2026-10-13-acknowledged-gap-sweep.md`](2026-10-13-acknowledged-gap-sweep.md).
That handoff retains the full acknowledged-comment inventory (including the
baseline 21 hits and widened 175-line search), the silent-pulse and all-89
assigned-special crosswalks, triage evidence, and copy-ready issue text. Its
itemized records remain
[`2026-10-13-acknowledged-gap-sweep-comments.tsv`](2026-10-13-acknowledged-gap-sweep-comments.tsv)
and
[`2026-10-13-acknowledged-gap-sweep-specials.tsv`](2026-10-13-acknowledged-gap-sweep-specials.tsv).
This train changes no dispositions outside the mobile-wandering candidate
below.

The fresh worktree is `/home/zach/.codex/worktrees/luna-gap-sweep-7/darkpawns`,
branch `codex/luna-acknowledged-gap-sweep-7`, based on freshly fetched
`origin/main` at `d8e8024ddbfe52fbde36d16e0b02e2d7606fac71`. PR #1955 also edits
`pkg/game/mobact.go`, but only its earlier charmed/scavenger blocks; this
train's edit is the later post-door wandering dispatch. The changed hunks do
not overlap.

## Triage checkpoint

| Item | Disposition | Evidence / owner |
|---|---|---|
| Water and flying sector gates in mobile wandering | Fixed in Train 7 | C `mobile_activity()` checks `CAN_SWIM` and `IS_FLYING`; Go now mirrors the capabilities and C loop `continue`. |
| Detailed `show shops` report | Fix candidate for a later train | Exact C report is `src/shop.c:1413` and `src/act.wizard.c:2414`; current Go list-only behavior is `pkg/session/wiz_info.go:237`. Copy-ready request is in the inherited handoff. |
| Room-wide `DoTurn` undead behavior | Fix candidate for a later train | `src/new_cmds2.c:960-1030` scans room occupants and preserves mana/RNG order; Go resolves one target in `pkg/game/skills2.go:492-567`. Copy-ready request is in the inherited handoff. |
| Corpse attack-number mapping | Fix candidate for a later train | Audit `src/fight.c:281-370` against `pkg/game/death.go:894-931`; multiple spell-number mappings are missing or point to unrelated values. |
| Jail-guard visibility gate | Fix candidate for a later train, retaining combat owner boundary | `src/fight.c:1374` checks `CAN_SEE(ch, victim)`; Go `pkg/combat/engine.go:954` does not. Do not alter the shared transcript/RNG cases owned by `mob.teleport-victim-combat-transcript`. |
| Lua character/mobile table fields | Separate script-table issue | The incomplete table bridge is inventoried in the inherited handoff; it is outside this movement family. |
| `room_activity` croak, `loud_mobs`, `flow_room` | Owned elsewhere | DP-1426 / GLM. |
| Lethal login replay | Owned elsewhere | DP-1419 / DeepSeek. |
| Character-order and global point-update object timers | Tracked / blocked depth work | DP-1364 and the blocked timer row; do not duplicate here. |
| Shared combat message/attack/RNG transcript | Owned elsewhere | `mob.teleport-victim-combat-transcript`. |
| Generic NPC command interpreter | Design-owned | `lua.npc-command-unported`, `force.npc-command-interpreter`; preserve for Zach's design session. |
| Entry, nanny, persistence, session lifecycle, transport, save formats | Stop tier | DP-1423/1424/1425 and the inherited stop-tier inventory. |

The inherited handoff's copy-ready issue text remains the source for `DoTurn`,
`show shops`, the Lua table bridge, and jail-guard visibility. For the newly
audited corpse mapping, the issue text is: “Compare every C `make_corpse()`
attack-number branch in `src/fight.c:281-370` with
`pkg/game/death.go:894-931`; correct missing or misnumbered spell mappings as
one class. Add a live death/corpse-path proof and an R5h control for the
mapping, and preserve exact attack IDs and corpse bytes (R1/R3/R5e).”

## Train 7 source audit and changes

The behavior class was checked against the live C path:

- `src/mobact.c:119-143` draws `number(0,18)`, validates movement topology,
  then checks the destination sector. A failed water/flying capability check
  uses `continue` in the outer mobile loop, skipping this mobile's remaining
  sound, scripts, aggression, and race-hate work.
- `src/utils.h:148-153` defines `IS_FLYING` and `CAN_SWIM`;
  `src/act.movement.c:58-78` defines `has_boat`. Go reuses
  `roomActivityHasBoat`, which covers waterwalk, fly, immortal level, and boat
  inventory/equipment. `dummy_mob` is a zero-initialized global at
  `src/db.c:104`, assigned to mob prototypes at `src/db.c:1281`, and has no
  preference-flag writes in `src/`; therefore NPC `PRF_NOHASSLE` does not grant
  swimming or flight.
- `src/act.movement.c:214-280` sends the departure and direction-sensitive
  arrival acts, suppressing them for sneaking mobiles. `src/comm.c:2397-2477`
  performs `$n` substitution and sentence capitalization.

Changes:

- `pkg/game/ai.go`: water sectors now require C-equivalent `CAN_SWIM`; the
  flying sector requires `AFF_FLY`. Room death/no-mob and stay-zone gates stay
  ahead of the sector checks, as in C. Successful wandering uses `Act` for the
  visible-invisibility, capitalization, sneak, and directional arrival bytes
  from `perform_move()`.
- `pkg/game/mobact.go`: a failed sector gate returns from the per-mobile
  dispatcher at the C `continue` boundary, preserving draw and dispatch order.
- `pkg/game/mobact_test.go`: the real `World.MobileActivity()` path now proves
  both water sectors, waterwalk/fly/boat/immortal access, flying-only access,
  room-flag ordering, the blocked-sector draw and aggression cut-off, and
  source/destination movement acts including sneak suppression.
- `cmd/dp-oracle-diff/scenarios/mob-wander-waterwalk.txt` and
  `mob-wander-flying.txt`: disposable topology with source and destination
  observers exercises the live C and Go pulse path.
- `docs/fidelity/depth/mobile-activity.tsv`: adds the D5 oracle and unit rows.

No shared production state or persistence format changed; race tests were not
needed. No source under `src/` or the oracle checkout was edited.

## Proofs

| Proof | Result |
|---|---|
| Focused `go test ./pkg/game -run 'TestMobileActivity(WaterAndFlyingSectorGates|BlockedSectorSkipsAggression|WanderActsMatchPerformMove)$' -count=1` | PASS after restore; includes exact flying-arrival wording |
| R5h mutation: make incapable water/flying entries return normally instead of taking the C `continue` | RED: water/swim/flying movement cases consumed the extra sound draw; `TestMobileActivityBlockedSectorSkipsAggression` started combat against `Visible`. Restored implementation, then focused suite PASS. |
| Direct `--show-oracle`, seed 1, both scenarios | CLEAN; waterwalk showed `The ice daemon leaves up.` and `The ice daemon climbs in from below.` Flying showed the source leave. Its destination observer is ejected by C `room_activity()` before the mobile arrives, so the flying-arrival act is unit-proven. |
| Direct `--show-oracle`, seed 2, both scenarios | CLEAN; both showed the mobile leaving `down`; waterwalk also showed `The ice daemon climbs in from above.` This exposes a different door selection under another seed and exercises the shared C/Go draw. |
| Direct seed 3 exploratory runs | Transcript-clean but not counted as movement proof: the mobile was absent from the source observer's initial `look`, so the intended movement was not visible in that probe. |
| Targeted census `luna-train7-wander` | CLEAN: 2 scenarios, 2 passed, no expected/unpinnable/stale/failed/infra/timeout/unstable rows; 47.167s. Results: `PASS mob-wander-flying 46`, `PASS mob-wander-waterwalk 46`. |

Direct show-oracle logs and dumps are retained in
`~/Archives/darkpawns/oracle-runs/2026-10-14/luna-train7-show-oracle/`.
The targeted census is retained in
`~/Archives/darkpawns/oracle-runs/2026-10-14/luna-train7-wander/`.

## Required gates

| Gate | Result |
|---|---|
| `make fmt` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./...` | PASS |
| `go test ./pkg/game/...` | PASS |
| `golangci-lint run ./...` | PASS, 0 issues |
| `make deadcode-ratchet` | PASS, 52 baselined and 0 new |
| `make fidelity-depth` | PASS; 5,525 cases, 5,435 proven/delegated, 99.8% actionable |
| `make fidelity-units` | PASS; 1,427 rows, 1,316 symbols, 1,611 claims |
| `make string-census` | PASS; 98 known Go-only segments, 0 unreviewed. It reported 24 obsolete baseline entries and stale generated reports; no report update was needed for this train. |
| `gosec -severity=high -quiet ./...` | PASS |
| Final combined census on committed Train 7 tip | Pending; append the retained result after completion. |

## Delivery

This is one mobile-wandering family, below the train boundary. Commit the code,
open its PR without merging, then run the required full combined census on that
committed tip and append its manifest/verdict here. Keep the inherited owner
boundaries and begin the next train from a freshly fetched `origin/main`.
