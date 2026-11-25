# Roadmap: finishing the port, then telling people

2026-11-10, Claude, for Zach and every agent working the port. Draft until Zach
approves it. Nothing here changes a rule; `AGENTS.md` and
`docs/fidelity/RULEBOOK.md` still govern. Measured at `origin/main` `dbd002bd5`
(two days after `v0.1.9`).

Earlier pieces this builds on:
`docs/research/port-completion-survey-2026-11-05.md` (the inventory) and
`docs/briefs/PLAN-2026-11-07-done-measurement-tooling.md` (the tools).

## Where we are (2026-11-25)

Measured at `origin/main` `f3d46e951`. Sections 1 to 8 are the plan as
written on 11-10 and settled on 11-11; this section is progress against it.
It replaces the 11-23 status.

**Phases 0 to 2 are done. Phase 3 is down to its measurements:** the three
stopping-rule test audits are complete, L1 through L5 have zero unruled open
items, and every survivor of the Round 4 population sweep is now killed,
reclassified with a C cite, or waiting on a Zach ruling. Round 4 itself
(look 4 of 4) is with Luna, pre-registered in
`docs/briefs/BRIEF-2026-11-25-luna-round-4.md`. Random play waits on the
play lab.

| List | Frozen 11-11 | Open now | What's left |
|---|---|---|---|
| L1. Uncovered C sites | 739 | **0** | confirmed by the 11-22 strict recount (PF-166) |
| L2. Known divergences | 58 | **0** | none open; DP-1610 is an approved divergence. Two new Go divergences found by the sweep were fixed on 11-25 (PF-179, PF-185) |
| L3. Blocked proof units | 62 | **0 unruled** | 1 blocked cell left (`spell.identify.matrix-staff-olc`), already covered by the 11-18 DP-1496 `time(0)` ruling; shop UB rows excluded under R1a |
| L4. Harness blind spots | 11 | **0** | none |
| L5. Duplicate and invented code | 5 | **0** | none; CI guards `function-index`. The sweep found four more player/switched-mobile duplicate pairs, now merged into one implementation each (PF-179, PF-180) |
| L6. Unverified research claims | 14 | 15 (14 plus PF-155) | not part of the stopping rule |

**L1 is at zero, confirmed.** Neon's strict recount on the 11-22 nightly dump
(`docs/research/l1-strict-recount-2026-11-22.md`, PF-166, #3021) classified
all 574 uncovered C segments: 307 excluded by a ruling, 226 unreachable with
a C cite, 33 witnessed in the multi-seed dumps, 8 unit-proven, and **0 open**.

**L2 and L3 have zero unruled items.** L2 closed on 11-22 (DP-1610 ruled as an
approved divergence; DP-1622 fixed in #2983; DP-1623 fixed in #2985, PF-162).
Across all 8,282 depth cases (`gen_fidelity_depth.py`) and nested surface
inventories, 7,968 are proven or delegated, 297 excluded, 16 approved
divergent, and **1** remains `blocked`: `cast.tsv:400`
(`spell.identify.matrix-staff-olc`, mirrored in `surface-inventory.tsv:57`
and `surface/spell-matrix.tsv:61`), which is already covered by Zach's
2026-11-18 DP-1496 ruling (`time(0)` age output stays unit-proven until the
post-v1.0.0 clock seam). The other 11 rows from `l3-blocked-2026-11-19.tsv`
were settled in #2866 (`58a8f9ef8`) and PF-159 (`l1-settlement-2026-11-21.tsv`).

**The census itself.** The nightly combined census runs at **02:30 on main**
(`~/bin/dp-nightly-census.sh`, results to Discord), with SQLite
`synchronous = NORMAL` in the census runner (#2996, #2997, PF-163; production
keeps SQLite's default). The last two nightlies are both `CLEAN` with no
rechecks and zero draw divergence: `2026-11-24/nightly-20261124` at
`ef2b14138` (7,068 pairs) and `2026-11-25/nightly-20261125` at `fe2838ce4`
(7,104 pairs, 2,285 deduplicated).

**The four numbers.**

| # | Number | Now | Measured on |
|---|---|---|---|
| 1 | Bytes | **0 FAIL** | nightly combined census `2026-11-25/nightly-20261125` at `fe2838ce4`: `CLEAN`, 7,104 pairs, no rechecks |
| 2 | Behavior | **0 draw divergence** | `2026-11-25/nightly-20261125`: 0 divergent across 2,493 full pairs (seed 1) and 6,896 claims pairs (22 seeds); `draw-results.tsv` present |
| 3 | Reach | **88.3%** of segments (4,328 of 4,902); 89.0% with the other seeds; 66.4% of sites (3,622 of 5,454) | `make census-coverage` on `2026-11-22/nightly-20261122` (PF-166). Up from 73.5% and 54.3% on 11-10. Not remeasured since; the sweep repairs added scenarios, so the next recount should be higher |
| 4 | Sensitivity | **64.3%** (corrected lower bound 51.7%) | mutation loop Round 3, 63 of 98 killed (PF-149). The bar is a corrected lower bound of 80% (90 of 100 at z = 2.50, 4 looks; 3 used). Round 4 is the last look |

**The sweep is finished.** Luna's whole-pool sweep (PF-169) left 308
survivors. Every one now has an outcome, recomputed from the triage, every
correction and the repair ledger
(`docs/research/mutation-loop/sweep/RESULT.md`):

| Bucket | Rows | Outcome |
|---|---:|---|
| A. no witness | 201 | **all 201 killed** by scenario-backed proofs (`REPAIR-LEDGER.tsv`, PF-171 to PF-185); the last, row 918, came with the paper-writing fix |
| B. masked | 0 | none found |
| C. Go-only branch | 37 | 33 removed or silenced (#3070, #3079); two approved divergences kept (DP-1373, the alias cap); rows 647-648 turned out to be a stand-in on a C path and were ported instead (#3074). See `GO-ONLY-REMOVALS.tsv` |
| D. C-unreachable | 57 | excluded, each with a C cite |
| E. not player-facing | 3 | collector false positives |
| F. ruling or harness | 10 | five R1a rows (144, 527-529, 530) wait on Zach; two need a no-database harness state (834, 838); rows 2, 4 and 908 have behavior or fixture limits |

The sweep isn't a measurement and uses no look. Its kill rate says nothing
about Round 4's result, which is measured fresh on whatever `origin/main`
holds at the draw.

**What the sweep found besides missing tests.** Mutation survivors led to
real Go fixes, not just scenarios:

- the switched-mobile `carve` was a hand-written stand-in with the wrong
  food items, no weapon check and invented messages (#3074, PF-179);
- `point`, `compare` and `sharpen` had separate player and switched-mobile
  copies; all four commands now share one implementation each, and the
  player paths now go through `act()` (#3074, #3075, PF-179, PF-180);
- Go set `PLR_WRITING` for a player writing on paper, which C never does
  (`src/act.comm.c:1098-1101`), so paper writers missed room actions,
  channels and tells (#3078, PF-185).

**Test audit: stopping-rule packages complete.**

| Package | Audited | Note |
|---|---|---|
| `pkg/command` | 173 of 173 | done |
| `pkg/session` | 1,537 of 1,537 | done (PF-165) |
| `pkg/game` | 1,991 of 1,991 | done (#3035, PF-168): R 1,788 / F 155 / C 37 / D 11 |

The blind cross-audit of 120 tests (Carbon, PF-170) found the session/game gap
in weak-test rates was mostly auditor strictness, and its findings were fixed
in #3045 and #3046. Section 9 records all of the test work.

**Random play** waits on DeepSeek's play lab. Its logging is being designed to
double as decision-model training data.

**What's left, in order:**

1. **Round 4** (Luna, in progress): draw with seed `20261123`, classify
   before any census, measure, then repair its misses. The terms are fixed in
   the brief. If the corrected lower bound is below 80%, the completion
   report publishes the measured rate.
2. **Zach's rulings** in `docs/research/mutation-loop/DECISIONS.md`: the R1a
   rows (148 outside the survivor set; 144, 527-529 and 530 inside it), and
   whether to delete the 111 level- and position-only registration tests
   (`docs/research/test-audit/cross-audit-2026-11-24/registration-class.tsv`).
   The row 918 transcript is evidence for the R1a rows: the pinned C build
   appended correctly through the same overlapping `sprintf` pattern.
3. **Optional audit calibration:** settle the three `pkg/spells` ledgers into
   one. It doesn't block the section 6 stopping rule.
4. **Phase 3 finish:** a fresh reach recount, the draw-site reach report (or
   an explicit deferral note), 24 hours of clean random play, and the release
   census with `--combined --release --draw-parity` on the tagged `v1.0.0`
   commit.

## 1. What "done" means

Zach's definition (2026-11-06): the port is done **"when anything a player sees
is identical to the original and the game behaves identically (RNG, etc) and we
can prove it."**

Nobody can prove that no bug exists. Every new way of looking at this port has
found more (the oracle, depth runs, playtests, returning players, random
mutations), and the next one will too. So "done" can't be "the bug queue is
empty": the queue refills from wherever we look next.

What can be finished is a **bounded claim**:

> Every item on a fixed set of finite lists is settled, four numbers are
> published with their limits, and a stopping rule that was written down in
> advance has been met.

"Settled" means one of four things, and nothing else: **proven** (a green that
can fail, R5h), **fixed**, **C-unreachable** (with the C cite), or an
**approved exception** Zach signed. That turns the finish into burn-downs with
a bottom. It is also the more interesting thing to publish, because it says
exactly what was checked and what wasn't.

## 2. The scoreboard today

| # | Number | Reading | Measured on |
|---|---|---|---|
| 1 | **Bytes**: FAIL rows in a full combined census | 0. 6,411 runs over 16 seeds, CLEAN_AFTER_RECHECK | `2026-11-08/release-v0.1.9` |
| 2 | **Behavior**: runs where the two engines drew different random numbers | 0 of 1,997 full and 6,335 claimed runs over 16 seeds, with the draw log present. The `v0.1.9` release census wrote no draw log because it ran without `--draw-parity` | `2026-11-10/go127-batch5-reds-combined` (Go 1.27, `3d14ccab6`) |
| 3 | **Reach**: C output sites that C actually printed during a census | 2,963 of 5,454 sites (54.3%); 3,601 of 4,902 segments (73.5%). Up from 50.3% and 68.3% on 11-06. About 25 burn-down batches have merged since and aren't counted | `make census-coverage` on the release dump, run today |
| 4 | **Sensitivity**: random one-letter output breaks the census catches | 23/40 (game, combat, command), 24/40 (session), 20/40 (spells). 13 of the 16 catchable misses were repaired afterward; no fresh sample since | PF-102, PF-117, PF-118 |

Supporting counts:

- **Depth manifests:** 7,264 cases; 7,121 proven or delegated, 37 blocked, 94
  excluded, 12 approved divergences (`scripts/gen_fidelity_depth.py`, today).
  The surface inventory sits outside that headline and still has blocked rows,
  including the lifecycle units (death, rent, save and load, regeneration,
  weather, descriptors).
- **Open issues:** 127 on GitHub, not all port work. About 22 of the 64 filed
  in the last week (DP-1457 to DP-1520) are still open divergences.
- **Test audits:** 2 of 33 packages (`pkg/combat`, `pkg/spells`).
- **Not yet built:** a list of C's random-draw sites, and a classification of
  the uncovered output sites.

Numbers 1 and 2 are at zero. That is real, and it is also where the port stood
on 2026-11-06 when the random sample showed the census missing four breaks in
ten. Numbers 3 and 4 are the distance between "census-clean" and "done".

## 3. The lists that have to reach zero

Each list is finite. An item leaves only by being settled (section 1).

| List | Source of truth | Size now |
|---|---|---|
| **L1. Uncovered player-facing C sites** | `coverage.tsv` from the latest full census, minus classified sites | 817 in the 11-07 queue; in burn-down under DP-1502 |
| **L2. Known divergences** | Linear, port-fidelity issues | roughly 50; recount in Phase 0 |
| **L3. Blocked proof units** | depth manifests, `surface-inventory.tsv`, shop branches | 37 depth cases plus the surface and shop rows |
| **L4. Harness blind spots** | Linear, harness issues | about 10 (section 5, stream D) |
| **L5. Duplicate and invented Go code** | the function inventory (not built yet) | unknown; first probe found 3 wrong pairs in 15 |
| **L6. Unverified research claims** | `EVIDENCE_LEDGER.tsv`, `needs-verification` rows | 13 on 11-05; recount in Phase 0 |

**The rule that keeps this honest:** a newly found bug is added to its list
with a note on how it was found. It doesn't reopen the definition. If a whole
new *kind* of looking turns up bugs (as random mutation did), that is a
proposed new list, and Zach decides whether it joins this release's bar or the
next one's.

## 4. Phases

### Phase 0: freeze the bar (Zach, one sitting)

Zach picks targets for the stopping rule in section 6 and signs the list
definitions above. Claude recounts L2 and L6 and writes the frozen lists to
`docs/research/` with a date. From here, every PR names the list item it
removes.

### Phase 1: burn the lists down (several agents in parallel)

Streams A to D in section 5. This is where most of the remaining work is, and
most of it is already in motion.

### Phase 2: one port of each function

Build the C→Go function index, remove or merge duplicate ports, delete
invented and dead code (L5). Add the ratchet: CI fails when two Go functions
cite the same C function. This comes after Phase 1's fixes land so the
inventory runs once, on settled code. The Go 1.27 pass is done
(#2452 and #2453 merged; CI now enforces the modernizers).

### Phase 3: the final measurement (pre-registered)

Written down before it runs, so the result can't be argued with afterward:

1. Build the draw-site inventory and report draw reach beside output reach.
2. Draw a **fresh** random mutation sample across every package that prints.
   The old samples are spent: their misses were repaired, so re-running them
   only measures the repairs.
3. Run random play through both engines (the play lab) and record new
   divergences per hour.
4. Run `--combined --release --draw-parity` (the flag is not on by default) on the exact commit to be
   tagged, and confirm `draw-results.tsv` exists before anyone reads the
   verdict.

### Phase 4: declare

Tag the release and publish a **completion report**: the four numbers with
their intervals, the stopping rule and how it was met, and the full list of
approved exceptions and unreachable sites. Freeze the evidence (run archives,
manifests, ledger) at that tag. After this, a new divergence is a maintenance
bug against a finished port, which is a normal thing for software to have.

## 5. Work streams for Phase 1

| Stream | What | Good fit | Smallest census that can fail |
|---|---|---|---|
| **A. Witness the uncovered sites** (L1) | Finish DP-1502's queue. Then classify what is still uncovered as C-unreachable, not player-facing, unit-proof only, or reachable-but-unwitnessed. Rerun coverage after each full census | DeepSeek in one-function slices (done 11-16); strict Phase C is Luna's | the new scenario, plus a one-letter break that turns it red |
| **B. Fix the known divergences** (L2) | Work by class, not by ticket (R5c): `act()` lines with embedded line breaks (DP-1509, DP-1519), hand-built `act()` lines (DP-1494), target lookups that bypass C's resolver (DP-1457, DP-1518), invented behavior (DP-1485, DP-1492), then the single items | Sol for one-ticket fixes; Luna for a whole class | targeted scenarios for the class |
| **C. Unblock the blocked units** (L3) | The lifecycle surfaces first: they are where returning players found most bugs. Shop branches that depend on C undefined behavior need a ruling from Zach (R1a), not a test | Luna (long runs); stop-tier items go to Zach | per unit |
| **D. Repair the harness** (L4) | Object-index fixtures (DP-1517), the one-player Go store (DP-1477), lost output on long clock pulses (DP-1439, DP-1515), mid-fight draw order with spec procs (DP-1507), the wall-clock read (DP-1496), trailing space and line endings hidden by the normalizer (DP-1471) | Sol (stop tier, so Zach merges) | a control that must go red, per fix |

Order within Phase 1: **D first where it gates the others** (a blind harness
makes every later green weaker), A and B in parallel, C as agents free up.

Standing rules for every agent on this roadmap, all from `AGENTS.md`:

- Name the list item in the PR title, with its Linear ID.
- A witness counts only with its revert proof (R5h) and no refused setup (R5j).
- Cite C from `src/`, from a line read in that session (R5g).
- End the PR with "Decisions for Zach". Stop and report; Claude reviews in
  Zach's session. Don't run a final full census; Claude batches it.

## 6. The stopping rule (Zach sets the numbers)

Zach set these targets on 2026-11-11 (Phase 0). The frozen lists and the
full decision record are in
[`docs/research/frozen-lists-2026-11-11/`](../research/frozen-lists-2026-11-11/README.md).

| Condition | Target |
|---|---|
| Bytes: FAIL rows on the tagged commit | 0 |
| Behavior: draw-divergent runs on the tagged commit | 0, with the draw log present |
| L1: player-facing sites that are reachable and unwitnessed | 0 |
| L2: open known divergences without an approval | 0 |
| L3: blocked units without a ruling | 0 |
| Fresh mutation sample, all printing packages | lower end of the 95% interval at or above 80%, and every catchable miss repaired |
| Random play | no new divergence in the last N hours of play (pick N; 24 is a reasonable first bar) |
| Test audit | `pkg/game`, `pkg/session` and `pkg/command` audited; the rest listed as not audited in the report |

Decided on 2026-11-11:

- **Unit tests as proof:** an oracle witness is the default. A unit test
  counts only where the oracle cannot reach the site (mid-combat states,
  DP-1202, and long-shot random rolls), with a C cite and a revert proof.
  The completion report carries unit-proven sites in their own column.
- **Long-shot random rows** are settled by one standing exception for the
  class; each row names its odds and its C line.
- **New kinds of findings** join the next release's bar. A FAIL or draw
  divergence in the final release census always blocks.
- **The tag** is `v1.0.0`.

## 7. After the tag: sharing it

The interesting story isn't that AI ported a MUD. It's that the original C ran
beside the port as a referee, so this project could **measure how often the AI,
the tests and even the referee were wrong**, and most projects can't. Lead with
the failures and the numbers.

**Tests are evidence, not a chapter** (Zach, 2026-11-22). Nobody wants to read
about testing, so no post or paper section is about the tests. Each test
finding goes where it proves a claim: fake and template tests are part of the
slop story (PF-157, PF-165, the deleted unreachability proofs), and mutation
kill rates and census results are part of the fidelity claim. Section 9 is the
internal record, not prose for readers.

Material that already exists in the ledger:

- A clean census and a 99.9% manifest, while a random sample showed four in
  ten one-letter breaks going unnoticed (PF-102, PF-117).
- "Proven by assignment" for nearly every output site against 50% measured
  reach (PF-111).
- The reference binary was itself wrong for weeks: an `-O2` build changed bytes
  through undefined behavior, and the port had copied them.
- The port "fixed" a 1990s macro bug and so stopped being faithful (PF-112).
- Two ports of one C function, written six months apart (PF-114).
- Seven models from seven labs, each optimistic in its own way (PF-110).
- Returning players from the 1990s finding what the tests didn't.

**Sequence:**

1. **Blog post on darkpawns.org**: "what does done mean?", built on the
   completion report. Write it with the `dp-blog` skill.
2. **Hacker News**, as a plain link to that post, play link at the bottom.
   Then r/golang with a more technical cut (the differential harness, seeded
   RNG and draw parity, in Go), then r/programming. r/mud only if it fits.
3. **The paper** (below).

**Before posting:** check the server can take a spike; verify any token total
from attributable sources (PF-115 is an upper bound, not "Dark Pawns used X");
check player privacy (handles only).

### The paper, made tractable

The study design already exists and Zach approved it:
`docs/research/fidelity-study/STUDY.md`. Four research questions extend Pan et
al., "Lost in Translation" (ICSE 2024), and the screening, slop track and
timeline are done or in progress. What's left is mostly assembly.

- **Genre:** an *experience report* or *industry-track* paper. That genre
  wants what this project has (a real system, honest failures, data) and
  doesn't require a new algorithm. The dataset can be a second, shorter
  artifact paper.
- **Where:** software-engineering venues fit the port paper better than a
  games venue (maintenance and migration conferences, industry tracks, and
  LLM-for-code workshops). The agents-playing work is the games paper, later.
  Check each venue's current call and deadline before aiming at one; none are
  verified here.
- **First concrete steps, in order:**
  1. Freeze the evidence at the completion tag and archive it somewhere that
     issues a DOI, so the paper cites a fixed snapshot.
  2. Turn the completion report into the results section; turn `STUDY.md`
     into the method section; write the limits plainly (one project, one
     human, tasks not randomized, only caught mistakes are counted).
  3. Post a preprint. First-time arXiv authors in computer science generally
     need an endorsement, which is one reason to do step 4 early.
  4. Find one academic reader or co-author (the planned outreach to nearby CS
     faculty). A co-author who knows the venue is the single biggest lever for
     someone outside academia, and the dataset is a real thing to offer them.

A blog post that lands and a public dataset are a legitimate outcome on their
own. The paper is worth doing but shouldn't hold the post hostage.

## 8. Decisions for Zach (settled 2026-11-11)

All five were settled in Phase 0. Section 1's bounded claim is the definition
of done; section 6 holds the targets and the other rulings; this file is on
main and `AGENTS.md` links to it; the tag is `v1.0.0`. The record is in
[`docs/research/frozen-lists-2026-11-11/README.md`](../research/frozen-lists-2026-11-11/README.md).

## 9. The test work

The finish line in section 6 includes auditing `pkg/game`, `pkg/session`
and `pkg/command`: 3,701 Go tests. With the earlier `pkg/combat` and
`pkg/spells` audits, the total is 4,100. This section records that work and
the checking around it, since no other single document does. It is the
internal record. Writing for readers weaves this material into other
stories (section 7).

### Four layers, four questions

Each layer answers a different question. None stands in for another.

| Layer | Question | Where it lives | State on 11-23 |
|---|---|---|---|
| **Oracle census** | Given the same input, do Go and C send the same bytes and draw the same random numbers? | `scripts/census.sh`; `docs/fidelity/oracle-differential-testing.md` | nightly at 02:30; 11-22 was `CLEAN` and 11-23 was `CLEAN_AFTER_RECHECK`, both with 0 draw divergences |
| **Depth proofs** | Does every C output site have a scenario or unit test that fails when the site breaks (R5h)? | `docs/fidelity/DEPTH_TESTING.md`; `docs/fidelity/depth/` | L1 confirmed 0 on 11-22 (PF-166); L3 has 0 unruled blocked rows (1 `blocked` cell left, covered by DP-1496) |
| **Mutation sampling** | If the port's output changes, does the census notice? | `docs/research/mutation-loop/`; earlier one-letter samples PF-102, PF-117, PF-118 | Rounds 1 to 3: 67.0%, 64.9%, 64.3%, against a bar of 80% (corrected lower bound); Round 4 population sweep complete (688 killed, 308 survived, 1 R1a unknown; PF-169) |
| **Test audit** | Does each Go test check what its name claims, on the code players run? | `docs/research/test-audit/` | all three stopping-rule packages complete (below) |

A green census says the scenarios agree. Mutation sampling says how often a
real break would show. The test audit says whether the Go tests themselves
can be trusted. Each layer has caught something the others missed.

### The test audit

The method is `docs/research/test-audit/pilot-combat/BRIEF.md`. Each test
gets one mark: **R** keep, **F** fix the check, **C** merge into a stronger
test, **D** delete. Junk codes J1 to J10 say why (J10 is a test that pins a
Go/C divergence). Every F, C or D carries a mutation proof: the old test stays
green under a break the fixed test catches, or the surviving test catches
what a removed one did. All-R batches get three random mutation spot checks.
The auditor writes predictions before reading any test.

| Package | Tests | Auditors | R | F | C | D | Record |
|---|---:|---|---:|---:|---:|---:|---|
| `pkg/combat` | 208 | MiMo, DeepSeek and Sol, blind; Claude settled disagreements | 172 | 17 | 15 | 4 | `pilot-combat/SETTLED.md`, PF-103 |
| `pkg/spells` | 191 | MiMo, DeepSeek and Sol, blind | | | | | three ledgers, not yet settled into one; PF-107, PF-108 |
| `pkg/command` | 173 | Kimi | 161 | 9 | 0 | 3 | `command/notes.md` |
| `pkg/session` | 1,537 | Kimi (T001 to T1045), then Neon | 1,529 | 4 | 4 | 0 | `session/notes.md`, PF-165 |
| `pkg/game` | 1,991 | GLM (pre-oracle), Luna (post-oracle) | 1,788 | 155 | 37 | 11 | `game/notes.md`, PF-168 (2,063 ledger rows; 72 functions audited twice) |

The other 49 directories with Go tests aren't audited. The completion
report lists them as not audited (section 6).

**What the audit found besides weak tests.** Auditing tests turned up live
divergences in the code under them:
- a second, wrong port of `counter_procs` in `pkg/combat`, pinned by its own
  test (PF-103);
- five live C divergences in spells that only the frontier auditor reported,
  each confirmed from source (PF-108);
- in `pkg/game`: the conjured-actor special (#2990), and, in review, the
  clan info terminator, clan treasury arithmetic and `ungroup` (#3018);
  #3018 also led to the class audit DP-1625 (#3024).

**Where the audit itself went wrong.** Each of these was caught in review,
and each changed how the next batch was run:
- GLM's first post-oracle `pkg/game` pass was a template: 900 identical rows
  (PF-157). It was closed and redone.
- A logged mutation proof didn't reproduce when rerun. The test lacked the
  exact-boundary case (#2987, PF-164).
- Neon deleted 11 tests as fakes because they never call Go. They were the
  depth manifests' proofs that C branches can't run, which is how those L1
  sites count as settled. CI's `fidelity-depth` check caught it, and they
  were restored before merge (#3003, #3004).
- Removed tests stayed named as proof in two live inventories that no gate
  reads (`surface/mob-command-calls.tsv`, `surface-inventory.tsv`), found in
  review of #3014 and #3017.
- A CSV writer re-quoted old ledger rows, undoing an earlier cleanup (#3017).
- `pkg/session`'s post-oracle tests came out 99.9% R from two auditors, while
  `pkg/game`'s came out 88.4% R from a stricter one. So the pre-oracle versus
  post-oracle comparison mixes auditor and date: `pkg/command` 13.5% to 5.1%
  weak, `pkg/session` 4.1% to 0.1%, `pkg/game` 3.3% to 11.6% (PF-168). PF-165
  is partially verified until a blind cross-audit sample settles it.

**Rules the audit now follows,** from those incidents:
- The reviewer reproduces mutation proofs, and doesn't take them from the
  log.
- Before marking anything C or D, grep `docs/fidelity` and
  `cmd/dp-oracle-diff` for the test's name, and update every live hit.
- Ledgers are raw tab-separated text, never written through a CSV writer.
- Every batch PR targets `main`. A stacked PR merges into its dead base
  branch, not main (#2994).
- A long all-R run from one auditor gets a blind sample from another.

**What's left (optional calibration):** settling the three `pkg/spells`
ledgers into one. The blind cross-audit sample on `pkg/session` and
`pkg/game` ran on 11-24 (PF-170, resolving PF-165).
The combat and spells audits already used the design that sample needs:
several blind auditors, then a settlement by mutation.
