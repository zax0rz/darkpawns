# Dated handoff: Phase 6.3 saving-throw table keying

Date: 2026-09-12
Branch: `glm/modernize-saving-throw-keys`
Implementation checkpoint: `f9e3178d5`
Base: `origin/main` `e54301fb4abfea6bf0c4811111d244c4ae141912e`

## Bounded result

The saving-throw table keying slice is implemented in
`pkg/spells/saving_throws.go`. All 12 class rows use existing named class
identifiers and all 5 category rows use existing `SavingThrowType` constants.
The table remains `[12][5][41]int`; all 2,460 values, zero/default cells,
level-0 sentinel cells, valid readers, invalid-input clamps/fallbacks, NPC
Warrior selection, formula arithmetic, and RNG draw order are unchanged.

The complete frozen pre-keying baseline was captured from the base commit and
compared independently with the C fixture. Both serialized 2,460-cell payloads
have SHA-256
`caf57e8dc021b352253db65c00274194fe1cf1ba393101aa008b1403ad9a6b80`.
The detailed audit, tests, source citations, focused commands, and logs are
in [`2026-09-12-saving-throws`](../../evidence/2026-09-12-saving-throws/README.md).

Focused live coverage is complete: poison 12/12 seeds, sleep 5/5 seeds, and
medusa 3/3 seeds; 20 unique runs, missing 0, duplicates 0, and all 20 with no
normalized divergence. Seed-1 `--show-oracle` blocks reached the intended
save-gated reader paths.

## Validation boundary

Formatting, build, vet, full Go tests, game tests, lint, `make fidelity-depth`,
and `make expected-divergences-check` pass. The required full
`make oracle-regression` was run at `f9e3178d5` with the current 940-scenario
census, seed 1, four workers, and 240-second timeout, but no final census
tally was emitted. `medit-entry-depth` had an infrastructure-shaped first
attempt; its bounded retry produced C's `Specify a mobile VNUM to edit.`,
`Yikes! Stop that, someone will get hurt!`, and `Sorry, there is no zone for
that number!` while Go emits `Huh?!?` for all three. Those retry fingerprints
exactly match the checked-in pinned baseline. The worker's infrastructure
retry path misclassified the pinned status-3 retry as `FAIL`; this is not a
new content divergence and not a saving-table regression. The exact output is
recorded in the evidence README and durable log.

This is outside the saving-throw scope. A one-worker rerun with the same
inputs was then started to avoid the retry race; it reached 18 statuses
(16 pass, 1 expected, 1 permitted unpinnable) before being stopped because
serial execution was impractically slow, with outer exit 141. The smallest
next action is to fix or validate the runner's infrastructure-retry
classification outside this PR, then rerun the full corpus with bounded
parallelism. This handoff does not claim a final all-green full-census tally.

## Phase boundary

This closes only the saving-throw table-keying slice for Phase 6.3. It does
not claim `LVL_IMMORT`, spellDB, THAC0, literal ladders, spell dispatch,
saving-throw formulas, broader combat changes, or all of Phase 6.3.

No `src/` or `darkpawns-c-oracle/` file was edited, and no save-file format or
driver/scenario input was changed.
