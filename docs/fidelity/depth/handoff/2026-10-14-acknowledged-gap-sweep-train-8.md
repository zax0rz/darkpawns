# Acknowledged Gap Sweep — Train 8 — 2026-10-14

## Scope and inherited inventory

Train 8 continues the finite inventory recorded before the first production
edit in [`2026-10-13-acknowledged-gap-sweep.md`](2026-10-13-acknowledged-gap-sweep.md).
That handoff retains the full A/B inventories, the baseline 21 acknowledged
comment hits and widened search, the silent-pulse and 89 assigned-special
crosswalks, triage evidence, and copy-ready issue text. Its itemized records
remain
[`2026-10-13-acknowledged-gap-sweep-comments.tsv`](2026-10-13-acknowledged-gap-sweep-comments.tsv)
and
[`2026-10-13-acknowledged-gap-sweep-specials.tsv`](2026-10-13-acknowledged-gap-sweep-specials.tsv).
Train 8 changes only the corpse attack-number mapping family.

The fresh worktree is
`/home/zach/.codex/worktrees/luna-gap-sweep-8/darkpawns`, branch
`codex/luna-acknowledged-gap-sweep-8`, based on freshly fetched `origin/main`
at `d8e8024ddbfe52fbde36d16e0b02e2d7606fac71`. This train does not include
commits from the other open sweep PRs.

## Triage checkpoint

| Item | Disposition | Evidence / owner |
|---|---|---|
| Corpse description attack-number mapping | Fixed in Train 8 | C `make_corpse()` switch: `src/fight.c:258-372`; Go `attackTypeToCorpseAttack`: `pkg/game/death.go:889-934`. |
| Disintegrate corpse branch | Already separately handled; preserved and tested | C diverts to `make_dust()` before the switch at `src/fight.c:267-270`; Go raw-kill death builders divert attack type 93 to dust. |
| Detailed `show shops` report | Candidate for a later train | C `src/shop.c:1413`, `src/act.wizard.c:2414`; Go `pkg/session/wiz_info.go:237`. Copy-ready request remains in the inherited inventory. |
| Room-wide undead `DoTurn` | Candidate for a later train | C `src/new_cmds2.c:960-1030`; Go `pkg/game/skills2.go:492-567`. |
| Jail-guard visibility gate | Candidate for a later train; retain combat owner boundary | C `src/fight.c:1374`; Go `pkg/combat/engine.go:954`. Do not change shared combat transcript/RNG cases owned by `mob.teleport-victim-combat-transcript`. |
| Lua character/mobile table fields | Separate script-table issue | Full field comparison remains in the inherited inventory. |
| DP-1426, DP-1419, DP-1364, DP-493; generic NPC command interpreter; stop tier | Owned, blocked, design-owned, or stop-tier; no change | Preserve the owners and boundaries recorded in the inherited handoff. |

## Train 8 source audit and changes

The complete C switch and real death call path were checked in `src/`:

- `src/fight.c:534-580` implements `raw_kill()`. The live Medusa special at
  `src/spec_procs2.c:1552-1580` reaches it with `SPELL_PETRIFY` at line 1577.
- `src/fight.c:267-270` handles `SPELL_DISINTEGRATE` before creating a corpse
  or entering the attack switch. `make_corpse()` then covers every group at
  `src/fight.c:281-372`.
- The spell and attack IDs are defined in `src/spells.h:60-105,142-169,177-238,264-280`.
  Compared against the entire switch, the exact classes are: fire 5/26/58/96;
  chill 8; blast 10/41/92/312; energy drain 25/83; lightning 30/6/37;
  psiblast 100; bruising skills 132/134/136/188/189/141/145/146/156 and
  types 305/307/313/302; hacked skills 150/131/173 and types 304/308/303;
  disembowel 184; neckbreak 190; crushed types 306/309/310; ventilated types
  311/314; drowning 103; petrify 104. All other values use the default corpse
  text, while disintegrate produces dust.
- Before the fix, Go mapped energy drain to 21 (detect poison), shocking grasp
  to 31 (locate object), psiblast to 34 (protection from evil), and petrify to
  35 (remove curse), omitted meteor swarm 41, and incorrectly included
  bearhug 142 and trip 144 in the bruised group. Those spell IDs are confirmed
  in `src/spells.h:85-105,98-105,165-169,189-198`; bearhug/trip are absent from
  the full `make_corpse()` switch.

Changes:

- `pkg/game/death.go`: replaced approximate numeric spell IDs with the exact
  exported constants from `pkg/spells`; added meteor swarm to the blast group;
  corrected energy drain, shocking grasp, psiblast, and petrify; removed
  bearhug/trip from the bruised group; removed the obsolete comment admitting
  approximate mapping. No draw or branch order changed (R3).
- `pkg/game/death_test.go`: expanded the mapping table across every C switch
  arm, nearby wrong IDs, the default arm, and `TYPE_UNDEFINED`; added real
  `RawKillCombatant` corpse-description dispatch coverage and a disintegrate
  dust-path check.
- `cmd/dp-oracle-diff/scenarios/spec-proc-medusa.txt`: added a follow-up room
  look after the live failed-save Medusa path so the stone-frozen corpse
  description is visible to the oracle.
- `docs/fidelity/depth/corpse.tsv` and `spec-procs.tsv`: recorded the exhaustive
  unit proof and the live Medusa corpse-description oracle case.

No shared production state, persistence format, C source, or oracle binary was
changed. Race tests were not required.

## Proofs

| Proof | Result |
|---|---|
| `go test ./pkg/game -run 'TestAttackTypeToCorpseAttack|TestRawKillCorpseAttackDispatch|TestRawKillDisintegrateDispatchesDust'` | PASS after restore. |
| R5h mutation: restore wrong IDs 21/31/34/35, omit meteor swarm 41, and re-add bearhug/trip | RED: the mapping table failed 11 assertions and the real raw-kill description dispatch failed 7 assertions. Restored the corrected implementation. |
| Live oracle `spec-proc-medusa`, follow-up room look | PASS: targeted census `dp-1371-train8-medusa`, seed 1, `CLEAN`; retained at `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train8-medusa/`. |
| Full combined census on committed Train 8 tip | Pending; append the retained verdict and summary after completion. |

The full handoff includes the D4 attack mapping row and D3 Medusa corpse
description row. The existing Medusa oracle vehicle pins the actual C
`raw_kill(..., SPELL_PETRIFY)` call and will fail if the corpse bytes remain
generic (R1/R5e/R5h).

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
| `make fidelity-depth` | PASS; 5,523 cases, 5,433 proven/delegated, 99.8% actionable |
| `make fidelity-units` | PASS; 1,426 rows, 1,316 symbols, 1,611 claims |
| `make string-census` | PASS; 98 known Go-only segments, 0 unreviewed. It reported 24 obsolete baseline entries and stale generated reports; no report update was needed for this train. |
| `gosec -severity=high -quiet ./...` | PASS |
| Targeted census `spec-proc-medusa` | PASS: `scenarios=1 passed=1 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=0`; `CLEAN` at Go HEAD `3694871a1`. |
| Final combined census on committed Train 8 tip | Pending. |

## Delivery

This is one corpse-description mapping family, below the train boundary. Open
the PR without merging, run the Medusa targeted census when available, then
run one full combined census on the final committed tip and append its retained
verdict. Start the next train from a freshly fetched `origin/main`.
