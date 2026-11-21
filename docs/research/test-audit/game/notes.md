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


## Post-oracle redo, batch 1 (tests 1–50, T339–T388; 2026-07-12..13)

The redo per the follow-up method: every test read in full with the
production code, C cites grepped this session (door precondition strings at
src/act.movement.c:545,627,630,633; say's INT-0 refusal at src/act.comm.c:647;
str_app 18/50 carry 280 at src/constants.c:1023-1049; group-share math
src/fight.c:661-685 with the 693 float-truncation pin; solo XP line
src/fight.c:1646; CAN_GET_OBJ at src/utils.h:544-549; social hide/
min_victim_position at src/act.social.c:53-54).

Marks: **R 49 / F 1 / C 0 / D 0.**

**F: TestConsumables_InstanceIsolation (J9/J3)** — all three halves were
vacuous: the default conditions (full=36, thirst=36) gated the drink shut
before any write, the eat arm only removed the item, and the pour resolved
one object for both shared keywords. No SetValue ever ran, so the isolation
assertions held with nothing happening. Fixed to drive real mutations
(taste decrement; conditions cleared before the sip; a distinct-keyword
pour target), honoring inventory's prepend resolution order. Proof: the
SetValue→prototype leak keeps the old body green and reds the fixed test.

R-mutation spot-checks (per the pace rule, 3 seeded-random picks):
TestPerformActSubstitutesEveryCode ($N renders the actor — red),
TestCanTakeObjUsesTakeBit (bit test → exact match — red; the first attempt,
skipping the check, stayed green because the test pins the positive arm —
recorded as a lesson), TestConsumables_InstanceIsolation (the F proof).


## Post-oracle redo, batch 2 (tests 51–100, T389–T438; 2026-07-13..14)

Marks: R 50 / F 0 / C 0 / D 0. Cites grepped this session: tell gates
src/act.comm.c:908-961, "You can't hold that." src/act.item.c:1705, movement
bytes src/act.movement.c:121,238,909, consider ladder
src/act.informative.c:2386-2388, how_good " (not learned)"
src/spec_procs.c:113.

R spot-checks (3 seeded picks): TestNewbieHometownRoom (Kir-Oshi room
swapped — red), TestDoTellEligibilityAndDelivery (not-found byte — red,
after recording that the NoPersonHere constant is not this path's source),
TestTimeWeatherSnapshotTracksCanonicalClock (snapshot loses the weather
half — red).

## Post-oracle redo, batch 4 (tests 151–200, T489–T538; 2026-07-16)

Marks: R 50 / F 0 / C 0 / D 0. The batch is three tight families plus
singles: the skillmsg reroute family (bash/headbutt/kick/trip ×5 — one
shared contract per shape, all citing src/fight.c:1023-1035 skill_message,
verified this session), the first-player-god bootstrap (citing
src/db.c:3016-3024, verified), the help loader/search family, yank's gates
(src/act.other.c:1620+ verified: no-arg, NOPERSON, self "That's wierd.",
non-follower "That probably wouldn't be appreciated.", already-up mounted/
unmounted, sleeping), plus give's NOPERSON and the gen-tog clan gate.

R spot-checks: yank pronoun byte red; help first-match backtrack reversed
red; quoted-keyword parse disabled red.
