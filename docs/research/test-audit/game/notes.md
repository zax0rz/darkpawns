# Test audit: pkg/game — pre-oracle set (consolidated)

Consolidates batches 1–7 (PRs #2936–#2943) into one cumulative ledger:
T001–T338, one header, contiguous IDs, one inventory, combined mutations and
notes, carrying the batch-1/2 code fixes. Method:
`docs/research/test-audit/pilot-combat/BRIEF.md` with the three
PROMPT-2026-11-20 changes (single agent; fix-as-you-go; in-repo output).

Baseline: `go test ./pkg/game/` green at base main. Coverage 64.4% before
and after (assertion strengthening only; no production change).

## Predictions (written before reading any test — from the batch-1 branch)

R 1,802 / F 120 / C 32 / D 37 overall, weak marks concentrated pre-oracle.
Actual pre-oracle outcome: **R 335 / F 3 / C 0 / D 0** — the weakness guess
overshot by an order of magnitude (see Finding below).

## Marks (338 pre-oracle tests, 2026-04-25..07-11)

| Mark | Count |
|---|---:|
| R keep | 335 |
| F fix | 3 (all fixed here, mutation-proven) |
| C merge | 0 |
| D delete | 0 |

The three F fixes, each with BOTH mutation halves recorded (old body green
under the mutation, fixed test red — mutations.tsv):

1. `TestCorpseAttackLongDesc` (J2/J3): the table named C corpse substrings
   (src/fight.c:290-362) but asserted only `len(got) == 0`. Now asserts
   `strings.Contains` per substring.
2. `TestLocationFieldSync` (J2): the AddItem branch called `t.Log` in an
   `if` — a branch that cannot fail. Now pins the real invariant (AddItem
   must not write Location; MoveObject owns updates).
3. `TestWeatherChange_AdjustsPressureAndSky` (J3): looped 20× with no value
   assertions and self-comparing accessors. Now pins C's weather_change
   arithmetic deterministically (seed 7: dice(1,4)=4, dice(2,6)=5,
   dice(2,6)=5, summer diff +2 → change 8, pressure 1013→1021; clamp band
   [960,1040]).

Housekeeping carried from batch 1: four stale "BUG: This test currently
FAILS" comment blocks corrected (the bugs were fixed long ago; the tests pin
the fixed behavior); `TestDoBackstab_NotThief`'s copy-pasted duplicate
assertion removed; `TestSaveLoadLocationRoundTrust`… noted as comment-only.

## Attribution (git only, per PF-152/PF-153)

Full-identity check of all 60 introducing commits against
`git log --format='%an <%ae>'`:

- `BRENDA69 <brenda@labz0rz.com>` → **OpenClaw (multi-model orchestrator)**
  (PF-152) — 269 tests.
- `zach <zach@karl-havoc.labz0rz.com>` → **OpenClaw (multi-model
  orchestrator)** (PF-153: karl-havoc is an OpenClaw commit identity,
  glm-5-turbo/deepseek-chat mix) — 25 tests. The earlier batches wrongly
  said "repository owner"; corrected here.
- `zach <zdgreene@gmail.com>` → **OpenClaw (multi-model orchestrator)** —
  44 tests. The author name is Zach's, but every one of these commits carries
  a `Co-Authored-By: BRENDA69` trailer, and 40 also credit Claude Sonnet 5.
  Zach doesn't write code by hand, so his git identity alone never means a
  human author; check the trailers (Claude review, 2026-11-21).
- The eight glm_batch_a tests are BRENDA69-authored (3f315db63); the file
  name names the repair batch, not the author.

Total: OpenClaw 338 (40 with a Claude Sonnet 5 trailer).

## Finding

**pkg/game's pre-oracle layer has no weak-test stratum.** The April
object_movement suite is the deliberate ObjectLocation regression set
(AGENTS.md-protected) with concrete pre/post assertions; May onward carries
DP cites, byte pins and draw-order parity. The "weak tests are pre-oracle"
pattern from pkg/combat and pkg/spells does not reproduce. J10 count: 0.

The known-but-unpinned design gap (objsave load never sets ObjectLocation)
is documented in object_movement_test.go's comment block; not a J10.
