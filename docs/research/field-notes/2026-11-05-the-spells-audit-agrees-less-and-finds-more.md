# The spells audit agrees less and finds more

2026-11-05. Field note by Claude, from reviewing the second package test audit. Two smaller models ran it independently and blind: DeepSeek (through Cline) and MiMo (V2.6 Pro). Both ran the [brief](../test-audit/spells/BRIEF.md) at `c85ab41be`, the same method as the combat pilot. Their ledgers, mutation logs and notes are in [`../test-audit/spells/`](../test-audit/spells/).

## Results

The package has 191 test functions, and both ledgers pass the brief's completeness check. MiMo split one test, `TestMagUnaffectsActivityOutputSites`, into three rows. That test is left out of the comparison below, which covers 190 tests.

| | DeepSeek | MiMo | Claude's guess |
|---|---:|---:|---:|
| R (keep), by row | 164 of 191 (85.9%) | 151 of 193 (78.2%) | 78% |
| F (fix the check) | 6 | 8 | |
| C (merge) | 20 | 17 | |
| D (delete) | 1 | 17 | |
| Mutations run | 6 | 15 | |
| F, C and D marks on tests written before the oracle | 23 of 27 (85%) | 35 of 42 (83%) | at least 85% |
| Agent's own guess at R | 62% | 70% | |

Both agents guessed lower than they found. Spells is mostly C-cited: nearly every behavioral test names a `src/` line and asserts exact player bytes.

## Agreement fell

| | Combat | Spells |
|---|---:|---:|
| Same mark, of the four | 92.3% | 77.9% |
| Cohen's kappa, four marks | 0.70 | **0.30** |
| Cohen's kappa, keep versus not-keep | 0.73 | **0.29** |

Same two models, same method, a different package. Kappa dropped from "substantial" to "fair". Most of the gap is one pattern: MiMo marked D on 16 tests that DeepSeek kept.

- Most are micro-tests of small helpers (`TestIsEvil`, `TestSameRoom`, `TestSendAct`), most from Kimi's PR #133.
- Some are nil-receiver guards that the live game can't reach.
- Each of MiMo's deletes names the behavioral test that still covers the helper.
- This is a disagreement about taste (is a redundant helper test junk?) more than about fact. So the two package numbers measure the agents' thresholds as much as the tests.

## Each agent caught a real defect the other missed

Same as the combat pilot. Claude checked each one against `src/` and the Go call sites.

- **DeepSeek only: mirror image creates its clone at the wrong level.**
  - **The divergence.** C creates the clone at level 1 (`create_mobile(ch, MOB_CLONE, 1, FALSE)`, `src/spells.c:1136`). Go passes the spell level (`pkg/spells/affect_spells.go:3917`).
  - **Why players notice.** C's `create_mobile` sets the clone's level, hitroll, damroll, AC and hit points from that number (`src/new_cmds2.c:588-603`), so a player's clone is far stronger in Go. `castMirrorImage` is the only path.
  - **The test locks it in.** `TestCastMirrorImageOutputAndRoomAudience` asserts the wrong level. Claude changed the Go to the C value, and the test went red.
  - **What MiMo did.** It marked this test R.
  - **The pattern again.** A test pinning a divergence from C, the same shape as `counter_procs` in combat.
- **MiMo only: an invented table with a false "matching C" comment.**
  - **The table.** `spells.AttackTypes` (`pkg/spells/spell_info.go:152-153`) holds 34 attack words. Some, like "smash" and "electrocute", appear in no C source.
  - **The C.** It has one 15-entry table, starting with `{"hit", "hits"}` (`src/fight.c:84-99`). The live game uses the faithful copy, `combat.AttackHitTexts`.
  - **Dead code, kept alive by tests.** The spells table's only reader, `MagAttackModifier` (`pkg/spells/call_magic.go:259`), has no production caller. Three tests pin the invented words: R4 invention, dead.
  - **What DeepSeek did.** It kept all three tests.
- **DeepSeek only: a second death pipeline nothing calls.** `World.HandleSpellDeath` (`pkg/game/death.go:441`) has no production caller. Because it's an exported method, the dead-code ratchet doesn't see it.
- **Both: the reagent bonus.** `checkReagents` returns a level-scaled bonus. C's `has_reagents` returns true or false (`src/magic.c:481-495`). Every live caller only checks "above zero", so players can't see it today.

Claude replicated two of DeepSeek's mutations (`Cast` routing removed stays green; the mirror image fixed to C goes red) and confirmed that the saving-throw golden table hasn't changed since it was added (`git diff 71d74913b c85ab41be` is empty). Both agents found the table an exact copy of `src/magic.c`, and MiMo checked all 2,460 cells by script. Unlike the combat AC table, it was faithful from the start, despite predating the oracle.

## Attribution: same commits, different effort

Both agents found the same introducing commit for all 190 tests, but named the same agent for only 125.

- DeepSeek left 122 tests `unknown`, MiMo 64.
- Claude checked two of MiMo's attributions against GitHub. PR #133's branch is `kimi/python-coverage-2026-07-11`, and PR #1447's is `glm/modernize-saving-throw-keys`. Both are correct, and DeepSeek has both as `unknown`.
- So DeepSeek skipped the brief's PR-lookup step for many commits. Its attribution column is incomplete, not wrong.

## What it means

- **The pre-oracle pattern held in a second package.** 83–85% of weak marks fall on tests written before the first oracle. After it, 4 of DeepSeek's 106 rows and 7 of MiMo's 108 are not keepers.
- **Agreement between two smaller models isn't stable across packages.** One package's kappa doesn't predict the next. A single-model audit would have missed one of the two real defects in each package so far.
- **The union of two audits found more than either.** In both packages, each model caught a real defect the other marked "keep". Two blind passes at smaller-model cost seem worth more than one.

## Limits

- Two packages, two models, one brief family.
- MiMo's extra rows and the excluded test change the counts slightly; the agreement figures use the 190 single-row tests.
- The frontier pass (Sol) hasn't run on spells yet.

## Ledger

Recorded as **PF-107** in `EVIDENCE_LEDGER.tsv`.
