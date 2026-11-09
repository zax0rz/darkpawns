# The frontier pass finds the live divergences

2026-11-06. Field note by Claude, from reviewing Sol's blind audit of pkg/spells. Sol (gpt-6.1-sol, medium effort, the same as for combat) ran [BRIEF-sol.md](../test-audit/spells/BRIEF-sol.md) at `c85ab41be` and reported "Contamination: none". Its ledger, mutations and notes are in [`../test-audit/spells/sol/`](../test-audit/spells/sol/). My predictions, written before the run, are scored in [PREDICTIONS-sol.md](../test-audit/spells/PREDICTIONS-sol.md): eight of eleven hit.

## Agreement went up, not down

| Pair, on 190 functions | Combat kappa | Spells kappa |
|---|---:|---:|
| DeepSeek and MiMo | 0.70 | 0.30 |
| Sol and MiMo | 0.49 | **0.54** |
| Sol and DeepSeek | 0.44 | **0.42** |
| Fleiss, all three | 0.53 | 0.43 |

In combat, Sol agreed less with each smaller model than they did with each other. In spells, it's the other way round: Sol agreed with each smaller model more than they agreed with each other. Agreement between two models doesn't predict agreement with a third. The same models change rank from one package to the next.

Sol marked fewer tests keep than either smaller model: 129 of 191 (67.5%), against DeepSeek's 85.9% and MiMo's 78.2%. That's below my interval (68–87%). Its weak marks fall before the oracle less often than theirs did: 74%, against 83–85%.

## What it found that neither smaller model reported

Claude checked each of these against `src/` and the Go at `origin/main`.

- **Spell alignment uses the sign, not ±350.**
  - **The divergence.** C's `IS_GOOD` and `IS_EVIL` are `>= 350` and `<= -350` (`src/utils.h:454-455`). The spells package's `isGood` and `isEvil` test `> 0` and `< 0` (`pkg/spells/damage_spells.go:441-453`), and they're live for dispel and affect paths.
  - **The rest of Go is right.** Its player and mob predicates use ±350.
  - **The tests pin it.** `TestIsEvil` and `TestIsGood` require -1 and +1 to count.
  - **What the others did.** DeepSeek marked them R. MiMo marked them D as redundant helpers, so it would have deleted the test without noticing it pins a divergence.
- **Hellfire knocks its victims to resting.**
  - **The divergence.** C sets `POS_SITTING` (`src/spells.c:737`), which is 6 (`src/structs.h:215`). Go writes `SetPosition(5) // POS_SITTING` (`pkg/spells/affect_spells.go:3144`), but Go's own `PosSitting` is 6 (`pkg/combat/formulas.go:39`), so 5 is resting. The comment is right and the number is wrong.
  - **No test asserts the position.**
- **Several spells use the cast level where C uses the caster's own level.**
  - **The sites.**
    - lay hands dice, `dice(3, GET_LEVEL(ch))` at `src/magic.c:1823`;
    - meteor swarm damage, `src/spells.c:1094`;
    - the circle of summoning timer, `src/spells.c:1018`.
  - **Why no test catches it.** The two levels are equal when a player casts, so every fixture passes.
  - **When players see it.** They differ when a spell comes from a scroll, wand or staff. That's the same surface the DP-1456 spell matrix just reopened.
- **Reagents are found by name, not vnum.** C matches `GET_OBJ_VNUM(tobj) == item0` (`src/magic.c:548`). Go's `ConsumeSpellReagent` finds the item by name (`pkg/game/spell_reagents.go:6`).
- **Smaller representation differences**, whose visible effect is still open:
  - Metalskin's affect is split into two records where C has one (`src/magic.c:1308-1312`).
  - Greater and lesser perception store `ApplyNone` where C stores `APPLY_SPELL`, with modifier 0 (`src/magic.c:1344`, `:1354`).
  - Identify on a wand or staff prints one space before the spell name where C's overlapping `sprintf` reads as two (`src/spells.c:533-534`). That's an R1a source-intent reading, so a transcript must decide it.

Sol also found both earlier spells defects: mirror image's clone level (DeepSeek's) and the invented `AttackTypes` table (MiMo's). It missed `World.HandleSpellDeath` (DeepSeek's), but found a different dead pipeline, the private `World.doDamage`.

## Tests that cannot fail

Sol's mutations found three tests that stay green when what they name is broken:

- `TestCast_RoutesToCallMagic` passes an invalid spell and checks only for silence, so removing the dispatch keeps it green.
- `TestMagAffectsMessages_RoomAudience` returns on the witness line before it checks the victim.
- `TestSavingThrowBeforeKeyingChecksum` hashes a frozen fixture, so it stays green when every production lookup changes.

## What it means

- **In spells, the frontier pass found the most.** In combat, Sol found one divergence the others missed: the NPC wait state, PF-104, now PF-105. In spells it found five live ones, and it missed only one of the three defects the smaller models had found.
- **A test that pins a divergence looks like a keeper, or like junk, depending on who reads it.** The ±350 tests are the third instance, after `counter_procs` in combat and mirror image in spells. Each was marked keep by at least one auditor, and the C-faithful fix turns it red. A deletion pass that trusts "redundant helper" marks would remove the evidence along with the test.
- **The union still matters.** No single model found every defect in either package.

## Limits

- One frontier model, two packages.
- Sol's subtest rows are collapsed to the weakest mark per function for the agreement figures.
- The divergences are confirmed from source, not yet from transcripts. Their issues and scenarios come next.

## Ledger

Recorded as **PF-108** in `EVIDENCE_LEDGER.tsv`.
