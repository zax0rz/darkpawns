# Dated handoff: Phase 6.3 saving-throw table keying

Date: 2026-09-12
Branch: `glm/modernize-saving-throw-keys`
Implementation checkpoint: `f9e3178d5`
Base: `origin/main` `e54301fb4abfea6bf0c4811111d244c4ae141912e`
Runner integration checkpoint: `d3871f69f`, integrating merged PR #1448
(`d580f1389`)

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

## Validation and census

Formatting, build, vet, full Go tests, game tests, lint, `make fidelity-depth`,
and `make expected-divergences-check` pass. After integrating the merged
runner fix, the complete frozen-input parallel census was run at
`d3871f69f` with seed 1, four workers, and a 240-second timeout. Its complete
tally was:

```text
scenarios=940 passed=930 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The aggregate exit was 2 solely because the existing exit ordering reports
the human-cleared `accuse-noarg-depth` UNPINNABLE baseline. Reconciliation
found 940 unique scenario names, no missing or unexpected scenarios, and one
duplicate display of that UNPINNABLE result by the aggregate printer; it was
not a duplicate execution. The durable run log, frozen-input manifest, and
reconciliation are under
`/home/zach/saving-throw-evidence-2026-09-12/full-census-d3871f69f/`.
`medit-entry-depth` is now correctly classified as pinned EXPECTED after its
infrastructure-shaped attempt, while the other observed recoveries ended as
PASS within the worker's shared three-attempt budget.

This runner integration is evidence for the census only; it does not broaden
the saving-throw slice. The runner fix is already merged in PR #1448 and no
runner source is changed by this saving-throw PR.

## Phase boundary

This closes only the saving-throw table-keying slice for Phase 6.3. It does
not claim `LVL_IMMORT`, spellDB, THAC0, literal ladders, spell dispatch,
saving-throw formulas, broader combat changes, or all of Phase 6.3.

No `src/` or `darkpawns-c-oracle/` file was edited, and no save-file format or
driver/scenario input was changed.
