# Modernization Phase 6.3 provenance reconciliation — 2026-09-13

## Recommendation

Accept the four merged slices as complete within their explicit scope:
spellDB (#1445), THAC0 (#1446), saving throws (#1447), and `LVL_IMMORT`
(#1449). Retain the `getTHAC0` input-domain guard as a literal with the
documented rationale below. Record the recovered NPC and player attack-count
thresholds in `GetAttacksPerRound` as two separately bounded future slices.
Defer the report’s inferred “weapon-skill” interpretation, the unnamed
remainder of “literal ladders,” and broader authority-level consolidation
because the recovered source does not establish those as finite original
Phase 6.3 scope.

This is a precise bounded disposition, not unconditional Phase 6.3 closure.
No implementation, scenario, test, source, oracle, or RNG code changed in this
PR. Phase 6.4 must not start from this handoff. The board threshold correction
in #1451 remains a separate fidelity correction and is not a modernization
delta.

## Audit boundary and recovered provenance

The audit ran in `/home/zach/darkpawns-provenance-audit`, a fresh worktree from
`origin/main` at `d688d51f1` (merge of #1451), on branch
`glm/audit-provenance-reconciliation`. GitHub showed no open modernization PRs
when this work started. The primary checkout’s existing
`docs/specs/tui-setup-wizard.md` edit was not touched. `src/` and
`darkpawns-c-oracle/` were read-only.

The original audit reports were later found at `/home/zach/dp-modernization`.
Before that recovery, #1450 correctly recorded that the reports were absent
from the tracked tree, reachable history, and the bounded unreachable-object
search. The new filesystem evidence changes the provenance conclusion, not
the implementation status: the relevant source is now preserved in
[`original-excerpts.md`](../../evidence/2026-09-13-phase6-3-provenance/original-excerpts.md)
with SHA-256 hashes, verbatim excerpts, and the source audit-clone commit.
The excerpts distinguish what the original report explicitly named from what
source inspection infers.

## Finite reconciliation table

“Explicit” means the recovered original report names the table, constant, or
historical formulas location. “Inference” means a current-source or label
interpretation that is not itself an original finite candidate list. Current
locations and readers were checked against the C and Go call paths under R5,
including R5e’s requirement to verify the actual call rather than trust a
summary.

| classification and original citation | historical function / meaning | current location and actual call path | merged work, if any | existing proof | disposition |
|---|---|---|---|---|---|
| Explicit: original roadmap Phase 6.3 row; `reports/06-roadmap.md:75` | spellDB keys and unkeyed spell records | C `src/spell_parser.c:51-209,257-266,1148-1178,1225-1232` and `src/spells.h:65-175`; Go `pkg/session/cast_cmds.go:20-137`, read by cast/mana/class-spell paths | #1445, `e59809f5d` | Complete `0..130`/103-record pre-keying fixture, C comparison, focused oracle matrix, and existing spellDB handoff/evidence | Already addressed within named scope. The known ID-54 issue remains separate. |
| Explicit: original roadmap row; `reports/06-roadmap.md:75` | THAC0 class rows; C table indexed by class and level | C `src/class.c:297-371`, read by `src/fight.c:1783-1786`; Go tables `pkg/combat/formulas.go:65-110` and `pkg/game/player.go`, read by `getTHAC0` → `CalculateHitChance` and `newCharacter` level-1 setup | #1446, `e38120cd3` | Independent fixture covers both Go copies (984 cells); C golden/semantic checksum; focused and full census evidence | Already addressed within named scope. Do not infer that all nearby formulas changed. |
| Explicit historical location, not a separate named keying target: `reports/raw/cisms-structural.md:111` | C directly indexes valid player levels; the Go guard clamps invalid input to the table domain | C `src/fight.c:1768,1783-1786`; Go `pkg/combat/formulas.go:294-309`; live path `CalculateHitChance` → `performOneHit` | No behavior change in #1446 | `TestTHAC0_Clamps` covers Go invalid levels and invalid classes; table proof covers levels 1–40 | Retained as a literal with a concrete rationale: `1` and `40` define the defensive Go table-domain guard, while C’s normal path supplies valid levels. Extracting it here would change or obscure invalid-input behavior and is not required to key the table. |
| Explicit: original roadmap row; `reports/06-roadmap.md:75` | saving-throw class/category rows and positional level data | C `src/magic.c:83-404,407-426`; Go `pkg/spells/saving_throws.go:27`, read by `GetSavingThrow` → `CheckSavingThrow` | #1447, `f9e3178d5` | Independent 2,460-cell fixture matches C checksum `caf57e8dc021b352253db65c00274194fe1cf1ba393101aa008b1403ad9a6b80`; focused oracle proof | Already addressed within named scope. Level positions, defaults, fallbacks, NPC handling, and formulas remain distinct. |
| Explicit: original roadmap/raw reports; `reports/02-c-isms.md:25`, `reports/raw/cisms-structural.md:80` | single source for `LVL_IMMORT=31` and its live authority boundary | C `src/structs.h:610-624`; Go canonical `pkg/combat/fight_core.go:54`, aliases/consumers in game, session, and scripting | #1449, `caf62b6f7` | Compile-time, consumer, boundary, Lua-export, focused oracle, and prior full-census evidence | Already addressed within named scope. Other authority levels are not automatically included. |
| Explicit recovered formula candidate: `reports/raw/cisms-structural.md:111,116` | NPC attack-count bands `31/30/27/20/10` and random `number(0,900)<level`, followed by haste/slow | C `src/fight.c:1910-1922`; Go `pkg/combat/formulas.go:563-588`; C caller `src/comm.c:822-823` → `perform_violence`; Go `pkg/engine/gameloop.go:320-323` → `CombatEngine.PerformRound` → `processCombatPair:519-523` | None | `pkg/combat/attack_rounds_golden_test.go:7-52` and `formulas_test.go:570-602`; combat depth is delegated/indirect, not an exhaustive threshold matrix | Suitable for a separately bounded future slice. Preserve literal thresholds and draw order until a C-backed, multi-seed, live `PerformRound` proof covers every boundary and random branch. |
| Explicit recovered formula candidate: `reports/raw/cisms-structural.md:111,116` | player extra attacks: class gates/chances at `>10`, `>12`, `>15`; all-player probes at `>25`, `>30`/`!number(0,500)`, and `>39` plus `+2` | C `src/fight.c:1924-1947`; Go `pkg/combat/formulas.go:589-640`; same C/Go violence call paths above; attack loop `pkg/combat/engine.go:589-596` | None | `pkg/combat/attack_rounds_golden_test.go:54-145` and `formulas_test.go:603-667`; named combat scenarios delegate to the combat-swing path but do not provide a complete threshold/draw-order matrix | Suitable for a separately bounded future slice. It is combat/RNG fidelity work, not authority naming; require seeded threshold and draw-position proof before any refactor. |
| Inference to be checked, based on the original label at `reports/raw/cisms-structural.md:111` | “weapon-skill learn thresholds” | Actual C practice path is `src/spec_procs.c:203-249` using `prac_params` from `src/class.c:261-267`; Go is `pkg/game/practice.go:9-27,91-124` plus `pkg/game/class_spells.go:19,346-365`. Neither path is `GetAttacksPerRound`. | None | C and Go practice/catalog call paths were inspected; the cited formulas were verified as attack counts | Deferred because the report’s label is not supported by the cited functions and the original Phase 6.3 row does not enumerate practice thresholds. Do not invent a skill-learning modernization slice from this wording. |
| Inference boundary: original roadmap phrase “name the literal ladders”; `reports/06-roadmap.md:75` | Any other unnamed numeric ladder, including `pkg/game/level.go:30-31` examples | Current examples include `pkg/game/level.go:21-96` and other formula/data sites, but no source-backed finite original list or shared meaning was recovered | None | Structural report gives examples and counts, not a complete candidate manifest | Deferred because proof and scope are insufficient. The four completed slices plus the two explicit attack-count formula candidates are the finite actionable reconciliation. |
| Current-source follow-on, not an explicit recovered Phase 6.3 candidate | broader `LVL_GOD`–`LVL_IMPL` authority consolidation | C `src/structs.h:610-624`; Go `pkg/game/limits.go:24-25`, `pkg/game/act_comm.go:47-49`, `pkg/game/houses.go:27-29`, `pkg/session/wizard_cmds.go:13-18`, `pkg/scripting/engine.go:719-720`; readers include item transfer, houses, wizard gates, and Lua export | #1449 addressed only `LVL_IMMORT` | Existing individual consumer tests; no complete independent C-derived matrix for every consumer | Deferred as a separately bounded, human-authorized authority-policy slice. Matching values do not establish shared concept; broader consolidation is not automatically original 6.3 scope. |

## Call-path and fidelity conclusions

The recovered formulas report is now reconciled under R5:

- C’s `perform_violence` is called from the 2-second violence pulse and walks
  the combat list. Go’s violence callback is called from the corresponding
  game-loop pulse and reaches `PerformRound`, `processCombatPair`,
  `GetAttacksPerRound`, and the attack loop. This proves reachability, not
  sufficient proof for a combat/RNG refactor.
- The historical `31`/`30` values in the NPC branch are threshold semantics,
  while `LVL_IMMORT=31` is an authority constant. Their numeric coincidence
  is not evidence of a shared constant. The same separation applies to `39`
  in the player extra-attack gate versus `LVL_IMPL=40`.
- The current Go function mirrors the cited C attack-count formula, but the
  existing tests are boundary/unit evidence plus delegated combat coverage,
  not a complete C-derived threshold and draw-position proof. The two formula
  rows therefore remain future slices.
- The report’s “weapon-skill” wording is a provenance correction, not a
  production defect. The actual practice table and skill catalog are a
  different C/Go path and remain outside this bounded reconciliation.

## Separate fidelity item

The board removal authority mismatch identified by #1451 is deliberately not
reopened. Current `origin/main` includes that merged correction’s evidence and
production change. It is a separate fidelity item, not a Phase 6.3
modernization delta; no board files were changed here.

## Current-main census and validation

The prior `caf62b6f7` census cannot be reused as current-main whole-corpus
evidence: the current base includes merged #1451, which changed board
production, tests, and scenario inputs after that checkpoint. The final
validation record below therefore uses a fresh `make oracle-regression` on
this branch after the documentation commit. Scripts and tested inputs were
frozen during the run; no driver or scenario inputs were changed.

The complete command record, exact tally, manual INFRA inspection, and the
per-gate results are recorded here before PR publication. A healthy result
must have `failed=0`, `infra=0`, `timed_out=0`, and `stale=0`; the only allowed
non-pass is the established human-cleared `accuse-noarg-depth` unpinnable
baseline. Exit status 2 by itself is not acceptance evidence.

## Handoff

This dated handoff recommends human review of the finite table and the two
future combat/RNG slice boundaries. It does not authorize implementation,
reopen 6.1/6.2, begin 6.4, or merge itself. The review decision still needed
is whether to accept the four completed slices plus these explicit retained
and deferred dispositions as the Phase 6.3 record, and separately whether to
authorize a future bounded `GetAttacksPerRound` proof slice.
