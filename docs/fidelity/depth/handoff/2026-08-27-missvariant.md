# Dated Handoff: 2026-08-27 (miss-variant investigation round)

- Round 11 chased the seed-1 miss-variant divergence (lib/misc/messages type-300
  blocks: C picked "You wildly punch at the air..." where Go picked "You swing
  your fist...") and RULED OUT every candidate mechanism, leaving the row
  blocked with a precise residual: an unknown stream-offset or value
  divergence that only the variant-index draw exposes.
- Verified equal: the two corpora are byte-identical; both sides parse 4
  variant blocks for attack type 300; C selects via
  `nr = dice(1, number_of_attacks)` then walks nr-1 links (fight.c:1031-1033)
  and Go mirrors with `Dice(1, len(variants))-1`; C's hit() dam=0 path draws
  exactly two (number(1,20) then the variant — damage()'s IS_NPC-gated
  switcheroo draws never fire for a player attacker); Go's production path is
  the same shape (the engine fallback path measured 1 draw with the recording
  roller — it has no variant draw, so MessageFunc must be wired for parity
  tests); uniform() is float32 with the identical 2.328306437e-10 scale
  constant on both sides, and C's float-to-double promotions in
  number()/percent_load match Go's.
- Behavioral signature that narrows the cause: hit-path damage bands matched
  on every hit seed (2/4/5 — "scratch", "barely hit"), and hit/miss outcomes
  matched on all six seeds; only the miss-variant INDEX diverged, and only on
  seed 1. So either the streams sit one draw apart entering the miss path on
  that seed's history, or a value difference hides inside an outcome band
  everywhere else.
- Suggested next instrument (its own round): a seed-replay harness that dumps
  both servers' raw draws bracketing the kill opener — e.g. a temporary C-side
  log() in number() is impossible without editing src/, so instead replay
  Go's seed-1 stream and compare which draw index produces C's observed
  variant; a one-index shift confirms a hidden extra/missing draw upstream in
  boot or the walk-in path.
