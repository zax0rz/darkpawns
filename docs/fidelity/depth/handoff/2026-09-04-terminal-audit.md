# Depth-fidelity terminal audit — 2026-09-04

## Initial handoff

This terminal-audit slice starts from `origin/main` at `2a4e13b86`, after the
final `yuball` social slice. The checked-out repository has no tracked
changes; the pre-existing untracked `website/static/images/` directory is
preserved and is outside this fidelity scope.

The per-case depth generator reports 4,758 total cases: 4,653
proven/delegated, 54 blocked, and 51 excluded. The separate
`surface-inventory.tsv` enumerates 70 rows and 4,926 weighted units; 8 rows
are `proven-already`, 61 are still `unproven`, and 1 is
`excluded-with-C-reason`. It is intentionally outside the generated case
count, so this slice audits it as a second, explicit denominator.

The terminal pass will record current-main reruns for the Phase 2 reds,
recheck the exclusion decisions and blocked clinic, then process the surface
inventory in source order. Each residual surface row will end as proven,
blocked with a named owner/evidence boundary, or excluded with an explicit C
reachability reason. No real player-visible gap will be relabeled as an
exclusion (R1/R2/R4/R5e).

No C oracle or `src/` file is editable in this slice. This file is committed
before the evidence and inventory edits, as required by the depth-loop
handoff protocol.

## Phase 2 and clinic recheck

The current-main probes used the pinned oracle with `--show-oracle`, a
240-second timeout, and seeds 1 and 2 where noted. The named red families
remain bounded blockers:

| family | current evidence | terminal classification |
| --- | --- | --- |
| `accuse-noarg-depth` | seed 1 still returns non-text pointer-like C bytes while Go returns `Accuse who??`; the earlier seed-2 attempt is recorded in `accuse.tsv` | blocked under R1/R4/R5e; do not copy an oracle/runtime anomaly |
| `force-mob` | seeds 1 and 2 still resolve the NPC only in C, which runs `sit`/`stand` through `command_interpreter`; Go returns `No-one by that name here` | blocked under R2/R4/R5b/R5c/R5e; shared NPC interpreter owner |
| `medit-entry-depth` / `medit-session-depth` | seed 1 still diverges on entry and valid `18305` session/menu bytes; the recorded session `q` also proves the missing state transition | blocked under R1/R2/R4/R5e; shared OLC owner |
| `redit-entry-depth` / `redit-session-depth` | seed 1 still diverges on entry and current-room session/menu bytes | blocked under R1/R2/R4/R5e; shared OLC owner |
| `spec-proc-dragon-breath-combat` | seed 1 still matches through the exact breath output, then C continues into melee/death/loot while Go stops; the seed-2 attempt is in the dragon handoff | blocked under R1/R3/R4/R5b/R5c/R5e; shared combat owner |

The clinic is likewise resolved without relabeling a gap as green or
excluded. `force-mob` is the two-attempt NPC-interpreter blocker above.
`sleep-spell-depth` remains green at its reachable cast boundary, while the
object-magic self-target entry remains blocked because C's `TAR_NOT_SELF`
gate makes that entry unreachable. The refreshed janitor vehicle completed
with empty C and Go pulse blocks, so it is an unexecuted dispatch proof, not a
parity claim. Cityguard seeds 1 and 2 still diverge at shared death/respawn;
cityguard-breed seeds 1 and 2 still omit C's `Die, nightbreed!!` line in Go.
Paladin is stable green on seeds 1, 2, 3, 5, and 8 and remains promoted in
`spec-procs.tsv`. Teleport-victim is green at seed 1 but divergent at seed 2
in the pre-special hit result and random landing room, so it remains blocked
at the shared combat/RNG boundary. These results satisfy the bounded-attempt
rule and preserve the existing clinic handoff evidence.

## Exclusion verification

The 51 excluded depth rows were checked for a non-empty C reachability or
undefined-behavior reason; all 51 have one. The key high-risk decisions remain
verified against the actual call paths:

- `spec_assign.c:421-422` assigns vnums 15808 and 15814, but their
  `lib/world/mob/158.mob` action flags are `153608` and `26650`; both are even
  and therefore lack `MOB_SPEC` (`structs.h:247`).
- `drink.vampire`, `eat.vampire`, and `eat.werewolf-corpse` remain blocked,
  not excluded, because reachable player transformation state can enter those
  branches.
- Descriptorless early returns, NPC-only page/flee branches, unregistered
  specials, script-only entry points, no-target socials, and the two
  overlapping-`sprintf` undefined-behavior rows retain their cited C reasons.
- `use.tattoo` and `room.jail-commandless-body` remain blocked after the prior
  false-exclusion correction; neither is silently dropped from the actionable
  frontier.

## Surface-inventory terminal classification

`surface-inventory.tsv` is a separate weighted enumeration, not a per-case
manifest. Its 70 rows cover 4,926 weighted units in C source order. The
initial 61 `unproven` rows have now been processed: each is explicitly
`blocked` with the terminal-audit owner token and a note naming its residual
source family/evidence boundary. This includes the full 1..299 cast vectors,
the skill-message and combat/death corpora, lifecycle rows, shop rows, and
every residual `act()`/`send_to_char()` file bucket. The eight already-proven
rows remain `proven-already` (887 units), and the one off-command handler
remains `excluded-with-C-reason` (1 unit). There are no `unproven` inventory
rows and no newly asserted exclusions:

```text
surface rows: 70
weighted units: 4,926
proven-already: 8 rows / 887 units
blocked: 61 rows / 4,038 units
excluded-with-C-reason: 1 row / 1 unit
unproven: 0 rows / 0 units
```

The unit count is an enumeration denominator, not a claim that every weighted
unit has a green transcript. The explicit blocked status preserves the real
remaining work and keeps the terminal inventory honest under R1/R2/R4/R5e.

## Manifest integrity

The terminal audit also checked every nonblank depth-manifest row and the
research ledger for rectangular TSV shape. It found 71 pre-existing depth
rows whose final `notes` field had shifted into the `c_site` column (including
the late social slices and `gen-tog`); those rows were repaired with their
source-path field restored. `scripts/gen_fidelity_depth.py` now rejects both
extra and missing tab-separated fields, so a shifted row cannot silently
pass as a green proof. The repaired manifests still generate the same
4,758-case report and no player-facing Go code changed.

## Social terminal state

The social command corpus contains 1,740 `do_action` rows: 1,734 are
proven/delegated, 4 are the bounded `accuse` anomaly/target-family blockers,
and 2 are the source records (`sniff` and `snore`) with no target-message
slots. All remaining social slices through `yuball` are merged on main; the
last slice is PR #1383 at `2a4e13b86`. The two exclusions are not silent
omissions, and the four `accuse` rows retain the two-attempt evidence and
R1/R4/R5e ruling above.

## Terminal verdict

The breadth-to-depth loop's terminal audit conditions are met at this dated
frontier: the remaining social queue is exhausted, the named red families
and clinic gaps have bounded dispositions, exclusions have C-path reasons,
and the off-command inventory has no `unproven` or newly asserted excluded
row. The repository remains a faithful-work frontier rather than a claim of
whole-game completion; the 54 modeled blockers plus the 61 weighted surface
blockers are deliberately retained for future owner-specific depth work.
