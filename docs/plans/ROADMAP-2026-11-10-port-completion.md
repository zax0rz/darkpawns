# Roadmap: finishing the port, then telling people

2026-11-10, Claude, for Zach and every agent working the port. Draft until Zach
approves it. Nothing here changes a rule; `AGENTS.md` and
`docs/fidelity/RULEBOOK.md` still govern. Measured at `origin/main` `dbd002bd5`
(two days after `v0.1.9`).

Earlier pieces this builds on:
`docs/research/port-completion-survey-2026-11-05.md` (the inventory) and
`docs/briefs/PLAN-2026-11-07-done-measurement-tooling.md` (the tools).

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
| **A. Witness the uncovered sites** (L1) | Finish DP-1502's queue. Then classify what is still uncovered as C-unreachable, not player-facing, unit-proof only, or reachable-but-unwitnessed. Rerun coverage after each full census | DeepSeek in one-function slices (already running, batches 44 and 45 open) | the new scenario, plus a one-letter break that turns it red |
| **B. Fix the known divergences** (L2) | Work by class, not by ticket (R5c): `act()` lines with embedded line breaks (DP-1509, DP-1519), hand-built `act()` lines (DP-1494), target lookups that bypass C's resolver (DP-1457, DP-1518), invented behavior (DP-1485, DP-1492), then the single items | DeepSeek or Sol for one-ticket fixes; Luna for a whole class | targeted scenarios for the class |
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
