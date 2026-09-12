# Saving-throw table-keying evidence

Date: 2026-09-12
Branch: `glm/modernize-saving-throw-keys`
Implementation checkpoint: `f9e3178d5`
Starting point: `origin/main` at `e54301fb4abfea6bf0c4811111d244c4ae141912e`

## Result and boundary

This is the bounded Phase 6.3 saving-throw table-keying slice only. The Go
initializer in `pkg/spells/saving_throws.go` keeps the exact
`[12][5][41]int` container, dimensions, element type, initialization, and
reader behavior, while naming the existing class and saving-category keys.
The only production logic adjustment is spelling the unchanged NPC Warrior
override with the existing named class constant. No values, formulas, spell
dispatch, THAC0, `LVL_IMMORT`, save format, or broader combat behavior changed.

The implementation proof is green. The required full `make oracle-regression`
was started at the implementation checkpoint but did not emit a final census
tally. The unrelated `medit-entry-depth` case had an infrastructure-shaped
first attempt; its bounded retry produced C output for all three probes while
Go emits `Huh?!?`. The three retry fingerprints are:

```text
medit              134cb9c489e39d317ab67587e401d50b8f0382d5ceb9cfdd3050ddaceaaa2eda
medit abc          5d8d92b3c45a9d7f753223206d917b47609be74601ef7c19ed48866c87d0abf9
medit 999999       0f8807f28888522a9809fedd3386d648b22fa17f977fbc3d9cf220f3acb576d3
```

Those fingerprints exactly match the checked-in `medit-entry-depth` entries in
`cmd/dp-oracle-diff/expected_divergence_pins.tsv`. The regression worker's
infrastructure-retry branch does not re-enter its pinned-divergence classifier
when the retry returns status 3, so it printed `FAIL` instead of `EXPECTED`.
This is a runner-classification gap, not a new unpinned content divergence or
a saving-table regression. The run was stopped after that bounded retry and
was not restarted. The durable partial log is
`/home/zach/saving-throw-evidence-2026-09-12/oracle-regression.log`.

A completed 940-scenario census is recorded in the immediately preceding
THAC0 handoff at source checkpoint `e38120cd3`, an ancestor of current
`origin/main`; current main adds only the THAC0 merge and documentation after
that checkpoint. That historical result is context, not a final tally for
this invocation. A one-worker rerun with the same 940 scenarios, seed, timeout,
and pins was then started to avoid the retry-classification race. It produced
18 status lines (16 pass, 1 expected, and the permitted 1 unpinnable) before
being stopped because serial execution was impractically slow; its outer exit
was 141. It is also not a final full-census tally. The durable partial log is
`/home/zach/saving-throw-evidence-2026-09-12/oracle-regression-serial.log`.

## Audit

The sole authoritative runtime table is `savingThrowTable` in
`pkg/spells/saving_throws.go:27-129`. It is initialized once and has no
runtime mutation. Its actual readers are:

- `GetSavingThrow` at `pkg/spells/saving_throws.go:139-161`, which clamps
  levels to 0..40, defaults invalid classes to class 0, and defaults invalid
  save categories to `SaveSpell`.
- `CheckSavingThrow` at `pkg/spells/saving_throws.go:164-216`, which obtains
  the character class/level and modifier, overrides NPCs to the Warrior row,
  preserves `MAX(1, save) < number(0,99)` semantics, and consumes one d100
  draw in the existing order.

`pkg/game.Player.SavingThrows` at `pkg/game/player.go:106-108` is not a second
base table: it stores per-character modifiers. `Player.GetSavingThrow` and
`SetSavingThrow` at `pkg/game/player_stats.go:398-419`, plus affect application
in `pkg/engine/affect_helpers.go:100-116`, supply the `GET_SAVE`-equivalent
modifier path. The special-procedure cleanup at
`pkg/game/spec_procs2.go:789-793` mutates those per-character modifiers. The
test-only `savingThrowGolden` fixture is a separate C transcription, not
runtime data.

The five category constants are in `pkg/spells/spell_info.go`:
`SaveParalysis=0`, `SaveRodStaff=1`, `SavePetrify=2`, `SaveBreath=3`, and
`SaveSpell=4` (`SaveCount=5`). The twelve class identities are the existing
combat constants: Mage 0, Cleric 1, Thief 2, Warrior 3, Magus 4, Avatar 5,
Assassin 6, Paladin 7, Ninja 8, Psionic 9, Ranger 10, Mystic 11. Levels are
the contiguous stored indices 0..40: level 0 is the stored 90 sentinel;
non-Warrior-family rows retain their stored zero/default cells at levels
31..40; Warrior, Paladin, and Ranger retain their nonzero high-level cells.

The C declarations and call path are:

- `src/magic.c:83-404`: `saving_throws[NUM_CLASSES][5][41]` and all rows.
- `src/magic.c:407-426`: `mag_savingthrow`, including the NPC Warrior
  selection, category/level indexing, modifier, clamp, and d100 draw.
- `src/structs.h:113-128`: class enum order and `NUM_CLASSES=12`.
- `src/spells.h:287-291`: saving-category identities.
- `src/structs.h:934-935`, `src/utils.h:325`, and `src/handler.c:190-203`:
  per-character `apply_saving_throw[5]` modifiers, distinct from the base
  table and consumed through `GET_SAVE`.
- Go spell readers include `pkg/spells/call_magic.go:84-93,208-212`,
  `pkg/spells/damage_spells.go:55-63,107-113`, and the special-procedure
  reader `pkg/game/spec_procs2.go:1667`.

The positional risk removed is silent row drift: inserting or reordering a
class or category row could preserve compilation while routing a save lookup
to another class/category. Existing identifiers now sit on both dimensions;
the level dimension remains deliberately positional because its contiguous
0..40 indexing is part of the reader contract.

## Independent preservation proof

Before editing, the complete 2,460-cell Go payload (12 classes × 5 categories
× 41 levels) was frozen from source commit
`e54301fb4abfea6bf0c4811111d244c4ae141912e`. The base source-file SHA-256 was
`a7403774dfe2daf01211ee6570756b5a982e96386bb01b8d934e096720da0cc8`; the
serialized numeric payload SHA-256 was
`caf57e8dc021b352253db65c00274194fe1cf1ba393101aa008b1403ad9a6b80`.

The new `pkg/spells/saving_throws_table_test.go` stores that complete numeric
fixture before the production table is read. Its exhaustive direct-cell test
checks all 2,460 post-keying cells, including level-zero sentinels and every
stored zero/default. A separate test compares the frozen pre-keying Go fixture
with the existing independently transcribed C fixture, and a checksum test
protects the frozen fixture itself. The C fixture's serialized payload has the
same SHA-256, so the two claims remain distinct: the post-keying Go data is
identical to the pre-keying Go data, and the frozen pre-keying data is
identical to the C data.

The test suite also independently checks all named class/category identities,
valid accessor endpoints, invalid class/category/level fallbacks, direct
stored sentinels/defaults, and the NPC-to-Warrior override using the same
reset dprng stream. Existing tests retain unsupported-character behavior and
the save formula/draw-boundary assertions. The C source's Cleric PARA level-2
value of 59 is preserved exactly; no discrepancy was found in the complete
valid table comparison and no C behavior was silently corrected.

These tests detect:

- swapped class/category rows: every independent `[class][category][level]`
  cell is compared;
- shifted or omitted level entries: all 41 levels per row are compared,
  including level 0 and high-level zeros;
- wrong named keys: class/category identity assertions plus all-cell values;
- changed defaults, sentinels, or dimensions: direct-cell fixture, sentinel
  checks, and the explicitly typed production array;
- changed fallback behavior: invalid class, category, and level cases are
  checked against the pre-keying fixture, while valid endpoint and NPC paths
  are exercised separately.

## Focused oracle proof

The existing live readers were exercised with current manifest vehicles:

- `quaff-poison-save`, seeds 1..12: object `4399`, `SAVING_ROD` potion path;
- `sleep-spell-depth`, seeds 1, 2, 3, 5, 8: cast sleep,
  `SAVING_SPELL`, failed-save `AFF_SLEEP` and wake path;
- `spec-proc-medusa`, seeds 1, 2, 8: medusa `SAVING_PETRI` actor/audience path.

Focused result census: 20 runs, 20 unique scenario/seed pairs, missing 0,
duplicates 0, failed 0, infrastructure 0, timed out 0, stale 0, and 20/20
`result: no normalized divergence`. The exact per-run ledger is in
`/home/zach/saving-throw-evidence-2026-09-12/matrix/summary.tsv`.

Seed-1 `--show-oracle` blocks were inspected for all three vehicles. They
included the poison fail text `You feel very sick.`, sleep's sand cast and
`Sleeptarget goes to sleep.` plus `You can't wake him up!`, and medusa's
actor-only horror text, bystander stone text, and death cry. These are
reader-path checks; matching final spell text is not being used as a proxy for
threshold arithmetic or draw parity.

The focused command shape was:

```bash
export PATH=/usr/local/go/bin:$PATH
export DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 1
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 2
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 3
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 4
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 5
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 6
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 7
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 8
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 9
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 10
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 11
go run ./cmd/dp-oracle-diff --scenario quaff-poison-save --seed 12
go run ./cmd/dp-oracle-diff --scenario sleep-spell-depth --seed 1
go run ./cmd/dp-oracle-diff --scenario sleep-spell-depth --seed 2
go run ./cmd/dp-oracle-diff --scenario sleep-spell-depth --seed 3
go run ./cmd/dp-oracle-diff --scenario sleep-spell-depth --seed 5
go run ./cmd/dp-oracle-diff --scenario sleep-spell-depth --seed 8
go run ./cmd/dp-oracle-diff --scenario spec-proc-medusa --seed 1
go run ./cmd/dp-oracle-diff --scenario spec-proc-medusa --seed 2
go run ./cmd/dp-oracle-diff --scenario spec-proc-medusa --seed 8
```

## Validation

At implementation checkpoint `f9e3178d5`:

- `gofumpt -l .` — pass
- `go build ./...` — pass
- `go vet ./...` — pass
- `go test ./...` — pass
- `go test ./pkg/game/...` — pass
- `golangci-lint run ./...` — `0 issues`
- `make fidelity-depth` — pass, current corpus 4,811 total / 4,692
  proven-delegated / 68 blocked / 51 excluded (98.6% actionable)
- `make expected-divergences-check` — pass; 26 ledger rows and pins OK
- `make oracle-regression` — no final counters; `medit-entry-depth` first
  hit an infrastructure-shaped failure, then its retry matched the existing
  pinned fingerprints but was misclassified by the worker retry path.

The next action is to fix or otherwise validate that infrastructure-retry
classification outside this scoped PR, then rerun `make oracle-regression`
from this branch with bounded parallelism. No saving-throw table change is
indicated by the runner gap.
