# Spells test audit — mimo notes

Read-only audit of the 191 `pkg/spells` test functions at `c85ab41be`, judged
against production code and `src/` (C oracle). Ledger: `ledger.tsv` (193 rows;
one test split into three subtest rows). Mutations: `mutations.tsv`.

## Pre-run guesses vs actual

Guesses written before reading any test (top of this file, kept above):

| | guessed | actual |
|---|---|---|
| R share | ~70% (~134 fn) | **78%** (151/193 rows) |
| F+C+D share | ~30% | **22%** (42 rows: F 8, C 17, D 17) |
| F/C/D on pre-oracle tests | ≥80% | **83%** (35/42 rows) |

Claude's recorded guess (78% R; F+C+D 22%; ≥85% pre-oracle) landed almost
exactly on the R/F+C+D split; my 70/30 guess was too pessimistic on R, and my
80% pre-oracle share was slightly low (actual 83%). The pre-oracle
concentration is real but not total: 7 F/C/D rows are post-oracle (keyed-table
refactor artifacts from glm PR #1447, and kimi's helper micro-tests landed two
days before the harness).

## Baseline

`go test ./pkg/spells/...` at c85ab41be: **ok, 0.013s** — all green, 2026-11-06.

## Counts

By mark (193 ledger rows = 191 functions + 2 extra subtest rows for
`TestMagUnaffectsActivityOutputSites`, which needed three marks):

- **R 151 · F 8 · C 17 · D 17**
- R rows born pre/post oracle: 50 / 101. F+C+D pre/post: 35 / 7.

By junk code (a row can carry several): J4 28, J7 18, J9 12, J5 8, J10 6,
J3 5, J8 3, J2 2. J1 and J6 never apply: no golden is Go-generated (all traced
to `src/`) and no test greps source text.

By author agent (rows, introducing commit): unknown 64, codex 64, glm 28,
claude 17, kimi 15, sol 3, deepseek 2. "unknown" is mostly `8c574d2b5`
(2026-05-04 "test: complete testing push", git author `zach`, 29 functions —
pre-oracle bulk push) plus BRENDA69-authored commits (BRENDA is not in the
brief's branch-prefix table) and seven git-author-Zach-Greene commits in the
codex dp1371 wave.

F+C+D rows by agent: unknown 23, kimi 9, glm 6, claude 3, codex 1.
The pre-oracle bulk push owns most junk; kimi's "cover spell helper functions"
(PR #133) owns the helper-micro D cluster; glm's keyed-table refactor (PR
#1447) owns the fixture-trio C/D rows.

By expected source: c-cited 115, hand-written 59, none 18, oracle 1.

## Three clearest junk examples

1. **`TestCheckSavingThrow_DoesNotPanic` — saving_throws_test.go:127 (J2).**
   Assertion is `if result != true && result != false` — a tautology on a bool
   return that cannot fail. Only an implicit panic fails the test. Why it
   exists: padding coverage, from the pre-oracle bulk push `8c574d2b5`
   ("test: complete testing push — parser fixes, combat formulas, spells").
   Mutation (CheckSavingThrow returns constant false) stays green.

2. **`TestCast_RoutesToCallMagic` — spell_helpers_test.go:356 (J3;J9).**
   Name promises it proves `Cast` routes to `CallMagic`, but the input is
   spell `-9999`, for which CallMagic is a no-op, and the assertion is only
   "no panic, no messages" — green even with the dispatch call deleted.
   Why it exists: a wrapper-sanity test written to cover a one-line wrapper
   while "proving" the routing guess; kimi `53b53c286` (PR #133, "test: cover
   spell helper functions and MagCreations (DP-662)"). Mutation (Cast returns
   false without calling CallMagic) stays green.

3. **`TestAttackTypes_IndexZeroIsEmpty` (+ `_TableSize`, `_EntriesHaveValues`)
   — spell_info_test.go:189-209 (J10).** These police a 34-entry
   `spells.AttackTypes` table whose production comment claims it matches "C
   attack_hit_types from custom spell parser"; the C has exactly one attack
   table, `attack_hit_text[]` (src/fight.c:84-99, 15 entries, `{"hit","hits"}`
   at index 0), and the test demands an *empty* index 0. Words like
   "smash"/"electrocute"/"trample" appear in no C source. The table is dead:
   `MagAttackModifier` has no production callers; the live copy is
   `combat.AttackHitTexts` (pkg/combat/fight_core.go:31), which does match the
   C. Why it exists: invented data with a fabricated provenance comment,
   written to match the Go file rather than the C — also from `8c574d2b5`.
   Mutation (live `AttackHitTexts` word drift) leaves all three green.

Runner-up (provenance slop rather than vacuous proof):
`TestMagSummons_AnimateDead_GiddyBlocked` / `_FollowerCapBlocked` assert the
refusal text "C's exact refusal" — but the expected text is the mock's own
`canRaiseMsg` field echoed back, so the content is not C-verified at this
layer (glm `72745e263`). Marked R (blocking state assertions are real) with
J8;J10.

## The saving-throw golden table

`saving_throws_golden_test.go` was added whole in `71d74913b` (2026-06-23,
"tier-2: add golden tests for dex_app, saving throws, XP curve, regen",
git author BRENDA69, PR #36 branch `tier2/golden-tests-combat-tables`) and has
**never been edited since** (no later correction exists in its history).

Where the values came from: the file claims "transcribed VERBATIM from the
original C source (src/magic.c)". Independently verified today: all **2,460
cells match `src/magic.c:83-233` exactly**, including the Cleric PARA
level-2 = 59 quirk (src/magic.c:103) that the file's comment calls out as a
likely original typo. The same 2,460 values are also byte-identical to the
production `savingThrowTable`, which was ported earlier in `8ec63dd6f`
(2026-04-27, "fix(C-09+H-16): port real saving_throw tables (d100)"). So the
bytes are consistent both with a true C transcription and with a copy of the
pre-existing Go table; the quirk note (which a pure Go-copy would have no
reason to write) favors the claimed transcription, but no oracle transcript
exists (predates the harness by 19 days). Bottom line for the ledger:
expected values trace to the C either way — the golden is R.

## Expected values that disagree with the C

No live game behavior disagrees. Two dead/misleading cases, both flagged J10:

1. `TestAttackTypes_IndexZeroIsEmpty` expects empty slot 0 where the C attack
   table has `{"hit","hits"}` (src/fight.c:86); siblings pin a 34-word table
   that exists nowhere in the C. **Not a game bug** — the table is a dead twin
   (no production callers); the live `combat.AttackHitTexts` matches
   src/fight.c:84-99 entry-for-entry.
2. `TestCheckReagents_BonusScalesWithLevel` pins `bonus = level/2` under a
   production comment "matching C behavior / original C bonus is approximately
   level/2" — the C `has_reagents` (src/magic.c:481-491) returns **bool**, and
   its uses are `?level:0` (mag_damage) or `0/1` arithmetic (mag_affects).
   **Not a live game bug** — the only magnitude consumers neutralize it
   (damage path bool-izes; metalskin passes `"flat:1"` to force 1; waterwalk
   discards it behind the `4+reag?20:0` quirk). A future caller trusting
   level/2 as a magnitude would diverge.

Quirks checked and confirmed C-faithful (they look wrong but are law):
`\n\r` terminators (sobriety src/spells.c:697, locate-object, lycanthropy),
the two distinct dream lines (`falling..` word-of-recall src/spells.c:165 vs
`falling...` summon src/spells.c:351), "in not an Outlaw" (charm),
"a ethereal light" (alter-objs), "Your feel your awareness improve",
blindness-with-reagent duration 3/magnitude -5 (C uses reag=1).

## Duplicate production implementations

- **`spells.AttackTypes` + `MagAttackModifier` (pkg/spells/spell_info.go:153,
  call_magic.go:259) vs `combat.AttackHitTexts` (pkg/combat/fight_core.go:31,
  used at fight_core.go:805 and session/wiz_stats.go:179).** The live game
  calls the `combat` copy (15 entries, C-faithful); `spells.AttackTypes` is a
  dead 34-entry twin with invented words and a false "matching C" comment.
  Its three tests (D, J10) pin the dead copy only.
- **`checkReagents` level/2 bonus (affect_spells.go:1638) vs C
  `has_reagents` bool** — see above; same function family, divergent contract
  neutralized at every live call site.
- Test-side triple copy of the saving-throw numbers: production
  `savingThrowTable` + `savingThrowBeforeKeying` fixture + `savingThrowGolden`
  fixture. C/D marks collapse it to one owner (`TestSavingThrow_GoldenAgainstCSource`).
- The C dream-line pair is **not** a duplicate: two different C lines for two
  different arrival paths, and both Go call sites are correct.

## Judgment calls (for review)

- Convention used: **C** = duplicate of a named owner's contract at the same
  layer (J4); **D** = no player-facing contract at all (fixture-integrity,
  dead code, unreachable branches, private plumbing); **R** may still carry a
  junk code when the keep-rule (R1/R3) overrides the pattern (e.g. J7 seams on
  draw-parity tests, J4 notes on CallMagic dispatch rows whose bytes are also
  pinned at the golden).
- Private-predicate micro-tests (`TestIsNPC/IsEvil/IsGood/GetHP/SameRoom/
  SendAct`) are D/J5: real assertions, but they pin 4-10 line type-dispatch
  helpers whose breakage fails boundary tests. `TestSameRoomWithObj` kept R
  because it is the only coverage of the object-target say path.
- `TestMagSummons_AnimateDead_GiddyBlocked`/`_FollowerCapBlocked` were first
  marked F (fixture-echoed message), then flipped to R+J8;J10: the assertion
  does catch delivery regressions; only the content provenance is bogus, and
  content lives in pkg/game.
- Attribution of `c3b8f0c59` (git author "Cline", PR #1775
  `fix/level-scale-class`): recorded **unknown** per the brief's channels
  (no `cline/` branch prefix, no trailer), with the author name in evidence —
  though the name maps to deepseek under the prefix table.
- Seven 2026-10-16 commits by "Zach Greene" inside the codex dp1371 wave:
  recorded **unknown** (no PR on the commit, no agent branch prefix on the
  first-parent merge for most; one merged via `origin/codex/dp1371-spell-matrix`).

## Open questions

- The dispel-evil/good self-damage arm (`dam = getHP(victim) - 10` when the
  caster is evil/good) has no boundary test in any package I could see;
  `TestGetHP` is its only, trivial, pin (D).
- `TestMobFlagBitsPinCPositions` pins only MOB_NOCHARM=13; its comment also
  promises MOB_NOSLEEP=15 and MOB_NOBLIND=17, and MOB_NOSUMMON's value is
  pinned nowhere (the summon tests only exercise the read path).
- Go clamps `GetSavingThrow` level lookups to 0..40; the C would index
  `saving_throws[][][level]` out of bounds for level > 40 (UB). Worth an
  oracle check of what the reference binary does at level 41+.
- `magAffectsSaveRoll`, `magSummonSaveRoll`, `magSummonOutlawResistanceRoll`,
  `magSummonBackfireRoll`, `magCharmSaveRoll`, `magDivineIntSaveRoll`,
  `animateDeadPfailRoll`, and `combat.WithRoller` are production hooks whose
  only callers outside their definitions are tests (J7, 18 rows). Worth a
  later seam-removal pass; deterministic seeding (`dprng.ResetStream`) is the
  alternative several tests already use.
- The statistical `CheckSavingThrow` band tests (F) could be fixed by seeding
  `dprng.ResetStream` and asserting exact draw counts, as
  `TestSavingThrowNPCUsesWarriorTable` already demonstrates.

## Method notes

- Independent measurement: did not read `docs/research/test-audit/pilot-combat/`,
  `/home/zach/dp-research/test-audit/`, or any combat field note before
  finishing. Read nothing under `/home/zach/dp-research/test-audit-spells/`
  except `mimo/`.
- Mutation runs (15 rows in `mutations.tsv`): all 8 F marks stay green under a
  one-line production mutation; the fixture-pair D claims stay green while
  `TestSavingThrow_GoldenAgainstCSource` (the named remaining proof) goes red;
  the AttackTypes trio stays green under live-table drift. Two mutation
  attempts broke the build and were redone compile-safe (discarded rows were
  removed from the TSV).
