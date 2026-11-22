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

## Post-oracle redo, batch 14 (T939–T988)

Marks: **R 47 / F 0 / C 2 / D 1.** Junk: J3 2, J4 3.

T964 (`TestMobCommandMovement_NoDeadlock`, J3/J4) directly called
`performMoveMob`; it never dispatched an NPC command, and the successful
transfer was already exercised with room output assertions by
`TestNPCPerformMoveLeaveAndArrivalBytes`. I removed the duplicate and added a
comment to the surviving transfer test documenting that it also exercises the
synchronous world write-lock path. T970 (`TestMobFlagComparisonCaseInsensitive`,
J3/J4) only compared hardcoded strings with `strings.EqualFold`; I removed it,
with `TestHasMobFlagCaseInsensitive` retaining the production-helper proof.
T982 (`TestMobHitGain_EvenLevelUnchanged`, J4) repeated the exact level-10
value asserted by T973, `TestMobGainSharedTail`, so I removed it.

No F or regression-claiming D rows were found, so this batch needed no mutation
proofs. No production behavior changed and no oracle census was required. I
read the tests and their relevant production/C paths; the C citations in this
batch were checked against `src/`. Corrections include T942's `mag_points`
source (`src/magic.c:1769-1770`), the ROOM_REGENROOM and MOB_STAY_ZONE
bit declarations (`src/structs.h:74,253`), the mayor's gate cycle
(`src/spec_procs.c:905-908`), and the NPC hit-gain branch (`src/limits.c:133-137`).
No C disagreement or J10 was found.

Weak-test authorship: T964 unknown, T970 OpenClaw, T982 OpenClaw. All other
rows are R. By introduction date, T939–T988 contains 11 pre-oracle rows
(R9/C1/D1) and 39 post-oracle rows (R38/C1). The pre-oracle rows are T939–T941,
T970, T972, T975, T977, and T982–T985.

Cumulatively, T001–T988 contains 371 pre-oracle rows (R364/F4/C2/D1) and 617
post-oracle rows (R587/F26/C3/D1). For the audited T639–T988 segment the totals
are R318/F26/C5/D1. Against the initial T639–T1991 forecast (R1240/F50/C28/D35),
the 1,003 remaining tests have a forecast remainder of R922/F24/C23/D34.
No package summary or PF row is due until T1991.

`GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game` reported 64.6% both before
and after this batch.

## Batch 15 preregistered estimate (before reading tests T989–T1038)

Based on the cumulative forecast remainder after T988, I expect this 50-test
slice to contain R46/F1/C1/D2. I expect roughly two pre-oracle tests and 48
post-oracle tests, with any weak cases concentrated in the older tests.

## Post-oracle redo, batch 15 (T989–T1038)

Marks: **R 47 / F 1 / C 2 / D 0.** Junk: J3 1, J4 2. All 50 tests were
introduced after the oracle was available; the preregistered date split of two
pre-oracle and 48 post-oracle tests was wrong.

T1022, `TestMobileEquipmentReaderArmor` (J3), had a vacuous loop: if a zone
reset failed to spawn the additional mobile, the E command logged an error and
the loop never asserted armor. In a detached mutation worktree, I changed the
max-in-world=2 spawn check in `pkg/game/spawner.go` to reject that spawn. The
old body stayed green; the repaired body failed because it requires exactly
one additional equipped mobile. The mutation was restored byte for byte. Its
old-body/fixed-body/restoration evidence is in `mutations.tsv`.

T995, `TestMobileActivityDivineAngelsStayOutOfCombat` (J4), repeated T989's
alignment-only no-attack contract. I folded both evil- and good-alignment
mobile cases into T989's full-pulse test and removed the duplicate. T1038,
`TestMobInstance_HitModifiers` (J4), duplicated T1027's positive/negative
`HitModifiers` contract; its unarmed assertion now lives beside the real
zone-loaded blessed-weapon assertion in T1027, and the duplicate was removed.

I corrected the berserk affect citation in T1036's test comment to
`src/new_cmds.c:2171-2184` and removed an unused placeholder import/reference.
The cited C call paths and the other C references in this batch were checked
against `src/`. No production behavior changed, so no oracle census was needed.
Weak-test authorship is unknown for all three weak rows (T995, T1022, T1038).

The audited T639–T1038 segment now totals R365/F27/C7/D1 across 400 rows.
The cumulative ledger through T1038 has 371 pre-oracle rows
(R364/F4/C2/D1) and 667 post-oracle rows (R634/F27/C5/D1). Against the initial
T639–T1991 estimate (R1240/F50/C28/D35), 953 tests remain with a forecast
remainder of R875/F23/C21/D34. No package summary or PF row is due until
T1991. `GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game` reports 64.6%,
unchanged from batch 14.

## Batch 16 preregistered estimate (before reading tests T1039–T1088)

Based on the cumulative forecast remainder after T1038, I expect this 50-test
slice to contain R46/F1/C1/D2. I expect all 50 to be post-oracle, continuing
the introduction-date pattern from batch 15.


## Batch 16 audit (T1039–T1088; introductions 2026-04-25–2026-11-18)

The preregistered estimate was R46/F1/C1/D2, with all 50 expected after the
oracle date. Actual: **R44/F5/C1/D0**. Introduction dates split the batch into
20 pre-oracle tests (R16/F3/C1/D0) and 30 later tests (R28/F2/C0/D0). The
forecast missed by four F marks and two D marks; its date split missed 20
pre-oracle rows.

Five F checks were repaired and mutation-proven. T1047 and T1048 had called
`emitMobSkillSurvival` directly, so they stayed green if the real mob-skill
damage seam stopped dispatching survival behavior. Both now enter through
`mobSkillDamage` and distinguish the player wimp-level gate from the
wounded-band message. T1074 checked the container-to-room location tag but not
room membership; T1075 checked invalid-destination errors but not source
preservation; T1077 checked the equipped location tag but not the equipment
slot. The mutations and old-green/fixed-red results are in `mutations.tsv`.

T1052, `TestMobStayZonePreventsWander`, only repeated the case-insensitive
flag helper check in T781; it never attempted a wander. I deleted it. T781
keeps the flag lookup check and T1908 reaches the actual STAY_ZONE wander gate.
T1079 remains R: transactional ObjectLocation rollback is a real Go state
invariant even though C has no matching centralized rollback API.

The object-movement fixtures now use World movement/equipment methods rather
than direct location/slice writes. I expanded the mob position loops to all
values 0..14. I corrected C references for prototype copying in `mob.go` and
`mob_position_test.go`, and corrected the red-gate selection citation in
`moon_gate_mount_test.go` to `src/gate.c:273-276` (mount transfer is
`:281-292`). No runtime behavior changed; no oracle census is needed.

Weak-test attribution in this batch: DeepSeek wrote T1047 and T1048; OpenClaw
wrote the duplicate T1052 and the three object-movement checks T1074, T1075,
and T1077.

The T639–T1088 segment totals R409/F32/C8/D1. Cumulative ledger totals are
**R1042/F36/C8/D2**. By the row-level introduction-date field, T001–T1088 now
contains 390 pre-oracle tests (R379/F7/C3/D1) and 698 later tests
(R663/F29/C5/D1). Recounting the ledger exposed a one-R pre/post split error
in the preceding batch-15 prose: through T1038 the rows are pre R363/F4/C2/D1
and post R635/F27/C5/D1; no earlier ledger rows were changed. Against the
initial T639–T1991 forecast (R1240/F50/C28/D35), the 903 remaining tests have
forecast remainder R831/F18/C20/D34. No package summary or PF row is due until
T1991.

`GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game` reported 64.6% before and
after this batch.

## Batch 17 preregistered estimate (before reading tests T1089–T1138)

From the forecast remainder after T1088, I expect this 50-test slice to
contain R46/F1/C1/D2. I expect 15 tests to predate the oracle and 35 to be
later, with weak marks more likely in the older group.

### Batch 17 roster correction, before adjudication

The preregistration used raw `inventory.txt` lines 1089–1138. Reconciliation
against the ledger found 11 functions in that slice already recorded at earlier
IDs: `TestMudTimePassedMatchesC`, `TestMudTimePassedSubHourDoesNotCarry`,
`TestNewbieHometownRoom`, `TestNewCharacterConstructorConsumesZeroLevelDraws`,
`TestNewCharacterGetsDefaultClassTitle`, `TestNewMobAppliesLoadpos`,
`TestNewMobCarriesActFlags`, `TestNewMobDoesNotReRollStatBoosts`,
`TestNewMobLowLevelStatsEqualPrototype`, `TestNewMobSeedsInnateAffects`, and
`TestNonTakeNameDescriptionUnchangedByEquipCycle`. No duplicate ledger rows
will be added. The 39 unledgered functions in that raw slice remain in their
inventory order, followed by the next 11 unledgered functions, inventory lines
1139–1149, to retain a 50-function batch. The preregistered estimate applies
to the original raw slice; the corrected roster makes it a forecast only, not
a target for marks.


## Post-oracle redo, batch 17 (T1089–T1138)

The roster correction was recorded before adjudication: eleven names in raw inventory lines 1089–1138 already had ledger rows, so this batch contains the 39 unledgered names from that slice followed by the next 11 unledgered inventory entries. All 50 were introduced after the 2026-07-12 oracle date.

Marks: **R 43 / F 5 / C 2 / D 0.** Junk: **J3 5 / J4 2.** The initial estimate (R46/F1/C1/D2; 15 pre-oracle and 35 later) was for the original raw slice. On the corrected roster it overestimated R by 3 and D by 2, and underestimated F by 4 and C by 1; the date split was also wrong because every corrected-roster test is post-oracle.

Five F checks were repaired and mutation-proven in the detached worktree. T1109 now pins the full compile-time string; the old nonempty/year-only assertion stayed green when the formatter moved one day. T1117 now checks the exact retained first-line prefix; the old length-only assertion stayed green when truncation kept the suffix. T1118 now calls World.DoWrite against prototype-written and instance-written notes; the old ActionDescription-only assertion stayed green when the refusal was disabled. T1132 now pins the complete mover-then-follower arrival bytes; the old substring checks stayed green when a duplicate leader arrival was added. T1137 now tests land-to-water refusal without a boat; the old rows stayed green when the destination-sector gate was removed. Each fixed test went red under its mutation, and each mutation was restored; detail is in `mutations.tsv`.

T1097 (`TestMudDateRecordHasNoTrailingNewlineLikeC`) is C/J4 and folds into `TestWriteMudDateFormatMatchesC`, which pins the exact three-integer date record bytes. T1113 (`TestNoMoveEastPlayerLegacyLaneRemainsActive`) is C/J4 and folds into the east row of `TestNoMoveGuardPlayerLaneMatchesC`; that row now asserts the exact player line. The other 43 rows are R.

All seven non-R tests were introduced by Zach Greene with no `Co-Authored-By` trailer, so PF-152/PF-153 attribution is **unknown**: T1097, T1109, T1113, T1117, T1118, T1132, and T1137. No C disagreement was found. The selected C references and test comments were checked against `src/`; changes are test and audit documentation only, so no census was required.

`GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game` rose from 64.6% before the batch to 64.7% after it; the DoWrite gate test now reaches additional production statements. All 11 repository gates passed individually. No census was required because the changes are tests and audit documentation only.

## Batch 18 preregistered estimate (before reading tests T1139–T1188)

The forecast remainder after T1088 was R831/F18/C20/D34 across 903 rows. Batch 17’s preregistered slice was R46/F1/C1/D2; subtracting that forecast leaves R785/F17/C19/D32 over 853 tests. For this next 50-test batch I expect R46/F1/C1/D2, with all 50 post-oracle and weak marks more likely among older introductions.

### Batch 18 roster correction, before adjudication

The next 50 unledgered functions start after the prior corrected slice: inventory lines 1150–1210 contain 61 names, 11 of which already have ledger rows. Skipped duplicates are `TestObjNameUsesRuntimeKeywords`, `TestPerformDispose`, `TestPerformDisposeGold`, `TestPerformDropNoDrop`, `TestPerformGiveNoDrop`, `TestPerformMove_Exhausted`, `TestPerformMove_FollowersFollow`, `TestPerformMove_Sneak`, `TestPerformMove_TunnelFull`, `TestPerformRemoveNoDrop`, and `TestPerformWearTwoHanded`. The batch roster is the other 50 names in inventory order; no duplicate ledger rows will be added.

### Batch 18 roster correction, superseding the preceding correction (before adjudication)

The preceding correction also used the wrong inventory span. T1138 is
`TestNPCSimpleMoveCharmedRefusal` at inventory line 1149. Scanning forward and
skipping every function already present in the ledger, the next 50 are the
following tests in inventory order, from line 1150 through line 1232:

`TestNPCSimpleMoveTunnelLimit`, `TestNPCSocialCharLineReachesSwitchedBody`,
`TestNpcSpecialDeclineNeedsANonEmptyCommand`,
`TestNPCsShareDummyMobKillCountForBrag`, `TestNpcStealRejectsImmortalLevelVictim`,
`TestNPCTellReachesSwitchedMobileDescriptor`,
`TestObjectActivityLogsContainerWeightFloorAfterGet`,
`TestObjectActivityRunsOSOnPulseScripts`, `TestObjectActivityWeightMudlog`,
`TestObjectExtraFlagsStartFromPrototype`, `TestObjectLaneConsultsDeclineTables`,
`TestOrderedGroupFollowersWalksNewestFirst`, `TestPalmEntryDrawPrecedesEveryGate`,
`TestPCRaceAndClassTablesMatchC`, `TestPeekImmortalConsumesPercentDraw`,
`TestPeekPreservesCLeadingActLineBreakBytes`,
`TestPendingAttackerSwingsWithoutEnrolling`,
`TestPendingBodyPeacefulRefusalStillSkipsEnrollment`,
`TestPendingBodySecondDeathRunsFullDeathBranch`,
`TestPendingBodyStaysAddressableUntilExtractionPass`,
`TestPerformGetFromRoomQuietSuppressesRoomAct`,
`TestPerformWearWieldWeightBeforeTwoHanded`,
`TestPhysicalAffectsEquipRemoveAllLocations`,
`TestPhysicalAffectsMultipleItemsAndRemovalOrder`,
`TestPhysicalAffectsSerializationBoundary`, `TestPhysicalAffectsTimedAffectFolds`,
`TestPhysicalAffectsWrap`, `TestPissedAlchemistAttacksMageAfterDirectSay`,
`TestPissedAlchemistFirstPulseUsesDirectSayCommand`,
`TestPissedAlchemistRejectsPlayerCommandAndFightingMob`,
`TestPKDeathMudlogLevelIsLVLIMMORT`, `TestPKDeathMudlogMobKillerArm`,
`TestPKDeathMudlogProducer`, `TestPKDeathMudlogThroughCombatCallback`,
`TestPKDeathMudlogTypeIsBRF`, `TestPlacementDrawsSurviveCNumberOverflow`,
`TestPlayerAffectExpiryRemovesOneRecord`,
`TestPlayerDeathCountersStillRunWithNegativeCon`,
`TestPlayerEquipmentBadStatEntryPoints`, `TestPlayerEquipmentBadStatPriority`,
`TestPlayerEquipmentBadStats`, `TestPlayerEquipmentCharismaHunting`,
`TestPlayerEquipmentDexUsesDamageTail`, `TestPlayer_HitModifiers`,
`TestPlayerJoinAffectUsesNewestMatchingRecord`,
`TestPlayerPersistenceIsSeparateFromWorldSnapshot`,
`TestPlayerPresenceLineUsesTitleWithoutInventedClassFallback`,
`TestPlayerSaveData_ConcurrentSnapshot`,
`TestPlayerSelfDeathMudlogThroughCombatCallback`,
`TestPlayerToSaveDataKeepsStoredArrayOrder`.

The 33 skipped entries already have ledger rows; no T1139–T1188 rows were
written before this correction. This roster supersedes both earlier batch 18
roster descriptions above.

## Post-oracle redo, batch 18 (T1139–T1188)

The superseding roster correction above was applied before adjudication: 50
unledgered functions in inventory order, with already-ledgered names skipped.
All 50 were introduced after the 2026-07-12 oracle date. Earlier ledger and
notes rows remain unchanged; this batch appends T1139–T1188.

Marks: **R 42 / F 6 / C 1 / D 1.** Junk: **J3 5 / J2 1 / J4 1 / J9 1.**
The forecast was R46/F1/C1/D2 with all 50 post-oracle. Actuals had four fewer
R, five more F, the same C count, and one fewer D; the date split matched.

The six F rows were repaired and mutation-proven in the detached worktree.
T1146 now distinguishes object script order, carried/floor contexts, and the
fountain skip. T1147 reaches room 0 by extracting a carried object, rather
than assigning its location directly. T1151 adds the full-inventory palm gate.
T1152 compares the complete race and class tables. T1156 checks the exact
peaceful refusal line. T1163 checks effective birth before and after
serialization/re-equip. Every old body stayed green, every fixed body went red,
and all seven mutations were restored byte-for-byte; see `mutations.tsv`.

T1143 (`TestNpcStealRejectsImmortalLevelVictim`) is D/J9: its direct helper
branch is unreachable through C's `SPECIAL(thief)` caller, which filters
immortal victims before invoking the helper (`src/spec_procs.c:301-308,389-405`).
The test was deleted; `TestSpecThief_Golden` covers the reachable caller gate.
T1173 is C/J4 and folds into T1171, whose producer assertions include BRF
routing and negative observer boundaries. The remaining weak rows' PF-152 /
PF-153 attribution is unknown for T1146, T1147, T1151, T1156, T1163, and T1173;
T1152 is Claude Opus 5.5; T1143 is Cline.

The object actor lane has a separately tracked C/Go gap: C's bank and horn
object specials accept named commands, while Go does not dispatch them for NPC
actors. The decline-table test now asserts only unrelated command declines;
issue #3011 records the missing actor-aware path. No test pins Go bytes against
C, and no production code changed, so no census was required.

`GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game` measured 64.7% before and
64.6% after the batch. All 11 repository gates passed individually after the
batch edits and formatting.


## Batch 19 preregistered estimate (before reading tests T1189–T1238)

The forecast remainder after T1188 is R739/F16/C18/D30 across 803 tests. For
this next 50-test slice I expect R46/F1/C1/D2, with all 50 introduced after
the 2026-07-12 oracle date and weak marks more likely among older introductions.

The roster is the next 50 unledgered names in inventory order after T1188.
Inventory lines 1233–1291:

- `TestPlayerToSaveDataReversesLiveAffectList`
- `TestPlayerToSaveDataTruncatesLiveListAtMaxAffect`
- `TestPlayerTransferMountVariant`
- `TestPlayerUnmountClearsAffectWithoutCachedMountName`
- `TestPlayerWearOffMultiRecordSpeaksOnce`
- `TestPlayerWearRefusalOutputMatchesC`
- `TestPLRAccessorsShareOneStore`
- `TestPointUpdateConditionOutputThresholds`
- `TestPointUpdateCutthroatDamageTail`
- `TestPointUpdateDecaysObjectsInCarriedAndContainedLocations`
- `TestPointUpdateDustAndCircleTimerOutputAcrossLocations`
- `TestPointUpdateEmptyWorld`
- `TestPointUpdateExpiresAssignedMoonGateByValueThree`
- `TestPointUpdateFieldObjectUsesCRealRoomIndex`
- `TestPointUpdateHeldLightOutputMatchesC`
- `TestPointUpdateInactiveRegen`
- `TestPointUpdateMountedPoisonDamageTail`
- `TestPointUpdateNotifiesObserverOnce`
- `TestPointUpdatePoisonDamageTail`
- `TestPointUpdateRegeneratesNPCNana`
- `TestPointUpdateRunsGlobalObjectPassAfterAllMemoryDraws`
- `TestPointUpdateWoundedDamageTail`
- `TestPortalToTempleAndItohUseDoLook`
- `TestPostmasterDisabledDispatch`
- `TestPRFAndPLRBitsDoNotOverlap`
- `TestProbeObjectsStayUnregistered`
- `TestPulseOnlyTableHasCCommandGate`
- `TestPutFindsRoomContainer`
- `TestPutRoomLookupConcurrentWithRoomWrites`
- `TestQuanLoCancelledMobDeathStillGossips`
- `TestQuanLoMobDeathSendsFiveFingersToGlobalGossip`
- `TestQuanLoPlayerDeathSendsFiveFingersToGlobalGossip`
- `TestQuitMudlogLevelDiscriminators`
- `TestQuitMudlogProducers`
- `TestQuitMudlogTypeIsNRM`
- `TestRaceHateEquipmentRebuildAndRemoval`
- `TestRaceHateModifierSlotSemantics`
- `TestRaceSayUsesCControlBytesAndGateOrder`
- `TestRaceSayUsesCPlayerRaceLanguageTables`
- `TestRangedDamageConcurrentHPUpdates`
- `TestRangedDeathExpLimits`
- `TestRangedDeathHasNoKillerBookkeeping`
- `TestRangedDirectHPThresholdsWithoutWoundMessages`
- `TestRangedRetaliationHunterCallback`
- `TestRangedTransferPreservesPostureAndCircle`
- `TestRawKillClearsEverySpellAffect`
- `TestRawKillCombatantPlayerRunsSequence`
- `TestRawKillCorpseAttackDispatch`
- `TestRawKillCorpseHoldsEachItemExactlyOnce`
- `TestRawKillCorpseHoldsInventoryEquipmentAndGold`

## Post-oracle redo, batch 19 (T1189–T1238)

The batch contains R40/F8/C1/D1. Junk codes are J3=8, J4=1, and J9=1; all
50 tests were introduced after the 2026-07-12 oracle date. The preregistered
forecast was R46/F1/C1/D2, so actual-minus-forecast was R−6/F+7/C0/D−1.
Attribution is unknown for 46 rows and Claude Opus 5.5 for four. Among the ten
weak rows, nine are unknown and T1218 is Claude Opus 5.5.

All eight F tests were repaired in this PR and mutation-proven in the separate
detached worktree: T1191, T1194, T1197, T1205, T1211, T1218, T1219, and T1220.
Each original body passed, each fixed body failed the corresponding mutant,
and every mutation was restored byte-for-byte; see `mutations.tsv`. T1191 now
checks both rider and mount movement. T1194 and T1197 assert full output bytes;
T1205 checks dismount ordering; T1211 excludes destination-room output; and
T1218–T1220 require exactly one gossip delivery.

T1192 is D/J9: `Player.Unmount()` has no production callers. Its test was
deleted, with the production `Unmount(rider, mount)` behavior still covered by
`TestLegacyUnmountClearsAffectsButRetainsMasterLinks`. The dismount depth
manifest was updated to remove the deleted test from its proof list. T1237 is
C/J4 and duplicates the corpse-content uniqueness proof in T1238, which now
checks all four distinct object references. T1235 continues to cover callback
cleanup.

Every C citation in the batch and new comments was checked against `src/`.
`GOEXPERIMENT=nojsonv2 go test -cover ./pkg/game` reports 64.6% coverage. No
census was required because the batch changes tests and research manifests but
does not change production behavior.

All 11 repository gates passed individually: `make fmt`, `GOEXPERIMENT=nojsonv2
go build ./...`, `GOEXPERIMENT=nojsonv2 go vet ./...`, `GOEXPERIMENT=nojsonv2
go test ./...`, `golangci-lint run ./...`, `make modernize-check`, `make
deadcode-ratchet`, `make string-census`, `make fidelity-depth`, `make
expected-divergences`, and `make function-index`. The first fidelity-depth run
identified the stale T1192 manifest reference; the manifest update was followed
by a clean rerun. `make expected-divergences` exited 0 with its existing
unresolved-proof notices.

## Batch 20 preregistered estimate (before reading tests T1239–T1288)

The prior working forecast after T1188 was R739/F16/C18/D30 across 803 tests.
Subtracting batch 19's actual R40/F8/C1/D1 leaves a forecast remainder of
R699/F8/C17/D29 across 753 tests. For this next 50-test slice I expect
R46/F2/C1/D1, with all 50 introduced after the 2026-07-12 oracle date.

The roster is the next 50 unledgered names in inventory order after T1238.
Fourteen previously audited names encountered in this inventory span were
skipped (T191, T192–T195, T456, T462–T465, T476, and T482–T484). The selected
names span inventory lines 1292–1355:

- `TestRawKillDeathCryPrecedesCorpse`
- `TestRawKillDisintegrateDispatchesDust`
- `TestRawKillLuaBinding`
- `TestRawKillProtectionBackfire`
- `TestRawKillRaceDust`
- `TestRawKillRemovesTattoo`
- `TestRawKillTeardownOrder`
- `TestRawKillUndeadAndVampireDispatchDust`
- `TestRawKillVampireManaClamp`
- `TestReadDisposableMobRollsReadMobileDraws`
- `TestReadDisposableObjectRollsRareVariance`
- `TestReadMudDateCorruptRecordKeepsComputedCalendar`
- `TestReadMudDateMissingRecordKeepsComputedCalendar`
- `TestReadMudDateRecordOverridesCalendar`
- `TestReadMudDateSkippedWhenFixedTimePinned`
- `TestRealLevel`
- `TestRecallUnmountsBeforeTransferAndLeavesMountBehind`
- `TestRecruiterDirectTellUsesTellDeliveryGates`
- `TestReditWorldPublicationConcurrentSnapshots`
- `TestRefreshLiveObjInstancesDiscardsLiveExtraDescs`
- `TestRegenActiveSpellFlags`
- `TestRegisteredObjectsHaveUniqueIDs`
- `TestRemorterClassMatchesCMapping`
- `TestRemorterRefusalCarriesCsLineEnding`
- `TestRemoveAffectBitRetainsTimedRecordWithoutRestoringBit`
- `TestRemoveSetsCrashFlag`
- `TestRenderSkillListImmortalCatalogAndOverflow`
- `TestRenderSkillListManaColumn`
- `TestRenderSkillListMappedSkills`
- `TestRenderSkillListNinjaLearnedArts`
- `TestRenderSkillListPracticeHeader`
- `TestRentOutDropsHeldLightOnce`
- `TestRenumConvertsLegacyRemoveToObject`
- `TestRenumDisablesInvalidCommandsAndTheirDependents`
- `TestReplacingShopPreservesRuntimeBank`
- `TestReplayCrashLoadBadStatsBleedingLineIsRedAtSparseColor`
- `TestReplayCrashLoadBadStatsFiresPerEquippedItem`
- `TestReplayCrashLoadBadStatsHoldsRoomAtNowhere`
- `TestReplayCrashLoadBadStatsLethalConLossDraw`
- `TestReplayCrashLoadBadStatsLethalConLossSecondDraw`
- `TestReplayCrashLoadBadStatsLethalExpLossClamp`
- `TestReplayCrashLoadBadStatsLethalStopsAtRawKill`
- `TestReplayEquipmentBadStatsFiresPerWornItem`
- `TestResolveCharInRoomUsesAuthoredKeywords`
- `TestResolvePlayerWorldRefusesInvisible`
- `TestRestartDoesNotRestoreLegacyWorldSnapshot`
- `TestRestoreSkillsMigratesCatalogNames`
- `TestReviewButlerZoneLoadedOrder`
- `TestReviewCannotShowPreRestartGossip`
- `TestReviewCorpseSpillDoesNotRetickEarlierObject`

## Post-oracle redo, batch 20 (T1239–T1288)

Marks: **R 45 / F 3 / C 2 / D 0.** Junk: J3 3, J4 2, J7 1. All 50
introducing commits are after 2026-07-12, matching the batch forecast.

The preregistered batch estimate was R46/F2/C1/D1. Actuals were R45/F3/C2/D0:
one fewer R, one more F, one more C, and one fewer D than predicted. The
remaining forecast moves from R699/F8/C17/D29 over 753 tests to
R654/F5/C15/D29 over 703.

**F repairs and mutation proofs:**

- `TestReadDisposableObjectRollsRareVariance` (J3): seed 3 rolled 29, so the
  old body never reached C's conditional sign draw. Added a hit seed 5 (roll
  14) alongside the miss case. Omitting the sign draw kept the old test green
  and made the repaired test red.
- `TestReditWorldPublicationConcurrentSnapshots` (J3): the old concurrent
  readers checked only room VNUM/order while writers changed room names and
  other fields. Readers now inspect changed names and compare the final
  lock-free room with the locked snapshot. Mutating a published room name in
  place kept the old body green under `-race` and made the repaired body red
  with a data race.
- `TestRemorterRefusalCarriesCsLineEnding` (J3): `HasSuffix` accepted extra
  bytes before C's single refusal line. Exact output equality now pins the
  complete write. A prefixed extra send kept the old test green and made the
  repaired test red.

**C folds:** T1246 `TestRawKillUndeadAndVampireDispatchDust` was deleted as a
subset of `TestRawKillRaceDust`, which also checks carried/worn scatter and
output. T1286 `TestReviewButlerZoneLoadedOrder` was deleted as a duplicate of
`TestSpecButler_CanGetPredicateAndFourItemCap/four item cap`. The corpse-spill
decay regression remains.

Attribution of the five weak tests: four `unknown` (Zach-authored without a
model trailer) and one DeepSeek. The J7 row is `TestReadMudDateSkippedWhenFixedTimePinned`: it protects the Go `DP_FIXED_TIME` harness seam,
which has no C counterpart. No expected value disagreed with C. C paths and
line ranges cited in the reviewed test files were checked; the stale mud-date
comment range `src/comm.c:2624-2638` was corrected to the file's final line,
2637. This batch changes tests, research ledgers and notes only, so no census
was needed.

Batch 20 verification: `make fmt`, `GOEXPERIMENT=nojsonv2 go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
`make modernize-check`, `make deadcode-ratchet`, `make string-census`,
`make fidelity-depth`, `make expected-divergences`, and `make function-index`
all exited successfully, each run separately. `make fidelity-depth` initially
reported the deleted duplicate as a missing unit-test symbol; the corpse depth
row now names the retained `TestRawKillRaceDust`, and the rerun passed.
`make function-index-check` also passed after restoring generated index churn.
The `pkg/game` coverage report is 64.6%. `make expected-divergences` exited 0
with its existing unresolved-proof notices. All new ledger citations and
selected-test C comment ranges resolve to source lines. No census was needed.

Call-order clarification for T1255: C `spell_recall` transfers the player with
`char_to_room` and then unmounts. Go clears mounted state before
`PlayerTransfer` because that API follows mounted bodies. The test asserts the
C-equivalent final placement/affect/link state; Go comments now state this
adapter difference accurately rather than attributing Go's internal order to C.
The 11 gates were rerun after the recall call-order comment correction and
passed again; the two Go comments and test comment now cite C's actual
`char_to_room`-then-unmount order.


## Batch 21 preregistered estimate (before reading tests T1289–T1338)

The prior remainder forecast after T1288 is R654/F5/C15/D29 across 703 tests.
For this 50-test slice I predict R45/F3/C1/D1, with all 50 introduced after
the 2026-07-12 oracle date. That leaves a forecast of R609/F2/C14/D28 across
653 tests.

The roster is the next 50 unledgered names in inventory order after T1288. They
span inventory lines 1356–1440, skipping already-ledgered names in that span:

- `TestReviewGossipColorsAtComplete`
- `TestReviewGossipHidesHigherInvisibility`
- `TestReviewGossipRetainsTheCappedHistoryWindow`
- `TestReviewGossipTraversesHistoryOldestFirst`
- `TestRiderWalkMovesMountOnceNotTwice`
- `TestRollMaximumUsesCArgumentParser`
- `TestRollNumberUsesCUnsignedConversion`
- `TestRoomActivityConCroakControls`
- `TestRoomActivityConCroakKillsMortalInPlace`
- `TestRoomActivityConCroakUsesEffectiveCon`
- `TestRoomActivityEarlyReturnTopology`
- `TestRoomActivityFlowRoom`
- `TestRoomActivityLoudMobs`
- `TestRoomEnterScriptFiresOnce`
- `TestRoomFlagReadersAgreeOnRealHighFlagRooms`
- `TestRoomHasOtherOccupant`
- `TestRoomLookBracketsTheContentListsOnce`
- `TestRoomLookDescriptionTrailingSpacePrecedesTheBrackets`
- `TestRoomMessageRevertControl`
- `TestRoomMessageUsesActRecipientGate`
- `TestRoomObjectLinesPadToC63Bytes`
- `TestRoomObjectLinesStackIdenticalObjectsWithCountPrefix`
- `TestRoomOnDropFiresOnlyOnDrop`
- `TestRoomOnGetFiresFromRoomAndContainer`
- `TestRoomOnPulseRunsBeforeFlamingDamage`
- `TestRoomShownFiresOnlyWhenARoomIsRendered`
- `TestRoomSpecialLoopsIncludePendingBodies`
- `TestRoutineMutationConcurrentDoorWritersAndReaders`
- `TestRoutineMutationSkipsStructuralRebuild`
- `TestSameShopObjectComparesCRNUMCostExtraFlagsAndAffects`
- `TestSaveDataToPlayerHydratesAutoGoldMirrorsFromFlags`
- `TestSavePlayer_CreateFailureReturnsError`
- `TestSavePlayerRecordReplaysEquipmentBadStats`
- `TestSaySpellRendersThroughPerformAct`
- `TestSaySpellTargetLineRendersThroughAct`
- `TestScriptableGiveSetsFlag`
- `TestScriptAdapterLogIsBriefImmortalNoFile`
- `TestScriptZoneEchoOnlyVisitsAwakePlayingDescriptors`
- `TestScroungeHandoverKeepsObjectUnique`
- `TestScroungeObjectRegistered`
- `TestScroungeSetsFlag`
- `TestSelectLoginRoom`
- `TestSendToRoomUsesCRecipientGate`
- `TestSendToZoneDeliversBareBytesLiterally`
- `TestSetLevelRefreshesCarryCapacity`
- `TestSetLiveObjExtraDescConcurrentReaders`
- `TestSetLiveObjExtraDescPatchesAPreexistingSnapshot`
- `TestSetLiveObjExtraDescPublishesCloneAndPatchesInstances`
- `TestSetRoomFlagBitConcurrent`
- `TestShootRealDuplicateMobRetaliation`

## Post-oracle redo, batch 21 (T1289–T1338)

Marks: **R 43 / F 4 / C 3 / D 0.** All 50 introducing commits are after
2026-07-12. The preregistered estimate was R45/F3/C1/D1: actual-minus-forecast
was R−2/F+1/C+2/D−1. The prior remainder forecast was R654/F5/C15/D29 over
703 tests; after this batch it is R611/F1/C12/D29 over 653 tests.

**F repairs and mutation proofs:**

- `TestRoomLookBracketsTheContentListsOnce` (J5): the old body only inspected
  the observation result, never the delivered literal bytes. It now renders
  through the player message sinks and requires one opening and closing raw
  bracket pair. Routing bare observations through the formatted sink kept the
  old test green and made the repaired assertion red.
- `TestRoomLookDescriptionTrailingSpacePrecedesTheBrackets` (J5): the old
  body likewise did not render the result and could not observe whether the
  brackets reached the player. Complete-color and color-off cases now inspect
  the delivered bytes and pin the C trailing-space placement; the same sink
  mutation kept the old body green and made the fixed test red.
- `TestScroungeObjectRegistered` (J3): the old city-room, skill-zero setup
  never reached object creation. A seeded forest success now verifies one
  inventory object, its nonzero ID, registry growth and pointer identity.
  Removing the constructor's registry insertion kept the old body green and
  made the fixed body red.
- `TestSetLiveObjExtraDescConcurrentReaders` (J3): the old race witness only
  read slice length. Synchronized readers now inspect both description values
  while edits publish snapshots. Returning the underlying mutable slice kept
  the old body green and made the fixed body red under `-race`.

**C folds:** T1293 `TestRiderWalkMovesMountOnceNotTwice` was removed as
covered by T1061 `TestMountedMovementPairStateAndAudiences` (also recorded at
T543); its asserted walk outcome could not occur on the cited C path. T1307
`TestRoomMessageRevertControl` was removed as a redundant control for T1308's
recipient gate. T1323's unique invisible-caster and sleeping-target cases were
folded into T1322 `TestSaySpellRendersThroughPerformAct`.

The four weak tests and three folds were checked against the C call paths in
`src/`; the selected C citations in the ledger and modified tests were checked
against the named source lines. No expected value disagreed with C. This batch
changes tests and research records only, so no census was needed. `pkg/game`
coverage was 64.6% on the preregistered commit and remains 64.6% after this
batch.

## Batch 22 preregistered estimate (before reading tests T1339–T1388)

The remainder forecast after T1338 is R611/F1/C12/D29 across 653 tests. For
this 50-test slice I predict R46/F1/C1/D2, with all 50 introduced after the
2026-07-12 oracle date. That leaves a forecast of R565/F0/C11/D27 across 603
tests.

The roster is the next 50 unledgered names in inventory order after T1338.
They span inventory lines 1441–1543, skipping names already recorded in the
ledger:

- `TestShootTargetCMatcherAndRoomOrder`
- `TestShootTargetConcurrentMovement`
- `TestShopAccountPurchaseAndSaleBanking`
- `TestShopNPCGodLevelPaysAndPrefixSale`
- `TestShopPricesUseCFloatMarkup`
- `TestShopPriceUsesCIntegerOrder`
- `TestShopProducesObjectRejectsPrototypeLessObject`
- `TestShopRoomWildcardUsesCRealRoomIndex`
- `TestShopTradeExpressionMatchesCStackSemantics`
- `TestShopTradeWithMatchesCRefusalOrder`
- `TestShopTransactionAmountUsesDigitOnlyLeadingCount`
- `TestShowShopsKeeperRNumZeroCollision`
- `TestSkillDamageAttackTypeMatchesC`
- `TestSkillDamageNilAttacker`
- `TestSkillDamageSeamRunsFullJailGuardRedirect`
- `TestSkillDeathOrderExactBytes`
- `TestSkillKillCorpseUsesCSkillAttackType`
- `TestSkillResultDamageAttackTypePrefersCSkillMessageKey`
- `TestSkillStorageName_AllKujiSealsUseCommandSkillKeys`
- `TestSkillStorageName_JinUsesCommandSkillKey`
- `TestSkillToAttackTypePreservesDamageSpellIDs`
- `TestSleepingVisibilityNamesCharactersAndObjects`
- `TestSnakeSpecialDrawsOnlyWithSameRoomVictim`
- `TestSpawnMobWithLevelIAppliesCreateMobileScaling`
- `TestSpawnObjectToPlayerInventoryIUsesObjToCharPlacement`
- `TestSpawnObjectToRoomForSpellUsesObjToRoomPlacement`
- `TestSpecAlienElevatorMovesOnlyOpenEastRoom`
- `TestSpecAssignmentsHaveLegacyLanesForThePlayerWalk`
- `TestSpecBackstabber_MissSetsVictimFightingPosition`
- `TestSpecBat_Arms`
- `TestSpecBatRoom_WakesSleeping`
- `TestSpecBeholderFightingArmDraws`
- `TestSpecBeholderPeacefulLoneRoomDoesNotDraw`
- `TestSpecBeholderPeacefulPulseDraws`
- `TestSpecBrainEater_BeheadsCorpseAndLevelsInstance`
- `TestSpecBrainEater_DamrollGrowthAtLevelThirty`
- `TestSpecBrainEater_EntryGates`
- `TestSpecBrainEater_SkipsNonQualifyingObjects`
- `TestSpecBrassDragonBlocksThroughAct`
- `TestSpecBrassDragonUsesMoverRoomAndImmortalGate`
- `TestSpecButler_GenDoorCallBytesMatchC`
- `TestSpecButler_GenDoorUsesGenericFindInventoryFirst`
- `TestSpecCarrionActorSpawnsForTheActor`
- `TestSpecCastleGuardEast_AssistsOnlyEast`
- `TestSpecCastleGuardEast_BlockAndHit`
- `TestSpecCleric_Lspell12FallsToDefaultHarm`
- `TestSpecClerk_HometownZoneNames`
- `TestSpecClerkTellArgumentsMatchC`
- `TestSpecClerk_VisibilityAndUnknownZone`
- `TestSpecConductorAsksTicketOnAutonomousPulse`

## Batch 22 post-read audit (T1339–T1388)

Actual marks: **R44 / F4 / C1 / D1**. All 50 tests were introduced after the
2026-07-12 oracle date. The preregistered estimate was R46/F1/C1/D2, so the
actual variance was R−2/F+3/C0/D−1. Junk totals were J3=4, J4=1 and J9=1.
The planned remainder forecast after T1388 remains R565/F0/C11/D27 across
603 tests.

**False greens repaired and mutation-proven:**

- T1340 `TestShootTargetConcurrentMovement` (J3): the former loop allowed
  every lookup to return nil. A start barrier and a positive-hit assertion
  now require the concurrent lookup to find the moving player. An always-nil
  resolver kept the old body green and made the repaired test red.
- T1382 `TestSpecCastleGuardEast_AssistsOnlyEast` (J3): the former test only
  counted combat starts. It now pins the guard-to-peer-target pair; replacing
  the C target with the peer itself kept the old body green and made the new
  assertion red.
- T1383 `TestSpecCastleGuardEast_BlockAndHit` (J3): the former test likewise
  counted a combat start without checking its defender. It now pins
  guard-to-mover combat; a self-target mutation kept the old body green and
  made the repaired test red.
- T1384 `TestSpecCleric_Lspell12FallsToDefaultHarm` (J3): the old utterance
  substring accepted any spell. It now also requires the target's HP to fall;
  replacing default harm with poison kept the old body green and made the
  repaired assertion red.

**Fold and deletion:** T1358 `TestSkillStorageName_JinUsesCommandSkillKey`
(J4) was removed because T1357's all-seals table already asserts the Jin
mapping. T1352 `TestSkillDamageNilAttacker` (J9) was deleted: no production
caller passes nil, and the C `damage()` path dereferences `ch->in_room`
(`src/fight.c:1328-1333`), so this Go-only nil case has no reachable C or
player behavior to protect. No unique behavior was removed.

Attribution for the six weak-mark introductions (T1340, T1352, T1358, and
T1382–T1384) is **unknown** under PF-152/PF-153: each commit has Zach's git
identity and no `Co-Authored-By` trailer. All selected C citations in these
rows and the touched test comments were checked against the named source
lines and call paths. No expected value disagreed with C. The batch changes
tests and research records only; no census was required.

`pkg/game` coverage was 64.6% at the preregistered commit; the post-change
coverage remains 64.6% (`GOEXPERIMENT=nojsonv2 go test ./pkg/game -cover`).

## Batch 23 preregistered estimate (before reading tests T1389–T1438)

The planned remainder forecast after T1388 is R565/F0/C11/D27 across 603
tests. For this 50-test slice I predict R47/F0/C1/D2, with all 50 introduced
after the 2026-07-12 oracle date.

The roster is the next 50 unledgered names in inventory order after T1388.
They span inventory lines 1544–1590, skipping tests already present in the
ledger:

- `TestSpecConductorFightLinesThroughAct`
- `TestSpecConjured_AutonomousRegisteredDispatch`
- `TestSpecConjured_DefaultSpeechAudienceAndExtraction`
- `TestSpecConjured_EntryAndCharmGates`
- `TestSpecConjured_FizzleAudienceAndExtraction`
- `TestSpecConSeller_BranchOrderAndSuccessAudience`
- `TestSpecConSeller_DirectTellVisibilityAndOriginalCon`
- `TestSpecConSeller_EntryGatesAndCommandSurface`
- `TestSpecConSeller_InvisibleActorRoomWarning`
- `TestSpecConSellerTellArgumentsMatchC`
- `TestSpecCouchActPreservesCLineBreakBytes`
- `TestSpecCouch_ExactGate`
- `TestSpecCouch_ExtractsCouch`
- `TestSpecCouch_LookerRoom`
- `TestSpecCuchi_CommandGate`
- `TestSpecCuchi_OrdinaryPat`
- `TestSpecCuchi_OrodrethNoPromotion`
- `TestSpecDracula_EntryGatesAndDelegation`
- `TestSpecDracula_ExistingVampireOrWerewolfSkipsTransformation`
- `TestSpecDracula_LookBiteAudienceAndVampireState`
- `TestSpecDumpActorClearsRoomAndDeclines`
- `TestSpecElementalRoomAffectsEveryMortalOccupant`
- `TestSpecElementsGaleruAlive_EntryAndExactMobGate`
- `TestSpecElementsGaleruAlive_UsesExactVNumAndMovesNPCs`
- `TestSpecElementsGaleruColumn_AudienceLookAndRelocation`
- `TestSpecElementsGaleruColumn_RejectsNilActor`
- `TestSpecElementsGaleruColumn_RequiresAllExactTalismans`
- `TestSpecElementsGuardian_CommandlessAndNilEntryGates`
- `TestSpecElementsGuardian_PairUsesRoomOrderAudienceAndHit`
- `TestSpecElementsGuardian_SoloUsesSelfDamageAndActPronouns`
- `TestSpecElementsLoadCylinders_GetRemovesOnlyCurrentRoomCylinder`
- `TestSpecElementsLoadCylinders_MapsAllRegisteredPillars`
- `TestSpecElementsLoadCylinders_RejectsWrongTalismanAndExistingCylinder`
- `TestSpecElementsMasterColumn_EntryAndTalismanDestinations`
- `TestSpecElementsMasterColumn_PreservesCStaleCarryStateAndAudience`
- `TestSpecElementsMasterColumn_RejectsNilActor`
- `TestSpecElementsMinionActorPortsC`
- `TestSpecElementsMinion_UsesKeywordsNotVnumsAndSkipsInvisibleObjects`
- `TestSpecElementsMinion_UsesOrderedVisibleKeywordPasses`
- `TestSpecElementsPlatforms_EntryAudienceAndRelocation`
- `TestSpecElementsPlatformsMovesMobiles`
- `TestSpecElementsPlatforms_RejectsNilActor`
- `TestSpecElevatorMovesOnlyFirstTwoCharacters`
- `TestSpecElevatorThroughAct`
- `TestSpecEnterCircleCountsPlayersAndConsumesOtherLook`
- `TestSpecEnterCircleThroughAct`
- `TestSpecEqThief_EntryGates`
- `TestSpecEqThief_GetAndJunkBlackContainerItem`
- `TestSpecEqThief_StealsFirstVisibleItemAndPreservesDraws`
- `TestSpecEvilLead_NeverHandles`

## Batch 23 post-read audit (T1389–T1438)

Actual marks: **R39 / F7 / C1 / D3**. All 50 introductions postdate the
2026-07-12 oracle. The preregistered estimate was R47/F0/C1/D2, so the actual
variance was R−8/F+7/C0/D+1. The batch reviewed 50 tests and leaves 553 tests after T1438. Junk totals overlap: J3=3, J4=1, J5=1, J9=5.

**False greens repaired and mutation-proven:**

- T1389 `TestSpecConductorFightLinesThroughAct` (J3): the old test sampled
  only draws 1 and 4 despite its “every fight line” comment, and accepted
  extra output. It now checks all five C lines as exact per-draw output. A
  duplicate Act kept the old body green and made the repaired test red.
- T1409 `TestSpecDumpActorClearsRoomAndDeclines`: the old substring assertion
  allowed duplicate bytes. It now pins the exact single vanish line. A
  duplicate room Act kept the old body green and made the repaired test red.
- T1410 `TestSpecElementalRoomAffectsEveryMortalOccupant` (J3): the old
  fixture included only one mortal. It now verifies a second mortal's damage
  and fire/death messages. Skipping that occupant kept the old body green and
  made the repaired test red.
- T1411 `TestSpecElementsGaleruAlive_EntryAndExactMobGate`: the live-Galeru
  refusal previously checked return and location but not silence. It now
  requires no output; a live-gate warning kept the old body green and made
  the repaired test red.
- T1415 `TestSpecElementsGaleruColumn_RequiresAllExactTalismans`: the partial
  setup previously checked only fallthrough, output, and character locations.
  It now checks that each placed talisman remains in its original room. An
  extraction on the incomplete branch kept the old body green and made the
  repaired test red.
- T1416 `TestSpecElementsGuardian_CommandlessAndNilEntryGates` (J9): the old
  test called the player wrapper with a nil actor, so the nil guard masked the
  commandless gate. It now drives the registered actor adapter with the nonnil
  mobile pulse actor and checks silent decline. Removing the adapter command
  gate kept the old body green and made the repaired test red.
- T1433 `TestSpecEnterCircleCountsPlayersAndConsumesOtherLook` (J3/J9): the
  previous non-circle look assertion checked only `true`, so the special
  could consume the command without calling `do_look`. It now clears the
  crowd response and requires look output. A return-true/no-look mutant kept
  the old body green and made the repaired test red.

**Fold and deletions:** T1398 `TestSpecConSellerTellArgumentsMatchC` (J4/J5)
was removed: its internal `do_tell` call seam duplicated player-visible
responses already asserted by T1395, T1394 and T1397. T1414
`TestSpecElementsGaleruColumn_RejectsNilActor`, T1424
`TestSpecElementsMasterColumn_RejectsNilActor`, and T1429
`TestSpecElementsPlatforms_RejectsNilActor` (J9) were deleted. Each called a
private Go wrapper with nil where C's active room-command path supplies a
character; no C/player behavior is associated with that input. The depth
manifest now points to the reachable command-entry proofs for master column
and platforms and drops the nil-only Galeru column row. No live behavior was
left without proof.

All eleven weak-mark introductions (seven F, one C, three D) have `unknown`
author attribution under PF-152/PF-153: the introducing commit has Zach's git
identity and no `Co-Authored-By` trailer. The current branch/PR was recorded
separately and not used to infer an author. No new expected value disagreed
with C. `TestSpecCuchi_OrodrethNoPromotion` remains the documented,
owner-approved DP-1373 divergence: the expected purr/no-promotion behavior is
intentional (PF-051), not an unfiled J10 finding. The batch changes tests,
test-audit records, and depth evidence only; no census was required.

`pkg/game` coverage was 64.6% at the preregistered commit and remains 64.6%
after these changes (`GOEXPERIMENT=nojsonv2 go test ./pkg/game -cover`).


## Batch 24 preregistered estimate (before reading tests T1439–T1488)

Before reading this batch, I compared the frozen inventory against all ledger
base names. The inventory has 1,991 distinct functions. T001–T1438 has 1,438
rows but only 1,366 distinct function names: 72 rows repeat a function already
recorded. That leaves **625 unique inventory functions** without a ledger row,
not the 553 forecast after T1438. I will preserve all prior rows under the
append-only rule and audit every remaining function, so the ledger will end at
T2063 if the inventory stays unchanged. The first-50 roster is the earliest
unledgered names in inventory order, scanning from the beginning and skipping
all functions already in the ledger; this supersedes the implicit cursor at
inventory line 1601 used by the prior roster.

The roster spans inventory lines 234–366 (50 unledgered functions):

- `TestDamageEntryBreaksFollower`
- `TestDamageNewbieExperienceBeforeMessage`
- `TestDamageRefusedNPCAttackerIsSilent`
- `TestDamageRefusedPeacefulRetaliation`
- `TestDamageRefusedProtections`
- `TestDamageRefusedShopkeeperSpecials`
- `TestDeathCryBoundedCallers`
- `TestDeathCryMuffledBehindClosedDoors`
- `TestDeathFollowerCleanupBeforeDropAndRemoval`
- `TestDeathFollowerOwnMasterAndRemoteFollower`
- `TestDeathObservationCoversGuardedAndRawPaths`
- `TestDeathPreservesCrashFlag`
- `TestDeathReleasesFollowers`
- `TestDeserializeMigratesVersion1Flags`
- `TestDiagConditionMatchesCThresholds`
- `TestDieWithKillerBothConLossDrawsAreUnfloored`
- `TestDieWithKillerConLossHasNoFloor`
- `TestDisarmVictimFlagDirect`
- `TestDisbandWalkIncludesMobAndKeepsCharmedFollow`
- `TestDismountDepthMatchesCOrderAndAffectUnmount`
- `TestDoActionFlirtSleepingTargetHitsPositionGate`
- `TestDoActionFondleSleepingTargetHitsPositionGate`
- `TestDoBearhug_SignedCharRollAbove127HitsLikeReferenceC`
- `TestDoCircle_PositionMultiplier`
- `TestDoClanBankOverflowWithdrawMatchesC`
- `TestDoClanInfoIncludesDefinedCDetails`
- `TestDoClanInfoListIncludesDefinedCHeading`
- `TestDoClanRenameOverflowRefusesLikeC`
- `TestDoClanSetAppfeeOverflowRefusesLikeC`
- `TestDoCutthroatGateOrder`
- `TestDoCutthroatImmortalTargetOverridesCaster`
- `TestDoCutthroatNoSkillDoesNotDraw`
- `TestDoCutthroatSuccessAndFailureContracts`
- `TestDoDamage_WornApplySpellProvidesSanctuary`
- `TestDoDetectDepthFailureMessageWaitAndRollRange`
- `TestDoDetectDepthGatesAndOutput`
- `TestDoDetectDepthSuccessOrderAndSecretMark`
- `TestDoDiagnoseUsesFightingFallbackAndPrivateOutput`
- `TestDoDisarmDepthFailureState`
- `TestDoDisarmDepthPlayerSuccess`
- `TestDoDisarmDepthRetaliatesWhenVictimWasNotFighting`
- `TestDoDisarmDepthWeaponGatePrecedesCombatGate`
- `TestDoDisembowelDepthUsesCMessageAndDrawOrder`
- `TestDoDisembowel_Mounted`
- `TestDoDisembowel_NoWeapon`
- `TestDoDisembowel_SkillRollFailureUsesDamageMessagePath`
- `TestDoDisembowel_WrongWeapon`
- `TestDoDragonKick_BlocksInsufficientMove`
- `TestDoDragonKick_MissDrawOrder`
- `TestDoDragonKick_WaitStateAlways22Pulses`

From recent batch results, my prediction is **R43 / F5 / C1 / D1**. I expect all
50 to postdate the 2026-07-12 oracle cutoff; I will verify dates from their
introducing commits during the audit.

## Batch 24 post-read audit (T1439–T1488)

Actual marks: **R42 / F7 / C0 / D1**. The preregistered estimate was
R43/F5/C1/D1 (variance R−1/F+2/C−1/D0). All 50 tests were introduced after
the 2026-07-12 oracle cutoff. Junk marks overlap: J3=8 (the seven F rows and
T1457), J8=1 (T1457). No all-R spot checks applied because the batch contains
F marks. The ledger now has 1,488 rows; scanning the inventory by unique test
name leaves 575 unledgered tests because 72 earlier rows repeat an existing
name.

**False greens repaired and mutation-proven:**

- T1443 `TestDamageRefusedProtections` (J3): changed the substring check to
  exact C CRLF bytes. An extra protection Act kept the old body green and made
  the repaired test red.
- T1448 `TestDeathFollowerOwnMasterAndRemoteFollower` (J3): replaced a prefix
  check with exact outgoing-master bytes. An extra room line kept the old body
  green and made the repaired test red.
- T1451 `TestDeathReleasesFollowers` (J3): require exactly one private
  stop-following line. Duplicating the private Act kept the old body green and
  made the repaired test red.
- T1463 `TestDoClanBankOverflowWithdrawMatchesC` (J3): assert the C overflow
  withdrawal’s player gold and clan treasury state as well as its
  acknowledgement. Restoring the prior negative-amount no-op kept the old body
  green and made the repaired test red.
- T1464 `TestDoClanInfoIncludesDefinedCDetails` (J3): compare the complete
  detail transcript, including C’s CRLF then LF plan ending. Restoring the
  extra CRLF kept the old body green and made the repaired test red.
- T1466 `TestDoClanRenameOverflowRefusesLikeC` and T1467
  `TestDoClanSetAppfeeOverflowRefusesLikeC` (J3): require exact C refusal
  bytes. A prefixed refusal kept each old substring assertion green and made
  each repaired test red.

**Deletion and replacement:** T1457 `TestDisbandWalkIncludesMobAndKeepsCharmedFollow`
(J3/J8) copied the production follower loop and therefore could not catch a
regression in the player command. Deleted it and added
`TestCmdUngroupDisbandIncludesMobileAndPreservesCharmedFollow` in `pkg/session`,
which drives the actual command and checks exact player output, group flags,
mobile inclusion, and charm-preserved following. Added
`TestCharacterUngroupDisbandsMobileFollowers` to retain direct game-layer
coverage of the production mobile actor body. Both deliberate mutations (skip
mobile followers; leave the leader’s AFF_GROUP bit set) kept the old handwritten
body green and made the command-path test red. The production command now uses
`game.SetGroupedActor` to clear the leader’s group state as C does.

The stricter clan assertions exposed two C-parity defects, both corrected in
this batch: an overflowing bank withdrawal wraps through C `atoi` to the
negative amount, so `SpendTreasure` must apply the negative balance change;
and clan info must end its plan with C’s `\r\n\n`, not an extra CRLF. No
expected divergence remains unfiled, and no governing fidelity rule was
changed. The owner-approved DP-1418 death-cry divergence remains documented;
it is not a new J10 finding.

Weak-test authorship under PF-152/PF-153: T1443 has the Claude Opus 5.5 trailer.
T1448, T1451, T1457, T1463, T1464, T1466, and T1467 have Zach’s git identity
without a model trailer and remain `unknown`; branch names were not used to
infer an author. Two other tests in the batch were authored by Luna, but both
were R. The weak rows were all post-oracle: pre-oracle 0, post-oracle R42/F7/D1.

`pkg/game` statement coverage was 64.7% before the batch and remains 64.7%.
All eleven required PR gates passed. `make string-census` passed its ratchet;
it reported the generated census reports as stale and they were not refreshed.
The targeted draw-parity census
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-ta-game-24-targeted`
was CLEAN: `clan-atoi-depth`, `clan-depth`, and `ungroup-disband-depth` all
passed, with 0 draw divergences. Its frozen source hashes match the production
and depth files changed by this batch.

## Batch 25 preregistered estimate (before reading tests T1489–T1538)

The batch continues the frozen inventory-order sweep: select the first 50 test
names not already present in any ledger row, scanning from the start of the
1,991-function inventory. T001–T1488 has 1,488 rows but 1,416 unique names,
leaving 575 unique inventory functions unledgered. The selected roster is
inventory lines 367–507:

- `TestDoDreamMatchesCOrderAudienceAndVisibility`
- `TestDoExitsBlindnessGate`
- `TestDoExitsInfravisionSeesDarkDestination`
- `TestDoFleeNpcWaitGateDoesNotDrawRandomness`
- `TestDoFollowShadowConsumesSkillRollDraw`
- `TestDoFollowShadowFailureUsesOrdinaryFollowAudience`
- `TestDoFollowShadowSuccessAppliesQuietAffect`
- `TestDoFollowShadowSwitchReplacesPriorAffect`
- `TestDoGenDoorFarSideActSkipsSleepingAndWritingObservers`
- `TestDoGenDoorFillWordOnlyArgument`
- `TestDoGroinripDepthGateOrder`
- `TestDoGroinripDepthResultContract`
- `TestDoHideDaytimeSectorGates`
- `TestDoHouseHandlesMalformedAndFullGuestControls`
- `TestDoHouseMatchesCArgumentAndStateBoundaries`
- `TestDoInactiveToggleState`
- `TestDoInvisUsesLevelAndThresholdAudience`
- `TestDoKick_WaitStateAlways22Pulses`
- `TestDoKujiKiri_JinFailureJoinsAsInertRecord`
- `TestDoKujiKiri_JinGuardMessages`
- `TestDoKujiKiri_JinSuccessPreservesCDefaultSecondAffect`
- `TestDoKujiKiri_JinUnmasteredConsumesCSuccessDraw`
- `TestDoKujiKiri_KaiFailureKeepsPrimaryLockout`
- `TestDoKujiKiri_KaiSuccessPreservesTwoCRecords`
- `TestDoKujiKiri_KyoFailureKeepsPrimaryLockout`
- `TestDoKujiKiri_KyoSuccessPreservesTwoCRecords`
- `TestDoKujiKiri_RinFailureMessages`
- `TestDoKujiKiri_RinSuccessMessages`
- `TestDoKujiKiriShaCapsAtMaxHP`
- `TestDoKujiKiriShaFailureLeavesNoLockout`
- `TestDoKujiKiriShaUsesLowLevelHealFloor`
- `TestDoLookForSpellGates`
- `TestDoLookInHeaderUsesCFName`
- `TestDoLookTargetSyntheticExtraDescHasNoExtraBlankLine`
- `TestDoMindlinkDepthGates`
- `TestDoMindlinkDrainedLineIsBare`
- `TestDoMoldRewritesLiveObjectFields`
- `TestDonateArrivalUsesActObjectVisibilityAndSleepAudience`
- `TestDonateGoldArrivalUsesActDarknessVisibility`
- `TestDonateGoldConcurrentJunksOnce`
- `TestDonatePlacementPrependsInDonationRoom`
- `TestDoNeckbreakDepthFailureRetaliationContract`
- `TestDoNeckbreakDepthGateOrder`
- `TestDoNeckbreakDepthPeacefulUsesCanonicalRoomBit`
- `TestDoNeckbreakDepthResultContractAndDrawOrder`
- `TestDoNewbieChannelDepthGatesAndFanout`
- `TestDoOpenRejectsKeywordPrefix`
- `TestDoorFarSideNameUsesCFName`
- `TestDoQuitRecordsLoadRoomThroughLVLIMMORT`
- `TestDoRideDirectGates`

Before reading these tests, my estimate is **R43 / F5 / C1 / D1**.

## Batch 25 post-read audit (T1489–T1538)

Actual marks: **R39 / F10 / C1 / D0**. The preregistered estimate was
R43/F5/C1/D1 (variance R−4/F+5/C0/D−1). All 50 tests were introduced after
the 2026-07-12 oracle cutoff. Junk marks overlap: J3=9, J9=1 (T1506), and
J4=1 (T1508). No all-R spot checks applied because the batch contains F marks.
The batch’s only C row, T1508 `TestDoKujiKiri_JinGuardMessages`, duplicated the
common class, fighting, lockout, mounted, and mastery refusals already covered
by `TestDoKujiKiri_NotNinja`, `TestDoKujiKiri_Fighting`,
`TestDoKujiKiri_AlreadyAffected`, `TestDoKujiKiri_Mounted`, and
`TestDoKujiKiri_SealNotMastered`; the Jin pre-mastery draw remains covered by
`TestDoKujiKiri_JinUnmasteredConsumesCSuccessDraw`. The duplicate was deleted,
and the live `jin.tsv` rows now cite the surviving guard tests. The ledger has
1,538 rows; scanning the 1,991 unique inventory names leaves 525 unledgered
tests.

**False greens repaired and mutation-proven:**

- T1498 `TestDoGenDoorFillWordOnlyArgument`: require the exact fillword-only
  open/close and empty-open transcripts. An extra refusal line kept the old
  checks green and made the repaired test red.
- T1520 `TestDoLookForSpellGates`: require exact stars and blind gate bytes. An
  added suffix kept the old substring check green and made the repaired test
  red.
- T1521 `TestDoLookInHeaderUsesCFName`: compare the exact first container
  header line. Added header text kept the old substring check green and made
  the repaired test red.
- T1530 `TestDoNeckbreakDepthFailureRetaliationContract`: require exact actor,
  victim, and room act text. Added actor text kept the old nonempty check green
  and made the repaired test red.
- T1533 `TestDoNeckbreakDepthResultContractAndDrawOrder`: require the exact
  seeded 18d(level) result and next draw. Halving damage after consuming the
  same dice kept the old positive-damage/draw check green and made the repaired
  test red. The replay consumes the percent and damage rolls once each.
- T1500 `TestDoGroinripDepthResultContract`: require the exact C room line. An
  added suffix kept the old substring check green and made the repaired test
  red.
- T1505 `TestDoInvisUsesLevelAndThresholdAudience`: compare the complete
  per-observer message list. A duplicated disappearance act kept the old
  substring check green and made the repaired test red.
- T1535 `TestDoOpenRejectsKeywordPrefix`: require only the exact no-door
  refusal and closed exit state. An extra line kept the old checks green and
  made the repaired test red.
- T1536 `TestDoorFarSideNameUsesCFName`: require the exact observer transcript,
  including C’s blank line. Appended text kept the old substring check green
  and made the repaired test red.
- T1506 `TestDoKick_WaitStateAlways22Pulses`: require a deterministic skill-1
  miss rather than skipping after repeated hits. A forced hit arm kept the old
  test green and made the repaired test red.

`pkg/game` statement coverage was 64.7% before the batch and remains 64.7%.
All eleven required PR gates passed: `make fmt`, build, vet, full tests,
`golangci-lint` v2.15.0, modernize, deadcode ratchet, string census, fidelity
depth, expected-divergences, and function-index. The string census ratchet
passed while reporting its generated reports as stale; they were not refreshed.
The fidelity-depth gate initially found four Jin manifest rows pointing to the
deleted duplicate; those rows were redirected to the surviving exact test
declarations and the gate then passed. No oracle census was needed for this
test and documentation audit.

## Batch 26 preregistered estimate (before reading tests T1539–T1588)

Continue the frozen inventory-order sweep: select the first 50 test names not already present in any ledger row, scanning from the start of the 1,991-function inventory. T001–T1538 has 1,538 rows and 1,466 unique test names, leaving 525 unique inventory functions unledgered. The selected roster is inventory lines 508–580:

- `TestDoRideMountedGates`
- `TestDoRideResolvesThroughGetCharRoomVis`
- `TestDoRideSuccessStateAndAudience`
- `TestDoRideUsesCOneArgument`
- `TestDoRoflSelfTargetUsesCCharAutoBytes`
- `TestDoSaveWritesStoreOfRecordWithNowhere`
- `TestDoScoutTerrainAndCrowdDescriptions`
- `TestDoScroungeDepthDeferredImprovePreservesDrawOrder`
- `TestDoScroungeDepthGatesAndDefault`
- `TestDoScroungeDepthSectorMapAndMessages`
- `TestDoSerpentKickGateOrder`
- `TestDoSerpentKickMissUsesCMessageContract`
- `TestDoSerpentKickResultContract`
- `TestDoSerpentKickSleepingTargetAutoHits`
- `TestDoSharpenConsumesOneCPercentDraw`
- `TestDoSharpenFailureMutatesEvenWithZeroSkill`
- `TestDoSharpenFightingGateFollowsObjectGates`
- `TestDoSharpenMutatesAndScalesSuccessfulWeapon`
- `TestDoSharpenRejectsNonSlashAndAlreadyAffected`
- `TestDoSharpenUsesCObjectGatesAndCarriedOnly`
- `TestDoSleeperDepthGateOrder`
- `TestDoSleeperMobProtectionAndSuccessShape`
- `TestDoSleeperPlayerLevelWindow`
- `TestDoSlug_GateOrderAndMessages`
- `TestDoSlug_ResultContract`
- `TestDoSlug_SkillMessageDrawOrder`
- `TestDoSmackheads_GateOrderAndMessages`
- `TestDoSmackheads_ResultContract`
- `TestDoSneakFailedRerollClearsSneakAndStealthAffects`
- `TestDoSneakMountedGatePrecedesRoll`
- `TestDoSpellDamageChargePainDraw`
- `TestDoSpellDamageCircleUsesCircleCorpseType`
- `TestDoSpellDamageEmitsMeteorSwarmMessages`
- `TestDoSpellDamageEmitsMeteorSwarmMissToPlayerCaster`
- `TestDoSpike_ImmortalTargetGate`
- `TestDoSpike_MissWaitState`
- `TestDoSpike_MobTarget`
- `TestDoSpike_SuccessProbability`
- `TestDoSpike_UsesObjectKeywords`
- `TestDoSpike_WeaponCheckUsesFirstKeyword`
- `TestDoStrike_HitContract`
- `TestDoStrike_MissRoutesThroughSkillMessage`
- `TestDoStrike_SleepingVictimRaisesProbTo100`
- `TestDoSubdueDepthGateOrder`
- `TestDoSubdueImmortalSuccess`
- `TestDoSubdueMissPinsAndShape`
- `TestDoSubdueOutlawRefusal`
- `TestDoSubdueSeededSuccessDrawCount`
- `TestDoSubdueShopkeeperGate`
- `TestDoTigerPunchDamage_SelectsVariantAfterHpUpdate`

Before reading these tests, my estimate is **R43 / F5 / C1 / D1**.

## Batch 26 audit (T1539–T1588; introductions 2026-08-30–2026-11-08)

The preregistered estimate was R43/F5/C1/D1. Actual: **R40/F10/C0/D0**;
all 50 tests were introduced after the 2026-07-12 oracle start. The estimate
missed five additional false greens and predicted one C and one D that the
review did not find. All ten weak tests were J3 assertion gaps, and their
introducing commits attribute to `unknown` under the audit rules.

I strengthened ten tests and proved each one in a detached mutation worktree
based on the refreshed Batch 25 tip. For every case the original body was
green under the mutation, the repaired body went red, and both production and
test files were restored byte-for-byte:

- T1559 combines fighting and mounted states so sleeper must keep C's earlier
  fighting refusal.
- T1560 now checks the aware/nosleep failure and success messages for all
  audiences exactly.
- T1563 derives slug damage from the same seeded C `number(1,4)` draw instead
  of accepting a broad interval; T1564 pins the exact set-146 room line as well
  as draw order.
- T1571 checks exact meteor-swarm hit/death output, including C's death-room
  tail. T1572 checks exact caster and room miss bytes.
- T1575 checks spike's exact three-audience acts before its raw-kill tail.
- T1582 combines fighting and mounted states so subdue preserves C's busy gate
  precedence. T1584 checks all three exact miss acts before retaliation.
- T1588 crosses the POS_DEAD threshold and requires set 189's exact death line,
  so selecting hit_msg before the HP update now fails.

No tests were deleted or folded, so no live inventory references needed
retargeting. The new ledger citations were checked against `src/` and the
read-only C message files. No production behavior changed and no oracle census
was needed. `pkg/game` statement coverage was 64.7% before and after.

I rebased this stack on refreshed Batch 25 tip `98715c15c` after its main merge.
The thirteen previously corrected rows match `origin/main` byte-for-byte, and
T1403 remains raw tab-separated text without spreadsheet quoting.

All eleven gates passed on the refreshed branch: `make fmt`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint` v2.15.0,
`make modernize-check`, `make deadcode-ratchet`, `make string-census`,
`make fidelity-depth`, `make expected-divergences`, and
`make function-index-check`. The string census ratchet passed with its
pre-existing generated reports marked stale. Expected-divergences exited 0
while reporting its existing unresolved proof selectors; the function-index
ratchet's expected fake-association red was caught and the ratchet ended clean.

Through T1588, T639–T1588 totals R826/F96/C20/D8. Cumulative T001–T1588 totals
are R1459/F100/C20/D9: the pre-oracle split is R379/F7/C3/D1 and the post-
oracle split is R1080/F93/C17/D8. No package summary or PF ledger row is due
until T1991; when it is due, use PF-168 or later because PF-167 is already
assigned.

## Batch 27 preregistered estimate (before reading tests T1589–T1638)

Continue the frozen inventory-order sweep: select the first 50 test names not already present in any ledger row, scanning from the start of the 1,991-function inventory. T001–T1588 has 1,588 rows and 1,516 unique test names, leaving 475 unique inventory functions unledgered. The selected roster is the next 50:

- `TestDoTigerPunch_HitContract`
- `TestDoTigerPunch_LegacyGatesPreserved`
- `TestDoTigerPunch_MissRoutesThroughSkillMessage`
- `TestDoTransformMatchesCStateMachines`
- `TestDoTurnDisintegratesEligibleUndeadAtFifteenLevelAdvantage`
- `TestDoTurnEarlyGatesDoNotDrawRandomness`
- `TestDoTurnFleesPlayerByNumericRaceThroughPlayerMovement`
- `TestDoTurnFleeUsesNpcDirectMovementAndPreservesDirectionDraw`
- `TestDoTurnLowManaSubtractsQuarterCostAfterRoomActs`
- `TestDoTurnScansCOrderedRoomBodiesByNumericRace`
- `TestDoUseEquipdQuirkStaffTap`
- `TestDoUseEquipdQuirkWandPoint`
- `TestDoUseEquipdQuirkWholeRemainder`
- `TestDoUseHeldFallbackWhenArgDoesNotMatch`
- `TestDoUseWandPassesManualSpellArgumentToControlWeather`
- `TestDoVisibleInvisAndSneakTogether`
- `TestDoVisiblePreservesCEntryGateOrder`
- `TestDoVisibleRemovesSneakAndStealthAffects`
- `TestDoVisibleStripsInvisSpellAffect`
- `TestDoWimpyUsesCIsdigitAndOneArgument`
- `TestDoWriteMissingObjectRefusals`
- `TestDoYank_SuccessRoomLineSkipsVictim`
- `TestDreamPointUpdateOutputBranches`
- `TestDreamTravelDrawOrderMatchesC`
- `TestDrinkDoseConsumedOnce`
- `TestDropGoldConcurrentWithGive`
- `TestDropPlacementsPrependInCOrder`
- `TestDummyMobSkillsUseSharedCState`
- `TestE2AttrAffectBoundaries`
- `TestFindObjectByNameUsesCGlobalObjectListOrderAndTokens`
- `TestFireMobScriptsShareDispatchContext`
- `TestFirstPlayerGodHasMartialArts`
- `TestFixedMailTextUsesFirstNULTerminator`
- `TestFleshAlteredTypeBands`
- `TestFleshAlterUnwieldSetsFlag`
- `TestFollowMobBodyCharmedGuardEmitsMasterMessage`
- `TestFormatCompileTimeMatchesC`
- `TestFullMoonTransformsWerewolf`
- `TestGameDiagnosticCrossRoomDamage`
- `TestGameDiagnosticDeathTrapMovement`
- `TestGameDiagnosticKillMilestone`
- `TestGameDiagnosticProtectionEvil`
- `TestGameDiagnosticProtectionGood`
- `TestGenBoardActorDeclinesDescriptorlessMobiles`
- `TestGenBoardEmptyHeadlinesDoNotExhaustSlots`
- `TestGenBoardFullBoardWriteRefusalOutput`
- `TestGenCommChannelColorMatchesC`
- `TestGenCommMobGossipColorMatchesC`
- `TestGenericFindEquipFirstInventoryBeforeRoom`
- `TestGenericFindEquipFirstLookAtTarget`

Before reading these tests, my estimate is **R43 / F5 / C1 / D1**.

## Batch 27 audit (T1589–T1638; introductions 2026-09-04–2026-11-16)

The preregistered estimate was R43/F5/C1/D1. Actual: **R46/F4/C0/D0**;
all 50 introductions are post-oracle. The estimate missed one additional R and
one fewer false green, and predicted one C and one D that the audit did not
find. All four weak-test authors are `unknown` under the commit-author and
trailer attribution rules.

I repaired four assertion gaps and proved each in a detached mutation
worktree. In every case the original test stayed green, the repaired test went
red, and the production mutation was restored byte-for-byte:

- T1590 now requires the exact wielded tiger-punch refusal punctuation.
- T1609 now compares exact output for all three missing-object refusal paths.
- T1614 checks Dropper, Giver, and Victim balances plus the one 60-coin floor
  pile, so the concurrent transfer still accounts for all 200 starting coins.
- T1619 records and checks the Lua-facing actor/mob references, room, and World
  context for both fight and death dispatches, not only the compatibility
  `Ch` name.

No tests were deleted or folded, so no live inventory reference needed
retargeting. T1633 records the approved DP-1388 Go behavior for empty board
headlines; C reserves the slot before checking an empty heading, while the
approved Go path releases it. This is recorded as an explicit hand-written
exception, not a new C parity claim.

The ledger and mutation log were appended as raw tab-separated text. The 13
corrected rows (T360, T377, T387, T391, T394, T395, T455, T465, T520, T535,
T548, T606, T769) and T1403 match `origin/main` byte-for-byte. The refreshed
branch is based on `origin/main` at `ab14720f9` (Batch 26 merged). Package
statement coverage was 64.7% before and after. All eleven required gates
passed; `make string-census` reports its existing generated reports as stale,
`make expected-divergences` exits clean with existing unresolved proof
selectors, and `make fidelity-depth` reports the current repository-wide
status. No production code changed and no census was needed.

Through T1638, T639–T1638 totals R872/F100/C20/D8. Cumulative T001–T1638
totals are R1505/F104/C20/D9: pre-oracle R379/F7/C3/D1 and post-oracle
R1126/F97/C17/D8. No package summary or PF evidence row is due until T1991;
use PF-168 or later when that final summary is written.

## Batch 28 preregistered estimate (before reading tests T1639–T1688)

Continue the frozen inventory-order sweep: select the first 50 test names not
already present in any ledger row, scanning from the start of the 1,991-function
inventory. T001–T1638 has 1,638 rows and 1,566 unique test names, leaving 425
unique inventory functions unledgered. The selected roster is the next 50:

- `TestGenericFindEquipFirstLookInObj`
- `TestGenericFindEquipFirstMagObjectMagic`
- `TestGetACIncludesWornApplyAC`
- `TestGetAllCharsInRoomUsesCPrependOrderAcrossBodyTypes`
- `TestGetArmorClassAppliesCSlotFactors`
- `TestGetCharVisRoomThenCharacterList`
- `TestGetCheckMoneyConsumesOnce`
- `TestGetDrinkAffectsColumnOrder`
- `TestGetFollowerActorsUsesCPrependOrder`
- `TestGetFromRoomStaleSnapshotCreditedOnce`
- `TestGetItemsInRoomSnapshotVsWriter`
- `TestGetWeaponInfoFleshAlter`
- `TestGhostShipOpensAndClosesMatchingRuntimeExits`
- `TestGhostShipSecondDockDepartureAnnouncedOnFirstDock`
- `TestGiveGoldConservationUnderConcurrency`
- `TestGo126JSONBytes`
- `TestGossipChannelHistoryKeepsSpeakerInvisibility`
- `TestGossipObserverMatchesMortalAndLeavesGameBytes`
- `TestGossipObserverOnlyPublicGossipAndIncludesNPC`
- `TestGossipObserverReceivesLuaBridge`
- `TestGroupedNPCKillerAwardsPlayerShares`
- `TestGroupExcludesOutOfRoomMembers`
- `TestGroupPenaltyRoundsAfterSubtraction`
- `TestGroupShareMessageOrderFollowsCFollowerList`
- `TestGroupSplitNoGoldLeaderSilent`
- `TestGuildGuardActorPortsDoorwayAndDeclines`
- `TestGuildGuardActorPortsFleeArm`
- `TestGuildGuardRoomActCarriesOwnBreak`
- `TestHandleDeathDrainsMobVictimExpBy37`
- `TestHandleDeathDyingSpecArmPuffSaysLine`
- `TestHandleDeathNeverDieCancelsDeathTail`
- `TestHandleDeathRunsRawKillSequence`
- `TestHandleMailAndNoteSlashAAbort`
- `TestHandleMobDeathClearsMemoryAndHunting`
- `TestHandlePlayerDeathPenaltiesAreSilent`
- `TestHandlePlayerDeathVampireDispatchesDust`
- `TestHcontrolBuildHouseRejects`
- `TestHcontrolBuildHouseWithoutLookup`
- `TestHcontrolBuildHouseWithWiredLookup`
- `TestHcontrolPayDeliversToActorAndWatcher`
- `TestHeldLightMovesWithPlayer`
- `TestHiddenAffectRestoredAtEveryTotalBoundary`
- `TestHitAndMoveRegenGearMatchC`
- `TestHitrollDamrollStorageIsSByte`
- `TestHoldLitLanternLightsDarkRoom`
- `TestHomeLoadRoomUsesExistingSaveRoomField`
- `TestHorseAppearsInRoomLook`
- `TestHorseFollowsPlayerAndStaysVisible`
- `TestHorseReappearsInLookAfterDismount`
- `TestHorseVisibleToInfravisionAfterDismount`

Before reading these tests, my estimate is **R43 / F5 / C1 / D1**.

## Batch 28 results (T1639–T1688)

The preregistered roster matched the next 50 unledgered inventory tests. The
actual marks are **R45 / F5 / C0 / D0**, against the estimate R43 / F5 / C1 /
D1. The five F rows are T1658, T1663, T1668, T1673, and T1678; all are J3
substring or partial-output checks strengthened to exact bytes or exact
silence. The batch adds five J3 marks and no other junk codes. Their introducing
commit attribution is unknown for T1658, T1663, T1668, and T1678 (Zach git
author, no model trailer); T1673 is Claude Opus 5.5 from its co-author trailer.

Each original weak assertion stayed green under a narrow output mutation, and
each fixed assertion failed. The five production mutations and old/fixed
results are recorded in `mutations.tsv`; the detached worktree was clean after
each restoration. The Puff mutation adds ` EXTRA.` while retaining the final
period, so its failure isolates the changed spoken line. No tests were deleted
or folded, so no live inventory references needed retargeting.

The ledger and mutation log were appended as raw tab-separated text. T1639–T1688
have 18 columns each; the ledger has 1,688 contiguous IDs, and the mutation log
has seven columns per row. The five tightened tests and the full `go test ./...`
pass. Package statement coverage is 64.8% before and after the batch. No
production code changed and no census was needed.

All eleven gates passed individually: `make fmt`; `GOEXPERIMENT=nojsonv2 go
build ./...`; `GOEXPERIMENT=nojsonv2 go vet ./...`; `GOEXPERIMENT=nojsonv2 go
test ./...`; `golangci-lint run ./...` (v2.15.0, 0 issues); `make
modernize-check`; `make deadcode-ratchet`; `make string-census`; `make
fidelity-depth`; `make expected-divergences-check`; and `make
function-index-check`. The string-census check reports its existing generated
reports as stale, the expected-divergence check reports existing unresolved
selectors, and the function-index check's intentional fake association is
rejected before the ratchet reports clean. `make fidelity-depth` reports 8,282
cases, with 7,968 proven/delegated, one blocked, 297 excluded, and 16
approved-divergent.

Through T1688, T639–T1688 totals R917/F105/C20/D8. Cumulative T001–T1688
totals are R1550/F109/C20/D9: pre-oracle R379/F7/C3/D1 and post-oracle
R1171/F102/C17/D8. The ledger now contains 1,616 unique inventory tests, with
375 unique tests remaining. The batch branch is stacked on Batch 27 commit
0132b1847; `origin/main` was fetched at ab14720f9, and both batch PRs target
`main`. Use PF-168 or later for the final package summary row.

## Batch 29 preregistered estimate (before reading tests T1689–T1738)

Continue the frozen inventory-order sweep, selecting the first 50 test names
not already present in any ledger row. T001–T1688 has 1,688 ledger rows and
1,616 unique test names, leaving 375 unique inventory functions unledgered.
The next roster is:

- `TestHouseBootDropsUnresolvableOwner`
- `TestHouseLoadExtractsUnrentable`
- `TestHouseLoadFloorsUnresolvableContainers`
- `TestHouseLoadLegacyRuntimeIDFilesFloorEverything`
- `TestHouseLoadNestsNothingIntoWrongObject`
- `TestHouseMudlogLevelDiscriminators`
- `TestHouseMudlogProducers`
- `TestHouseSaveLoadFloorsContainerContents`
- `TestHouseSaveLoadFloorsDepthTwo`
- `TestHuntItemsConsumesDrawBeforeHunterEligibility`
- `TestHuntItemsUsesWearOrderAndFirstAvailableCharacter`
- `TestHuntVictimAuctionAndShoutReachEveryListener`
- `TestHuntVictim_CommandArgumentsMatchC`
- `TestHuntVictimEvasionSayReachesTheRoom`
- `TestHuntVictimGossipIsGlobalAndRecorded`
- `TestHuntVictimKeepsPlayerIDWhenDescriptorMissing`
- `TestHuntVictimKeywordObjectShadowsDoor`
- `TestHuntVictimLeavesLinklessHuntingIDUnresolved`
- `TestHuntVictimMultiWordKeywordDoesNotOpen`
- `TestHuntVictimNonIntelligentRaceDoesNotOpen`
- `TestHuntVictimOpensClosedUnlockedDoor`
- `TestHuntVictimOpensMatchingContainer`
- `TestHuntVictimRefusesLockedDoor`
- `TestHuntVictimSameKeywordOpensOnlyTravelExit`
- `TestHuntVictimSayReachesTheRoom`
- `TestHuntVictimSecretCaseKeywordIsFindable`
- `TestHuntVictimTellUsesTheTellPath`
- `TestIdentifierValCost`
- `TestIdleForceRentMudlogLevelIsLVLGOD`
- `TestIdleForceRentMudlogProducer`
- `TestIdleForceRentMudlogTypeIsCMP`
- `TestIdleForceRentUsesRoomIndexThree`
- `TestIgnoreLegacyWorldStateIsNotFatalAndKeepsTheFile`
- `TestInspectLegacyWorldStateCountsTransientSections`
- `TestInspectLegacyWorldStateToleratesBadInput`
- `TestInstakillPreservesCrashFlag`
- `TestInstakillRunsRawKillSequence`
- `TestInvalidClassAntiClassBitsMatchCStructs`
- `TestInventoryAccessIsLockedAgainstMoves`
- `TestInventoryMovesSetCrashFlag`
- `TestIsRoomDarkSparesCityStreets`
- `TestJailGuardActorPortsMatchC`
- `TestJailPointUpdateOnlyDecrements`
- `TestJanitorAutonomousAssignedDispatch`
- `TestJunkCoinsOverflowMatchesC`
- `TestKenderStealCaughtRetaliationDrawsAndEnrols`
- `TestKenderStealCaughtRetaliationRefusedInPeacefulRoom`
- `TestKenderStealSetsFlag`
- `TestKeylessDoorMatchesCNoKeyGuard`
- `TestLegacySnapshotWithManySyntheticObjectsLogsOnce`

Before reading these tests, my estimate is **R43 / F5 / C1 / D1**.

## Batch 29 results (T1689–T1738)

The preregistered roster matched the next 50 unledgered inventory functions.
The estimate was R43/F5/C1/D1; the result is **R47/F2/C1/D0**. The two F
rows are T1733 and T1737, both J3 partial-output assertions now pinned to the
complete C response. Their introducing commits carry Zach's personal git
identity without a model trailer, so both authors are recorded as unknown.
The one C row is T1704: its same-room linkless hunting ID check duplicated
T1706's descriptorless unresolved-ID contract. I folded the same-room case
into `TestHuntVictimLeavesLinklessHuntingIDUnresolved`, retaining both
same-room and unreachable-room subtests, and removed the duplicate function.
Before removing it, I searched `git grep` in `docs/fidelity` and
`cmd/dp-oracle-diff`; the live hit in `docs/fidelity/depth/hunt-items.tsv`
now names the surviving test. The dated handoff remains historical evidence.

Both F tests went green under their old assertions after adding ` EXTRA` to
the C output, then red under the exact fixed assertions. T1737 has separate
refusal and click mutations; the click-only mutant also stays green under the
old body and fails the fixed assertion. All three mutations were restored
byte-for-byte in the detached mutation worktree. The 50 ledger rows and three
mutation rows were appended as raw tab-separated text, with no CSV quoting.
The ledger is contiguous through T1738, has 18 columns per row, and the
mutation file has seven columns per row. There are 325 unledgered functions
left in the frozen 1,991-function inventory.

Package statement coverage measured 64.7% on Batch 28 commit `0f8c26fc8` and
64.8% after this batch. No production code changed. The duplicate proof row
remains in the ledger as C and names T1706; its live fidelity inventory entry
was updated before the function was removed. No census was needed.

All eleven gates passed individually: `make fmt`; `GOEXPERIMENT=nojsonv2 go
build ./...`; `GOEXPERIMENT=nojsonv2 go vet ./...`; `GOEXPERIMENT=nojsonv2 go
test ./...`; `golangci-lint run ./...` (0 issues); `make modernize-check`;
`make deadcode-ratchet` (35 baselined, 0 new); `make string-census` (ratchet
clean with existing stale generated reports and four obsolete baseline
entries); `make fidelity-depth` (8,282 cases, 7,968 proven/delegated, one
blocked, 297 excluded, 16 approved-divergent); `make
expected-divergences-check` (clean with existing unresolved selectors); and
`make function-index-check` (the expected-red fake association was rejected,
then the ratchet reported clean).

The fetched `origin/main` is `ab14720f9`. Historical rows T360, T377, T387,
T391, T394, T395, T455, T465, T520, T535, T548, T606, T769, and T1403 match
`origin/main` byte-for-byte, including the mainline unquoting of T1403.
Through T1738, T639–T1738 totals R964/F107/C21/D8. Cumulative T001–T1738
is R1597/F111/C21/D9: pre-oracle R379/F7/C3/D1 and post-oracle
R1218/F104/C18/D8. This batch adds two J3 rows and one J4 row.

## Batch 30 preregistered estimate (before reading tests T1739–T1788)

Continue the frozen inventory-order sweep, selecting the first 50 test names
not already represented in the ledger. T001–T1738 has 1,738 rows, 1,666
unique audited names, and 325 unledgered inventory functions. The next roster
is:

- `TestLegacySwitchedCommandRefusalsUseBodyOutput`
- `TestLegacyUnmountClearsAffectsButRetainsMasterLinks`
- `TestLegacyUnmountClearsMountWithNilRider`
- `TestLiquidTablesMatchC`
- `TestListsHiddenBodyMatchesCHolylightClause`
- `TestLiveObjectsInCOrder`
- `TestLoadCompileTimeIgnoresThePinnedGameClock`
- `TestLoadCompileTimeUsesTheBootClock`
- `TestLoadFailureClearsMobRoomAndObjectFlags`
- `TestLocateObjectInvisibleOwnerReusesSaySpellBuffer`
- `TestLocateObjectResultsMatchCOutputAndNoLocateOrder`
- `TestLogHuntingStartGates`
- `TestLookAtCharacterNotifiesTheRoom`
- `TestLookAtCharTattooColorLevelsMatchC`
- `TestLookAtMobDescriptionIsNotCapitalized`
- `TestSpecEviltradeTellArgumentsMatchC`
- `TestSpecFighterHeadbuttRefusesImmortalDirectTarget`
- `TestSpecFlyExitUp_EntryGatesAndAudience`
- `TestSpecGuildPracticeListUsesDescriptorPagerCallback`
- `TestSpecHiscDelegatesAutonomousPulseToCleric`
- `TestSpecHornObjectReceiverAudienceAndGates`
- `TestSpecialActorLanesRegistered`
- `TestSpecialInterceptTableMatchesRecordedCommandSets`
- `TestSpecIdentifierEntryGatesAndValue`
- `TestSpecIdentifierGiveGatesAndPrice`
- `TestSpecIdentifierSuccessAudienceAndState`
- `TestSpecIdentifierTellArgumentsMatchC`
- `TestSpecIra_ExclaimAndHit`
- `TestSpecItohActPreservesCLineBreakBytesAndAudience`
- `TestSpecJail_CommandPathFallsThrough`
- `TestSpecJailGuard_Audience`
- `TestSpecJailGuard_EntryGates`
- `TestSpecJailPulseGates`
- `TestSpecJailPulseRelease`
- `TestSpecJailReleaseLeavesMountInCell`
- `TestSpecKeySellerTellArgumentsMatchC`
- `TestSpecLittleBoy_Exclaim`
- `TestSpecMallory_FightingBarks`
- `TestSpecMallory_Revenge`
- `TestSpecMallory_StandAndRevengeGate`
- `TestSpecMayorWakesAndFallsThroughToMobileActivity`
- `TestSpecMedusa_EntryGatesAndFightingDelegation`
- `TestSpecMedusa_PetrifyAudienceAndState`
- `TestSpecMedusa_SaveFallsThrough`
- `TestSpecMickey_ApproachBeats`
- `TestSpecMickey_ApproachPeaceful`
- `TestSpecMickey_EntryGates`
- `TestSpecMickey_FightingBarks`
- `TestSpecMickey_VictimChangeResets`
- `TestSpecMindflayer_EntryGatesAndFallthrough`

Before reading these tests, my estimate is **R44 / F4 / C1 / D1**.

## Batch 30 results (T1739–T1788)

The actual marks are R44 / F5 / C1 / D0 versus the preregistered estimate of
R44 / F4 / C1 / D1. There are five F rows, all J3, and the single C row is
J4. I found no D row. All 50 tests were introduced after the 2026-07-12
oracle cutoff. The batch authors are unknown for 49 tests and DeepSeek for
T1753; among the five weak tests, four are unknown-authored and one is
DeepSeek-authored.

I tightened five weak output checks: T1753 tattoo color, T1766 Ira's room
act, T1772 jail release transcript, T1777 Mallory's hunting log, and T1779
the mayor's wake act. For each, the old assertion stayed green with an extra
output mutant and the exact assertion turned red; all five mutants were
restored byte-for-byte in the detached mutation worktree. The C-marked
T1768 command-fallthrough test is folded into T1771's jail pulse-gate table,
which checks the same return, silence, room, gold, and movement contract. I
checked for live documentation/oracle-harness citations before folding it;
there were no hits. No D deletion or C divergence was needed.

Package statement coverage was 64.7% at preregistration and 64.8% after the
batch. No production code changed and no census was needed. The 50 ledger
rows and five mutation rows were appended as raw tab-separated text. The
ledger is contiguous through T1788 with 18 columns per row, and the mutation
file has seven columns per row. There are 203 functions left in the frozen
1,991-function inventory.

All eleven required gates passed individually on the final code: `make fmt`;
`GOEXPERIMENT=nojsonv2 go build ./...`; `GOEXPERIMENT=nojsonv2 go vet ./...`;
`GOEXPERIMENT=nojsonv2 go test ./...`; `golangci-lint run ./...` (0 issues);
`make modernize-check`; `make deadcode-ratchet` (35 baselined, 0 new);
`make string-census` (ratchet clean with the existing stale generated reports
and four obsolete baseline entries); `make fidelity-depth` (8,282 cases,
7,968 proven/delegated, one blocked, 297 excluded, 16 approved-divergent);
`make expected-divergences-check` (exit 0 with the existing unresolved
selectors); and `make function-index-check` (the expected-red fake association
was rejected, then the ratchet reported clean).

## Batch 31 preregistered estimate (before reading tests T1789–T1838)

Continue the frozen inventory-order sweep, selecting the first 50 test names
not already represented in the ledger. T001–T1788 has 1,788 rows, 1,716
unique audited names, and 275 unledgered inventory functions. The next roster
is:

- `TestSpecMindflayer_PsiblastUsesDirectDamage`
- `TestSpecMindflayer_RngArms`
- `TestSpecMindflayer_SoulLeechUsesDirectDamageAndHeals`
- `TestSpecMiniThiefUsesCChanceAndAmountBands`
- `TestSpecMortician_CommandSurfaceAndTellBranches`
- `TestSpecMortician_RetrievePredicateGlobalOrderAndState`
- `TestSpecMortician_SleepingMobStillHandlesCommand`
- `TestSpecMorticianTellArgumentsMatchC`
- `TestSpecNewbieZoneEntrance_Gates`
- `TestSpecNoGetHandStrikeSwings`
- `TestSpecOroQuartersRoom_GatesAudienceHPAndFallthrough`
- `TestSpecOroStudyRoom_GatesAudienceHPAndFallthrough`
- `TestSpecOutOfJailGuard_Audience`
- `TestSpecOutOfJailGuard_EntryGates`
- `TestSpecPaladinDirectDisarmResolvesTheBuiltNameArgument`
- `TestSpecPortalRoom_Teleport`
- `TestSpecPortalToTemple_ApostropheAlias`
- `TestSpecPortalToTemple_EntryAndExactArgumentGates`
- `TestSpecPortalToTemple_SayAudienceTeleportAndLandingLook`
- `TestSpecPrayForItems_FallsThroughOrdinarySocial`
- `TestSpecPrayForItems_ItemRewardBranch`
- `TestSpecPrayForItems_StaffNamesPatched`
- `TestSpecProstitute_BlindMobUsesHiddenAudience`
- `TestSpecProstitute_CanSeeUsesC_LightGate`
- `TestSpecProstitute_DirectTellsOnlyReachActor`
- `TestSpecProstitute_EntryGatesAndFallthrough`
- `TestSpecProstitute_InvisibleTargetUsesRoomAudience`
- `TestSpecProstituteTellArgumentsMatchC`
- `TestSpecQuanLoActorNamesTheActor`
- `TestSpecQuanLo_AwakeCommandGate`
- `TestSpecQuanLo_FleeAliasesUseGlobalMobGossip`
- `TestSpecQuanLo_LookAndExamineUseExactKeywords`
- `TestSpecRecruiterTellArgumentsMatchC`
- `TestSpecRemorterRejectsPlayerCommandsAtTheCorrectGates`
- `TestSpecRemorterSuccessResetsStateAndSeedsClassSkills`
- `TestSpecRemorterTellArgumentsMatchC`
- `TestSpecRoachEntryGates`
- `TestSpecRoachFoodFailureStillExtractsObject`
- `TestSpecRoachFoodSuccessGrowthAndExtraction`
- `TestSpecRoachIdleOutput`
- `TestSpecRoachSplitResetsOriginalAndSpawnsQuietRoach`
- `TestSpecRoachStarvationExtractsWithoutCorpse`
- `TestSpecRoachTeleportSubgateAndAudience`
- `TestSpecShopKeeperActorKenderActorShouts`
- `TestSpecShopKeeperActorPortsC`
- `TestSpecSnake_ActBytesMatchC`
- `TestSpecStableboyAutonomousEntry`
- `TestSpecStableboyMountedStableClearsMountAndShowsFollowerStop`
- `TestSpecStableboyTellArgumentsMatchC`
- `TestSpecStableboyUsesAnyUnmountedMountableFollower`

Before reading these tests, my estimate is **R44 / F4 / C1 / D1**. These are
all drawn from the post-oracle suffix, so the estimate is **0 pre-oracle and
R44 / F4 / C1 / D1 post-oracle**.

## Batch 31 results (T1789–T1838)

The actual marks are **R40 / F8 / C2 / D0**, versus the preregistered
estimate R44/F4/C1/D1. All 50 tests are post-oracle. Seven tests carry J3,
two carry J4, eight carry J7, one carries J9, and one carries J10. T1810
(`TestSpecPrayForItems_StaffNamesPatched`) pins the already-approved DP-1373
security divergence from PF-051: C granted immortality to reserved names,
while Go refuses. Forty-nine tests were introduced by commits with Zach's git
identity and no model trailer, so their author is unknown; T1810 has a Claude
Opus 5.5 trailer. All eight F tests have unknown authors.

I tightened the checks in T1792 mini-thief, T1804 portal room, T1807
portal-to-temple, T1817 Quan Lo actor, T1822 and T1823 remorter, T1825 roach
entry gates, and T1832 shop keeper Kender shout. Each old assertion stayed
green under its output mutation, the corrected exact assertion turned red,
and the source was restored byte-for-byte in the detached mutation worktree.
The two C tests were folded into T1791 and T1807. Before folding them, I ran
`git grep -n` over `docs/fidelity` and `cmd/dp-oracle-diff`; the live
`docs/fidelity/depth/spec-procs.tsv` proofs now name the surviving tests.
Remaining name hits are in dated evidence only. No D deletion was needed.

The 50 ledger rows were appended as raw tab-separated text, without a CSV
writer; all have 18 fields, the IDs are contiguous through T1838, and the
mutation file has 146 seven-field rows. The current `origin/main` is
`ab14720f9`. Historical rows T360, T377, T387, T391, T394, T395, T455,
T465, T520, T535, T548, T606, T769, and T1403 match it byte-for-byte,
including T1403's unquoted row. The frozen inventory has 153 unledgered
functions remaining.

Package statement coverage was 64.8% at the Batch 30 baseline and remains
64.8%. No production code changed, so no census was needed.

All eleven required gates passed individually: `make fmt`; `GOEXPERIMENT=nojsonv2
go build ./...`; `GOEXPERIMENT=nojsonv2 go vet ./...`; `GOEXPERIMENT=nojsonv2
go test ./...`; `golangci-lint run ./...` (0 issues); `make modernize-check`;
`make deadcode-ratchet` (35 baselined, 0 new); `make string-census` (ratchet
clean, with the existing stale generated reports and four obsolete baseline
entries); `make fidelity-depth` (8,282 cases, 7,968 proven/delegated, one
blocked, 297 excluded, 16 approved-divergent); `make
expected-divergences-check` (exit 0 with existing unresolved selectors); and
`make function-index-check` (the expected-red fake association was rejected,
then the ratchet reported clean).

Cumulatively, T001–T1838 has R1681/F124/C24/D9. The pre-oracle segment remains
R379/F7/C3/D1; post-oracle is R1302/F117/C21/D8. From T639 through T1838 the
counts are R1048/F120/C24/D8.
