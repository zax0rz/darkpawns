# Phase 6.3 spellDB keying handoff — 2026-09-11

## Boundary

This handoff covers only the spellDB table-keying slice. It does not claim
THAC0, saving throws, `LVL_IMMORT`, other literal ladders, spell-dispatch
consolidation, Phase 6.1 equipment-map proof, Phase 6.2 inventory work, or
whole-Phase 6.3 completion.

The isolated worktree is `/home/zach/darkpawns-spelldb`, branch
`glm/modernize-spelldb-keys`, based on `origin/main` at `e961be906` (PR #1444
merged). The primary checkout’s uncommitted `docs/specs/tui-setup-wizard.md`
change was not touched.

## Audit result

The roadmap’s historical “spellDB 509 L” description was not treated as a
current representation claim. The current Go table was already
`map[int]*spellData` with 103 records. The actual positional risks were:

1. Unkeyed `spellData` literals could silently move values if field order
   changed.
2. Raw numeric map keys could silently drift from the named spell identity.
3. `SpellNum` duplicates the map-key identity and is read by `manaCost`; that
   coupling is retained rather than widened into a representation redesign.

There is no `spellDB` iteration or contiguous-range consumer. The readers are
the cast command lookup, `manaCost`, and `grantClassSpells`’ legacy
`SpellMap` population. The map container, lookup absence behavior, and
iteration behavior elsewhere are therefore unchanged.

## Change

`pkg/session/cast_cmds.go` now uses existing `pkg/spells` named constants for
all 103 keys and named fields for every production `spellData` literal. No
value, key, field type, initialization effect, formula, dispatch path, or
sentinel was corrected or invented. The existing 54/calliope record is
explicitly retained and marked because C’s ID 54 is lycanthropy; that is a
separate fidelity issue, not a refactor correction.

`pkg/session/spell_db_test.go` adds an independent frozen pre-keying fixture
and exhaustive comparison over all 131 indices (`0..130`). It verifies 103
present records, 28 absent/default positions (`0`, `103`, `104`, `106..130`),
all six fields, the zero `MinLevel[12]` defaults, and no out-of-range keys.
The fixture does not import production spell constants or copy the new
initializer’s keys, so it can detect a shifted entry, wrong key, omission,
field/default change, or sentinel change.

## C boundary

The C ID declarations and `MAX_SPELLS` are in `src/spells.h:65-175`; the
catalog is `src/spell_parser.c:51-209`; the all-slots initialization and
`spello()`/`unused_spell()` meanings are `src/spell_parser.c:1148-1178` and
`:1225-1232`; C mana consumption is `:257-266`. The Go-before/Go-after test
proves preservation only. The existing C/Go spell-info golden test supplies
the independent metadata comparison. The known ID-54 discrepancy and legacy
Go `Name` label differences remain documented rather than silently changed.

## Evidence

See [`2026-09-11-spelldb evidence`](../evidence/2026-09-11-spelldb/README.md)
for checksums, the complete census, reader coverage, focused seeds, and gate
results. Focused cast proof passed for `cast-depth@1,2,3,5,8`,
`cast-gating@1`, and `cast-mob-innate@1,2,3,5,8`; seed-1 runs were inspected
with `--show-oracle`.

Final full-corpus counts, exact final commit, and PR URL are added to the
evidence file before the branch is handed to human review.
