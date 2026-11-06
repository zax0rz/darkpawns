# pkg/spells test audit — notes (agent: deepseek)

Identity: I am the `deepseek` agent. In this repo the agent the briefs call
"DeepSeek" is the Cline CLI: the DeepSeek-designated combat fix brief
(`/home/zach/dp-research/BRIEF-test-fix-phase-combat.md`) told that agent to use
`~/wt/deepseek-test-fix` + branch `cline/combat-test-fix`, and the resulting
commit `e7632db6b` carries `Co-Authored-By: Cline <noreply@cline.bot>`. The
brief's own attribution table maps `cline/` -> deepseek. I ran under `cline-cli`.

## Guesses, written before reading any test

- R 62% (~118), F 14% (~27), C 13% (~25), D 11% (~21); F+C+D ~38%.
- >= 80% of the F/C/D marks on tests written before the oracle.

Claude's recorded guess: ~78% R, F+C+D 22%; >= 85% of F/C/D pre-oracle.

## Actual vs guess

| | guessed (me) | guessed (Claude) | actual |
|---|---|---|---|
| R | 62% | 78% | **164 (85.9%)** |
| F | 14% | — | **6 (3.1%)** |
| C | 13% | — | **20 (10.5%)** |
| D | 11% | — | **1 (0.5%)** |
| F+C+D | 38% | 22% | **27 (14.1%)** |

Both guesses overestimated junk, mine by ~2.7x. This package is unusually
C-cited: nearly every behavioural test names a `src/*.c` line in a comment and
asserts exact player bytes, which is the shape R1 and DEPTH_TESTING reward.

Pre-oracle share of F/C/D: **23 of 27 (85%)**. Pre-oracle tests are 85 of 191
functions but hold 23 of the 27 weak marks — matching both predictions and the
combat pilot.

## Baseline

```
ok  github.com/zax0rz/darkpawns/pkg/spells  0.015s
```

All green at `c85ab41be`.

## Counts

Marks: R 164 · F 6 · C 20 · D 1.

Junk codes used (27 rows carry at least one):

| code | n | where |
|---|---|---|
| J4 duplicate contract | 18 | 15 in `saving_throws_test.go`, plus chill touch / bless / armour / summon-flag duplicates of later golden tables |
| J7 test-only seam | 4 | `StartCombat`, `FollowingBody`, `SpellCreatedObjectAct`, `ExecuteSpellCounterattack` |
| J2 assertion cannot fail | 4 | `TestCheckSavingThrow_DoesNotPanic`, `TestRandBool`, `TestCast_RoutesToCallMagic`, `TestCheckPosition` |
| J1 expected from code under test | 4 | `TestSetAndGetSpellInfo`, `TestSetupSpellInfo`, the frozen saving-throw baseline and its checksum |
| J3 name promises unexercised behaviour | 3 | `TestCast_RoutesToCallMagic`, `TestObfuscateSpellName_PoisonMatchesOracle`, `TestCastHellfire_BlocksPeacefulRoom` |
| J10 value disagrees with C / no C citation | 2 | `TestCastMirrorImageOutputAndRoomAudience`, `TestCheckReagents_BonusScalesWithLevel` |
| J5 implementation shape | 1 | `TestSetAndGetSpellInfo` |

By `author_agent` (git only): unknown 123 · codex 44 · claude 17 · glm 4 ·
sol 2 · deepseek 1. The 123 "unknown" are mostly commits pushed straight to
`main` by `Zach Greene`/`zach` with no PR merge and no agent `Co-Authored-By`,
so git records no agent for them (see open questions).

Before/after the oracle (2026-07-12): 85 functions pre-oracle, 106 post.
R 62 pre / 102 post. F+C+D 23 pre / 4 post. The four post-oracle weak marks are
`TestCastMirrorImageOutputAndRoomAudience` (F), `TestObfuscateSpellName_Poison
MatchesOracle` (C), `TestSummonMobFlagPositionsMatchC` (C) and
`TestSetAndGetSpellInfo` (D).

## The three clearest junk examples

1. **`saving_throws_test.go` — 15 tests one later test absorbs.**
   `TestGetSavingThrow_WarriorLevel1Para` … `TestGetSavingThrow_SaveTypeOutOf
   Range` (`saving_throws_test.go:16-125`) each assert one or two cells of the
   table. `TestSavingThrow_GoldenAgainstCSource`
   (`saving_throws_golden_test.go:189`) walks all 12x5x41 = 2460 cells, and
   `TestSavingThrowAccessorEndpointsAndFallbacks`
   (`saving_throws_table_test.go:243`) covers every invalid-input fallback they
   were reaching for. They look like **coverage padding** from the first
   testing push (`8c574d2b5`, 2026-05-04, "test: complete testing push — parser
   fixes, combat formulas, spells, validation"), written before any C-derived
   table existed. All 15 are C (merge); their values are correct.

2. **`TestCastHellfire_BlocksPeacefulRoom` proves only half its name.**
   `area_spells_test.go:64` asserts no damage and an untouched victim but never
   asserts the refusal line C sends (`src/spells.c:706-710`, "This room just
   has such a peaceful, easy feeling..\r\n"). Mutation: rewrite that production
   string at `affect_spells.go:3089` — the test stays **green**. It exists
   because the DP-938 author was chasing "does not damage" (a real bug then) and
   stopped there.

3. **`TestCast_RoutesToCallMagic` never observes routing.**
   `spell_helpers_test.go:356` calls `Cast(caster, nil, -9999, 10, nil)` and
   asserts only that no message was sent; the comment admits "if it does not
   panic … the wrapper is sane". Mutation: make `Cast` return `false` without
   calling `CallMagic` (`spells.go:206`) — the test stays **green**. It fills in
   a thin wrapper's coverage while its name claims a contract the body never
   checks.

Two more, because they are the same shape:
`TestRandBool` (`spell_helpers_test.go:145`) calls `randBool(100)` and discards
the result, and `TestCheckPosition`'s "dead" subtest (`spell_helpers_test.go:
370`) passes a `poserMock` to `checkPosition` then inspects an unrelated
`msgMock`, so "You can't cast spells while dead!" is never asserted. Both stay
green under the corresponding production mutation.

## The saving-throw golden table

- Added 2026-06-23 in `71d74913b` ("tier-2: add golden tests for dex_app,
  saving throws, XP curve, regen"), git author name `BRENDA69`, no PR merge.
- Its header says the table is "transcribed VERBATIM from the original C source
  (src/magic.c: `const byte saving_throws[NUM_CLASSES][5][41]`)". I checked it
  against the C myself: Mages PARA/ROD/PETRI/BREATH/SPELL and Clerics PARA all
  match `src/magic.c:86-118`, including the Cleric PARA level-2 `59` the header
  flags as "possibly a typo" — C really does print `{90, 50, 59, 48, ...}` at
  `src/magic.c:113`.
- **Nothing in the file changed after 71d74913b**: `git diff --stat 71d74913b
  c85ab41be -- pkg/spells/saving_throws_golden_test.go` is empty, so its current
  values are the originally transcribed ones, not post-oracle corrections.
  (`git log --follow` reports an earlier history only because rename detection
  pairs it with `saving_throws.go` — a 71% similarity match on the copied
  table, not a real rename.)
- The values trace to the C, so the test is R despite predating the oracle.

There are three copies of the same 12x5x41 table: the production
`savingThrowTable` (`saving_throws.go:27`, keyed by `combat.Class*`/`Save*`),
`savingThrowGolden` (C transcription) and `savingThrowBeforeKeying` (a frozen
copy of the pre-refactor Go table, from `e54301fb4`). `TestSavingThrowBefore
KeyingMatchesCFixture` is what keeps the Go-frozen copy honest; that is why the
J1 flavour of `TestSavingThrowTablePreservesEveryPreKeyingCell` is tolerable and
the checksum test (`TestSavingThrowBeforeKeyingChecksum`) is redundant.


## Tests whose expected value disagrees with the C (possible game bugs)

1. **`TestCastMirrorImageOutputAndRoomAudience`** —
   `act_spell_outputs_test.go:193`, assertion at `:210`
   (`world.level != caster.level`). It asserts the clone is spawned at the
   **caster's** level (22 in the test).
   - C: `src/spells.c:1136` `create_mobile(ch, MOB_CLONE, 1, FALSE)` — a literal
     **1**, and `create_mobile` writes `GET_LEVEL(mob) = level`
     (`src/new_cmds2.c:592`).
   - Go: `pkg/spells/affect_spells.go:3917` passes the spell's `level`, and
     `pkg/game/world.go:1014 SpawnMobWithLevelI` does `mob.SetLevel(level)`.
   - Mutation proof: pass `1` at `affect_spells.go:3917` (the C value) — the
     test goes **red**, so the test pins the divergent value. A level-22 clone
     vs a level-1 clone is player-visible in combat (`create_mobile` derives
     AC/hit/dam from the level). Flag for Zach as a possible game bug (R1/R4).

2. **`TestCheckReagents_BonusScalesWithLevel`** —
   `checkreagents_test.go:115,121-124` asserts `bonus == level/2` (level 30 ->
   15).
   - C: `has_reagents`/`mag_materials` return a boolean; the reagent bonus is
     **1** where it is used at all (`src/magic.c:481-495`; consumed at
     `src/magic.c:1274` waterwalk and `:1300` metalskin, both `reag = 0/1`).
   - Go: `pkg/spells/affect_spells.go:1667-1671` returns `level / 2`.
   - Today this is **latent, not a live bug**: waterwalk uses `4+reag?20:0`
     (always 20 either way, `src/magic.c:1279`), and metalskin is called with
     the `"flat:1"` override so it returns 1 (`affect_spells.go:66-67`), and
     `consumeReagent` only tests `> 0` (`damage_spells.go:284-286`). But the
     asserted contract is not the C contract, and any future consumer that reads
     the magnitude would inherit the wrong value. No C citation is offered.

I found no other test whose expected value contradicts the C. Everything else I
spot-checked as C-faithful, including: all 27 `magDamageFormula` rows
(`src/magic.c:618-819`), all healing rows (`src/magic.c:1765-1824`), the affect
table (`src/magic.c:863-1430`), `say_spell` and the syllable loop
(`src/spell_parser.c:269-343`, note `$S` is the possessive HSHR via
`src/comm.c:2423-2425`, which the mental-lapse test gets right), hellfire/meteor
/calliope/mirror/divine-int/zen/coc/locate/identify/charm/summon
(`src/spells.c:218-1218`), mag_alter_objs and mag_creations
(`src/magic.c:1890-1999`), and the saving-throw table and roll
(`src/magic.c:83-425`).


## Duplicate production implementations

I looked for a second live copy of spell logic outside `pkg/spells`.

1. **`World.HandleSpellDeath` is an orphaned second death pipeline.**
   `pkg/game/death.go:440-441` still defines `func (w *World) HandleSpellDeath
   (victim any)`, described as "a bridge for the spell system". The spell system
   no longer calls it: `inflictDamage` routes a spell kill through
   `HandleDeath(victim, killer, attackType)` (`damage_spells.go:400-402`), and
   `damage_spells_death_test.go:126` asserts the old bridge is never called.
   A repo-wide grep finds no caller of the `any`-signature method (the
   `HandleSpellDeath` in `world_scriptable.go` and `pkg/scripting/types.go` is a
   different, string-based Lua seam). **Live copy: `HandleDeath`.**
   `HandleSpellDeath(victim any)` looks like dead code the deadcode ratchet does
   not see because it is an exported method.
2. **`checkReagents` / `consumeReagent` / `consumeSpellReagentVNum` (spells) vs
   `Player.ConsumeSpellReagent` / `Player.ConsumeSpellReagentVNum` (game).**
   Two layers of reagent handling. The **live** inventory mutation is
   `pkg/game/spell_reagents.go`; the `pkg/spells` functions are dispatch-side
   wrappers plus the `level/2` bonus (see the disagreement above).
3. **`SaySpell`** exists only in `pkg/spells/say_spell.go`; `pkg/game` holds the
   delivery seam `World.SaySpellDeliver` / `SpellActToRoom` etc.
   (`pkg/game/say_spell_act.go`). Not a second implementation, but note
   `pkg/spells/say_spell.go` also carries its own room-iteration fallbacks
   (`sendAffectRoomFallback`, `sendToRoom`) used when the world lacks the seam —
   a second delivery path that only tests exercise.
4. **Trivial utility duplicates:** `dice` is defined in `pkg/spells/call_magic
   .go:270`, `pkg/game/weather.go:591` and again as `diceRoll` in
   `pkg/game/damage_stubs.go:468`; `boolToInt` exists in both `pkg/spells
   /affect_spells.go:1461` and `pkg/game/item_consumable.go:526`. All feed the
   same `dprng` draws, so they are cosmetic, not fidelity risk.
5. **Test-side table triplication:** the saving-throw table exists three times
   (production + two test copies) — see above. The saving-throw fixture pair is
   the only place where a J1-shaped test survives, and only because a cross-check
   against the C transcription exists.

I did **not** find a second copy of `mag_damage`, `mag_affects`, `mag_points`,
`castHellfire`, `ObfuscateSpellName` or the damage formulas outside
`pkg/spells`; `pkg/combat` owns the shared `ApplyDamageModifiers` /
`UpdatePositionAfterDamage` the spell path calls into, which is a single
implementation.

## Open questions

- **Agent attribution is mostly unknown (123/191).** Most commits that added
  these tests were pushed directly to `main` from `Zach Greene`/`zach` with no
  merge commit and no agent `Co-Authored-By`, so the brief's git-only method
  cannot name the agent. If the private transcripts may be consulted later, the
  "unknown" bucket is where the answer lives. I did not open them.
- **Is the mirror-image level a bug or an unrecorded decision?** No test, comment
  or C citation defends passing the caster's level, and C is explicit (1136).
  Worth a Linear ticket either way.
- **`SpellSilkenMissile = 200`** is declared "DP original — not in C spell_parser
  spello table" (`spells.go:116`) and is dispatched from `ExecuteManualSpell`
  (`affect_spells.go:1387`), but no `spello()` entry registers its routines, so
  it is unreachable through `CallMagic`. It has no test. Is it a live feature?
- **`pkg/spells/act_delivery.go`** (`canReceiveAct`, 18 lines) is only used by
  the fallback delivery paths; I could not find a non-fallback production caller
  in `pkg/spells`. Possible test-only/unused helper; I did not mark it because it
  is production code, not a test.
- **`TestObfuscateSpellNameMatchesCCompoundAndCharacterPass`** values are asserted
  as "C output". The C loop (`src/spell_parser.c:282-289`) is deterministic and
  the Go reproduces it, but I did not re-derive all 24 strings from the `syls[]`
  table by hand; medium confidence.
- **`spell_info_golden_test.go`** claims "transcribed verbatim from
  src/spell_parser.c mag_assign_spells()". I verified `SpellArmor` (15/30/3,
  PosFighting, RoutineAffects, TarCharRoom) but not all ~90 rows; medium
  confidence.
- **`TestAttackTypes_*`** assert shape (>= 20 entries, index 0 empty, singular !=
  plural) but not the C spellings; I did not read `attack_hit_types` in the C to
  confirm the port's 33 forms match.

