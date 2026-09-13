# Modernization Phase 6.3 audit — 2026-09-12

## Disposition

This is a bounded audit of Phase 6.3 on current `origin/main`, not a new
implementation task. The four explicitly named slices are complete within
their named scope:

- spellDB named keys and keyed fields (#1445);
- THAC0 class keys in both runtime tables (#1446);
- saving-throw class/category keys (#1447); and
- canonical `LVL_IMMORT` with compatibility aliases and consumers (#1449).

The whole roadmap item is not closed as an unbounded claim. The original audit
reports have now been recovered at `/home/zach/dp-modernization`; durable
verbatim excerpts and source hashes are preserved in the
[`2026-09-13 provenance evidence`](../../evidence/2026-09-13-phase6-3-provenance/original-excerpts.md).
That recovery changes the provenance conclusion, not the implementation
status. It establishes two additional explicit formula candidates in
`GetAttacksPerRound` (NPC attack-count bands and player extra-attack
thresholds/chances). They remain separately bounded future combat/RNG slices;
the report's “weapon-skill” label is corrected by the actual C and Go call
paths. The recommendation is **accept a bounded reconciliation disposition**:
record the four slices as complete within scope, retain the two formula
candidates for separately authorized future work, and defer inferred or
broader authority-level candidates whose original scope is not finite.

Audit vehicle: `/home/zach/darkpawns-provenance-audit`, branch
`glm/audit-provenance-reconciliation`, based on `origin/main` at
`d688d51f1`. The primary checkout’s existing
`docs/specs/tui-setup-wizard.md` edit was left untouched. `src/` and
`darkpawns-c-oracle/` were not modified.

The governing `AGENTS.md`, `docs/fidelity/RULEBOOK.md`,
`docs/fidelity/DEPTH_TESTING.md`, this roadmap, all four implementation
handoffs/evidence records, and the relevant current C and Go call paths were
reviewed.

## PR and history verification

All listed PRs are merged. Each implementation commit is an ancestor of the
current `HEAD`; no later production change touches the audited implementation
files.

| PR | merge commit | implementation checkpoint | disposition |
|---|---|---|---|
| #1445 | `2ea3474002e5f055395ac9d90251e22bb84eabf6` | `e59809f5d` | Phase 6.3 spellDB slice merged |
| #1446 | `e54301fb4abfea6bf0c4811111d244c4ae141912e` | `e38120cd3` | Phase 6.3 THAC0 slice merged |
| #1447 | `dd2e4638a583d5c876f59af3aca732f37f90e7b9` | `f9e3178d5` | Phase 6.3 saving-throw slice merged |
| #1448 | `d580f138991af215683076e3e4e62360e36d14cc` | `d3871f69f` integration | separate oracle retry-classification repair; excluded from Phase 6.3 delta |
| #1449 | `3b131e940b8cece457044d9895ff8ba45d942092` | `caf62b6f7` | Phase 6.3 `LVL_IMMORT` slice merged |

The four implementation commits were checked with
`git merge-base --is-ancestor`. The current `HEAD` also includes merged PR
#1451, which changed `pkg/boards/boards.go`, its tests, the board depth input,
the board ledger, and the board handoff after `caf62b6f7`. Therefore the prior
`caf62b6f7` census is historical evidence for the four implementation slices,
not a reusable current-main whole-corpus census. This PR runs a fresh census
after the documentation checkpoint.

There were no open PRs when the audit checked GitHub. The audit itself will
open one documentation-only PR after this handoff and roadmap update; it will
remain unmerged.

## Phase 6.3 disposition matrix

| target and C source | current representation and actual readers | merged implementation and tested checkpoint | independent preservation / C evidence | limitations and verdict |
|---|---|---|---|---|
| spellDB raw keys and unkeyed records; `src/spells.h:65-175`, `src/spell_parser.c:51-209,257-266,1148-1178,1225-1232` | `pkg/session/cast_cmds.go:20-137` remains `map[int]*spellData`; named spell constants and keyed fields. Readers: `manaCost`, `cmdCastCommand`, and `grantClassSpells` in `pkg/session/spell_level.go`. | #1445 / `e59809f5d`; current preservation command passes. | Independent pre-keying fixture covers all indices `0..130`, 103 present records, 28 absent/default positions, and all fields. Existing spell-info/C comparison and source citations remain separate from the initializer. | The known spellDB ID-54 mismatch remains a separate fidelity debt; no keying regression was found. **Complete within named scope.** |
| THAC0 class rows; `src/class.c:297-371`, class IDs `src/structs.h:115-128`, reader `src/fight.c:1763-1786` | Both Go copies remain `[12][41]int`, keyed by existing class identifiers in `pkg/combat/formulas.go` and `pkg/game/player.go`. Readers: `getTHAC0` → `CalculateHitChance`, and `newCharacter` level-1 initialization. | #1446 / `e38120cd3`; current THAC0 preservation, identifier, clamp, golden, and constructor tests pass. | Independent numeric fixture covers both 12×41 copies (984 cells total); C golden fixture and semantic checksum `38562ecc51aa1043b365442250e3cf8684b26c9d055ee07dc781e12ea1d85b9b` cover the 492-cell table. | NPC/file THAC0 handling and derived combat formulas were not changed. **Complete within named scope.** |
| saving-throw class/category rows; `src/magic.c:83-404,407-426`, class/category IDs in `src/structs.h` and `src/spells.h` | `pkg/spells/saving_throws.go:27` remains `[12][5][41]int`; class and category dimensions use existing identifiers. Readers: `GetSavingThrow` and `CheckSavingThrow`. | #1447 / `f9e3178d5`; current table, identifier, sentinel/default, accessor-boundary, NPC, and saving tests pass. | Independent pre-keying fixture covers 2,460 cells and matches the C fixture at SHA-256 `caf57e8dc021b352253db65c00274194fe1cf1ba393101aa008b1403ad9a6b80`. | Level `0..40` is intentionally positional; fallbacks, NPC Warrior handling, save formulas, and draw order remain separate. **Complete within named scope.** |
| single-source `LVL_IMMORT`; `src/structs.h:610-624`, especially line 620 | Canonical untyped compile-time `combat.LVL_IMMORT = 31` in `pkg/combat/fight_core.go:54`. `pkg/game` and `pkg/session` compatibility aliases, plus the Lua export, now consume it. Live checks include combat gates, `cmdLevels`, XP, `fly_exit_up`, and scripting export. | #1449 / `caf62b6f7`; current compile-time, consumer, and focused boundary tests pass. | Independent C expectation is `31`; tests exercise constant arithmetic/array use, level-30/31 boundaries, command output, and Lua numeric export. | Other authority levels, formula thresholds, and privilege policy were deliberately not unified. **Complete within named scope.** |
| roadmap phrase “name the literal ladders” | The recovered phrase still does not enumerate a finite candidate list; see the corrected provenance section below. | No PR claims this as complete. | The recovered source supports two explicit combat-formula candidates; other numeric examples require separate scope proof. | **Explicitly deferred / bounded reconciliation disposition.** |

## Proof-boundary review

The evidence supports behavior-preserving modernization, with the following
boundaries kept explicit:

- The spellDB, THAC0, and saving-throw preservation fixtures are independent
  of the new initializers. Their complete-table results show that values and
  positions survived; they are not by themselves a full C-fidelity proof.
  The separate C fixtures/source comparisons and focused oracle matrices are
  the C evidence.
- Saving-throw accessor fallbacks and clamped endpoints are distinct from
  stored sentinel/default cells. The tests cover both. The same distinction
  applies to THAC0 invalid-class/level fallbacks and absent spellDB map
  indices.
- The `LVL_IMMORT` aliases are Go `const` aliases, not runtime variables.
  The compile-time tests cover arithmetic and array contexts, while the
  consumers cover behavior at the relevant boundaries.
- The spellDB ID-54 issue predates and is independent of keying. It remains a
  separate fidelity debt. Existing expected divergences, the permitted
  `accuse-noarg-depth` unpinnable case, and unrelated pre-existing debts are
  not charged against these behavior-preserving slices.
- Matching integers were not treated as shared meaning. Spell IDs, class IDs,
  table dimensions, sentinels, formulas, authority thresholds, and authored
  board levels remain separate unless C and a live call path establish the
  relationship.

## “Name the literal ladders” provenance and current-source follow-on inventory

The roadmap says it was derived from `reports/01–05`. Those report files are
still absent from the tracked tree, reachable history, and the bounded
unreachable-object search; the recovered home-directory copy is the newly
identified original source. `git log --all -S'name the literal ladders'`
continues to find the roadmap wording, not a finite candidate manifest. The
recovered reports therefore narrow the boundary by proving the two
`GetAttacksPerRound` formula candidates, but they do not turn every numeric
literal into unfinished modernization work.

The following is the finite current-source replacement inventory for human
review.

### A. C authority-level family — candidate, not yet authorized

C defines the authority family in `src/structs.h:610-624`:

```text
LVL_IMPL=40, LVL_GRGOD=38, LVL_HIGOD=36, LVL_LEGEND=35,
LVL_GOD=34, LVL_IMMORT=31, LVL_FREEZE=LVL_GRGOD
```

`LVL_IMMORT` is closed by #1449. The remaining Go definitions or literal
exports are:

- `pkg/game/limits.go:24-25`: `LVL_GOD=34`, `LVL_IMPL=40`;
- `pkg/game/act_comm.go:47-49`: package-local `lvlGod=34`, read by
  `pkg/game/item_transfer.go:516,527`;
- `pkg/game/houses.go:27-29`: `LVL_GRGOD=38`, used by house administration;
- `pkg/session/wizard_cmds.go:13-18`: `LVL_GOD`, `LVL_LEGEND`, `LVL_HIGOD`,
  `LVL_GRGOD`, and `LVL_IMPL`, used by wizard command checks and related
  session code; and
- `pkg/scripting/engine.go:719-720`: the canonical `LVL_IMMORT` export and a
  remaining literal `LVL_IMPL=40` Lua export.

The live call paths include item-transfer authority checks, house access,
wizard movement/transfer and command gates, and the Lua global export. The
risk is duplicated authority policy drifting across packages; the package
boundary also matters because `pkg/boards` is a leaf package and cannot simply
import `pkg/game` without a cycle. C’s `LVL_FREEZE`/`LEVEL_*` aliases have no
identified live Go consumer in this bounded search, so no consumer is invented.

Existing tests cover individual wizard/command behaviors, but there is no
independent C-derived inventory test for this whole remaining family and no
complete boundary matrix for every live consumer. The smallest justified
future slice is a human-approved constants inventory followed by compile-time
aliases and independent threshold tests for the identified live consumers;
it must explicitly exclude formula bands and authored values. Leave it
unchanged during this audit because the recovered original scope explicitly
names `LVL_IMMORT`, not a complete authority-policy family, and unifying
privilege thresholds may change behavior.

### B. `GetAttacksPerRound` — recovered explicit formula candidates

The recovered structural report explicitly cites the historical
`formulas.go:582-637` thresholds and percentile checks. Source inspection
against `src/fight.c:1898-1947` proves that the cited code is
`GetAttacksPerRound`/`perform_violence` behavior: NPC bands at
`31/30/27/20/10`, player gates at `>10/>12/>15/>25/>30/>39`, and the C random
probes. These are formula thresholds inside one live function, not authority
levels. The C call is `comm.c:822-823` → `perform_violence`; Go reaches the
function from `pkg/engine/gameloop.go:320-323` → `PerformRound` →
`processCombatPair:519-523`. Existing unit/golden tests and delegated combat
coverage do not yet constitute a complete C-derived threshold and draw-order
matrix. The two candidates remain unchanged and are suitable only for
separately bounded future combat/RNG slices.

The original report's phrase “weapon-skill learn thresholds” is not supported
by the cited functions. Actual C practice uses `spec_procs.c:203-249` and
`class.c:261-267`; current Go uses `pkg/game/practice.go:9-27,91-124` and
`pkg/game/class_spells.go`. No practice modernization is authorized by the
recovered Phase 6.3 source.

### C. Concrete separate fidelity debt: board removal authority bypass

`pkg/boards/boards.go:545`, `BoardSystem.RemoveMsg`, checks
`ch.GetLevel() < mi.Level && ch.GetLevel() < 59`. The comment claims
`LVL_IMPL-1`, but C `src/boards.c:403-405` checks
`GET_LEVEL(ch) < LVL_IMPL-1`; with C `LVL_IMPL=40`, the bypass threshold is
level `39`, not `59`.

The live path is `pkg/game/spec_assign.go:336-355` registration of `gen_board`
→ `pkg/game/boards.go:45-94` → `genBoard`’s remove case at lines 85-86 →
`BoardSystem.RemoveMsg`. On a board whose `RemoveLvl` permits the actor (for
example, a board configured with `RemoveLvl=0`), a level-39 actor removing a
level-40 author’s message is accepted by C but rejected by Go, after passing
the board read/remove gates. This demonstrates the mismatch within C’s
normal level range. Existing `pkg/boards/boards_test.go`
coverage proves low-level rejection and read-level gating, but has no
level-38/39/40 boundary case against a level-40 author. `git blame` attributes the literal to the
package extraction commit `4c1759cc72` (2026-07-06), with no later repair.

This is behavior-changing fidelity work, not evidence that the completed
keying slices are incomplete. The smallest repair is one C-threshold boundary
test and one dependency-neutral or injected authority value; importing
`pkg/game` directly into `pkg/boards` is not safe because `pkg/game` already
imports `pkg/boards`. No production or test code was changed in this audit.

### Explicit exclusions

`FindExp`’s class/level ladder was already recorded as a completed Phase 6.1
code/data slice and is not double-counted here. Spell IDs, class IDs, array
dimensions, stored sentinels/defaults, board `ReadLvl`/`RemoveLvl` authored
values, and other C formulas are not candidates merely because an integer
matches another integer.

## Measured deltas

The following are exclusive PR merge diffs against each merge’s first parent,
with production, tests, and documentation separated. #1448 is not included.
This avoids counting the same roadmap/evidence lines in multiple adjacent
history ranges.

| slice | production | tests | docs/evidence | total |
|---|---:|---:|---:|---:|
| #1445 spellDB | +107 / −104 | +168 / −0 | +225 / −0 | +500 / −104 |
| #1446 THAC0 | +24 / −36 | +94 / −5 | +287 / −0 | +405 / −41 |
| #1447 saving throws | +76 / −74 | +315 / −0 | +335 / −0 | +726 / −74 |
| #1449 `LVL_IMMORT` | +20 / −15 | +53 / −0 | +217 / −0 | +290 / −15 |
| **aggregate** | **+227 / −229** | **+630 / −5** | **+1,064 / −0** | **+1,921 / −234** |

These are churn measurements, not claims that tests or prose are production
deletions. #1448’s runner changes are deliberately absent from the table.

## Validation and reused census checkpoints

The following focused current-main commands all passed:

```bash
export PATH=/usr/local/go/bin:$PATH
export DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle

go test ./pkg/session -run 'TestSpellDBPreservesCompletePreKeyingTable|TestManaCostUsesClassMinimumLevel|TestCast|TestCmdLevels' -count=1
go test ./pkg/combat -run 'TestTHAC0|TestLVLImmort' -count=1
go test ./pkg/game -run 'TestTHAC0|TestNewCharacterTHAC0' -count=1
go test ./pkg/spells -run 'TestSavingThrow' -count=1
go test ./pkg/game -run 'TestSpecFlyExitUp_EntryGatesAndAudience|TestGainExp_MortalCannotReachImmortal' -count=1
go test ./pkg/scripting -run 'TestLVLImmort' -count=1
```

Repository gates on the audit worktree:

| command | result |
|---|---|
| `make fmt` | PASS; no worktree changes |
| `git diff --check` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./...` | PASS |
| `go test ./pkg/game/...` | PASS |
| `golangci-lint run ./...` | PASS; 0 issues |
| `make fidelity-depth` | PASS; 4,811 total / 4,692 proven-or-delegated / 68 blocked / 51 excluded; actionable 4,692/4,760 = 98.6% |
| `make expected-divergences-check` | PASS; expected pins OK; 26 unresolved rows across 10 scenarios |

The expected-divergence command’s unresolved rows are the existing ledger
state, not a new content-red result. No unexpected or flaky content-red was
found.

The following focused current-main commands all passed before this
documentation-only change, as recorded below. The old full-corpus checkpoints
are retained as slice evidence, but are not claimed as the final current-main
census because of #1451:

- spellDB: focused 11-run matrix and full census at `bba25758b`, recorded as
  `940/930/9/1/0/0/0/0`, elapsed `7048.494s`; the spellDB production file is
  unchanged after its implementation checkpoint;
- THAC0: 20 focused oracle runs and full census at `e38120cd3`, log
  `/home/zach/thac0-full-oracle-regression-2026-09-12.log`,
  `940/930/9/1/0/0/0/0`, elapsed `7046.330s`;
- saving throws: 20 focused oracle runs and full census with runner repair at
  `d3871f69f`, log
  `/home/zach/saving-throw-evidence-2026-09-12/full-census-d3871f69f/run.log`,
  `940/930/9/1/0/0/0/0`, elapsed `7038.405s`; and
- `LVL_IMMORT`: 13 focused oracle runs and final full census at `caf62b6f7`,
  log `/home/zach/lvl-immort-evidence-2026-09-12/full-oracle-regression-caf62b6f7.log`,
  `940/930/9/1/0/0/0/0`, elapsed `7044.516s`.

In each tally the fields are `scenarios/passed/expected/unpinnable/stale/
failed/infra/timed_out`. The recorded full censuses exit 2 only because the
existing human-cleared `accuse-noarg-depth` result is `UNPINNABLE`; each has
zero failed, infrastructure, timed-out, and stale results. The final
current-main census is recorded in the 2026-09-13 provenance handoff, after
the documentation checkpoint. The saved historical logs and manifests remain
outside self-cleaning directories.

## Documentation corrections made

The roadmap’s top-level Phase 6.3 status now distinguishes bounded-green
named slices from whole-phase review. The spellDB entry now identifies merged
PR #1445 and its implementation checkpoint instead of describing an unmerged
PR. This audit is corrected by the 2026-09-13 provenance reconciliation,
which records per-target dispositions, original-source excerpts/hashes, the
actual formula call paths, the corrected “weapon-skill” label, the board
boundary, excluded #1448 delta, measured changes, tests, and the fresh
current-main census. Historical slice handoffs remain intact.

Recommendation remains: accept the bounded reconciliation disposition and stop
for human review. See the dated provenance handoff for the fresh current-main
census and the exact next bounded task. Do not merge the documentation PR
automatically and do not start Phase 6.4 or a remaining candidate
implementation from this audit.
