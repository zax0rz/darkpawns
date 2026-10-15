# 2026-10-14: The last bugs are about when, not what

## Scope

- **Tracks:** legacy-port fidelity (residual-defect taxonomy, why the tail resists)
- **Window:** 2026-10-13 to 2026-10-14
- **Primary artifacts:**
  - PR #1982 (DP-1438) and its handoff `docs/fidelity/depth/handoff/2026-10-14-dp1438-combat-entry.md`
  - PR #1974 and the handoff `docs/fidelity/depth/handoff/2026-10-14-dp1371-backlog-batch.md` (blocked row `mobact.peaceful-first-hit-enrolment`)
  - Linear DP-1364 (PR #1969), DP-1433, DP-1440
  - `src/fight.c:1160-1248, 1318-1450, 1693-1697`; `src/graph.c:282-400`
- **Status:** observed. These are five worked cases. This is not a measured distribution.

## What was observed

Zach, after a day of agent PRs on the last blocked cases: "I see why these final cases have been blocked for so long. They are NOT easy." The defects found and fixed in these two days share a shape. **None of them is a wrong string.** Each prints C's exact text, but at the wrong time, in the wrong order, to the wrong audience, or not at all:

| Case | What Go got wrong | Kind |
|---|---|---|
| DP-1438 / peaceful first hit | Go enrolled both fighters **before** the first swing. C enrols inside `damage()`, **after** the refusal checks (`fight.c:1404-1408`). Refused swings skipped `hit()`'s roll, and a mob's first swing in a peaceful room landed. | **when** something happens |
| DP-1364 | Mobs were walked in sorted order. C walks the prepend-ordered `character_list`, so the same draws were consumed in a different order. | **order** |
| DP-1440 | Kill brags were computed and their RNG drawn, then handed to a callback nothing assigns. Hunter says went to the mob itself, and hunter gossip reached one room. | **whether**, **who hears** |
| DP-1433 | The cited char-table builder is never reached in production. The live one was already right. | **which path runs** |
| PF-067 (`loud_mobs`) | One ambient draw was missing during setup and only showed up later, in a combat roll. | **whether** (ambient) |

## Why this tail resists

1. **The fix is cross-cutting.** A missing string is fixed in one place. A wrong *point in the sequence* is shared by every caller. DP-1438 had to convert five C `hit()` callers in one engine change, and still left the mobile call sites for a later train. An audit has to decide caller by caller, from `src/`, whether each is a real `hit()` or a bare `set_fighting`.
2. **The byte census can't see most of it.** The text is right, so transcripts match until the drift reaches an observable roll. That roll can be in a different subsystem (PF-067), or it never comes at all (a coincidence green, PF-068). These cases were found by the draw-parity census, by populated-world soaks without `quiet-mobs`, and by playtest (DP-1440 came from Zach noticing that `review` was always empty).
3. **The fixes collide.** Structural fixes touch the same hot paths: mobact arms, the death sequence, combat entry. Three agents working in parallel had to be sequenced by call site, and one census queue turned that into hours of waiting. That drove the 2026-10-14 switch to batched final censuses (AGENTS.md, PR #1981).
4. **Stubs look finished.** DP-1440's sinks were labelled "Stub helpers... Remove as native mob comm is added". Everything compiled, the C text sat in Go, and the unit tests that inspected the computed message passed. It's the "registered but never called" pattern: the content port was complete and the delivery port never happened.

## Claim

At the end of a byte-fidelity port, the remaining defects are mostly **sequencing and delivery** (when, in what order, whether, to whom), not **content**. They need different tools: draw-stream comparison, populated worlds, delivery-path tests that read *every* listener, and human play. They also need different logistics: caller-class audits and serialized integration. That's why a manifest at "99.8% green" can still hide a long tail (compare the "what does done mean" blog seed).

## Caveats

- These are five cases from two days, and they were chosen by the work that happened to be in flight. A real taxonomy would label every defect fixed since DP-1371 by kind (content / order / timing / delivery / reachability) from the fidelity TSV corpus.
- "Mostly" is impressionistic until that labelling is done.
