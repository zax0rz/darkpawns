# Spells test audit at c85ab41be

## Prediction before reading tests

Of 191 functions, I predict 147 R, 24 F, 14 C, and 6 D. I expect 75% of F/C/D to predate the oracle.

Contamination: none

## Baseline and counts

Pinned revision `c85ab41be`. Fresh `go test ./pkg/spells/...` passed in 0.011s. All 30 test files and 191 top-level functions are represented. Mixed judgments use subtest rows. The package has 7,126 test lines and 6,062 production lines. CI runs the suite through `.github/workflows/ci.yml:122`.

| Unit | R | F | C | D | Mixed |
|---|---:|---:|---:|---:|---:|
| Function prediction | 147 | 24 | 14 | 6 | 0 |
| Functions observed | 129 | 18 | 26 | 13 | 5 |
| Ledger rows | 187 | 26 | 27 | 13 | — |
| Before July 12, ledger rows | 89 | 17 | 23 | 9 | — |
| July 12 onward, ledger rows | 98 | 9 | 4 | 4 | — |

Four functions mix R/F and one mixes R/C. 66 of 253 rows (26.1%) are F/C/D. Of those, 49 (74.2%) predate the oracle, against my 75% guess and the brief's 85% guess. These are row counts and subtests of the same function share attribution.

Junk counts (multiple per row) are J1 2, J2 1, J3 19, J4 30, J5 14, J6 0, J7 21, J8 2, J9 4, J10 15. The J7 rows keep real branch proof while recording mutable RNG overrides used only by tests. Attribution by row is unknown 114, codex 73, glm 29, claude 17, kimi 15, sol 3, deepseek 2, mimo 0. Git identified all introducing commits. Unknown means the allowed PR branch and trailers did not identify an agent.

The mutation ledger has 38 executed test observations, 26 green and 12 red. It includes repaired-C mutations that are expected to go red, alongside surviving regressions. Every changed file was restored byte for byte. The package baseline and focused Go unit tests supplied execution evidence. C expectations came from source review.

## Three clear failures of proof

- `pkg/spells/spell_helpers_test.go:356`, `TestCast_RoutesToCallMagic`. A complete no-op stayed green. The fixture passes an invalid spell, ignores the return, and only checks for silence. The introducing commit `53b53c286` calls itself “cover spell helper functions and MagCreations” and the associated branch is `kimi/python-coverage-2026-07-11` (PR 133). That evidence points to coverage expansion as the motivation, with a fixture too weak to detect loss of dispatch.
- `pkg/spells/affect_messages_golden_test.go:280`, `TestMagAffectsMessages_RoomAudience`. The witness's correct line returns before the victim check. A mutation that delivered the rendered bytes to the victim stayed green. Commit `6d1390ce5` (PR 638) describes fixing affect effect messages for object magic. This looks like a genuine byte-regression check with an audience assertion accidentally placed behind the success return.
- `pkg/spells/saving_throws_table_test.go:159`, `TestSavingThrowBeforeKeyingChecksum`. The checksum covers a frozen test fixture, so changing all production lookups stayed green while the C golden went red. Commit `f9e3178d5` (PR 1447) explicitly keys the table by class and category. The checksum appears to have been added to protect the migration baseline. Its continuing coverage is fixture integrity.

## Saving-throw golden provenance

`git log -p -- pkg/spells/saving_throws_golden_test.go` contains only `71d74913b` (June 23, PR 36, Git author BRENDA69, agent unknown). The commit and its original comment identify the C initializer as the transcription source. I parsed all 60 initializer rows at `src/magic.c:83`, expanded C's implicit zero fill, and matched all 2,460 cells including Cleric and Avatar paralysis level 2 at 59. The first oracle commit is `ba05d9906` on July 12. There was no later test-file correction. The literal values and history establish a C-faithful fixture. They do not establish how the author performed the transcription.

## Expectations or asserted representations that differ from C

| Test | C | Go and scope |
|---|---|---|
| `TestIsEvil`, `TestIsGood` | `src/utils.h:454–455` uses ±350 | `pkg/spells/damage_spells.go:444,452` uses sign. Tests require -1/+1 to qualify. Live Dispel and affect logic use these helpers. Both tests went red with C thresholds restored. |
| `TestCastMirrorImageOutputAndRoomAudience` | `src/spells.c:1136` creates a level-one clone; `src/new_cmds2.c:592` assigns the requested level | `pkg/spells/affect_spells.go:3917` passes cast level. Fixture requires 22. Restoring one went red. |
| `TestCastIdentifyObjectOutputBranches`, wand and staff | `src/spells.c:533–534` has a trailing space and appends another leading space | `pkg/spells/affect_spells.go:2541` uses one. Both rows went red with two. This is the source-intended reading under R1a, with overlapping C `sprintf` and no transcript verification. |
| `TestMagAffects_Metalskin` and the Metalskin row of `TestMagAffectsApply_GoldenAgainstCSource` | `src/magic.c:1308–1312` combines AC, duration and bit in one record | `pkg/spells/affect_spells.go:416` splits the record. Both tests went red with one record. Effective AC arithmetic agrees; record count and rendering implications need separate proof. |
| Greater and Lesser Perception rows of `TestMagAffectsApply_GoldenAgainstCSource` | `src/magic.c:1344,1354` assigns `APPLY_SPELL` to the Sense Life/Infravision slot | `pkg/spells/affect_spells.go:429,433` stores `ApplyNone`. C's zero modifier makes the special apply inert at `src/handler.c:240`; visible impact is open. Both tests went red with the C locations. |
| Lay Hands rows of `TestMagPointsFormula_GoldenAgainstCSource` | `src/magic.c:1823` uses caster body level for dice sides | `pkg/spells/affect_spells.go:689,804` uses the supplied cast level. Equal-level formulas agree. Changing only the production dispatch argument stayed green; the mismatch is at the live call boundary when levels differ. |
| `TestCheckReagents_ItemConsumedOnExactShortDescMatch` | `src/magic.c:548` matches prototype vnum | `pkg/spells/affect_spells.go:1652` delegates name selection to `pkg/game/spell_reagents.go:6`. Fixture selects by short description itself. Changing the live consumer's requested name stayed green. |
| `TestCheckReagents_BonusScalesWithLevel` | `src/magic.c:481` returns Boolean availability; missile applies a full level at `src/magic.c:621` | `pkg/spells/affect_spells.go:1676` pins an internal half-level convention. The observed live callers use truthiness or the flat-one override, so this test mismatch alone does not establish a damage bug. |
| `TestAttackTypes_TableSize`, `TestAttackTypes_IndexZeroIsEmpty` | `src/fight.c:84–100` defines 15 weapon verbs beginning with hit/hits | `pkg/spells/spell_info.go:153–155` has an oversized, differently indexed table. The tests pin a minimum of 20 and an empty index zero. The live weapon table is separate. |

Additional gaps masked by fixtures include Meteor Swarm and Circle of Summoning using the supplied cast level where C reads the caster body level (`src/spells.c:1094,1018`; `pkg/spells/affect_spells.go:2995,2097)). The tests pass equal levels. Hellfire's knockdown assignment uses 5 at `pkg/spells/affect_spells.go:3144`, while C assigns `POS_SITTING` (6) at `src/spells.c:737` and `src/structs.h:215`. The damage fixtures do not assert this position.

## Duplicate implementations and live consumers

- **Mana metadata.** The golden suite checks the `pkg/spells` template table, which supplies live positions, target flags and routines via `pkg/session/cast_cmds.go:219` and `pkg/spells/call_magic.go:69`. Actual cast and practice mana use `game.ManaCost` and `game.SpellManaTable` at `pkg/game/spell_mana.go:144`, called by `pkg/session/cast_cmds.go:262` and `pkg/game/practice.go:137`. The golden suite proves only its own mana copy. The game table retains a Calliope record under Lycanthropy as well as the proper Calliope record.
- **Weapon verbs.** `spells.AttackTypes` is consumed only by the uncalled `MagAttackModifier` at `pkg/spells/call_magic.go:259`. `combat.AttackHitTexts` drives live output at `pkg/combat/fight_core.go:805–811` and wizard stats at `pkg/session/wiz_stats.go:179–181`. Breaking the live first verb pair left all three spells table tests green.
- **Alignment.** Player predicates in `pkg/game/player_affects.go:193,196` and mob predicates in `pkg/game/mobact.go:96,103` use ±350. Spell sign predicates in `pkg/spells/damage_spells.go:441,449` are live for Dispel and affect paths. Enchant also has inline sign checks, while Divine Intervention uses ±350. These are separate call paths.
- **Damage orchestration.** Standard spell damage and breath use `spells.inflictDamage` at `pkg/spells/damage_spells.go:320`. Manual Hellfire and Meteor Swarm use `World.DoSpellDamage` at `pkg/game/damage_stubs.go:138`, entering `combat.TakeDamageAfterGate`. Skill-owned damage has its own tail at `pkg/game/damage_stubs.go:164`. The paths share modifiers and pre-message logic but maintain separate orchestration. The legacy private `World.doDamage` at line 361 had no production call site.
- **Act delivery.** Spell-room and chant fallback paths are independently tested with small stubs. Live `game.World` exposes `SaySpellDeliver`, `SpellActToRoom`, `SpellCreatedObjectAct` and other Act adapters in `pkg/game/say_spell_act.go`; it also exposes `SpellObjectAct` at `pkg/game/world_bridge.go:1133`. `SaySpell` in `pkg/spells/say_spell.go:85` chooses the world adapter before its own recipient loop. Small-mock tests establish fallback routing or the handoff, depending on which interface they implement.
- **Reagent lookup.** `checkReagents` prefers the live name-based `Player.ConsumeSpellReagent` over the inventory-interface fallback used by its unit tests. `Player.ConsumeSpellReagentVNum` in the same game file is independently live for Sleep, Curse and Blindness/Smokescreen. `CheckSavingThrow` itself has one table/roll implementation. The identically named player and mob methods expose save modifiers, not a second threshold computation.

## Open questions

- The audit separates direct C source disagreement from confirmed transcript drift. Charged-item Identify relies on the R1a source-intent reading; affect representation changes need a player-visible trace before treating all of them as gameplay regressions.
- Same-seed threshold tests, correct live RNG bounds and exact body-vs-cast-level fixtures remain gaps. Seven mutable RNG overrides simplify branch tests without independently exercising production draw order.
- Sparse evidence about older authors leaves 114 rows unattributed by agent. Test authorship is assigned to the introducing function commit, including for subsequently edited subtests. The ledger does not assign individual expectations to later authors.
