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

## Post-oracle redo, batch 3 (tests 101–150, T439–T488; 2026-07-14..16)

Marks: R 49 / F 0 / C 0 / D 1. Cites grepped this session: mobact draws
src/mobact.c:120,146; steal draws/strings src/act.other.c:221,280,395-405,481;
mud_time_passed src/utils.c with beginning_of_time src/db.c:417,420.

**D: TestRealtimeWorldTickersFreezeWhenDPClockIsSet (J9/J4)** — pkg/game owns
no realtime ticker (world.go:400: PointUpdate is driven only by the server
heartbeat), so the 20ms assertion holds with DP_CLOCK unset too (probed).
The freeze contract is tested for real in pkg/engine
(gameloop_test.go:40,117). Deleted; the engine tests are the remaining proof.

R spot-checks: backstab extra draw red; recall dream byte red; steal extra
draw red (after two wrong-target lessons recorded in-session: the first
anchor hit DoStealth's roll, not DoSteal's).

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

## Post-oracle redo, batch 5 (tests 201–250, T539–T588; 2026-07-17..18)

Marks: R 50 / F 0 / C 0 / D 0. (This branch's ledger splices the unmerged
batch-3/4 rows T439–T538 so the file is complete T001–T588; the PRs are a
train per the follow-up.) Cites grepped this session: drink gates
src/act.item.c:933,939,944; drunk speech speak_drunk src/act.comm.c:757-789;
give-weight IS_CARRYING_W(vict) src/act.item.c:690.

R spot-checks: onget trigger disabled red; leader look moved after greet
red; wand charge not decremented red.

## Post-oracle redo, batch 6 (tests 251–300, T589–T638; 2026-07-18..21)

Marks: R 50 / F 0 / C 0 / D 0. The depth-skill families (behead, bite, carve,
first aid, flesh-alter with the src/new_cmds.c:1843+ weapon-band table,
gen-write) and the spec-proc wave (cityguard, dragon breath, fido, guild
guard incl. the GuildInfo table, janitor, paladin, fighter, guild). Cites
grepped this session: love-bite act trio src/new_cmds.c:1238-1241; flesh
bands src/new_cmds.c:1843; gen-write "That must be a mistake..."
src/act.other.c:1115. Ledger complete T001-T638 on this branch.

R spot-checks: GuildInfo room red; flesh-alter failure wait red; carve
initials byte red (after the recorded wrong-arm lesson).


## Post-oracle redo, batches 8 onward — prediction before reading tests

The remaining inventory is T639–T1991: 1,353 tests, all later than the 2026-07-12 oracle start. Before opening any remaining test body, I predict R 1,240 / F 50 / C 28 / D 35 (1,353 total). There are no pre-oracle tests left to classify; the completed pre-oracle result remains R 335 / F 3 / C 0 / D 0. I expect most weak marks to involve vacuous checks, duplicate contracts, or wrong-path negatives, and few or no J10s because these tests postdate the oracle harness.

## Post-oracle redo, batch 8 (T639–T688; 2026-04-25..11-13)

Marks: **R 46 / F 3 / C 1 / D 0.** Junk: J2 1, J3 1, J4 1, J10 1.

The initial prediction said all remaining tests were introduced after the oracle
start. Git attribution corrected that: **16 of these 50** were introduced before
2026-07-12 (pre-oracle split: R 14 / F 1 / C 1; post-oracle split: R 32 / F 2).
Through T688, cumulative counts are pre-oracle R 349 / F 4 / C 1 / D 0 and
post-oracle R 330 / F 3 / C 0 / D 1. The 1,353-test prediction remains open
until the audit finishes.

**F fixes, each mutation-proven (both halves in `mutations.tsv`):**

1. `TestEnterCircleLookBytesMatchC` (J10): its first line expected CRLF, but C
   sends LFCR at `src/spec_procs.c:1964-1975`. The Go helper now emits C's bytes,
   and the assertion checks the literal LFCR. `enter_circle` has no C room or
   mobile assignment (`src/spec_assign.c:573-635`; R2/R4/R5e), so there is no
   player scenario that can reach the changed line. A targeted census would
   have no scenario capable of failing on this edit; the direct unit test is
   the available proof.
2. `TestEquipmentToInventory` (J2): the old predicate accepted
   `ObjNowhere` after a successful unequip. It now requires the exact
   `LocInventoryPlayer(player.Name)`.
3. `TestExecDNSAddDeletePrefixAndPersistenceState` (J3): despite its name, the
   old body checked only in-memory state. It now reloads a fresh World after
   add and delete and checks the persisted list. Removing both save calls kept
   the old body green and failed the fixed test.

**C merge:** `TestFindExp_GoldenClassModifiers` (J4) repeated the level-20 case
already covered by `TestFindExp_GoldenFormulaLevel13Plus` across all classes and
levels 13–40. Removed the duplicate and redirected the depth proof to the
retained test. Corrected the stale formula citation from approximately 340 to
`src/class.c:1177-1178`; fixed XP values are at `1131-1158` and multipliers at
`1093-1128`.

Other comment/citation cleanup: replaced stale BUG and Carrier-workaround
comments on extraction tests; removed an orphan comment for a nonexistent
object-visibility test; narrowed the steal-test comment to its actual mob-item
transfer assertion; corrected the room-light `obj_to_room` citation.

**Weak-test author attribution (git identity/trailers):** T657 DeepSeek; T663
OpenClaw (multi-model orchestrator); T665 unknown; the merged T686 duplicate
was OpenClaw (multi-model orchestrator). Zach's `zdgreene` identity without a
model trailer remains unknown under PF-152/PF-153. No model was assigned from
the Claude Sonnet trailer on T681 because the commit-model attribution table
does not list that commit.

Coverage: **64.4% before / 64.5% after** `GOEXPERIMENT=nojsonv2 go test -cover
./pkg/game`.

All 11 required repository gates passed on the final batch state: `make fmt`,
Go build/vet/test, pinned golangci-lint, modernize, deadcode, string census,
fidelity-depth, expected-divergences and function-index. The first full test
attempt exposed a spacing error in the newly added DNS persistence expectation;
that expectation was corrected, the focused DNS test passed and the full suite
passed on retry. `make string-census` passed its ratchet while reporting its
generated reports as stale.

R-mutation spot checks were not required because this batch contained F/C
marks. The detached mutation tree was restored byte-for-byte after each of the
three proofs.

## Batch 9 — T689–T738 (special procedures)

This batch audits the 50 remaining spec-procedure tests from the GLM draft. Its IDs follow the merged T639–T688 rows in the current ledger. Results: **43 R, 5 F, 2 C, 0 D**. All 50 introductions are post-oracle (2026-08-28/29). The five F tests were strengthened in this batch; the partial Puff table was folded into its full 0–90 partition, and the game-local atoi test was folded into `pkg/common/catoi_test.go:TestAtoiCMatchesC` with the unique wraparound case retained.

Junk annotations: J1=1 (backstab damage expectation reuses independently tested formula helpers), J3=5 (Puff gates, assassin refusal/hire routing, cleric blindness threshold and weather gate), J4=2 (the two folds), J7=8 (four citizen and four cleric tests use the `citizenNumber`/`clericNumber` seams), and J8=1 (normal-checker target is observed through a test combat engine). Every non-R row traces to git author Zach without a model trailer and is `unknown` under PF-152/PF-153, despite GLM branch names.

All five F rows have detached mutations on exact base `ffc5dbde9`: the original test stayed green, the corrected test failed, and the mutation tree was restored byte-for-byte after every proof. Baseline and final package coverage were **64.5% / 64.5%** (`GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game`); the C folds did not reduce coverage. All 11 required gates passed on the final rebased base `4999da772`; no census was needed because this batch changes tests and research records only.

The clearest weak examples here are the Puff gate test, which could not see an extra RNG draw; the assassin success test, whose aggregate sink allowed a private line to reach the victim and still pass; and the cleric threshold test, whose old blind roll never entered the disputed cast branch. Each green gap is documented in `mutations.tsv`.

## Post-oracle redo, batch 10 (T739–T788; 2026-09-16..11-07)

Marks: **R 48 / F 2 / C 0 / D 0.** Junk: J2 1, J3 1. All 50 tests were introduced after the 2026-07-12 oracle start. This is the next 50 not-yet-ledgered functions in `inventory.txt`, after excluding functions already audited in T001–T738.

**F fixes, both mutation-proven in `mutations.tsv`:**

1. `TestApplySkillDamageEmitsTheHurtLineForEveryAttackType` (T758, J3): despite the name, the old body sent only Headbutt through the shared damage path. It now covers every supported skill attack ID and both generic attack types, checking the C pain line at `src/fight.c:1587-1592`. Skipping the bash default-arm callback kept the old body green and made the bash subtest fail.
2. `TestBanManagerConcurrentReadWrite` (T781, J2): the old test only exercised concurrent calls and asserted no final state. It now checks that the final added substring ban remains visible. Returning from `AddBan` before storing the site kept the old body green and failed the new `IsBanned` postcondition.

**Weak-test attribution:** T758 was authored by DeepSeek. T781 is `unknown` under PF-152/PF-153: its introducing commit has Zach’s git identity and no model trailer. R spot checks were not required because the batch contains F marks.

Coverage was **64.5% before / 64.5% after** (`GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game`) on the refreshed base. The targeted tests passed. All 11 gates passed after rebasing to the current main: `make fmt`, Go build, vet and full test; `golangci-lint run ./...`; modernize, deadcode, string-census, fidelity-depth, expected-divergences and function-index checks. The first full-suite attempt had two `cmd/server` readiness timeouts under package parallelism; `go test ./cmd/server -count=1` passed, and the repeated full suite passed. `make string-census` passed its ratchet while reporting stale generated reports; fidelity-depth and function-index passed their intentional expected-red self-tests.

The 1,353-test forecast made before T639 was R1240/F50/C28/D35. Through T788, T639–T788 actuals are R137/F10/C3/D0, leaving 1,203 tests and forecast remainder R1103/F40/C25/D35. No census was needed because this batch changes tests and research records only.

## Batch 11 — T789–T838 (visibility, actor bytes, clan and combat-body paths)

This batch follows the exact T739–T788 commit from PR #2984 on current
`origin/main`, so the ledger remains contiguous while the earlier append is
reviewed. It audits the next 50 names not already present in the pinned
`inventory.txt`. Results: **46 R, 4 F, 0 C, 0 D**; all 50 introductions are
post-oracle (2026-08-31 through 2026-11-20).

Junk annotations: J2=1 (T822’s mismatched output sink made the negative-only
assertion vacuous), J3=3 (T800’s 30-letter row is beyond C’s defined buffer
contract; T821’s two refusal predicates were both true in the only case;
T824’s old below-cap input was not the claimed equality boundary), J5=1
(T799 directly pins the C plain-char conversion helper), and J9=1 (T804’s
negative-only assertion passed if the observer delivered nothing).

The four F tests were strengthened in this batch:

- T804 (`out_of_band_test.go:112`): the PR #1553 commit,
  `feat(gmcp): Mudlet-grade GMCP message set, fired from the text's own
  moments`, appears to have aimed to suppress the no-repeat acknowledgement
  from the mirror. Its test rejected that actor record but did not require a
  delivered recipient record; bypassing observer delivery left the old test
  green. Git author/trailer attribution: Claude Opus 5.5.
- T821 (`spec_entry_gates_test.go:100`): PR #1986,
  `fix(spec): no_get, elements guardian and cityguard bribe swing hit()`,
  appears to have aimed to pin the refusal swing and draw. The old amount=10,
  roll-zero case made both arms of C’s OR true, so losing either condition
  stayed green. The fixed test isolates each arm and checks the hit draw.
  Git author/trailer attribution: unknown (Zach identity, no model trailer).
- T822 (`clan_is_playing_test.go:10`): PR #2543,
  `fix(clan): gate live-vs-stored clan lookups on CON_PLAYING`, targeted the
  live-versus-stored applicant path. The old sink listened to `Leader` while
  the applicant-list command ran as `Mortal`, and had no positive live
  applicant assertion. The corrected test captures the caller and requires a
  live applicant while excluding linkless state. Git author/trailer
  attribution: unknown (Zach identity, no model trailer).
- T824 (`clan_plan_overflow_test.go:12`): PR #2139,
  `fix: review follow-ups — C's clan plan editor semantics, stream race,
  ceiling refresh`, appears to have been a C-boundary follow-up. The prior
  “boundary” case exercised the first-line branch and did not test equality in
  the later-line comparison at `src/modify.c:146`. The corrected test starts
  with a three-byte buffer, accepts a 1018-byte line where
  `3 + 1018 + 3 == MaxClanPlanBytes`, then skips a 1019-byte line.
  Git author/trailer attribution: unknown (Zach identity, no model trailer).

Mutation correction for T824 (2026-11-21): the earlier `mutations.tsv` row said
that changing the later-line comparison from `>` to `>=` made the fixed test
red. That was not reproducible: the earlier body's equality case hit the
first-line branch, so the old body and the pre-fix T789 body both passed under
the mutant. I cannot explain the earlier recorded red; its output was not
retained. On the rebased branch, the old body passed under the mutant and the
fixed later-line equality test failed (`got 3 buffer bytes, want 1023`). The
unmutated test passes. The accurate rerun is recorded in `mutations.tsv`.

The other three F rows have detached production mutations on base `3229914ce`:
old body green, fixed test red, and restoration checked byte-for-byte. All four
rows are in `mutations.tsv`; no all-R spot checks apply.

By introducing identity: **unknown=41, Claude Opus 5.5=7, DeepSeek=1, Luna=1**.
All 50 are post-oracle. No test was found with expected bytes or numbers that
disagree with defined C behavior. T800’s 30-letter input would overrun C’s
static `holder[30]` when writing its terminator, so the test explicitly treats
that row as Go behavior rather than defined C output. A scan of 2,402 object
and door keyword first runs in `lib/world` found none at 30 letters or longer
(maximum 13).

Coverage was **64.6% before and after** (`GOEXPERIMENT=nojsonv2 go test
-cover ./pkg/game`). The focused four-test run and the full suite passed. All
11 requested gates passed; the first golangci-lint invocation collided with a
shared-host lint run, then the pinned v2.15.0 retry passed with 0 issues.
`make string-census` passed its ratchet while noting generated reports are
stale. `make function-index` completed but regenerated broad pre-existing
line-number changes; those generated report changes were restored, and
`make function-index-check` passed.

Scope note: current `origin/main` adds
`TestClanQuitClanlessKeepsApprovedRefusal`, which is absent from the prompt’s
pinned 1,991-name inventory. It was left out of this batch’s IDs; the audit
remains scoped to the authoritative inventory until Zach refreshes it.

## Batch 12 — T839–T888 (combat-body identity, death paths, and actor specials)

This batch classifies the next 50 entries in the pinned inventory: **42 R, 8 F,
0 C, 0 D**. Junk codes are J2=2 (T847, T850), J3=4 (T861, T862, T874,
T883), J9=1 (T875), and J10=1 (T867). All 50 were introduced after the
2026-07-12 oracle cutoff. Attribution by introducing commit is unknown=48,
DeepSeek=1 (T869), and Claude Opus 5.5=1 (T887).

The eight weak tests and their fixes:

- T847, `TestCombatBodyReferencesConcurrentAndReciprocal` (J2): the introducing
  commit is titled “key combat membership by concrete bodies.” Its concurrent
  loops called the setters/getters/stoppers but never checked a value. The
  fixed body checks each selected body and name plus the final cleared state.
- T850, `TestCombatBodyScriptDispatch` (J2): the callback/message refactor
  intended to retain exact body and actor identities through script dispatch,
  but a recorder receiving zero calls made the old identity loop vacuous. The
  fixed test requires 80 fight and 80 death records.
- T861 and T862 (J3): PR #2003’s combat-round special audit states that C runs
  ordinary hits before `MOB_SPEC`. The old tests could pass without establishing
  an ordinary-hit transcript. Both now compare the special round against an
  otherwise identical seeded round without the special and require the full
  ordinary transcript as an exact prefix.
- T867, `TestConjuredActorDeclinesOnlyForCharmedBodies` (J10): PR #2084
  intentionally ported only the charmed actor decline and called the uncharmed
  leave-the-world branch an explicit gap. C handles that branch, so the actor
  lane now delegates to `specConjured`, and the test asserts both message
  recipients and extraction. The switched NPC scenario was added to depth
  coverage; its targeted census is still pending at this note's first write.
- T874, `TestCorpsesMoneyAndAshDoNotSurviveRestart` (J3): the restart audit
  covered transient world state, but this test's setup created only synthetic
  money and a corpse despite promising ash. It now creates prototype vnum 18
  dust and checks all three objects before and after boot.
- T875, `TestCounterProcsBlessingOnlyReachesPlayingDescriptors` (J9): the
  descriptor-count change targeted global rewards, but the test checked a peer
  recipient without pinning that the actor was excluded. It now requires the
  actor's own milestone line to be its only output.
- T883, `TestCreateMoneyObjectExtraDescs` (J3): PR #2694 matched C's
  randomized `create_money` description and draw, while the test only required
  the stable text prefix. It now resets the stream and checks the exact seeded
  7000-coin description.

All eight F rows have detached mutation proofs in `mutations.tsv`; for each,
the old body stayed green, the fixed body failed, and the mutation was restored.
No R spot-checks were required because this batch contains F marks. After the
conjured change, no test in this batch still pins output or state that disagrees
with C. T843 remains an R with medium confidence: it deliberately tests the
Go movement transaction's admitted in-flight behavior, for which C's
single-threaded path has no concurrent interleaving analogue. The stale #1907
claim was removed from its comment.

By the ledger's `born_before_oracle` field, T639–T888 contains 16 pre-oracle
rows (R14/F1/C1/D0) and 234 post-oracle rows (R211/F21/C2/D0). Cumulatively,
T001–T888 contains 354 pre-oracle rows (R349/F4/C1/D0) and 534 post-oracle
rows (R509/F22/C2/D1). The forecast from before T639 was R1240/F50/C28/D35
for the remaining 1,353 tests. Through T888, T639–T888 actuals are R225/F22/C3/D0,
leaving 1,103 tests and forecast remainder R1015/F28/C25/D35. The remaining
1,103 tests continue in the next batch; no package summary or PF row is due yet.


## Post-oracle redo, batch 13 (T889–T938)

Marks: **R 46 / F 4 / C 0 / D 0.** Junk: J3 4.

The four F tests have detached mutation proofs in `mutations.tsv`; each old
body stayed green, each repaired body failed under the same mutation, and every
mutation was restored. No R spot-checks were required because this batch has F
marks.

**F repairs:**

1. T895, `TestLuaCharTableMobileFields` (J3): the old mobile was at full health,
   so `hp` and `maxhp` were equal and independently sourced expectations could
   not detect a swapped binding. The fixture now sets 37/52 and requires both
   literal values. The test was introduced in PR #1975, whose title describes
   a field-for-field C char-table match; the weakness was the fixture’s lack of
   a distinguishing state.
2. T915, `TestMagicUserSelectsFightingMobVictim` (J3): its previous assertion
   accepted any utterance, so it did not establish that the fighting mobile was
   selected. The repaired seed declines the room-candidate roll and chooses
   shocking grasp; it checks damage to the mobile and unchanged HP for the
   unrelated player. The introducing PR #2003 was a combat-round special audit,
   while this test’s earlier body only witnessed a cast.
3. T916, `TestMailFirstLineOverflowTruncatesAndKeepsWriting` (J3): uniform
   input plus a length-only assertion could not tell whether the editor kept a
   prefix or suffix. Patterned input now pins the exact retained prefix. The
   test arrived with PR #2475’s `string_add` overflow behavior fix.
4. T923, `TestMailReadWriteSharedFilePositioning` (J3): offset zero made an
   ignored file position indistinguishable from a correct seek. It now targets
   the third block and verifies the first block survives; the introducing PR
   #1392 was the mail modernization/consolidation change.

The audited C call paths agreed with the tests; no C disagreement or J10 was
found. T914 is a Go-only concurrent charge-consumption contract (C is
single-threaded), and T921 covers a Go pre-initialization lookup fallback with
no direct C API equivalent; both have medium confidence. T918 is classified as
hand-written because the directory-store guard is Go-specific robustness. T920
is the child-process phase dispatcher required by the surrounding integration
tests.

Authorship for this batch: OpenClaw 7 (T893, T929, T931–T935), Claude Opus 4.8
1 (T891), Luna 2 (T895–T896), unknown 40. Weak-test authorship: Luna 1 and
unknown 3. By introduction date, six rows are pre-oracle (T929, T931–T935):
R6/F0/C0/D0. The remaining 44 are post-oracle: R40/F4/C0/D0.

Cumulative T001–T938: pre-oracle R355/F4/C1/D0 (360 rows); post-oracle
R549/F26/C2/D1 (578 rows). The initial T639–T1991 forecast remains
R1240/F50/C28/D35. Through T938, the audited T639–T938 actual is R271/F26/C3/D0;
1,053 tests remain, with the forecast remainder now R969/F24/C25/D35. No
package summary or PF row is due until T1991.

`GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game` remains at 64.6% statement
coverage, unchanged from batch 12; this batch improves assertion strength
without adding production statements.

Batch 12’s first targeted census was setup-refused because its fixture used
`switch horse` for spawned mob 14410 (Silk), and the attempt had no draw-parity
artifact. Commit `8546e1df4` corrects the warmup to `switch silk`; a proper
four-scenario `--draw-parity` recheck is queued in PR #2990 at
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-21/luna-ta-game-12-conjured-actor-recheck`.
