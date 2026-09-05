# AI Slop Squash: The `yank` Case

> Imported from Daeron's workspace on 2026-08-23. This is a provisional thesis
> note, not a verified result. See claim `PF-002` in `EVIDENCE_LEDGER.tsv`.

The historical Go `doYank` is a candidate canonical illustration of what the
Dark Pawns fidelity process is designed to detect.

According to Daeron's contemporaneous note, the implementation was not visibly
broken: it compiled, was registered, ran, and did something recognizably shaped
like `yank`. But it was reportedly a loose paraphrase of C `do_yank`: different
error strings, omitted self and sleeping branches, a different position
threshold, character names where C used pronouns, and player-only behavior where
C also handled mob followers.

That is the important failure mode. AI-assisted ports need not produce nonsense.
They can produce locally reasonable behavior that survives casual review because
the reviewer recognizes the intended feature. Plausibility becomes camouflage
for semantic drift.

The proposed corrective mechanism is structural rather than stylistic:

1. cite the C authority;
2. inspect the real C call path;
3. run or capture the original behavior;
4. assert exact observable output and hidden state where necessary;
5. reject inventions even when they appear cleaner or more idiomatic.

The candidate thesis is that AI participated in both the drift and its repair.
The difference was not simply a better model. It was the addition of an external
behavioral referent and rules that forced the agent to remain anchored to it.
This may generalize to migrations of any system whose original implementation
can be executed as a ground-truth oracle.

Before this example enters paper prose, verify it against the original C
`do_yank`, the relevant Go history, DP-1214, and byte-exact/oracle tests. Preserve
any original misspellings or awkward output because those details demonstrate
why unaided plausibility is an insufficient fidelity criterion.
