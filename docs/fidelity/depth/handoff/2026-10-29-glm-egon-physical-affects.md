# Worn AGE / CHAR_WEIGHT / CHAR_HEIGHT — 2026-10-29 (GLM)

This round implements the three physical APPLY locations #2216 inventoried
but left unimplemented: location 9 shifts `player.time.birth` by
`modifier * SECS_PER_MUD_YEAR`, location 10 adds to weight, location 11 adds
to height (`src/handler.c:160-166`).

## Storage-boundary semantics established from `src/`

C mutates `player.weight`, `player.height`, and `player.time.birth` **in
place**: `equip_char`/`unequip_char` call `affect_modify_ar` per item
(`src/handler.c:744,788`), timed affects go through the same switch via
`affect_modify` (`src/handler.c:280-292`), and `affect_total`'s remove+apply
passes cancel each other for these fields because they live in player data,
not `aff_abils` (`src/handler.c:314-345`). `char_to_store` unwinds every
modifier before writing — unequip all, strip affected spells — and reapplies
afterwards (`src/db.c:2531-2637`), so the save file always carries the BASE
values and `Crash_load`'s re-equips rebuild the effective values without
double-counting. The Go port keeps `player.Height/Weight/Birth` as that base
(they were already the serialization contract) and folds the current
worn/timed/tattoo modifiers on demand in the getters — equivalent to C's
in-place value at every boundary and drift-free across repeated rebuilds.

Wrapping follows C's storage: object-affect and timed-affect modifiers are
`sbyte` (`src/structs.h:736-739,1050`; the mobile path already wraps with
`mobileSignedPoint`), and weight/height are `ubyte` (`src/structs.h:885-886`)
— `250+30 → 24`, `10−20 → 246`, no plausible-range clamping anywhere.

## Proof

- Oracle `equipment-physical-affects` (green, seeds 1-3, draw parity MATCH):
  obj 3126 medallion (AGE+10) moves the score line `Age: 17` → `Age: 27`;
  with obj 20301 gauntlets (HEIGHT+3) and obj 19515 bow (WEIGHT−10) also
  worn, `cast 'identify'` self reports `27 years`, `Height 173 cm`,
  `Weight 115 pounds`; `remove medallion` unwinds the age; repeated `score`
  between changes is byte-stable. Revert-red: with only the fix reverted the
  scenario diverges on the first post-wear score (C prints `Age: 27`); the
  red and green transcripts are retained under
  `oracle-runs/2026-10-29/glm-egon-physical/gates/`.
- Units (`pkg/game/player_physical_test.go`): equip/remove for all three
  locations with repeated rebuilds; positive and negative modifiers; ubyte
  wrap in both directions; multiple same-location items with order-independent
  removal; timed-affect folding and expiry; and the serialization boundary —
  `SerializePlayer` emits the base, `DeserializePlayer` restores it, and
  re-equipping reproduces the effective values exactly once. Revert-red fails
  the suite; the green and red runs are retained in the same gates directory.

## World scan (regenerated this round, not copied)

`lib/world/obj/*.obj` standalone A records: location 9 = 6 (e.g.
`lib/world/obj/31.obj:259` → vnum 3126), location 10 = 10 (e.g.
`lib/world/obj/195.obj:156` → vnum 19515), location 11 = 3 (e.g.
`lib/world/obj/203.obj:9` → vnum 20301). All three files are in the shipped
obj index, so the fixtures spawn real prototypes; the counts describe records
in files, not live placements.

## Boundaries and adjacent findings (not fixed here)

- Mobile equipment already applies these locations in place with the same
  wrap (`pkg/game/mobile_equipment.go:167-169`); no Go consumer exposes a
  mobile's weight/height/age today — C's identify refuses NPC victims
  ("The magicks fail horribly!", `src/spells.c:594-597`), and Go's identify
  has the same NPC-fail arm, so the mobile side has no observable surface to
  prove beyond what `mob.conjured`/mobile-equipment cases already cover.
- C's `make_corpse` sets the corpse object's weight to
  `GET_WEIGHT(ch) + IS_CARRYING_W(ch)` (`src/fight.c:391`); Go's
  `makeCorpse` never assigns it. Corpses are not takeable, so no player
  command is known to print it — recorded as a proof gap, not a confirmed
  divergence.
- `Player.GetStat("Age")` computes an invented "100 real seconds = 1 MUD
  year" age. No live caller reaches the "Age" arm (verified by grep); it is
  unreachable today but would be R4 if wired up.
- The apply-modifier `sbyte` wrap at parse: the mobile path wraps modifiers
  before use, and this round's player fold wraps too; an object file
  carrying a modifier outside [-128,127] would need the parser-level audit
  (same sbyte class as the `set` hitroll/damroll finding in the group
  rewards round).
