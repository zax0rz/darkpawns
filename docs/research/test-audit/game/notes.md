# Test audit: pkg/game — notes

Audit run: 2026-11-21, branch `glm/test-audit-game`, base `origin/main`
(706084c68). Method: `docs/research/test-audit/pilot-combat/BRIEF.md` with the
three changes from PROMPT-2026-11-20 (single agent self-checks non-R marks;
fix-as-you-go; output in-repo) and the PROMPT-2026-11-21 additions (BRENDA69
rows attribute to "OpenClaw (multi-model orchestrator)" per PF-152).

Baseline: `go test ./pkg/game/` passes at 706084c68 (3.7s).
Coverage baseline: **64.4%** of statements.

Ground truth: 1,991 test functions in 487 files (the roadmap's "~1,989").
`go test -list` shows 1,991 on Linux. Attribution via per-file
`git log --reverse -p` first-introduction parsing: all 1,991 mapped to an
introducing commit and date.

Era shape: 338 tests born before the first oracle harness (2026-07-12) —
19 from April, 62 from May, 84 from June, 173 from early July. The bulk is
post-oracle: July 371, August 329, September 238, October 646, November 242
(tests attributed to July 12+ count as later; early-July pre-harness tests
count as pre-oracle).

Authors (introducing-commit git author): Zach Greene 1105, zach 403 (the
same human under two identities), BRENDA69 416 (OpenClaw, multi-model
orchestrator — PF-152, no model assigned), DeepSeek/deepseek 41, Luna 15,
remainder single digits.

## Predictions (written before reading any game test)

pkg/command landed at R 161 / F 9 / D 3 (93% keep); pkg/session predicted
94.1% keepers. pkg/game is the oldest and biggest package: it carries the
April 2026 layer — the port's very first weeks, before ANY harness or
coverage discipline — plus the largest BRENDA69 share (416). The April/May
smoke-test layer should be the weakest of any audited package; the huge
October/November layer postdates the depth-test style and should be mostly R.
pkg/game also holds the combat/skill/death formulas, so J10 (a test pinning
Go bytes C disagrees with) is more likely here than anywhere — the pilot and
both Kimi audits each found one.

| Mark | Pre-oracle (338) | Later (1653) | Total (1991) |
|---|---|---|---|
| R keep | 262 | 1540 | 1802 (90.5%) |
| F fix | 48 | 72 | 120 (6.0%) |
| C merge | 12 | 20 | 32 (1.6%) |
| D delete | 16 | 21 | 37 (1.9%) |

Junk codes expected: J2 (cannot-fail assertions) and J3 (name overpromise)
dominating the April–June smoke layer; J8 (mock supplies the behavior)
around the manager/engine seams; a handful of J5 (implementation-shape pins)
on the engine internals; at least one J10 (Go/C divergence pinned as
expected). Coverage: roughly flat overall — F fixes strengthen assertions,
D deletions only where a named test keeps the behavior.

Confidence: low on the exact F/C/D split; the R share and the weak-marks-
concentrate-pre-oracle pattern are the firm guesses.

## Batches

- Batch 1–3+: the 338 pre-oracle tests in date order (~50–70 per batch).
- Then the 1,653 later tests in date order, ~50–100 per batch.


## Batch 1 results (first 50 pre-oracle tests, 2026-04-25..05-13)

Marks: R 48, F 2, C 0, D 0 (96% keepers). Junk: J2 ×2, J3 ×1. The batch is
the port's oldest layer: 25 object_movement tests (Location/inventory
invariants, zach), 5 death tests, 10 combat skill-gate tests, 5 command-exec
callback tests and 5 message-routing tests (BRENDA69 = OpenClaw).

Predictions vs actual: predicted the pre-oracle layer at ~78% R — actual
96%. The April object-movement layer is far stronger than guessed: it is the
AGENTS.md-protected ObjectLocation regression suite with concrete pre/post
assertions and error contracts, not smoke tests. The weak marks:

- `TestCorpseAttackLongDesc` (F, J2/J3): the table names C corpse substrings
  (src/fight.c:290-362) but only asserted `len(got) == 0`. Fixed to assert
  `strings.Contains`; mutation (charred→charrid) reds it.
- `TestLocationFieldSync` (F, J2): the AddItem branch logged instead of
  asserting. Fixed to pin the real invariant (AddItem must not change
  Location; MoveObject owns updates); mutation (addItem sets Location) reds it.

Housekeeping (no mark change): four stale "BUG: This test currently FAILS"
comment blocks in object_movement_test.go describe early-port bugs that were
fixed long ago (the tests pass and pin the fixed behavior) — the comments now
say the bugs were fixed; `TestDoBackstab_NotThief` had a copy-pasted duplicate
assertion (removed). `TestSaveLoadLocationRoundTrip` exists only as a comment
block documenting a design gap (no function; not in the inventory).

## Batch 2 results (pre-oracle tests 51–100, 2026-05-13..06-18)

Marks: R 49, F 1, C 0, D 0 (98% keepers). Junk: J3 ×1. (Ledger on branch
`glm/test-audit-game-2`; this shared file carries the running summary.)

- `TestWeatherChange_AdjustsPressureAndSky` (F, J3): the name promises
  pressure/sky behavior but the 20-iteration loop asserted nothing and the
  tail only checked two accessors. Fixed to pin C's formula deterministically:
  seed 7 draws dice(1,4)=4, dice(2,6)=5, dice(2,6)=5, summer diff +2, so
  change = 8 and pressure 1013 → 1021, plus the [960,1040] clamp band.
  Mutation (summer diff +2 → −2) reds it.

The May–June layer is uniformly strong: movement gates, C-cited starting-kit
pins (obj_to_char order, container children), the canSee fidelity matrix
(with a C-cited note that AFF_HIDE is NOT part of CAN_SEE), save round-trips,
weather/wraps, ban substring semantics, and the item donation/wear/transfer
family with exact C refusal strings. No J10 found yet — the class both prior
audits each hit remains open.

## Batch 3 results (pre-oracle tests 101–150, 2026-06-18..06-28)

Marks: R 50 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-3`.)

The June–early-July layer is the DP-era: every kuji-kiri/berserk test carries
exact C refusal strings, gate order and draw-site pins (including the
C dangling-else improve_skill bug, documented in-test); the regen/find_exp
goldens transcribe src/limits.c and src/class.c tables verbatim; the spec
procs and item NODROP tests assert C messages and the bit-7-vs-bit-0
regression guard. No fixes needed; no mutations run (nothing to prove).
Prediction check: the later half of the pre-oracle set is even better than
guessed — the April-only layer carried all of the batch-1 weakness.

## Batch 4 results (pre-oracle tests 151–200, 2026-06-28..07-04)

Marks: R 50 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-4`.)

The June 28–July 4 tier is uniformly DP-era quality: circle/spike/stake
gates with scripted rollers, the room-flags decimal-parse regression guard
(bit 15 vs bit 8), real-world high-flag room checks against lib/world,
affect-update duration semantics (DP-669, C magic.c:441-450), the safe spec
caller's panic recovery, and C-structs.h bit-order pins. Four batches in:
197 R / 3 F / 0 C / 0 D — every weak mark was in April.

## Batch 5 results (pre-oracle tests 201–250, 2026-07-04..07-05)

Marks: R 50 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-5`.)

The July 4–5 tier keeps the DP-era standard: bash/headbutt/kick/rescue/trip/
dragon-kick gate families with scripted rollers and C line citations
(act.offensive.c:488-489 unconditional wait, new_cmds.c:443-445 recoil,
DP-936 flat damage), the attribute golden tables transcribed verbatim from
src/constants.c:1124-1183, and the nil-ch spec guards. Running total:
247 R / 3 F / 0 C / 0 D.

## Batch 6 results (pre-oracle tests 251–300, 2026-07-05..07-11)

Marks: R 50 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-6`.)

The July 5–11 tier: spec-proc goldens (cityguard/cleric/dragon-breath/dump/
fighter/guild/janitor/magic-user/snake/thief), the smoke-all panic contract
over every registered spec, practice-parameter and skill-formula goldens
(bash success rate pinned to C's 60/101 within ±3%), race-hate aggression
gates (the mobact.c family the L1 work knows well), player-store versioning
warnings, and the animate-dead charm/cap gates. Running total: 297 R / 3 F /
0 C / 0 D.

## Batch 7 results (pre-oracle tests 301–338, 2026-07-11) — pre-oracle COMPLETE

Marks: R 38 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-7`.)

The last pre-oracle tier (July 11): mob AI aggression gates (aggr24/aggressive/
memory with sight, NOHASSLE and PEACEFUL skips), movement-cost tables,
alignment shifts, HP band thresholds, sanctuary halving, the inventory
CAN_GET_OBJ order, and str_app-derived capacity.

## Pre-oracle summary (338 tests, complete)

**R 335 / F 3 / C 0 / D 0** — 99.1% keepers. All three F marks were in the
April object-movement/early-May layer (batch 1); every F was fixed in its
batch PR with a mutation proof. Predictions (262/48/12/16) overshot the
weakness badly: the actual pre-oracle layer is nearly all keepers, because
the April suite is the deliberate ObjectLocation regression set and May–July
tests were already DP-cited. The audit's "weak tests are pre-oracle" pattern
from pkg/combat and pkg/spells does NOT reproduce here — pkg/game's pre-oracle
layer was written with contracts from the start.

Authors: zach 139, BRENDA69/OpenClaw 195, glm 4. J10 count: 0 (no test pins
Go bytes C disagrees with in the pre-oracle set; the known design gap —
objsave load never sets ObjectLocation — is documented, not pinned).

## Batch 8 results (post-oracle tests 1–100, 2026-07-12..07-18)

Marks: R 100 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-8`;
post-oracle ledger ids continue from T339.)

The July 12–18 era is oracle-era quality as expected: recall fidelity with
C's exact dream bytes, zone-reset seam tests with ordered call hooks
(floating-O, percent-load, dependent commands), stealth draw-order pins,
the shipped-MOTD fixture compare against the C file, and the race-hate /
scavenge / sound-draw order families. Heuristic scan found zero
assertion-less bodies; deep reads agree.

## Batch 9 results (post-oracle tests 101–200, 2026-07-16..07-27)

Marks: R 100 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-9`.)

July 16–27 era: movement command gates, paladin spec (charge arithmetic,
disarm audience/state), get/carve depth tests with act.item.c byte citations,
guild-guard/dragon-breath/bite/bearhug/ambush depth families. The heuristic
scan flagged four mobact draw-order tests; their assertMobactDraws helper
ends in t.Fatalf, so they are real draw-sequence pins.

## Batch 10 results (post-oracle tests 201–300, 2026-07-18..08-01)

Marks: R 100 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-10`.)

The late-July/early-August era is the spec-proc golden wave: tattooist,
butler (GenDoor byte pins, four-item cap), puff, mindflayer, identifier,
clerk, cleric, citizen, and the castle-guard directions — every file with
exact output-byte pins. Heuristic scan clean.

## Batch 11 results (post-oracle tests 301–400, 2026-08-01..08-14)

Marks: R 100 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-11`.)

The August era opens with the spec wave continuing (roach, teleport-victim,
tattooist 2–4 with exact list bytes, prostitute, quan-lo, teleporter,
elements) plus the disembowel depth family. Heuristic scan clean; assertion
density high throughout.

## Batch 12 results (post-oracle tests 401–500, 2026-08-14..09-03)

Marks: R 100 / F 0 / C 0 / D 0. (Ledger on branch `glm/test-audit-game-12`.)

The August–September era is the skill-depth wave: sharpen, spike, shadow,
serpent, jin (with C's joined lockout affect record), neckbreak, mount,
slug, sleeper, and the werewolf spec — each with exact C strings, seeded
draws and affect-record shape pins.
