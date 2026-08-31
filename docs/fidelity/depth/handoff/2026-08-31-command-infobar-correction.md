# Depth-fidelity handoff correction: `infobar`

Date: 2026-08-31

## Reason for correction

After the original `infobar` handoff was merged, review of the oracle harness
showed that a `[probe:mortal]` section selects the mortal peer for the entire
probe stream. The merged `infobar-depth.txt` therefore did not durably prove
the immortal branch claimed by its manifest row, even though that same
immortal-only vehicle had been run before the peer was added. This correction
keeps the evidence honest under R1/R5e; it does not change the implementation
or frontier counts.

## Evidence correction

The combined vehicle was split into two explicit scenarios:

- `infobar-depth.txt` has only the first-player immortal actor and proves the
  OFF/ON status, default screen size, exact immortal frame, already-on/off,
  first-token usage, and off-frame cases.
- `infobar-mortal-depth.txt` creates an immortal primary plus an ordinary
  mortal peer and selects `[probe:peer]`, so the mortal-only level and
  needed-experience fields are emitted by the client that is actually
  diffed.

The manifest's `infobar.mortal-frame` proof now points to
`infobar-mortal-depth@1,2,3,5,8`. `make fidelity-depth` remains green at 2,294
total cases: 2,234 proven/delegated, 16 blocked, and 44 excluded (99.3% of
2,250 actionable cases). No `src/` or `darkpawns-c-oracle/` file was edited,
and no new case was claimed.

Both corrected vehicles are green at seeds 1, 2, 3, 5, and 8. Seed 1 was run
with `--show-oracle`; the intended immortal and mortal C blocks are visible.
The only failed attempt was an infrastructure-only C WHOD bind collision at
immortal seed 8; rerunning that seed succeeded with no normalized divergence.

## Queue position

The feature PR #972 and original handoff PR #973 remain merged with green
required checks. This correction is a proof-vehicle follow-up only; it does
not repick `infobar`, the exhausted special-procedure inventory, or the
blocked `objmagic.sleep-entry-gates` row.

After this correction is merged, start the next clean-main session at
`insult`, `src/interpreter.c:518`, and leave another dated handoff. Continue
the interpreter table in order under R1/R2/R3/R4/R5e, with shared ownership
recorded under R5b/R5c.
