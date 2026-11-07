# What we already have for measuring "done"

2026-11-07, Claude, for Zach. A note to come back to when we start the final push toward port completion. Nothing here changes a rule.

## The definition we're measuring against

Zach, 2026-11-06: the port is done "when anything a player sees is identical to the original and the game behaves identically (RNG, etc) and we can prove it."

That gives four numbers. Each needs a finite list behind it, so "done" is a burn-down with a bottom, not a bug queue that refills from wherever we look.

| # | Number | What it counts | Status |
|---|---|---|---|
| 1 | **Bytes** | FAIL rows in a full combined census | Measured. 0 FAIL on 2026-11-06 (`mimo-dp1474-full`, 1,871 scenarios); 16 pinned scenarios, 12 of them spell items briefed 2026-11-07 |
| 2 | **Behavior** | `draw_divergent` rows in a full census with `--draw-parity` | First full measurement: `2026-11-07/claude-rehearsal-1107` |
| 3 | **Reach** | the share of C output sites (and later, draw sites) that C actually printed during a census | **Measurable today, see below.** No draw-site list yet |
| 4 | **Sensitivity** | the share of planted random mutations the census catches | 23 of 40 at PF-102 (2026-11-05), output functions only |

## Tools and inventories that already exist

### Output-site inventory, by assignment
- **File:** `docs/fidelity/depth/surface-inventory.tsv`, the `activity-call-sites` rows, with per-file detail in `docs/fidelity/depth/surface/<file>.tsv`.
- **What it is:** source-order audits from October 2026, mostly Luna's long surface run. They list every `act()`/`send_to_char()`/`stc()` call in each C file and assign it to a proof.
- **Status:** almost every family says `proven-already`. About 15 sites in seven files are still blocked: boards, mail, clan, interpreter, objsave, new_cmds and act.wizard.
- **The catch:** "proven" there means *assigned* to a scenario or a unit test. It doesn't mean measured. Unit proofs count, for example dream.c's 29 sites are credited to a `World.PointUpdate` unit matrix plus one oracle case. That was before DP-1456 found 269 rows passing on refused setups.

### String census
- **Tool:** `cmd/dp-string-census`, run with `make string-census-update`. Output is in `docs/fidelity/strings/`.
- **What it does:** cross-references every literal at an output sink in C (5,547 candidates) against Go (1,850). It reports Go-only strings, which are invention candidates under R4, and C strings Go never emits.
- **Status:** last resynced 2026-10-30.

### Census coverage: measured reach
- **Tool:** `cmd/dp-census-coverage`, run with `make census-coverage CENSUS_DUMP=<run>/census-dump`.
- **What it does:** reads a census dump and reports which C output strings **C actually printed** during the scenarios. Only C's blocks count; Go output never does.
- **Status:** the tool existed but nobody had run it. Every census run directory already keeps the `census-dump/` it reads, so running it costs one command, with no extra census.
- **First run, 2026-11-07,** on `2026-11-06/mimo-dp1474-full`:
  - **C sites:** 2,744 of 5,454 covered (50.3%).
  - **C segments:** 3,350 of 4,902 covered (68.3%).
  - **Unverifiable:** 1,336 sites whose fixed text is shorter than 8 characters, excluded from the percentage.
  - **Least covered:**
    - `dream.c` 0 of 35 segments
    - `db.c` 1 of 21
    - `scripts.c` 1 of 20
    - `objsave.c` 10 of 54
    - `mobact.c` 2 of 10
    - `comm.c` 12 of 36
    - `spec_procs.c` 84 of 208
  - **Output kept** in `~/Archives/darkpawns/notes/coverage-2026-11-07/`: `coverage.tsv` holds every C site with the scenarios that reached it.

**How to read the gap.** The 50% isn't comparable to the inventory's "proven" yet:
- Its denominator includes strings players never see (syslog-style messages, `constants.c` tables), sites that are dead in C, and sites proven only by unit tests.
- So the uncovered half is a to-do list, not a failure count. Each uncovered site gets one class: **C-unreachable**, **not player-facing**, **unit-proof only**, or **reachable but never witnessed**.
- Only the last class is a hole in the proof. The unit-only class is a weaker proof than an oracle witness, and Zach should decide whether it counts.

### Command-table reachability (Daeron's weekly cron)
- **Generator:** `scripts/gen_reachability.py`, plus `scripts/reachability_weekly.py` for the TSV, the delta and the JSONL. Run with `make reachability` or `make reachability-weekly`.
- **History:** `docs/research/metrics/reachability-history.jsonl`.
- **What it measures:** whether each C **command** is registered in Go (registered, missing, unwired, specproc). That's command presence, not scenario reach.
- **Status:** the history has 3 entries: 2026-06-18 (manual), 2026-07-22 and 2026-08-26. The last showed 286 registered and 27 missing.
- **Why it stopped being useful:**
  - The cron ran from Daeron's separate clone, which drifted from origin.
  - Its Discord embed dumped raw delta JSON, which is the "illegible garbage" Zach remembers.
  - Command presence has been saturated since August, so it stopped telling us anything.
- **Wiring:** OpenClaw cron `add8cadf`, Mondays 09:00 ET. The wrapper lives in `~/.openclaw/workspace/scripts/`.

### Other pieces
- **Mutation sampling:** PF-102's pre-registered sample of 40 output mutations (`docs/research/field-notes/2026-11-05-a-random-sample-finds-the-census-blind-side.md`).
- **Depth manifests:** `docs/fidelity/depth/*.tsv`, about 7,100 case rows. On 2026-11-07, 106 were `blocked`: 79 spell-matrix and 23 surface-inventory, including six lifecycle units: death, rent, save/load, regen, weather/time and descriptor.
- **Random play:** the DeepSeek play lab (`~/dp-play-lab`, a separate repo). It can serve as a discovery-rate meter: new divergences per hour of random play through both engines.

## What doesn't exist yet

1. **A draw-site inventory.** Nothing lists C's `number()`/`dice()` call sites or measures which ones a census exercises. Number 2 needs it.
2. **A classification of the uncovered output sites** into the four classes above.
3. **Per-site reach for draws:** the draw log records positions, but nothing maps them back to C call sites.
4. **A mutation sample beyond output functions.** The kickoff named `pkg/session` and `pkg/spells`.
5. **A replacement for the weekly cron** that reports the four numbers, readably, from origin/main.

## The final push, sketched

When Zach calls it:
1. **Classify the uncovered sites** from `coverage.tsv`. This is mostly mechanical and makes a good brief for a long-running agent.
2. **Build the draw-site inventory** and extend `dp-census-coverage` to report draw reach.
3. **Witness the "reachable but never witnessed" sites** and the blocked lifecycle units that aren't stop tier (weather, time, regen) with new scenarios.
4. **Rerun the mutation sample**, wider, pre-registered.
5. **Random play** until the discovery rate stays flat.
6. **Publish the four numbers with error bars,** plus the list of what isn't covered and why. That's the "what is done?" post.
