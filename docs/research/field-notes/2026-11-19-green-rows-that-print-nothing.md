# Green rows that print nothing (2026-11-19)

Status: field note. The counts are checked against
`docs/research/l1-strict-2026-11-19.tsv` and
`docs/research/l1-phasec-queue-2026-11-16.tsv`; the one worked example is
checked by hand against the retained census dump. Ledger: PF-132.

## What happened

Luna's Phase C worked all 571 queue orders from the 11-16 strict L1 queue.
The roadmap then carried L1 as "0 by Luna's reports". The 11-19 recount
applied one rule to every uncovered segment in a fresh full-census dump: a
segment counts as witnessed only if a C transcript in that dump prints its
exact bytes. It found 163 open.

Joining the open segments against the Phase C queue:

- 102 of the 163 were in the queue, so a batch had already reported them
  settled: 50 cited-not-witnessed, 47 wide-cite-only, 4 uncited and 1 unit
  row that fails the 11-11 unit rule.
- 61 were never in the queue. 53 of those are `src/constants.c`
  `spell_wear_off_msg` entries. The 11-16 population took only segments some
  manifest already cited, so a C table nobody cited could not enter it.

## The worked example

`docs/fidelity/depth/consider.tsv` row `consider.health-lot-better-output`
was added in Phase C batch 3 (83ab27f19, 2026-11-16). It is `oracle-green`,
proof `consider-output-branches-depth@1`, and its note says the target's HP
"reach[es] the lot-better line" (`src/act.informative.c:2402-2403`). The C
transcript of that scenario in the 11-19 dump prints "in about the same
physical shape as you" and never "in a lot better physical shape". No
transcript in the dump prints the lot-better line at all.

The scenario is green because both engines print the same wrong branch. The
row's claim about which branch it reaches was never checked against the C
bytes.

## Why it matters

This is the slop-squash pattern one level up. The yank mis-port was
plausible code; this is plausible evidence. A green scenario, a C cite and a
sentence saying which branch it reaches all look like a witness, and per-batch
review accepted them. A mechanical check over the C transcript, applied to
the whole population at once, did not.

The fix to the method is the check itself: a row is witnessed only when the
C transcript of its named scenario contains the segment's bytes. Phase C
batches should have run that check per row; the next queue does.

## Not the finding

The `!Fireball!`-style placeholders are C's own text
(`src/constants.c:1218` onward), ported verbatim to `pkg/game/constants.go`.
C prints any non-empty entry when an affect of that type expires
(`src/magic.c:449-451`), so each one is either reachable or not, and each
needs its own answer. They are uncited, not invented.

## A second case: a citation copied 1,487 times

The same day turned up a second, separate instance of evidence that looks
right and isn't. On main, 226 files under `docs/` cite `comm.c:910-947`,
1,487 times in all, mostly as the second half of a position-gate cite such
as `interpreter.c:419; comm.c:910-947`. Those lines of `src/comm.c` are the
body of `record_usage()` (declared at `src/comm.c:903`), which counts open
sockets and writes a usage line to the log. They have nothing to do with the
command interpreter or any position check. `src/comm.c` has not changed
since 2026-04-23, so the cite was wrong from its first appearance in the doh
social row (#842, 2026-08-30) and was then copied into each new manifest.

The rows themselves were not wrong: the interpreter cite next to it is the
real gate, and the scenarios witness it. What failed is the same thing as in
the consider example. Each row was reviewed by itself, and a cite that looks
like a C line range reads as checked. Nobody opened the lines. Luna's batch-one
PR (#2861, open as of this note) removes the cite from every manifest; what
is left after it are historical research TSVs, which record what the
manifests said at the time.

The check that would have caught it is cheap: for every distinct cite in the
manifests, open the range once and confirm it contains the code the note
names. A cite repeated across hundreds of rows is checked once, not once per
row.
